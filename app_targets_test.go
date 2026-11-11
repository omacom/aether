package main

import (
	"os"
	"path/filepath"
	"testing"

	"aether/internal/theme"
)

// Exercises the production template wiring: the real embedded templates must
// render for the btop and vscode targets (not fail) with injected commands.
func TestApplyThemeRealTemplatesRenderAppTargets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local/state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local/share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	w := theme.NewWriter(EmbeddedTemplates, "templates")
	w.Run = func(string, ...string) (string, error) { return "", nil }
	w.Have = func(string) bool { return true }
	res, err := w.ApplyTheme(theme.NewThemeState(), theme.Settings{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, st := range res.Targets {
		seen[st.ID] = st.State
		if st.ID == "btop" || st.ID == "vscode" || st.ID == "hyprlock" || st.ID == "rofi" {
			if st.State == "failed" {
				t.Errorf("%s failed: %s", st.ID, st.Error)
			}
		}
	}
	if seen["btop"] == "" || seen["vscode"] == "" {
		t.Fatalf("targets missing: %v", seen)
	}
	if _, err := os.Stat(filepath.Join(home, ".config/btop/themes/aether.theme")); err != nil {
		t.Error(err)
	}
}

func TestListTargetsReportsDetection(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config/nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := listTargets(home, func(bin string) bool { return bin == "kitty" })
	by := map[string]TargetInfo{}
	for _, ti := range got {
		by[ti.ID] = ti
	}
	if !by["kitty"].Detected || by["kitty"].Label == "" || by["kitty"].Path == "" {
		t.Fatalf("kitty = %+v", by["kitty"])
	}
	if by["btop"].Detected {
		t.Fatalf("btop detected without binary: %+v", by["btop"])
	}
	if !by["neovim"].Detected {
		t.Fatalf("neovim not detected from ~/.config/nvim: %+v", by["neovim"])
	}
	if len(got) < 10 {
		t.Fatalf("got %d targets", len(got))
	}
}

func TestImportYozakuraThemes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := NewApp()
	fixture := "internal/yozakura/testdata/curated.json"
	res := app.ImportYozakuraThemes(fixture)
	if res.Error != "" || len(res.Imported) != 6 || res.Default != "icho" {
		t.Fatalf("res = %+v", res)
	}
	if ok, _ := app.blueprints.FindByName("icho"); ok == nil {
		t.Fatal("icho blueprint not saved")
	}
	if again := app.ImportYozakuraThemes(fixture); again.Error == "" {
		t.Fatal("second import must refuse to overwrite")
	}
	if bad := app.ImportYozakuraThemes("/nonexistent.json"); bad.Error == "" {
		t.Fatal("missing file must report an error")
	}
}
