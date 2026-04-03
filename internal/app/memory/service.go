package memory

import (
	"errors"
	"strings"
)

type RunningMemoryProvider interface {
	Yesterday() (string, error)
	Today() (string, error)
}

type MemoryService struct {
	runningMemoryProvider RunningMemoryProvider
}

func NewMemoryService(p RunningMemoryProvider) *MemoryService {
	return &MemoryService{
		runningMemoryProvider: p,
	}
}

// TODO:
// - NO common method for retrivial Memory only separated method (because it use search)

func (s *MemoryService) RunningMemory() (string, error) {

	var sb strings.Builder
	var errs error

	yesterday, err := s.runningMemoryProvider.Yesterday()
	errs = errors.Join(errs, err)
	if yesterday != "" {
		_, err = sb.WriteString("# Yesterday:\n")
		errs = errors.Join(errs, err)
		_, err = sb.WriteString(yesterday)
		errs = errors.Join(errs, err)
	}

	today, err := s.runningMemoryProvider.Today()
	errors.Join(errs, err)
	if today != "" {
		_, err = sb.WriteString("# Today:\n")
		errs = errors.Join(errs, err)
		_, err = sb.WriteString(today)
		errs = errors.Join(errs, err)
	}

	return sb.String(), errs
}
