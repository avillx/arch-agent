package conversation_test

import (
	"arch-agent/internal/domain/conversation"
	"testing"
)

// lenTokenizer считает токены как длину строки
type lenTokenizer struct{}

func (t lenTokenizer) Calc(s string) int {
	return len(s)
}

// newConversation — хелпер: создаёт конверс с уже существующими сообщениями
func newConversation(existing []conversation.Message) *conversation.Conversation {
	return conversation.NewConversation(lenTokenizer{}, existing)
}

// --- Messages() ---

func TestMessages_Empty(t *testing.T) {
	c := newConversation(nil)

	msgs := c.Messages()

	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestMessages_OnlyNew(t *testing.T) {
	c := newConversation(nil)
	c.AddUserMessage("hello")

	msgs := c.Messages()

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content() != "hello" {
		t.Errorf("unexpected content: %s", msgs[0].Content())
	}
}

func TestMessages_ExistingAndNew_OrderPreserved(t *testing.T) {
	existing := []conversation.Message{
		conversation.NewUserMessage("first"),
	}
	c := newConversation(existing)
	c.AddUserMessage("second")

	msgs := c.Messages()

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content() != "first" || msgs[1].Content() != "second" {
		t.Errorf("wrong order: %v", msgs)
	}
}

// --- NewMessages() ---

func TestNewMessages_ReturnsOnlyNew(t *testing.T) {
	existing := []conversation.Message{
		conversation.NewUserMessage("old"),
	}
	c := newConversation(existing)
	c.AddUserMessage("new")

	newMsgs := c.NewMessages()

	if len(newMsgs) != 1 {
		t.Fatalf("expected 1 new message, got %d", len(newMsgs))
	}
	if newMsgs[0].Content() != "new" {
		t.Errorf("unexpected content: %s", newMsgs[0].Content())
	}
}

func TestNewMessages_Empty_WhenNothingAdded(t *testing.T) {
	existing := []conversation.Message{
		conversation.NewUserMessage("old"),
	}
	c := newConversation(existing)

	newMsgs := c.NewMessages()

	if len(newMsgs) != 0 {
		t.Errorf("expected 0 new messages, got %d", len(newMsgs))
	}
}

// --- IsOverflow() ---

func TestIsOverflow_False_WhenUnderLimit(t *testing.T) {
	c := newConversation(nil)
	c.AddUserMessage("short")

	if c.IsOverflow() {
		t.Error("expected no overflow")
	}
}

func TestIsOverflow_True_WhenAtLimit(t *testing.T) {
	c := newConversation(nil)
	// lenTokenizer считает байты, TokenLimit = 20000
	c.AddUserMessage(makeString(conversation.TokenLimit))

	if !c.IsOverflow() {
		t.Error("expected overflow at limit")
	}
}

func TestIsOverflow_True_WhenOverLimit(t *testing.T) {
	c := newConversation(nil)
	c.AddUserMessage(makeString(conversation.TokenLimit + 1))

	if !c.IsOverflow() {
		t.Error("expected overflow above limit")
	}
}

// makeString создаёт строку заданной длины
func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}