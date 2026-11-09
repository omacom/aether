package theme

import (
	"path/filepath"
	"testing"

	"aether/internal/platform"
)

func TestCandidatePathsUsesStandaloneTheme(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	paths := candidatePaths()
	want := filepath.Join(platform.ThemeDir(), "colors.toml")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("candidatePaths() = %v, want [%q]", paths, want)
	}
}
