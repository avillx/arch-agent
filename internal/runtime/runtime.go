package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	harness *Harness,
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

		done, err := r.runTurn(ctx, model, agt, tools, sess, harness, evCh)
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

func (r *AgentRuntime) buildContext(
	sess session.Session,
	agt agent.Agent,
	tools []agent.Tool,
	model agent.Model,
) []agent.Message {

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

	return excludeUnsupportedModalities(inputMessages, model.SupportedModalities())
}

func (r *AgentRuntime) runTurn(
	ctx context.Context,
	model agent.Model,
	agt agent.Agent,
	tools []agent.Tool,
	sess session.Session,
	harness *Harness,
	evCh chan Event,
) (bool, error) {
	// check compaction
	if shouldCompact(sess.InputTokens(), sess.OutputTokens(), model.ContextLimit()) {
		return false, ErrContextOverflow
	}

	// context
	agentContext := r.buildContext(sess, agt, tools, model)

	// run completion
	completion, err := r.processCompletion(ctx, agentContext, agt, model, tools, sess, harness, evCh)
	if err != nil {
		return true, fmt.Errorf("completion processing: %w", err)
	}

	// run toolCalls
	if err := r.processToolCalls(ctx, agt, tools, completion.ToolCalls, sess, harness, evCh); err != nil {
		// can't just call tools and fogot when errors occured
		return false, fmt.Errorf("tool calls processing: %w", err)
	}

	return completion.Done, nil
}

func (r *AgentRuntime) processCompletion(
	ctx context.Context,
	completionContext []agent.Message,
	agt agent.Agent,
	model agent.Model,
	tools []agent.Tool,
	sess session.Session,
	harness *Harness,
	evCh chan Event,
) (*agent.Completion, error) {

	completion, err := model.Complete(
		ctx,
		tools,
		completionContext,
	)
	if err != nil {
		return nil, fmt.Errorf("completion: %w", err)
	}

	slog.Debug("completion", "agent", agt.ID(), "result", completion)

	sess.ApplyCompletion(completion)
	evCh <- NewCompleteEvent(agt.ID(), sess.ID(), completion)

	if harness != nil && harness.OnComplete != nil {

		// apply completion hooks
		var newCompletion *agent.Completion
		newCompletion, err = harness.OnComplete.Apply(sess.ID(), agt, completion)

		// check on agent ,mistakes
		var agentMistake *types.AgentMistakeError
		if err != nil {
			if !errors.As(err, &agentMistake) {
				return nil, fmt.Errorf("harness: %w", err)
			}
			slog.Warn("agent make completion mistakes", "mistake", agentMistake.Message())
			sess.AddMessages(agent.NewUserMessage(agentMistake.Message()))
		}

		completion = newCompletion
	}

	return completion, nil
}

func (r *AgentRuntime) processToolCalls(
	ctx context.Context,
	agt agent.Agent,
	tools []agent.Tool,
	toolcalls []*agent.ToolCall,
	sess session.Session,
	harness *Harness,
	evCh chan Event,
) error {

	toolMap := toolsToMap(tools)

	var onCallHooks HookSet[*agent.ToolCall]
	if harness != nil {
		onCallHooks = harness.OnToolCall
	}

	var errs []error
	for _, call := range toolcalls {
		msg, err := r.processToolCall(ctx, agt, sess, toolMap, call, onCallHooks, harness.OnToolCallResultMessage)
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

const defaultToolCallTimeout = 30 * time.Second

func (r *AgentRuntime) processToolCall(
	ctx context.Context,
	agt agent.Agent,
	sess session.Session,
	toolkit map[agent.ToolName]agent.Tool,
	call *agent.ToolCall,
	onCallHooks HookSet[*agent.ToolCall],
	afterCallHooks HookSet[*AfterToolCall],
) (result string, err error) {

	tool, exist := toolkit[call.ToolName]
	if !exist {
		return "", types.NewAgentMistakeError(fmt.Sprintf("tool %s is not exist", call.ToolName))
	}

	if onCallHooks != nil {
		newCall, err := onCallHooks.Apply(sess.ID(), agt, call)
		var agentMistake *types.AgentMistakeError
		if err != nil {
			if errors.As(err, &agentMistake) {
				return "", err
			}
		}
		call = newCall
	}

	timeout := defaultToolCallTimeout
	if customTO, ok := tool.(interface{ TimeOut() time.Duration }); ok {
		timeout = customTO.TimeOut()
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer func() {
		cancel()
		// so if tool is in panic we must not interrupt loop
		// agent should recive internal error message, log reason and enough
		if p := recover(); p != nil {
			err = fmt.Errorf("tool %s panicked: %v", call.ToolName, p)
		}
	}()

	result, err = tool.Call(ctx, call.Arguments)

	if afterCallHooks != nil {
		atc, hErr := afterCallHooks.Apply(
			sess.ID(),
			agt,
			&AfterToolCall{
				ToolCall: call,
				ToolCallResult: &agent.ToolCallResult{
					Result: result,
				},
			},
		)
		err = errors.Join(err, hErr)
		result = atc.Result
	}

	return
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
