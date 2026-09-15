package theme

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"aether/internal/platform"
)

func TestGenerateWallpaperVariantPreservesSourceAndRestoresCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	state := NewThemeState()
	state.WallpaperPath = source
	state.WallpaperBlur = true
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	for _, native := range []bool{false, true} {
		output := t.TempDir()
		generate := writer.GenerateOnly
		if native {
			generate = writer.GenerateOmarchyV4Only
		}
		for i := 0; i < 2; i++ {
			if err := generate(state, DefaultApplySettings(), output); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(filepath.Join(output, "backgrounds"))
			if err != nil || len(entries) != 2 {
				t.Fatalf("backgrounds = %v, error = %v", entries, err)
			}
			copied, err := os.ReadFile(filepath.Join(output, "backgrounds", "source.png"))
			if err != nil || !bytes.Equal(copied, data.Bytes()) {
				t.Fatal("original image changes")
			}
			if state.WallpaperPath != source || !state.WallpaperBlur || state.OriginalWallpaperPath != "" {
				t.Fatal("generation changes editor source state")
			}
			if err := os.RemoveAll(platform.BlurDir()); err != nil {
				t.Fatal(err)
			}
		}
	}
}
