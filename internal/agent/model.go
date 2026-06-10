package agent

import (
	"context"
)

type ModelID string

type ModelSettings map[string]any

type Modality string

const (
	TextModality  Modality = "text"
	ImageModality Modality = "image"
)

type Model interface {
	Settings() ModelSettings
	ContextLimit() int64
	SetSettings(ModelSettings) error
	Complete(context.Context, []Tool, []Message) (*Completion, error)
	SupportedModalities() []Modality
}

type ModelRepository interface {
	Get(ModelID) (Model, error)
	Delete(ModelID) error
	Save(ModelID, Model) error
}

type Completion struct {
	ToolCalls        []*ToolCall
	Content          string
	Done             bool
	InputTokens      int64
	CompletionTokens int64
}
