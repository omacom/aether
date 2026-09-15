package wallpaper

import (
	"path/filepath"
	"strings"
)

// IsImageFile reports whether the filename has a supported still-image extension.
func IsImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}
