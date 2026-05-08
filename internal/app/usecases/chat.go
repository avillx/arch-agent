package usecases

import (
	service "arch-agent/internal/app"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"arch-agent/internal/domain/types"
	"context"
	"errors"
)

type ChatLoop struct {
	agentService   *service.AgentService
	sessionService *service.SessionService
}

func NewChatLoop(
	agentService *service.AgentService,
	sessionService *service.SessionService,
) *ChatLoop {
	return &ChatLoop{
		agentService:   agentService,
		sessionService: sessionService,
	}
}

func (uc *ChatLoop) Chat(
	ctx context.Context,
	sessionID session.ID,
	agentID agent.ID,
	request string,
	onContent func(string),
) error {
	a, err := uc.agentService.GetAgent(agentID)
	if err != nil {
		return err
	}

	sess, err := uc.sessionService.Get(agentID, sessionID)
	if err != nil {
		return err
	}

	userMsg := types.NewUserMessage(request)
	conversation := append(sess.Messages(), userMsg)

	a.OnContent(onContent)

	var errc error
	newMsgs, err := a.Chat(ctx, conversation)
	if err != nil {
		errc = errors.Join(errc, err)
	}

	err = uc.sessionService.AppendMessages(agentID, sess, append([]types.Message{userMsg}, newMsgs...))
	if err != nil {
		errc = errors.Join(errc, err)
	}

	return errc
}
