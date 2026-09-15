package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/omarchy"
	"aether/internal/platform"
	"aether/internal/theme"
)

func TestThemeFolderExists(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "standalone", true: "native"}[native], func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			bin := t.TempDir()
			t.Setenv("PATH", bin)
			root := platform.SavedThemesDir()
			if native {
				if err := os.WriteFile(filepath.Join(bin, "omarchy"), []byte("#!/bin/sh\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				root = omarchy.UserThemesDir()
			}
			app := NewApp()
			if app.ThemeFolderExists("midnight") {
				t.Fatal("missing folder exists")
			}
			if err := os.MkdirAll(filepath.Join(root, "midnight"), 0o755); err != nil {
				t.Fatal(err)
			}
			if !app.ThemeFolderExists(" Midnight ") {
				t.Fatal("existing folder is missing")
			}
			for _, bad := range []string{"", "-x", "foo/bar", "foo bar", "..", "."} {
				if app.ThemeFolderExists(bad) {
					t.Errorf("invalid folder name accepted: %q", bad)
				}
			}
		})
	}
}

func TestBlueprintKeepsWallpaperSourceAndBlurIntent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	app := NewApp()
	source := writeTestImage(t, t.TempDir(), "source.png", 16, 16)
	if err := app.SaveBlueprint(SaveBlueprintRequest{Name: "Blurred", Palette: theme.DefaultPalette[:], WallpaperPath: source, WallpaperBlur: true}); err != nil {
		t.Fatal(err)
	}
	if err := app.LoadBlueprint("Blurred"); err != nil {
		t.Fatal(err)
	}
	if app.state.WallpaperPath != source || !app.state.WallpaperBlur {
		t.Fatal("blueprint loses the source or blur intent")
	}
	listed, err := app.ListBlueprints()
	if err != nil {
		t.Fatal(err)
	}
	palette := listed[0]["palette"].(map[string]interface{})
	if palette["wallpaper"] != source || palette["wallpaperBlur"] != true {
		t.Fatalf("listed wallpaper = %+v", palette)
	}
}

func TestWallpaperOnlyPreservesActiveTheme(t *testing.T) {
	home, bin := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("PATH", bin)
	log := filepath.Join(t.TempDir(), "commands")
	t.Setenv("AETHER_COMMAND_LOG", log)
	script := "#!/bin/sh\nprintf '%s|%s\\n' \"$#\" \"$*\" >> \"$AETHER_COMMAND_LOG\"\n"
	if err := os.WriteFile(filepath.Join(bin, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(omarchy.CurrentStateDir(), "theme.name")
	if err := os.MkdirAll(filepath.Dir(active), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(active, []byte("existing-theme\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := writeTestImage(t, t.TempDir(), "wall paper.png", 16, 16)
	app := NewApp()
	before := app.state.Palette
	if err := app.ApplyWallpaperOnly(source); err != nil {
		t.Fatal(err)
	}
	commands, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(string(commands)), "4|theme bg set "+source; got != want {
		t.Fatalf("commands = %q, want %q", got, want)
	}
	current, err := os.ReadFile(active)
	if err != nil || string(current) != "existing-theme\n" || app.state.Palette != before {
		t.Fatal("wallpaper-only operation changes the theme")
	}
	if _, err := os.Stat(platform.OmarchyThemeDir()); !os.IsNotExist(err) {
		t.Fatalf("wallpaper-only operation creates a theme: %v", err)
	}
}

func TestSaveAndApplyThemeActivatesTheRenderedVariant(t *testing.T) {
	home, bin := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("PATH", bin)
	log := filepath.Join(t.TempDir(), "commands")
	t.Setenv("AETHER_COMMAND_LOG", log)
	script := "#!/bin/sh\nprintf '%s|%s\\n' \"$OMARCHY_THEME_SKIP_BACKGROUND\" \"$*\" >> \"$AETHER_COMMAND_LOG\"\n"
	if err := os.WriteFile(filepath.Join(bin, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	source := writeTestImage(t, t.TempDir(), "source.png", 16, 16)
	app := NewApp()
	result, err := app.SaveAndApplyTheme(SaveAndApplyThemeRequest{Name: "blur-theme", Palette: theme.DefaultPalette[:], WallpaperPath: source, WallpaperBlur: true})
	if err != nil || result == nil || !result.Success {
		t.Fatalf("save and apply: %+v, %v", result, err)
	}
	commands, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(commands)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "source-blurred-") || lines[1] != "1|theme set blur-theme" {
		t.Fatalf("wrong activation: %q", lines)
	}
	if _, err := os.Stat(filepath.Join(result.ThemePath, "backgrounds", "source.png")); err != nil {
		t.Fatal("original image is missing from the theme")
	}
}

func writeTestImage(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 0xff})
		}
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test image: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	return path
}

func TestBlurWallpaperCreatesVariant(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	app := NewApp()
	src := writeTestImage(t, t.TempDir(), "photo.png", 128, 80)

	got, err := app.BlurWallpaper(src)
	if err != nil {
		t.Fatalf("BlurWallpaper: %v", err)
	}
	if got == "" {
		t.Fatal("BlurWallpaper returned empty path")
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("variant not created: %v", err)
	}
	ext := filepath.Ext(got)
	if ext != ".jpg" {
		t.Errorf("variant ext = %q; want .jpg", ext)
	}
}

func TestBlurWallpaperRejectsNonImage(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	app := NewApp()
	txt := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(txt, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := app.BlurWallpaper(txt); err == nil {
		t.Error("BlurWallpaper() error = nil; want error for non-image file")
	}
}

func TestBlurWallpaperRejectsMissingFile(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	app := NewApp()
	if _, err := app.BlurWallpaper("/nonexistent/photo.png"); err == nil {
		t.Error("BlurWallpaper() error = nil; want error for missing file")
	}
}
