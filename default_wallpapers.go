package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"aether/internal/platform"
)

const defaultWallpapersDir = "assets/default-wallpapers"

//go:embed assets/default-wallpapers/*.jpg
var defaultWallpapers embed.FS

// installDefaultWallpapers copies bundled Aether artwork into the configured
// wallpaper folder without overwriting user files. A configured folder that is
// missing is not created, so an unmounted drive does not get a stub directory.
func installDefaultWallpapers() error {
	folder := wallpaperFolderFromConfig(readConfigJSON("settings.json"))
	if folder != platform.WallpaperDir() {
		if info, err := os.Stat(folder); err != nil || !info.IsDir() {
			return fmt.Errorf("configured wallpaper folder %q is not a directory", folder)
		}
	}

	destination := filepath.Join(folder, "Aether")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create default wallpapers directory: %w", err)
	}

	entries, err := fs.ReadDir(defaultWallpapers, defaultWallpapersDir)
	if err != nil {
		return fmt.Errorf("list bundled wallpapers: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		destinationPath := filepath.Join(destination, entry.Name())
		if _, err := os.Stat(destinationPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect default wallpaper %q: %w", entry.Name(), err)
		}
		data, err := fs.ReadFile(defaultWallpapers, filepath.Join(defaultWallpapersDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read bundled wallpaper %q: %w", entry.Name(), err)
		}
		if err := os.WriteFile(destinationPath, data, 0o644); err != nil {
			return fmt.Errorf("write default wallpaper %q: %w", entry.Name(), err)
		}
	}
	return nil
}
