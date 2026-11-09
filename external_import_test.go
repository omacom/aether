package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/pending"
)

func TestStageImportRejectsStaleConfirmation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	app := NewApp()
	latest := &pending.Import{SourceURL: "aether://apply?wallpaper=https://example.com/new.png"}
	app.pending.curr = latest

	if _, err := app.stageImportIntoState("aether://apply?wallpaper=https://example.com/old.png"); err == nil {
		t.Fatal("stageImportIntoState() accepted a stale confirmation")
	}
	if app.pending.curr != latest {
		t.Fatal("stale confirmation replaced the latest pending import")
	}
}

func TestStageImportClearsYozakuraRoles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var content strings.Builder
	content.WriteString("hyprland_active_border = \"45deg #112233 #445566\"\n")
	for i := 0; i < 16; i++ {
		fmt.Fprintf(&content, "color%d = \"#%06x\"\n", i, i)
	}
	colorsPath := filepath.Join(t.TempDir(), "colors.toml")
	if err := os.WriteFile(colorsPath, []byte(content.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.state.YozakuraRoles = map[string]string{"accent": "#ffffff"}
	app.pending.curr = &pending.Import{SourceURL: "aether://test", ColorsToml: colorsPath}
	if _, err := app.stageImportIntoState("aether://test"); err != nil {
		t.Fatal(err)
	}
	if len(app.state.YozakuraRoles) != 0 {
		t.Fatalf("stale yozakura roles survived import: %v", app.state.YozakuraRoles)
	}
}
