//go:build windows
// +build windows

package docker

import (
	"io/fs"
)

// getUnixFileInfo is a no-op on Windows, returns default values
func getUnixFileInfo(info fs.FileInfo, defaultMode uint32) (owner, group string, mode uint32) {
	// On Windows, we can't get Unix-style owner/group/mode
	// Return empty strings and the default mode
	return "", "", defaultMode
}
