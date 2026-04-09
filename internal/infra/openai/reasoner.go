package openaiadapter

import (
	"arch-agent/internal/app/answer"
	"arch-agent/internal/app/executioncontext"
	internalmessage "arch-agent/internal/app/message"
	"arch-agent/internal/infra/llm"
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
)

type Reasoner struct {
	client openai.Client
	model  string
	prompt llm.ReasoningPrompt
	extras map[string]any
}

func NewReasoner(client openai.Client, model string, extras map[string]any) *Reasoner {
	return &Reasoner{
		client: client,
		model:  model,
		prompt: llm.NewReasoningPrompt(),
		extras: extras,
	}
}

func (r *Reasoner) RenderPrompt(params executioncontext.ReasonParams) (string, error) {
	return r.prompt.Render(llm.ReasoningPromptParams{
		Role:                 params.Agent.Role,
		Reflection:           params.Reflection,
		CommunicationContext: params.ContextDescription,
		Preferences:          params.Agent.Preferences,
		KeyPhrases:           params.Agent.Keyphrases,
		BannedSentences:      params.Agent.BannedSlang,
		Memory:               params.Memory,
		Strategy:             params.Strategy,
		Time:                 params.Time.Format("15:04, 2 January of 2006"),
	})
}

func (r *Reasoner) Reason(ctx context.Context, params executioncontext.ReasonParams) (*answer.ReasonResult, error) {
	prompt, err := r.RenderPrompt(params)
	if err != nil {
		return nil, err
	}

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(params.Messages))
	messages = append(messages, openai.SystemMessage(prompt))
	messages = append(messages, messagesToOpenAI(params.Messages)...)

	agentTools := toolDefenitionsToOpenAI(params.Tools)

	completionParams := r.builtCompletionParams(messages, agentTools)

	res, err := r.client.Chat.Completions.New(ctx, completionParams)
	if err != nil {
		return nil, err
	}

	return OpenAICompletionToReasonResult(res)
}

func (r *Reasoner) builtCompletionParams(
	messages []openai.ChatCompletionMessageParamUnion,
	agentTools []openai.ChatCompletionToolUnionParam,
) openai.ChatCompletionNewParams {
	completionParams := openai.ChatCompletionNewParams{
		Model:           r.model,
		Messages:        messages,
		Tools:           agentTools,
		ReasoningEffort: openai.ReasoningEffortMedium,
		ToolChoice:      OpenAIToolChoice(),
		TopP:            openai.Float(1),
		Temperature:     openai.Float(1),
	}
	if r.extras != nil {
		completionParams.SetExtraFields(r.extras)
	}
	return completionParams
}

func OpenAICompletionToReasonResult(completion *openai.ChatCompletion) (*answer.ReasonResult, error) {

	if len(completion.Choices) == 0 {
		return nil, errors.New("empty choices")
	}

	message := completion.Choices[0].Message
	toolCalls := []*internalmessage.ToolCall{}

	if len(message.ToolCalls) > 0 {
		toolCalls = append(toolCalls, openAIToToolCalls(message.ToolCalls)...)
	}

	result := answer.NewReasonResult(toolCalls, message.Content)

	return result, nil
}

func OpenAIToolChoice() openai.ChatCompletionToolChoiceOptionUnionParam {
	return openai.ChatCompletionToolChoiceOptionUnionParam{
		OfAuto: openai.String(string(openai.ChatCompletionAllowedToolsModeAuto)),
	}
}
