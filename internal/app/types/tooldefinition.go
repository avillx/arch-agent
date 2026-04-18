package types

type ToolDefinition struct {
	Name        string
	Description string
	Properties  []ToolProperty
}

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
