package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const cacheLifeTime = 30 * time.Minute

type SystemMessageCache struct {
	messages     map[sessionKey]string
	deleteTimers map[sessionKey]*time.Timer

	mu sync.Mutex
}

func NewSystemMessageCache() *SystemMessageCache {
	return &SystemMessageCache{
		messages:     map[sessionKey]string{},
		deleteTimers: map[sessionKey]*time.Timer{},
	}
}

func (c *SystemMessageCache) Put(key sessionKey, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// try to delete
	delete(c.messages, key)
	if t, ok := c.deleteTimers[key]; ok {
		t.Stop()
		delete(c.deleteTimers, key)
	}

	c.messages[key] = message
	c.deleteTimers[key] = time.AfterFunc(cacheLifeTime, func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		delete(c.messages, key)
		delete(c.deleteTimers, key)
	})
}

func (c *SystemMessageCache) Get(key sessionKey) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	msg, ok := c.messages[key]

	if ok {
		c.deleteTimers[key].Reset(cacheLifeTime)
	}

	return msg, ok
}

type SystemMessageParticipant interface {
	Part(ctx context.Context) (string, error)
}

type ContextAssembler struct {
	skillRepo  SkillsRepo
	memoryRepo MemoryRepo
	cache      *SystemMessageCache
}

func NewContextAssembler(
	skillRepo SkillsRepo,
	memoryRepo MemoryRepo,
) *ContextAssembler {
	return &ContextAssembler{
		skillRepo:  skillRepo,
		memoryRepo: memoryRepo,
		cache:      NewSystemMessageCache(),
	}
}

func (a *ContextAssembler) BuildSystemMessage(
	ctx context.Context,
	agt agent.Agent,
	toolServers []agent.ToolServer,
	sess session.Session,
) (*agent.SystemMessage, error) {

	key := sessionKey{
		AgentID:   agt.ID(),
		SessionID: sess.ID(),
	}

	if cached, hit := a.cache.Get(key); hit {
		return agent.NewSystemMessage(cached), nil
	}

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

	a.cache.Put(key, text)
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
		fmt.Fprintf(&sb, "- (%s) %s\n", k, v)
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
		fmt.Fprintf(&sb, "- (%s) %s\n", p, desc)
	}

	return prompt.SkillGuidance(sb.String()), nil
}

// tool aware behaviour
var _ SystemMessageParticipant = (*ToolAwarePart)(nil)

type ToolInstructer interface {
	Instruction() string
}

type PerAgentToolInstructer interface {
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
		if instructedTool, ok := t.(ToolInstructer); ok {
			instructions = append(instructions, instructedTool.Instruction())
		}
		if agentInstructedTool, ok := t.(PerAgentToolInstructer); ok {
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
	if len(extras) <= 0 {
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
