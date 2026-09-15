package theme

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/omarchy"
	"aether/internal/platform"
)

//go:embed testdata/v4
var omarchyV4TestTemplates embed.FS

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

func TestProcessOmarchyV4TemplatesKeepsIconsTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	themeDir := t.TempDir()
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	writer.processOmarchyV4Templates(themeDir, map[string]string{
		"background": "#1e1e2e",
		"magenta":    "#ff0000",
		"mode":       "dark",
	}, Settings{IncludedApps: map[string]bool{"icons": true}}, nil, nil)

	colors, err := os.ReadFile(filepath.Join(themeDir, "colors.toml"))
	if err != nil {
		t.Fatalf("read colors.toml: %v", err)
	}
	if got, want := string(colors), "background = \"#1e1e2e\"\nmode = \"dark\"\n"; got != want {
		t.Errorf("colors.toml = %q, want %q", got, want)
	}

	icons, err := os.ReadFile(filepath.Join(themeDir, "icons.theme"))
	if err != nil {
		t.Fatalf("read icons.theme: %v", err)
	}
	if got, want := string(icons), "Yaru-red\n"; got != want {
		t.Errorf("icons.theme = %q, want %q", got, want)
	}
}

func TestProcessOmarchyV4TemplatesDefaultsToColorsOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	for name, settings := range map[string]Settings{
		"nil included apps":   {},
		"empty included apps": {IncludedApps: map[string]bool{}},
	} {
		t.Run(name, func(t *testing.T) {
			themeDir := t.TempDir()
			writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
			writer.processOmarchyV4Templates(themeDir, map[string]string{
				"background": "#1e1e2e",
				"foreground": "#cdd6f4",
				"magenta":    "#cba6f7",
				"mode":       "dark",
			}, settings, nil, nil)

			if _, err := os.Stat(filepath.Join(themeDir, "colors.toml")); err != nil {
				t.Errorf("colors.toml was not generated: %v", err)
			}
			if _, err := os.Stat(filepath.Join(themeDir, "icons.theme")); err != nil {
				t.Errorf("icons.theme was not generated: %v", err)
			}
			for _, name := range []string{"kitty.conf", "vscode-theme.json"} {
				if _, err := os.Stat(filepath.Join(themeDir, name)); !os.IsNotExist(err) {
					t.Errorf("%s was generated without a target or color override: %v", name, err)
				}
			}
		})
	}
}

func TestProcessOmarchyV4TemplatesIncludesColorOverrideApp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	themeDir := t.TempDir()
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	writer.processOmarchyV4Templates(themeDir, map[string]string{
		"background": "#1e1e2e",
		"foreground": "#cdd6f4",
		"magenta":    "#cba6f7",
		"mode":       "dark",
	}, Settings{IncludedApps: map[string]bool{}}, map[string]map[string]string{
		"kitty": {"foreground": "#fab387"},
	}, nil)

	kitty, err := os.ReadFile(filepath.Join(themeDir, "kitty.conf"))
	if err != nil {
		t.Fatalf("read kitty.conf: %v", err)
	}
	if got, want := string(kitty), "foreground #fab387\nbackground #1e1e2e\n"; got != want {
		t.Errorf("kitty.conf = %q, want %q", got, want)
	}
}

func TestProcessOmarchyV4TemplatesAppliesOverrideInsteadOfNeovimPreset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	themeDir := t.TempDir()
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	writer.processOmarchyV4Templates(themeDir, map[string]string{
		"background": "#1e1e2e",
		"foreground": "#cdd6f4",
		"magenta":    "#cba6f7",
		"mode":       "dark",
	}, Settings{
		IncludedApps:         map[string]bool{"neovim": true},
		SelectedNeovimConfig: "preset config",
	}, map[string]map[string]string{
		"neovim": {"foreground": "#fab387"},
	}, nil)

	neovim, err := os.ReadFile(filepath.Join(themeDir, "neovim.lua"))
	if err != nil {
		t.Fatalf("read neovim.lua: %v", err)
	}
	if got, want := string(neovim), "foreground = '#fab387'\n"; got != want {
		t.Errorf("neovim.lua = %q, want %q", got, want)
	}
}

func TestProcessOmarchyV4TemplatesWritesVSCodeThemeOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	themeDir := t.TempDir()
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	writer.processOmarchyV4Templates(themeDir, map[string]string{
		"background": "#1e1e2e",
		"foreground": "#cdd6f4",
		"magenta":    "#cba6f7",
		"mode":       "dark",
	}, Settings{}, map[string]map[string]string{
		"vscode": {"background": "#1e1e2e"},
	}, nil)

	if _, err := os.Stat(filepath.Join(themeDir, "vscode.json")); !os.IsNotExist(err) {
		t.Errorf("vscode.json descriptor was generated: %v", err)
	}
	theme, err := os.ReadFile(filepath.Join(themeDir, "vscode-theme.json"))
	if err != nil {
		t.Fatalf("read vscode-theme.json: %v", err)
	}
	if got, want := string(theme), "{\"background\":\"#1e1e2e\"}\n"; got != want {
		t.Errorf("vscode-theme.json = %q, want %q", got, want)
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

	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	writer.processTemplates(nil, themeDir, Settings{IncludedApps: map[string]bool{}}, nil, nil)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("disabled kitty target was not removed: %v", err)
	}
}

func TestGenerateOmarchyV4OnlyRemovesLegacyFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	themeDir := t.TempDir()
	legacyFile := filepath.Join(themeDir, "waybar.css")
	if err := os.WriteFile(legacyFile, []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := omarchy.MarkManagedTheme(themeDir); err != nil {
		t.Fatal(err)
	}

	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	state := NewThemeState()
	state.ColorRoles.Background = "#1e1e2e"
	state.ColorRoles.Magenta = "#ff0000"
	settings := Settings{IncludedApps: map[string]bool{"icons": true}}
	if err := writer.GenerateOmarchyV4Only(state, settings, themeDir); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"colors.toml", "icons.theme"} {
		if _, err := os.Stat(filepath.Join(themeDir, name)); err != nil {
			t.Errorf("%s was not generated: %v", name, err)
		}
	}
	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Errorf("legacy file still exists: %v", err)
	}
}

func TestGenerateOmarchyV4OnlyRefusesUnmanagedTheme(t *testing.T) {
	themeDir := t.TempDir()
	userFile := filepath.Join(themeDir, "notes.txt")
	if err := os.WriteFile(userFile, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	err := writer.GenerateOmarchyV4Only(NewThemeState(), Settings{}, themeDir)
	if err == nil || !strings.Contains(err.Error(), "unmanaged") {
		t.Fatalf("GenerateOmarchyV4Only() error = %v, want unmanaged-theme refusal", err)
	}
	if data, err := os.ReadFile(userFile); err != nil || string(data) != "keep me" {
		t.Fatalf("foreign theme content changed: data=%q err=%v", data, err)
	}
}

func TestReplaceThemeDirRestoresTargetChangedDuringGeneration(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "aether")
	staging := filepath.Join(parent, ".aether-staging")
	if err := os.Mkdir(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "colors.toml"), []byte("generated"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(target, "notes.txt")
	if err := os.WriteFile(foreign, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := replaceThemeDir(staging, target, nil)
	if err == nil || !strings.Contains(err.Error(), "changed during generation") {
		t.Fatalf("replaceThemeDir() error = %v, want ownership race refusal", err)
	}
	data, readErr := os.ReadFile(foreign)
	if readErr != nil || string(data) != "keep me" {
		t.Fatalf("foreign target was not restored: data=%q err=%v", data, readErr)
	}
	if _, err := os.Stat(filepath.Join(staging, "colors.toml")); err != nil {
		t.Fatalf("staging content was lost after refusal: %v", err)
	}
}

func TestGenerateOmarchyV4OnlyPreservesThemeWhenWallpaperCopyFails(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	themeDir := t.TempDir()
	background := filepath.Join(themeDir, "backgrounds", "current.jpg")
	if err := os.MkdirAll(filepath.Dir(background), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(background, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := omarchy.MarkManagedTheme(themeDir); err != nil {
		t.Fatal(err)
	}

	state := NewThemeState()
	state.WallpaperPath = filepath.Join(t.TempDir(), "missing.jpg")
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	if err := writer.GenerateOmarchyV4Only(state, Settings{}, themeDir); err == nil {
		t.Fatal("GenerateOmarchyV4Only() succeeded with a missing wallpaper")
	}
	data, err := os.ReadFile(background)
	if err != nil || string(data) != "original" {
		t.Fatalf("existing background changed: data=%q err=%v", data, err)
	}
}

func TestGenerateOmarchyV4OnlyPreservesThemeWhenTemplatesFail(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	themeDir := t.TempDir()
	colorsPath := filepath.Join(themeDir, "colors.toml")
	if err := os.WriteFile(colorsPath, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := omarchy.MarkManagedTheme(themeDir); err != nil {
		t.Fatal(err)
	}

	writer := NewWriter(omarchyV4TestTemplates, "missing")
	if err := writer.GenerateOmarchyV4Only(NewThemeState(), Settings{}, themeDir); err == nil {
		t.Fatal("GenerateOmarchyV4Only() succeeded without templates")
	}
	data, err := os.ReadFile(colorsPath)
	if err != nil || string(data) != "original" {
		t.Fatalf("existing theme changed: data=%q err=%v", data, err)
	}
}

func TestGenerateOnlyRejectsTemplateInjection(t *testing.T) {
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
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
		"OMARCHY_PATH":    "omarchy",
		"PATH":            "bin",
	} {
		path := filepath.Join(home, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv(key, path)
	}
	t.Setenv("OMARCHY_THEME_SKIP_BACKGROUND", "")
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
				writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
				if tc.name == "missing templates" {
					writer.templatesDir = "missing"
				}
				state := NewThemeState()
				state.LightMode = tc.lightMode
				var err error
				if apply {
					var result *ApplyResult
					result, err = writer.ApplyTheme(state, tc.settings)
					if result == nil || result.Success || result.IsOmarchy || result.ThemePath != target {
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

func TestApplyThemeReturnsEditorErrorsBeforeCustomApps(t *testing.T) {
	for _, editor := range []string{"zed", "vscode"} {
		t.Run(editor, func(t *testing.T) {
			home := setupWriterTestEnv(t)
			target := platform.ThemeDir()
			writeWriterTestFile(t, filepath.Join(target, zedSourceFilename), "generated Zed theme")
			block := filepath.Join(home, ".config", "zed")
			wantError := "apply Zed theme"
			if editor == "vscode" {
				block = filepath.Join(home, ".vscode")
				wantError = "apply VSCode theme"
			}
			writeWriterTestFile(t, block, "not a directory")
			custom := filepath.Join(platform.CustomDir(), "test-app")
			writeWriterTestFile(t, filepath.Join(custom, "config.json"), `{"template":"theme.ini"}`)
			writeWriterTestFile(t, filepath.Join(custom, "theme.ini"), "{background}")

			writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
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
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	result, err := writer.ApplyTheme(NewThemeState(), Settings{})
	if err == nil || !strings.Contains(err.Error(), "process custom apps") || !strings.Contains(err.Error(), "broken") || result == nil || result.Success {
		t.Fatalf("ApplyTheme() = %+v, %v; want custom app error", result, err)
	}
}

func TestGenerateOnlyReturnsVSCodeExportError(t *testing.T) {
	setupWriterTestEnv(t)
	target := platform.ThemeDir()
	writeWriterTestFile(t, filepath.Join(target, "vscode-extension"), "not a directory")
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	err := writer.GenerateOnly(NewThemeState(), Settings{IncludeVscode: true}, target)
	if err == nil || !strings.Contains(err.Error(), "export VSCode extension") {
		t.Fatalf("GenerateOnly() error = %v, want VSCode export error", err)
	}
}

func TestGenerateOmarchyV4OnlyCopiesSelfSourceMedia(t *testing.T) {
	setupWriterTestEnv(t)
	target := platform.OmarchyThemeDir()
	state := NewThemeState()
	state.WallpaperPath = filepath.Join(target, "backgrounds", "wall.png")
	state.AdditionalImages = []string{filepath.Join(target, "backgrounds", "extra.png")}
	writeWriterTestFile(t, state.WallpaperPath, "wallpaper")
	writeWriterTestFile(t, state.AdditionalImages[0], "extra")
	archive := filepath.Join(target, "backgrounds", "archive", "keep.jpg")
	writeWriterTestFile(t, archive, "archived wallpaper")
	if err := omarchy.MarkManagedTheme(target); err != nil {
		t.Fatal(err)
	}
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	if err := writer.GenerateOmarchyV4Only(state, Settings{}, target); err != nil {
		t.Fatal(err)
	}
	assertWriterTestFile(t, state.WallpaperPath, "wallpaper")
	assertWriterTestFile(t, state.AdditionalImages[0], "extra")
	assertWriterTestFile(t, filepath.Join(target, "preview.png"), "wallpaper")
	assertWriterTestFile(t, archive, "archived wallpaper")
	if err := writer.GenerateOmarchyV4Only(NewThemeState(), Settings{}, target); err != nil {
		t.Fatal(err)
	}
	assertWriterTestFile(t, archive, "archived wallpaper")
	assertWriterTestFile(t, state.WallpaperPath, "wallpaper")
}

func TestApplyOmarchyThemeKeepsBundleTransactionThroughActivation(t *testing.T) {
	for _, basename := range []string{"wall.png", "different.png"} {
		for _, failure := range []string{"success", "background", "theme", "color only"} {
			t.Run(basename+"/"+failure, func(t *testing.T) {
				home := setupWriterTestEnv(t)
				target := platform.OmarchyThemeDir()
				oldBackground := filepath.Join(target, "backgrounds", "wall.png")
				writeWriterTestFile(t, oldBackground, "old wallpaper\n")
				writeWriterTestFile(t, filepath.Join(target, "backgrounds", "extra.png"), "old extra\n")
				writeWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors\n")
				writeWriterTestFile(t, filepath.Join(target, "preview.png"), "old preview\n")
				if err := omarchy.MarkManagedTheme(target); err != nil {
					t.Fatal(err)
				}
				writeWriterTestFile(t, filepath.Join(omarchy.CurrentStateDir(), "theme.name"), "aether\n")
				backgroundLink := filepath.Join(omarchy.CurrentStateDir(), "background")
				if err := os.Symlink(oldBackground, backgroundLink); err != nil {
					t.Fatal(err)
				}
				commandLog := filepath.Join(home, "commands")
				mediaLog := filepath.Join(home, "selected-media")
				t.Setenv("AETHER_COMMAND_LOG", commandLog)
				t.Setenv("AETHER_MEDIA_LOG", mediaLog)
				t.Setenv("AETHER_BACKGROUND_LINK", backgroundLink)
				t.Setenv("AETHER_TEST_FAILURE", failure)
				script := `#!/bin/sh
printf '%s|%s\n' "$OMARCHY_THEME_SKIP_BACKGROUND" "$*" >> "$AETHER_COMMAND_LOG"
if [ "$1 $2 $3" = "theme bg set" ]; then
    if [ ! -f "$4" ]; then
        printf 'missing background: %s\n' "$4" >&2
        exit 8
    fi
    IFS= read -r content < "$4"
    printf '%s\n' "$content" >> "$AETHER_MEDIA_LOG"
    /bin/ln -sfn "$4" "$AETHER_BACKGROUND_LINK" || exit 9
    if [ "$AETHER_TEST_FAILURE" = "background" ] && [ "$content" = "new wallpaper" ]; then
        printf 'background failed\n' >&2
        exit 7
    fi
fi
if [ "$1 $2" = "theme set" ] && [ "$AETHER_TEST_FAILURE" != "success" ]; then
    printf 'theme failed\n' >&2
    exit 7
fi
`
				if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "omarchy"), []byte(script), 0o755); err != nil {
					t.Fatal(err)
				}
				state := NewThemeState()
				state.WallpaperPath = filepath.Join(home, "incoming", basename)
				writeWriterTestFile(t, state.WallpaperPath, "new wallpaper\n")
				wallpaperDest := filepath.Join(target, "backgrounds", basename)
				if failure == "color only" {
					state.WallpaperPath = ""
				}
				writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
				result, err := writer.ApplyTheme(state, DefaultApplySettings())
				if result == nil || !result.IsOmarchy || result.ThemePath != target || result.Success != (failure == "success") {
					t.Fatalf("unexpected result: %+v, err = %v", result, err)
				}
				wantCommands := "|theme bg set " + wallpaperDest + "\n1|theme set aether\n"
				wantMedia := "new wallpaper\n"
				wantBackground := wallpaperDest
				if failure == "success" {
					if err != nil {
						t.Fatal(err)
					}
					assertWriterTestFile(t, wallpaperDest, "new wallpaper\n")
					assertWriterTestFile(t, filepath.Join(target, "preview.png"), "new wallpaper\n")
					if _, err := os.Stat(filepath.Join(target, "backgrounds", "extra.png")); !os.IsNotExist(err) {
						t.Fatalf("old media remained after successful activation: %v", err)
					}
				} else {
					wantError := "theme failed"
					if failure == "background" {
						wantError = "background failed"
						wantCommands = "|theme bg set " + wallpaperDest + "\n"
					} else if failure == "color only" {
						wantCommands = "|theme set aether\n"
						wantMedia = ""
					}
					if err == nil || !strings.Contains(err.Error(), wantError) || strings.Contains(err.Error(), "restore previous background") {
						t.Fatalf("ApplyTheme() error = %v, want %q with successful rollback", err, wantError)
					}
					wantCommands += "|theme bg set " + oldBackground + "\n"
					wantMedia += "old wallpaper\n"
					wantBackground = oldBackground
					assertWriterTestFile(t, oldBackground, "old wallpaper\n")
					assertWriterTestFile(t, filepath.Join(target, "backgrounds", "extra.png"), "old extra\n")
					assertWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors\n")
					assertWriterTestFile(t, filepath.Join(target, "preview.png"), "old preview\n")
				}
				assertWriterTestFile(t, commandLog, wantCommands)
				assertWriterTestFile(t, mediaLog, wantMedia)
				if selected, err := filepath.EvalSymlinks(backgroundLink); err != nil || selected != wantBackground {
					t.Fatalf("current background = %q, %v; want %q", selected, err, wantBackground)
				}
				snapshots, err := os.ReadDir(filepath.Join(platform.DataDir(), "background-recovery"))
				if err != nil || len(snapshots) != 0 {
					t.Fatalf("unneeded snapshots remain after restoring the bundle path: %v, %v", snapshots, err)
				}
				entries, err := os.ReadDir(filepath.Dir(target))
				if err != nil || len(entries) != 1 || entries[0].Name() != "aether" {
					t.Fatalf("unexpected transaction leftovers: %v, %v", entries, err)
				}
				for _, path := range []string{platform.ThemeDir(), filepath.Join(home, ".vscode"), filepath.Join(home, ".config", "zed")} {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatalf("native apply wrote standalone output %s: %v", path, err)
					}
				}
			})
		}
	}
}

func TestApplyOmarchyThemeRestoresCopiedRuntimeBackground(t *testing.T) {
	for _, basename := range []string{"wall.png", "different.png"} {
		for _, failure := range []string{"theme", "background", "color only"} {
			t.Run(basename+"/"+failure, func(t *testing.T) {
				home := setupWriterTestEnv(t)
				target := platform.OmarchyThemeDir()
				sourceBackground := filepath.Join(target, "backgrounds", "wall.png")
				writeWriterTestFile(t, sourceBackground, "bundle wallpaper\n")
				writeWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors\n")
				if err := omarchy.MarkManagedTheme(target); err != nil {
					t.Fatal(err)
				}
				runtimeTheme := filepath.Join(omarchy.CurrentStateDir(), "theme")
				runtimeBackground := filepath.Join(runtimeTheme, "backgrounds", "wall.png")
				writeWriterTestFile(t, runtimeBackground, "runtime wallpaper\n")
				writeWriterTestFile(t, filepath.Join(omarchy.CurrentStateDir(), "theme.name"), "aether\n")
				backgroundLink := filepath.Join(omarchy.CurrentStateDir(), "background")
				if err := os.Symlink(runtimeBackground, backgroundLink); err != nil {
					t.Fatal(err)
				}
				commandLog, mediaLog := filepath.Join(home, "commands"), filepath.Join(home, "selected-media")
				t.Setenv("AETHER_COMMAND_LOG", commandLog)
				t.Setenv("AETHER_MEDIA_LOG", mediaLog)
				t.Setenv("AETHER_RUNTIME_THEME", runtimeTheme)
				t.Setenv("AETHER_RUNTIME_FILENAME", basename)
				t.Setenv("AETHER_BACKGROUND_LINK", backgroundLink)
				t.Setenv("AETHER_TEST_FAILURE", failure)
				script := `#!/bin/sh
replace_runtime_theme() {
    /bin/rm -rf "$AETHER_RUNTIME_THEME" || exit 9
    /bin/mkdir -p "$AETHER_RUNTIME_THEME/backgrounds" || exit 9
    printf 'replacement wallpaper\n' > "$AETHER_RUNTIME_THEME/backgrounds/$AETHER_RUNTIME_FILENAME"
}
printf '%s|%s\n' "$OMARCHY_THEME_SKIP_BACKGROUND" "$*" >> "$AETHER_COMMAND_LOG"
if [ "$1 $2" = "theme set" ]; then
    replace_runtime_theme
    printf 'activation failed\n' >&2
    exit 7
fi
if [ "$1 $2 $3" = "theme bg set" ]; then
    [ -f "$4" ] || exit 8
    IFS= read -r content < "$4"
    printf '%s\n' "$content" >> "$AETHER_MEDIA_LOG"
    /bin/ln -sfn "$4" "$AETHER_BACKGROUND_LINK" || exit 9
    if [ "$AETHER_TEST_FAILURE" = "background" ] && [ "$content" = "new wallpaper" ]; then
        replace_runtime_theme
        printf 'background failed\n' >&2
        exit 7
    fi
fi
`
				if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "omarchy"), []byte(script), 0o755); err != nil {
					t.Fatal(err)
				}
				state := NewThemeState()
				state.WallpaperPath = filepath.Join(home, "incoming", basename)
				writeWriterTestFile(t, state.WallpaperPath, "new wallpaper\n")
				wallpaperDest := filepath.Join(target, "backgrounds", basename)
				wantCommands := "|theme bg set " + wallpaperDest + "\n1|theme set aether\n"
				wantMedia := "new wallpaper\nruntime wallpaper\n"
				wantError := "activation failed"
				if failure == "color only" {
					state.WallpaperPath = ""
					wantCommands = "|theme set aether\n"
					wantMedia = "runtime wallpaper\n"
				} else if failure == "background" {
					wantCommands = "|theme bg set " + wallpaperDest + "\n"
					wantError = "background failed"
				}
				writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
				result, err := writer.ApplyTheme(state, Settings{})
				if result == nil || result.Success || !result.IsOmarchy || err == nil || !strings.Contains(err.Error(), wantError) {
					t.Fatalf("ApplyTheme() = %+v, %v; want %q", result, err, wantError)
				}
				if strings.Contains(err.Error(), "restore previous background") {
					t.Fatalf("background restoration failed: %v", err)
				}
				selected, err := filepath.EvalSymlinks(backgroundLink)
				if err != nil {
					t.Fatalf("restored background did not survive transaction cleanup: %v", err)
				}
				recoveryRoot := filepath.Join(platform.DataDir(), "background-recovery")
				if !strings.HasPrefix(selected, recoveryRoot+string(filepath.Separator)) {
					t.Fatalf("restored background is not a durable recovery copy: %s", selected)
				}
				assertWriterTestFile(t, selected, "runtime wallpaper\n")
				assertWriterTestFile(t, mediaLog, wantMedia)
				assertWriterTestFile(t, commandLog, wantCommands+"|theme bg set "+selected+"\n")
				assertWriterTestFile(t, sourceBackground, "bundle wallpaper\n")
				assertWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors\n")
				assertWriterTestFile(t, filepath.Join(runtimeTheme, "backgrounds", basename), "replacement wallpaper\n")
				if basename != "wall.png" {
					if _, err := os.Stat(runtimeBackground); !os.IsNotExist(err) {
						t.Fatalf("Aether wrote the old wallpaper into Omarchy runtime state: %v", err)
					}
				}
				entries, err := os.ReadDir(recoveryRoot)
				if err != nil || len(entries) != 1 {
					t.Fatalf("expected one retained background snapshot: %v, %v", entries, err)
				}
			})
		}
	}
}

func TestApplyOmarchyThemeDoesNotActivateAfterGenerationFailure(t *testing.T) {
	home := setupWriterTestEnv(t)
	commandLog := filepath.Join(home, "commands")
	t.Setenv("AETHER_COMMAND_LOG", commandLog)
	if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "omarchy"), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$AETHER_COMMAND_LOG\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := platform.OmarchyThemeDir()
	writeWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors")
	if err := omarchy.MarkManagedTheme(target); err != nil {
		t.Fatal(err)
	}
	writer := NewWriter(omarchyV4TestTemplates, "missing")
	result, err := writer.ApplyTheme(NewThemeState(), Settings{})
	if err == nil || result == nil || result.Success {
		t.Fatalf("ApplyTheme() = %+v, %v; want generation failure", result, err)
	}
	assertWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors")
	if _, err := os.Stat(commandLog); !os.IsNotExist(err) {
		t.Fatalf("Omarchy ran after generation failed: %v", err)
	}
}

func TestReplaceThemeDirRetainsBackupWhenRollbackFails(t *testing.T) {
	root := setupWriterTestEnv(t)
	target := filepath.Join(root, "aether")
	staging := filepath.Join(root, ".aether-staging")
	writeWriterTestFile(t, filepath.Join(target, "colors.toml"), "old colors")
	writeWriterTestFile(t, filepath.Join(staging, "colors.toml"), "new colors")
	if err := omarchy.MarkManagedTheme(target); err != nil {
		t.Fatal(err)
	}
	err := replaceThemeDir(staging, target, func() error {
		writeWriterTestFile(t, filepath.Join(staging, "block"), "cannot move installed theme here")
		return os.ErrPermission
	})
	if err == nil || !strings.Contains(err.Error(), "move failed Omarchy theme aside") {
		t.Fatalf("replaceThemeDir() error = %v, want rollback failure", err)
	}
	backups, globErr := filepath.Glob(filepath.Join(root, ".aether.backup-*"))
	if globErr != nil || len(backups) != 1 {
		t.Fatalf("backup lost: %v, %v", backups, globErr)
	}
	assertWriterTestFile(t, filepath.Join(backups[0], "colors.toml"), "old colors")
	if !strings.Contains(err.Error(), backups[0]) {
		t.Fatalf("rollback error does not identify recovery path: %v", err)
	}
}
