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

	// wraps event channel for observing avtivity
	if logActivity {
		evCh = r.observer.Intercept(ctx, []agent.Message{sess.GetLastUserMessage()}, agt.ID(), sess.ID(), evCh)
	}

	defer close(evCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		done, err := r.runTurn(ctx, model, agt, tools, sess, evCh)
		if err != nil {
			evCh <- NewErrEvent(agt.ID(), sess.ID(), err)
			return err
		}

		if done {
			return nil
		}
	}
}

func (r *AgentRuntime) runTurn(
	ctx context.Context,
	model agent.Model,
	agt agent.Agent,
	tools []agent.Tool,
	sess session.Session,
	evCh chan Event,
) (bool, error) {

	// build system message
	contextMessages := []agent.Message{
		r.contextAssembler.assembeSystemMessage(agt, tools),
	}

	// resolve precontext hooks
	hooks := r.contextAssembler.resolvePreContextHooks(agt, sess)
	if len(hooks) > 0 {
		contextMessages = append(contextMessages, hooks...)
	}

	// check compaction
	if shouldCompact(sess.InputTokens(), sess.OutputTokens(), model.ContextLimit()) {
		doCompact(ctx, sess, agt, model, evCh)
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
		return true, err
	}

	sess.ApplyCompletion(result)

	// add event
	evCh <- NewCompleteEvent(agt.ID(), sess.ID(), result)

	r.processToolCalls(ctx, agt, tools, result.ToolCalls, sess, evCh)

	return result.Done, nil
}

func (r *AgentRuntime) processToolCalls(
	ctx context.Context,
	agt agent.Agent,
	tools []agent.Tool,
	toolcalls []*agent.ToolCall,
	sess session.Session,
	evCh chan Event,
) {
	toolkit := toolsToMap(tools)

	for _, call := range toolcalls {

		tool, exist := toolkit[call.ToolName]
		if !exist {
			sess.AddMessages(
				agent.NewToolResultMessage(
					call.ID,
					fmt.Sprintf("tool %s is not exist", call.ToolName),
				),
			)
			return
		}

		res, err := tool.Call(ctx, call.Arguments)
		if err != nil {
			evCh <- NewErrToolCallEvent(agt.ID(), sess.ID(), err)

			var agentMistakeErr *types.AgentMistakeError
			if errors.As(err, &agentMistakeErr) {
				slog.Warn(
					"agent tool call",
					"cause", err,
					"agentID", agt.ID(),
					"sessID", sess.ID(),
					"tool", call.ToolName,
					"args", call.Arguments,
					"explanation", agentMistakeErr.Message(),
				)
				res += fmt.Sprintf("errors occured:\n%s", agentMistakeErr.Message())
			} else {
				res += "\ninternal error"
				slog.Error(
					"unexpected error on agent tool call",
					"agentID", agt.ID(),
					"sessID", sess.ID(),
					"tool", call.ToolName,
					"args", call.Arguments,
					"error", err,
				)
			}
		}

		slog.Debug("tool called", "result", res)

		sess.AddMessages(
			agent.NewToolResultMessage(
				call.ID,
				res,
			),
		)

		evCh <- NewToolCallResultEvent(agt.ID(), sess.ID(), res)
	}
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
