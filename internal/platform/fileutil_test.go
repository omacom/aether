package platform

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
)

func TestAtomicFileWrites(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(src, []byte("new contents"), 0750); err != nil {
		t.Fatal(err)
	}
	sourceInfo, err := os.Stat(src)
	if err != nil {
		t.Fatal(err)
	}

	writers := []struct {
		name  string
		write func(string) error
		want  string
		mode  os.FileMode
	}{
		{"JSON", func(path string) error { return WriteJSON(path, map[string]int{"value": 1}) }, "{\n  \"value\": 1\n}\n", 0644},
		{"text", func(path string) error { return WriteText(path, "new contents") }, "new contents", 0644},
		{"copy", func(path string) error { return CopyFile(src, path) }, "new contents", sourceInfo.Mode().Perm()},
	}
	for _, writer := range writers {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing=%t", writer.name, existing), func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "nested", "output")
				var old *os.File
				if existing {
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte("old contents"), 0600); err != nil {
						t.Fatal(err)
					}
					old, err = os.Open(path)
					if err != nil {
						t.Fatal(err)
					}
					defer old.Close()
				}

				if err := writer.write(path); err != nil {
					t.Fatal(err)
				}
				assertFileContent(t, path, writer.want)
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if existing {
					if info.Mode().Perm() != 0600 {
						t.Errorf("permissions = %o, want 600", info.Mode().Perm())
					}
					data, err := io.ReadAll(old)
					if err != nil || string(data) != "old contents" {
						t.Errorf("previous inode was modified: %q, %v", data, err)
					}
				} else if got := info.Mode().Perm(); got&^writer.mode != 0 || got&0600 != 0600 {
					t.Errorf("new permissions = %o, want %o subject to umask", got, writer.mode)
				}
				assertNoAtomicTemps(t, filepath.Dir(path))
			})
		}
	}
}

func TestAtomicWritesPreserveDestinationSymlink(t *testing.T) {
	for _, operation := range []string{"JSON", "text", "copy"} {
		t.Run(operation, func(t *testing.T) {
			dir := t.TempDir()
			targetDir := t.TempDir()
			target := filepath.Join(targetDir, "output")
			if err := os.WriteFile(target, []byte("old contents"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(targetDir, filepath.Join(dir, "target-dir")); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(dir, "link")
			linkTarget := filepath.Join("target-dir", "output")
			if err := os.Symlink(linkTarget, link); err != nil {
				t.Fatal(err)
			}
			var err error
			want := "new contents"
			switch operation {
			case "JSON":
				err = WriteJSON(link, "new contents")
				want = "\"new contents\"\n"
			case "text":
				err = WriteText(link, want)
			case "copy":
				src := filepath.Join(dir, "source")
				if err := os.WriteFile(src, []byte(want), 0644); err != nil {
					t.Fatal(err)
				}
				err = CopyFile(src, link)
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.Readlink(link)
			if err != nil || got != linkTarget {
				t.Fatalf("destination symlink changed: %q, %v", got, err)
			}
			assertFileContent(t, target, want)
			info, err := os.Stat(target)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("target permissions changed: %v, %v", info, err)
			}
			assertNoAtomicTemps(t, dir)
			assertNoAtomicTemps(t, targetDir)
		})
	}
}

func TestAtomicWritePermissionsRespectUmask(t *testing.T) {
	if os.Getenv("AETHER_TEST_UMASK") != "1" {
		// Umask is process-wide, so change it only in a dedicated subprocess.
		shell, err := exec.LookPath("sh")
		if err != nil {
			t.Skip("requires a POSIX shell")
		}
		cmd := exec.Command(shell, "-c", `umask 027; exec "$1" -test.run '^TestAtomicWritePermissionsRespectUmask$'`, "sh", os.Args[0])
		cmd.Env = append(os.Environ(), "AETHER_TEST_UMASK=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("umask test failed: %v\n%s", err, output)
		}
		return
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	if err := os.WriteFile(src, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(src, 0755); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name  string
		write func(string) error
		mode  os.FileMode
	}{
		{"text", func(path string) error { return WriteText(path, "contents") }, 0640},
		{"JSON", func(path string) error { return WriteJSON(path, "contents") }, 0640},
		{"copy", func(path string) error { return CopyFile(src, path) }, 0750},
	} {
		path := filepath.Join(dir, tt.name)
		if err := tt.write(path); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != tt.mode {
			t.Fatalf("new %s mode = %v, %v; want %o", tt.name, info, err, tt.mode)
		}
		for _, mode := range []os.FileMode{0600, 0644} {
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			if err := tt.write(path); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != mode {
				t.Fatalf("existing %s mode = %v, %v; want %o", tt.name, info, err, mode)
			}
		}
	}
}

func TestWriteFileAtomicReadFailure(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing=%t", existing), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "output")
			if existing {
				if err := os.WriteFile(path, []byte("valid contents"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			readErr := errors.New("source read failed")
			reader := io.MultiReader(strings.NewReader("partial contents"), iotest.ErrReader(readErr))
			if err := writeFileAtomic(path, reader, 0644); !errors.Is(err, readErr) {
				t.Fatalf("write error = %v, want source error", err)
			}
			if existing {
				assertFileContent(t, path, "valid contents")
			} else if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatalf("failed write published a destination: %v", err)
			}
			assertNoAtomicTemps(t, dir)
		})
	}
}

func TestWriteFileAtomicReportsCleanupFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output")
	if err := os.WriteFile(path, []byte("valid contents"), 0600); err != nil {
		t.Fatal(err)
	}
	readErr := errors.New("source read failed")
	reader := fileReaderFunc(func(p []byte) (int, error) {
		paths, err := filepath.Glob(filepath.Join(dir, ".aether-*.tmp"))
		if err != nil || len(paths) != 1 {
			t.Fatalf("expected one same-directory temporary file: %v, %v", paths, err)
		}
		// Replace the temporary pathname with a non-empty directory to force an
		// unlink failure without relying on permissions or running as non-root.
		if err := os.Remove(paths[0]); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(paths[0], 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(paths[0], "keep"), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		return copy(p, "partial contents"), readErr
	})
	err := writeFileAtomic(path, reader, 0644)
	if !errors.Is(err, readErr) || !strings.Contains(err.Error(), "remove temporary file") {
		t.Fatalf("expected both read and cleanup errors, got %v", err)
	}
	assertFileContent(t, path, "valid contents")
}

func TestWriteFileAtomicRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output")
	reader := fileReaderFunc(func(p []byte) (int, error) {
		// A competing writer installs a directory after staging starts.
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "keep"), []byte("user data"), 0600); err != nil {
			t.Fatal(err)
		}
		return copy(p, "new contents"), io.EOF
	})
	if err := writeFileAtomic(path, reader, 0644); err == nil {
		t.Fatal("expected rename failure")
	}
	assertFileContent(t, filepath.Join(path, "keep"), "user data")
	assertNoAtomicTemps(t, dir)
}

func TestAtomicWriteFailuresPreserveDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	if err := os.WriteFile(path, []byte("valid contents"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSON(path, make(chan int)); err == nil {
		t.Fatal("expected JSON marshal failure")
	}
	assertFileContent(t, path, "valid contents")
	for _, src := range []string{filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		if err := CopyFile(src, path); err == nil {
			t.Fatal("expected copy failure")
		}
		assertFileContent(t, path, "valid contents")
	}
	assertNoAtomicTemps(t, filepath.Dir(path))

	link := filepath.Join(t.TempDir(), "dangling")
	missing := filepath.Join(t.TempDir(), "missing")
	if err := os.Symlink(missing, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteText(link, "new contents"); err == nil {
		t.Fatal("expected a dangling output symlink to be rejected")
	}
	if got, err := os.Readlink(link); err != nil || got != missing {
		t.Fatalf("dangling symlink changed: %q, %v", got, err)
	}
	if err := WriteText(t.TempDir(), "new contents"); err == nil {
		t.Fatal("expected a directory output to be rejected")
	}
	if err := WriteText(filepath.Join(path, "child"), "new contents"); err == nil {
		t.Fatal("expected parent directory creation to fail")
	}
	assertFileContent(t, path, "valid contents")
}

func TestCopyFileOntoItself(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(path, []byte("keep contents"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(path, path); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, path, "keep contents")
}

func TestAtomicWritesConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output")
	contents := make([]string, 4)
	valid := make(map[string]bool)
	for i := range contents {
		contents[i] = strings.Repeat(fmt.Sprint(i), 32*1024)
		valid[contents[i]] = true
	}
	if err := WriteText(path, contents[0]); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	readerResult := make(chan error, 1)
	go func() {
		for {
			data, err := os.ReadFile(path)
			if err != nil {
				readerResult <- err
				return
			}
			if !valid[string(data)] {
				readerResult <- fmt.Errorf("reader saw partial or mixed contents (%d bytes)", len(data))
				return
			}
			select {
			case <-done:
				readerResult <- nil
				return
			default:
			}
		}
	}()
	var writers sync.WaitGroup
	for _, content := range contents {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for i := 0; i < 10; i++ {
				if err := WriteText(path, content); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	writers.Wait()
	close(done)
	if err := <-readerResult; err != nil {
		t.Fatal(err)
	}
	assertNoAtomicTemps(t, dir)
}

func TestCreateSymlinkProtectsExistingPaths(t *testing.T) {
	for _, existing := range []string{"absent", "owned", "owned relative", "owned dangling", "regular", "foreign", "foreign dangling", "directory"} {
		t.Run(existing, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "target")
			foreign := filepath.Join(dir, "foreign")
			for _, path := range []string{target, foreign} {
				if err := os.WriteFile(path, []byte("keep contents"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(dir, "link")
			linkTarget := ""
			switch existing {
			case "owned", "owned dangling":
				linkTarget = target
			case "owned relative":
				linkTarget = "target"
			case "foreign":
				linkTarget = foreign
			case "foreign dangling":
				linkTarget = filepath.Join(dir, "missing")
			case "regular":
				if err := os.WriteFile(link, []byte("user file"), 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(link, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if linkTarget != "" {
				if err := os.Symlink(linkTarget, link); err != nil {
					t.Fatal(err)
				}
			}
			if existing == "owned dangling" {
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
			}
			err := CreateSymlink(target, link)
			wantError := existing == "regular" || existing == "foreign" || existing == "foreign dangling" || existing == "directory"
			if wantError && !errors.Is(err, os.ErrExist) || !wantError && err != nil {
				t.Fatalf("CreateSymlink() error = %v, want conflict = %t", err, wantError)
			}
			if existing == "absent" {
				linkTarget = target
			}
			if linkTarget != "" {
				if got, err := os.Readlink(link); err != nil || got != linkTarget {
					t.Errorf("symlink changed: %q, %v; want %q", got, err, linkTarget)
				}
			}
			if existing == "regular" {
				assertFileContent(t, link, "user file")
			}
			if existing != "owned dangling" {
				assertFileContent(t, target, "keep contents")
			}
			assertFileContent(t, foreign, "keep contents")
		})
	}
}

func TestCreateSymlinkFailureLeavesExistingLink(t *testing.T) {
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink("original", link); err != nil {
		t.Fatal(err)
	}
	if err := CreateSymlink("invalid\x00target", link); err == nil {
		t.Fatal("expected invalid target to fail")
	}
	if got, err := os.Readlink(link); err != nil || got != "original" {
		t.Fatalf("original symlink was lost: %q, %v", got, err)
	}
}

type fileReaderFunc func([]byte) (int, error)

func (f fileReaderFunc) Read(p []byte) (int, error) { return f(p) }

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Errorf("%s contents = %q, want %q", path, data, want)
	}
}

func assertNoAtomicTemps(t *testing.T, dir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, ".aether-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Errorf("temporary files left behind: %v", paths)
	}
}
