package session

import (
	"arch-agent/internal/agent"
	"context"
	"log/slog"
	"strings"
)

type SessionChatService struct {
	chatSvc        agent.ChatSvc
	sessionService *SessionService
}

func NewSessionChatService(
	chatSvc agent.ChatSvc,
	sessionService *SessionService,
) *SessionChatService {
	return &SessionChatService{
		chatSvc:        chatSvc,
		sessionService: sessionService,
	}
}

func (s *SessionChatService) SessionChat(
	ctx context.Context,
	agentID agent.ID,
	sessionID ID,
	presummaryAdditioanlSystemPrompt string,
	postsummaryAdditioanlSystemPrompt string,
	request string,
	onResult func(result *agent.ReasonResult),
) error {
	slog.Debug("Session chat", "agent", agentID, "request", request)

	sess, err := s.sessionService.Get(agentID, sessionID)
	if err != nil {
		return err
	}

	summaries := sess.Summaries()
	if summaries != "" {
		summaries = strings.Join([]string{
			"<summary>",
			"# Here's previus conversation summary:",
			summaries,
			"</summary>",
		}, "\n")
	}

	additionalSystemPrompt := strings.Join([]string{
		presummaryAdditioanlSystemPrompt,
		summaries,
		postsummaryAdditioanlSystemPrompt}, "\n")

	userMessages := []agent.Message{agent.NewUserMessage(request)}
	newMessages, err := s.chatSvc.Chat(
		ctx,
		agentID,
		additionalSystemPrompt,
		append(sess.Messages(), userMessages...),
		onResult,
		nil,
	)
	if err != nil {
		return err
	}

	sess.AddMessages(userMessages)
	sess.AddMessages(newMessages)

	if err := sess.ProcessOverflow(OverflowPolicy{
		TokenLimit: 500000,
		OnOverflow: func(sess *Session) error {
			return s.truncateSession(ctx, sess)
		},
	}); err != nil {
		slog.Error("truncate session", "agent", agentID, "session", sess.ID, "error", err)
	}

	return s.sessionService.Save(agentID, sess)
}

func (s *SessionChatService) truncateSession(ctx context.Context, sess *Session) error {
	// TODO: refactor
	messages := sess.Messages()
	half := len(messages) / 2
	conver := agent.StringifyConversation(messages[:half])
	request := []agent.Message{agent.NewUserMessage(conver)}
	result, err := s.chatSvc.Chat(ctx, "summarizer", "", request, nil, nil)
	if err != nil {
		return err
	}
	summary := result[len(result)-1].Content()
	sess.AddSummary(summary)
	sess.OverwriteMessages(messages[half:])

	return nil
}
