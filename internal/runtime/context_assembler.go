package runtime

import (
	"arch-agent/internal/agent"
	"fmt"
	"strings"
)

// type contextAssembler struct {
// }

// <skills>
//   <skill name="search">... description ...</skill>
//   <skill name="code">...</skill>
// </skills>

func assembeSystemMessage(agt agent.Agent, toolKit []agent.Tool) *agent.SystemMessage {

	systemPrompt := agt.SystemPrompt()
	instructions := extractInstructions(agt, toolKit)

	assembled := strings.Join(
		[]string{
			systemPrompt,
			instructions,
		}, "\n")

	return agent.NewSystemMessage(assembled)
}

type Instructed interface {
	Instruction() string
}

type PerAgentInstructed interface {
	AgentInstruction(agent.Agent) string
}

func extractInstructions(agt agent.Agent, toolKit []agent.Tool) string {
	var sb strings.Builder

	for _, t := range toolKit {
		if instructedTool, ok := t.(Instructed); ok {
			fmt.Fprintf(&sb, "%s\n\n", instructedTool.Instruction())
		}
		if agentInstructedTool, ok := t.(PerAgentInstructed); ok {
			fmt.Fprintf(&sb, "%s\n\n", agentInstructedTool.AgentInstruction(agt))
		}
	}

	return sb.String()
}
