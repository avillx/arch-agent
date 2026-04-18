package session

import (
	"arch-agent/internal/app/activity"
	"arch-agent/internal/app/types"
	"context"
	"log/slog"
	"time"
)

type Transcriptor interface {
	Transcribe(messages []types.Message) error
}

type SessionRepository interface {
	Load() (*Session, error)
	Save(*Session) error
	Drop() error
}

type Tokenizer interface {
	Calc(string) int
}

type Service struct {
	repo            SessionRepository
	idleDetector    *IdleDetector
	transcriptor    Transcriptor
	tokenizer       Tokenizer
	activityService *activity.Service
}

func NewSessionService(
	repo SessionRepository,
	tr Transcriptor,
	tk Tokenizer,
	as *activity.Service,
) *Service {

	s := &Service{
		repo:            repo,
		transcriptor:    tr,
		tokenizer:       tk,
		activityService: as,
	}

	s.idleDetector = NewIdleDetector(time.Minute*10, func() {
		active, err := s.repo.Load()

		if err != nil {
			slog.Error("bad idle session load", "error", err)
		}

		if err := s.Drop(context.Background(), active); err != nil {
			slog.Error("bad session idle drop", "error", err)
		}

	})

	return s
}

func (r *Service) Session() (*Session, error) {
	r.idleDetector.Touch()
	return r.repo.Load()
}

func (r *Service) Close(ctx context.Context, s *Session) error {
	r.idleDetector.Touch()

	s.Tokens = r.tokenizer.Calc(extractMessagesContent(s.Messages()))

	if s.IsOverflow() {
		if err := r.reduce(ctx, s); err != nil {
			return err
		}
	}

	if err := r.repo.Save(s); err != nil {
		return err
	}

	return nil
}

// calls on idle
func (r *Service) Drop(ctx context.Context, s *Session) error {
	messages := s.Messages()

	if err := r.transcriptor.Transcribe(messages); err != nil {
		return err
	}

	if err := r.activityService.LogActiviy(ctx, messages); err != nil {
		return err
	}

	return r.repo.Drop()
}

// calls on overflow
func (r *Service) reduce(ctx context.Context, s *Session) error {

	front := len(s.messages) * 2 / 3
	head := s.Messages()[:front]

	if err := r.transcriptor.Transcribe(head); err != nil {
		return err
	}

	if err := r.activityService.LogActiviy(ctx, head); err != nil {
		return err
	}

	tail := s.Messages()[front:]
	s.OverwriteMessages(tail)

	return nil
}

// helpers
func extractMessagesContent(msgs []types.Message) string {
	content := ""
	for _, m := range msgs {
		content += m.Content()
	}
	return content
}
