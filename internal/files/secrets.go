package files

import (
	"arch-agent/internal/secrets"
	"bytes"

	toml "github.com/pelletier/go-toml/v2"
)

const SecretsFile = "secrets.toml"
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
	fs *FileSystem
}

func NewSecretsFiles(fs *FileSystem) (*SecretsFiles, error) {

	if err := ensureFilePlaceholder(fs, SecretsFile, []byte(secretsFileDoc)); err != nil {
		return nil, err
	}

	return &SecretsFiles{
		fs: fs,
	}, nil
}

func (sf *SecretsFiles) Load() (map[string]string, error) {
	data, err := sf.fs.ReadFile(SecretsFile)
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

	return sf.fs.WriteToFile(SecretsFile, dataWithDoc)
}
