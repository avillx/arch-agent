package files

import (
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
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
		dir:   filepath.Clean(dir),
	}, nil
}

func (fs *FileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	unlock := fs.locks.RLock(path)
	defer unlock()

	res, err := os.ReadDir(fs.resolveAbsolutePath(path))
	return res, toInternalNotExist(err)
}

func (f *FileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	absPath := f.resolveAbsolutePath(root)
	return filepath.WalkDir(absPath, fn)
}

func (f *FileSystem) ToLocal(p string) (string, error) {
	cleanPath := filepath.Clean(p)
	after, found := strings.CutPrefix(cleanPath, f.dir)
	if !found {
		return "", fmt.Errorf("'%s' is not a abs path", cleanPath)
	}
	return after, nil
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
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return types.ErrIsNotExist
	}
	if errors.Is(err, os.ErrExist) {
		return types.ErrAlreadyExist
	}
	return err
}

func ensureFilePlaceholder(
	fs *FileSystem,
	pathToFile string,
	defaultEntry []byte,
) error {
	if _, err := fs.ReadFile(pathToFile); err != nil {
		if !errors.Is(err, types.ErrIsNotExist) {
			return err
		}
		if err := fs.WriteToFile(TaskFile, defaultEntry); err != nil {
			return err
		}
	}
	return nil
}

func resolveFrontmatter[T any](data []byte) (T, error) {
	var zero T
	const delim = "---"
	s := strings.ReplaceAll(string(data), "\r\n", "\n")

	after, ok := strings.CutPrefix(s, delim+"\n")
	if !ok {
		return zero, fmt.Errorf("hook file must start with ---")
	}

	fmEnd := strings.Index(after, "\n"+delim)
	if fmEnd == -1 {
		return zero, fmt.Errorf("unclosed frontmatter")
	}

	var dto T
	if err := yaml.Unmarshal([]byte(after[:fmEnd]), &dto); err != nil {
		return zero, err
	}

	return dto, nil
}
