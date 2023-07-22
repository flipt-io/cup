package controller

import (
	"io/fs"
	"os"
)

// FSConfig encapsulates the configuration required to establish the root
// directory of the wazero runtime when performing controller actions.
type FSConfig struct {
	fs  fs.FS
	dir *string
}

// NewFSConfig constructs an FSConfig which wraps an implementation of fs.FS (read-only).
func NewFSConfig(fs fs.FS) FSConfig {
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
func (c *FSConfig) ToFS() fs.FS {
	if c.dir != nil {
		return os.DirFS(*c.dir)
	}

	return c.fs
}
