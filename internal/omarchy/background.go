package omarchy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aether/internal/platform"
)

// UserBackgroundsDir returns Omarchy's per-theme user backgrounds directory
// (~/.config/omarchy/backgrounds/<themeName>).
func UserBackgroundsDir(themeName string) (string, error) {
	name := strings.TrimSpace(themeName)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return "", fmt.Errorf("invalid Omarchy theme name %q", themeName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "omarchy", "backgrounds", name), nil
}

// CurrentThemeBackgroundsDir returns the user backgrounds directory and name
// for the currently active Omarchy theme.
func CurrentThemeBackgroundsDir() (string, string, error) {
	if !IsInstalled() {
		return "", "", fmt.Errorf("adding to theme backgrounds requires Omarchy")
	}
	themeName := GetCurrentThemeName()
	if themeName == "" {
		return "", "", fmt.Errorf("no active Omarchy theme")
	}
	dir, err := UserBackgroundsDir(themeName)
	if err != nil {
		return "", "", err
	}
	return dir, themeName, nil
}

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
