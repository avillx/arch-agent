package openaiadapter

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/types"
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

const NotProvidedFloat = -200
const NotProvidedString = "NotProvided"

type Reasoner struct {
	mu      sync.Mutex
	client  *openai.Client
	repo    SettingsRepo
	secrets SecretsStorage
	params  ModelSettings
}

func NewReasoner(repo SettingsRepo, secrets SecretsStorage) (*Reasoner, error) {
	r := &Reasoner{
		repo:    repo,
		secrets: secrets,
	}
	if err := r.ApplySettings(repo.Value()); err != nil {
		return nil, err
	}
	repo.OnChange(func(t LLMSettings) {
		if err := r.ApplySettings(t); err != nil {
			slog.Error("apply reasoner setting", "error", err)
		}
	})

	return r, nil
}

func (r *Reasoner) ApplySettings(s LLMSettings) error {
	apiKey, found := r.secrets.Get(s.Client.APIKeyName)
	if !found {
		return fmt.Errorf("api key %s is not exist", s.Client.APIKeyName)
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(s.Client.OpenAIURL),
	)
	r.client = &client

	r.params = s.Params

	return nil
}

func (r *Reasoner) Reason(
	ctx context.Context,
	toolDefs []types.ToolDefinition,
	internalMsgs []types.Message,
) (*reasoning.ReasonResult, error) {

	messages := messagesToOpenAI(internalMsgs)
	agentTools := toolDefenitionsToOpenAI(toolDefs)
	completionParams := r.builtCompletionParams(messages, agentTools)

	// TODO Create a fallbacks for unexpected end of json
	res, err := r.client.Chat.Completions.New(ctx, completionParams)
	if err != nil {
		return nil, err
	}

	return OpenAICompletionToReasonResult(res)
}

// TODO - find a way to avoid strict default values and constant stubs
func (r *Reasoner) builtCompletionParams(
	messages []openai.ChatCompletionMessageParamUnion,
	agentTools []openai.ChatCompletionToolUnionParam,
) openai.ChatCompletionNewParams {

	completionParams := openai.ChatCompletionNewParams{
		Model:    r.params.Model,
		Messages: messages,
	}

	if agentTools != nil {
		completionParams.Tools = agentTools
	}

	if r.params.MaxOutputTokens != nil {
		completionParams.MaxTokens = openai.Int(*r.params.MaxOutputTokens)
	}

	// MaxCompletionTokens incudes reasonuing tokens
	if r.params.MaxCompletionTokens != nil {
		completionParams.MaxCompletionTokens = openai.Int(*r.params.MaxCompletionTokens)
	}

	if r.params.ReasoningEffort != nil {
		completionParams.ReasoningEffort = shared.ReasoningEffort(*r.params.ReasoningEffort)
	}

	if r.params.ToolChoice != nil {
		completionParams.ToolChoice = OpenAIToolChoice(*r.params.ToolChoice)
	}

	if r.params.TopP != nil {
		completionParams.TopP = openai.Float(float64(*r.params.TopP))
	}

	if r.params.Temperature != nil {
		completionParams.Temperature = openai.Float(float64(*r.params.Temperature))
	}

	if r.params.FrequencyPenalty != nil {
		completionParams.FrequencyPenalty = openai.Float(float64(*r.params.FrequencyPenalty))
	}

	if r.params.PresencePenalty != nil {
		completionParams.PresencePenalty = openai.Float(float64(*r.params.PresencePenalty))
	}

	if r.params.Extras != nil {
		completionParams.SetExtraFields(*r.params.Extras)
	}
	return completionParams
}
