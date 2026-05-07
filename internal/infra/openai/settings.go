package openaiadapter

type SecretsStorage interface {
	Get(name string) (string, bool)
}

type SettingsRepo interface {
	OnChange(f func(t LLMSettings))
	Value() LLMSettings
}

type LLMSettings struct {
	Client ClientSettings `json:"client"`
	Params ModelSettings  `json:"params"`
}

type ModelSettings struct {
	Model               string          `json:"model,omitempty"`
	MaxOutputTokens     *int64          `json:"max_output_tokens"`
	MaxCompletionTokens *int64          `json:"max_completion_tokens,omitempty"`
	ToolChoice          *string         `json:"tool_choice,omitempty"`
	ReasoningEffort     *string         `json:"reasoning_effort,omitempty"`
	Temperature         *float32        `json:"temperature,omitempty"`
	FrequencyPenalty    *float32        `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float32        `json:"presence_penalty,omitempty"`
	TopP                *float32        `json:"top_p,omitempty"`
	Extras              *map[string]any `json:"extras,omitempty"`
}

type ClientSettings struct {
	OpenAIURL  string `json:"openai_api_url"`
	APIKeyName string `json:"api_key_name"`
}
