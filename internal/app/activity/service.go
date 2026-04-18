package activity

import (
	"arch-agent/internal/app/summarization"
	"arch-agent/internal/app/types"
	"context"
	"errors"
	"time"
)

const TimeFormat = "15:04"

var ErrNotFound = errors.New("Activity on date is not found")

type ActivityStorage interface {
	Log(Record) error
	GetActivity(date time.Time) (string, error)
}

type Service struct {
	summarizer *summarization.Service
	storage    ActivityStorage
}

func NewService(summarizer *summarization.Service, storage ActivityStorage) *Service {
	return &Service{
		summarizer: summarizer,
		storage:    storage,
	}
}

func (s *Service) GetActivity(date time.Time) (string, error) {
	return s.storage.GetActivity(date)
}

func (s *Service) LogActiviy(ctx context.Context, msgs []types.Message) error {
	result, err := s.summarizer.Summarize(ctx, msgs)
	if err != nil {
		return err
	}

	record := NewRecord(result)
	return s.storage.Log(record)
}
