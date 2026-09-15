package template

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aether/internal/platform"
)

func TestProcessCustomApps_NoDir(t *testing.T) {
	themeDir := t.TempDir()
	// Point to a non-existent custom dir — should return nil, not error.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "nonexistent"))

	err := ProcessCustomApps(themeDir, map[string]string{"background": "#1e1e2e"})
	if err != nil {
		t.Fatalf("expected nil error for missing custom dir, got: %v", err)
	}
}

func TestProcessCustomApps_EmptyDir(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	themeDir := t.TempDir()
	err := ProcessCustomApps(themeDir, map[string]string{"background": "#1e1e2e"})
	if err != nil {
		t.Fatalf("expected nil error for empty custom dir, got: %v", err)
	}
}

func TestProcessCustomApps_SkipsFiles(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a regular file (not a directory) — should be skipped.
	if err := os.WriteFile(filepath.Join(customDir, "stray-file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	themeDir := t.TempDir()
	err := ProcessCustomApps(themeDir, map[string]string{})
	if err != nil {
		t.Fatalf("expected nil error when skipping non-dir entries, got: %v", err)
	}
}

func TestProcessCustomApps_FullWorkflow(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	appDir := filepath.Join(customDir, "testapp")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write config.json
	config := customAppConfig{
		Template:    "theme.conf",
		Destination: filepath.Join(t.TempDir(), "dest", "testapp-theme"),
	}
	configData, _ := json.Marshal(config)
	if err := os.WriteFile(filepath.Join(appDir, "config.json"), configData, 0644); err != nil {
		t.Fatal(err)
	}

	// Write template file
	templateContent := `bg = {background}
fg = {foreground}
accent_strip = {accent.strip}
rgb = {red.rgb}
rgba = {blue.rgba:0.7}
`
	if err := os.WriteFile(filepath.Join(appDir, "theme.conf"), []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	themeDir := t.TempDir()
	variables := map[string]string{
		"background": "#1e1e2e",
		"foreground": "#cdd6f4",
		"accent":     "#89b4fa",
		"red":        "#f38ba8",
		"blue":       "#89b4fa",
	}

	err := ProcessCustomApps(themeDir, variables)
	if err != nil {
		t.Fatalf("ProcessCustomApps failed: %v", err)
	}

	// Verify the processed file was written to themeDir
	outputPath := filepath.Join(themeDir, "testapp-theme.conf")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected output file at %s: %v", outputPath, err)
	}

	output := string(data)

	// Check substitutions
	assertContains(t, output, "bg = #1e1e2e")
	assertContains(t, output, "fg = #cdd6f4")
	assertContains(t, output, "accent_strip = 89b4fa")
	assertContains(t, output, "rgb = 243,139,168")
	assertContains(t, output, "rgba = rgba(137, 180, 250, 0.7)")

	// Verify symlink was created
	linkTarget, err := os.Readlink(config.Destination)
	if err != nil {
		t.Fatalf("expected symlink at %s: %v", config.Destination, err)
	}
	if linkTarget != outputPath {
		t.Errorf("symlink target = %s, want %s", linkTarget, outputPath)
	}
	before, err := os.Lstat(config.Destination)
	if err != nil {
		t.Fatal(err)
	}
	variables["background"] = "#000000"
	if err := ProcessCustomApps(themeDir, variables); err != nil {
		t.Fatalf("reapplying an owned symlink failed: %v", err)
	}
	after, err := os.Lstat(config.Destination)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("owned symlink was replaced: %v", err)
	}
	updated, err := os.ReadFile(config.Destination)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(updated), "bg = #000000")
}

func TestProcessCustomApps_MissingConfigJSON(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	appDir := filepath.Join(customDir, "noconfigapp")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	// No config.json — should log and skip, not error.
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	themeDir := t.TempDir()
	err := ProcessCustomApps(themeDir, map[string]string{})
	if err != nil {
		t.Fatalf("expected nil error for missing config.json, got: %v", err)
	}
}

func TestProcessCustomApps_EmptyTemplateField(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	appDir := filepath.Join(customDir, "emptytemplate")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := customAppConfig{Template: "", Destination: ""}
	configData, _ := json.Marshal(config)
	if err := os.WriteFile(filepath.Join(appDir, "config.json"), configData, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	themeDir := t.TempDir()
	err := ProcessCustomApps(themeDir, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "missing 'template'") {
		t.Fatalf("expected error for empty template field, got: %v", err)
	}
}

func TestProcessCustomApps_MissingTemplateFile(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	appDir := filepath.Join(customDir, "missingfile")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := customAppConfig{Template: "nonexistent.conf", Destination: ""}
	configData, _ := json.Marshal(config)
	if err := os.WriteFile(filepath.Join(appDir, "config.json"), configData, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	themeDir := t.TempDir()
	err := ProcessCustomApps(themeDir, map[string]string{})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected error for missing template file, got: %v", err)
	}
}

func TestProcessCustomApps_InstallFailuresSkipHooks(t *testing.T) {
	for _, existing := range []string{"regular", "foreign symlink", "dangling symlink", "directory", "blocked parent", "blocked output"} {
		t.Run(existing, func(t *testing.T) {
			customDir, themeDir, hookLog := setupCustomAppsTest(t)
			dir := t.TempDir()
			destination := filepath.Join(dir, "destination")
			foreign := filepath.Join(dir, "foreign")
			if err := os.WriteFile(foreign, []byte("user data"), 0600); err != nil {
				t.Fatal(err)
			}
			linkTarget := ""
			switch existing {
			case "regular":
				if err := os.WriteFile(destination, []byte("user config"), 0600); err != nil {
					t.Fatal(err)
				}
			case "foreign symlink":
				linkTarget = foreign
			case "dangling symlink":
				linkTarget = filepath.Join(dir, "missing")
			case "directory":
				if err := os.Mkdir(destination, 0755); err != nil {
					t.Fatal(err)
				}
			case "blocked parent":
				destination = filepath.Join(foreign, "destination")
			case "blocked output":
				if err := os.Mkdir(filepath.Join(themeDir, "testapp-theme.conf"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if linkTarget != "" {
				if err := os.Symlink(linkTarget, destination); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.Lstat(destination)
			writeCustomApp(t, customDir, "testapp", destination, true)
			err := ProcessCustomApps(themeDir, map[string]string{"background": "#123456"})
			if err == nil || !strings.Contains(err.Error(), "testapp") {
				t.Fatalf("expected a contextual install error, got %v", err)
			}
			if before != nil {
				after, err := os.Lstat(destination)
				if err != nil || !os.SameFile(before, after) {
					t.Fatalf("destination was replaced: %v", err)
				}
			}
			if linkTarget != "" {
				if got, err := os.Readlink(destination); err != nil || got != linkTarget {
					t.Errorf("foreign symlink changed: %q, %v", got, err)
				}
			}
			if existing == "regular" {
				data, err := os.ReadFile(destination)
				if err != nil || string(data) != "user config" {
					t.Errorf("user config changed: %q, %v", data, err)
				}
			}
			data, err := os.ReadFile(foreign)
			if err != nil || string(data) != "user data" {
				t.Errorf("foreign target changed: %q, %v", data, err)
			}
			// Hooks start asynchronously; give an incorrectly launched fake hook time
			// to leave its marker before checking that none ran.
			time.Sleep(100 * time.Millisecond)
			if _, err := os.Stat(hookLog); !os.IsNotExist(err) {
				t.Fatalf("hook ran after failed installation: %v", err)
			}
		})
	}
}

func TestProcessCustomApps_ReturnsAllErrors(t *testing.T) {
	customDir, themeDir, _ := setupCustomAppsTest(t)
	invalid := writeCustomApp(t, customDir, "a-invalid", "", false)
	if err := os.WriteFile(filepath.Join(invalid, "config.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "user.conf")
	if err := os.WriteFile(destination, []byte("user config"), 0600); err != nil {
		t.Fatal(err)
	}
	writeCustomApp(t, customDir, "b-conflict", destination, false)
	writeCustomApp(t, customDir, "c-valid", "", false)

	err := ProcessCustomApps(themeDir, map[string]string{"background": "#123456"})
	var syntaxError *json.SyntaxError
	if !errors.As(err, &syntaxError) || !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected both parse and installation errors, got %v", err)
	}
	assertContains(t, err.Error(), "a-invalid")
	assertContains(t, err.Error(), "b-conflict")
	data, err := os.ReadFile(filepath.Join(themeDir, "c-valid-theme.conf"))
	if err != nil || string(data) != "bg=#123456\n" {
		t.Fatalf("valid app was not processed after failures: %q, %v", data, err)
	}
}

func TestProcessCustomApps_HookAfterInstall(t *testing.T) {
	customDir, themeDir, hookLog := setupCustomAppsTest(t)
	destination := filepath.Join(t.TempDir(), "destination")
	appDir := writeCustomApp(t, customDir, "testapp", destination, true)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relativeThemeDir, err := filepath.Rel(cwd, themeDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessCustomApps(relativeThemeDir, map[string]string{"background": "#123456"}); err != nil {
		t.Fatal(err)
	}
	if got, err := os.Readlink(destination); err != nil || got != filepath.Join(themeDir, "testapp-theme.conf") {
		t.Fatalf("relative theme directory produced an incorrect symlink: %q, %v", got, err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "bg=#123456\n" {
		t.Fatalf("destination contents = %q, %v", data, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(hookLog)
		if err == nil && string(data) == filepath.Join(appDir, "post-apply.sh")+"\n" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("fake hook did not run: %q, %v", data, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestProcessCustomApps_ReturnsHookStartFailure(t *testing.T) {
	customDir, themeDir, _ := setupCustomAppsTest(t)
	t.Setenv("PATH", t.TempDir())
	writeCustomApp(t, customDir, "testapp", "", true)
	err := ProcessCustomApps(themeDir, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "start post-apply.sh") {
		t.Fatalf("expected hook start failure, got %v", err)
	}
}

func setupCustomAppsTest(t *testing.T) (customDir, themeDir, hookLog string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	binDir := t.TempDir()
	hookLog = filepath.Join(home, "hooks.log")
	t.Setenv("AETHER_TEST_HOOK_LOG", hookLog)
	t.Setenv("PATH", binDir)
	// The fake interpreter only records invocations; it never runs an app hook.
	if err := os.WriteFile(filepath.Join(binDir, "bash"), []byte("#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"$AETHER_TEST_HOOK_LOG\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return platform.CustomDir(), t.TempDir(), hookLog
}

func writeCustomApp(t *testing.T, customDir, name, destination string, hook bool) string {
	t.Helper()
	appDir := filepath.Join(customDir, name)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(customAppConfig{Template: "theme.conf", Destination: destination})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"config.json": string(data), "theme.conf": "bg={background}\n"}
	if hook {
		files["post-apply.sh"] = "# fake hook\n"
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(appDir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return appDir
}

func TestProcessCustomApps_NoDestination(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	appDir := filepath.Join(customDir, "nodest")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	config := customAppConfig{Template: "theme.conf", Destination: ""}
	configData, _ := json.Marshal(config)
	if err := os.WriteFile(filepath.Join(appDir, "config.json"), configData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "theme.conf"), []byte("bg={background}"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	themeDir := t.TempDir()
	err := ProcessCustomApps(themeDir, map[string]string{"background": "#000000"})
	if err != nil {
		t.Fatalf("expected nil error with no destination, got: %v", err)
	}

	// Output file should still exist
	outputPath := filepath.Join(themeDir, "nodest-theme.conf")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if string(data) != "bg=#000000" {
		t.Errorf("output = %q, want %q", string(data), "bg=#000000")
	}
}

func TestProcessCustomApps_MultipleApps(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	// Create two custom apps
	for _, appName := range []string{"app1", "app2"} {
		appDir := filepath.Join(customDir, appName)
		if err := os.MkdirAll(appDir, 0755); err != nil {
			t.Fatal(err)
		}
		config := customAppConfig{Template: "t.conf"}
		configData, _ := json.Marshal(config)
		os.WriteFile(filepath.Join(appDir, "config.json"), configData, 0644)
		os.WriteFile(filepath.Join(appDir, "t.conf"), []byte("{foreground}"), 0644)
	}

	themeDir := t.TempDir()
	err := ProcessCustomApps(themeDir, map[string]string{"foreground": "#ffffff"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both outputs should exist
	for _, name := range []string{"app1-t.conf", "app2-t.conf"} {
		data, err := os.ReadFile(filepath.Join(themeDir, name))
		if err != nil {
			t.Errorf("missing output %s: %v", name, err)
			continue
		}
		if string(data) != "#ffffff" {
			t.Errorf("%s content = %q, want %q", name, string(data), "#ffffff")
		}
	}
}

func TestReadCustomOverride_Found(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Place a custom override file
	overrideContent := "custom bg={background}\ncustom fg={foreground}"
	if err := os.WriteFile(filepath.Join(customDir, "kitty.conf"), []byte(overrideContent), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	content, found := ReadCustomOverride("kitty.conf")
	if !found {
		t.Fatal("expected override to be found")
	}
	if content != overrideContent {
		t.Errorf("content = %q, want %q", content, overrideContent)
	}
}

func TestReadCustomOverride_NotFound(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	_, found := ReadCustomOverride("nonexistent.conf")
	if found {
		t.Fatal("expected override NOT to be found")
	}
}

func TestReadCustomOverride_IgnoresDirectories(t *testing.T) {
	xdgConfig := t.TempDir()
	customDir := filepath.Join(xdgConfig, "aether", "custom")
	// Create a directory with the same name as a template file
	if err := os.MkdirAll(filepath.Join(customDir, "kitty.conf"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	_, found := ReadCustomOverride("kitty.conf")
	if found {
		t.Fatal("expected directory to NOT be treated as an override")
	}
}

func TestReadCustomOverride_CustomDirMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "nonexistent"))

	_, found := ReadCustomOverride("kitty.conf")
	if found {
		t.Fatal("expected false when custom dir doesn't exist")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"tilde prefix", "~/foo/bar", filepath.Join(home, "foo/bar")},
		{"absolute path", "/usr/local/bin", "/usr/local/bin"},
		{"relative path", "some/relative", "some/relative"},
		{"tilde only", "~/", home},
		{"tilde no slash", "~nope", "~nope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHome(tt.in)
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}
