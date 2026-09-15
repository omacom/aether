package omarchy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActivateThemeUsesPublicCLIAndExplicitBackground(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "commands")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PATH", binDir)
	t.Setenv("AETHER_COMMAND_LOG", logPath)
	managedTheme := filepath.Join(home, ".config", "omarchy", "themes", "aether")
	if err := os.MkdirAll(managedTheme, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := MarkManagedTheme(managedTheme); err != nil {
		t.Fatal(err)
	}

	writeThemeName(t, filepath.Join(home, ".local", "state", "omarchy", "current", "theme.name"), "ethereal\n")
	script := "#!/bin/sh\nprintf '%s|%s\\n' \"$OMARCHY_THEME_SKIP_BACKGROUND\" \"$*\" >> \"$AETHER_COMMAND_LOG\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	wallpaper := filepath.Join(home, "wallpaper.png")
	if err := ActivateTheme("aether", wallpaper); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{
		"|theme bg set " + wallpaper,
		"1|theme set aether",
	}
	if len(lines) != len(want) {
		t.Fatalf("commands = %q, want %q", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("command %d = %q, want %q", i, lines[i], want[i])
		}
	}

	previous, err := os.ReadFile(filepath.Join(home, ".config", "aether", "previous-omarchy-theme"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(previous)); got != "ethereal" {
		t.Errorf("previous theme = %q, want ethereal", got)
	}
}

func TestActivateForeignThemeDoesNotReplaceRevertTarget(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PATH", binDir)

	writeThemeName(t, filepath.Join(home, ".local", "state", "omarchy", "current", "theme.name"), "ethereal\n")
	if err := os.MkdirAll(filepath.Join(home, ".config", "omarchy", "themes", "foreign"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := ActivateTheme("foreign", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(previousThemeFile()); !os.IsNotExist(err) {
		t.Fatalf("foreign activation wrote previous theme: %v", err)
	}
}

func TestRevertThemeDoesNothingAfterExternalThemeChange(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "commands")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PATH", binDir)
	t.Setenv("AETHER_COMMAND_LOG", logPath)

	writeThemeName(t, filepath.Join(home, ".local", "state", "omarchy", "current", "theme.name"), "foreign\n")
	writeThemeName(t, previousThemeFile(), "ethereal\n")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >> \"$AETHER_COMMAND_LOG\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RevertTheme(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("RevertTheme() invoked Omarchy after external change: %v", err)
	}
}

func TestRevertThemeFallsBackForLegacyManagedTheme(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "commands")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PATH", binDir)
	t.Setenv("AETHER_COMMAND_LOG", logPath)

	managedTheme := filepath.Join(home, ".config", "omarchy", "themes", "aether")
	if err := os.MkdirAll(managedTheme, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := MarkManagedTheme(managedTheme); err != nil {
		t.Fatal(err)
	}
	writeThemeName(t, filepath.Join(home, ".local", "state", "omarchy", "current", "theme.name"), "aether\n")
	script := "#!/bin/sh\nprintf '%s\n' \"$*\" >> \"$AETHER_COMMAND_LOG\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RevertTheme(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != "theme set tokyo-night" {
		t.Fatalf("command = %q, want theme set tokyo-night", got)
	}
}

func TestActivateThemeReturnsCommandOutput(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PATH", binDir)
	script := "#!/bin/sh\necho 'theme transaction failed' >&2\nexit 7\n"
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	err := ActivateTheme("aether", "")
	if err == nil || !strings.Contains(err.Error(), "theme transaction failed") {
		t.Fatalf("ActivateTheme() error = %v, want command stderr", err)
	}
}

func TestActivateThemeDoesNotChangeThemeWhenBackgroundFails(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "commands")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PATH", binDir)
	t.Setenv("AETHER_COMMAND_LOG", logPath)

	managedTheme := filepath.Join(home, ".config", "omarchy", "themes", "aether")
	if err := os.MkdirAll(managedTheme, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := MarkManagedTheme(managedTheme); err != nil {
		t.Fatal(err)
	}
	writeThemeName(t, filepath.Join(home, ".local", "state", "omarchy", "current", "theme.name"), "ethereal\n")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$AETHER_COMMAND_LOG"
if [ "$1 $2 $3" = "theme bg set" ]; then
    echo "background failed" >&2
    exit 7
fi
`
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	err := ActivateTheme("aether", filepath.Join(home, "wallpaper.png"))
	if err == nil || !strings.Contains(err.Error(), "background failed") {
		t.Fatalf("ActivateTheme() error = %v, want background error", err)
	}
	data, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := "theme bg set " + filepath.Join(home, "wallpaper.png")
	if got := strings.TrimSpace(string(data)); got != want {
		t.Fatalf("commands = %q, want %q", got, want)
	}
}

func TestActivateThemeRestoresBackgroundWhenThemeSetFails(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "commands")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("PATH", binDir)
	t.Setenv("AETHER_COMMAND_LOG", logPath)

	managedTheme := filepath.Join(home, ".config", "omarchy", "themes", "aether")
	if err := os.MkdirAll(managedTheme, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := MarkManagedTheme(managedTheme); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(home, ".local", "state", "omarchy", "current")
	writeThemeName(t, filepath.Join(stateDir, "theme.name"), "aether\n")
	oldBackground := filepath.Join(home, "old.png")
	if err := os.WriteFile(oldBackground, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldBackground, filepath.Join(stateDir, "background")); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$AETHER_COMMAND_LOG"
if [ "$1 $2" = "theme set" ]; then
    echo "theme failed" >&2
    exit 7
fi
`
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	newBackground := filepath.Join(home, "new.png")
	err := ActivateTheme("aether", newBackground)
	if err == nil || !strings.Contains(err.Error(), "theme failed") {
		t.Fatalf("ActivateTheme() error = %v, want theme activation error", err)
	}
	data, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	want := "theme bg set " + newBackground + "\ntheme set aether\ntheme bg set " + oldBackground
	if got := strings.TrimSpace(string(data)); got != want {
		t.Fatalf("commands = %q, want %q", got, want)
	}
}

func TestActivateThemeWithInstallDoesNotActivateOnInstallFailure(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	logPath := filepath.Join(home, "commands")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("PATH", binDir)
	t.Setenv("AETHER_COMMAND_LOG", logPath)
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$AETHER_COMMAND_LOG\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	writeThemeName(t, filepath.Join(CurrentStateDir(), "theme.name"), "foreign\n")
	err := ActivateThemeWithInstall("aether", "", func(activate func() error) error {
		return os.ErrPermission
	})
	if !os.IsPermission(err) {
		t.Fatalf("ActivateThemeWithInstall() error = %v, want install error", err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("activation ran after failed install: %v", err)
	}
	if _, err := os.Stat(previousThemeFile()); !os.IsNotExist(err) {
		t.Fatalf("failed install changed previous theme record: %v", err)
	}
}

func TestActivateThemeReportsBackgroundRestoreFailure(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("PATH", binDir)
	oldBackground := filepath.Join(home, "old.png")
	if err := os.WriteFile(oldBackground, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeThemeName(t, filepath.Join(CurrentStateDir(), "theme.name"), "foreign\n")
	if err := os.Symlink(oldBackground, filepath.Join(CurrentStateDir(), "background")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_OLD_BACKGROUND", oldBackground)
	script := `#!/bin/sh
if [ "$1 $2" = "theme set" ]; then
    printf 'activation failed\n' >&2
    exit 7
fi
if [ "$4" = "$AETHER_OLD_BACKGROUND" ]; then
    printf 'background restore failed\n' >&2
    exit 8
fi
`
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	err := ActivateTheme("aether", filepath.Join(home, "new.png"))
	if err == nil || !strings.Contains(err.Error(), "activation failed") || !strings.Contains(err.Error(), "restore previous background: background restore failed") {
		t.Fatalf("ActivateTheme() error = %v, want activation and rollback errors", err)
	}
	snapshots, globErr := filepath.Glob(filepath.Join(home, ".local", "share", "aether", "background-recovery", "*", "old.png"))
	if globErr != nil || len(snapshots) != 1 || !strings.Contains(err.Error(), snapshots[0]) {
		t.Fatalf("rollback error does not identify a retained snapshot: %v, %v, %v", err, snapshots, globErr)
	}
	if data, err := os.ReadFile(snapshots[0]); err != nil || string(data) != "old" {
		t.Fatalf("recovery copy = %q, %v; want old wallpaper", data, err)
	}
}

func TestActivateThemeDoesNotInstallWhenSnapshotFails(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "blocked"))
	t.Setenv("PATH", binDir)
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte("#!/bin/sh\nexit 9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeThemeName(t, filepath.Join(CurrentStateDir(), "theme.name"), "foreign\n")
	oldBackground := filepath.Join(home, "old.png")
	if err := os.WriteFile(oldBackground, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldBackground, filepath.Join(CurrentStateDir(), "background")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "blocked"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	installed := false
	err := ActivateThemeWithInstall("aether", "", func(activate func() error) error {
		installed = true
		return activate()
	})
	if installed || err == nil || !strings.Contains(err.Error(), "create background recovery directory") {
		t.Fatalf("ActivateThemeWithInstall() installed = %v, error = %v; want snapshot failure before installation", installed, err)
	}
	if selected := currentBackgroundPath(); selected != oldBackground {
		t.Fatalf("background changed after failed snapshot: %s", selected)
	}
}

func TestActivateThemeCleansUnusedSnapshots(t *testing.T) {
	for _, installFails := range []bool{false, true} {
		name := "success"
		if installFails {
			name = "install failure"
		}
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			binDir := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
			t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
			t.Setenv("PATH", binDir)
			if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte("#!/bin/sh\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			oldBackground := filepath.Join(CurrentStateDir(), "theme", "backgrounds", "old.png")
			writeThemeName(t, oldBackground, "old wallpaper")
			if err := os.Symlink(oldBackground, filepath.Join(CurrentStateDir(), "background")); err != nil {
				t.Fatal(err)
			}
			err := ActivateThemeWithInstall("aether", "", func(activate func() error) error {
				if installFails {
					return os.ErrPermission
				}
				return activate()
			})
			if installFails && !os.IsPermission(err) || !installFails && err != nil {
				t.Fatalf("unexpected activation error: %v", err)
			}
			entries, err := os.ReadDir(filepath.Join(home, "data", "aether", "background-recovery"))
			if err != nil || len(entries) != 0 {
				t.Fatalf("unused snapshots were not cleaned up: %v, %v", entries, err)
			}
			if data, err := os.ReadFile(oldBackground); err != nil || string(data) != "old wallpaper" {
				t.Fatalf("snapshot cleanup changed the original background: %q, %v", data, err)
			}
		})
	}
}

func TestActivateThemeDoesNotReuseChangedBackgroundPath(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("PATH", binDir)
	oldBackground := filepath.Join(home, "old.png")
	writeThemeName(t, oldBackground, "old wallpaper\n")
	writeThemeName(t, filepath.Join(CurrentStateDir(), "theme.name"), "foreign\n")
	backgroundLink := filepath.Join(CurrentStateDir(), "background")
	if err := os.Symlink(oldBackground, backgroundLink); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_OLD_BACKGROUND", oldBackground)
	t.Setenv("AETHER_BACKGROUND_LINK", backgroundLink)
	t.Setenv("AETHER_FAIL_ACTIVATION", "1")
	script := `#!/bin/sh
if [ "$1 $2" = "theme set" ] && [ "$AETHER_FAIL_ACTIVATION" = "1" ]; then
    printf 'changed wallpaper\n' > "$AETHER_OLD_BACKGROUND"
    printf 'activation failed\n' >&2
    exit 7
fi
if [ "$1 $2 $3" = "theme bg set" ]; then
    [ -f "$4" ] || exit 8
    /bin/ln -sfn "$4" "$AETHER_BACKGROUND_LINK" || exit 9
fi
`
	if err := os.WriteFile(filepath.Join(binDir, "omarchy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	err := ActivateTheme("aether", "")
	if err == nil || !strings.Contains(err.Error(), "activation failed") || strings.Contains(err.Error(), "restore previous background") {
		t.Fatalf("ActivateTheme() error = %v, want activation failure with successful background restoration", err)
	}
	selected := currentBackgroundPath()
	if selected == oldBackground || selected == "" {
		t.Fatalf("reused a changed original background: %q", selected)
	}
	if data, err := os.ReadFile(selected); err != nil || string(data) != "old wallpaper\n" {
		t.Fatalf("restored background = %q, %v; want original bytes", data, err)
	}
	if data, err := os.ReadFile(oldBackground); err != nil || string(data) != "changed wallpaper\n" {
		t.Fatalf("rollback overwrote the changed original: %q, %v", data, err)
	}

	// A later transaction must clean only its own snapshot, not the retained
	// recovery file that can still be referenced by Omarchy or another consumer.
	t.Setenv("AETHER_FAIL_ACTIVATION", "0")
	if err := ActivateTheme("aether", oldBackground); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(selected); err != nil || string(data) != "old wallpaper\n" {
		t.Fatalf("later cleanup removed the retained recovery copy: %q, %v", data, err)
	}
	entries, err := os.ReadDir(filepath.Join(home, "data", "aether", "background-recovery"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected only the retained recovery snapshot: %v, %v", entries, err)
	}
}
