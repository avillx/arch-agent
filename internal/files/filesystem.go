package files

import (
	"arch-agent/internal/types"
	"os"
	"path/filepath"
)

const FileMode = 0644

type FileSystem struct {
	locks *lockTable
	dir   string
}

func NewFS(dir string) (*FileSystem, error) {
	if err := os.MkdirAll(dir, FileMode); err != nil {
		return nil, err
	}
	return &FileSystem{
		locks: newLockTable(),
		dir:   dir,
	}, nil
}

func (fs *FileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	unlock := fs.locks.RLock(path)
	defer unlock()

	res, err := os.ReadDir(fs.resolveAbsolutePath(path))
	return res, toInternalNotExist(err)
}

func (fs *FileSystem) ReadFile(path string) ([]byte, error) {
	unlock := fs.locks.RLock(path)
	defer unlock()

	res, err := os.ReadFile(fs.resolveAbsolutePath(path))
	return res, toInternalNotExist(err)
}

func (fs *FileSystem) WriteToFile(path string, data []byte) error {
	unlock := fs.locks.Lock(path)
	defer unlock()

	fullPath := fs.resolveAbsolutePath(path)

	if err := os.MkdirAll(filepath.Dir(fullPath), FileMode); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, FileMode)
}

func (fs *FileSystem) AppendToFile(path string, data []byte) error {
	unlock := fs.locks.Lock(path)
	defer unlock()

	fullPath := fs.resolveAbsolutePath(path)

	if err := os.MkdirAll(filepath.Dir(fullPath), FileMode); err != nil {
		return err
	}

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, FileMode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)

	return err
}

func (fs *FileSystem) Delete(path string) error {
	unlock := fs.locks.Lock(path)
	defer unlock()

	err := os.Remove(fs.resolveAbsolutePath(path))
	return toInternalNotExist(err)
}

func (fs *FileSystem) DeleteAll(path string) error {
	unlock := fs.locks.Lock(path)
	defer unlock()

	err := os.RemoveAll(fs.resolveAbsolutePath(path))
	return toInternalNotExist(err)
}

func (fs *FileSystem) Cwd() string {
	return fs.dir
}

func (fs *FileSystem) resolveAbsolutePath(relativePath string) string {
	return filepath.Join(fs.dir, relativePath)
}

func toInternalNotExist(err error) error {
	if os.IsNotExist(err) {
		return types.ErrIsNotExist
	}
	return err
}
