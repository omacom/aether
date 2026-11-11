package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aether/internal/blueprint"
	"aether/internal/theme"
)

func isolatedApp(t *testing.T) (*App, string, *[]string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	app := NewApp()
	var calls []string
	app.writer.Run = func(name string, args ...string) (string, error) {
		calls = append(calls, name)
		return "", nil
	}
	app.writer.Have = func(string) bool { return true }
	return app, home, &calls
}

func readYozakuraJSON(t *testing.T, home string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".local/state/yozakura/theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func saveLabelBlueprint(t *testing.T, app *App, name, display string) {
	t.Helper()
	bp := &blueprint.Blueprint{Name: name, DisplayName: display}
	bp.Palette.Colors = theme.DefaultPalette[:]
	if err := app.blueprints.Save(name, *bp); err != nil {
		t.Fatal(err)
	}
}

func TestApplyBlueprintLabelsThemeJSON(t *testing.T) {
	app, home, _ := isolatedApp(t)
	saveLabelBlueprint(t, app, "sumi", "Sumi Ink")
	if _, err := app.ApplyBlueprint("sumi"); err != nil {
		t.Fatal(err)
	}
	if m := readYozakuraJSON(t, home); m["id"] != "sumi" || m["name"] != "Sumi Ink" {
		t.Fatalf("id=%v name=%v", m["id"], m["name"])
	}
}

func TestApplyThemeCarriesBlueprintLabel(t *testing.T) {
	app, home, _ := isolatedApp(t)
	req := ApplyThemeRequest{Palette: theme.DefaultPalette[:], Settings: theme.DefaultApplySettings()}
	if _, err := app.ApplyTheme(req); err != nil {
		t.Fatal(err)
	}
	if m := readYozakuraJSON(t, home); m["id"] != "auto" || m["name"] != "Auto" {
		t.Fatalf("extraction id=%v name=%v", m["id"], m["name"])
	}
	req.ThemeID, req.ThemeName = "sumi", "Sumi Ink"
	if _, err := app.ApplyTheme(req); err != nil {
		t.Fatal(err)
	}
	if m := readYozakuraJSON(t, home); m["id"] != "sumi" || m["name"] != "Sumi Ink" {
		t.Fatalf("blueprint id=%v name=%v", m["id"], m["name"])
	}
}

func TestApplyBlueprintSkipsOptedOutTarget(t *testing.T) {
	app, home, calls := isolatedApp(t)
	cfg := filepath.Join(home, ".config/aether")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "settings.json"), []byte(`{"includedApps":{"kitty":false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	saveLabelBlueprint(t, app, "sumi", "")
	if _, err := app.ApplyBlueprint("sumi"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config/kitty/aether-theme.conf")); err == nil {
		t.Fatal("opted-out kitty target was written")
	}
	for _, c := range *calls {
		if c == "pkill" {
			t.Fatalf("opted-out target was reloaded: %v", *calls)
		}
	}
}
