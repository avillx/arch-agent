package answer_test

import (
	"arch-agent/internal/app/answer"
	executioncontext "arch-agent/internal/app/executioncontext"
	tools "arch-agent/internal/app/toolexecutor"
	"arch-agent/internal/domain/conversation"
	"context"
	"errors"
	"testing"
)

// ─── Stubs & Spies ────────────────────────────────────────────────────────────

type stubTokenizer struct{ perCall int }

func (s *stubTokenizer) Calc(_ string) int { return s.perCall }

// ---

type spyConversationRepo struct {
	conv           *conversation.Conversation
	savedMessages  []conversation.Message
	optimizeCalled bool
}

func newConvRepo(tokenCostPerMessage int) *spyConversationRepo {
	conv := conversation.NewConversation(&stubTokenizer{perCall: tokenCostPerMessage}, nil)
	return &spyConversationRepo{conv: conv}
}

func newConvRepoWithHistory(msgs []conversation.Message) *spyConversationRepo {
	conv := conversation.NewConversation(&stubTokenizer{}, msgs)
	return &spyConversationRepo{conv: conv}
}

func (r *spyConversationRepo) Get() *conversation.Conversation  { return r.conv }
func (r *spyConversationRepo) Save(msgs []conversation.Message) { r.savedMessages = msgs }
func (r *spyConversationRepo) Optimize()                        { r.optimizeCalled = true }

// ---

type stubAgentRepo struct{}

func (r *stubAgentRepo) Get() executioncontext.AgentConfig { return executioncontext.AgentConfig{} }

// ---

type stubMemoryProvider struct{}

func (p *stubMemoryProvider) Snapshot(_ context.Context, _ []conversation.Message) executioncontext.Memory {
	return executioncontext.Memory{}
}

// ---

type stubReflector struct{}

func (r *stubReflector) Reflect(_ context.Context, _ []conversation.Message, _ string) executioncontext.Reflection {
	return executioncontext.Reflection{}
}

// ---

type spyContentReceiver struct {
	sent []string
	err  error
}

func (r *spyContentReceiver) Send(_ context.Context, content string) error {
	r.sent = append(r.sent, content)
	return r.err
}

// ---

type stubToolReceiver struct {
	result string
	err    error
}

func (r *stubToolReceiver) SendCall(_ context.Context, _ string, _ conversation.ToolArguments) (string, error) {
	return r.result, r.err
}

// ---

// stubReasoner returns responses in sequence; repeats the last one when exhausted.
type stubReasoner struct {
	responses []answer.ReasonResult
	callCount int
	err       error
}

func (r *stubReasoner) Reason(_ context.Context, _ executioncontext.ReasonParams) (answer.ReasonResult, error) {
	if r.err != nil {
		return answer.ReasonResult{}, r.err
	}
	idx := r.callCount
	if idx >= len(r.responses) {
		idx = len(r.responses) - 1
	}
	r.callCount++
	return r.responses[idx], nil
}

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func buildUseCase(reasoner answer.Reasoner, convRepo answer.ConversationRespository) *answer.AnswerUseCase {
	factory := executioncontext.NewRequestContextFactory(&stubMemoryProvider{}, &stubReflector{})
	return answer.NewAnswerUseCase(reasoner, convRepo, &stubAgentRepo{}, factory)
}

func makeToolCallReciver(result string, err error) *stubToolReceiver {
	return &stubToolReceiver{result: result, err: err}
}

func okToolCallReciver() *stubToolReceiver { return makeToolCallReciver("ok", nil) }

// toolDef creates a ToolDefinition. reasonOnce=true means no follow-up after call.
func toolDef(name string, reasonOnce bool) tools.ToolDefinition {
	return tools.ToolDefinition{Name: name, ReasonOnce: reasonOnce}
}

// toolCall dereferences *ToolCall → ToolCall (value) as expected by ReasonResult.
func toolCall(name string) conversation.ToolCall {
	return *conversation.NewToolCall("call-id", name, nil)
}

func noToolCalls(content string) answer.ReasonResult {
	return answer.NewReasonResult(nil, content)
}

func withToolCall(name, content string) answer.ReasonResult {
	return answer.NewReasonResult([]conversation.ToolCall{toolCall(name)}, content)
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// 1. Happy path ─────────────────────────────────────────────────────────────

func TestHappyPath_SingleReason_NoToolCalls(t *testing.T) {
	convRepo := newConvRepo(0)
	cr := &spyContentReceiver{}
	reasoner := &stubReasoner{responses: []answer.ReasonResult{noToolCalls("hello")}}

	err := buildUseCase(reasoner, convRepo).Execute(
		context.Background(), cr, okToolCallReciver(),
		answer.NewAnswerCommand("hi", nil),
	)

	assertNoError(t, err)
	assertEqual(t, 1, len(cr.sent), "Send call count")
	assertEqual(t, "hello", cr.sent[0], "Send content")
	assertNotNil(t, convRepo.savedMessages, "Save must be called")
}

// 2. Agentic loop ───────────────────────────────────────────────────────────

func TestAgenticLoop_TwoIterations(t *testing.T) {
	const tool = "search"
	convRepo := newConvRepo(0)
	reasoner := &stubReasoner{responses: []answer.ReasonResult{
		withToolCall(tool, "thinking..."),
		noToolCalls("done"),
	}}

	err := buildUseCase(reasoner, convRepo).Execute(
		context.Background(), &spyContentReceiver{},
		makeToolCallReciver("result", nil),
		answer.NewAnswerCommand("hi", []tools.ToolDefinition{toolDef(tool, false)}),
	)

	assertNoError(t, err)
	assertEqual(t, 2, reasoner.callCount, "reasoner call count")
}

// 3. Loop termination ───────────────────────────────────────────────────────

// ReasonOnce=true → tool doesn't require a follow-up reason, loop stops after 1 iteration.
func TestLoop_StopsWhenReasonOnce(t *testing.T) {
	const tool = "oneshot"
	reasoner := &stubReasoner{responses: []answer.ReasonResult{withToolCall(tool, "")}}

	err := buildUseCase(reasoner, newConvRepo(0)).Execute(
		context.Background(), &spyContentReceiver{}, okToolCallReciver(),
		answer.NewAnswerCommand("hi", []tools.ToolDefinition{toolDef(tool, true)}),
	)

	assertNoError(t, err)
	assertEqual(t, 1, reasoner.callCount, "reasoner call count")
}

// When follow-up budget runs out the loop must terminate — no infinite loop.
func TestLoop_BudgetExhausted(t *testing.T) {
	const tool = "search"
	reasoner := &stubReasoner{
		responses: []answer.ReasonResult{withToolCall(tool, "")},
	}

	err := buildUseCase(reasoner, newConvRepo(0)).Execute(
		context.Background(), &spyContentReceiver{},
		makeToolCallReciver("ok", nil),
		answer.NewAnswerCommand("hi", []tools.ToolDefinition{toolDef(tool, false)}),
	)

	assertNoError(t, err)
	assertEqual(t, executioncontext.DefaultFollowUpBudget, reasoner.callCount, "reasoner calls == budget")
}

// Context timeout interrupts the loop but Save is still called.
func TestLoop_ContextTimeout_SaveStillCalled(t *testing.T) {
	convRepo := newConvRepo(0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already done

	err := buildUseCase(&stubReasoner{}, convRepo).Execute(
		ctx, &spyContentReceiver{}, okToolCallReciver(),
		answer.NewAnswerCommand("hi", nil),
	)

	if err == nil {
		t.Error("expected timeout error")
	}
	assertNotNil(t, convRepo.savedMessages, "Save must be called even on timeout")
}

// 4. Errors ─────────────────────────────────────────────────────────────────

// Reasoner error is fatal — Save must NOT be called (messages would be silently lost).
func TestError_ReasonerFails_SaveNotCalled(t *testing.T) {
	convRepo := newConvRepo(0)
	reasoner := &stubReasoner{err: errors.New("llm down")}

	err := buildUseCase(reasoner, convRepo).Execute(
		context.Background(), &spyContentReceiver{}, okToolCallReciver(),
		answer.NewAnswerCommand("hi", nil),
	)

	if err == nil {
		t.Error("expected error")
	}
	if convRepo.savedMessages != nil {
		t.Error("Save must NOT be called when reasoner fails — messages would be lost")
	}
}

func TestError_ContentReceiverFails_ReturnsError(t *testing.T) {
	reasoner := &stubReasoner{responses: []answer.ReasonResult{noToolCalls("hi")}}
	cr := &spyContentReceiver{err: errors.New("send failed")}

	err := buildUseCase(reasoner, newConvRepo(0)).Execute(
		context.Background(), cr, okToolCallReciver(),
		answer.NewAnswerCommand("hi", nil),
	)

	if err == nil {
		t.Error("expected error from content receiver")
	}
}

// Executor error is non-fatal: joined into result, loop still continues.
func TestError_ExecutorFails_ErrorJoinedLoopContinues(t *testing.T) {
	const tool = "search"
	reasoner := &stubReasoner{responses: []answer.ReasonResult{
		withToolCall(tool, ""),
		noToolCalls("done"),
	}}

	err := buildUseCase(reasoner, newConvRepo(0)).Execute(
		context.Background(), &spyContentReceiver{},
		makeToolCallReciver("", errors.New("tool failed")),
		answer.NewAnswerCommand("hi", []tools.ToolDefinition{toolDef(tool, false)}),
	)

	if err == nil {
		t.Error("expected joined error from executor")
	}
	// Loop continued despite the error — both iterations completed.
	assertEqual(t, 2, reasoner.callCount, "loop must continue after executor error")
}

// 5. State persistence ──────────────────────────────────────────────────────

// Save receives only new messages — historical ones must not be duplicated.
func TestSave_ReceivesOnlyNewMessages(t *testing.T) {
	historical := conversation.NewUserMessage("old msg")
	convRepo := newConvRepoWithHistory([]conversation.Message{historical})
	reasoner := &stubReasoner{responses: []answer.ReasonResult{noToolCalls("new response")}}

	err := buildUseCase(reasoner, convRepo).Execute(
		context.Background(), &spyContentReceiver{}, okToolCallReciver(),
		answer.NewAnswerCommand("new msg", nil),
	)

	assertNoError(t, err)
	for _, m := range convRepo.savedMessages {
		if m.Content() == "old msg" {
			t.Error("historical message must not appear in Save payload")
		}
	}
}

func TestOptimize_CalledOnTokenOverflow(t *testing.T) {
	// Each Calc() call returns TokenLimit → first message triggers overflow.
	convRepo := newConvRepo(conversation.TokenLimit)
	reasoner := &stubReasoner{responses: []answer.ReasonResult{noToolCalls("hi")}}

	err := buildUseCase(reasoner, convRepo).Execute(
		context.Background(), &spyContentReceiver{}, okToolCallReciver(),
		answer.NewAnswerCommand("hi", nil),
	)

	assertNoError(t, err)
	if !convRepo.optimizeCalled {
		t.Error("Optimize must be called when tokens exceed limit")
	}
}

func TestOptimize_NotCalledWhenWithinLimit(t *testing.T) {
	convRepo := newConvRepo(0) // zero cost — never overflows
	reasoner := &stubReasoner{responses: []answer.ReasonResult{noToolCalls("hi")}}

	err := buildUseCase(reasoner, convRepo).Execute(
		context.Background(), &spyContentReceiver{}, okToolCallReciver(),
		answer.NewAnswerCommand("hi", nil),
	)

	assertNoError(t, err)
	if convRepo.optimizeCalled {
		t.Error("Optimize must NOT be called when tokens are within limit")
	}
}

// ─── Assert Helpers ───────────────────────────────────────────────────────────

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertEqual[T comparable](t *testing.T, want, got T, label string) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %v, got %v", label, want, got)
	}
}

func assertNotNil(t *testing.T, v any, label string) {
	t.Helper()
	if v == nil {
		t.Errorf("%s: expected non-nil", label)
	}
}
