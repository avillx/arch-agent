package service

import (
	"arch-agent/internal/domain/activity"
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/session"
	"arch-agent/internal/domain/types"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const LiveSessionExpiresTime = 3 * time.Minute

type ActivityRepo interface {
	Log(agent.ID, activity.Record) error
	GetActivity(agent.ID, time.Time) (string, error)
}

type LiveSession struct {
	Timer *time.Timer
	ID    session.ID
}

type LiveChatService struct {
	liveSessions map[agent.ID]*LiveSession
	mu           sync.Mutex

	sessionService *SessionService
	activityRepo   ActivityRepo
	agentService   *AgentService
}

func NewLiveChatService(
	sessionService *SessionService,
	activityRepo ActivityRepo,
	agentService *AgentService,
) *LiveChatService {
	return &LiveChatService{
		liveSessions:   map[agent.ID]*LiveSession{},
		sessionService: sessionService,
		activityRepo:   activityRepo,
		agentService:   agentService,
	}
}

func (s *LiveChatService) createLiveSession(agentID agent.ID) (*LiveSession, error) {

	sessionID, err := s.sessionService.Create(agentID)
	if err != nil {
		return nil, err
	}

	ls := &LiveSession{
		ID:    sessionID,
		Timer: s.dropSessionTimer(agentID, sessionID, LiveSessionExpiresTime),
	}

	s.mu.Lock()
	s.liveSessions[agentID] = ls
	s.mu.Unlock()

	return ls, nil

}

func (s *LiveChatService) Chat(
	ctx context.Context,
	agentID agent.ID,
	request string,
	onResult func(result *agent.ReasonResult),
) error {

	s.mu.Lock()
	liveSession, ok := s.liveSessions[agentID]
	s.mu.Unlock()

	if ok {
		liveSession.Timer.Stop()
		liveSession.Timer.Reset(LiveSessionExpiresTime)
	} else {
		newLiveSession, err := s.createLiveSession(agentID)
		if err != nil {
			return err
		}
		liveSession = newLiveSession
	}

	sess, err := s.sessionService.Get(agentID, liveSession.ID)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, SessionContextKey, sess)

	memoryContent, err := s.memoryContent(agentID)
	if err != nil {
		return err
	}

	userMessage := []types.Message{types.NewUserMessage(request)}

	newMessages, err := s.agentService.Chat(
		ctx,
		agentID,
		memoryContent,
		nil,
		sess.Messages(),
		userMessage,
		onResult,
	)
	if err != nil {
		return err
	}

	sess.AddMessages(userMessage)
	sess.AddMessages(newMessages)

	if err := sess.ProcessOverflow(session.OverflowPolicy{
		TokenLimit: 10000,
		OnOverflow: func(sess *session.Session) error {
			return s.truncateSession(ctx, agentID, sess)
		},
	}); err != nil {
		slog.Error("truncate session", "agent", agentID, "session", sess.ID, "error", err)
	}

	return s.sessionService.Save(agentID, sess)
}

func (s *LiveChatService) dropSessionTimer(agentID agent.ID, sessionID session.ID, d time.Duration) *time.Timer {
	return time.AfterFunc(d, func() {
		if err := s.dropSession(context.Background(), agentID, sessionID); err != nil {
			slog.Error("live session drop", "agent", agentID, "error", err)
		}
		s.mu.Lock()
		delete(s.liveSessions, agentID)
		s.mu.Unlock()
	})
}

func (s *LiveChatService) dropSession(ctx context.Context, agentID agent.ID, sessionID session.ID) error {
	sess, err := s.sessionService.Get(agentID, sessionID)
	if err != nil {
		return err
	}

	if err := s.memorizeMessages(ctx, agentID, sess.Messages()); err != nil {
		return err
	}

	return s.sessionService.Delete(agentID, sess.ID)
}

func (s *LiveChatService) truncateSession(ctx context.Context, agentID agent.ID, sess *session.Session) error {
	messages := sess.Messages()
	halfLen := len(messages) / 2
	head := messages[:halfLen]
	tail := messages[halfLen:]

	if err := s.memorizeMessages(ctx, agentID, head); err != nil {
		return err
	}

	sess.OverwriteMessages(tail)

	return nil
}

func (s *LiveChatService) memorizeMessages(ctx context.Context, agentID agent.ID, messages []types.Message) error {

	memoryContent := ""

	_, err := s.agentService.Chat(
		ctx,
		agentID,
		"",
		nil,
		messages,
		[]types.Message{types.NewUserMessage(Memorize())},
		func(result *agent.ReasonResult) {
			memoryContent += result.Content
		},
	)

	if err != nil {
		return err
	}

	return s.activityRepo.Log(agentID, activity.NewRecord(memoryContent))
}

// Creates a message with activity for agent
func (s *LiveChatService) memoryContent(agentID agent.ID) (string, error) {
	yesterday, err := s.activityRepo.GetActivity(agentID, time.Now().Local().AddDate(0, 0, -1))
	if err != nil {
		if !errors.Is(err, activity.ErrNoActivity) {
			return "", err
		}
	}

	today, err := s.activityRepo.GetActivity(agentID, time.Now())
	if err != nil {
		if !errors.Is(err, activity.ErrNoActivity) {
			return "", err
		}
	}

	if yesterday == "" && today == "" {
		return "", nil
	}

	return buildMemoryContent(yesterday, today), nil
}

func buildMemoryContent(yesterday, today string) string {
	var sb strings.Builder

	sb.WriteString("<memory>\n")

	if yesterday != "" {
		sb.WriteString("# Yesterday:\n")
		sb.WriteString(yesterday)
	}

	if today != "" {
		sb.WriteString("# Today:\n")
		sb.WriteString(today)
	}

	sb.WriteString("\n</memory>")

	return sb.String()

}
