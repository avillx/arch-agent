package a2a

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/session"

	"arch-agent/internal/runtime"

	"context"
	"fmt"
	"strings"
)

type Service struct {
	sessionSevice *session.SessionService
	chatSvc       *chat.ChatService
}

func NewService(
	chatSvc *chat.ChatService,
	sessionSvc *session.SessionService,
) *Service {
	return &Service{
		chatSvc:       chatSvc,
		sessionSevice: sessionSvc,
	}
}

func (s *Service) Call(
	ctx context.Context,
	callerAgentID agent.ID,
	recivierAgentID agent.ID,
	sessionID session.ID,
	request string,
) (string, error) {

	sess, err := s.sessionSevice.Get(callerAgentID, sessionID)
	if err != nil {
		return "", err
	}

	subSessionID, ok := sess.Subsession(recivierAgentID)
	if !ok {
		subSessionID, err = s.sessionSevice.Create(recivierAgentID)
		if err != nil {
			return "", err
		}
		sess.AddSubsession(recivierAgentID, subSessionID)
	}

	lastAgentMessageContent := ""
	evReader := runtime.EventReader{
		OnComplete: func(i1 agent.ID, i2 session.ID, c *agent.Completion) {
			lastAgentMessageContent = c.Content
		},
	}

	err = s.chatSvc.Chat(ctx, recivierAgentID, subSessionID, request, evReader)

	return lastAgentMessageContent, err
}

func wrapMessageToPrompt(caller agent.ID, message string) string {
	var sb strings.Builder

	sb.WriteString("Write answer on agent message, all out put will be sended to caller. Never use use call_agent for answer back.\n\n")
	sb.WriteString("## Agent message\n")
	fmt.Fprintf(&sb, "From: %s \n", string(caller))
	sb.WriteString(message)

	return sb.String()
}
