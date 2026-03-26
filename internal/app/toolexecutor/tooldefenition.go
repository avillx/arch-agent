package tools

type ToolDefinition struct {
	Name       string
	ReasonOnce bool
	Schema     Schema
}

type Schema struct {
	Description string
	Properties  []ToolProperty
}

type PropertyType string

const (
	TypeString  PropertyType = "string"
	TypeNumber  PropertyType = "number"
	TypeBoolean PropertyType = "boolean"
)

type ToolProperty struct {
	Required    bool
	Type        PropertyType
	Description string
	Enum        []string
}
