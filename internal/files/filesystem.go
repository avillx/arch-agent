package files

import (
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type FileMode = fs.FileMode
type FileInfo = os.FileInfo

const (
	ModeFilePerm = 0644
	ModeDirPerm  = 0755

	ModePerm       = os.ModePerm
	ModeAppend     = os.ModeAppend
	ModeExclusive  = os.ModeExclusive
	ModeTemporary  = os.ModeTemporary
	ModeSymlink    = os.ModeSymlink
	ModeDevice     = os.ModeDevice
	ModeNamedPipe  = os.ModeNamedPipe
	ModeSocket     = os.ModeSocket
	ModeSetuid     = os.ModeSetuid
	ModeSetgid     = os.ModeSetgid
	ModeCharDevice = os.ModeCharDevice
	ModeSticky     = os.ModeSticky
	ModeIrregular  = os.ModeIrregular

	O_RDONLY = os.O_RDONLY
	O_WRONLY = os.O_WRONLY
	O_RDWR   = os.O_RDWR
	O_APPEND = os.O_APPEND
	O_CREATE = os.O_CREATE
	O_EXCL   = os.O_EXCL
	O_SYNC   = os.O_SYNC
	O_TRUNC  = os.O_TRUNC
)

type File struct {
	*os.File
	unlockFunc func()
}

func (f *File) Close() error {
	defer f.unlockFunc()
	return f.File.Close()
}

type FileStorage interface {
	Chmod(name string, mode FileMode) error
	Chown(name string, uid int, gid int) error
	Chtimes(name string, atime time.Time, mtime time.Time) error
	Close() error
	Create(name string) (*File, error)
	FS() fs.FS
	Lchown(name string, uid int, gid int) error
	Link(oldname string, newname string) error
	Lstat(name string) (FileInfo, error)
	Mkdir(name string, perm FileMode) error
	MkdirAll(name string, perm FileMode) error
	Name() string
	Open(name string) (*File, error)
	OpenFile(name string, flag int, perm FileMode) (*File, error)
	// OpenRoot(name string) (*fileStorage, error)
	ReadFile(name string) ([]byte, error)
	Readlink(name string) (string, error)
	Remove(name string) error
	RemoveAll(name string) error
	Rename(oldname string, newname string) error
	Stat(name string) (FileInfo, error)
	Symlink(oldname string, newname string) error
	WriteFile(name string, data []byte, perm FileMode) error
}

var _ FileStorage = (*fileStorage)(nil)

type fileStorage struct {
	root  *os.Root
	flock *lockTable
}

func NewFileStorage(dir string) (*fileStorage, error) {

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}

	return &fileStorage{
		root:  root,
		flock: newLockTable(),
	}, nil
}

func (r *fileStorage) Chmod(name string, mode FileMode) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Chmod(name, mode)
}

func (r *fileStorage) Chown(name string, uid int, gid int) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Chown(name, uid, gid)
}

func (r *fileStorage) Chtimes(name string, atime time.Time, mtime time.Time) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Chtimes(name, atime, mtime)
}

func (r *fileStorage) Close() error {
	return r.root.Close()
}

func (r *fileStorage) Create(name string) (*File, error) {
	unlock := r.flock.Lock(name)
	defer unlock()

	f, err := r.root.Create(name)
	if err != nil {
		return nil, err
	}

	return &File{unlockFunc: unlock, File: f}, nil
}

func (r *fileStorage) FS() fs.FS {
	return r.root.FS()
}

func (r *fileStorage) Lchown(name string, uid int, gid int) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Lchown(name, uid, gid)
}

func (r *fileStorage) Link(oldname string, newname string) error {
	unlockOld := r.flock.Lock(oldname)
	defer unlockOld()

	unlockNew := r.flock.Lock(newname)
	defer unlockNew()

	return r.root.Link(oldname, newname)
}

func (r *fileStorage) Lstat(name string) (FileInfo, error) {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Lstat(name)
}

func (r *fileStorage) Mkdir(name string, perm FileMode) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Mkdir(name, perm)
}

func (r *fileStorage) MkdirAll(name string, perm FileMode) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.MkdirAll(name, perm)
}

func (r *fileStorage) Name() string {
	return r.root.Name()
}

func (r *fileStorage) Open(name string) (*File, error) {
	unlock := r.flock.Lock(name)

	f, err := r.root.Open(name)
	if err != nil {
		unlock()
		return nil, err
	}

	return &File{unlockFunc: unlock, File: f}, nil
}

func (r *fileStorage) OpenFile(name string, flag int, perm FileMode) (*File, error) {
	unlock := r.flock.Lock(name)

	f, err := r.root.OpenFile(name, flag, perm)
	if err != nil {
		unlock()
		return nil, err
	}

	return &File{unlockFunc: unlock, File: f}, nil
}

func (r *fileStorage) ReadFile(name string) ([]byte, error) {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.ReadFile(name)
}

func (r *fileStorage) Readlink(name string) (string, error) {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Readlink(name)
}

func (r *fileStorage) Remove(name string) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Remove(name)
}

func (r *fileStorage) RemoveAll(name string) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.RemoveAll(name)
}

func (r *fileStorage) Rename(oldname string, newname string) error {
	unlockOld := r.flock.Lock(oldname)
	defer unlockOld()

	unlockNew := r.flock.Lock(newname)
	defer unlockNew()

	return r.root.Rename(oldname, newname)
}

func (r *fileStorage) Stat(name string) (FileInfo, error) {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.Stat(name)
}

func (r *fileStorage) Symlink(oldname string, newname string) error {
	unlockOld := r.flock.Lock(oldname)
	defer unlockOld()

	unlockNew := r.flock.Lock(newname)
	defer unlockNew()

	return r.root.Symlink(oldname, newname)
}

func (r *fileStorage) WriteFile(name string, data []byte, perm FileMode) error {
	unlock := r.flock.Lock(name)
	defer unlock()

	return r.root.WriteFile(name, data, perm)
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
	root FileStorage,
	pathToFile string,
	defaultEntry []byte,
) error {
	if _, err := root.ReadFile(pathToFile); err != nil {

		if !errors.Is(err, types.ErrIsNotExist) {
			return err
		}
		if err := root.WriteFile(pathToFile, defaultEntry, ModePerm); err != nil {
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
