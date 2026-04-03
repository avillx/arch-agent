package openaiadapter

import (
	"arch-agent/internal/app/executioncontext"
	"arch-agent/internal/app/message"
	"arch-agent/internal/infra/llm"
	"context"
	"encoding/json"
	"errors"

	"github.com/openai/openai-go/v3"
)

type ReflectionDTO struct {
	Trigger        string `json:"trigger" jsonschema:"required"`
	Traits         string `json:"traits" jsonschema:"required"`
	Feeling        string `json:"feeling" jsonschema:"required"`
	Desire         string `json:"true_desire" jsonschema:"required"`
	InnerMonologue string `json:"inner_monologue" jsonschema:"required"`
	Tone           string `json:"tone" jsonschema:"required"`
}

type Reflector struct {
	client openai.Client
	model  string
	prompt llm.ReflectionPrompt
	extras map[string]any
}

func NewReflector(client openai.Client, model string, extras map[string]any) *Reflector {
	return &Reflector{
		client: client,
		model:  model,
		prompt: llm.NewReflectionPrompt(),
		extras: extras,
	}
}

func (r *Reflector) buildPrompt(personality string) (string, error) {
	return r.prompt.Render(llm.ReflectionParams{Personality: personality})
}

func (r *Reflector) builtMessages(conversation []message.Message, prompt string) []openai.ChatCompletionMessageParamUnion {
	return append([]openai.ChatCompletionMessageParamUnion{openai.SystemMessage(prompt)}, messagesToOpenAI(conversation)...)
}

func (r *Reflector) builtParams(conversation []message.Message, personality string) (openai.ChatCompletionNewParams, error) {
	prompt, err := r.buildPrompt(personality)
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	params := openai.ChatCompletionNewParams{
		Model:          r.model,
		Messages:       r.builtMessages(conversation, prompt),
		ResponseFormat: openAIResponseFormat[ReflectionDTO](true),
	}

	if r.extras != nil {
		params.SetExtraFields(r.extras)
	}

	return params, nil
}

func (r *Reflector) Reflect(ctx context.Context, conversation []message.Message, personality string) (*executioncontext.Reflection, error) {

	params, err := r.builtParams(conversation, personality)
	if err != nil {
		return nil, err
	}

	completion, err := r.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return completionToRefleciton(completion)
}

func completionToRefleciton(completion *openai.ChatCompletion) (*executioncontext.Reflection, error) {
	if len(completion.Choices) == 0 {
		return nil, errors.New("empty choices")
	}

	var resultDTO ReflectionDTO
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &resultDTO); err != nil {
		return nil, err
	}

	result := ReflectionDTOToReflection(resultDTO)

	return &result, nil
}

func ReflectionDTOToReflection(dto ReflectionDTO) executioncontext.Reflection {
	return executioncontext.NewReflection(
		dto.Trigger,
		dto.Traits,
		dto.Feeling,
		dto.Desire,
		dto.InnerMonologue,
		dto.Tone,
	)
}
