package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"context"
	"fmt"
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
) error {

	// WithLogging
	// WithActivityPreload

	ctx = withAgentID(ctx, agt.ID())
	ctx = withSessionID(ctx, sess.ID())

	// wraps event channel for observing avtivity
	sink := r.observer.Intercept(ctx, []agent.Message{sess.GetLastUserMessage()}, agt.ID(), sess.ID(), evCh)
	defer close(sink)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		done, err := r.runTurn(ctx, model, agt, tools, sess, sink)
		if err != nil {
			sink <- NewErrEvent(agt.ID(), sess.ID(), err)
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
	sink chan Event,
) (bool, error) {

	// build system message
	precontextMessages := []agent.Message{
		r.contextAssembler.assembeSystemMessage(agt, tools),
	}

	if sess.Summary() != "" {
		precontextMessages = append(
			precontextMessages,
			preContextHookDialogue(sess.Summary())...,
		)
	}

	// check compaction
	if shouldCompact(sess.InputTokens(), sess.OutputTokens(), model.ContextLimit()) {
		doCompact(ctx, sess, agt, model, sink)
	}

	inputMessages := append(precontextMessages, sess.Messages()...)

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
	sink <- NewCompleteEvent(agt.ID(), sess.ID(), result)

	r.processToolCalls(ctx, agt, tools, result.ToolCalls, sess, sink)

	return result.Done, nil
}

func (r *AgentRuntime) processToolCalls(
	ctx context.Context,
	agt agent.Agent,
	tools []agent.Tool,
	toolcalls []*agent.ToolCall,
	sess session.Session,
	sink chan Event,
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
			sink <- NewErrToolCallEvent(agt.ID(), sess.ID(), err)
			res += err.Error()
		}

		sess.AddMessages(
			agent.NewToolResultMessage(
				call.ID,
				res,
			),
		)

		sink <- NewToolCallResultEvent(agt.ID(), sess.ID(), res)
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

func preContextHookDialogue(instructions string) []agent.Message {
	return []agent.Message{
		agent.NewUserMessage(prompt.SummaryExplanation(instructions)),
		agent.NewAgentMessage("okay i will account it", nil),
	}
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
