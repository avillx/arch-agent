package reflection

import (
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/types"
	"context"
)

type ReflectionParams struct {
	Personality string
}

type PromptRenderer interface {
	Render(ReflectionParams) (string, error)
}

type Service struct {
	personality      string
	promptRenderer   PromptRenderer
	reasoningService *reasoning.Service
}

func NewService(
	personality string,
	promptRenderer PromptRenderer,
	reasoningService *reasoning.Service,
) *Service {
	return &Service{
		personality:      personality,
		promptRenderer:   promptRenderer,
		reasoningService: reasoningService,
	}
}

func (s *Service) Reflect(ctx context.Context, msgs []types.Message) (string, error) {

	cut := tail(6, msgs)
	prompt, err := s.promptRenderer.Render(ReflectionParams{Personality: s.personality})
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
