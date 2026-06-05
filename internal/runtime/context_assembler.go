package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"fmt"
	"log/slog"
	"strings"
)

type SkillIndexer interface {
	GetIndex() map[agent.SkillID]agent.Skill
}

type ContextAssembler struct {
	indexer SkillIndexer
}

func NewContextAssembler(indexer SkillIndexer) *ContextAssembler {
	return &ContextAssembler{
		indexer: indexer,
	}
}

func (a *ContextAssembler) assembeSystemMessage(agt agent.Agent, toolKit []agent.Tool) *agent.SystemMessage {

	systemPrompt := agt.SystemPrompt()
	instructions := extractInstructions(agt, toolKit)

	completionContext := []string{
		systemPrompt,
		instructions,
	}

	if len(agt.Skills()) > 0 {
		idx := buildBoundedSkillIndex(a.indexer.GetIndex(), agt)
		skillGuidance := prompt.SkillGuidance(idx)
		completionContext = append(completionContext, skillGuidance)
	}

	assembled := strings.Join(completionContext, "\n")
	return agent.NewSystemMessage(assembled)
}

func buildBoundedSkillIndex(idx map[agent.SkillID]agent.Skill, agt agent.Agent) string {

	var sb strings.Builder
	for _, skillID := range agt.Skills() {
		skill, ok := idx[skillID]
		if !ok {
			slog.Warn("agent has non existing skill", "agent", agt.ID(), "skill", skill)
			continue
		}
		fmt.Fprintf(&sb, "* %s - %s\n", skillID, skill.Description())
	}

	return sb.String()
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
