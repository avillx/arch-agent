package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type SkillIndexer interface {
	GetIndex() map[agent.SkillID]agent.Skill
}

type MemoryIndexer interface {
	GetMemoryIndex(agent.ID) (string, error)
}

type ContextAssembler struct {
	indexer       SkillIndexer
	activityRepo  agent.ActivityRepo
	memoryIndexer MemoryIndexer
}

func NewContextAssembler(
	indexer SkillIndexer,
	activityRepo agent.ActivityRepo,
	memoryIndexer MemoryIndexer,
) *ContextAssembler {
	return &ContextAssembler{
		indexer:       indexer,
		activityRepo:  activityRepo,
		memoryIndexer: memoryIndexer,
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

	if agt.HasMemory() {
		completionContext = append(completionContext, prompt.MemoryHeaderPrompt())
		completionContext = append(completionContext, prompt.EpisodicMemoryPrompt())

		// load persistent memory prompt
		idx, err := a.memoryIndexer.GetMemoryIndex(agt.ID())
		if err != nil {
			slog.Error("memory index is not reached", "error", err)
		} else {
			completionContext = append(completionContext, prompt.PersistentMemoryPrompt(idx))
		}
	}

	assembled := strings.Join(completionContext, "\n")
	return agent.NewSystemMessage(assembled)
}

func (a *ContextAssembler) resolvePreContextMessages(agt agent.Agent, sess session.Session) []agent.Message {
	var msgs []string

	if summary := sess.Summary(); summary != "" {
		msgs = append(msgs, prompt.SummaryExplanation(summary))
	}

	if agt.HasMemory() {
		if activity := a.resolveActivity(agt, sess); activity != "" {
			msgs = append(msgs, prompt.ActivityExplanation(activity))
		}
	}

	if len(msgs) == 0 {
		return nil
	}

	return preContextHookDialogue(strings.Join(msgs, "\n"))
}

const activityStorageKey = "activity"

func (a *ContextAssembler) resolveActivity(agt agent.Agent, sess session.Session) string {
	extras := sess.Extras()
	if extras == nil {
		return ""
	}

	// cache hit
	if cached, ok := extras[activityStorageKey].(string); ok {
		return cached
	}

	// cache miss
	activity, err := a.activityRepo.GetActivity(agt.ID(), time.Now())
	if err != nil {
		if !errors.Is(err, agent.ErrNoActivity) {
			slog.Error("failed to get activity", "error", err, "agent", agt.ID())
		}
		return ""
	}

	activity = truncateActivity(activity)

	extras[activityStorageKey] = activity
	sess.SetExtras(extras)

	return activity
}

func buildBoundedSkillIndex(idx map[agent.SkillID]agent.Skill, agt agent.Agent) string {

	var sb strings.Builder
	for _, skillID := range agt.Skills() {
		skill, ok := idx[skillID]
		if !ok {
			slog.Warn("agent has non existing skill", "agent", agt.ID(), "skill", skill)
			continue
		}
		fmt.Fprintf(&sb, "\n* %s - %s\n", skillID, skill.Description())
	}

	return sb.String()
}

func preContextHookDialogue(instructions string) []agent.Message {
	return []agent.Message{
		agent.NewUserMessage(prompt.SummaryExplanation(instructions)),
		agent.NewAgentMessage("okay i will account it", nil),
	}
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

const ActivityLinesLimit = 70

func truncateActivity(s string) string {
	normal := strings.ReplaceAll(s, "\r\n", "\n")
	normal = strings.ReplaceAll(normal, "\r", "\n")

	lines := strings.Split(normal, "\n")
	keep := min(len(lines), ActivityLinesLimit)
	start := len(lines) - keep

	return strings.Join(lines[start:], "\n")
}
