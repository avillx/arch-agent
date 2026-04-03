package openaiadapter

import (
	"arch-agent/internal/app/message"
	tools "arch-agent/internal/app/toolexecutor"
	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

func messagesToOpenAI(internalFromatMessages []message.Message) []openai.ChatCompletionMessageParamUnion {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(internalFromatMessages))
	for _, msg := range internalFromatMessages {
		messages = append(messages, messageToOpenAI(msg))
	}
	return messages
}

func messageToOpenAI(internalFromatMessage message.Message) openai.ChatCompletionMessageParamUnion {

	var result openai.ChatCompletionMessageParamUnion

	switch msg := internalFromatMessage.(type) {
	case *message.AgentMessage:
		assistantMsg := openai.AssistantMessage(msg.Content())
		toolCalls := msg.ToolCalls()
		if len(toolCalls) > 0 {
			assistantMsg.OfAssistant.ToolCalls = toolCallsToOpenAi(toolCalls)
		}
		result = assistantMsg
	case *message.SystemMessage:
		result = openai.SystemMessage(msg.Content())
	case *message.UserMessage:
		result = openai.UserMessage(msg.Content())
	case *message.ToolResultMessage:
		result = openai.ToolMessage(msg.Content(), msg.ToolCallID())
	}

	return result
}

func toolCallsToOpenAi(toolCalls []*message.ToolCall) []openai.ChatCompletionMessageToolCallUnionParam {
	openAIToolcalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(toolCalls))
	for _, call := range toolCalls {
		openAIToolcalls = append(openAIToolcalls, toolCallToOpenAi(call))
	}
	return openAIToolcalls
}

func toolCallToOpenAi(toolCall *message.ToolCall) openai.ChatCompletionMessageToolCallUnionParam {
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

func openAIToToolCalls(openaiToolCalls []openai.ChatCompletionMessageToolCallUnion) []*message.ToolCall {
	toolCalls := []*message.ToolCall{}
	for _, tc := range openaiToolCalls {
		newToolCall := message.NewToolCall(
			tc.ID,
			tc.Function.Name,
			message.ToolArguments(
				tc.Function.Arguments,
			),
		)
		toolCalls = append(toolCalls, newToolCall)
	}
	return toolCalls
}

func toolDefenitionsToOpenAI(toolDefs []tools.ToolDefinition) []openai.ChatCompletionToolUnionParam {
	toolParams := make([]openai.ChatCompletionToolUnionParam, 0, len(toolDefs))

	for _, def := range toolDefs {
		newToolParam := toolDefenitionToOpenAI(def)
		toolParams = append(toolParams, newToolParam)
	}

	return toolParams
}

func toolDefenitionToOpenAI(def tools.ToolDefinition) openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionToolUnionParam{
		OfFunction: &openai.ChatCompletionFunctionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name: def.Name,
				// Strict:      openai.Bool(def.Strict),
				Description: openai.String(def.Schema.Description),
				Parameters:  schemaToOpenAI(def.Schema),
			},
		},
	}
}

func schemaToOpenAI(schema tools.Schema) shared.FunctionParameters {
	functionParams := shared.FunctionParameters{"type": "object"}

	properties := map[string]any{}
	required := []string{}

	for _, prop := range schema.Properties {
		properties[prop.Name] = propertyToOpenAI(prop)

		if prop.Required {
			required = append(required, prop.Name)
		}
	}

	functionParams["properties"] = properties
	functionParams["required"] = required

	return functionParams
}

func propertyToOpenAI(prop tools.ToolProperty) map[string]any {
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
