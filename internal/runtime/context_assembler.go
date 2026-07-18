package runtime

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"slices"
	"strings"
	"time"
)

type ContextAssembler struct {
	indexer       agent.SkillRepo
	activityRepo  agent.ActivityRepo
	memoryIndexer agent.MemoryIndexer
}

func NewContextAssembler(
	indexer agent.SkillRepo,
	activityRepo agent.ActivityRepo,
	memoryIndexer agent.MemoryIndexer,
) *ContextAssembler {
	return &ContextAssembler{
		indexer:       indexer,
		activityRepo:  activityRepo,
		memoryIndexer: memoryIndexer,
	}
}

type BuildContextRequest struct {
	IncludeMemory       bool
	IncludeSkills       bool
	AddInstuctions      bool
	AllowOptimizeImages bool
	Additional          string
}

func (a *ContextAssembler) buildContext(
	sess session.Session,
	agt agent.Agent,
	toolServers []agent.ToolServer,
	model agent.Model,
	req BuildContextRequest,
) []agent.Message {

	// system prompt
	completionContext := []string{agt.SystemPrompt()}

	// instructions
	if req.AddInstuctions {
		if instructions := toolInstructions(agt, toolServers); len(instructions) > 0 {
			completionContext = append(completionContext, instructions...)
		}
	}

	// Skills
	if req.IncludeSkills {
		completionContext = append(completionContext, a.buildSkillGuidance(agt.ID()))
	}

	// Memory
	if req.IncludeMemory && agt.HasMemory() {
		completionContext = append(completionContext, a.buildMemory(agt.ID(), sess))
	}

	// Additional
	if req.Additional != "" {
		completionContext = append(completionContext, req.Additional)
	}

	// Context
	contextMessages := []agent.Message{
		agent.NewSystemMessage(strings.Join(completionContext, "\n\n")),
	}

	// slog.Debug("system message", "message", contextMessages)

	conversationMessages := sess.Messages()

	// optimize messsages
	if req.AllowOptimizeImages && resolveImageOptimize(model) {
		conversationMessages = eliminateOldImages(conversationMessages)
	}

	conversationMessages = excludeUnsupportedModalities(conversationMessages, model.SupportedModalities())

	contextMessages = append(contextMessages, conversationMessages...)

	return contextMessages
}

func (a *ContextAssembler) buildSkillGuidance(agentID agent.ID) string {
	skills, err := a.indexer.GetSkills(agentID)
	if err != nil {
		slog.Error("skill load", "error", err)
	}
	if !(len(skills) > 0) {
		return ""
	}

	return prompt.SkillGuidance(buildSkillIndex(skills))
}

func (a *ContextAssembler) buildMemory(agentID agent.ID, sess session.Session) string {
	// load persistent memory prompt
	idx, err := a.memoryIndexer.MemoryIndex(agentID)
	if err != nil {
		if joinedErrs, ok := err.(interface{ Unwrap() []error }); ok {
			errs := joinedErrs.Unwrap()
			for _, e := range errs {
				slog.Error("memory index", "agent", agentID, "error", e)
			}
		} else {
			slog.Error("memory index is not reached", "agent", agentID, "error", err)
		}
	}

	var sb strings.Builder
	for k, v := range idx {
		fmt.Fprintf(&sb, "(%s)[%s] - %s\n", strings.TrimSuffix(path.Base(k), path.Ext(k)), k, v)
	}

	activity := a.resolveActivity(agentID, sess)
	if activity == "" {
		activity = "you has no activity at last 24h"
	}
	activity = strings.TrimSuffix(activity, "\n")
	return prompt.PersistentMemory(agentID, sb.String(), activity)
}

const activityStorageKey = "activity"

func (a *ContextAssembler) resolveActivity(agentID agent.ID, sess session.Session) string {
	extras := sess.Extras()
	if extras == nil {
		return ""
	}

	// cache hit
	if cached, ok := extras[activityStorageKey].(string); ok {
		return cached
	}

	// cache miss
	activity, err := a.activityRepo.GetActivity(agentID, time.Now())
	if err != nil {
		if !errors.Is(err, types.ErrIsNotExist) {
			slog.Error("failed to get activity", "error", err, "agent", agentID)
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

func toolInstructions(agt agent.Agent, toolServers []agent.ToolServer) []string {

	instructions := []string{}

	if len(toolServers) > 0 {
		instructions = append(instructions, prompt.ToolUsageGuide())
	}

	for _, t := range toolServers {
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

func eliminateOldImages(messages []agent.Message) []agent.Message {
	result := make([]agent.Message, len(messages))
	copy(result, messages)

	cutoff := len(messages) - 10
	for i := 0; i < cutoff; i++ {
		content := result[i].Content()
		newContent := make([]agent.ContentPart, len(content))
		copy(newContent, content)
		for j := range newContent {
			newContent[j].ImageURL = ""
		}
		result[i].SetContent(newContent)
	}
	return result
}

func resolveImageOptimize(m agent.Model) bool {
	if optimize, ok := m.Settings()["optimize_images"]; ok {
		optimizeTyped, ok := optimize.(bool)
		if ok {
			return optimizeTyped
		}
		slog.Error(
			"model has wrong 'image optimize' field.",
			"error",
			fmt.Errorf("want bool, has %T", optimizeTyped),
		)
	}
	return false
}
