package ruledfiles

import (
	"arch-agent/internal/files"
	"fmt"
	"os"
	"slices"
	"strings"
)

type Rules[T any] struct {
	rules []func(T) (T, error)
}

func NewRules[T any]() *Rules[T] {
	return &Rules[T]{
		rules: make([]func(T) (T, error), 0),
	}
}

func (f *Rules[T]) AddRule(r func(T) (T, error)) {
	f.rules = append(f.rules, r)
}

func (f *Rules[T]) Apply(value T) (T, error) {
	for _, v := range f.rules {
		newValue, err := v(value)
		if err != nil {
			return value, err
		}
		value = newValue
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
	readDirPath *Rules[string]
	readPath    *Rules[string]
	writePath   *Rules[string]
	appendPath  *Rules[string]
	deletePath  *Rules[string]

	// Data rules
	dirOutput    *Rules[DirResult]
	writeInput   *Rules[[]byte]
	appendOutput *Rules[[]byte]
	readOutput   *Rules[[]byte]
}

func NewRuledFileSystem(fs *files.FileSystem, opts ...Option) (*RuledFileSystem, error) {
	if fs == nil {
		return nil, fmt.Errorf("file system must be non nil")
	}

	rfs := &RuledFileSystem{
		fs:          fs,
		readDirPath: NewRules[string](),
		readPath:    NewRules[string](),
		writePath:   NewRules[string](),
		appendPath:  NewRules[string](),
		deletePath:  NewRules[string](),

		dirOutput:    NewRules[DirResult](),
		writeInput:   NewRules[[]byte](),
		appendOutput: NewRules[[]byte](),
		readOutput:   NewRules[[]byte](),
	}
	for _, opt := range opts {
		opt(rfs)
	}

	return rfs, nil
}

func (rfs *RuledFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	safePath, err := rfs.readDirPath.Apply(path)
	if err != nil {
		return nil, err
	}

	entries, err := rfs.fs.ReadDir(safePath)
	if err != nil {
		return nil, err
	}

	moded, err := rfs.dirOutput.Apply(DirResult{path: safePath, entries: entries})
	if err != nil {
		return nil, err
	}

	return moded.entries, nil
}

func (rfs *RuledFileSystem) ReadFile(path string) ([]byte, error) {
	safePath, err := rfs.readPath.Apply(path)
	if err != nil {
		return nil, err
	}
	data, err := rfs.fs.ReadFile(safePath)
	if err != nil {
		return nil, err
	}

	data, err = rfs.readOutput.Apply(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (rfs *RuledFileSystem) ReadLines(path string, from, to *int) (string, error) {
	safePath, err := rfs.readPath.Apply(path)
	if err != nil {
		return "", err
	}

	rawData, err := rfs.fs.ReadFile(safePath)
	if err != nil {
		return "", err
	}

	extract := extractLines(rawData, from, to)

	safeData, err := rfs.readOutput.Apply([]byte(extract))
	if err != nil {
		return "", err
	}

	return string(safeData), nil
}

func (rfs *RuledFileSystem) WriteFile(path string, data []byte) error {
	safePath, err := rfs.writePath.Apply(path)
	if err != nil {
		return err
	}
	safeData, err := rfs.writeInput.Apply(data)
	if err != nil {
		return err
	}
	return rfs.fs.WriteToFile(safePath, safeData)
}

func (rfs *RuledFileSystem) AppendToFile(path string, input []byte) error {
	safePath, err := rfs.appendPath.Apply(path)
	if err != nil {
		return err
	}

	data, err := rfs.fs.ReadFile(safePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		data = []byte{}
	}

	appendedData := slices.Concat(data, input)

	safeData, err := rfs.appendOutput.Apply(appendedData)
	if err != nil {
		return err
	}

	return rfs.fs.WriteToFile(safePath, safeData)
}

func (rfs *RuledFileSystem) Delete(path string) error {
	safePath, err := rfs.deletePath.Apply(path)
	if err != nil {
		return err
	}

	return rfs.fs.Delete(safePath)
}

func extractLines(data []byte, from, to *int) string {
	lines := strings.Split(string(data), "\n")
	total := len(lines)

	startLine := 1
	endLine := total

	if from != nil {
		startLine = *from
	}
	if to != nil {
		endLine = *to
	}

	startLine = max(1, min(startLine, total))
	endLine = max(startLine, min(endLine, total))

	return strings.Join(lines[startLine-1:endLine], "\n")
}
