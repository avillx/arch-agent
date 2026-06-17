package ruledfiles

import (
	"arch-agent/internal/files"
	"os"
)

type Rules[T any] struct {
	validators   []func(T) error
	modificators []func(T) T
}

func NewRules[T any]() *Rules[T] {
	return &Rules[T]{
		modificators: make([]func(T) T, 0),
		validators:   make([]func(T) error, 0),
	}
}

func (f *Rules[T]) Apply(value T) (T, error) {
	for _, v := range f.validators {
		if err := v(value); err != nil {
			return value, err
		}
	}

	for _, m := range f.modificators {
		value = m(value)
	}

	return value, nil
}

type Option func(*RuledFileSystem)

type DirResult struct {
	path    string
	entries []os.DirEntry
}
type RuledFileSystem struct {
	fs *files.FileSystem

	// Path rules
	ReadPath   *Rules[string]
	WritePath  *Rules[string]
	AppendPath *Rules[string]
	DeletePath *Rules[string]

	// Data rules
	DirOutput   *Rules[DirResult]
	WriteInput  *Rules[[]byte]
	AppendInput *Rules[[]byte]
}

func NewRuledFileSystem(fs *files.FileSystem, opts ...Option) *RuledFileSystem {
	rfs := &RuledFileSystem{
		fs:         fs,
		ReadPath:   NewRules[string](),
		WritePath:  NewRules[string](),
		AppendPath: NewRules[string](),
		DeletePath: NewRules[string](),

		DirOutput:   NewRules[DirResult](),
		WriteInput:  NewRules[[]byte](),
		AppendInput: NewRules[[]byte](),
	}
	for _, opt := range opts {
		opt(rfs)
	}
	return rfs
}

func (rfs *RuledFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	safePath, err := rfs.ReadPath.Apply(path)
	if err != nil {
		return nil, err
	}

	entries, err := rfs.fs.ReadDir(safePath)
	if err != nil {
		return nil, err
	}

	moded, err := rfs.DirOutput.Apply(DirResult{path: path, entries: entries})
	if err != nil {
		return nil, err
	}

	return moded.entries, nil
}

func (rfs *RuledFileSystem) ReadFile(path string) ([]byte, error) {
	safePath, err := rfs.ReadPath.Apply(path)
	if err != nil {
		return nil, err
	}
	data, err := rfs.fs.ReadFile(safePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (rfs *RuledFileSystem) WriteFile(path string, data []byte) error {
	safePath, err := rfs.WritePath.Apply(path)
	if err != nil {
		return err
	}
	safeData, err := rfs.WriteInput.Apply(data)
	if err != nil {
		return err
	}
	return rfs.fs.WriteToFile(safePath, safeData)
}

func (rfs *RuledFileSystem) AppendToFile(path string, data []byte) error {
	safePath, err := rfs.AppendPath.Apply(path)
	if err != nil {
		return err
	}
	safeData, err := rfs.AppendInput.Apply(data)
	if err != nil {
		return err
	}

	return rfs.fs.WriteToFile(safePath, safeData)
}

func (rfs *RuledFileSystem) Delete(path string) error {
	safePath, err := rfs.DeletePath.Apply(path)
	if err != nil {
		return err
	}

	return rfs.fs.Delete(safePath)
}
