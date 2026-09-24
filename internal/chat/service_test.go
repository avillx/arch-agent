package chat_test

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// mockAgentRepo implements agent.Repo.
type mockAgentRepo struct {
	getFn func(agent.ID) (agent.Agent, error)
}

func (m *mockAgentRepo) All() ([]agent.Agent, error) { return nil, nil }
func (m *mockAgentRepo) Get(id agent.ID) (agent.Agent, error) {
	if m == nil || m.getFn == nil {
		return nil, fmt.Errorf("agent %s is not exist", id)
	}
	return m.getFn(id)
}
func (m *mockAgentRepo) Save(agent.Agent) error { return nil }
func (m *mockAgentRepo) Delete(agent.ID) error  { return nil }

// mockSessionsRepo implements session.SessionsRepo.
type mockSessionsRepo struct {
	sess    session.Session
	sessErr error
	saveFn  func(agent.ID, session.Session) error

	saved        bool
	savedAgentID agent.ID
	savedSession session.Session
}

func (m *mockSessionsRepo) Session(agent.ID, session.ID) (session.Session, error) {
	return m.sess, m.sessErr
}

func (m *mockSessionsRepo) Save(agentID agent.ID, s session.Session) error {
	m.saved = true
	m.savedAgentID = agentID
	m.savedSession = s
	if m.saveFn != nil {
		return m.saveFn(agentID, s)
	}
	return nil
}

func (m *mockSessionsRepo) Delete(agent.ID, session.ID) error                 { return nil }
func (m *mockSessionsRepo) Headers(agent.ID) ([]session.SessionHeader, error) { return nil, nil }

// mockModelRegistry implements agent.ModelRegistry.
type mockModelRegistry struct {
	model agent.Model
	err   error
}

func (m *mockModelRegistry) Get(string) (agent.Model, error) {
	return m.model, m.err
}

// mockToolRegistry implements agent.ToolRegistry.
type mockToolRegistry struct {
	toolServersFn func(...string) ([]agent.ToolServer, error)
}

func (m *mockToolRegistry) ToolServers(names ...string) ([]agent.ToolServer, error) {
	if m == nil || m.toolServersFn == nil {
		return nil, nil
	}
	return m.toolServersFn(names...)
}

// mockSystemMessageBuilder implements chat.SystemMessageBuilder.
type mockSystemMessageBuilder struct {
	msg *agent.SystemMessage
	err error
}

func (m *mockSystemMessageBuilder) BuildSystemMessage(
	context.Context,
	agent.Agent,
	[]agent.ToolServer,
	session.Session,
) (*agent.SystemMessage, error) {
	if m == nil {
		return agent.NewSystemMessage("system"), nil
	}
	return m.msg, m.err
}

// mockActivityLogger implements chat.ActivityLogger.
type activityCommit struct {
	agentID   agent.ID
	sessionID session.ID
	msgs      []agent.Message
}

type mockActivityLogger struct {
	commits []activityCommit
}

func (m *mockActivityLogger) Commit(agentID agent.ID, sessionID session.ID, msgs []agent.Message) {
	m.commits = append(m.commits, activityCommit{
		agentID:   agentID,
		sessionID: sessionID,
		msgs:      msgs,
	})
}

// mockModel implements agent.Model.
type chatMockCompletion struct {
	completion *agent.Completion
	err        error
}

type chatMockModel struct {
	settings   agent.ModelSettings
	ctxLimit   int64
	responses  []chatMockCompletion
	idx        int
	completeFn func(context.Context, []agent.Tool, []agent.Message) (*agent.Completion, error)

	lastCtx      context.Context
	lastMessages []agent.Message
}

func (m *chatMockModel) Settings() agent.ModelSettings { return m.settings }
func (m *chatMockModel) ContextLimit() int64           { return m.ctxLimit }
func (m *chatMockModel) SupportedModalities() []agent.Modality {
	return []agent.Modality{agent.TextModality}
}
func (m *chatMockModel) Complete(ctx context.Context, tools []agent.Tool, msgs []agent.Message) (*agent.Completion, error) {
	m.lastCtx = ctx
	m.lastMessages = msgs

	if m.completeFn != nil {
		return m.completeFn(ctx, tools, msgs)
	}

	if m.idx >= len(m.responses) {
		return &agent.Completion{Done: true}, nil
	}

	resp := m.responses[m.idx]
	m.idx++
	return resp.completion, resp.err
}

// dispatcherModel implements agent.Model for dispatcher tests, where two requests
// run concurrently; it keeps no mutable state.
type dispatcherModel struct {
	completeFn func(context.Context, []agent.Tool, []agent.Message) (*agent.Completion, error)
}

func (m *dispatcherModel) Settings() agent.ModelSettings { return agent.ModelSettings{} }
func (m *dispatcherModel) ContextLimit() int64           { return 100_000 }
func (m *dispatcherModel) SupportedModalities() []agent.Modality {
	return []agent.Modality{agent.TextModality}
}
func (m *dispatcherModel) Complete(ctx context.Context, tools []agent.Tool, msgs []agent.Message) (*agent.Completion, error) {
	return m.completeFn(ctx, tools, msgs)
}

// mockTool implements agent.Tool.
type mockTool struct {
	name   string
	callFn func(context.Context, agent.ToolArguments) ([]agent.ContentPart, error)
}

func (m *mockTool) Name() agent.ToolName { return agent.ToolName(m.name) }
func (m *mockTool) Description() string  { return m.name }
func (m *mockTool) Schema() any          { return map[string]any{} }
func (m *mockTool) Call(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {
	if m.callFn != nil {
		return m.callFn(ctx, args)
	}
	return agent.NewContent("ok"), nil
}

// mockToolServer implements agent.ToolServer.
type mockToolServer struct {
	tools []agent.Tool
}

func (m *mockToolServer) Tools() []agent.Tool {
	return m.tools
}

func newTestSession(id session.ID) session.Session {
	return session.NewRestoredSession(
		session.NewHeader(id, 0, 0, time.Now(), time.Now(), map[string]any{}),
		[]agent.Message{agent.NewSystemMessage("system")},
	)
}

func newMockAgent(id agent.ID, toolServers ...string) agent.Agent {
	return agent.NewAgent(id, "desc", "system", "model-1", toolServers, false)
}

func newTestService(
	t *testing.T,
	agentRepo agent.Repo,
	sessionRepo session.SessionsRepo,
	modelRepo agent.ModelRegistry,
	toolRegistry agent.ToolRegistry,
	builder chat.SystemMessageBuilder,
	activity chat.ActivityLogger,
) *chat.Service {
	t.Helper()

	if sessionRepo == nil {
		sessionRepo = &mockSessionsRepo{sess: newTestSession("sess-1")}
	}
	if builder == nil {
		builder = &mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")}
	}
	if activity == nil {
		activity = &mockActivityLogger{}
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return chat.NewService(agentRepo, sessionRepo, modelRepo, toolRegistry, builder, activity, nil, logger)
}

func newChatRequest() chat.Request {
	return chat.Request{
		AgentID:     "agent-1",
		SessionID:   "sess-1",
		UserMessage: agent.NewUserMessage("hello"),
		Sink:        make(chan runtime.Event, 16),
	}
}

func runChat(t *testing.T, svc *chat.Service, ctx context.Context, req chat.Request) (error, []runtime.Event) {
	t.Helper()

	done := make(chan error, 1)
	go func() {
		done <- svc.Chat(ctx, req)
	}()

	var events []runtime.Event
	for {
		select {
		case ev, ok := <-req.Sink:
			if !ok {
				t.Fatal("sink closed before loop exit event")
				return nil, nil
			}
			events = append(events, ev)
			if _, ok := ev.(*runtime.LoopExitEvent); ok {
				return <-done, events
			}

		case <-time.After(2 * time.Second):
			t.Fatal("chat did not terminate in time")
			return nil, nil
		}
	}
}

func findEvent[T any](events []runtime.Event) (T, bool) {
	for _, ev := range events {
		if typed, ok := ev.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}

func requireEvent[T any](t *testing.T, ev runtime.Event) T {
	t.Helper()

	var zero T
	typed, ok := ev.(T)
	if !ok {
		t.Fatalf("unexpected event type: want %T, got %T", zero, ev)
	}
	return typed
}

func captureContextIDs(t *testing.T, agentID agent.ID, sessionID session.ID) (agent.ID, bool, session.ID, bool) {
	t.Helper()

	var gotAgentID agent.ID
	var okAgent bool
	var gotSessionID session.ID
	var okSession bool

	model := &chatMockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		completeFn: func(ctx context.Context, tools []agent.Tool, msgs []agent.Message) (*agent.Completion, error) {
			gotAgentID, okAgent = chat.AgentIDFromContext(ctx)
			gotSessionID, okSession = chat.SessionIDFromContext(ctx)
			return &agent.Completion{Done: true, Content: "final"}, nil
		},
	}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) { return newMockAgent(agentID), nil }},
		&mockSessionsRepo{sess: newTestSession(sessionID)},
		&mockModelRegistry{model: model},
		&mockToolRegistry{},
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		&mockActivityLogger{},
	)

	req := newChatRequest()
	req.AgentID = agentID
	req.SessionID = sessionID

	err, _ := runChat(t, svc, context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected chat error: %v", err)
	}

	return gotAgentID, okAgent, gotSessionID, okSession
}

func TestChat_ContextContainsAgentID(t *testing.T) {
	gotAgentID, ok, _, _ := captureContextIDs(t, "agent-1", "sess-1")

	if !ok {
		t.Fatal("expected agentID in context")
	}
	if gotAgentID != "agent-1" {
		t.Fatalf("expected agentID %q, got %q", "agent-1", gotAgentID)
	}
}

func TestChat_ContextContainsSessionID(t *testing.T) {
	_, _, gotSessionID, ok := captureContextIDs(t, "agent-1", "sess-1")

	if !ok {
		t.Fatal("expected sessionID in context")
	}
	if gotSessionID != "sess-1" {
		t.Fatalf("expected sessionID %q, got %q", "sess-1", gotSessionID)
	}
}

func TestChat_IgnoresMissingToolServers(t *testing.T) {
	registry := &mockToolRegistry{
		toolServersFn: func(names ...string) ([]agent.ToolServer, error) {
			return nil, fmt.Errorf("tool server %q is not exist: %w", names[0], os.ErrNotExist)
		},
	}

	model := &chatMockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []chatMockCompletion{
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return newMockAgent("agent-1", "missing"), nil
		}},
		&mockSessionsRepo{sess: newTestSession("sess-1")},
		&mockModelRegistry{model: model},
		registry,
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		&mockActivityLogger{},
	)

	err, events := runChat(t, svc, context.Background(), newChatRequest())
	if err != nil {
		t.Fatalf("expected no error on missing tool servers, got %v", err)
	}

	if _, ok := findEvent[*runtime.LoopExitEvent](events); !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
}

func TestChat_SavesSessionMessages(t *testing.T) {
	echo := &mockTool{
		name: "echo",
		callFn: func(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {
			return agent.NewContent("tool result"), nil
		},
	}

	registry := &mockToolRegistry{
		toolServersFn: func(names ...string) ([]agent.ToolServer, error) {
			return []agent.ToolServer{&mockToolServer{tools: []agent.Tool{echo}}}, nil
		},
	}

	model := &chatMockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []chatMockCompletion{
			{completion: &agent.Completion{
				Done: false,
				ToolCalls: []*agent.ToolCall{
					agent.NewToolCall("call-1", "echo", agent.ToolArguments(`{}`)),
				},
			}},
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	sessionsRepo := &mockSessionsRepo{sess: newTestSession("sess-1")}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return newMockAgent("agent-1", "echoserver"), nil
		}},
		sessionsRepo,
		&mockModelRegistry{model: model},
		registry,
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		&mockActivityLogger{},
	)

	err, _ := runChat(t, svc, context.Background(), newChatRequest())
	if err != nil {
		t.Fatalf("unexpected chat error: %v", err)
	}

	if !sessionsRepo.saved {
		t.Fatal("expected session to be saved")
	}

	msgs := sessionsRepo.savedSession.Messages()
	wantRoles := []agent.Role{
		agent.SystemMessageRole,
		agent.UserMessageRole,
		agent.AgentMessageRole,
		agent.ToolMessageRole,
		agent.AgentMessageRole,
	}

	if len(msgs) != len(wantRoles) {
		t.Fatalf("expected %d messages, got %d", len(wantRoles), len(msgs))
	}

	for i, want := range wantRoles {
		if got := msgs[i].Role(); got != want {
			t.Fatalf("message %d role: want %s, got %s", i, want, got)
		}
	}

	toolMsg, ok := msgs[3].(*agent.ToolResultMessage)
	if !ok {
		t.Fatalf("expected tool result message, got %T", msgs[3])
	}
	if got := toolMsg.Content()[0].Text; got != "tool result" {
		t.Fatalf("unexpected tool result %q", got)
	}

	firstAgent, ok := msgs[2].(*agent.AgentMessage)
	if !ok {
		t.Fatalf("expected agent message, got %T", msgs[2])
	}
	if len(firstAgent.ToolCalls()) != 1 {
		t.Fatalf("expected one tool call in agent message, got %d", len(firstAgent.ToolCalls()))
	}

	lastAgent, ok := msgs[4].(*agent.AgentMessage)
	if !ok {
		t.Fatalf("expected agent message, got %T", msgs[4])
	}
	if got := lastAgent.Content()[0].Text; got != "final" {
		t.Fatalf("unexpected final content %q", got)
	}
}

func TestRequest_ValidationErrors(t *testing.T) {
	svc := newTestService(t, nil, nil, nil, nil, nil, nil)

	base := func() chat.Request {
		return newChatRequest()
	}

	tests := []struct {
		name   string
		mutate func(*chat.Request)
		want   string
	}{
		{
			name:   "missing agent id",
			mutate: func(r *chat.Request) { r.AgentID = "" },
			want:   "agentID",
		},
		{
			name:   "missing session id",
			mutate: func(r *chat.Request) { r.SessionID = "" },
			want:   "sessionID",
		},
		{
			name:   "missing user message",
			mutate: func(r *chat.Request) { r.UserMessage = nil },
			want:   "user message",
		},
		{
			name:   "empty user message",
			mutate: func(r *chat.Request) { r.UserMessage = agent.NewUserMessage([]agent.ContentPart{}) },
			want:   "has content",
		},
		{
			name:   "missing sink",
			mutate: func(r *chat.Request) { r.Sink = nil },
			want:   "sink",
		},
		{
			name:   "empty content part",
			mutate: func(r *chat.Request) { r.UserMessage = agent.NewUserMessage("") },
			want:   "no empty content parts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base()
			tt.mutate(&req)

			err := svc.Chat(context.Background(), req)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestRequest_ValidationHappyPath(t *testing.T) {
	sentinel := errors.New("agent repo broken")

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return nil, sentinel
		}},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	err := svc.Chat(context.Background(), newChatRequest())
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected agent repo error to propagate, got %v", err)
	}
}

func TestChat_LogsUserAndAgentMessages(t *testing.T) {
	model := &chatMockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []chatMockCompletion{
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	activity := &mockActivityLogger{}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return newMockAgent("agent-1"), nil
		}},
		&mockSessionsRepo{sess: newTestSession("sess-1")},
		&mockModelRegistry{model: model},
		&mockToolRegistry{},
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		activity,
	)

	req := newChatRequest()
	req.Logging = true

	err, _ := runChat(t, svc, context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected chat error: %v", err)
	}

	if len(activity.commits) != 2 {
		t.Fatalf("expected 2 activity commits, got %d", len(activity.commits))
	}

	userCommit := activity.commits[0]
	if len(userCommit.msgs) != 1 {
		t.Fatalf("expected one message in user commit, got %d", len(userCommit.msgs))
	}
	userMsg, ok := userCommit.msgs[0].(*agent.UserMessage)
	if !ok {
		t.Fatalf("expected user message, got %T", userCommit.msgs[0])
	}
	if got := userMsg.Content()[0].Text; got != "hello" {
		t.Fatalf("unexpected logged user content %q", got)
	}

	agentCommit := activity.commits[1]
	if len(agentCommit.msgs) != 1 {
		t.Fatalf("expected one message in agent commit, got %d", len(agentCommit.msgs))
	}
	agentMsg, ok := agentCommit.msgs[0].(*agent.AgentMessage)
	if !ok {
		t.Fatalf("expected agent message, got %T", agentCommit.msgs[0])
	}
	if got := agentMsg.Content()[0].Text; got != "final" {
		t.Fatalf("unexpected logged agent content %q", got)
	}
}

func TestChat_HappyPath(t *testing.T) {
	model := &chatMockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []chatMockCompletion{
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	sessionsRepo := &mockSessionsRepo{sess: newTestSession("sess-1")}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return newMockAgent("agent-1"), nil
		}},
		sessionsRepo,
		&mockModelRegistry{model: model},
		&mockToolRegistry{},
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		&mockActivityLogger{},
	)

	err, events := runChat(t, svc, context.Background(), newChatRequest())
	if err != nil {
		t.Fatalf("unexpected chat error: %v", err)
	}

	complete, ok := findEvent[*runtime.CompleteEvent](events)
	if !ok {
		t.Fatalf("expected complete event, got %#v", events)
	}
	if got := complete.Complete().Content; got != "final" {
		t.Fatalf("unexpected completion content %q", got)
	}

	loopExit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if loopExit.Err() != nil {
		t.Fatalf("expected clean loop exit, got %v", loopExit.Err())
	}

	if !sessionsRepo.saved {
		t.Fatal("expected session to be saved")
	}
}

func TestChat_ForwardsAllEvents(t *testing.T) {
	echo := &mockTool{
		name: "echo",
		callFn: func(ctx context.Context, args agent.ToolArguments) ([]agent.ContentPart, error) {
			return agent.NewContent("tool result"), nil
		},
	}

	registry := &mockToolRegistry{
		toolServersFn: func(names ...string) ([]agent.ToolServer, error) {
			return []agent.ToolServer{&mockToolServer{tools: []agent.Tool{echo}}}, nil
		},
	}

	model := &chatMockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
		responses: []chatMockCompletion{
			{completion: &agent.Completion{
				Done: false,
				ToolCalls: []*agent.ToolCall{
					agent.NewToolCall("call-1", "echo", agent.ToolArguments(`{}`)),
				},
			}},
			{completion: &agent.Completion{Done: true, Content: "final"}},
		},
	}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return newMockAgent("agent-1", "echoserver"), nil
		}},
		&mockSessionsRepo{sess: newTestSession("sess-1")},
		&mockModelRegistry{model: model},
		registry,
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		&mockActivityLogger{},
	)

	err, events := runChat(t, svc, context.Background(), newChatRequest())
	if err != nil {
		t.Fatalf("unexpected chat error: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d: %#v", len(events), events)
	}

	first := requireEvent[*runtime.CompleteEvent](t, events[0])
	if got := len(first.Complete().ToolCalls); got != 1 {
		t.Fatalf("expected first completion to have 1 tool call, got %d", got)
	}

	requireEvent[*runtime.ToolResultEvent](t, events[1])

	second := requireEvent[*runtime.CompleteEvent](t, events[2])
	if !second.Complete().Done {
		t.Fatal("expected second completion to be done")
	}

	loopExit := requireEvent[*runtime.LoopExitEvent](t, events[3])
	if loopExit.Err() != nil {
		t.Fatalf("expected clean loop exit, got %v", loopExit.Err())
	}
}

func TestChat_CanceledLoopEventNotForwarded(t *testing.T) {
	model := &chatMockModel{
		settings: agent.ModelSettings{},
		ctxLimit: 100_000,
	}

	sessionsRepo := &mockSessionsRepo{sess: newTestSession("sess-1")}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return newMockAgent("agent-1"), nil
		}},
		sessionsRepo,
		&mockModelRegistry{model: model},
		&mockToolRegistry{},
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		&mockActivityLogger{},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err, events := runChat(t, svc, ctx, newChatRequest())
	if err != nil {
		t.Fatalf("unexpected chat error: %v", err)
	}

	loopExit, ok := findEvent[*runtime.LoopExitEvent](events)
	if !ok {
		t.Fatalf("expected loop exit event, got %#v", events)
	}
	if loopExit.Err() != nil {
		t.Fatalf("loop exit must not propagate context cancellation, got %v", loopExit.Err())
	}

	if !sessionsRepo.saved {
		t.Fatal("expected session to be saved on cancellation")
	}
}

func TestDispatcher_NewRequestCancelsPrevious(t *testing.T) {
	var calls atomic.Int32

	firstStarted := make(chan struct{})
	firstCanceled := make(chan struct{})

	model := &dispatcherModel{
		completeFn: func(ctx context.Context, tools []agent.Tool, msgs []agent.Message) (*agent.Completion, error) {
			if calls.Add(1) == 1 {
				close(firstStarted)
				<-ctx.Done()
				close(firstCanceled)
			}
			return &agent.Completion{Done: true, Content: "final"}, nil
		},
	}

	svc := newTestService(
		t,
		&mockAgentRepo{getFn: func(agent.ID) (agent.Agent, error) {
			return newMockAgent("agent-1"), nil
		}},
		&mockSessionsRepo{sess: newTestSession("sess-1")},
		&mockModelRegistry{model: model},
		&mockToolRegistry{},
		&mockSystemMessageBuilder{msg: agent.NewSystemMessage("system")},
		&mockActivityLogger{},
	)
	d := chat.NewDispatcher(svc)

	firstErr := make(chan error, 1)
	go func() {
		firstErr <- d.Chat(context.Background(), newChatRequest())
	}()

	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not reach the model call")
	}

	if err := d.Chat(context.Background(), newChatRequest()); err != nil {
		t.Fatalf("unexpected second chat error: %v", err)
	}

	select {
	case <-firstCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("previous request was not canceled")
	}

	select {
	case err := <-firstErr:
		if err != nil {
			t.Fatalf("unexpected first chat error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("previous request did not return")
	}
}
