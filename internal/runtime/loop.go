package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultTurnsLimit      = 100
	defaultToolCallTimeout = 30 * time.Second
	maxCompletionMistakes  = 3
)

var (
	ErrContextOverflow  = errors.New("context is overflow")
	ErrMaxTurnsExceeded = errors.New("max turns limit exceed")
	ErrTooManyMistakes  = errors.New("agent makes too much mistakes")
)

// blocking
func RunAgentLoop(
	ctx context.Context,
	model agent.Model,
	messages []agent.Message,
	tools []agent.Tool,
	evCh chan Event,
	hooks []any,
) {

	completionMistakes := 0

	for range resolveMaxTurns(model.Settings()) {
		select {
		case <-ctx.Done():
			evCh <- NewLoopExitEvent(ctx.Err())
			return
		default:
		}

		// run completion
		completion, err := processCompletion(ctx, model, tools, messages, hooks)
		if err != nil {

			// overflow handling
			if errors.Is(err, ErrContextOverflow) {
				messages, err = doCompact(ctx, model, messages, evCh)
				if err != nil {
					evCh <- NewLoopExitEvent(fmt.Errorf("unhandled overflow: %w", err))
					return
				}
				continue
			}

			// agent mistake handling
			var mistakeErr *types.AgentMistakeError
			if errors.As(err, &mistakeErr) {
				evCh <- NewCompletionMistakeEvent(err)
				completionMistakes++

				if completionMistakes > maxCompletionMistakes {
					evCh <- NewLoopExitEvent(ErrTooManyMistakes)
					return
				}

				agentMsg := agent.NewAgentMessage(completion.Content, nil)
				messages = append(messages, agentMsg)

				cautionMsg := agent.NewUserMessage(mistakeErr.Message())
				messages = append(messages, cautionMsg)

				continue
			}

			evCh <- NewLoopExitEvent(fmt.Errorf("completion: %w", err))
			return
		}
		evCh <- NewCompleteEvent(completion)
		agentMsg := agent.NewAgentMessage(completion.Content, completion.ToolCalls)
		messages = append(messages, agentMsg)

		// compact if needed
		if shouldCompact(
			completion.InputTokens,
			completion.CompletionTokens,
			model.ContextLimit(),
		) {
			messages, err = doCompact(ctx, model, messages, evCh)
			if err != nil {
				evCh <- NewLoopExitEvent(fmt.Errorf("thereshold compaction: %w", err))
				return
			}
		}

		// tool calls
		toolMsgs := processToolCalls(ctx, tools, completion.ToolCalls, hooks, evCh)
		messages = append(messages, toolMsgs...)

		if completion.Done {
			evCh <- NewLoopExitEvent(nil)
			return
		}
	}

	evCh <- NewLoopExitEvent(ErrMaxTurnsExceeded)
}

func processCompletion(
	ctx context.Context,
	model agent.Model,
	tools []agent.Tool,
	messages []agent.Message,
	hooks []any,
) (*agent.Completion, error) {

	completion, err := model.Complete(
		ctx,
		tools,
		messages,
	)
	if err != nil {
		return nil, err
	}

	if len(hooks) > 0 {
		completion, err = ApplyHooks(ctx, hooks, completion)
		if err != nil {
			return completion, err
		}
	}

	return completion, nil
}

func processToolCalls(
	ctx context.Context,
	tools []agent.Tool,
	toolcalls []*agent.ToolCall,
	hooks []any,
	evCh chan Event,
) []agent.Message {
	toolMap := toolsToMap(tools)
	resultMessages := []agent.Message{}

	for _, call := range toolcalls {

		result, err := processToolCall(ctx, toolMap, call, hooks)
		if err != nil {
			evCh <- NewErrToolCallEvent(call.ToolName, call.Arguments, err)
			result = handleToolCallErr(call.ID, err)
		}
		evCh <- NewToolCallResultEvent(call, result)

		resultMessages = append(resultMessages, agent.NewToolResultMessage(result))
	}

	return resultMessages
}

func handleToolCallErr(callID string, err error) *agent.ToolResult {

	msgs := []string{"toolcall error"}

	if errors.Is(err, context.DeadlineExceeded) {
		msgs = append(msgs, "timeout error")
	}

	var agentMistakeErr *types.AgentMistakeError
	if errors.As(err, &agentMistakeErr) {
		msgs = append(msgs, agentMistakeErr.Message())
	}

	return agent.NewToolResult(callID, strings.Join(msgs, ", "))
}

func processToolCall(
	ctx context.Context,
	toolkit map[agent.ToolName]agent.Tool,
	call *agent.ToolCall,
	hooks []any,
) (*agent.ToolResult, error) {

	tool, exist := toolkit[call.ToolName]
	if !exist {
		msg := fmt.Sprintf("tool %s is not exist", call.ToolName)
		return nil, types.NewAgentMistakeError(msg)
	}

	// apply pre call hooks
	if len(hooks) > 0 {
		var err error
		call, err = ApplyHooks(ctx, hooks, call)
		if err != nil {
			return nil, err
		}
	}

	result, err := resolveCallSafely(ctx, tool, call)
	if err != nil {
		return nil, err
	}

	// apply post call hooks
	if len(hooks) > 0 {
		afterToolCall := &AfterToolCall{
			ToolCall:   call,
			ToolResult: result,
		}
		afterToolCall, err = ApplyHooks(ctx, hooks, afterToolCall)
		if err != nil {
			return nil, err
		}

		result = afterToolCall.ToolResult
	}

	return result, err
}

func resolveCallSafely(
	ctx context.Context,
	tool agent.Tool,
	call *agent.ToolCall,
) (result *agent.ToolResult, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("tool %s panicked: %v", tool.Name(), p)
		}
	}()

	ctx, cancel := contextWithToolAwareTimeout(ctx, tool)
	defer cancel()

	content, err := tool.Call(ctx, call.Arguments)
	if err != nil {
		return nil, err
	}

	return agent.NewToolResult(call.ID, content), nil
}

func contextWithToolAwareTimeout(
	ctx context.Context,
	tool agent.Tool,
) (context.Context, context.CancelFunc) {
	timeout := defaultToolCallTimeout
	if customTO, ok := tool.(interface{ TimeOut() time.Duration }); ok {
		timeout = customTO.TimeOut()
	}
	return context.WithTimeout(ctx, timeout)
}

func resolveMaxTurns(settings agent.ModelSettings) int {
	v, ok := settings["max_turns"]
	if !ok {
		return defaultTurnsLimit
	}

	maxTurns, ok := v.(int)
	if !ok {
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
