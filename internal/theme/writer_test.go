package theme

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/platform"
)

//go:embed testdata/v4
var testTemplates embed.FS

// TestPrepareThemeDirCopiesWallpaperVariants covers the blur pair: the
// applied variant (WallpaperPath) and the unblurred source
// (OriginalWallpaperPath) must both land in backgrounds/ when they differ,
// and the returned destination must be the applied variant.
func TestPrepareThemeDirCopiesWallpaperVariants(t *testing.T) {
	srcDir := t.TempDir()
	original := filepath.Join(srcDir, "photo.png")
	blurred := filepath.Join(srcDir, "photo-blurred-a1b2c3d4.jpg")
	if err := os.WriteFile(original, []byte("original-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blurred, []byte("blurred-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	targetDir := t.TempDir()
	dest, err := prepareThemeDir(targetDir, &ThemeState{
		WallpaperPath:         blurred,
		OriginalWallpaperPath: original,
	})
	if err != nil {
		t.Fatal(err)
	}

	bgDir := filepath.Join(targetDir, "backgrounds")
	wantDest := filepath.Join(bgDir, "photo-blurred-a1b2c3d4.jpg")
	if dest != wantDest {
		t.Errorf("dest = %q, want %q", dest, wantDest)
	}

	got, err := os.ReadFile(wantDest)
	if err != nil {
		t.Fatalf("blurred variant missing: %v", err)
	}
	if string(got) != "blurred-bytes" {
		t.Error("blurred variant content mismatch")
	}

	got, err = os.ReadFile(filepath.Join(bgDir, "photo.png"))
	if err != nil {
		t.Fatalf("original wallpaper missing: %v", err)
	}
	if string(got) != "original-bytes" {
		t.Error("original wallpaper content mismatch")
	}

	// Identical paths (blur off) must not duplicate the file.
	onlyDir := t.TempDir()
	dest, err = prepareThemeDir(onlyDir, &ThemeState{
		WallpaperPath:         original,
		OriginalWallpaperPath: original,
	})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(onlyDir, "backgrounds"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("backgrounds has %d entries with identical paths, want 1", len(entries))
	}
	if dest != filepath.Join(onlyDir, "backgrounds", "photo.png") {
		t.Errorf("dest = %q, want the original copy", dest)
	}
}

func TestPrepareThemeDirRemovesLegacyGTKStylesheet(t *testing.T) {
	targetDir := t.TempDir()
	legacyFile := filepath.Join(targetDir, "gtk.css")
	if err := os.WriteFile(legacyFile, []byte(legacyGTKMarker), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := prepareThemeDir(targetDir, &ThemeState{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Errorf("legacy GTK stylesheet still exists: %v", err)
	}
}

func TestPrepareThemeDirPreservesUnownedGTKStylesheet(t *testing.T) {
	targetDir := t.TempDir()
	userFile := filepath.Join(targetDir, "gtk.css")
	if err := os.WriteFile(userFile, []byte("user styles"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := prepareThemeDir(targetDir, &ThemeState{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(userFile); err != nil {
		t.Errorf("unowned GTK stylesheet was removed: %v", err)
	}
}

func TestGetAppNameFromFileNameMapsZedTemplate(t *testing.T) {
	if got, want := getAppNameFromFileName("aether.zed.json"), "zed"; got != want {
		t.Errorf("getAppNameFromFileName() = %q, want %q", got, want)
	}
}

func TestProcessTemplatesRemovesDisabledTarget(t *testing.T) {
	themeDir := t.TempDir()
	stale := filepath.Join(themeDir, "kitty.conf")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	writer := NewWriter(testTemplates, "testdata/v4")
	writer.processTemplates(nil, themeDir, Settings{IncludedApps: map[string]bool{}}, nil, nil)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("disabled kitty target was not removed: %v", err)
	}
}

func TestGenerateOnlyRejectsTemplateInjection(t *testing.T) {
	writer := NewWriter(testTemplates, "testdata/v4")
	state := NewThemeState()
	state.SetColor(1, `#ff0000"; os.execute("touch /tmp/pwned"); --`)

	if err := writer.GenerateOnly(state, Settings{}, t.TempDir()); err == nil {
		t.Fatal("GenerateOnly() accepted a non-color template payload")
	}
}

func setupWriterTestEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for key, dir := range map[string]string{
		"XDG_CONFIG_HOME": ".config",
		"XDG_DATA_HOME":   ".local/share",
		"XDG_STATE_HOME":  ".local/state",
		"XDG_CACHE_HOME":  ".cache",
		"XDG_RUNTIME_DIR": "run",
		"PATH":            "bin",
	} {
		path := filepath.Join(home, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv(key, path)
	}
	t.Setenv("AETHER_EXTRA_THEME_DIRS", "")
	return home
}

func writeWriterTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertWriterTestFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != want {
		t.Fatalf("%s = %q, err = %v; want %q", path, data, err, want)
	}
}

func TestPrepareThemeDirPreservesMediaOnFailure(t *testing.T) {
	for _, tc := range []struct {
		name, wallpaper string
		additional      []string
		wantError       string
	}{
		{"missing wallpaper", "incoming/missing.png", nil, "copy wallpaper"},
		{"missing later image", "incoming/wall.png", []string{"theme/backgrounds/extra.png", "incoming/missing.png"}, "copy additional image 2"},
		{"unreadable image", "incoming/wall.png", []string{"incoming/directory.png"}, "copy additional image 1"},
		{"wallpaper collision", "incoming/wall.png", []string{"other/wall.png"}, "basename collision"},
		{"additional collision", "", []string{"incoming/wall.png", "other/wall.png"}, "basename collision"},
		{"archive collision", "incoming/archive", nil, "preserve background directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := setupWriterTestEnv(t)
			target := filepath.Join(root, "theme")
			writeWriterTestFile(t, filepath.Join(target, "backgrounds", "wall.png"), "old wallpaper")
			writeWriterTestFile(t, filepath.Join(target, "backgrounds", "extra.png"), "old extra")
			writeWriterTestFile(t, filepath.Join(target, "backgrounds", "archive", "keep.jpg"), "archived wallpaper")
			writeWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors")
			writeWriterTestFile(t, filepath.Join(root, "incoming", "wall.png"), "new wallpaper")
			writeWriterTestFile(t, filepath.Join(root, "incoming", "archive"), "colliding wallpaper")
			writeWriterTestFile(t, filepath.Join(root, "other", "wall.png"), "colliding wallpaper")
			writeWriterTestFile(t, filepath.Join(root, "incoming", "directory.png", "keep"), "not an image")
			state := NewThemeState()
			if tc.wallpaper != "" {
				state.WallpaperPath = filepath.Join(root, tc.wallpaper)
			}
			for _, path := range tc.additional {
				state.AdditionalImages = append(state.AdditionalImages, filepath.Join(root, path))
			}

			dest, err := prepareThemeDir(target, state)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) || dest != "" {
				t.Fatalf("prepareThemeDir() = %q, %v; want %q error", dest, err, tc.wantError)
			}
			assertWriterTestFile(t, filepath.Join(target, "backgrounds", "wall.png"), "old wallpaper")
			assertWriterTestFile(t, filepath.Join(target, "backgrounds", "extra.png"), "old extra")
			assertWriterTestFile(t, filepath.Join(target, "backgrounds", "archive", "keep.jpg"), "archived wallpaper")
			assertWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors")
			entries, err := os.ReadDir(target)
			if err != nil || len(entries) != 2 {
				t.Fatalf("staging files left after failure: %v, err = %v", entries, err)
			}
		})
	}
}

func TestPrepareThemeDirStagesRepeatedPathsOnce(t *testing.T) {
	root := setupWriterTestEnv(t)
	target := filepath.Join(root, "theme")
	wallpaper := filepath.Join(root, "incoming", "wall.png")
	extra := filepath.Join(root, "incoming", "extra.png")
	writeWriterTestFile(t, wallpaper, "wallpaper")
	writeWriterTestFile(t, extra, "extra")
	state := NewThemeState()
	state.WallpaperPath = wallpaper
	state.AdditionalImages = []string{extra, wallpaper, extra}

	dest, err := prepareThemeDir(target, state)
	if err != nil {
		t.Fatalf("prepareThemeDir() error = %v", err)
	}
	if want := filepath.Join(target, "backgrounds", "wall.png"); dest != want {
		t.Fatalf("prepareThemeDir() = %q, want %q", dest, want)
	}
	assertWriterTestFile(t, filepath.Join(target, "backgrounds", "wall.png"), "wallpaper")
	assertWriterTestFile(t, filepath.Join(target, "backgrounds", "extra.png"), "extra")
	entries, err := os.ReadDir(filepath.Join(target, "backgrounds"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("backgrounds = %v, err = %v; want 2 files", entries, err)
	}
}

func TestPrepareThemeDirSupportsExistingMediaSources(t *testing.T) {
	for _, mode := range []string{"self source", "symlink source", "additional only", "color only"} {
		t.Run(mode, func(t *testing.T) {
			root := setupWriterTestEnv(t)
			target := filepath.Join(root, "theme")
			wallpaper := filepath.Join(target, "backgrounds", "wall.png")
			extra := filepath.Join(target, "backgrounds", "extra.png")
			unused := filepath.Join(target, "backgrounds", "unused.png")
			writeWriterTestFile(t, wallpaper, "wallpaper")
			writeWriterTestFile(t, extra, "extra")
			writeWriterTestFile(t, unused, "unused")
			archive := filepath.Join(target, "backgrounds", "archive")
			writeWriterTestFile(t, filepath.Join(archive, "keep.jpg"), "archived wallpaper")
			if err := os.Chmod(archive, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Join(archive, "keep.jpg"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("keep.jpg", filepath.Join(archive, "linked.jpg")); err != nil {
				t.Fatal(err)
			}
			state := NewThemeState()
			state.WallpaperPath = wallpaper
			state.AdditionalImages = []string{extra}
			if mode == "symlink source" {
				state.WallpaperPath = filepath.Join(root, "linked.png")
				if err := os.Symlink(wallpaper, state.WallpaperPath); err != nil {
					t.Fatal(err)
				}
			} else if mode == "additional only" || mode == "color only" {
				state.WallpaperPath = ""
				if mode == "color only" {
					state.AdditionalImages = nil
				}
			}

			dest, err := prepareThemeDir(target, state)
			if err != nil {
				t.Fatal(err)
			}
			assertWriterTestFile(t, extra, "extra")
			assertWriterTestFile(t, filepath.Join(archive, "keep.jpg"), "archived wallpaper")
			if link, err := os.Readlink(filepath.Join(archive, "linked.jpg")); err != nil || link != "keep.jpg" {
				t.Fatalf("archive symlink = %q, %v; want keep.jpg", link, err)
			}
			for path, mode := range map[string]os.FileMode{archive: 0o700, filepath.Join(archive, "keep.jpg"): 0o600} {
				if info, err := os.Stat(path); err != nil || info.Mode().Perm() != mode {
					t.Fatalf("archive permissions changed for %s: %v, %v", path, info, err)
				}
			}
			if state.WallpaperPath != "" {
				want := filepath.Join(target, "backgrounds", filepath.Base(state.WallpaperPath))
				if dest != want {
					t.Fatalf("wallpaper destination = %q, want %q", dest, want)
				}
				assertWriterTestFile(t, dest, "wallpaper")
				if info, err := os.Lstat(dest); err != nil || !info.Mode().IsRegular() {
					t.Fatalf("wallpaper was not copied as a regular file: %v, %v", info, err)
				}
			} else if dest != "" {
				t.Fatalf("unexpected wallpaper destination %q", dest)
			}
			if mode == "color only" {
				assertWriterTestFile(t, wallpaper, "wallpaper")
				assertWriterTestFile(t, unused, "unused")
			} else if _, err := os.Stat(unused); !os.IsNotExist(err) {
				t.Fatalf("unused background was not removed: %v", err)
			}
		})
	}
}

func TestApplyThemeReturnsEditorErrorsBeforeCustomApps(t *testing.T) {
	for _, editor := range []string{"zed"} {
		t.Run(editor, func(t *testing.T) {
			home := setupWriterTestEnv(t)
			target := platform.ThemeDir()
			writeWriterTestFile(t, filepath.Join(target, zedSourceFilename), "generated Zed theme")
			block := filepath.Join(home, ".config", "zed")
			wantError := "apply Zed theme"
			writeWriterTestFile(t, block, "not a directory")
			custom := filepath.Join(platform.CustomDir(), "test-app")
			writeWriterTestFile(t, filepath.Join(custom, "config.json"), `{"template":"theme.ini"}`)
			writeWriterTestFile(t, filepath.Join(custom, "theme.ini"), "{background}")

			writer := newTestWriter(t)
			result, err := writer.ApplyTheme(NewThemeState(), Settings{IncludedApps: map[string]bool{editor: true}})
			if err == nil || !strings.Contains(err.Error(), wantError) || result == nil || result.Success {
				t.Fatalf("ApplyTheme() = %+v, %v; want %q error", result, err, wantError)
			}
			if _, err := os.Stat(filepath.Join(target, "test-app-theme.ini")); !os.IsNotExist(err) {
				t.Fatalf("custom apps ran after editor failure: %v", err)
			}
		})
	}
}

func TestApplyThemeReturnsCustomAppError(t *testing.T) {
	setupWriterTestEnv(t)
	writeWriterTestFile(t, filepath.Join(platform.CustomDir(), "broken", "config.json"), "invalid json")
	writer := newTestWriter(t)
	result, err := writer.ApplyTheme(NewThemeState(), Settings{})
	if err == nil || !strings.Contains(err.Error(), "process custom apps") || !strings.Contains(err.Error(), "broken") || result == nil || result.Success {
		t.Fatalf("ApplyTheme() = %+v, %v; want custom app error", result, err)
	}
}

func TestGenerateOnlyReturnsVSCodeExportError(t *testing.T) {
	setupWriterTestEnv(t)
	target := platform.ThemeDir()
	writeWriterTestFile(t, filepath.Join(target, "vscode-extension"), "not a directory")
	writer := NewWriter(testTemplates, "testdata/v4")
	err := writer.GenerateOnly(NewThemeState(), Settings{IncludeVscode: true}, target)
	if err == nil || !strings.Contains(err.Error(), "export VSCode extension") {
		t.Fatalf("GenerateOnly() error = %v, want VSCode export error", err)
	}
}

func TestStandaloneWriterReturnsGenerationErrors(t *testing.T) {
	for _, apply := range []bool{false, true} {
		operation := "generate"
		if apply {
			operation = "apply"
		}
		for _, tc := range []struct {
			name, block, wantError string
			settings               Settings
			lightMode              bool
		}{
			{name: "missing templates", wantError: "list templates"},
			{name: "template write", block: "kitty.conf", wantError: "write processed template kitty.conf", settings: Settings{IncludedApps: map[string]bool{"kitty": true}}},
			{name: "stale template removal", block: "kitty.conf", wantError: "remove stale template kitty.conf", settings: Settings{IncludedApps: map[string]bool{}}},
			{name: "neovim preset", block: "neovim.lua", wantError: "write custom neovim.lua", settings: Settings{IncludeNeovim: true, SelectedNeovimConfig: "preset"}},
			{name: "light marker creation", block: "light.mode", wantError: "update light mode marker", lightMode: true},
			{name: "light marker removal", block: "light.mode", wantError: "update light mode marker"},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				home := setupWriterTestEnv(t)
				target := platform.ThemeDir()
				if tc.block != "" {
					writeWriterTestFile(t, filepath.Join(target, tc.block, "keep"), "blocked")
				}
				custom := filepath.Join(platform.CustomDir(), "test-app")
				writeWriterTestFile(t, filepath.Join(custom, "config.json"), `{"template":"theme.ini"}`)
				writeWriterTestFile(t, filepath.Join(custom, "theme.ini"), "{background}")
				writeWriterTestFile(t, filepath.Join(custom, "post-apply.sh"), "exit 0\n")
				hookLog := filepath.Join(home, "hook-ran")
				t.Setenv("AETHER_TEST_HOOK", hookLog)
				if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "bash"), []byte("#!/bin/sh\nprintf hook > \"$AETHER_TEST_HOOK\"\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				writer := newTestWriter(t)
				if tc.name == "missing templates" {
					writer.templatesDir = "missing"
				}
				state := NewThemeState()
				state.LightMode = tc.lightMode
				var err error
				if apply {
					var result *ApplyResult
					result, err = writer.ApplyTheme(state, tc.settings)
					if result == nil || result.Success || result.ThemePath != target {
						t.Fatalf("unexpected failure result: %+v", result)
					}
				} else {
					err = writer.GenerateOnly(state, tc.settings, target)
				}
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				for _, path := range []string{filepath.Join(target, "test-app-theme.ini"), hookLog} {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatalf("custom apps ran after failed generation: %s, %v", path, err)
					}
				}
			})
		}
	}
}

// newTestWriter returns a Writer whose commands never reach the real system:
// every Run is recorded (not executed) and no binary is considered installed.
func newTestWriter(t *testing.T) *Writer {
	t.Helper()
	w := NewWriter(testTemplates, "testdata/v4")
	w.Run = func(string, ...string) (string, error) { return "", nil }
	w.Have = func(string) bool { return false }
	return w
}

func TestApplyThemeReturnsTargets(t *testing.T) {
	home := setupWriterTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	var calls [][]string
	writeWriterTestFile(t, filepath.Join(home, ".config/hypr/hyprland.lua"), "-- user\n")
	w := newTestWriter(t)
	w.Have = func(string) bool { return true }
	w.Run = func(name string, args ...string) (string, error) {
		calls = append(calls, append([]string{name}, args...))
		return "", nil
	}
	result, err := w.ApplyTheme(NewThemeState(), Settings{})
	if err != nil || result == nil || !result.Success {
		t.Fatalf("ApplyTheme() = %+v, %v", result, err)
	}
	byID := map[string]string{}
	for _, st := range result.Targets {
		byID[st.ID] = st.State
	}
	if byID["yozakura"] != "written" || byID["hyprland"] != "reloaded" {
		t.Fatalf("targets = %+v", result.Targets)
	}
	if _, err := os.Stat(filepath.Join(home, ".local/state/yozakura/theme.json")); err != nil {
		t.Fatal(err)
	}
	if len(calls) == 0 {
		t.Fatal("expected reload commands through the injected runner")
	}
}

func TestApplyThemeHonoursDisabledTarget(t *testing.T) {
	home := setupWriterTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	w := newTestWriter(t)
	result, err := w.ApplyTheme(NewThemeState(), Settings{IncludedApps: map[string]bool{"yozakura": false}})
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	for _, st := range result.Targets {
		if st.ID == "yozakura" && st.State != "skipped" {
			t.Fatalf("yozakura = %+v", st)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".local/state/yozakura/theme.json")); err == nil {
		t.Fatal("disabled target wrote its file")
	}
}

func TestTargetEnabledDefaultsToTrue(t *testing.T) {
	if !(Settings{}).TargetEnabled("kitty") || !(Settings{IncludedApps: map[string]bool{"kitty": true}}).TargetEnabled("kitty") {
		t.Fatal("targets should default to enabled")
	}
	if (Settings{IncludedApps: map[string]bool{"kitty": false}}).TargetEnabled("kitty") {
		t.Fatal("explicit false must disable")
	}
}

func btopTargetTheme(t *testing.T, home string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".config/btop/themes/aether.theme"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func applyBtopWriter(t *testing.T, state *ThemeState) *Writer {
	t.Helper()
	w := newTestWriter(t)
	w.Have = func(bin string) bool { return bin == "btop" }
	if _, err := w.ApplyTheme(state, Settings{}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestApplyThemeTargetsHonourAppOverrides(t *testing.T) {
	home := setupWriterTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	state := NewThemeState()
	state.AppOverrides = map[string]map[string]string{"btop": {"background": "#123456"}}
	applyBtopWriter(t, state)
	if got := btopTargetTheme(t, home); !strings.Contains(got, "#123456") {
		t.Fatalf("override missing from btop target output: %q", got)
	}
}

func TestApplyThemeTargetsHonourCustomTemplate(t *testing.T) {
	home := setupWriterTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	writeWriterTestFile(t, filepath.Join(platform.CustomDir(), "btop.theme"), "custom {background}\n")
	applyBtopWriter(t, NewThemeState())
	got := btopTargetTheme(t, home)
	if !strings.HasPrefix(got, "custom #") {
		t.Fatalf("custom template not used: %q", got)
	}
}

func TestApplyThemeTargetsHonourSelectedNeovimConfig(t *testing.T) {
	home := setupWriterTestEnv(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	if err := os.MkdirAll(filepath.Join(home, ".config/nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	w := newTestWriter(t)
	w.Have = func(string) bool { return false }
	if _, err := w.ApplyTheme(NewThemeState(), Settings{IncludeNeovim: true, SelectedNeovimConfig: "-- picked"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(home, ".config/nvim/lua/aether/theme.lua"))
	if err != nil || string(b) != "-- picked" {
		t.Fatalf("theme.lua = %q, %v", b, err)
	}
}
