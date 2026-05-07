package secretfiles

import (
	"arch-agent/internal/infra/storage/filesystem"
	"encoding/json"
	"maps"
	"os"
	"slices"
	"sync"
)

const secretsFile = ".secrets"

type Storage struct {
	mu      sync.RWMutex
	secrets map[string]string
	fs      filesystem.FileSystem
}

func New(dir string) (*Storage, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}

	sf := &Storage{
		fs: fs,
	}

	if err := sf.load(); err != nil {
		return nil, err
	}

	return sf, nil
}

func (sf *Storage) load() error {
	data, err := sf.fs.ReadFile(secretsFile)
	if err != nil && os.IsNotExist(err) {
		return sf.fs.WriteToFile(secretsFile, []byte{})
	}
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &sf.secrets); err != nil {
		return err
	}

	return nil
}

func (sf *Storage) GetNames() []string {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	return slices.Collect(maps.Keys(sf.secrets))
}

func (sf *Storage) Remove(name string) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	delete(sf.secrets, name)

	return sf.saveSecrets()
}

func (sf *Storage) Get(name string) (string, bool) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	val, ok := sf.secrets[name]
	return val, ok
}

func (sf *Storage) Set(name, value string) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.secrets[name] = value
	return sf.saveSecrets()
}

func (sf *Storage) saveSecrets() error {
	data, err := json.Marshal(sf.secrets)
	if err != nil {
		return err
	}
	return sf.fs.WriteToFile(secretsFile, data)
}
