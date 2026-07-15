package subagent

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/prompt"
	"arch-agent/internal/session"
	"arch-agent/internal/types"
	"errors"
	"fmt"

	"arch-agent/internal/runtime"

	"context"
)

type ctxKey struct{}

type subAgentCall struct {
	subagent agent.ID
}

const maxSubAgentDepth = 3

func subAgentCallStack(ctx context.Context, s subAgentCall) (context.Context, bool) {
	callStack, _ := ctx.Value(ctxKey{}).([]subAgentCall)
	if len(callStack) >= maxSubAgentDepth {
		return ctx, true
	}
	newStack := append([]subAgentCall{s}, callStack...)
	return context.WithValue(ctx, ctxKey{}, newStack), false
}

type Service struct {
	harness   *runtime.Harness
	rt        *runtime.AgentRuntime
	toolRepo  agent.ToolRegistry
	agentRepo agent.Repo
	modelRepo agent.ModelRepository
}

func NewService(
	harness *runtime.Harness,
	rt *runtime.AgentRuntime,
	toolRepo agent.ToolRegistry,
	agentRepo agent.Repo,
	modelRepo agent.ModelRepository,
) *Service {
	return &Service{
		harness:   harness,
		rt:        rt,
		toolRepo:  toolRepo,
		agentRepo: agentRepo,
		modelRepo: modelRepo,
	}
}

var ErrCallStackOverflow = errors.New("sub agent call is overflow")

func (s *Service) Call(
	ctx context.Context,
	subAgentID agent.ID,
	request string,
) (string, error) {

	ctx, isOverflow := subAgentCallStack(ctx, subAgentCall{subagent: subAgentID})
	if isOverflow {
		return "", ErrCallStackOverflow
	}

	sess := session.NewSession("")
	sess.AddMessages(agent.NewUserMessage(prompt.SubAgentGuidance(request)))

	//agent
	agt, err := s.agentRepo.Get(subAgentID)
	if err != nil {
		return "", err
	}

	// model
	model, err := s.modelRepo.Get(agt.Model())
	if err != nil {
		return "", err
	}

	// tools
	tools, err := s.toolRepo.GetServerTools(agt.ToolServers())
	if err != nil {
		if err := types.DistillErrNotExist(fmt.Sprintf("subagent %s", subAgentID), err); err != nil {
			return "", err
		}
	}

	// sink
	lastAgentMessageContent := ""
	evReader := runtime.EventReader{
		OnComplete: func(_ agent.ID, _ session.ID, c *agent.Completion) {
			lastAgentMessageContent = c.Content
		},
	}

	evCh := make(chan runtime.Event, 16)
	go evReader.Read(evCh)

	err = s.rt.RunStream(
		ctx,
		runtime.RunStramRequest{
			Model:   model,
			Tools:   tools,
			Sess:    sess,
			Agent:   agt,
			EvCh:    evCh,
			Harness: s.harness,
			BuildContextRequest: runtime.BuildContextRequest{
				IncludeMemory:       true,
				IncludeSkills:       true,
				AllowOptimizeImages: true,
				AddInstuctions:      true,
			},
		},
	)

	return lastAgentMessageContent, err
}
