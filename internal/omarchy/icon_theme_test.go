package omarchy

import (
	"os"
	"path/filepath"
	"testing"

	"aether/internal/icontheme"
)

func TestThemeDiscoveryPreservesIconOverlaySelection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OMARCHY_PATH", t.TempDir())
	user, stock := t.TempDir(), t.TempDir()
	t.Setenv(extraThemeDirsEnv, user+string(os.PathListSeparator)+stock)
	for _, root := range []string{user, stock} {
		if err := os.MkdirAll(filepath.Join(root, "sample"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(user, "sample", "icons.theme")
	if err := os.WriteFile(path, []byte("User-Icons\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stock, "sample", "icons.theme"), []byte("Stock-Icons\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	themes, err := LoadAllThemes()
	if err != nil || len(themes) != 1 {
		t.Fatalf("themes = %+v, error = %v", themes, err)
	}
	if got := themes[0].IconTheme; got != (icontheme.Selection{Mode: icontheme.SelectionExplicit, ID: "User-Icons"}) {
		t.Fatalf("icon selection = %+v", got)
	}
	if err := os.WriteFile(path, []byte("../invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	themes, err = LoadAllThemes()
	if err != nil || len(themes) != 1 || themes[0].IconTheme != icontheme.Automatic() {
		t.Fatalf("invalid selection does not default to automatic: %+v, %v", themes, err)
	}
}
