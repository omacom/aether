package wallpaper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRemoteURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "public HTTPS", url: "https://8.8.8.8/theme.json"},
		{name: "HTTP", url: "http://example.com/theme.json", wantErr: true},
		{name: "credentials", url: "https://user:pass@example.com/theme.json", wantErr: true},
		{name: "loopback", url: "https://127.0.0.1/theme.json", wantErr: true},
		{name: "private IPv4", url: "https://192.168.1.1/theme.json", wantErr: true},
		{name: "link local", url: "https://169.254.169.254/latest/meta-data", wantErr: true},
		{name: "private IPv6", url: "https://[fd00::1]/theme.json", wantErr: true},
		{name: "localhost name", url: "https://localhost/theme.json", wantErr: true},
		{name: "mDNS name", url: "https://host.local/theme.json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemoteURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRemoteURL(%q) error = %v; wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloadContextReachesTransportAndPreventsPublication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cancel()
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	dir := t.TempDir()
	err := DownloadFileContext(ctx, client, "https://example.com/wallpaper.png", filepath.Join(dir, "wallpaper.png"), 1024)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("cancelled download leaves files: %v, %v", entries, err)
	}
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

type trackedBody struct {
	io.Reader
	read   int
	closed bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += n
	return n, err
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestPublicHTTPClientPolicy(t *testing.T) {
	client := NewPublicHTTPClient()
	defer client.CloseIdleConnections()
	transport := client.Transport.(*http.Transport)
	if transport.Proxy != nil || transport.DialContext == nil || client.CheckRedirect == nil {
		t.Fatal("public HTTP client must disable proxies and enforce dial/redirect checks")
	}
	for _, address := range []string{"127.0.0.1:443", "192.168.1.1:443", "[::1]:443", "[::ffff:127.0.0.1]:443", "[fe80::1]:443", "100.64.0.1:443"} {
		// Literal addresses need neither DNS nor a network connection to reject.
		conn, err := transport.DialContext(context.Background(), "tcp", address)
		if conn != nil {
			conn.Close()
		}
		if err == nil || !strings.Contains(err.Error(), "no reachable public address") {
			t.Errorf("dial %s: error = %v; want non-public rejection", address, err)
		}
	}
	for _, target := range []string{"http://example.com/a", "https://127.0.0.1/a", "https://user@example.com/a", "https://host.internal/a", "https://example.com/loop"} {
		t.Run(target, func(t *testing.T) {
			calls := 0
			client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {target}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			})
			_, err := client.Get("https://example.com/start")
			wantCalls := 1
			if target == "https://example.com/loop" {
				wantCalls = 5
			}
			if err == nil || calls != wantCalls {
				t.Fatalf("redirect: error %v, requests %d; want error after %d", err, calls, wantCalls)
			}
		})
	}
}

func TestDownloadFileBoundsAndCleanup(t *testing.T) {
	for _, tc := range []struct {
		name          string
		body          string
		contentLength int64
		limit         int64
		wantErr       bool
		wantRead      int
	}{
		{"exact limit", "1234", 4, 4, false, 4},
		{"streamed overflow", "123456789", -1, 4, true, 5},
		{"misleading length", "123456789", 1, 4, true, 5},
		{"advertised overflow", "12345", 5, 4, true, 0},
		{"zero limit", "1234", -1, 0, true, 0},
		{"negative limit", "1234", -1, -1, true, 0},
		{"overflowing limit", "1234", -1, 1<<63 - 1, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			dest := filepath.Join(dir, "image.png")
			body := &trackedBody{Reader: strings.NewReader(tc.body)}
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: tc.contentLength}, nil
			})}
			err := DownloadFile(client, "https://example.com/image.png", dest, tc.limit)
			if (err != nil) != tc.wantErr || body.read != tc.wantRead {
				t.Fatalf("error %v, read %d; want error %v, read %d", err, body.read, tc.wantErr, tc.wantRead)
			}
			if calls > 0 && !body.closed {
				t.Error("response body was not closed")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantErr {
				if len(entries) != 0 {
					t.Fatalf("failed download left files: %v", entries)
				}
			} else if len(entries) != 1 || entries[0].Name() != "image.png" {
				t.Fatalf("expected only the published file, got %v", entries)
			}
		})
	}
}

func TestDownloadFilePublishesOnlyCompleteData(t *testing.T) {
	for _, fail := range []bool{false, true} {
		dir := t.TempDir()
		dest := filepath.Join(dir, "image.png")
		reads := 0
		body := &trackedBody{Reader: readerFunc(func(p []byte) (int, error) {
			reads++
			if reads == 1 {
				return copy(p, "partial"), nil
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatalf("partial download became visible: %v", err)
			}
			if fail {
				return 0, errors.New("interrupted download")
			}
			return copy(p, " complete"), io.EOF
		})}
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body, ContentLength: -1}, nil
		})}
		err := DownloadFile(client, "https://example.com/image.png", dest, 100)
		if (err != nil) != fail || !body.closed {
			t.Fatalf("error %v, closed %v; want failure %v", err, body.closed, fail)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if fail {
			if len(entries) != 0 {
				t.Fatalf("interrupted download left files: %v", entries)
			}
		} else {
			data, err := os.ReadFile(dest)
			if err != nil || string(data) != "partial complete" || len(entries) != 1 {
				t.Fatalf("publication: data %q, files %v, error %v", data, entries, err)
			}
			info, err := os.Stat(dest)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("downloaded image mode = %04o; want 0600", info.Mode().Perm())
			}
		}
	}
}

func TestDownloadFileValidatesBeforeCacheHit(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "image.png")
	if err := os.WriteFile(dest, []byte("cached"), 0600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Fatal("unexpected network request")
		return nil, errors.New("unexpected network request")
	})}
	for _, rawURL := range []string{"http://example.com/image.png", "https://127.0.0.1/image.png", "https://localhost/image.png"} {
		if err := DownloadFile(client, rawURL, dest, 100); err == nil {
			t.Errorf("accepted unsafe cached URL %q", rawURL)
		}
	}
	if err := DownloadFile(client, "https://example.com/image.png", dest, 100); err != nil {
		t.Fatalf("safe cache hit: %v", err)
	}
	link := filepath.Join(dir, "link.png")
	if err := os.Symlink(dest, link); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{dir, link} {
		if err := DownloadFile(client, "https://example.com/image.png", target, 100); err == nil {
			t.Errorf("accepted non-regular cache entry %q", target)
		}
	}
}

func pngHeader(width, height uint32) []byte {
	data := make([]byte, 33)
	copy(data, "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	binary.BigEndian.PutUint32(data[16:20], width)
	binary.BigEndian.PutUint32(data[20:24], height)
	data[24], data[25] = 8, 6
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	return data
}

func TestImageHeaderLimits(t *testing.T) {
	for _, size := range [][2]uint32{{16384, 1}, {8000, 5000}, {16385, 1}, {1, 16385}, {8000, 5001}} {
		path := filepath.Join(t.TempDir(), "header.png")
		if err := os.WriteFile(path, pngHeader(size[0], size[1]), 0600); err != nil {
			t.Fatal(err)
		}
		unsafe := size[0] > maxImageDimension || size[1] > maxImageDimension || uint64(size[0])*uint64(size[1]) > maxImagePixels
		err := ValidateImageFile(path)
		if (err != nil) != unsafe {
			t.Errorf("header %dx%d: error %v, want rejection %v", size[0], size[1], err, unsafe)
		}
		if unsafe {
			if _, err := loadImage(path); err == nil || !strings.Contains(err.Error(), "unsafe image dimensions") {
				t.Errorf("loadImage %dx%d: error %v; want dimension limit", size[0], size[1], err)
			}
		}
	}
}

func TestIsPublicAddressRejectsSpecialRanges(t *testing.T) {
	for _, raw := range []string{"100.64.0.1", "198.18.0.1", "192.0.2.1", "2001:db8::1"} {
		if isPublicAddress(netip.MustParseAddr(raw)) {
			t.Errorf("isPublicAddress(%q) = true; want false", raw)
		}
	}
	if !isPublicAddress(netip.MustParseAddr("8.8.8.8")) {
		t.Error("public address was rejected")
	}
}

func TestDownloadToCacheRejectsOversizedCachedFile(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	rawURL := "https://8.8.8.8/theme.json"
	dir, err := webImportsDir()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(rawURL))
	name := hex.EncodeToString(sum[:8]) + extFromURL(rawURL)
	if err := os.WriteFile(filepath.Join(dir, name), []byte("too large"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = DownloadToCache(rawURL, 4)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("DownloadToCache() error = %v; want size-limit error", err)
	}
}

func TestValidateImageFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallpaper.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.White)
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ValidateImageFile(path); err != nil {
		t.Fatalf("ValidateImageFile() error = %v", err)
	}

	invalid := filepath.Join(t.TempDir(), "not-an-image.png")
	if err := os.WriteFile(invalid, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateImageFile(invalid); err == nil {
		t.Fatal("ValidateImageFile() accepted invalid image data")
	}
}
