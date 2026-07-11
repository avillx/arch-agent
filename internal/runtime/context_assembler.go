package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
)

type SkillRepo interface {
	GetSkills(agentID agent.ID) ([]agent.SkillFrontmatter, error)
}

type MemoryIndexer interface {
	GetMemoryIndex(agent.ID) (string, error)
}

type ContextAssembler struct {
	indexer       SkillRepo
	activityRepo  agent.ActivityRepo
	memoryIndexer MemoryIndexer
}

func NewContextAssembler(
	indexer SkillRepo,
	activityRepo agent.ActivityRepo,
	memoryIndexer MemoryIndexer,
) *ContextAssembler {
	return &ContextAssembler{
		indexer:       indexer,
		activityRepo:  activityRepo,
		memoryIndexer: memoryIndexer,
	}
}

func (a *ContextAssembler) buildContext(
	sess session.Session,
	agt agent.Agent,
	tools []agent.Tool,
	model agent.Model,
) []agent.Message {

	// build system message
	contextMessages := []agent.Message{
		a.assembeSystemMessage(agt, sess, tools),
	}

	// resolve precontext hooks
	preContextMessages := a.resolvePreContextMessages(agt, sess)
	if len(preContextMessages) > 0 {
		contextMessages = append(contextMessages, preContextMessages...)
	}

	distillMessages := eliminateOldImages(sess.Messages())

	contextMessages = append(contextMessages, distillMessages...)

	return excludeUnsupportedModalities(contextMessages, model.SupportedModalities())
}

func (a *ContextAssembler) assembeSystemMessage(agt agent.Agent, sess session.Session, toolKit []agent.Tool) *agent.SystemMessage {

	completionContext := []string{agt.SystemPrompt()}

	// instructions
	if instructions := toolInstructions(agt, toolKit); len(instructions) > 0 {
		completionContext = append(completionContext, instructions...)
	}

	// Skills
	skills, err := a.indexer.GetSkills(agt.ID())
	if err != nil {
		slog.Error("skill load", "error", err)
	}
	if len(skills) > 0 {
		completionContext = append(completionContext, prompt.SkillGuidance(buildSkillIndex(skills)))
	}

	// Memory
	if agt.HasMemory() {
		// load persistent memory prompt
		idx, err := a.memoryIndexer.GetMemoryIndex(agt.ID())
		if err != nil {
			slog.Error("memory index is not reached", "error", err)
		} else {
			activity := a.resolveActivity(agt, sess)
			if activity == "" {
				activity = "you has no activity at last 24h"
			}
			activity = strings.TrimSuffix(activity, "\n")
			completionContext = append(completionContext, prompt.PersistentMemory(agt.ID(), idx, activity))
		}
	}

	assembled := strings.Join(completionContext, "\n\n")
	return agent.NewSystemMessage(assembled)
}

func (a *ContextAssembler) resolvePreContextMessages(_ agent.Agent, sess session.Session) []agent.Message {

	summary := sess.Summary()
	if summary == "" {
		return nil
	}

	return []agent.Message{
		agent.NewUserMessage(prompt.SummaryExplanation(summary)),
		agent.NewAgentMessage("okay i will account it", nil),
	}
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
		if !errors.Is(err, types.ErrIsNotExist) {
			slog.Error("failed to get activity", "error", err, "agent", agt.ID())
		}
		return ""
	}

	activity = truncateActivity(activity)

	extras[activityStorageKey] = activity
	sess.SetExtras(extras)

	return activity
}

func buildSkillIndex(skillsFronts []agent.SkillFrontmatter) string {

	var sb strings.Builder
	for _, skill := range skillsFronts {
		fmt.Fprintf(&sb, "[%s](%s) - %s\n", skill.ID, skill.StoreHint, skill.Description)
	}

	return sb.String()
}

type Instructed interface {
	Instruction() string
}

type PerAgentInstructed interface {
	AgentInstruction(agent.Agent) string
}

func toolInstructions(agt agent.Agent, toolKit []agent.Tool) []string {

	instructions := []string{}

	if len(toolKit) > 0 {
		instructions = append(instructions, prompt.ToolUsageGuide())
	}

	for _, t := range toolKit {
		if instructedTool, ok := t.(Instructed); ok {
			instructions = append(instructions, instructedTool.Instruction())
		}
		if agentInstructedTool, ok := t.(PerAgentInstructed); ok {
			instructions = append(instructions, agentInstructedTool.AgentInstruction(agt))
		}
	}

	return instructions
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

func excludeUnsupportedModalities(msgs []agent.Message, mdls []agent.Modality) []agent.Message {
	var distill []agent.Message

	if !slices.Contains(mdls, agent.ImageModality) {
		for i, m := range msgs {

			var shouldReplaceMsg bool

			contentParts := m.Content()
			for contentPartIdx := range contentParts {

				if contentParts[contentPartIdx].ImageURL != "" {

					if !shouldReplaceMsg {
						shouldReplaceMsg = true
						contentParts = slices.Clone(contentParts)
					}

					contentParts[contentPartIdx].ImageURL = ""
					contentParts[contentPartIdx].Text += prompt.ExcludedUnsupportedModality(agent.ImageModality)
				}

			}

			if shouldReplaceMsg {
				if distill == nil {
					distill = slices.Clone(msgs)
				}

				distill[i] = agent.CloneMessage(msgs[i])
				distill[i].SetContent(contentParts)
			}
		}
	}

	if distill == nil {
		return msgs
	}

	return distill
}
