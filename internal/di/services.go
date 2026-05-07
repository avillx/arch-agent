package di

import (
	"arch-agent/internal/app/activity"
	"arch-agent/internal/app/knowledge"
	"arch-agent/internal/app/reasoning"
	"arch-agent/internal/app/session"
	"arch-agent/internal/app/summarization"
	"arch-agent/internal/app/tools"
	openaiadapter "arch-agent/internal/infra/openai"
	activityfiles "arch-agent/internal/infra/storage/activity"
	knowledgefiles "arch-agent/internal/infra/storage/knowledge"
	"arch-agent/internal/infra/tokenizer"
)

// activity
func BuildActivityService(
	summarizationService *summarization.Service,
	activityStorage *activityfiles.Storage,
) *activity.Service {
	return activity.NewService(
		summarizationService,
		activityStorage,
	)
}

// kownledge
func BuildKnowledgeService(storage *knowledgefiles.Storage) *knowledge.Service {
	return knowledge.NewService(
		storage,
		tokenizer.NewTokenizer(),
	)
}

// reasoning
func BuildReasoningService(
	reasoningSettings openaiadapter.SettingsRepo,
	secrets openaiadapter.SecretsStorage,
	maxFollowUps int,
) (*reasoning.Service, error) {

	reasoner, err := openaiadapter.NewReasoner(reasoningSettings, secrets)
	if err != nil {
		return nil, err
	}

	return reasoning.NewService(
		maxFollowUps,
		reasoner,
	), nil
}

// session
func BuildSessionService(
	repo session.SessionRepository,
	tr session.Transcriptor,
	as *activity.Service,
) *session.Service {
	return session.NewSessionService(
		repo,
		tr,
		tokenizer.NewTokenizer(),
		as,
	)
}

// summarization
func BuildSummarizationService(
	sumSettings openaiadapter.SettingsRepo,
	secretsRepo openaiadapter.SecretsStorage,
) (*summarization.Service, error) {

	reasoningService, err := BuildReasoningService(sumSettings, secretsRepo, 5)
	if err != nil {
		return nil, err
	}

	return summarization.NewService(
		reasoningService,
	), nil
}

// tools
func BuildToolService(servers []tools.Server) *tools.Service {
	orchestra := tools.NewToolService()
	for _, s := range servers {
		orchestra.Connect(s)
	}
	return orchestra
}
