//go:build !windows
// +build !windows

package docker

import (
	"io/fs"
	"strconv"
	"syscall"
)

// getUnixFileInfo extracts Unix-specific file information (owner, group, mode)
func getUnixFileInfo(info fs.FileInfo, defaultMode uint32) (owner, group string, mode uint32) {
	owner = ""
	group = ""
	mode = defaultMode

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		owner = strconv.FormatUint(uint64(stat.Uid), 10)
		group = strconv.FormatUint(uint64(stat.Gid), 10)
		mode = uint32(stat.Mode & 0o777)
	}

	return owner, group, mode
}
