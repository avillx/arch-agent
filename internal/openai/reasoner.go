package openai

import (
	"arch-agent/internal/agent"
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

var _ agent.Model = (*OpenAIReasoner)(nil)

type OpenAIReasoner struct {
	client     *openai.Client
	settings   OpenAIModelSettings
	settingsMu sync.RWMutex
}

func NewOpenAIReasoner(client *openai.Client, settings OpenAIModelSettings) *OpenAIReasoner {

	return &OpenAIReasoner{
		client:   client,
		settings: settings,
	}
}

func (r *OpenAIReasoner) ContextLimit() int64 {
	return *r.settings.ContextLimit
}

func (r *OpenAIReasoner) Settings() agent.ModelSettings {
	r.settingsMu.RLock()
	defer r.settingsMu.RUnlock()

	return agent.ModelSettings{
		"model":                 r.settings.Model,
		"context_limit":         r.settings.ContextLimit,
		"tool_choice":           r.settings.ToolChoice,
		"reasoning_effort":      r.settings.ReasoningEffort,
		"recall_budget":         r.settings.RecallBudget,
		"temperature":           r.settings.Temperature,
		"top_p":                 r.settings.TopP,
		"frequency_penalty":     r.settings.FrequencyPenalty,
		"presence_penalty":      r.settings.PresencePenalty,
		"max_output_tokens":     r.settings.MaxOutputTokens,
		"max_completion_tokens": r.settings.MaxCompletionTokens,
	}
}

func (r *OpenAIReasoner) SupportedModalities() []agent.Modality {
	modalities := r.settings.Modalities
	if modalities == nil {
		return []agent.Modality{agent.TextModality}
	}

	if !slices.Contains(modalities, agent.TextModality) {
		modalities = append(modalities, agent.TextModality)
	}

	return modalities
}

func (r *OpenAIReasoner) Complete(
	ctx context.Context,
	tools []agent.Tool,
	internalMsgs []agent.Message,
) (*agent.Completion, error) {

	messages := messagesToOpenAI(internalMsgs)
	agentTools := toolsToOpenAI(tools)
	completionParams := r.buildCompletionParams(messages, agentTools)

	res, err := r.client.Chat.Completions.New(ctx, completionParams)
	if err != nil {
		return nil, err
	}

	castedRes, err := OpenAICompletionToReasonResult(res)

	slog.Debug("reasoning", "result", castedRes)

	return castedRes, err
}

func (r *OpenAIReasoner) buildCompletionParams(
	messages []openai.ChatCompletionMessageParamUnion,
	agentTools []openai.ChatCompletionToolUnionParam,
) openai.ChatCompletionNewParams {
	s := r.settings

	completionParams := openai.ChatCompletionNewParams{
		Messages: messages,
	}

	if s.Model != nil {
		completionParams.Model = *s.Model
	}

	if agentTools != nil {
		completionParams.Tools = agentTools
	}

	if s.MaxOutputTokens != nil {
		completionParams.MaxTokens = openai.Int(*s.MaxOutputTokens)
	}

	if s.MaxCompletionTokens != nil {
		completionParams.MaxCompletionTokens = openai.Int(*s.MaxCompletionTokens)
	}

	if s.ReasoningEffort != nil {
		completionParams.ReasoningEffort = shared.ReasoningEffort(*s.ReasoningEffort)
	}

	if s.ToolChoice != nil {
		completionParams.ToolChoice = OpenAIToolChoice(*s.ToolChoice)
	}

	if s.TopP != nil {
		completionParams.TopP = openai.Float(float64(*s.TopP))
	}

	if s.Temperature != nil {
		completionParams.Temperature = openai.Float(float64(*s.Temperature))
	}

	if s.FrequencyPenalty != nil {
		completionParams.FrequencyPenalty = openai.Float(float64(*s.FrequencyPenalty))
	}

	if s.PresencePenalty != nil {
		completionParams.PresencePenalty = openai.Float(float64(*s.PresencePenalty))
	}

	if s.Extras != nil {
		completionParams.SetExtraFields(s.Extras)
	}

	return completionParams
}
