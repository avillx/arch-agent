package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
)

type SystemMessageParticipant interface {
	Part(ctx context.Context) (string, error)
}

type ContextAssembler struct {
	skillRepo  SkillsRepo
	memoryRepo MemoryRepo
}

func NewContextAssembler(
	skillRepo SkillsRepo,
	memoryRepo MemoryRepo,
) *ContextAssembler {
	return &ContextAssembler{
		skillRepo:  skillRepo,
		memoryRepo: memoryRepo,
	}
}

func (a *ContextAssembler) BuildSystemMessage(
	ctx context.Context,
	agt agent.Agent,
	toolServers []agent.ToolServer,
	sess session.Session,
) (*agent.SystemMessage, error) {

	parts := []SystemMessageParticipant{
		&AgentPart{agt: agt},
		&MemoryPart{agentID: agt.ID(), repo: a.memoryRepo},
		&SkillPart{agentID: agt.ID(), repo: a.skillRepo},
		&ToolAwarePart{agt: agt, toolServers: toolServers},
		&SessionPart{sess: sess},
	}

	text, err := AssembleParts(ctx, parts)
	if err != nil {
		return nil, err
	}

	return agent.NewSystemMessage(text), nil
}

func AssembleParts(ctx context.Context, parts []SystemMessageParticipant) (string, error) {
	var systemMessage strings.Builder
	for _, p := range parts {
		text, err := p.Part(ctx)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&systemMessage, "%s\n\n", text)
	}
	return systemMessage.String(), nil
}

// agent system Prompt
var _ SystemMessageParticipant = (*AgentPart)(nil)

type AgentPart struct {
	agt agent.Agent
}

func (i *AgentPart) Part(_ context.Context) (string, error) {
	if systemPrompt := i.agt.SystemPrompt(); systemPrompt != "" {
		return systemPrompt, nil
	}
	return prompt.DefaultAgent(), nil
}

// memory index
var _ SystemMessageParticipant = (*MemoryPart)(nil)

type MemoryRepo interface {
	// expected map[/path/to/memory_file]memory_description
	MemoryIndex(agent.ID) (map[string]string, error)
}

type MemoryPart struct {
	agentID agent.ID
	repo    MemoryRepo
}

func (i *MemoryPart) Part(ctx context.Context) (string, error) {

	// load persistent memory prompt
	idx, err := i.repo.MemoryIndex(i.agentID)
	if err != nil {
		return "", fmt.Errorf("agent %s memory index is not reached: %w", i.agentID, err)
	}

	// has no memories, no need to guide it
	if len(idx) <= 0 {
		return "", nil
	}

	var sb strings.Builder
	for k, v := range idx {
		fileName := strings.TrimSuffix(path.Base(k), path.Ext(k))
		fmt.Fprintf(&sb, "(%s)[%s] - %s\n", fileName, k, v)
	}

	return prompt.PersistentMemory(i.agentID, sb.String(), ""), nil
}

// skill index
var _ SystemMessageParticipant = (*SkillPart)(nil)

type SkillsRepo interface {
	// expected: map[/path/to/skill]skill_description
	Skills(agent.ID) (map[string]string, error)
}

type SkillPart struct {
	agentID agent.ID
	repo    SkillsRepo
}

func (i *SkillPart) Part(ctx context.Context) (string, error) {
	skills, err := i.repo.Skills(i.agentID)
	if err != nil {
		slog.Error("skill load", "error", err)
	}

	// has no skills no need to guide it
	if len(skills) <= 0 {
		return "", nil
	}

	// build index
	var sb strings.Builder
	for p, desc := range skills {
		fmt.Fprintf(&sb, "- (%s): %s\n", p, desc)
	}

	return prompt.SkillGuidance(sb.String()), nil
}

// tool aware behaviour
var _ SystemMessageParticipant = (*ToolAwarePart)(nil)

type Instructed interface {
	Instruction() string
}

type PerAgentInstructed interface {
	AgentInstruction(agent.Agent) string
}

type ToolAwarePart struct {
	agt         agent.Agent
	toolServers []agent.ToolServer
}

func (i *ToolAwarePart) Part(ctx context.Context) (string, error) {

	// if has no tools no need to guide it
	if len(i.toolServers) <= 0 {
		return "", nil
	}

	instructions := []string{}

	if len(i.toolServers) > 0 {
		instructions = append(instructions, prompt.ToolUsageGuide())
	}

	for _, t := range i.toolServers {
		if instructedTool, ok := t.(Instructed); ok {
			instructions = append(instructions, instructedTool.Instruction())
		}
		if agentInstructedTool, ok := t.(PerAgentInstructed); ok {
			instructions = append(instructions, agentInstructedTool.AgentInstruction(i.agt))
		}
	}

	return strings.Join(instructions, "\n\n"), nil
}

// session additional context
var _ SystemMessageParticipant = (*SessionPart)(nil)

type SessionPart struct {
	sess session.Session
}

func (i *SessionPart) Part(ctx context.Context) (string, error) {
	extras := i.sess.Extras()
	if len(extras) > 0 {
		return "", nil
	}

	instruction, ok := extras["instruction"]
	if !ok {
		return "", nil
	}

	stringInstruction, ok := instruction.(string)
	if !ok {
		return "", fmt.Errorf("session instruction: bad type: want string, have: %T", instruction)
	}

	return stringInstruction, nil
}
