package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"context"
	"fmt"
	"time"
)

type EventCallbacks struct {
	OnLoopExit   func(*runtime.LoopExitEvent)
	OnComplete   func(*runtime.CompleteEvent)
	OnToolResult func(*runtime.ToolResultEvent)
	OnCompaction func(*runtime.CompactionEvent)
	OnToolErr    func(*runtime.ToolCallErrEvent)
	OnEvent      func(runtime.Event)
}

type Request struct {
	AgentID             agent.ID
	SessionID           session.ID
	UserMessage         *agent.UserMessage
	ProvidedToolServers []agent.ToolServer
	Logging             bool
	EventCallbacks
}

func (r Request) validate() error {
	if r.AgentID == "" {
		return fmt.Errorf("completion request must include agentID")
	}

	if r.SessionID == "" {
		return fmt.Errorf("completion request must include sessionID")
	}

	if r.UserMessage == nil {
		return fmt.Errorf("completion request must include user message")
	}

	if len(r.UserMessage.Content()) <= 0 {
		return fmt.Errorf("user message must has content")
	}

	for _, c := range r.UserMessage.Content() {
		if c.ImageURL == "" && c.Text == "" {
			return fmt.Errorf("user message must has no empty content parts")
		}
	}

	if r.OnCompaction == nil {
		return fmt.Errorf("on compaction callback is empty")
	}

	if r.OnComplete == nil {
		return fmt.Errorf("on complete callback is empty")
	}

	if r.OnEvent == nil {
		return fmt.Errorf("on event callback is empty")
	}

	if r.OnLoopExit == nil {
		return fmt.Errorf("on loop exit callback is empty")
	}

	if r.OnToolErr == nil {
		return fmt.Errorf("on tool error callback is empty")
	}

	if r.OnToolResult == nil {
		return fmt.Errorf("on tool result callback is empty")
	}

	return nil
}

var _ ChatExecutor = (*Service)(nil)

type ChatExecutor interface {
	Chat(ctx context.Context, r Request) error
}

type Service struct {
	agentRepo        agent.Repo
	sessionSvc       *session.Service
	modelRepo        agent.ModelRegistry
	toolRegistry     agent.ToolRegistry
	contextAssembler *ContextAssembler
	observer         *memory.Observer
	hooks            []any
}

func NewService(
	agentRepo agent.Repo,
	sessionSvc *session.Service,
	modelRepo agent.ModelRegistry,
	toolRegistry agent.ToolRegistry,
	contextAssembler *ContextAssembler,
	observer *memory.Observer,
	hooks []any,
) *Service {
	return &Service{
		agentRepo:        agentRepo,
		sessionSvc:       sessionSvc,
		modelRepo:        modelRepo,
		toolRegistry:     toolRegistry,
		observer:         observer,
		contextAssembler: contextAssembler,
		hooks:            hooks,
	}
}

func (s *Service) Chat(
	ctx context.Context,
	r Request,
) error {

	// validate request
	if err := r.validate(); err != nil {
		return err
	}

	//agent
	agt, err := s.agentRepo.Get(r.AgentID)
	if err != nil {
		return err
	}

	// model
	model, err := s.modelRepo.Get(agt.Model())
	if err != nil {
		return err
	}

	// tools
	toolServers, err := resolveToolServers(s.toolRegistry, agt, r.ProvidedToolServers)
	if err != nil {
		return err
	}

	// session
	sess, err := s.sessionSvc.Get(agt.ID(), r.SessionID)
	if err != nil {
		return err
	}

	sess.AddMessages(r.UserMessage)

	r.EventCallbacks, err = s.attachSession(
		r.AgentID,
		sess,
		r.EventCallbacks,
	)
	if err != nil {
		return err
	}

	// logging
	if r.Logging {
		r.EventCallbacks = s.attachLogging(
			r.AgentID,
			r.SessionID,
			r.UserMessage,
			r.EventCallbacks,
		)
	}

	// build context
	systemMessage, err := s.contextAssembler.BuildSystemMessage(
		ctx,
		agt,
		toolServers,
		sess,
	)
	if err != nil {
		return err
	}

	agentContext := []agent.Message{}
	agentContext = append(agentContext, systemMessage)
	agentContext = append(agentContext, sess.Messages()...)

	// inject id's
	ctx = withAgentID(ctx, r.AgentID)
	ctx = withSessionID(ctx, r.SessionID)

	// run
	return runAgentLoopWithCallbacks(
		ctx,
		model,
		agentContext,
		extractTools(toolServers),
		r.EventCallbacks,
		s.hooks,
	)
}

func (s *Service) attachSession(
	agentID agent.ID,
	sess session.Session,
	eventCallbacks EventCallbacks,
) (EventCallbacks, error) {

	// wrap completion
	wrappedOnComplete := eventCallbacks.OnComplete
	eventCallbacks.OnComplete = func(ev *runtime.CompleteEvent) {
		sess.ApplyCompletion(ev.Complete())

		wrappedOnComplete(ev)
	}

	// wrap toolresult
	wrappedOnToolResult := eventCallbacks.OnToolResult
	eventCallbacks.OnToolResult = func(ev *runtime.ToolResultEvent) {
		sess.AddMessages(agent.NewToolResultMessage(ev.Result()))

		wrappedOnToolResult(ev)
	}

	// wrap compaction
	wrappedOnCompaction := eventCallbacks.OnCompaction
	eventCallbacks.OnCompaction = func(ev *runtime.CompactionEvent) {
		sess.OverwriteMessages(0, ev.CompactedContext())

		wrappedOnCompaction(ev)
	}

	// wrap loop exit
	wrappedOnLoopExit := eventCallbacks.OnLoopExit
	eventCallbacks.OnLoopExit = func(ev *runtime.LoopExitEvent) {
		if err := s.sessionSvc.Save(agentID, sess); err != nil {
			// log it
		}
		wrappedOnLoopExit(ev)
	}

	return eventCallbacks, nil
}

func (e *Service) attachLogging(
	agentID agent.ID,
	sessID session.ID,
	userMessage agent.Message,
	eventCallbacks EventCallbacks,
) EventCallbacks {
	// commit user message
	e.observer.Commit(agentID, sessID, []agent.Message{userMessage})

	// wrap on complete for log shit
	wrappedOnComplete := eventCallbacks.OnComplete
	eventCallbacks.OnComplete = func(ev *runtime.CompleteEvent) {
		completion := ev.Complete()
		msg := agent.NewAgentMessage(completion.Content, completion.ToolCalls)
		msgs := []agent.Message{msg}
		e.observer.Commit(agentID, sessID, msgs)

		wrappedOnComplete(ev)
	}

	return eventCallbacks
}

func readEvents(
	ch <-chan runtime.Event,
	OnLoopExit func(*runtime.LoopExitEvent),
	OnComplete func(*runtime.CompleteEvent),
	OnToolResult func(*runtime.ToolResultEvent),
	OnCompaction func(*runtime.CompactionEvent),
	OnToolErr func(*runtime.ToolCallErrEvent),
	OnEvent func(runtime.Event),
) {
	for ev := range ch {
		OnEvent(ev)

		switch typedEv := ev.(type) {
		case *runtime.LoopExitEvent:
			OnLoopExit(typedEv)
		case *runtime.CompactionEvent:
			OnCompaction(typedEv)
		case *runtime.CompleteEvent:
			OnComplete(typedEv)
		case *runtime.ToolResultEvent:
			OnToolResult(typedEv)
		case *runtime.ToolCallErrEvent:
			OnToolErr(typedEv)
		}
	}
}

func runAgentLoopWithCallbacks(
	ctx context.Context,
	model agent.Model,
	agentContext []agent.Message,
	toolKit []agent.Tool,
	evCallbacks EventCallbacks,
	hooks []any,
) error {
	// sink
	evCh := make(chan runtime.Event, 16)
	defer close(evCh)

	done := make(chan struct{})
	go func() {
		readEvents(
			evCh,
			evCallbacks.OnLoopExit,
			evCallbacks.OnComplete,
			evCallbacks.OnToolResult,
			evCallbacks.OnCompaction,
			evCallbacks.OnToolErr,
			evCallbacks.OnEvent,
		)
		close(done)
	}()

	// run
	loopErr := runtime.RunAgentLoop(
		ctx,
		model,
		agentContext,
		toolKit,
		evCh,
		hooks,
	)

	// await of event processing
	awaitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	select {
	case <-awaitCtx.Done():
	case <-done:
	}

	return loopErr
}

func resolveToolServers(reg agent.ToolRegistry, agt agent.Agent, provided []agent.ToolServer) ([]agent.ToolServer, error) {
	toolServers, err := reg.ToolServers(agt.ToolServers()...)
	if err != nil {
		if err := types.DistillErrNotExist(fmt.Sprintf("agent %s", err), err); err != nil {
			return nil, err
		}
	}

	if provided != nil {
		toolServers = append(toolServers, provided...)
	}

	return toolServers, nil
}

func extractTools(toolServers []agent.ToolServer) []agent.Tool {
	toolKit := []agent.Tool{}
	for _, ts := range toolServers {
		toolKit = append(toolKit, ts.Tools()...)
	}
	return toolKit
}

type agentIDCTXKey struct{}
type sessionIDCTXKey struct{}

func withAgentID(ctx context.Context, id agent.ID) context.Context {
	return context.WithValue(ctx, agentIDCTXKey{}, id)
}
func AgentIDFromContext(ctx context.Context) (agent.ID, bool) {
	id, ok := ctx.Value(agentIDCTXKey{}).(agent.ID)
	return id, ok
}

func withSessionID(ctx context.Context, id session.ID) context.Context {
	return context.WithValue(ctx, sessionIDCTXKey{}, id)
}
func SessionIDFromContext(ctx context.Context) (session.ID, bool) {
	id, ok := ctx.Value(sessionIDCTXKey{}).(session.ID)
	return id, ok
}
