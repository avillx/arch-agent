package ruledfiles

import "os"

var _ os.DirEntry = (*VirtualDirEntry)(nil)

type VirtualDirEntry struct {
	name  string
	isDir bool
}

func (e *VirtualDirEntry) Name() string               { return e.name }
func (e *VirtualDirEntry) IsDir() bool                { return e.isDir }
func (e *VirtualDirEntry) Type() os.FileMode          { return os.ModePerm }
func (e *VirtualDirEntry) Info() (os.FileInfo, error) { return nil, nil }
