package openaiadapter

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"errors"
	"fmt"

	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type SecretsRepo interface {
	Get(string) (string, bool)
}

type openAIFactory struct {
	secrets SecretsRepo
}

func NewOpenAIFactory(s SecretsRepo) *openAIFactory {
	return &openAIFactory{
		secrets: s,
	}
}

func (f *openAIFactory) Type() string { return "open_ai" }
func (f *openAIFactory) Produce(settings service.LLMSettings) (service.LLM, error) {
	return NewOpenAIReasoner(f.secrets, settings)
}

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
			ID: toolCall.ID,
			Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
				Name:      toolCall.ToolName,
				Arguments: string(toolCall.Arguments),
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

func OpenAICompletionToReasonResult(completion *openai.ChatCompletion) (*agent.ReasonResult, error) {

	if len(completion.Choices) == 0 {
		return nil, errors.New("empty choices")
	}

	message := completion.Choices[0].Message
	toolCalls := []*types.ToolCall{}

	if len(message.ToolCalls) > 0 {
		toolCalls = append(toolCalls, openAIToToolCalls(message.ToolCalls)...)
	}

	result := &agent.ReasonResult{
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

// validators

func getString(settings map[string]any, key string) (string, bool, error) {
	v, ok := settings[key]
	if !ok {
		return "", false, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", true, fmt.Errorf("%s must be a string, got %T", key, v)
	}
	return s, true, nil
}

func getInt(settings map[string]any, key string) (int, bool, error) {
	v, ok := settings[key]
	if !ok {
		return 0, false, nil
	}
	i, ok := v.(int)
	if !ok {
		return 0, true, fmt.Errorf("%s must be int, got %T", key, v)
	}
	return i, true, nil
}

func getInt64(settings map[string]any, key string) (int64, bool, error) {
	v, ok := settings[key]
	if !ok {
		return 0, false, nil
	}
	i, ok := v.(int64)
	if !ok {
		return 0, true, fmt.Errorf("%s must be int64, got %T", key, v)
	}
	return i, true, nil
}

func getFloat32(settings map[string]any, key string) (float32, bool, error) {
	v, ok := settings[key]
	if !ok {
		return 0, false, nil
	}
	f, ok := v.(float32)
	if !ok {
		return 0, true, fmt.Errorf("%s must be float32, got %T", key, v)
	}
	return f, true, nil
}

func getExtras(settings map[string]any, key string) (map[string]any, bool, error) {
	v, ok := settings[key]
	if !ok {
		return nil, false, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, true, fmt.Errorf("%s must be map[string]any, got %T", key, v)
	}
	return m, true, nil
}
