package files

import (
	"arch-agent/internal/types"
	"bytes"
	"errors"
	"maps"
	"slices"
	"sync"

	toml "github.com/pelletier/go-toml/v2"
)

const SecretsFile = "secrets.toml"
const secretsFileDoc = `# Secrets storage

# Key names should prefer upper snake case (e.g. SOME_VARIABLE)
# Variables must be unique, file format is toml, one variable per line
# SOME_VARIABLE='sk-100abc200'

# Content is hidden from the agent behind placeholders (e.g. { secret.SOME_VARIABLE } )
# Whenever the agent uses the { secret.SOME_VARIABLE } placeholder,
# it gets swapped for the actual content from this file

# In shell, these secrets are available as environment variables`

type SecretsFiles struct {
	mu      sync.RWMutex
	secrets map[string]string
	fs      *FileSystem
}

func NewSecretsFiles(fs *FileSystem) (*SecretsFiles, error) {

	if err := ensureFilePlaceholder(fs, SecretsFile, []byte(secretsFileDoc)); err != nil {
		return nil, err
	}

	sf := &SecretsFiles{
		fs: fs,
	}

	secrets, err := sf.load()
	if err != nil {
		return nil, err
	}

	sf.secrets = secrets
	return sf, nil
}

func (sf *SecretsFiles) GetNames() []string {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	return slices.Collect(maps.Keys(sf.secrets))
}

func (sf *SecretsFiles) Remove(name string) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	delete(sf.secrets, name)
	return sf.save()
}

func (sf *SecretsFiles) Get(name string) (string, bool) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	val, ok := sf.secrets[name]
	return val, ok
}

func (sf *SecretsFiles) Set(name, value string) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.secrets[name] = value
	return sf.save()
}

func (sf *SecretsFiles) load() (map[string]string, error) {
	data, err := sf.fs.ReadFile(SecretsFile)
	if err != nil {
		if !errors.Is(err, types.ErrIsNotExist) {
			return nil, err
		}

		if err := ensureFilePlaceholder(sf.fs, SecretsFile, []byte(secretsFileDoc)); err != nil {
			return nil, err
		}

		return map[string]string{}, nil
	}

	var secrets map[string]string
	if err := toml.Unmarshal(data, &secrets); err != nil {
		return nil, err
	}
	return secrets, nil
}

func (sf *SecretsFiles) save() error {
	data, err := toml.Marshal(sf.secrets)
	if err != nil {
		return err
	}

	dataWithDoc := bytes.Join(
		[][]byte{[]byte(secretsFileDoc), data},
		[]byte("\n\n"),
	)

	return sf.fs.WriteToFile(SecretsFile, dataWithDoc)
}
