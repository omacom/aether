package main

import (
	"log"
	"runtime"

	"aether/internal/platform"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Wails v2 draws GtkFileChooserDialog directly on Linux, which bypasses
// xdg-desktop-portal and so the user's configured file picker. These helpers
// ask the portal first and fall back to the Wails dialog when it is missing.

func (a *App) openFileDialog(opts wailsrt.OpenDialogOptions) (string, error) {
	if path, ok := a.portalOpenFile(opts, false); ok {
		return path, nil
	}
	return wailsrt.OpenFileDialog(a.ctx, opts)
}

func (a *App) openDirectoryDialog(opts wailsrt.OpenDialogOptions) (string, error) {
	if path, ok := a.portalOpenFile(opts, true); ok {
		return path, nil
	}
	return wailsrt.OpenDirectoryDialog(a.ctx, opts)
}

// portalOpenFile reports ok=false when the portal could not be used, so the
// caller should show the Wails dialog instead. A cancelled pick is ok=true
// with an empty path.
func (a *App) portalOpenFile(opts wailsrt.OpenDialogOptions, directory bool) (string, bool) {
	if runtime.GOOS != "linux" {
		return "", false
	}
	filters := make([]platform.FileFilter, 0, len(opts.Filters))
	for _, f := range opts.Filters {
		filters = append(filters, platform.FileFilter{Name: f.DisplayName, Patterns: []string{f.Pattern}})
	}
	path, err := platform.PortalOpenFile(a.ctx, platform.FileChooserOptions{
		Title:         opts.Title,
		Directory:     directory,
		CurrentFolder: opts.DefaultDirectory,
		Filters:       filters,
	})
	if err != nil {
		log.Printf("file chooser portal unavailable, using GTK dialog: %v", err)
		return "", false
	}
	return path, true
}
