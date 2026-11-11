package cli

import (
	"embed"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aether/internal/icontheme"
	"aether/internal/theme"
)

func TestRunGenerateRejectsRemovedGTKOptions(t *testing.T) {
	for _, args := range [][]string{
		{"wallpaper.png", "--gtk"},
		{"wallpaper.png", "--no-gtk"},
		{"--gtk", "wallpaper.png"},
		{"wallpaper.png", "--output", "--gtk"},
	} {
		got, stderr := captureGenerateStderr(t, func() int {
			return runGenerate(args, embed.FS{})
		})
		if got != 1 {
			t.Errorf("runGenerate(%q) = %d, want 1", args, got)
		}
		if !strings.Contains(stderr, "Error: Unknown option:") {
			t.Errorf("runGenerate(%q) stderr = %q, want unknown option error", args, stderr)
		}
	}
}

func TestParseIconThemeOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		want      icontheme.Selection
		remaining []string
		wantErr   bool
	}{
		{
			name:      "omitted is automatic",
			args:      []string{"wallpaper.png", "--no-apply"},
			want:      icontheme.Automatic(),
			remaining: []string{"wallpaper.png", "--no-apply"},
		},
		{
			name:      "explicit automatic",
			args:      []string{"wallpaper.png", "--icon-theme", "automatic", "--no-apply"},
			want:      icontheme.Automatic(),
			remaining: []string{"wallpaper.png", "--no-apply"},
		},
		{
			name: "explicit safe missing ID",
			args: []string{"wallpaper.png", "--icon-theme", "Missing-But-Safe", "--no-apply"},
			want: icontheme.Selection{
				Mode: icontheme.SelectionExplicit,
				ID:   "Missing-But-Safe",
			},
			remaining: []string{"wallpaper.png", "--no-apply"},
		},
		{
			name:    "unsafe ID",
			args:    []string{"wallpaper.png", "--icon-theme", "../escape"},
			wantErr: true,
		},
		{
			name:    "missing value",
			args:    []string{"wallpaper.png", "--icon-theme"},
			wantErr: true,
		},
		{
			name:    "another option is not a value",
			args:    []string{"wallpaper.png", "--icon-theme", "--no-apply"},
			wantErr: true,
		},
		{
			name:    "duplicate option",
			args:    []string{"wallpaper.png", "--icon-theme", "Papirus", "--icon-theme", "Yaru"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, remaining, err := parseIconThemeOption(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseIconThemeOption(%q) = %+v, %q, nil; want error", tt.args, got, remaining)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseIconThemeOption(%q): %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("selection = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(remaining, tt.remaining) {
				t.Errorf("remaining = %q, want %q", remaining, tt.remaining)
			}
		})
	}
}

func captureGenerateStderr(t *testing.T, run func() int) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = original }()

	code := run()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return code, string(output)
}

func TestDisabledTargetsMergesFlagsAndGUIOptOuts(t *testing.T) {
	cfg := isolate(t)
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(cfg, "settings.json"), []byte(`{"includedApps":{"kitty":false,"btop":true}}`), 0o644)
	s := theme.Settings{DisabledTargets: disabledTargets(true, false)}
	if s.TargetEnabled("vscode") || s.TargetEnabled("kitty") {
		t.Fatal("--no-vscode and a GUI opt-out must disable their targets")
	}
	if !s.TargetEnabled("neovim") || !s.TargetEnabled("btop") {
		t.Fatal("other targets must stay enabled")
	}
}
