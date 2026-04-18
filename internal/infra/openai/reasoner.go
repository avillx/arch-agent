package openaiadapter

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/types"
	"context"

	"github.com/openai/openai-go/v3"
)

// type ToolCallRecivier interface {
// 	ReciveCall(ctx context.Context, call *types.ToolCall) (string, error)
// 	Tools() ([]types.ToolDefinition, error)
// }

type ReasonerConfig struct {
	Client           openai.Client
	Model            string
	ReasoningEffort  string
	ToolChoice       string
	TopP             float32
	FrequencyPenalty float32
	PresencePenalty  float32
	Temperature      float32
	Extras           map[string]any
}

type Reasoner struct {
	baseCompletor
}

func NewReasonerFromConfig(c ReasonerConfig) *Reasoner {
	return &Reasoner{
		baseCompletor: baseCompletor{
			client:           c.Client,
			model:            c.Model,
			reasoningEffort:  c.ReasoningEffort,
			toolChoice:       c.ToolChoice,
			topP:             c.TopP,
			frequencyPenalty: c.FrequencyPenalty,
			presencePenalty:  c.PresencePenalty,
			temperature:      c.Temperature,
			extras:           c.Extras,
		},
	}
}

func (r *Reasoner) Reason(
	ctx context.Context,
	prompt string,
	toolDefs []types.ToolDefinition,
	internalMsgs []types.Message,
) (*reasoning.ReasonResult, error) {

	messages := builtMessages(internalMsgs, prompt)
	agentTools := toolDefenitionsToOpenAI(toolDefs)
	completionParams := r.builtCompletionParams(messages, agentTools)

	res, err := r.client.Chat.Completions.New(ctx, completionParams)
	if err != nil {
		return nil, err
	}

	return OpenAICompletionToReasonResult(res)
}
