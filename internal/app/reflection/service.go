package reflection

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/types"
	"context"
	"fmt"
)

type ReflectionParams struct {
	Personality string
	Emotions    string
}

type PromptRenderer interface {
	Render(ReflectionParams) (string, error)
}

type Service struct {
	personality      string
	promptRenderer   PromptRenderer
	emoService       *EmotionalService
	reasoningService *reasoning.Service
}

func NewService(
	personality string,
	promptRenderer PromptRenderer,
	emoService *EmotionalService,
	reasoningService *reasoning.Service,
) *Service {
	return &Service{
		personality:      personality,
		promptRenderer:   promptRenderer,
		emoService:       emoService,
		reasoningService: reasoningService,
	}
}

func (s *Service) Feelings(ctx context.Context, msgs []types.Message) (string, error) {
	toughts, err := s.reflect(ctx, msgs)
	if err != nil {
		return "", err
	}

	snapshot := s.emoService.Snapshot()
	emotions := snapshot.AugmentationTop()

	return fmt.Sprintf("%s\n%s", emotions, toughts), nil
}

func (s *Service) reflect(ctx context.Context, msgs []types.Message) (string, error) {

	cut := tail(6, msgs)
	prompt, err := s.promptRenderer.Render(ReflectionParams{
		Personality: s.personality,
		Emotions:    s.emoService.Snapshot().AugmentationFull(),
	})
	if err != nil {
		return "", err
	}

	result, err := s.reasoningService.Reason(ctx, prompt, nil, cut)
	if err != nil {
		return "", err
	}

	return result[len(result)-1].Content(), nil
}

func tail[T any](tailSize int, arr []T) []T {
	limit := min(len(arr), tailSize)
	return arr[len(arr)-limit:]
}
