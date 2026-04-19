package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSystem struct {
	dir string
}

func New(dir string) (FileSystem, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return FileSystem{}, err
	}
	return FileSystem{dir: dir}, nil
}

func (fs FileSystem) ReadDir() ([]string, error) {
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func (fs FileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(fs.pathTo(name))
}

func (fs FileSystem) WriteToFile(name string, data []byte) error {
	path := fs.pathTo(name)
	if err := fs.touchFile(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (fs FileSystem) AppendToFile(name string, data []byte) error {
	f, err := os.OpenFile(fs.pathTo(name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("can't open file %s: %w", name, err)
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (fs FileSystem) ReplaceInFile(name, old, new string) error {
	data, err := os.ReadFile(fs.pathTo(name))
	if err != nil {
		return err
	}
	return fs.WriteToFile(name, []byte(strings.Replace(string(data), old, new, 1)))
}

func (fs FileSystem) DeleteFile(name string) error {
	return os.Remove(fs.pathTo(name))
}

func (fs FileSystem) RenameFile(name, newName string) error {
	return os.Rename(fs.pathTo(name), fs.pathTo(newName))
}

// helpers
func (fs FileSystem) pathTo(name string) string {
	return filepath.Join(fs.dir, name)
}

func (fs FileSystem) touchFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return f.Close()
}
