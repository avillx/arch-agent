package openai

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"errors"

	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type SecretsRepo interface {
	Get(string) (string, bool)
}

func messagesToOpenAI(internalFromatMessages []agent.Message) []openai.ChatCompletionMessageParamUnion {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(internalFromatMessages))
	for _, msg := range internalFromatMessages {
		messages = append(messages, messageToOpenAI(msg))
	}
	return messages
}

func messageToOpenAI(internalFromatMessage agent.Message) openai.ChatCompletionMessageParamUnion {

	var result openai.ChatCompletionMessageParamUnion

	switch msg := internalFromatMessage.(type) {
	case *agent.AgentMessage:

		internalContent := msg.Content()
		openAIContent := make([]openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion, len(internalContent))
		for i, cp := range internalContent {
			openAIContent[i] = openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
				OfText: &openai.ChatCompletionContentPartTextParam{
					Text: cp.Text,
				},
			}
		}

		assistantMsg := openai.AssistantMessage(openAIContent)

		toolCalls := msg.ToolCalls()
		if len(toolCalls) > 0 {
			assistantMsg.OfAssistant.ToolCalls = toolCallsToOpenAi(toolCalls)
		}
		result = assistantMsg

	case *agent.SystemMessage:

		internalContent := msg.Content()
		openAIContent := make([]openai.ChatCompletionContentPartTextParam, len(internalContent))
		for i, cp := range internalContent {
			openAIContent[i] = openai.ChatCompletionContentPartTextParam{Text: cp.Text}
		}
		result = openai.SystemMessage(openAIContent)
	case *agent.UserMessage:

		internalContent := msg.Content()
		openAIContent := make([]openai.ChatCompletionContentPartUnionParam, len(internalContent))
		for i, cp := range internalContent {
			if cp.Text != "" {
				openAIContent[i].OfText = &openai.ChatCompletionContentPartTextParam{Text: cp.Text}
			}
			if cp.ImageURL != "" {
				openAIContent[i].OfImageURL = &openai.ChatCompletionContentPartImageParam{
					ImageURL: openai.ChatCompletionContentPartImageImageURLParam{URL: cp.ImageURL},
				}
			}
		}
		result = openai.UserMessage(openAIContent)

	case *agent.ToolResultMessage:

		internalContent := msg.Content()

		openAIContent := make([]openai.ChatCompletionContentPartTextParam, len(internalContent))

		for i, cp := range internalContent {
			openAIContent[i] = openai.ChatCompletionContentPartTextParam{Text: cp.Text}
		}

		result = openai.ToolMessage(openAIContent, msg.ToolCallID())
	}

	return result
}

func toolCallsToOpenAi(toolCalls []*agent.ToolCall) []openai.ChatCompletionMessageToolCallUnionParam {
	openAIToolcalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(toolCalls))
	for _, call := range toolCalls {
		openAIToolcalls = append(openAIToolcalls, toolCallToOpenAi(call))
	}
	return openAIToolcalls
}

func toolCallToOpenAi(toolCall *agent.ToolCall) openai.ChatCompletionMessageToolCallUnionParam {
	return openai.ChatCompletionMessageToolCallUnionParam{
		OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
			ID: toolCall.ID,
			Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
				Name:      string(toolCall.ToolName),
				Arguments: string(toolCall.Arguments),
			},
		},
	}
}

func openAIToToolCalls(openaiToolCalls []openai.ChatCompletionMessageToolCallUnion) []*agent.ToolCall {
	toolCalls := []*agent.ToolCall{}
	for _, tc := range openaiToolCalls {
		newToolCall := agent.NewToolCall(
			tc.ID,
			agent.ToolName(tc.Function.Name),
			agent.ToolArguments(
				tc.Function.Arguments,
			),
		)
		toolCalls = append(toolCalls, newToolCall)
	}
	return toolCalls
}

func toolsToOpenAI(tools []agent.Tool) []openai.ChatCompletionToolUnionParam {
	toolParams := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))

	for _, t := range tools {
		newToolParam := toolToOpenAI(t)
		toolParams = append(toolParams, newToolParam)
	}

	return toolParams
}

func toolToOpenAI(t agent.Tool) openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionToolUnionParam{
		OfFunction: &openai.ChatCompletionFunctionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name: string(t.Name()),
				// Strict:      openai.Bool(true),
				Description: openai.String(t.Description()),
				Parameters:  propertiesToOpenAI(t.Schema()),
			},
		},
	}
}

func propertiesToOpenAI(internalProps []agent.ToolProperty) shared.FunctionParameters {
	functionParams := shared.FunctionParameters{"type": "object"}

	properties := map[string]any{}
	required := []string{}

	for _, prop := range internalProps {
		properties[prop.Name] = propertyToOpenAI(prop)

		if prop.Required {
			required = append(required, prop.Name)
		}
	}

	functionParams["properties"] = properties
	functionParams["required"] = required

	return functionParams
}

func propertyToOpenAI(prop agent.ToolProperty) map[string]any {

	result := map[string]any{
		"type":        prop.Type,
		"description": prop.Description,
	}

	if prop.IsArray {
		result["type"] = "array"
		result["items"] = map[string]any{"type": prop.Type}
	}

	if prop.Enum != nil {
		result["enum"] = prop.Enum
	}

	return result
}

func openAIResponseFormat[T any](strict bool) openai.ChatCompletionNewParamsResponseFormatUnion {
	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
			JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:   "Response",
				Strict: openai.Bool(strict),
				Schema: generateScheme[T](),
			}},
	}
}

func generateScheme[T any]() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties:  false,
		RequiredFromJSONSchemaTags: true,
	}
	var v T
	typeName := reflect.TypeOf(v).Name()

	scheme := reflector.Reflect(v).Definitions[typeName]
	return scheme
}

func OpenAIToolChoice(tc string) openai.ChatCompletionToolChoiceOptionUnionParam {
	return openai.ChatCompletionToolChoiceOptionUnionParam{
		OfAuto: openai.String(tc),
	}
}

func builtMessages(conversation []agent.Message, prompt string) []openai.ChatCompletionMessageParamUnion {
	return append([]openai.ChatCompletionMessageParamUnion{openai.SystemMessage(prompt)}, messagesToOpenAI(conversation)...)
}

func OpenAICompletionToContent(completion *openai.ChatCompletion) (string, error) {
	if len(completion.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	return completion.Choices[0].Message.Content, nil
}

func OpenAICompletionToReasonResult(completion *openai.ChatCompletion) (*agent.Completion, error) {

	if len(completion.Choices) == 0 {
		return nil, errors.New("empty choices")
	}

	message := completion.Choices[0].Message
	toolCalls := []*agent.ToolCall{}

	if len(message.ToolCalls) > 0 {
		toolCalls = append(toolCalls, openAIToToolCalls(message.ToolCalls)...)
	}

	finishReason := openai.CompletionChoiceFinishReason(completion.Choices[0].FinishReason)
	result := &agent.Completion{
		ToolCalls:        toolCalls,
		Content:          message.Content,
		Done:             IsDoneOpenAI(finishReason),
		InputTokens:      completion.Usage.PromptTokens,
		CompletionTokens: completion.Usage.CompletionTokens,
	}

	if IsContextLimit(finishReason) {
		return result, runtime.ErrContextOverflow
	}

	return result, nil
}

func IsDoneOpenAI(finishReason openai.CompletionChoiceFinishReason) bool {
	return finishReason == openai.CompletionChoiceFinishReasonStop
}

func IsContextLimit(finishReason openai.CompletionChoiceFinishReason) bool {
	return finishReason == openai.CompletionChoiceFinishReasonLength
}
