package wallpaper

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type observedImage struct {
	image.Image
	once  sync.Once
	check func()
}

func (img *observedImage) At(x, y int) color.Color {
	img.once.Do(img.check)
	return img.Image.At(x, y)
}

func TestWritePNGAtomicPublication(t *testing.T) {
	for _, existing := range []bool{false, true} {
		dir := t.TempDir()
		dest := filepath.Join(dir, "preview.png")
		old := []byte("previous cached image")
		if existing {
			if err := os.WriteFile(dest, old, 0600); err != nil {
				t.Fatal(err)
			}
		}
		checked := false
		img := &observedImage{Image: image.NewRGBA(image.Rect(0, 0, 16, 8)), check: func() {
			checked = true
			data, err := os.ReadFile(dest)
			if existing {
				if err != nil || !bytes.Equal(data, old) {
					t.Fatalf("encoding changed published file: data %q, error %v", data, err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("partial PNG was published during encoding: %v", err)
			}
		}}
		if err := writePNG(dest, img); err != nil {
			t.Fatal(err)
		}
		if !checked {
			t.Fatal("encoding observation did not run")
		}
		info, err := os.Stat(dest)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("published PNG mode = %04o (existing: %t); want 0600", info.Mode().Perm(), existing)
		}
		if _, err := loadImage(dest); err != nil {
			t.Fatalf("published PNG is incomplete: %v", err)
		}
		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if err := writePNG(dest, image.NewRGBA(image.Rectangle{})); err == nil {
			t.Fatal("expected encoding error for empty image")
		}
		after, err := os.ReadFile(dest)
		if err != nil || !bytes.Equal(after, data) {
			t.Fatal("failed encoding changed the published image")
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 1 {
			t.Fatalf("temporary files leaked: %v, error %v", entries, err)
		}
	}
}

func TestThumbnailAndPreview(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 16, 8))); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		get  func(string) (string, error)
		size int
	}{{"thumbnail", GetThumbnail, thumbnailSize}, {"preview", GetPreview, previewSize}} {
		t.Run(tc.name, func(t *testing.T) {
			path, err := tc.get(source)
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("generated image mode = %04o; want 0600", info.Mode().Perm())
			}
			img, err := loadImage(path)
			if err != nil {
				t.Fatal(err)
			}
			if img.Bounds().Dx() != tc.size || img.Bounds().Dy() != tc.size/2 {
				t.Fatalf("incorrect scaled bounds: %v", img.Bounds())
			}
			if cached, err := tc.get(source); err != nil || cached != path {
				t.Fatalf("cache hit: path %q, error %v", cached, err)
			}
		})
	}
}
