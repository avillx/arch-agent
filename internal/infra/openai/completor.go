package openaiadapter

import (
	"arch-agent/internal/infra/config"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type baseCompletor struct {
	client           openai.Client
	model            string
	reasoningEffort  string
	toolChoice       string
	topP             float32
	frequencyPenalty float32
	presencePenalty  float32
	temperature      float32
	extras           map[string]any
}

func (c baseCompletor) builtCompletionParams(
	messages []openai.ChatCompletionMessageParamUnion,
	agentTools []openai.ChatCompletionToolUnionParam,
) openai.ChatCompletionNewParams {
	completionParams := openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: messages,
	}

	if agentTools != nil {
		completionParams.Tools = agentTools
	}

	if c.reasoningEffort != config.NotProvidedString {
		completionParams.ReasoningEffort = shared.ReasoningEffort(c.reasoningEffort)
	}

	if c.toolChoice != config.NotProvidedString {
		completionParams.ToolChoice = OpenAIToolChoice(c.toolChoice)
	}

	if c.topP != config.NotProvidedFloat {
		completionParams.TopP = openai.Float(float64(c.topP))
	}

	if c.temperature != config.NotProvidedFloat {
		completionParams.Temperature = openai.Float(float64(c.temperature))
	}

	if c.frequencyPenalty != config.NotProvidedFloat {
		completionParams.FrequencyPenalty = openai.Float(float64(c.frequencyPenalty))
	}

	if c.presencePenalty != config.NotProvidedFloat {
		completionParams.PresencePenalty = openai.Float(float64(c.presencePenalty))
	}

	if c.extras != nil {
		completionParams.SetExtraFields(c.extras)
	}
	return completionParams
}
