package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/omarchy"
)

func TestInstallOmarchyThemeCreatesAndActivatesNewTheme(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	omarchyDir := t.TempDir()
	activatedPath := filepath.Join(t.TempDir(), "activated")
	bgSetPath := filepath.Join(t.TempDir(), "bgset")

	t.Setenv("HOME", home)
	t.Setenv("OMARCHY_PATH", omarchyDir)
	t.Setenv("AETHER_TEST_ACTIVATED", activatedPath)
	t.Setenv("AETHER_TEST_BGSET", bgSetPath)
	t.Setenv("PATH", binDir)
	if err := os.MkdirAll(filepath.Join(omarchyDir, "shell"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(omarchyDir, "shell", "shell.qml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nif [ \"$1 $2 $3\" = \"theme bg set\" ]; then\n  printf '%s' \"$4\" > \"$AETHER_TEST_BGSET\"\nelif [ \"$1 $2\" = \"theme set\" ]; then\n  printf '%s' \"$3\" > \"$AETHER_TEST_ACTIVATED\"\nfi\n"
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// A wallpaper so the theme has a background to apply.
	srcDir := t.TempDir()
	wallpaper := filepath.Join(srcDir, "photo.png")
	if err := os.WriteFile(wallpaper, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	state := NewThemeState()
	state.WallpaperPath = wallpaper

	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	if err := writer.InstallOmarchyTheme(state, Settings{}, "web-theme"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(activatedPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "web-theme" {
		t.Errorf("activated theme = %q; want web-theme", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "omarchy", "themes", "web-theme")); err != nil {
		t.Fatalf("installed theme missing: %v", err)
	}

	// The theme's own wallpaper copy must have been applied explicitly —
	// omarchy-theme-set alone cycles backgrounds and may pick a stock image.
	applied, err := os.ReadFile(bgSetPath)
	if err != nil {
		t.Fatalf("wallpaper was not applied: %v", err)
	}
	want := filepath.Join(omarchy.UserThemesDir(), "web-theme", "backgrounds", "photo.png")
	if string(applied) != want {
		t.Errorf("applied wallpaper = %q; want %q", applied, want)
	}

	err = writer.InstallOmarchyTheme(state, Settings{}, "web-theme")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second install error = %v; want already-exists error", err)
	}
}

func TestValidOmarchyThemeName(t *testing.T) {
	for _, name := range []string{"theme", "Theme-2", "theme_name.v3"} {
		if !ValidOmarchyThemeName(name) {
			t.Errorf("ValidOmarchyThemeName(%q) = false; want true", name)
		}
	}
	for _, name := range []string{"", "../theme", "theme/name", "theme name", strings.Repeat("a", 65)} {
		if ValidOmarchyThemeName(name) {
			t.Errorf("ValidOmarchyThemeName(%q) = true; want false", name)
		}
	}
}

func TestInstallOmarchyThemeRemovesBundleAfterActivationFailure(t *testing.T) {
	home := setupWriterTestEnv(t)
	logPath := filepath.Join(home, "commands")
	t.Setenv("AETHER_COMMAND_LOG", logPath)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$AETHER_COMMAND_LOG"
if [ "$1 $2" = "theme set" ]; then
    printf 'activation failed\n' >&2
    exit 7
fi
`
	if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	oldBackground := filepath.Join(home, "old.png")
	writeWriterTestFile(t, oldBackground, "old wallpaper")
	writeWriterTestFile(t, filepath.Join(omarchy.CurrentStateDir(), "theme.name"), "foreign\n")
	if err := os.Symlink(oldBackground, filepath.Join(omarchy.CurrentStateDir(), "background")); err != nil {
		t.Fatal(err)
	}
	state := NewThemeState()
	state.WallpaperPath = filepath.Join(home, "incoming.png")
	writeWriterTestFile(t, state.WallpaperPath, "new wallpaper")
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	err := writer.InstallOmarchyTheme(state, Settings{}, "web-theme")
	if err == nil || !strings.Contains(err.Error(), "activation failed") {
		t.Fatalf("InstallOmarchyTheme() error = %v, want activation failure", err)
	}
	target := filepath.Join(omarchy.UserThemesDir(), "web-theme")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("failed install left a theme that blocks retry: %v", err)
	}
	assertWriterTestFile(t, logPath, "theme bg set "+filepath.Join(target, "backgrounds", "incoming.png")+"\ntheme set web-theme\ntheme bg set "+oldBackground+"\n")
	assertWriterTestFile(t, oldBackground, "old wallpaper")
	entries, err := os.ReadDir(omarchy.UserThemesDir())
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed install left transaction files: %v, %v", entries, err)
	}
}
