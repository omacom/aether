package githubsource

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aether/internal/wallpaper"
)

func TestParseRepositoryURLs(t *testing.T) {
	for _, test := range []struct{ url, branch, path string }{
		{"https://github.com/example-user/wallpapers", "", ""},
		{"https://github.com/example-user/wallpapers.git", "", ""},
		{"https://github.com/example-user/wallpapers/tree/main/nature", "main", "nature"},
		{"https://github.com/example-user/wallpapers/blob/main/one.png", "main", "one.png"},
		{"https://github.com/example-user/wallpapers/tree/feature%2Fimages/folder%20name", "feature/images", "folder name"},
		{"https://raw.githubusercontent.com/example-user/wallpapers/main/one.png", "main", "one.png"},
		{"https://raw.githubusercontent.com/example-user/wallpapers/refs/heads/main/one.png", "main", "one.png"},
	} {
		t.Run(test.url, func(t *testing.T) {
			got, err := parseURL(test.url)
			if err != nil || got.Owner != "example-user" || got.Repo != "wallpapers" || got.Branch != test.branch || got.Path != test.path {
				t.Fatalf("parsed URL = %+v, error = %v", got, err)
			}
		})
	}
}

func TestRejectUnsupportedRepositoryURLs(t *testing.T) {
	for _, value := range []string{
		"", "http://github.com/owner/repo", "https://user@github.com/owner/repo",
		"https://127.0.0.1/repo", "https://gitlab.com/owner/repo", "https://github.com/owner",
		"https://github.com/owner/repo/tree", "https://github.com/owner/repo/%2e%2e/other",
		"https://github.com/owner%2frepo/other", "https://github.com:8443/owner/repo",
		"https://example-user.github.io/wallpapers", "https://raw.githubusercontent.com/owner/repo/main",
	} {
		if _, err := parseURL(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestListImagesUsesEscapedRequestsAndCachesMetadata(t *testing.T) {
	client := NewClient()
	calls := 0
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Host != "api.github.com" || req.URL.Query().Get("ref") != "feature/images" || req.URL.Path != "/repos/owner/repo/contents/folder name" {
			t.Fatalf("request = %s", req.URL)
		}
		body := `[{"name":"dir","path":"dir","type":"dir"},{"name":"one.png","path":"one.png","type":"file","download_url":"https://raw.githubusercontent.com/owner/repo/main/one.png"},{"name":"bad.png","type":"file","download_url":"https://127.0.0.1/bad.png"},{"name":"README.md","type":"file"}]`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	for i := 0; i < 2; i++ {
		result, err := client.ListImages("https://github.com/owner/repo/tree/feature%2Fimages/folder%20name")
		if err != nil || len(result.Items) != 2 || result.Items[1].Name != "one.png" {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
	}
	if calls != 1 {
		t.Fatalf("metadata cache makes %d requests", calls)
	}
}

func TestListImagesBoundsResponsesAndReturnsErrors(t *testing.T) {
	for _, status := range []int{403, 404, 429, 500, 200} {
		client := NewClient()
		client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", maxAPIBytes+1)))}, nil
		})
		if _, err := client.ListImages("https://github.com/owner/repo"); err == nil {
			t.Errorf("HTTP %d accepts an error or oversized body", status)
		}
	}
}

func TestThumbnailUsesSharedImageLimits(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 20, 10))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := downloadThumbnail("https://example.com/image.png", func(url string, limit int64) (string, error) {
		if limit != wallpaper.MaxImageBytes {
			t.Fatal("download limit changed")
		}
		return path, nil
	})
	if err != nil || result.Width != 20 || result.Height != 10 {
		t.Fatalf("thumbnail = %+v, error = %v", result, err)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(result.DataURL, "data:image/png;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width > 300 || config.Height > 300 {
		t.Fatalf("thumbnail dimensions = %+v, %v", config, err)
	}
	oversized := append([]byte(nil), encoded.Bytes()...)
	binary.BigEndian.PutUint32(oversized[16:20], 20000)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	if err := os.WriteFile(path, oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := downloadThumbnail("https://example.com/huge.png", func(string, int64) (string, error) { return path, nil }); err == nil {
		t.Fatal("oversized source header is accepted")
	}
}

func TestThumbnailRejectsPrivateDestinationsBeforeDownload(t *testing.T) {
	for _, value := range []string{"http://example.com/image.png", "https://127.0.0.1/image.png", "https://192.168.1.1/image.png", "https://localhost/image.png"} {
		_, err := downloadThumbnail(value, func(string, int64) (string, error) { t.Fatal("private destination reaches downloader"); return "", nil })
		if err == nil {
			t.Errorf("accepted %s", value)
		}
	}
}

func TestMetadataCacheExpiryAndEviction(t *testing.T) {
	cache := newTTLCache(time.Minute, 2)
	for _, key := range []string{"one", "two", "three"} {
		cache.set(key, &ListContentsResult{})
	}
	if _, ok := cache.get("one"); ok {
		t.Fatal("oldest entry survives eviction")
	}
	if _, ok := cache.get("three"); !ok {
		t.Fatal("new entry is missing")
	}
	cache.items["three"].expiresAt = time.Now().Add(-time.Second)
	if _, ok := cache.get("three"); ok {
		t.Fatal("expired entry survives")
	}
	cache.clear()
	if _, ok := cache.get("two"); ok {
		t.Fatal("clear preserves an entry")
	}
}
