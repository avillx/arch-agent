package openaiadapter

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/types"
	"errors"

	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

func messagesToOpenAI(internalFromatMessages []types.Message) []openai.ChatCompletionMessageParamUnion {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(internalFromatMessages))
	for _, msg := range internalFromatMessages {
		messages = append(messages, messageToOpenAI(msg))
	}
	return messages
}

func messageToOpenAI(internalFromatMessage types.Message) openai.ChatCompletionMessageParamUnion {

	var result openai.ChatCompletionMessageParamUnion

	switch msg := internalFromatMessage.(type) {
	case *types.AgentMessage:
		assistantMsg := openai.AssistantMessage(msg.Content())
		toolCalls := msg.ToolCalls()
		if len(toolCalls) > 0 {
			assistantMsg.OfAssistant.ToolCalls = toolCallsToOpenAi(toolCalls)
		}
		result = assistantMsg
	case *types.SystemMessage:
		result = openai.SystemMessage(msg.Content())
	case *types.UserMessage:
		result = openai.UserMessage(msg.Content())
	case *types.ToolResultMessage:
		result = openai.ToolMessage(msg.Content(), msg.ToolCallID())
	}

	return result
}

func toolCallsToOpenAi(toolCalls []*types.ToolCall) []openai.ChatCompletionMessageToolCallUnionParam {
	openAIToolcalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(toolCalls))
	for _, call := range toolCalls {
		openAIToolcalls = append(openAIToolcalls, toolCallToOpenAi(call))
	}
	return openAIToolcalls
}

func toolCallToOpenAi(toolCall *types.ToolCall) openai.ChatCompletionMessageToolCallUnionParam {
	return openai.ChatCompletionMessageToolCallUnionParam{
		OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
			ID: toolCall.ID(),
			Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
				Name:      toolCall.ToolName(),
				Arguments: string(toolCall.Arguments()),
			},
		},
	}
}

func openAIToToolCalls(openaiToolCalls []openai.ChatCompletionMessageToolCallUnion) []*types.ToolCall {
	toolCalls := []*types.ToolCall{}
	for _, tc := range openaiToolCalls {
		newToolCall := types.NewToolCall(
			tc.ID,
			tc.Function.Name,
			types.ToolArguments(
				tc.Function.Arguments,
			),
		)
		toolCalls = append(toolCalls, newToolCall)
	}
	return toolCalls
}

func toolDefenitionsToOpenAI(toolDefs []types.ToolDefinition) []openai.ChatCompletionToolUnionParam {
	toolParams := make([]openai.ChatCompletionToolUnionParam, 0, len(toolDefs))

	for _, def := range toolDefs {
		newToolParam := toolDefenitionToOpenAI(def)
		toolParams = append(toolParams, newToolParam)
	}

	return toolParams
}

func toolDefenitionToOpenAI(def types.ToolDefinition) openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionToolUnionParam{
		OfFunction: &openai.ChatCompletionFunctionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name: def.Name,
				// Strict:      openai.Bool(true),
				Description: openai.String(def.Description),
				Parameters:  propertiesToOpenAI(def.Properties),
			},
		},
	}
}

func propertiesToOpenAI(internalProps []types.ToolProperty) shared.FunctionParameters {
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

func propertyToOpenAI(prop types.ToolProperty) map[string]any {
	propRepresntation := map[string]any{
		"type":        prop.Type,
		"description": prop.Description,
	}

	if prop.Enum != nil {
		propRepresntation["enum"] = prop.Enum
	}

	return propRepresntation
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

func builtMessages(conversation []types.Message, prompt string) []openai.ChatCompletionMessageParamUnion {
	return append([]openai.ChatCompletionMessageParamUnion{openai.SystemMessage(prompt)}, messagesToOpenAI(conversation)...)
}

func OpenAICompletionToContent(completion *openai.ChatCompletion) (string, error) {
	if len(completion.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	return completion.Choices[0].Message.Content, nil
}

func OpenAICompletionToReasonResult(completion *openai.ChatCompletion) (*reasoning.ReasonResult, error) {

	if len(completion.Choices) == 0 {
		return nil, errors.New("empty choices")
	}

	message := completion.Choices[0].Message
	toolCalls := []*types.ToolCall{}

	if len(message.ToolCalls) > 0 {
		toolCalls = append(toolCalls, openAIToToolCalls(message.ToolCalls)...)
	}

	result := &reasoning.ReasonResult{
		ToolCalls: toolCalls,
		Content:   message.Content,
		Done:      IsDoneOpenAI(completion.Choices[0].FinishReason),
	}

	return result, nil
}

func IsDoneOpenAI(finishReason string) bool {
	if finishReason == "stop" {
		return true
	}
	return false
}
