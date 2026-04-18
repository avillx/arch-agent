package dreaming

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/types"
	"context"
	"errors"
	"time"
)

var ErrNeverDream = errors.New("never been dreamed")

type PromptRenderer interface {
	Render() (string, error)
}

type DreamingStorage interface {
	LastDreaming() (time.Time, error)
	SaveReport(content string) error
}

type Service struct {
	reasoningService *reasoning.Service
	promptRenderer   PromptRenderer
	storage          DreamingStorage
}

func NewService(
	reasoningService *reasoning.Service,
	promptRenderer PromptRenderer,
	storage DreamingStorage,
) *Service {
	return &Service{
		reasoningService: reasoningService,
		promptRenderer:   promptRenderer,
		storage:          storage,
	}
}

func (s *Service) Dream(ctx context.Context, data string) error {
	prompt, err := s.promptRenderer.Render()
	if err != nil {
		return err
	}

	msgs := []types.Message{types.NewUserMessage(data)}

	res, err := s.reasoningService.Reason(ctx, prompt, nil, msgs)
	if err != nil {
		return err
	}

	return s.storage.SaveReport(res[len(res)-1].Content())
}

func (s *Service) LastDreaming() (time.Time, error) {
	return s.storage.LastDreaming()
}
