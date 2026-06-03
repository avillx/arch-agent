package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"context"
	"fmt"
)

type AgentRuntime struct {
	observer *Observer
}

func NewAgentRuntime(observer *Observer) *AgentRuntime {
	return &AgentRuntime{
		observer: observer,
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

	ctx = withAgentID(ctx, agt.ID())
	ctx = withSessionID(ctx, sess.ID())

	// wraps event channel for observing avtivity
	sink := r.observer.Intercept(ctx, sess.GetLastUserMessageContent(), agt.ID(), sess.ID(), evCh)
	defer close(sink)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if shouldCompact(sess, model) {
			doCompact(ctx, CompactionRequest{
				sess:  sess,
				agt:   agt,
				model: model,
			})
			sink <- NewCompactionEvent(
				agt.ID(),
				sess.ID(),
				sess.Summary(),
			)

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

	messages := sess.Messages()

	summary := sess.Summary()
	if summary != "" {
		messages = append(summaryToDialog(summary), messages...)
	}

	result, err := model.Complete(
		ctx,
		tools,
		append([]agent.Message{assembeSystemMessage(agt, tools)}, messages...),
	)
	if err != nil {
		return true, err
	}

	sess.AddMessages([]agent.Message{agent.NewAgentMessage(result.Content, result.ToolCalls)})

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
		var resultMessage *agent.ToolResultMessage

		tool, exist := toolkit[call.ToolName]

		if exist {
			res, err := tool.Call(ctx, call.Arguments)
			if err != nil {
				sink <- NewErrToolCallEvent(agt.ID(), sess.ID(), err)
				res += err.Error()
			}
			resultMessage = agent.NewToolResultMessage(call.ID, res)

		} else {
			resultMessage = agent.NewToolResultMessage(
				call.ID,
				fmt.Sprintf("tool %s is not exist", call.ToolName),
			)
		}
		sess.AddMessages([]agent.Message{resultMessage})
		sink <- NewToolCallResultEvent(agt.ID(), sess.ID(), resultMessage.Content())
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

func summaryToDialog(summary string) []agent.Message {
	return []agent.Message{
		agent.NewUserMessage(fmt.Sprintf("Here is summary of previous conversations%s", summary)),
		agent.NewAgentMessage("okay i will account it", nil),
	}
}
