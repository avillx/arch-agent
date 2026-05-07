package di

import (
	activityfiles "arch-agent/internal/infra/storage/activity"
	agentfiles "arch-agent/internal/infra/storage/agent"
	knowledgefiles "arch-agent/internal/infra/storage/knowledge"
	secretfiles "arch-agent/internal/infra/storage/secrets"
	sessionfiles "arch-agent/internal/infra/storage/session"
	settingfiles "arch-agent/internal/infra/storage/settings"
	transcribtionfiles "arch-agent/internal/infra/storage/transcribtions"
	"path/filepath"
)

type StoragesDI struct {
	Secret         *secretfiles.Storage
	Setting        *settingfiles.Storage
	Session        *sessionfiles.Storage
	Transcriptions *transcribtionfiles.Storage
	Agent          *agentfiles.Storage
	Knowledge      *knowledgefiles.Storage
	Activity       *activityfiles.Storage
}

func BuildFileStorage(datapath string) (*StoragesDI, error) {
	absoluteDataPath, _ := filepath.Abs(datapath)

	secretFiles, err := secretfiles.New(absoluteDataPath)
	if err != nil {
		return nil, err
	}

	settingsFiles, err := settingfiles.New(absoluteDataPath)
	if err != nil {
		return nil, err
	}

	sessionFiles, err := sessionfiles.New(absoluteDataPath)
	if err != nil {
		return nil, err
	}

	transcribtionFiles, err := transcribtionfiles.New(absoluteDataPath + "/transciptions")
	if err != nil {
		return nil, err
	}

	agentFiles, err := agentfiles.New(absoluteDataPath + "/agent")
	if err != nil {
		return nil, err
	}

	knowledgeFiles, err := knowledgefiles.New(absoluteDataPath + "/knowledges")
	if err != nil {
		return nil, err
	}

	activityFiles, err := activityfiles.New(absoluteDataPath + "/activity")
	if err != nil {
		return nil, err
	}

	return &StoragesDI{
		Secret:         secretFiles,
		Setting:        settingsFiles,
		Session:        sessionFiles,
		Transcriptions: transcribtionFiles,
		Agent:          agentFiles,
		Knowledge:      knowledgeFiles,
		Activity:       activityFiles,
	}, nil
}
