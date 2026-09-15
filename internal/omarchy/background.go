package omarchy

import (
	"fmt"
	"os"
	"path/filepath"

	"aether/internal/platform"
)

// SetBackground changes the native background without activating another theme.
func SetBackground(path string) error {
	if !IsInstalled() {
		return fmt.Errorf("a background-only change requires Omarchy")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve wallpaper: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("inspect wallpaper: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("wallpaper is not a regular file")
	}
	if _, err := platform.RunSync("omarchy", "theme", "bg", "set", abs); err != nil {
		return fmt.Errorf("set Omarchy background: %w", err)
	}
	return nil
}
