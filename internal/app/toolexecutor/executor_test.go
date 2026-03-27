package tools_test

import (
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/domain/conversation"
	"context"
	"errors"
	"testing"
)

// stubReceiver — заглушка ToolCallReceiver
type stubReceiver struct {
	fn    func(toolName string, args conversation.ToolArguments) (string, error)
	tools []tools.ToolDefinition
}

func (s *stubReceiver) SendCall(_ context.Context, toolName string, args conversation.ToolArguments) (string, error) {
	return s.fn(toolName, args)
}

func (s *stubReceiver) Tools() []tools.ToolDefinition {
	return s.tools
}

func okReceiver(result string) *stubReceiver {
	return &stubReceiver{
		fn: func(_ string, _ conversation.ToolArguments) (string, error) {
			return result, nil
		},
	}
}

func errReceiver(err error) *stubReceiver {
	return &stubReceiver{
		fn: func(_ string, _ conversation.ToolArguments) (string, error) {
			return "", err
		},
	}
}

// helpers
func makeCall(id, toolName string) *conversation.ToolCall {
	return conversation.NewToolCall(id, toolName, nil)
}

func findByID(results []conversation.ToolCallResult, id string) (conversation.ToolCallResult, bool) {
	for _, r := range results {
		if r.ID == id {
			return r, true
		}
	}
	return conversation.ToolCallResult{}, false
}

// --- тесты ---

func TestExecute_Empty_NoErrorNoResults(t *testing.T) {
	e := tools.NewExecutor(okReceiver("anything"))

	results, err := e.Execute(context.Background(), nil)

	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestExecute_SingleCall_ReturnsCorrectResult(t *testing.T) {
	e := tools.NewExecutor(okReceiver("pong"))
	calls := []*conversation.ToolCall{makeCall("id-1", "ping")}

	results, err := e.Execute(context.Background(), calls)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "id-1" {
		t.Errorf("expected ID id-1, got %s", results[0].ID)
	}
	if results[0].Result != "pong" {
		t.Errorf("expected result pong, got %s", results[0].Result)
	}
}

func TestExecute_MultipleCalls_AllResultsCollected(t *testing.T) {
	e := tools.NewExecutor(okReceiver("ok"))
	calls := []*conversation.ToolCall{
		makeCall("id-1", "tool-a"),
		makeCall("id-2", "tool-b"),
		makeCall("id-3", "tool-c"),
	}

	results, err := e.Execute(context.Background(), calls)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, call := range calls {
		if _, found := findByID(results, call.ID()); !found {
			t.Errorf("result with ID %s not found", call.ID())
		}
	}
}

func TestExecute_ParallelCalls_NoResultsLost(t *testing.T) {
	e := tools.NewExecutor(okReceiver("ok"))

	calls := make([]*conversation.ToolCall, 50)
	for i := range calls {
		calls[i] = makeCall(string(rune('A'+i%26))+string(rune('0'+i%10)), "tool")
	}

	results, err := e.Execute(context.Background(), calls)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != len(calls) {
		t.Errorf("expected %d results, got %d", len(calls), len(results))
	}
}

func TestExecute_OneCallFails_ReturnsError(t *testing.T) {
	boom := errors.New("boom")
	callCount := 0
	receiver := &stubReceiver{
		fn: func(toolName string, _ conversation.ToolArguments) (string, error) {
			callCount++
			if toolName == "bad-tool" {
				return "", boom
			}
			return "ok", nil
		},
	}
	e := tools.NewExecutor(receiver)
	calls := []*conversation.ToolCall{
		makeCall("id-1", "good-tool"),
		makeCall("id-2", "bad-tool"),
	}

	_, err := e.Execute(context.Background(), calls)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("expected boom error, got: %v", err)
	}
}

func TestExecute_AllCallsFail_ReturnsError(t *testing.T) {
	boom := errors.New("all broken")
	e := tools.NewExecutor(errReceiver(boom))
	calls := []*conversation.ToolCall{
		makeCall("id-1", "tool-a"),
		makeCall("id-2", "tool-b"),
	}

	_, err := e.Execute(context.Background(), calls)

	if err == nil {
		t.Error("expected error, got nil")
	}
}
