package openaiadapter

import (
	"arch-agent/internal/app/message"
	"arch-agent/internal/infra/llm"
	"context"
	"encoding/json"
	"errors"

	"github.com/openai/openai-go/v3"
)

type Summarizer struct {
	client openai.Client
	model  string
	prompt llm.SummaryPrompt
	extras map[string]any
}

func NewSummarizer(client openai.Client, model string, extras map[string]any) *Summarizer {
	return &Summarizer{
		client: client,
		model:  model,
		prompt: llm.NewSummaryPrompt(),
		extras: extras,
	}
}

func (r *Summarizer) RenderPrompt() (string, error) {
	return r.prompt.Render(llm.SummaryParams{})
}

func (r *Summarizer) Sum(ctx context.Context, messagesForSum []message.Message) (string, error) {
	prompt, err := r.RenderPrompt()
	if err != nil {
		return "", err
	}

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messagesForSum))
	messages = append(messages, openai.SystemMessage(prompt))
	msgs, err := marshalMessages(messagesForSum)
	if err != nil {
		return "", err
	}

	messages = append(messages, openai.UserMessage(msgs))
	completionParams := r.builtCompletionParams(messages)

	res, err := r.client.Chat.Completions.New(ctx, completionParams)
	if err != nil {
		return "", err
	}

	return OpenAICompletionToContent(res)
}

func (r *Summarizer) builtCompletionParams(
	messages []openai.ChatCompletionMessageParamUnion,
) openai.ChatCompletionNewParams {
	completionParams := openai.ChatCompletionNewParams{
		Model:           r.model,
		Messages:        messages,
		ReasoningEffort: openai.ReasoningEffortHigh,
	}
	return completionParams
}

func OpenAICompletionToContent(completion *openai.ChatCompletion) (string, error) {
	if len(completion.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	message := completion.Choices[0].Message
	return message.Content, nil
}

type messageDTO struct {
	Role     string   `json:"role"`
	Content  string   `json:"content"`
	ToolCall []string `json:"tool_call"`
}

func marshalMessages(messages []message.Message) (string, error) {
	onMarshal := make([]messageDTO, 0, len(messages))
	for _, msg := range messages {

		switch v := msg.(type) {
		case *message.AgentMessage:
			dto := messageDTO{
				Role:    string(v.Role()),
				Content: v.Content(),
			}
			for _, tc := range v.ToolCalls() {
				dto.ToolCall = append(dto.ToolCall, tc.ToolName()+string(tc.Arguments()))
			}
			onMarshal = append(onMarshal, dto)
			continue
		}

		onMarshal = append(onMarshal, messageDTO{
			Role:    string(msg.Role()),
			Content: msg.Content(),
		})

	}

	result, err := json.Marshal(onMarshal)

	return string(result), err
}
