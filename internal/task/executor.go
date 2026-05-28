package task

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/session"
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type taskExecutor struct {
	agentRepo  agent.Repo
	sessionSvc *session.SessionService
	modelRepo  agent.ModelRepository
	runtime    *runtime.AgentRuntime
}

// executor
func NewTaskExecutor(
	agentRepo agent.Repo,
	sessionSvc *session.SessionService,
	modelRepo agent.ModelRepository,
	runtime *runtime.AgentRuntime,
) *taskExecutor {
	return &taskExecutor{
		agentRepo:  agentRepo,
		sessionSvc: sessionSvc,
		modelRepo:  modelRepo,
		runtime:    runtime,
	}
}

// func (s *taskExecutor) Validate(t Task) error {
// 	// get existed agents
// 	agentsCfgs, err := s.agentSvc.List()
// 	if err != nil {
// 		return err
// 	}

// 	// validate agents
// 	agentMap := map[agent.ID]struct{}{}
// 	for _, cfg := range agentsCfgs {
// 		agentMap[cfg.ID()] = struct{}{}
// 	}
// 	for _, agent := range t.Recipients {
// 		if _, ok := agentMap[agent]; !ok {
// 			return fmt.Errorf("agent %s is not exist", agent)
// 		}
// 	}

// 	return nil
// }

func (s *taskExecutor) execute(ctx context.Context, t Task) {
	for _, r := range t.Recipients {
		slog.Info("processing task", "agent", r, "task", t.Name)
		if err := s.processRecipientTask(ctx, agent.ID(r), t.Name, t.Request); err != nil {
			slog.Error("task processing", "agent", r, "task", t.Name, "error", err)
		}
	}
}

func (s *taskExecutor) processRecipientTask(ctx context.Context, agentID agent.ID, taskName, request string) error {

	const autonomusWorking = "Now You working autonomusly if somthing wrong try to contact with someone"

	slog.Info("task executes in background", "task", taskName)

	agt, err := s.agentRepo.Get(agentID)
	if err != nil {
		return err
	}

	var errc error
	evReader := runtime.EventReader{
		OnError: func(i1 agent.ID, i2 session.ID, err error) {
			errc = errors.Join(errc, err)
		},
	}

	sessionID, err := s.sessionSvc.Create(agentID)
	if err != nil {
		return err
	}

	sess, err := s.sessionSvc.Get(agentID, sessionID)
	if err != nil {
		return err
	}

	sess.AddMessages(
		[]agent.Message{agent.NewUserMessage(fmt.Sprintf("%s\n\n%s", autonomusWorking, request))},
	)

	model, err := s.modelRepo.Get(agt.Model())
	if err != nil {
		return err
	}

	sink := s.runtime.RunStream(ctx, model, agt, agt.Tools(), sess)
	evReader.Read(sink)

	s.sessionSvc.Save(agentID, sess)
	return nil
}
