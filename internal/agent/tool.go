package agent

import (
	"context"
)

type ToolName string

type PropertyType string

const (
	TypeString  PropertyType = "string"
	TypeNumber  PropertyType = "integer"
	TypeBoolean PropertyType = "boolean"
)

type ToolRegistry interface {
	GetServerTools([]string) ([]Tool, error)
}

type ToolProperty struct {
	Name        string
	Description string
	Required    bool
	IsArray     bool
	Type        PropertyType
	Enum        []string
}

type Tool interface {
	Name() ToolName
	Description() string
	Schema() any
	Call(context.Context, ToolArguments) ([]ContentPart, error)
}
