package chat

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/memory"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type Request struct {
	AgentID             agent.ID
	SessionID           session.ID
	UserMessage         *agent.UserMessage
	ProvidedToolServers []agent.ToolServer
	Logging             bool
	Sink                chan runtime.Event
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

	if r.Sink == nil {
		return fmt.Errorf("sink is required")
	}

	for _, c := range r.UserMessage.Content() {
		if c.ImageURL == "" && c.Text == "" {
			return fmt.Errorf("user message must has no empty content parts")
		}
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
	logger           *slog.Logger
}

func NewService(
	agentRepo agent.Repo,
	sessionSvc *session.Service,
	modelRepo agent.ModelRegistry,
	toolRegistry agent.ToolRegistry,
	contextAssembler *ContextAssembler,
	observer *memory.Observer,
	hooks []any,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentRepo:        agentRepo,
		sessionSvc:       sessionSvc,
		modelRepo:        modelRepo,
		toolRegistry:     toolRegistry,
		observer:         observer,
		contextAssembler: contextAssembler,
		hooks:            hooks,
		logger:           logger.WithGroup("chat"),
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

	if r.Logging {
		s.observer.Commit(agt.ID(), sess.ID(), []agent.Message{r.UserMessage})
	}

	// inject id's
	ctx = withAgentID(ctx, r.AgentID)
	ctx = withSessionID(ctx, r.SessionID)

	s.runAgentLoopWithCallbacks(
		ctx,
		model,
		r.AgentID,
		sess,
		agentContext,
		extractTools(toolServers),
		r.Logging,
		s.hooks,
		r.Sink,
	)

	return nil
}

func (s *Service) runAgentLoopWithCallbacks(
	ctx context.Context,
	model agent.Model,
	agentID agent.ID,
	sess session.Session,
	agentContext []agent.Message,
	toolKit []agent.Tool,
	observe bool,
	hooks []any,
	sink chan runtime.Event,
) {
	logger := s.logger.With(
		"agent", agentID,
		"session", sess.ID(),
	)

	// sink
	evCh := make(chan runtime.Event, 16)

	go func() {
		defer close(evCh)
		runtime.RunAgentLoop(
			ctx,
			model,
			agentContext,
			toolKit,
			evCh,
			hooks,
		)
	}()

	for rawEv := range evCh {
		switch ev := rawEv.(type) {

		// loop exit event
		case *runtime.LoopExitEvent:
			if err := ev.Err(); err != nil {
				logger.Error("loop exit with error",
					"error", err,
				)
			} else {
				logger.Info("loop exit")
			}

			if err := s.sessionSvc.Save(agentID, sess); err != nil {
				logger.Error("save session", "error", err)
			}

		// compaction event
		case *runtime.CompactionEvent:
			logger.Warn("session compacted")
			sess.OverwriteMessages(0, ev.CompactedContext())

		// complete event
		case *runtime.CompleteEvent:
			c := ev.Complete()
			logger.Info("completion",
				"tool_calls", len(c.ToolCalls),
				"input_tokens", c.InputTokens,
				"output_tokens", c.CompletionTokens,
			)
			// modify session
			sess.ApplyCompletion(c)

			// observe
			if observe {
				completion := ev.Complete()
				msg := agent.NewAgentMessage(completion.Content, completion.ToolCalls)
				msgs := []agent.Message{msg}

				// forward to request
				s.observer.Commit(agentID, sess.ID(), msgs)
			}

		// completion mistake
		case *runtime.CompletionMistakeEvent:
			logger.Info("compltion mistake",
				"error", ev.Err(),
			)

		// tool result
		case *runtime.ToolResultEvent:
			logger.Info("tool result recivied",
				"tool", ev.Call().ToolName,
			)
			sess.AddMessages(agent.NewToolResultMessage(ev.Result()))

		// tool call error
		case *runtime.ToolCallErrEvent:
			err := ev.Err()
			var mistake *types.AgentMistakeError
			if errors.As(err, &mistake) {
				logger.Warn("bad tool call",
					"tool", ev.ToolName(),
				)
			} else {
				logger.Error("tool call error",
					"tool", ev.ToolName(),
					"error", ev.Err(),
				)
			}
		}

		// forward to origin
		sink <- rawEv
	}
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

// TODO:
// chan forwarding via  x <-evCh; fEvCh <- x;
// processing via [T Event]EventHanlder(ev Event)
// for each handler
// if h,ok := handler.([T]EventHandler); ok {h(ev)}
// process_shit()
