package agent

import (
	"context"
)

type PropertyType string

const (
	TypeString  PropertyType = "string"
	TypeNumber  PropertyType = "integer"
	TypeBoolean PropertyType = "boolean"
)

type ToolRegistry interface {
	GetTools([]string) ([]Tool, error)
}

type ToolProperty struct {
	Name        string
	Required    bool
	Type        PropertyType
	Description string
	Enum        []string
}

type Tool interface {
	Name() string
	Description() string
	Schema() []ToolProperty
	Call(context.Context, ToolArguments) (string, error)
}
