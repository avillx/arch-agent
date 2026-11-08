package agent

import (
	"context"
)

type ToolName string

type ToolRegistry interface {
	ToolServers(...string) ([]ToolServer, error)
}

type ToolServer interface {
	Tools() []Tool
}

type Tool interface {
	Name() ToolName
	Description() string
	Schema() map[string]any
	Call(context.Context, ToolArguments) ([]ContentPart, error)
}
