package files

import (
	"arch-agent/internal/secrets"
	"arch-agent/internal/sentinel"
	"bytes"
	"context"

	"github.com/fsnotify/fsnotify"
	toml "github.com/pelletier/go-toml/v2"
)

const SecretsConfigFile = "secrets.toml"
const secretsFileDoc = `# Secrets storage

# Key names should prefer upper snake case (e.g. SOME_VARIABLE)
# Variables must be unique, file format is toml, one variable per line
# SOME_VARIABLE='sk-100abc200'

# Content is hidden from the agent behind placeholders (e.g. { secret.SOME_VARIABLE } )

# In shell, these secrets are available as environment variables

# Do not touch this comment!
# After edit, ensure file consistency and comment integrity`

var _ secrets.Repo = (*SecretsFiles)(nil)

type SecretsFiles struct {
	storage FileStorage
}

func NewSecretsFiles(storage FileStorage) (*SecretsFiles, error) {

	if err := ensureFilePlaceholder(storage, SecretsConfigFile, []byte(secretsFileDoc)); err != nil {
		return nil, err
	}

	return &SecretsFiles{
		storage: storage,
	}, nil
}

func (sf *SecretsFiles) Load() (map[string]string, error) {
	data, err := sf.storage.ReadFile(SecretsConfigFile)
	if err != nil {
		return nil, err
	}

	var secrets map[string]string
	if err := toml.Unmarshal(data, &secrets); err != nil {
		return nil, err
	}
	return secrets, nil
}

func (sf *SecretsFiles) Save(secrets map[string]string) error {
	data, err := toml.Marshal(secrets)
	if err != nil {
		return err
	}

	dataWithDoc := bytes.Join(
		[][]byte{[]byte(secretsFileDoc), data},
		[]byte("\n\n"),
	)

	return sf.storage.WriteFile(SecretsConfigFile, dataWithDoc, ModeFilePerm)
}

// secretsSent
func NewSecretsReloader(secretSvc *secrets.Service) sentinel.Action {
	return func(ctx context.Context, ev fsnotify.Event) error {
		return secretSvc.Reload(ctx)
	}
}
