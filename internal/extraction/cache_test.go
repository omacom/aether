package extraction

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractionRejectsUnknownModesBeforeCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	imagePath := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(imagePath, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	var palette [16]string
	for i := range palette {
		palette[i] = "#123456"
	}
	content, err := json.Marshal(cacheData{Palette: palette, Version: CacheVersion})
	if err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(getCacheDir(), "..", "sentinel.json")
	if err := os.MkdirAll(getCacheDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sentinel, content, 0600); err != nil {
		t.Fatal(err)
	}

	for _, mode := range []string{"/../../sentinel", `..\sentinel`, "unknown", "", "Normal"} {
		t.Run(mode, func(t *testing.T) {
			if key := buildCacheKey(GetCacheKey(imagePath, false), mode); key != "" {
				t.Errorf("unsafe mode produced cache key %q", key)
			}
			if _, err := ExtractColors(imagePath, false, mode); err == nil || !strings.Contains(err.Error(), "unknown extraction mode") {
				t.Errorf("single extraction error = %v; want unknown mode", err)
			}
			if _, _, err := ExtractColorsFromImages([]string{imagePath}, false, mode); err == nil || !strings.Contains(err.Error(), "unknown extraction mode") {
				t.Errorf("multi extraction error = %v; want unknown mode", err)
			}
		})
	}

	for _, key := range []string{"../sentinel", "/../../sentinel", `..\sentinel`, "", "a/b", "a\x00b", strings.Repeat("a", 129)} {
		if _, ok := LoadCachedPalette(key); ok {
			t.Errorf("loaded unsafe cache key %q", key)
		}
		SavePaletteToCache(key, [16]string{})
		if got := buildCacheKey(key, "normal"); got != "" {
			t.Errorf("unsafe base produced cache key %q", got)
		}
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("traversal modified sentinel: content %q, error %v", got, err)
	}
	entries, err := os.ReadDir(getCacheDir())
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid inputs created cache files: %v, error %v", entries, err)
	}
}

func TestKnownModesPreserveCacheKeys(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	imagePath := filepath.Join(t.TempDir(), "cached.png")
	if err := os.WriteFile(imagePath, []byte("cache hit must avoid decoding"), 0600); err != nil {
		t.Fatal(err)
	}
	modes := []string{
		"normal", "monochromatic", "analogous", "pastel", "material", "colorful", "muted", "bright",
		"complementary", "triadic", "split-complementary", "tetradic", "fire", "ocean", "forest",
		"earthtone", "neon", "sunset", "vaporwave", "midnight", "aurora", "high-contrast", "duotone",
	}
	var want [16]string
	for i := range want {
		want[i] = "#abcdef"
	}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			for _, light := range []bool{false, true} {
				for _, base := range []string{GetCacheKey(imagePath, light), GetMultiCacheKey([]string{imagePath}, light)} {
					key := base + "_" + mode
					if mode == "normal" {
						key = base + "_v6"
					} else if mode == "monochromatic" {
						key += "_v5"
					}
					if got := buildCacheKey(base, mode); got != key {
						t.Fatalf("cache key = %q, want %q", got, key)
					}
					SavePaletteToCache(key, want)
				}
				if got, err := ExtractColors(imagePath, light, mode); err != nil || got != want {
					t.Fatalf("single cache hit: palette %v, error %v", got, err)
				}
				if got, skipped, err := ExtractColorsFromImages([]string{imagePath}, light, mode); err != nil || skipped != 0 || got != want {
					t.Fatalf("multi cache hit: palette %v, skipped %d, error %v", got, skipped, err)
				}
			}
		})
	}
}
