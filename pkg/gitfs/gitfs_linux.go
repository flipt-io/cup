//go:build linux

package gitfs

import (
	"syscall"
)

func (i FileInfo) Sys() any {
	return &syscall.Stat_t{
		Mode: uint32(i.Mode()),
		Size: i.size,
		Ino:  1,
	}
}
