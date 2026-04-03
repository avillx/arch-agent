package filestorage

import (
	"os"
	"path/filepath"
)

func touchPath(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return nil
}
