//go:build windows
// +build windows

package path

import (
	"path/filepath"
	"strings"
)

var NeedToSanitize map[rune]bool

func init() {
	NeedToSanitize = map[rune]bool{
		'<': true, '>': true, ':': true, '"': true, '|': true, '?': true, '*': true, '\x00': true,
	}
}

// sanitizePath cleans a path string by removing or replacing invalid Windows file name characters.
func sanitizePath(path string, sanitize sanitizer, toSanitize map[rune]bool) string {
	// replace all slashes with backslashes
	path = filepath.FromSlash(path)

	// replace all invalid characters
	return strings.Map(func(r rune) rune {
		if _, isInvalid := toSanitize[r]; isInvalid {
			return sanitize(r, toSanitize)
		}
		return r
	}, path)
}

// sanitizer defined how to handle and replace invalid file name characters.
type sanitizer func(rune, map[rune]bool) rune

// SanitizePath replaces invalid characters in a Windows path with a placeholder.
func SanitizePath(path string) string {
	volumeName := filepath.VolumeName(path)
	// Only sanitize the part of the path after the volume name
	sanitized := sanitizePath(path[len(volumeName):], func(r rune, invalidChars map[rune]bool) rune {
		if _, ok := invalidChars[r]; ok {
			return '_'
		}
		return r
	}, NeedToSanitize)

	return volumeName + sanitized
}
