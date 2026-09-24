package session_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/session"
	"reflect"
	"testing"
	"time"
)

func newSession(t *testing.T, msgs ...agent.Message) session.Session {
	t.Helper()

	now := time.Now()
	header := session.NewHeader("test-session", 0, 0, now, now, map[string]any{})
	return session.NewRestoredSession(header, msgs)
}

func contentTexts(parts []agent.ContentPart) []string {
	texts := make([]string, len(parts))
	for i, part := range parts {
		texts[i] = part.Text
	}
	return texts
}

func newToolCall(id string) *agent.ToolCall {
	return agent.NewToolCall(id, "test-tool", agent.ToolArguments(`{}`))
}

func TestAddMessages_MergesConsecutiveUserMessages(t *testing.T) {
	s := newSession(t, agent.NewUserMessage("hello"))

	s.AddMessages(agent.NewUserMessage("world"))

	messages := s.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	got, ok := messages[0].(*agent.UserMessage)
	if !ok {
		t.Fatalf("expected *agent.UserMessage, got %T", messages[0])
	}
	want := []string{"hello", "world"}
	if !reflect.DeepEqual(contentTexts(got.Content()), want) {
		t.Fatalf("unexpected content %v, want %v", contentTexts(got.Content()), want)
	}
}

func TestAddMessages_MergesConsecutiveAgentMessages(t *testing.T) {
	oldCall := newToolCall("old-call")
	newCall := newToolCall("new-call")
	s := newSession(t, agent.NewAgentMessage("old", []*agent.ToolCall{oldCall}))

	s.AddMessages(agent.NewAgentMessage("new", []*agent.ToolCall{newCall}))

	messages := s.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	got, ok := messages[0].(*agent.AgentMessage)
	if !ok {
		t.Fatalf("expected *agent.AgentMessage, got %T", messages[0])
	}
	wantContent := []string{"old", "new"}
	if !reflect.DeepEqual(contentTexts(got.Content()), wantContent) {
		t.Fatalf("unexpected content %v, want %v", contentTexts(got.Content()), wantContent)
	}
	wantCalls := []*agent.ToolCall{oldCall, newCall}
	if !reflect.DeepEqual(got.ToolCalls(), wantCalls) {
		t.Fatalf("unexpected tool calls %v, want %v", got.ToolCalls(), wantCalls)
	}
}

func TestAddMessages_NoMergeOnDifferentRoles(t *testing.T) {
	agentMsg := agent.NewAgentMessage("answer", nil)
	userMsg := agent.NewUserMessage("follow-up")

	userThenAgent := newSession(t, agent.NewUserMessage("hello"))
	userThenAgent.AddMessages(agentMsg)
	if got := len(userThenAgent.Messages()); got != 2 {
		t.Fatalf("expected 2 messages after user+agent, got %d", got)
	}
	if userThenAgent.Messages()[1] != agentMsg {
		t.Fatalf("expected agent message appended untouched")
	}

	agentThenUser := newSession(t, agentMsg)
	agentThenUser.AddMessages(userMsg)
	if got := len(agentThenUser.Messages()); got != 2 {
		t.Fatalf("expected 2 messages after agent+user, got %d", got)
	}
	if agentThenUser.Messages()[1] != userMsg {
		t.Fatalf("expected user message appended untouched")
	}
}

func TestAddMessages_NoopWhenEmpty(t *testing.T) {
	s := newSession(t, agent.NewUserMessage("hello"))

	s.AddMessages()

	if got := len(s.Messages()); got != 1 {
		t.Fatalf("expected 1 message, got %d", got)
	}
}

func TestAddMessages_ColdStartEmptySession(t *testing.T) {
	s := session.NewSession("test-session")

	s.AddMessages(agent.NewUserMessage("hello"))

	messages := s.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	got, ok := messages[0].(*agent.UserMessage)
	if !ok {
		t.Fatalf("expected *agent.UserMessage, got %T", messages[0])
	}
	if want := []string{"hello"}; !reflect.DeepEqual(contentTexts(got.Content()), want) {
		t.Fatalf("unexpected content %v, want %v", contentTexts(got.Content()), want)
	}
}

func TestApplyCompletion_AppendsAgentMessage(t *testing.T) {
	call := newToolCall("call-1")
	s := newSession(t, agent.NewUserMessage("hello"))

	s.ApplyCompletion(&agent.Completion{
		ToolCalls: []*agent.ToolCall{call},
		Content:   "answer",
	})

	messages := s.Messages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	got, ok := messages[1].(*agent.AgentMessage)
	if !ok {
		t.Fatalf("expected *agent.AgentMessage, got %T", messages[1])
	}
	if want := []string{"answer"}; !reflect.DeepEqual(contentTexts(got.Content()), want) {
		t.Fatalf("unexpected content %v, want %v", contentTexts(got.Content()), want)
	}
	if !reflect.DeepEqual(got.ToolCalls(), []*agent.ToolCall{call}) {
		t.Fatalf("unexpected tool calls %v", got.ToolCalls())
	}
}

func TestApplyCompletion_UpdatesTokens(t *testing.T) {
	s := newSession(t, agent.NewUserMessage("hello"))

	s.ApplyCompletion(&agent.Completion{
		Content:          "answer",
		InputTokens:      100,
		CompletionTokens: 42,
	})

	if got := s.InputTokens(); got != 100 {
		t.Fatalf("expected 100 input tokens, got %d", got)
	}
	if got := s.OutputTokens(); got != 42 {
		t.Fatalf("expected 42 output tokens, got %d", got)
	}
}

func TestOverwriteMessages_ReplacesMessagesAndResetsTokens(t *testing.T) {
	newMsgs := []agent.Message{agent.NewUserMessage("new")}
	s := newSession(t, agent.NewUserMessage("old"))
	s.ApplyCompletion(&agent.Completion{Content: "answer", InputTokens: 5, CompletionTokens: 7})

	s.OverwriteMessages(10, newMsgs)

	if !reflect.DeepEqual(s.Messages(), newMsgs) {
		t.Fatalf("unexpected messages %v, want %v", s.Messages(), newMsgs)
	}
	if got := s.InputTokens(); got != 10 {
		t.Fatalf("expected 10 input tokens, got %d", got)
	}
	if got := s.OutputTokens(); got != 0 {
		t.Fatalf("expected 0 output tokens, got %d", got)
	}
}

func TestOverwriteMessages_EmptyMessages(t *testing.T) {
	s := newSession(t, agent.NewUserMessage("old"))

	s.OverwriteMessages(0, []agent.Message{})

	if got := len(s.Messages()); got != 0 {
		t.Fatalf("expected 0 messages, got %d", got)
	}
}

func TestGetLastUserMessage_ReturnsLastUserMessage(t *testing.T) {
	user1 := agent.NewUserMessage("first")
	user2 := agent.NewUserMessage("second")
	s := newSession(t, agent.NewAgentMessage("answer", nil), user1, agent.NewAgentMessage("answer", nil), user2)

	got := s.GetLastUserMessage()
	if got != user2 {
		t.Fatalf("expected last user message, got %#v", got)
	}
}

func TestGetLastUserMessage_ReturnsNilWhenNone(t *testing.T) {
	s := newSession(t, agent.NewAgentMessage("answer", nil))

	if got := s.GetLastUserMessage(); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestGetLastAgentMessage_ReturnsLastAgentMessage(t *testing.T) {
	agent1 := agent.NewAgentMessage("answer 1", nil)
	agent2 := agent.NewAgentMessage("answer 2", nil)
	s := newSession(t, agent.NewUserMessage("hello"), agent1, agent.NewUserMessage("follow-up"), agent2)

	got := s.GetLastAgentMessage()
	if got != agent2 {
		t.Fatalf("expected last agent message, got %#v", got)
	}
}

func TestGetLastAgentMessage_ReturnsNilWhenNone(t *testing.T) {
	s := newSession(t, agent.NewUserMessage("hello"))

	if got := s.GetLastAgentMessage(); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}
