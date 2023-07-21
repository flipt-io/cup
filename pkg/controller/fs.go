package controller

import "io/fs"

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
