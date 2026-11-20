package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func writeTestPNG(t *testing.T, path string, fill color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSaveWallpaperToFolder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	folder := filepath.Join(t.TempDir(), "Wallpapers")
	app := NewApp()
	if err := app.SaveSettings(map[string]interface{}{
		wallpaperFolderSetting: folder,
	}); err != nil {
		t.Fatal(err)
	}

	srcDir := t.TempDir()
	first := filepath.Join(srcDir, "a1b2c3d4.png")
	second := filepath.Join(srcDir, "e5f6a7b8.png")
	writeTestPNG(t, first, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	writeTestPNG(t, second, color.RGBA{R: 200, G: 100, B: 50, A: 255})

	// Uses preferred filename when valid.
	saved, err := app.SaveWallpaperToFolder(first, "nested/mountain.png")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(folder, "mountain.png")
	if saved != want {
		t.Fatalf("saved path = %q; want %q", saved, want)
	}

	// Re-saving identical content is idempotent.
	again, err := app.SaveWallpaperToFolder(first, "mountain.png")
	if err != nil {
		t.Fatal(err)
	}
	if again != want {
		t.Fatalf("idempotent save path = %q; want %q", again, want)
	}

	// Saving different content with the same filename avoids overwriting.
	collision, err := app.SaveWallpaperToFolder(second, "mountain.png")
	if err != nil {
		t.Fatal(err)
	}
	wantCollision := filepath.Join(folder, "mountain-2.png")
	if collision != wantCollision {
		t.Fatalf("collision save path = %q; want %q", collision, wantCollision)
	}

	// Rejects non-image files.
	notImage := filepath.Join(srcDir, "notes.txt")
	if err := os.WriteFile(notImage, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := app.SaveWallpaperToFolder(notImage, ""); err == nil {
		t.Fatal("SaveWallpaperToFolder(notes.txt) error = nil; want error")
	}
}

func TestAddWallpaperToCurrentTheme(t *testing.T) {
	themeBgDir := filepath.Join(t.TempDir(), "backgrounds", "tokyo-night")
	prev := currentThemeBackgroundsDir
	currentThemeBackgroundsDir = func() (string, string, error) {
		return themeBgDir, "tokyo-night", nil
	}
	t.Cleanup(func() { currentThemeBackgroundsDir = prev })

	src := filepath.Join(t.TempDir(), "wallhaven-abc123.png")
	writeTestPNG(t, src, color.RGBA{R: 40, G: 60, B: 90, A: 255})

	themeName, err := NewApp().AddWallpaperToCurrentTheme(src, "")
	if err != nil {
		t.Fatal(err)
	}
	if themeName != "tokyo-night" {
		t.Fatalf("themeName = %q; want %q", themeName, "tokyo-night")
	}
	if _, err := os.Stat(filepath.Join(themeBgDir, "wallhaven-abc123.png")); err != nil {
		t.Fatalf("expected wallpaper in theme backgrounds dir: %v", err)
	}
}

func TestSaveWallpaperToFolderConcurrentSameName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	folder := filepath.Join(t.TempDir(), "Wallpapers")
	app := NewApp()
	if err := app.SaveSettings(map[string]interface{}{
		wallpaperFolderSetting: folder,
	}); err != nil {
		t.Fatal(err)
	}

	const count = 12
	srcDir := t.TempDir()
	srcPaths := make([]string, count)
	srcContents := make([][]byte, count)
	for i := 0; i < count; i++ {
		p := filepath.Join(srcDir, fmt.Sprintf("src-%d.png", i))
		writeTestPNG(t, p, color.RGBA{R: uint8(i * 20), G: uint8(10 + i*10), B: uint8(200 - i*10), A: 255})
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		srcPaths[i] = p
		srcContents[i] = data
	}

	type result struct {
		path string
		err  error
	}
	results := make([]result, count)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			path, err := app.SaveWallpaperToFolder(srcPaths[idx], "shared.png")
			results[idx] = result{path: path, err: err}
		}(i)
	}
	close(start)
	wg.Wait()

	seenPaths := make(map[string]bool, count)
	for i, res := range results {
		if res.err != nil {
			t.Fatalf("save %d failed: %v", i, res.err)
		}
		if seenPaths[res.path] {
			t.Fatalf("duplicate destination path %q returned for distinct images", res.path)
		}
		seenPaths[res.path] = true
		got, err := os.ReadFile(res.path)
		if err != nil {
			t.Fatalf("read saved file %q: %v", res.path, err)
		}
		if !bytes.Equal(got, srcContents[i]) {
			t.Fatalf("saved file %q was overwritten with different content", res.path)
		}
	}
}
