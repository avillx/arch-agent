package chat_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/session"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockMemoryRepo implements chat.MemoryRepo.
type mockMemoryRepo struct {
	idx map[string]string
	err error
}

func (m *mockMemoryRepo) MemoryIndex(agent.ID) (map[string]string, error) {
	return m.idx, m.err
}

// mockSkillsRepo implements chat.SkillsRepo.
type mockSkillsRepo struct {
	skills map[string]string
}

func (m *mockSkillsRepo) Skills(agent.ID) (map[string]string, error) {
	return m.skills, nil
}

// plain tool server without any instruction interfaces
type plainToolServer struct{}

func (p *plainToolServer) Tools() []agent.Tool { return nil }

// instructedToolServer implements chat.ToolInstructer.
type instructedToolServer struct {
	instruction string
}

func (s *instructedToolServer) Tools() []agent.Tool { return nil }
func (s *instructedToolServer) Instruction() string { return s.instruction }

// agentInstructedToolServer implements chat.PerAgentToolInstructer.
type agentInstructedToolServer struct {
	instruction string
}

func (s *agentInstructedToolServer) Tools() []agent.Tool { return nil }
func (s *agentInstructedToolServer) AgentInstruction(agent.Agent) string {
	return s.instruction
}

// bothInstructedToolServer implements chat.ToolInstructer and chat.PerAgentToolInstructer.
type bothInstructedToolServer struct {
	instruction      string
	agentInstruction string
}

func (s *bothInstructedToolServer) Tools() []agent.Tool { return nil }
func (s *bothInstructedToolServer) Instruction() string { return s.instruction }
func (s *bothInstructedToolServer) AgentInstruction(agent.Agent) string {
	return s.agentInstruction
}

func newAssemblerAgent(prompt string) agent.Agent {
	return agent.NewAgent("agent-1", "test agent", prompt, "model-1", nil, false)
}

func newAssemblerSession(extras map[string]any) session.Session {
	if extras == nil {
		extras = map[string]any{}
	}
	return session.NewRestoredSession(
		session.NewHeader("sess-1", 0, 0, time.Now(), time.Now(), extras),
		nil,
	)
}

func buildSystemMessage(
	t *testing.T,
	memoryRepo chat.MemoryRepo,
	skillRepo chat.SkillsRepo,
	agt agent.Agent,
	toolServers []agent.ToolServer,
	sess session.Session,
) (string, error) {
	t.Helper()

	assembler := chat.NewContextAssembler(skillRepo, memoryRepo)
	msg, err := assembler.BuildSystemMessage(context.Background(), agt, toolServers, sess)
	if err != nil {
		return "", err
	}

	content := msg.Content()
	if len(content) != 1 {
		t.Fatalf("expected single content part, got %d", len(content))
	}
	return content[0].Text, nil
}

func requireContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q to contain %q", text, want)
	}
}

func requireNotContains(t *testing.T, text, want string) {
	t.Helper()
	if strings.Contains(text, want) {
		t.Fatalf("expected %q to not contain %q", text, want)
	}
}

func requireInOrder(t *testing.T, text string, markers ...string) {
	t.Helper()

	last := -1
	for _, marker := range markers {
		idx := strings.Index(text, marker)
		if idx < 0 {
			t.Fatalf("expected %q to contain %q", text, marker)
		}
		if idx <= last {
			t.Fatalf("expected %q to appear after the previously found marker in %q", marker, text)
		}
		last = idx
	}
}

func TestBuildSystemMessage_HappyPath(t *testing.T) {
	const agentPrompt = "you are a test agent"
	const memoryOnePath = "/memory/tasks.md"
	const memoryTwoPath = "/memory/notes.md"
	const skillOnePath = "/skills/py/SKILL.md"
	const skillTwoPath = "/skills/sh/SKILL.md"
	const toolInstruction = "use shell carefully"
	const agentInstruction = "call other agents when needed"
	const sessionInstruction = "answer in russian"

	memoryRepo := &mockMemoryRepo{idx: map[string]string{
		memoryOnePath: "task memory",
		memoryTwoPath: "notes memory",
	}}
	skillRepo := &mockSkillsRepo{skills: map[string]string{
		skillOnePath: "python skill",
		skillTwoPath: "shell skill",
	}}

	toolServers := []agent.ToolServer{
		&plainToolServer{},
		&instructedToolServer{instruction: toolInstruction},
		&agentInstructedToolServer{instruction: agentInstruction},
	}

	text, err := buildSystemMessage(
		t,
		memoryRepo,
		skillRepo,
		newAssemblerAgent(agentPrompt),
		toolServers,
		newAssemblerSession(map[string]any{"instruction": sessionInstruction}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// every part contributes its content
	for _, want := range []string{
		agentPrompt,
		memoryOnePath,
		memoryTwoPath,
		skillOnePath,
		skillTwoPath,
		toolInstruction,
		agentInstruction,
		sessionInstruction,
	} {
		requireContains(t, text, want)
	}

	// parts keep the assembler order
	requireInOrder(
		t,
		text,
		agentPrompt,
		memoryOnePath,
		skillOnePath,
		toolInstruction,
		agentInstruction,
		sessionInstruction,
	)
}

func TestAgentPart_IncludesCustomSystemPrompt(t *testing.T) {
	const prompt = "custom system prompt"

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(prompt), nil,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, text, prompt)
}

func TestAgentPart_FallsBackToDefaultPrompt(t *testing.T) {
	const prompt = "custom system prompt"

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if text == "" {
		t.Fatal("expected non-empty default prompt")
	}
	requireNotContains(t, text, prompt)
}

func TestMemoryPart_IncludesMemoryIndex(t *testing.T) {
	const path = "/memory/notes.md"
	const desc = "notes about the project"

	repo := &mockMemoryRepo{idx: map[string]string{path: desc}}

	text, err := buildSystemMessage(t, repo, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, text, "- ("+path+") "+desc)
}

func TestMemoryPart_SkipsEmptyIndex(t *testing.T) {
	const path = "/memory/notes.md"

	repo := &mockMemoryRepo{idx: map[string]string{}}

	text, err := buildSystemMessage(t, repo, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireNotContains(t, text, path)
}

func TestMemoryPart_ErrorWhenIndexNotReached(t *testing.T) {
	repo := &mockMemoryRepo{err: errors.New("memory repo is down")}

	_, err := buildSystemMessage(t, repo, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(nil))
	if err == nil {
		t.Fatal("expected error when memory index is not reached")
	}
}

func TestSkillPart_IncludesSkillIndex(t *testing.T) {
	const path = "/skills/py/SKILL.md"
	const desc = "python coding skill"

	repo := &mockSkillsRepo{skills: map[string]string{path: desc}}

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, repo,
		newAssemblerAgent(""), nil,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, text, "- ("+path+") "+desc)
}

func TestSkillPart_SkipsEmptyIndex(t *testing.T) {
	const path = "/skills/py/SKILL.md"

	repo := &mockSkillsRepo{skills: map[string]string{}}

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, repo,
		newAssemblerAgent(""), nil,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireNotContains(t, text, path)
}

func TestToolAwarePart_IncludesToolInstructions(t *testing.T) {
	const toolInstruction = "shell tool instruction"
	const agentInstruction = "call agent instruction"

	toolServers := []agent.ToolServer{
		&instructedToolServer{instruction: toolInstruction},
		&agentInstructedToolServer{instruction: agentInstruction},
	}

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), toolServers,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, text, toolInstruction)
	requireContains(t, text, agentInstruction)
}

func TestToolAwarePart_IncludesBothInstructionsFromOneServer(t *testing.T) {
	const toolInstruction = "fs instruction"
	const agentInstruction = "fs agent instruction"

	toolServers := []agent.ToolServer{
		&bothInstructedToolServer{instruction: toolInstruction, agentInstruction: agentInstruction},
	}

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), toolServers,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, text, toolInstruction)
	requireContains(t, text, agentInstruction)
}

func TestToolAwarePart_IgnoresServersWithoutInstructions(t *testing.T) {
	const toolInstruction = "shell tool instruction"

	toolServers := []agent.ToolServer{
		&plainToolServer{},
		&instructedToolServer{instruction: toolInstruction},
	}

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), toolServers,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, text, toolInstruction)
}

func TestToolAwarePart_SkipsWhenNoTools(t *testing.T) {
	const toolInstruction = "shell tool instruction"
	const agentInstruction = "call agent instruction"

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireNotContains(t, text, toolInstruction)
	requireNotContains(t, text, agentInstruction)
}

func TestSessionPart_IncludesInstruction(t *testing.T) {
	const instruction = "always answer in english"

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(map[string]any{"instruction": instruction}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, text, instruction)
}

func TestSessionPart_SkipsWithoutInstruction(t *testing.T) {
	const instruction = "always answer in english"

	text, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(map[string]any{"other": "no instruction here"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireNotContains(t, text, instruction)
}

func TestSessionPart_ErrorOnBadInstructionType(t *testing.T) {
	_, err := buildSystemMessage(t, &mockMemoryRepo{}, &mockSkillsRepo{},
		newAssemblerAgent(""), nil,
		newAssemblerSession(map[string]any{"instruction": 42}))
	if err == nil {
		t.Fatal("expected error when session instruction is not a string")
	}
}
