package session

import (
	"arch-agent/internal/app/message"
	"context"
	"log/slog"
	"time"
)

type ActivityLogger interface {
	Log(summary string) error
}

type Transcriptor interface {
	Transcribe(sessionID string, messages []message.Message) error
}

type SessionRepository interface {
	Load() *Session
	Update(*Session) error
	Drop(*Session)
}

type Summarizer interface {
	Sum(ctx context.Context, data []message.Message) (string, error)
}

type Tokenizer interface {
	Calc(string) int
}

type SessionService struct {
	repo           SessionRepository
	idleDetector   *IdleDetector
	transcriptor   Transcriptor
	tokenizer      Tokenizer
	summarizer     Summarizer
	activityLogger ActivityLogger
}

func NewSessionService(
	repo SessionRepository,
	tr Transcriptor,
	tk Tokenizer,
	sm Summarizer,
	al ActivityLogger,
) *SessionService {

	s := &SessionService{
		repo:           repo,
		transcriptor:   tr,
		tokenizer:      tk,
		summarizer:     sm,
		activityLogger: al,
	}

	s.idleDetector = NewIdleDetector(time.Minute*10, func() {
		if active := s.repo.Load(); active != nil {
			if err := s.drop(context.Background(), active); err != nil {
				slog.Error("session bad drop", "error", err)
			}
			return
		}
		slog.Warn("try to drop inactive session")
	})

	return s
}

func (r *SessionService) Session() *Session {
	r.idleDetector.Touch()
	return r.repo.Load()
}

func (r *SessionService) Close(ctx context.Context, s *Session) error {
	r.idleDetector.Touch()

	s.Tokens = r.tokenizer.Calc(extractMessagesContent(s.Messages()))

	if err := r.repo.Update(s); err != nil {
		return err
	}

	if s.IsOverflow() {
		if err := r.reduce(ctx, s); err != nil {
			return err
		}
	}

	return nil
}

// calls on idle
func (r *SessionService) drop(ctx context.Context, s *Session) error {

	// TODO:
	// - Drop and Recuce have a shared abstraction on writin process

	messages := s.Messages()

	if err := r.transcriptor.Transcribe(s.id, messages); err != nil {
		return err
	}

	summary, err := r.summarizer.Sum(ctx, messages)
	if err != nil {
		return err
	}

	if err := r.activityLogger.Log(summary); err != nil {
		return err
	}

	r.repo.Drop(s)

	return nil
}

// calls on overflow
func (r *SessionService) reduce(ctx context.Context, s *Session) error {

	front := len(s.messages) * 2 / 3
	head := s.Messages()[:front]

	if err := r.transcriptor.Transcribe(s.id, head); err != nil {
		return err
	}

	summary, err := r.summarizer.Sum(ctx, head)
	if err != nil {
		return err
	}

	if err := r.activityLogger.Log(summary); err != nil {
		return err
	}

	tail := s.Messages()[front:]
	s.Reduce(tail)

	return nil
}

// helpers
func extractMessagesContent(msgs []message.Message) string {
	content := ""
	for _, m := range msgs {
		content += m.Content()
	}
	return content
}
