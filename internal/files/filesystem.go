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

var _ fs.FS = (*FileSystem)(nil)

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

func (fs *FileSystem) ReadDir(p string) ([]os.DirEntry, error) {
	unlock := fs.locks.RLock(p)
	defer unlock()

	p, err := fs.resolvePath(p)
	if err != nil {
		return nil, err
	}

	res, err := os.ReadDir(p)
	return res, toInternalNotExist(err)
}

func (fs *FileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	p, err := fs.resolvePath(root)
	if err != nil {
		return err
	}

	return filepath.WalkDir(p, fn)
}

func (f *FileSystem) ToAbs(p string) (string, error) {
	cleanPath := filepath.Clean(p)
	if filepath.IsAbs(cleanPath) {
		return cleanPath, nil
	}
	return filepath.Join(f.dir, cleanPath), nil
}

func (f *FileSystem) ToLocal(p string) (string, error) {
	cleanPath := filepath.Clean(p)
	after, found := strings.CutPrefix(cleanPath, f.dir)
	if !found {
		return "", fmt.Errorf("'%s' is not a abs path", cleanPath)
	}
	return after, nil
}

func (fs *FileSystem) Rename(old, new string) error {
	unlockNew := fs.locks.RLock(new)
	defer unlockNew()

	unlockOld := fs.locks.RLock(old)
	defer unlockOld()

	new, err := fs.resolvePath(new)
	if err != nil {
		return err
	}

	old, err = fs.resolvePath(old)
	if err != nil {
		return err
	}

	if err := os.Rename(old, new); err != nil {
		return toInternalNotExist(err)
	}

	return nil
}

func (fs *FileSystem) ReadFile(p string) ([]byte, error) {
	unlock := fs.locks.RLock(p)
	defer unlock()

	p, err := fs.resolvePath(p)
	if err != nil {
		return nil, err
	}

	res, err := os.ReadFile(p)
	return res, toInternalNotExist(err)
}

func (fs *FileSystem) WriteToFile(p string, data []byte) error {
	unlock := fs.locks.Lock(p)
	defer unlock()

	p, err := fs.resolvePath(p)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), FileMode); err != nil {
		return err
	}

	return os.WriteFile(p, data, FileMode)
}

func (fs *FileSystem) AppendToFile(p string, data []byte) error {
	unlock := fs.locks.Lock(p)
	defer unlock()

	p, err := fs.resolvePath(p)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), FileMode); err != nil {
		return err
	}

	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, FileMode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)

	return err
}

type openedFile struct {
	fs.File
	unlockFunc func()
}

func (f *openedFile) Close() error {
	defer f.unlockFunc()
	return f.File.Close()
}

func (fs *FileSystem) Open(p string) (fs.File, error) {

	p, err := fs.resolvePath(p)
	if err != nil {
		return nil, err
	}

	unlock := fs.locks.Lock(p)

	file, err := os.Open(p)
	if err != nil {
		unlock()
		return nil, toInternalNotExist(err)
	}

	return &openedFile{
		File:       file,
		unlockFunc: unlock,
	}, nil
}

func (fs *FileSystem) Info(p string) (os.FileInfo, error) {
	unlock := fs.locks.Lock(p)
	defer unlock()

	p, err := fs.resolvePath(p)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(p)
	if err != nil {
		return nil, toInternalNotExist(err)
	}

	return info, nil
}

func (fs *FileSystem) Delete(p string) error {
	unlock := fs.locks.Lock(p)
	defer unlock()

	p, err := fs.resolvePath(p)
	if err != nil {
		return err
	}

	err = os.Remove(p)
	return toInternalNotExist(err)
}

func (fs *FileSystem) MkdirAll(p string) error {

	p, err := fs.resolvePath(p)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(p, FileMode); err != nil {
		return toInternalNotExist(err)
	}
	return nil
}

func (fs *FileSystem) DeleteAll(p string) error {
	unlock := fs.locks.Lock(p)
	defer unlock()

	p, err := fs.resolvePath(p)
	if err != nil {
		return err
	}

	err = os.RemoveAll(p)
	return toInternalNotExist(err)
}

func (fs *FileSystem) Cwd() string {
	return fs.dir
}

func (fs *FileSystem) resolvePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		localPath, err := fs.ToLocal(p)
		if err != nil {
			return "", err
		}
		p = localPath
	}
	return filepath.Join(fs.dir, p), nil
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
