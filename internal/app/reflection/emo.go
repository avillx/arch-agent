package reflection

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"
)

type EmotionalService struct {
	state EmoState
	mu    sync.RWMutex
}

func NewEmotionalService() *EmotionalService {
	service := &EmotionalService{
		state: EmoState{
			"joy":            0.0,
			"hope":           0.0,
			"relief":         0.0,
			"pride":          0.0,
			"gratitude":      0.0,
			"love":           0.0,
			"distress":       0.0,
			"fear":           0.0,
			"disappointment": 0.0,
			"remorse":        0.0,
			"anger":          0.0,
			"hate":           0.0,
			// "joy":      0.0,
			// "trust":    0.0,
			// "fear":     0.0,
			// "surprise": 0.0,
			// "sadness":  0.0,
			// "disgust":  0.0,
			// "anger":    0.0,
			// "shame":    0.0,
		},
	}
	service.run(context.Background()) // TODO - Пробросить основной контекст
	return service
}

func (s *EmotionalService) Emotions() []string {
	return slices.Collect(maps.Keys(s.state))
}

func (s *EmotionalService) Boost(name string, val float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.state[name]; ok {
		s.state[name] += val
		return nil
	}
	return fmt.Errorf("emotion %s is not exist", name)
}

func (s *EmotionalService) Snapshot() EmoState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state
}

func (s *EmotionalService) run(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.evaluate()
				slog.Debug("emotion service", "state", s.Snapshot())
			}
		}
	}()
}

func (s *EmotionalService) evaluate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range s.state {
		s.state[k] = v + (0 - v/50)
	}
}
