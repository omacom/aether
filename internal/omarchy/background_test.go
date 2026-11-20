package omarchy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserBackgroundsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := UserBackgroundsDir("tokyo-night")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "omarchy", "backgrounds", "tokyo-night")
	if got != want {
		t.Fatalf("UserBackgroundsDir = %q; want %q", got, want)
	}

	for _, bad := range []string{"", " ", ".", "..", "../evil", "a/b", `a\b`, "a\x00b"} {
		if _, err := UserBackgroundsDir(bad); err == nil {
			t.Errorf("UserBackgroundsDir(%q) error = nil; want error", bad)
		}
	}
}

func TestCurrentThemeBackgroundsDir(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir)

	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := CurrentThemeBackgroundsDir(); err == nil {
		t.Fatal("CurrentThemeBackgroundsDir() without active theme error = nil; want error")
	}

	stateDir := filepath.Join(home, ".local", "state", "omarchy", "current")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "theme.name"), []byte("ethereal\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir, name, err := CurrentThemeBackgroundsDir()
	if err != nil {
		t.Fatal(err)
	}
	if name != "ethereal" {
		t.Errorf("theme name = %q; want %q", name, "ethereal")
	}
	want := filepath.Join(home, ".config", "omarchy", "backgrounds", "ethereal")
	if dir != want {
		t.Errorf("dir = %q; want %q", dir, want)
	}
}
