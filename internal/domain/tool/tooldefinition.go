package tool

import (
	"context"
)

type ToolArguments []byte
type Meta map[string]any

type PropertyType string

const (
	TypeString  PropertyType = "string"
	TypeNumber  PropertyType = "integer"
	TypeBoolean PropertyType = "boolean"
)

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
