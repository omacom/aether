package extraction

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndSamplePixelsWebP(t *testing.T) {
	const fixture = "UklGRjYAAABXRUJQVlA4ICoAAACwAQCdASoEAAQAAgA0JaACdLoABGaAAP7udn/3BmfV2OH9zcW5+hQAAAA="

	data, err := base64.StdEncoding.DecodeString(fixture)
	if err != nil {
		t.Fatalf("decode WebP fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "fixture.webp")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write WebP fixture: %v", err)
	}

	pixels, err := LoadAndSamplePixels(path)
	if err != nil {
		t.Fatalf("LoadAndSamplePixels() error = %v", err)
	}
	if len(pixels) == 0 {
		t.Fatal("LoadAndSamplePixels() returned no pixels")
	}
}

func TestLoadAndSamplePixelsRejectsUnsafeHeader(t *testing.T) {
	for _, size := range [][2]uint32{{16385, 1}, {1, 16385}, {10000, 5000}} {
		// A header alone exercises the allocation guard without a giant image.
		data := make([]byte, 33)
		copy(data, "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
		binary.BigEndian.PutUint32(data[16:20], size[0])
		binary.BigEndian.PutUint32(data[20:24], size[1])
		data[24], data[25] = 8, 6
		binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
		path := filepath.Join(t.TempDir(), "oversized.png")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadAndSamplePixels(path); err == nil || !strings.Contains(err.Error(), "unsafe image dimensions") {
			t.Errorf("header %dx%d: error = %v; want dimension limit", size[0], size[1], err)
		}
	}
}
