//go:build unix

package gitfs

import (
	"syscall"
)

func (i FileInfo) Sys() any {
	return &syscall.Stat_t{
		Mode: uint16(i.Mode()),
		Size: i.size,
		Ino:  1,
	}
}
