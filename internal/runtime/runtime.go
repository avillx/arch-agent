package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"
)

const (
	defaultTurnsLimit   = 100
	maxCompactApptempts = 3
)

type AgentRuntime struct {
	// TODO: expose observer out of runtime as higher level conception
	observer         *Observer
	contextAssembler *ContextAssembler
}

func NewAgentRuntime(
	observer *Observer,
	contextAssembler *ContextAssembler,
) *AgentRuntime {
	return &AgentRuntime{
		observer:         observer,
		contextAssembler: contextAssembler,
	}
}

// blocking
func (r *AgentRuntime) RunStream(
	ctx context.Context,
	model agent.Model,
	agt agent.Agent,
	tools []agent.Tool,
	sess session.Session,
	evCh chan Event,
	logActivity bool,
) error {

	ctx = withAgentID(ctx, agt.ID())
	ctx = withSessionID(ctx, sess.ID())

	// wraps event channel for observing aсtivity
	if logActivity {
		evCh = r.observer.Intercept(ctx, []agent.Message{sess.GetLastUserMessage()}, agt.ID(), sess.ID(), evCh)
	}

	defer close(evCh)

	compactAttempts := 0

	maxTurns := resolveMaxTurns(model.Settings())

	for i := 0; i < maxTurns; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		done, err := r.runTurn(ctx, model, agt, tools, sess, evCh)
		if err != nil {

			// compact session
			// This exists because the context limit is set by model settings.
			// A custom context limit lets us set a limit lower than the actual
			// model limit to avoid hallucinations. Edge case: if settings are
			// misconfigured and the context limit is greater than the actual
			// model limit, completion will return ErrContextOverflow — that's fine,
			// since it will be compacted anyway.
			if errors.Is(err, ErrContextOverflow) {

				compactAttempts++
				if compactAttempts > maxCompactApptempts {
					return NewRuntimeError(sess.ID(), agt.ID(), fmt.Errorf("compaction limit exceeded"))
				}

				if err := doCompact(ctx, sess, agt, model, evCh); err != nil {
					return NewRuntimeError(sess.ID(), agt.ID(), fmt.Errorf("session compaction: %w", err))
				}
				continue
			}

			evCh <- NewErrEvent(agt.ID(), sess.ID(), err)

			// if problem is in tool call, interruption of loop is not needed
			// agent and user is notified about internal error
			// agent should continue acting with account a problem
			var toolCallError *ToolCallError
			if errors.As(err, &toolCallError) {
				slog.Error("tool call", "error", toolCallError)
				continue
			}
			// unknown problem expecting as a completion problem when model can't
			// complete a request
			return NewRuntimeError(sess.ID(), agt.ID(), err)
		}

		if done {
			return nil
		}
	}

	return NewRuntimeError(sess.ID(), agt.ID(), fmt.Errorf("max turns limit exceed"))
}

func (r *AgentRuntime) runTurn(
	ctx context.Context,
	model agent.Model,
	agt agent.Agent,
	tools []agent.Tool,
	sess session.Session,
	evCh chan Event,
) (bool, error) {

	// check compaction
	if shouldCompact(sess.InputTokens(), sess.OutputTokens(), model.ContextLimit()) {
		return false, ErrContextOverflow
	}

	// build system message
	contextMessages := []agent.Message{
		r.contextAssembler.assembeSystemMessage(agt, tools),
	}

	// resolve precontext hooks
	preContextMessages := r.contextAssembler.resolvePreContextMessages(agt, sess)
	if len(preContextMessages) > 0 {
		contextMessages = append(contextMessages, preContextMessages...)
	}

	inputMessages := append(contextMessages, sess.Messages()...)

	distillMessages := excludeUnsupportedModalities(inputMessages, model.SupportedModalities())

	// run completion
	result, err := model.Complete(
		ctx,
		tools,
		distillMessages,
	)
	if err != nil {
		return true, fmt.Errorf("completion: %w", err)
	}

	slog.Debug("completion", "result", result)
	sess.ApplyCompletion(result)
	evCh <- NewCompleteEvent(agt.ID(), sess.ID(), result)

	if err := r.processToolCalls(ctx, agt, tools, result.ToolCalls, sess, evCh); err != nil {
		// can't just call tools and fogot when errors occured
		return false, fmt.Errorf("tool call process: %w", err)
	}

	return result.Done, nil
}

func (r *AgentRuntime) processToolCalls(
	ctx context.Context,
	agt agent.Agent,
	tools []agent.Tool,
	toolcalls []*agent.ToolCall,
	sess session.Session,
	evCh chan Event,
) error {

	toolMap := toolsToMap(tools)
	var errs []error
	for _, call := range toolcalls {

		msg, err := r.processToolCall(ctx, toolMap, call)
		if err != nil {

			var agentMistakeErr *types.AgentMistakeError
			if errors.As(err, &agentMistakeErr) {
				// if error is by agent mistake, no needed to return it.
				// agent should recive error message and correct call
				msg = fmt.Sprintf("%s \nerrors occured: \n%s", msg, agentMistakeErr.Message())
				slog.Warn(
					"bad tool call",
					"agent", agt.ID(),
					"model", agt.Model(),
					"tool", call.ToolName,
					"args", string(call.Arguments),
					"error", err,
				)

				evCh <- NewErrToolCallEvent(agt.ID(), sess.ID(), err)
			} else {

				switch {
				case errors.Is(err, context.DeadlineExceeded):
					msg = fmt.Sprintf("%s tool: timeout error", msg)
				default:
					msg = fmt.Sprintf("%s errors occured: internal error, is not your mistake", msg)
				}

				errs = append(errs, NewToolCallError(call, err))
			}

		}

		slog.Debug("tool called", "result message", msg)

		sess.AddMessages(
			agent.NewToolResultMessage(
				call.ID,
				msg,
			),
		)

		evCh <- NewToolCallResultEvent(agt.ID(), sess.ID(), msg)
	}

	return errors.Join(errs...)
}

func (r *AgentRuntime) processToolCall(
	ctx context.Context,
	toolkit map[agent.ToolName]agent.Tool,
	call *agent.ToolCall,
) (result string, err error) {

	tool, exist := toolkit[call.ToolName]
	if !exist {
		return "", types.NewAgentMistakeError(fmt.Sprintf("tool %s is not exist", call.ToolName))
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)

	defer func() {
		cancel()
		// so if tool is in panic we must not interrupt loop
		// agent should recive internal error message, log reason and enough
		if p := recover(); p != nil {
			err = fmt.Errorf("tool %s panicked: %v", call.ToolName, p)
		}
	}()

	return tool.Call(ctx, call.Arguments)
}

type agentIDCTXKey struct{}
type sessionIDCTXKey struct{}

func withAgentID(ctx context.Context, id agent.ID) context.Context {
	return context.WithValue(ctx, agentIDCTXKey{}, id)
}
func AgentIDFromContext(ctx context.Context) (agent.ID, bool) {
	id, ok := ctx.Value(agentIDCTXKey{}).(agent.ID)
	return id, ok
}

func withSessionID(ctx context.Context, id session.ID) context.Context {
	return context.WithValue(ctx, sessionIDCTXKey{}, id)
}
func SessionIDFromContext(ctx context.Context) (session.ID, bool) {
	id, ok := ctx.Value(sessionIDCTXKey{}).(session.ID)
	return id, ok
}

func resolveMaxTurns(settings agent.ModelSettings) int {
	v, ok := settings["max_turns"]
	if !ok {
		return defaultTurnsLimit
	}

	maxTurns, ok := v.(int)
	if !ok {
		slog.Error("resolve max turns", "error", "bad value type", "used default value", defaultTurnsLimit)
		return defaultTurnsLimit
	}

	return maxTurns
}

func toolsToMap(tools []agent.Tool) map[agent.ToolName]agent.Tool {
	toolMap := map[agent.ToolName]agent.Tool{}

	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}

	return toolMap
}

func excludeUnsupportedModalities(msgs []agent.Message, mdls []agent.Modality) []agent.Message {
	var distill []agent.Message

	if !slices.Contains(mdls, agent.ImageModality) {
		for i, m := range msgs {

			var shouldReplaceMsg bool

			contentParts := m.Content()
			for contentPartIdx := range contentParts {

				if contentParts[contentPartIdx].ImageURL != "" {

					if !shouldReplaceMsg {
						shouldReplaceMsg = true
						contentParts = slices.Clone(contentParts)
					}

					contentParts[contentPartIdx].ImageURL = ""
					contentParts[contentPartIdx].Text += prompt.ExcludedUnsupportedModality(agent.ImageModality)
				}

			}

			if shouldReplaceMsg {
				if distill == nil {
					distill = slices.Clone(msgs)
				}

				distill[i] = agent.CloneMessage(msgs[i])
				distill[i].SetContent(contentParts)
			}
		}
	}

	if distill == nil {
		return msgs
	}

	return distill
}
