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
