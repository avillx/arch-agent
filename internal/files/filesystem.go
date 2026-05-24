package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type FileSystem struct {
	locks *lockTable
	dir   string
}

func NewFS(dir string) (*FileSystem, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &FileSystem{
		locks: newLockTable(),
		dir:   dir,
	}, nil
}

func (fs *FileSystem) ReadDir(path string) ([]string, error) {
	entries, err := os.ReadDir(fs.pathTo(path))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func (fs *FileSystem) ReadFile(path string) ([]byte, error) {
	e := fs.locks.RLock(path)
	defer fs.locks.RUnlock(path, e)

	return os.ReadFile(fs.pathTo(path))
}

func (fs *FileSystem) WriteToFile(path string, data []byte) error {
	e := fs.locks.Lock(path)
	defer fs.locks.Unlock(path, e)

	fullPath := fs.pathTo(path)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {

		var pathErr *os.PathError
		if !errors.As(err, &pathErr) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(fullPath, data, 0644)
	}
	return nil
}

func (fs *FileSystem) AppendToFile(path string, data []byte) error {
	e := fs.locks.RLock(path)
	defer fs.locks.RUnlock(path, e)

	f, err := os.OpenFile(fs.pathTo(path), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("can't open file %s: %w", path, err)
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (fs *FileSystem) DeleteFile(path string) error {
	e := fs.locks.RLock(path)
	defer fs.locks.RUnlock(path, e)

	return os.Remove(fs.pathTo(path))
}

func (fs *FileSystem) Dir() string {
	return fs.dir
}

func (fs *FileSystem) pathTo(name string) string {
	return filepath.Join(fs.dir, name)
}

func (fs *FileSystem) touchFile(path string) error {
	f, err := os.OpenFile(fs.pathTo(path), os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return f.Close()
}
