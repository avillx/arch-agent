package agent

import (
	"context"
)

type ModelID string

type ModelSettings map[string]any

type Model interface {
	Settings() ModelSettings
	ContextLimit() int
	SetSettings(ModelSettings) error
	Complete(context.Context, []Tool, []Message) (*Completion, error)
}

type ModelRepository interface {
	Get(ModelID) (Model, error)
	Delete(ModelID) error
	Save(ModelID, Model) error
}

type Completion struct {
	ToolCalls []*ToolCall
	Content   string
	Done      bool
}
