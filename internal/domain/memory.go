package domain

import (
	"fmt"
	"slices"
	"time"
)

type SemanticData struct {
	// some fact or anythin short and importnat 2-3 sentences
	Data string

	// origin of this fact
	From Episode
}

func (d SemanticData) String() string {
	return fmt.Sprintf("{\"%s\":\"%s\"}", d.Data, d.From.String())
}

// is a Running memory that saved in database
type Episode struct {
	// when this Episode created for temporal search and additional context for model
	TimeStamp time.Time

	Content string
}

func (e Episode) String() string {
	return fmt.Sprintf("{\"%s\":\"%s\"}", e.TimeStamp.String(), e.Content)
}

type Memory struct {

	// semantic memories
	Semantic []SemanticData

	// Recent is last 2-3 conver
	RecentEpisodes []Episode

	// top 3 matches that found by temporal (data range) search
	RelevantEpisodes []Episode

	// is a concated multiple summaries of dialog
	RunningMemory string
}

func NewMemory(sd []SemanticData, recEp []Episode, relEp []Episode, runningMemory string) *Memory {
	// similar episodes should be eliminated before this constructor
	return &Memory{
		Semantic:         sd,
		RecentEpisodes:   recEp,
		RelevantEpisodes: relEp,
		RunningMemory:    runningMemory,
	}
}

func (m *Memory) sortedEpisodes() []Episode {
	eps := append(m.RelevantEpisodes, m.RecentEpisodes...)
	slices.SortFunc(eps, func(a, b Episode) int {
		return a.TimeStamp.Compare(b.TimeStamp)
	})
	return eps
}

func (m *Memory) Augmentation() string {
	// strings builder is not used becouse it can return errors
	// errors should be minimized in domain layer
	result := ""

	for _, s := range m.Semantic {
		result += s.String()
	}

	for _, e := range m.sortedEpisodes() {
		result += e.String()
	}

	return result
}
