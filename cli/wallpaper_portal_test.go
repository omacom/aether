package cli

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestWallpaperPathFromURI(t *testing.T) {
	tests := []struct {
		uri  string
		want string
		ok   bool
	}{
		{"file:///home/me/Pictures/sky.jpg", "/home/me/Pictures/sky.jpg", true},
		{"file:///home/me/My%20Pictures/sky.png", "/home/me/My Pictures/sky.png", true},
		{"https://example.com/sky.jpg", "", false},
		{"file:///home/me/notes.txt", "", false},
		{"file://", "", false},
	}
	for _, tt := range tests {
		got, err := wallpaperPathFromURI(tt.uri)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("wallpaperPathFromURI(%q) = %q, %v; want %q, ok=%v", tt.uri, got, err, tt.want, tt.ok)
		}
	}
}

func TestSetWallpaperURIRejectsLockScreenOnly(t *testing.T) {
	options := map[string]dbus.Variant{"set-on": dbus.MakeVariant("lockscreen")}
	got, err := wallpaperPortal{}.SetWallpaperURI("/request", "", "", "file:///home/me/sky.jpg", options)
	if err != nil || got != portalResponseOther {
		t.Fatalf("SetWallpaperURI() = %d, %v; want %d", got, err, portalResponseOther)
	}
}

func TestSetWallpaperURIRejectsNonImage(t *testing.T) {
	got, err := wallpaperPortal{}.SetWallpaperURI("/request", "", "", "file:///home/me/notes.txt", nil)
	if err != nil || got != portalResponseOther {
		t.Fatalf("SetWallpaperURI() = %d, %v; want %d", got, err, portalResponseOther)
	}
}
