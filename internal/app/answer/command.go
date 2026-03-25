package answer

import tools "arch-agent/internal/app/toolexecutor"

type AnswerCommand struct {
	Content  string
	toolDefs []tools.ToolDefinition
}

func NewAnswerCommand(content string, toolDefs []tools.ToolDefinition) *AnswerCommand {
	return &AnswerCommand{
		Content:  content,
		toolDefs: toolDefs,
	}
}
