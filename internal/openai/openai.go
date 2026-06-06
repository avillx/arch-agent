package openai

import (
	"arch-agent/internal/agent"
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
func (f *openAIFactory) Produce(settings agent.ModelSettings) (agent.Model, error) {
	return NewOpenAIReasoner(f.secrets, settings)
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
		assistantMsg := openai.AssistantMessage(msg.Content())
		toolCalls := msg.ToolCalls()
		if len(toolCalls) > 0 {
			assistantMsg.OfAssistant.ToolCalls = toolCallsToOpenAi(toolCalls)
		}
		result = assistantMsg
	case *agent.SystemMessage:
		result = openai.SystemMessage(msg.Content())
	case *agent.UserMessage:
		result = openai.UserMessage(msg.Content())
	case *agent.ToolResultMessage:
		result = openai.ToolMessage(msg.Content(), msg.ToolCallID())
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

	result := &agent.Completion{
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
	switch n := v.(type) {
	case int:
		return n, true, nil
	case float64:
		return int(n), true, nil
	}
	return 0, true, fmt.Errorf("%s must be int, got %T", key, v)
}

func getInt64(settings map[string]any, key string) (int64, bool, error) {
	v, ok := settings[key]
	if !ok {
		return 0, false, nil
	}
	switch n := v.(type) {
	case int64:
		return n, true, nil
	case float64:
		return int64(n), true, nil
	}
	return 0, true, fmt.Errorf("%s must be int64, got %T", key, v)
}

func getFloat32(settings map[string]any, key string) (float32, bool, error) {
	v, ok := settings[key]
	if !ok {
		return 0, false, nil
	}
	switch n := v.(type) {
	case float32:
		return n, true, nil
	case float64:
		return float32(n), true, nil
	}
	return 0, true, fmt.Errorf("%s must be float32, got %T", key, v)
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
