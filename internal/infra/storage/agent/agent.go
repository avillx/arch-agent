package agentfiles

import (
	"arch-agent/internal/infra/storage/filesystem"
	"os"
	"sync"
)

const (
	roleFile        = "role.md"
	personalityFile = "personality.md"
	toneFile        = "tone.md"
)

type Storage struct {
	fs            filesystem.FileSystem
	roleMU        sync.RWMutex
	personalityMU sync.RWMutex
	toneMU        sync.RWMutex
}

func New(dir string) (*Storage, error) {
	fs, err := filesystem.New(dir)
	if err != nil {
		return nil, err
	}
	return &Storage{
		fs: fs,
	}, nil
}

func (f *Storage) Role() (string, error) {
	f.roleMU.RLock()
	defer f.roleMU.RUnlock()

	return f.readTextFile(roleFile)
}
func (f *Storage) Tone() (string, error) {
	f.toneMU.RLock()
	defer f.toneMU.RUnlock()

	return f.readTextFile(toneFile)
}
func (f *Storage) Personality() (string, error) {
	f.personalityMU.RLock()
	defer f.personalityMU.RUnlock()

	return f.readTextFile(personalityFile)
}

func (f *Storage) readTextFile(filename string) (string, error) {
	data, err := f.fs.ReadFile(filename)
	if err != nil && os.IsNotExist(err) {
		return "", f.fs.WriteToFile(filename, []byte{})
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f *Storage) SetRole(content string) error {
	f.roleMU.Lock()
	defer f.roleMU.Unlock()

	return f.fs.WriteToFile(roleFile, []byte(content))
}
func (f *Storage) SetTone(content string) error {
	f.toneMU.Lock()
	defer f.toneMU.Unlock()

	return f.fs.WriteToFile(toneFile, []byte(content))
}
func (f *Storage) SetPersonality(content string) error {
	f.personalityMU.Lock()
	defer f.personalityMU.Unlock()

	return f.fs.WriteToFile(personalityFile, []byte(content))
}
