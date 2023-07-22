package controller

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
)

// FSConfig encapsulates the configuration required to establish the root
// directory of the wazero runtime when performing controller actions.
type FSConfig struct {
	fs  billy.Filesystem
	dir *string
}

// NewFSConfig constructs an FSConfig which wraps an implementation of fs.FS (read-only).
func NewFSConfig(fs billy.Filesystem) FSConfig {
	return FSConfig{fs: fs}
}

// NewDirFSConfig constructs an FSConfig which idenitifes a target directory on disk
// to be leveraged when mounting the wazero FS (currently to support writes).
func NewDirFSConfig(dir string) FSConfig {
	return FSConfig{
		dir: &dir,
	}
}

// ToFS returns either the configured fs.FS implementation or it
// adapts the desired directory into an fs.FS using os.DirFS
// depending on how the config was configured
func (c *FSConfig) ToFS() billy.Filesystem {
	if c.dir != nil {
		return osfs.New(*c.dir)
	}

	return c.fs
}
