package openai

import "arch-agent/internal/agent"

type OpenAIModelSettings struct {
	Model               *string          `json:"model"`
	ToolChoice          *string          `json:"tool_choice,omitempty"`
	ReasoningEffort     *string          `json:"reasoning_effort,omitempty"`
	ContextLimit        *int64           `json:"context_limit,omitempty"`
	MaxOutputTokens     *int64           `json:"max_output_tokens,omitempty"`
	MaxCompletionTokens *int64           `json:"max_completion_tokens,omitempty"`
	Temperature         *float32         `json:"temperature,omitempty"`
	TopP                *float32         `json:"top_p,omitempty"`
	FrequencyPenalty    *float32         `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float32         `json:"presence_penalty,omitempty"`
	RecallBudget        *int             `json:"recall_budget,omitempty"`
	Modalities          []agent.Modality `json:"modalities,omitempty"`
	Extras              map[string]any   `json:"extras,omitempty"`
}
