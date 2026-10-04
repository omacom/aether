package update

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeFakeCommand(t *testing.T, dir, name string, exitCode int) {
	t.Helper()
	script := []byte(fmt.Sprintf("#!/bin/sh\nexit %d\n", exitCode))
	if err := os.WriteFile(filepath.Join(dir, name), script, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDetectPackageUpdateCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("package detection runs on Linux only")
	}
	tests := []struct {
		name     string
		pacman   int // -1 means no pacman on PATH
		commands []string
		want     string
	}{
		{"no pacman", -1, []string{"omarchy-update"}, ""},
		{"not owned by a package", 1, []string{"omarchy-update"}, ""},
		{"omarchy", 0, []string{"omarchy-update", "yay"}, "omarchy-update"},
		{"aur helper", 0, []string{"paru", "yay"}, "yay -Syu"},
		{"plain pacman", 0, nil, "sudo pacman -Syu"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("PATH", dir)
			if tt.pacman >= 0 {
				writeFakeCommand(t, dir, "pacman", tt.pacman)
			}
			for _, command := range tt.commands {
				writeFakeCommand(t, dir, command, 0)
			}
			if got := detectPackageUpdateCommand(); got != tt.want {
				t.Fatalf("detectPackageUpdateCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateCommandPrefersBuildSetting(t *testing.T) {
	previous := packageUpdateCommand
	packageUpdateCommand = "apt upgrade"
	t.Cleanup(func() { packageUpdateCommand = previous })

	if got := UpdateCommand(); got != "apt upgrade" {
		t.Fatalf("UpdateCommand() = %q, want %q", got, "apt upgrade")
	}
}
