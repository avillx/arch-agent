package runtime_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/runtime"
	"context"
	"errors"
	"testing"
)

func TestRunAgentLoop_CompactsOnOverflow(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{err: runtime.ErrContextOverflow},
			{completion: &agent.Completion{Content: "compacted summary"}},
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	msgs := []agent.Message{
		agent.NewSystemMessage("system"),
		agent.NewUserMessage("turn 1"),
		agent.NewUserMessage("turn 2"),
		agent.NewUserMessage("turn 3"),
	}

	events := runLoop(t, context.Background(), model, msgs, nil, nil)

	compaction, ok := findEvent[*runtime.CompactionEvent](events)
	if !ok {
		t.Fatalf("expected compaction event, got %#v", events)
	}
	if compaction.Summary() != "compacted summary" {
		t.Fatalf("unexpected summary: %q", compaction.Summary())
	}

	compacted := compaction.CompactedContext()
	if len(compacted) == 0 {
		t.Fatal("expected non-empty compacted context")
	}
	if system, ok := compacted[0].(*agent.SystemMessage); !ok || system.Content()[0].Text != "system" {
		t.Fatalf("expected system message to be preserved first, got %#v", compacted[0])
	}

	calls := model.Calls()
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 model calls, got %d", len(calls))
	}

	compactionCall := calls[1]
	if len(compactionCall) == 0 {
		t.Fatal("expected compaction call to carry messages")
	}
	system, ok := compactionCall[0].(*agent.SystemMessage)
	if !ok || system.Content()[0].Text != prompt.Compaction() {
		t.Fatalf("expected compaction prompt as system message, got %#v", compactionCall[0])
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected clean exit after compaction, got %#v", events)
	}
}

func TestRunAgentLoop_CompactsOnThreshold(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100,
		responses: []mockCompletion{
			{completion: &agent.Completion{
				Done:             false,
				InputTokens:      95,
				CompletionTokens: 10,
			}},
			{completion: &agent.Completion{Content: "compacted summary"}},
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewSystemMessage("system"), agent.NewUserMessage("hello")},
		nil,
		nil,
	)

	compaction, ok := findEvent[*runtime.CompactionEvent](events)
	if !ok {
		t.Fatalf("expected compaction event, got %#v", events)
	}
	if compaction.Summary() != "compacted summary" {
		t.Fatalf("unexpected summary: %q", compaction.Summary())
	}

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok || exit.Err() != nil {
		t.Fatalf("expected clean exit, got %#v", events)
	}
}

func TestRunAgentLoop_CompactionFailureExits(t *testing.T) {
	model := &mockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []mockCompletion{
			{err: runtime.ErrContextOverflow},
			{err: context.DeadlineExceeded},
		},
	}

	events := runLoop(
		t,
		context.Background(),
		model,
		[]agent.Message{agent.NewSystemMessage("system"), agent.NewUserMessage("hello")},
		nil,
		nil,
	)

	exit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if exit.Err() == nil || !errors.Is(exit.Err(), context.DeadlineExceeded) {
		t.Fatalf("expected unhandled overflow error, got %v", exit.Err())
	}
}
