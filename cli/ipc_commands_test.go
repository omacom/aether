package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/ipc"
)

func TestRunIPCWallpaperPaths(t *testing.T) {
	home, requests := startIPCTestServer(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for _, cmd := range []string{"extract", "set-wallpaper"} {
		for _, tt := range []struct {
			name, path, want string
		}{
			{"relative", "wallpaper.jpg", filepath.Join(cwd, "wallpaper.jpg")},
			{"dot segments", "./wallpapers/../forest view.jpg", filepath.Join(cwd, "forest view.jpg")},
			{"parent", "../wallpaper.jpg", filepath.Join(cwd, "..", "wallpaper.jpg")},
			{"home", "~/wallpapers/forest view.jpg", filepath.Join(home, "wallpapers", "forest view.jpg")},
			{"absolute", filepath.Join(home, "wallpaper.jpg"), filepath.Join(home, "wallpaper.jpg")},
		} {
			t.Run(cmd+"/"+tt.name, func(t *testing.T) {
				if code := RunIPC([]string{cmd, tt.path}); code != 0 {
					t.Fatalf("RunIPC() = %d, want 0", code)
				}
				select {
				case req := <-requests:
					if req.Cmd != cmd || req.Path != tt.want {
						t.Errorf("request = %+v, want command %q and path %q", req, cmd, tt.want)
					}
					if req.Mode != "" || req.LightMode != nil {
						t.Errorf("unspecified extraction options were set: %+v", req)
					}
				default:
					t.Fatal("no IPC request received")
				}
			})
		}
	}
}

func TestRunIPCAllowedModes(t *testing.T) {
	_, requests := startIPCTestServer(t)
	for mode := range validModes {
		for _, cmd := range []string{"extract", "set-mode"} {
			t.Run(cmd+"/"+mode, func(t *testing.T) {
				args := []string{cmd, mode}
				if cmd == "extract" {
					args = []string{cmd, "wallpaper.jpg", "--mode", mode, "--light-mode"}
				}
				if code := RunIPC(args); code != 0 {
					t.Fatalf("RunIPC(%q) = %d, want 0", args, code)
				}
				select {
				case req := <-requests:
					if req.Cmd != cmd || req.Mode != mode {
						t.Errorf("request = %+v, want command %q and mode %q", req, cmd, mode)
					}
					if cmd == "extract" && (req.LightMode == nil || !*req.LightMode) {
						t.Errorf("--light-mode not preserved: %+v", req)
					}
				default:
					t.Fatal("no IPC request received")
				}
			})
		}
	}
}

func TestRunIPCRejectsInvalidArgsBeforeSend(t *testing.T) {
	_, requests := startIPCTestServer(t)
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"no command", nil, "Usage: aether <command> [--json]"},
		{"extract mode", []string{"extract", "wallpaper.jpg", "--mode", "invalid"}, "Invalid extraction mode: invalid"},
		{"extract empty mode", []string{"extract", "wallpaper.jpg", "--mode", ""}, "Invalid extraction mode: "},
		{"extract missing mode", []string{"extract", "wallpaper.jpg", "--mode"}, "Invalid extraction mode: "},
		{"extract flag as mode", []string{"extract", "wallpaper.jpg", "--mode", "--light-mode"}, "Invalid extraction mode: --light-mode"},
		{"set-mode", []string{"set-mode", "invalid"}, "Invalid extraction mode: invalid"},
		{"set-mode case", []string{"set-mode", "Pastel"}, "Invalid extraction mode: Pastel"},
		{"set-mode empty", []string{"set-mode", ""}, "Invalid extraction mode: "},
	} {
		for _, jsonOut := range []bool{false, true} {
			name := tt.name
			args := append([]string(nil), tt.args...)
			if jsonOut {
				name += "/json"
				args = append(args, "--json")
			}
			t.Run(name, func(t *testing.T) {
				code, stdout, stderr := captureIPCOutput(t, args)
				if code != 1 {
					t.Errorf("RunIPC(%q) = %d, want 1", args, code)
				}
				select {
				case req := <-requests:
					t.Errorf("invalid arguments sent an IPC request: %+v", req)
				default:
				}
				if jsonOut {
					var result map[string]string
					if err := json.Unmarshal([]byte(stdout), &result); err != nil {
						t.Fatalf("invalid JSON output %q: %v", stdout, err)
					}
					if result["error"] != tt.want || stderr != "" {
						t.Errorf("stdout = %q, stderr = %q; want JSON error %q only", stdout, stderr, tt.want)
					}
				} else if stdout != "" || stderr != tt.want+"\n" {
					t.Errorf("stdout = %q, stderr = %q; want stderr %q only", stdout, stderr, tt.want)
				}
			})
		}
	}
}

type ipcTestDispatcher chan ipc.Request

func (requests ipcTestDispatcher) HandleIPC(req ipc.Request) ipc.Response {
	requests <- req
	return ipc.Response{OK: true}
}

func startIPCTestServer(t *testing.T) (string, <-chan ipc.Request) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	socketPath := ipc.DefaultSocketPath()
	if !strings.HasPrefix(socketPath, home+string(os.PathSeparator)) {
		t.Fatalf("refusing to use socket outside temporary HOME: %s", socketPath)
	}
	requests := make(ipcTestDispatcher, 1)
	server, err := ipc.NewServer(socketPath, requests)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
	})
	return home, requests
}

func captureIPCOutput(t *testing.T, args []string) (int, string, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	code, stderr := captureGenerateStderr(t, func() int { return RunIPC(args) })
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	stdout, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return code, string(stdout), stderr
}
