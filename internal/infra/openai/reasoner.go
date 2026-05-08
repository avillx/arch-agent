package openaiadapter

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAIReasoner struct {
	id service.LLMID
	openai.Client

	settings    map[string]any
	settingsMu  sync.RWMutex
	secretsRepo Screts
}

type Screts interface {
	Get(string) (string, bool)
}

func NewOpenAIReasoner(secrets Screts, settings map[string]any) (*OpenAIReasoner, error) {
	url, ok := settings["url"].(string)
	if !ok {
		return nil, fmt.Errorf("open ai reasoner base url is not exist")
	}

	keyName, ok := settings["key_name"].(string)
	if !ok {
		return nil, fmt.Errorf("open ai reasoner key name must be non nil")
	}

	apiKey, found := secrets.Get(keyName)
	if !found {
		return nil, fmt.Errorf("api key %s is not exist", keyName)
	}

	id, ok := settings["id"].(string)
	if !ok {
		return nil, fmt.Errorf("open ai reasoner has no ID")
	}

	return &OpenAIReasoner{
		id: service.LLMID(id),
		Client: openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(url),
		),
		settings:    settings,
		secretsRepo: secrets,
	}, nil
}

func (r *OpenAIReasoner) ID() service.LLMID { return r.id }
func (r *OpenAIReasoner) Settings() service.LLMSettings {
	r.settingsMu.RLock()
	defer r.settingsMu.RUnlock()
	return r.settings
}

func (r *OpenAIReasoner) SetSettings(newSettings service.LLMSettings) error {

	r.settingsMu.Lock()
	defer r.settingsMu.Unlock()

	_, newUrlFound := newSettings["url"]
	_, newKeyFound := newSettings["key_name"]
	if newUrlFound && newKeyFound {

		pathedSettings := maps.Clone(r.settings)
		maps.Insert(pathedSettings, maps.All(newSettings))

		newReasoner, err := NewOpenAIReasoner(r.secretsRepo, pathedSettings)
		if err != nil {
			return err
		}

		newReasoner.settingsMu.Lock()
		r = newReasoner
		newReasoner.settingsMu.Unlock()
	}

	return nil
}

func (r *OpenAIReasoner) Reason(
	ctx context.Context,
	toolDefs []types.ToolDefinition,
	internalMsgs []types.Message,
) (*agent.ReasonResult, error) {

	messages := messagesToOpenAI(internalMsgs)
	agentTools := toolDefenitionsToOpenAI(toolDefs)
	completionParams, err := r.buildCompletionParams(messages, agentTools)
	if err != nil {
		return nil, err
	}

	// TODO Create a fallbacks for unexpected end of json
	res, err := r.Client.Chat.Completions.New(ctx, *completionParams)
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
) (*openai.ChatCompletionNewParams, error) {
	model, _, err := getString(r.settings, "model")
	if err != nil {
		return nil, err
	}

	completionParams := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	if agentTools != nil {
		completionParams.Tools = agentTools
	}

	if v, ok, err := getInt64(r.settings, "max_output_tokens"); err != nil {
		return nil, err
	} else if ok {
		completionParams.MaxTokens = openai.Int(v)
	}

	if v, ok, err := getInt64(r.settings, "max_completion_tokens"); err != nil {
		return nil, err
	} else if ok {
		completionParams.MaxCompletionTokens = openai.Int(v)
	}

	if v, ok, err := getString(r.settings, "reasoning_effort"); err != nil {
		return nil, err
	} else if ok {
		completionParams.ReasoningEffort = shared.ReasoningEffort(v)
	}

	if v, ok, err := getString(r.settings, "tool_choice"); err != nil {
		return nil, err
	} else if ok {
		completionParams.ToolChoice = OpenAIToolChoice(v)
	}

	if v, ok, err := getFloat32(r.settings, "top_p"); err != nil {
		return nil, err
	} else if ok {
		completionParams.TopP = openai.Float(float64(v))
	}

	if v, ok, err := getFloat32(r.settings, "temperature"); err != nil {
		return nil, err
	} else if ok {
		completionParams.Temperature = openai.Float(float64(v))
	}

	if v, ok, err := getFloat32(r.settings, "frequency_penalty"); err != nil {
		return nil, err
	} else if ok {
		completionParams.FrequencyPenalty = openai.Float(float64(v))
	}

	if v, ok, err := getFloat32(r.settings, "presence_penalty"); err != nil {
		return nil, err
	} else if ok {
		completionParams.PresencePenalty = openai.Float(float64(v))
	}

	if v, ok, err := getExtras(r.settings, "extras"); err != nil {
		return nil, err
	} else if ok {
		completionParams.SetExtraFields(v)
	}

	return &completionParams, nil
}
