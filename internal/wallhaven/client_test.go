package wallhaven

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/platform"
	"aether/internal/wallpaper"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type repeatingReader byte

func (r repeatingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(r)
	}
	return len(p), nil
}

type trackedBody struct {
	io.Reader
	read   int64
	closed bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += int64(n)
	return n, err
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestRejectUnsafeURLsBeforeCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	for _, dir := range []string{platform.DownloadDir(), filepath.Join(platform.CacheDir(), "wallhaven-thumbs")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "image.jpg"), []byte("cached"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	c := NewClient()
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Fatal("unsafe input reached HTTP transport")
		return nil, errors.New("unexpected network request")
	})
	for _, rawURL := range []string{
		"", "file:///image.jpg", "http://example.com/image.jpg", "https://user@example.com/image.jpg",
		"https://127.0.0.1/image.jpg", "https://192.168.1.1/image.jpg", "https://[::1]/image.jpg",
		"https://[::ffff:127.0.0.1]/image.jpg", "https://169.254.169.254/image.jpg",
		"https://localhost/image.jpg", "https://host.local/image.jpg", "https://host.internal/image.jpg",
	} {
		for _, call := range []func(string) (string, error){c.ResolveImageURL, c.Download, c.DownloadThumb, c.DownloadFromURL} {
			if _, err := call(rawURL); err == nil {
				t.Errorf("accepted unsafe URL %q", rawURL)
			}
		}
	}
	for _, id := range []string{"", "..", "../search", "abc?apikey=other", "abc/def", "abc#fragment"} {
		if _, err := c.Info(id); err == nil {
			t.Errorf("accepted unsafe wallpaper ID %q", id)
		}
	}
	for _, rawURL := range []string{"https://example.com/", "https://example.com/..", "https://example.com/%5Cname.jpg", "https://example.com/%00name.jpg"} {
		for _, call := range []func(string) (string, error){c.Download, c.DownloadThumb} {
			if _, err := call(rawURL); err == nil {
				t.Errorf("accepted unsafe download filename from %q", rawURL)
			}
		}
	}
}

func TestResolveImageURL(t *testing.T) {
	c := NewClient()
	c.SetAPIKey("a&b")
	calls := 0
	imageURL := "https://w.wallhaven.cc/full/ab/wallhaven-abc123.jpg"
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Hostname() != "wallhaven.cc" || r.URL.Path != "/api/v1/w/abc123" || r.URL.Query().Get("apikey") != "a&b" {
			t.Fatalf("unexpected API request: host %q, path %q", r.URL.Hostname(), r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":{"path":"` + imageURL + `"}}`))}, nil
	})
	for _, rawURL := range []string{imageURL, "https://example.com/wallhaven.cc/w/abc123", "https://wallhaven.cc.example.com/w/abc123", "https://example.com/?url=wallhaven.cc/w/abc123"} {
		if got, err := c.ResolveImageURL(rawURL); err != nil || got != rawURL || calls != 0 {
			t.Fatalf("direct URL resolution: got %q, error %v, requests %d", got, err, calls)
		}
	}
	for _, page := range []string{"https://wallhaven.cc/w/abc123", "https://www.wallhaven.cc/w/abc123/"} {
		if got, err := c.ResolveImageURL(page); err != nil || got != imageURL {
			t.Fatalf("page resolution: got %q, error %v", got, err)
		}
	}
	for _, bad := range []string{"", "http://example.com/image.jpg", "https://127.0.0.1/image.jpg", "https://user@example.com/image.jpg"} {
		imageURL = bad
		if _, err := c.ResolveImageURL("https://wallhaven.cc/w/abc123"); err == nil {
			t.Errorf("accepted unsafe API image path %q", bad)
		}
	}
}

func TestClientUsesSharedRedirectPolicy(t *testing.T) {
	c := NewClient()
	transport := c.http.Transport.(*http.Transport)
	if transport.Proxy != nil || transport.DialContext == nil || c.http.CheckRedirect == nil {
		t.Fatal("wallhaven client must use public HTTP policy")
	}
	for _, target := range []string{"http://example.com/image.jpg", "https://127.0.0.1/image.jpg", "https://host.internal/image.jpg", "https://user@example.com/image.jpg"} {
		calls := 0
		c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {target}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		})
		if _, err := c.Search(SearchParams{}); err == nil || calls != 1 {
			t.Fatalf("redirect %q: error %v, requests %d", target, err, calls)
		}
	}
}

func TestAPIBodyBounds(t *testing.T) {
	for _, api := range []struct {
		name    string
		payload string
		call    func(*Client) error
	}{
		{"search", `{"data":[],"meta":{"total":0}}`, func(c *Client) error { _, err := c.Search(SearchParams{}); return err }},
		{"info", `{"data":{"path":"https://example.com/image.jpg"}}`, func(c *Client) error { _, err := c.Info("abc123"); return err }},
		{"resolve", `{"data":{"path":"https://example.com/image.jpg"}}`, func(c *Client) error {
			_, err := c.ResolveImageURL("https://wallhaven.cc/w/abc123")
			return err
		}},
	} {
		for _, tc := range []struct {
			name          string
			status        int
			contentLength int64
			padding       int64
			wantRead      int64
			wantErr       string
		}{
			{"valid", 200, -1, 0, int64(len(api.payload)), ""},
			{"exact limit", 200, maxAPIResponseBytes, maxAPIResponseBytes - int64(len(api.payload)), maxAPIResponseBytes, ""},
			{"streamed overflow", 200, -1, maxAPIResponseBytes * 2, maxAPIResponseBytes + 1, "exceeds"},
			{"advertised overflow", 200, maxAPIResponseBytes + 1, 0, 0, "exceeds"},
			{"error body", 500, -1, maxAPIResponseBytes * 2, maxErrorBytes, "returned 500"},
		} {
			t.Run(api.name+"/"+tc.name, func(t *testing.T) {
				body := &trackedBody{Reader: io.MultiReader(strings.NewReader(api.payload), io.LimitReader(repeatingReader(' '), tc.padding))}
				c := NewClient()
				c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: tc.status, Body: body, ContentLength: tc.contentLength}, nil
				})
				err := api.call(c)
				if tc.wantErr == "" && err != nil || tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				if body.read != tc.wantRead || !body.closed {
					t.Fatalf("read %d, closed %v; want %d and closed", body.read, body.closed, tc.wantRead)
				}
				if tc.status != 200 && int64(len(err.Error())) > maxErrorBytes+100 {
					t.Fatal("oversized diagnostic body")
				}
			})
		}
	}
}

func TestDownloadBoundsAndCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	for _, tc := range []struct {
		name  string
		limit int64
		call  func(*Client, string) (string, error)
		dir   string
	}{
		{"image", wallpaper.MaxImageBytes, (*Client).Download, platform.DownloadDir()},
		{"thumbnail", maxThumbnailBytes, (*Client).DownloadThumb, filepath.Join(platform.CacheDir(), "wallhaven-thumbs")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient()
			body := &trackedBody{Reader: strings.NewReader("image data")}
			length := tc.limit + 1
			calls := 0
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: length}, nil
			})
			const rawURL = "https://w.wallhaven.cc/image.jpg?token=value#fragment"
			if _, err := tc.call(c, rawURL); err == nil || !strings.Contains(err.Error(), "exceeds") || body.read != 0 || !body.closed {
				t.Fatalf("advertised overflow: error %v, read %d, closed %v", err, body.read, body.closed)
			}
			length = -1
			path, err := tc.call(c, rawURL)
			if err != nil || path != filepath.Join(tc.dir, "image.jpg") {
				t.Fatalf("download: path %q, error %v", path, err)
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "image data" {
				t.Fatalf("download contents: %q, error %v", data, err)
			}
			if cached, err := tc.call(c, rawURL); err != nil || cached != path || calls != 2 {
				t.Fatalf("cache hit: path %q, error %v, requests %d", cached, err, calls)
			}
			// A sparse file tests the cache size check without allocating a huge fixture.
			if err := os.Truncate(path, tc.limit+1); err != nil {
				t.Fatal(err)
			}
			if _, err := tc.call(c, rawURL); err == nil || !strings.Contains(err.Error(), "exceeds") || calls != 2 {
				t.Fatalf("oversized cache: error %v, requests %d", err, calls)
			}
		})
	}
}

func TestRemoteSourcesWithSameFilenameUseDistinctCacheEntries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	client := NewClient()
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(req.URL.Path))}, nil
	})
	one, err := client.Download("https://example.com/one/wall.png")
	if err != nil {
		t.Fatal(err)
	}
	two, err := client.Download("https://example.com/two/wall.png")
	if err != nil {
		t.Fatal(err)
	}
	if one == two {
		t.Fatal("different remote sources share a cache path")
	}
	data, err := os.ReadFile(two)
	if err != nil || string(data) != "/two/wall.png" {
		t.Fatalf("wrong source bytes: %q, %v", data, err)
	}
}
