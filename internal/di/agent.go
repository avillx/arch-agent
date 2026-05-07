package di

import (
	"arch-agent/internal/app/agent"
	"arch-agent/internal/app/tools"
	"arch-agent/internal/infra/config"
)

// TODO
// - On di assmbly use only Ifaces for strages
// - expose storages out of UC builder as iface data struct {Storageiface1,Storageiface2}
func BuildArchAgent(cfg config.Config, dataPath string, externalServers []tools.Server) (*agent.ArchAgent, error) {

	fileStorage, err := BuildFileStorage(dataPath)
	if err != nil {
		return nil, err
	}

	summarizationService, err := BuildSummarizationService(
		fileStorage.Setting.SummarizationSettings,
		fileStorage.Secret,
	)
	if err != nil {
		return nil, err
	}

	activityService := BuildActivityService(summarizationService, fileStorage.Activity)

	sessionService := BuildSessionService(
		fileStorage.Session,
		fileStorage.Transcriptions,
		activityService,
	)

	knowledgeService := BuildKnowledgeService(fileStorage.Knowledge)

	toolServers := append(
		[]tools.Server{tools.NewKnowledgeReaderTS(knowledgeService)},
		externalServers...)

	toolService := BuildToolService(toolServers)

	reasoningService, err := BuildReasoningService(
		fileStorage.Setting.ReasoningSettings,
		fileStorage.Secret,
		10,
	)
	if err != nil {
		return nil, err
	}

	return agent.NewArchAgent(
		"",
		fileStorage.Agent,
		activityService,
		reasoningService,
		sessionService,
		toolService,
	), nil
}
