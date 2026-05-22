package files

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"sync"
)

type SecretsFiles struct {
	mu      sync.RWMutex
	secrets map[string]string
	fs      *FileSystem
}

func NewSecretsFiles(fs *FileSystem) (*SecretsFiles, error) {
	sf := &SecretsFiles{
		fs: fs,
	}
	if err := sf.load(); err != nil {
		return nil, err
	}
	return sf, nil
}

func (sf *SecretsFiles) load() error {
	data, err := sf.fs.ReadFile(".secrets")
	if err != nil && os.IsNotExist(err) {
		return sf.fs.WriteToFile(".secrets", []byte{})
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &sf.secrets); err != nil {
		return err
	}
	return nil
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

func (sf *SecretsFiles) save() error {
	data, err := json.MarshalIndent(sf.secrets, "", "	")
	if err != nil {
		return err
	}
	return sf.fs.WriteToFile(".secrets", data)
}
