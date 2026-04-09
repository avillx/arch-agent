package filestorage

import (
	"os"
	"path/filepath"
)

type dirBase struct {
	directory string
}

func (r dirBase) saveToFile(data []byte, filepath string) error {
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return err
	}
	return nil
}

func (d dirBase) path(filename string) string {
	return filepath.Join(d.directory, filename)
}

func (d dirBase) touchDir(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return nil
}
