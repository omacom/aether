package theme

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"aether/internal/omarchy"
	"aether/internal/platform"
	"aether/internal/wallpaper"
)

// IsOmarchyInstalled reports whether the public Omarchy CLI is available.
func IsOmarchyInstalled() bool {
	return omarchy.IsInstalled()
}

// HandleLightModeMarker creates or removes the light.mode marker file in
// the theme directory. The presence of this file signals light mode to
// consumers.
func HandleLightModeMarker(themeDir string, lightMode bool) error {
	markerPath := filepath.Join(themeDir, "light.mode")

	if lightMode {
		// Create empty light.mode file (ignore if it already exists)
		f, err := os.OpenFile(markerPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		log.Println("Created light.mode marker file")
		return nil
	}

	// Remove light.mode file if it exists
	err := os.Remove(markerPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err == nil {
		log.Println("Removed light.mode marker file")
	}
	return err
}

// IsImageFile returns true for still-image formats that can be decoded directly.
func IsImageFile(path string) bool {
	return wallpaper.IsImageFile(path)
}

// ClearTheme removes Aether's standalone override and restores the last native
// Omarchy theme that was active before an Aether-managed theme.
func ClearTheme() error {
	if err := RetireLegacyGTKStylesheets(); err != nil {
		return fmt.Errorf("retire legacy GTK stylesheets: %w", err)
	}

	// Delete Aether override CSS file in theme dir (cross-platform)
	overrideCss := filepath.Join(platform.ThemeDir(), "aether.override.css")
	if err := platform.DeleteFile(overrideCss); err != nil {
		log.Printf("Warning: could not delete theme override CSS: %v", err)
	}

	if !IsOmarchyInstalled() {
		return nil
	}
	return omarchy.RevertTheme()
}
