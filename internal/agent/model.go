package agent

import (
	"context"
)

type ModelName string

type ModelSettings map[string]any

type Modality string

const (
	TextModality  Modality = "text"
	ImageModality Modality = "image"
)

type Model interface {
	Settings() ModelSettings
	Complete(context.Context, []Tool, []Message) (*Completion, error)
	ContextLimit() int64
	SupportedModalities() []Modality
}

type ModelRepository interface {
	Get(ModelName) (Model, error)
	Delete(ModelName) error
	Save(ModelName, Model) error
}

type Completion struct {
	ToolCalls        []*ToolCall
	Content          string
	Done             bool
	InputTokens      int64
	CompletionTokens int64
}
