package platform

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadJSON reads and unmarshals a JSON file into T.
func ReadJSON[T any](path string) (T, error) {
	var v T
	data, err := os.ReadFile(path)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return v, err
	}
	return v, nil
}

// WriteJSON atomically writes indented JSON to path, creating parent directories
// as needed. Existing permissions and destination symlinks are preserved.
func WriteJSON[T any](path string, v T) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, bytes.NewReader(data), 0644)
}

// ReadText reads a file and returns its contents as a string.
func ReadText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteText atomically writes content to path, creating parent directories as
// needed. New files use 0644 (before umask); existing permissions and destination
// symlinks are preserved.
func WriteText(path string, content string) error {
	return writeFileAtomic(path, strings.NewReader(content), 0644)
}

// CopyFile atomically copies src to dst. New files use the source permissions
// (before umask); existing destination permissions and symlinks are preserved.
func CopyFile(src, dst string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := srcFile.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	return writeFileAtomic(dst, srcFile, info.Mode().Perm())
}

func writeFileAtomic(path string, content io.Reader, perm os.FileMode) (err error) {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	info, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	preservePerm := err == nil
	if preservePerm {
		// Replace the referent, not an intentional output symlink. Refuse dangling
		// links rather than silently replacing them with regular files.
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		info, err = os.Stat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cannot replace non-regular file %q", path)
		}
		perm = info.Mode().Perm()
	}

	var temp *os.File
	for {
		var suffix [16]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return err
		}
		tempPath := filepath.Join(filepath.Dir(path), fmt.Sprintf(".aether-%x.tmp", suffix))
		// O_EXCL prevents collisions; creation with perm also respects the umask.
		temp, err = os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
		if !os.IsExist(err) {
			break
		}
	}
	if err != nil {
		return err
	}

	closed := false
	defer func() {
		if !closed {
			if closeErr := temp.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
		if err != nil {
			if removeErr := os.Remove(temp.Name()); removeErr != nil && !os.IsNotExist(removeErr) {
				err = errors.Join(err, fmt.Errorf("remove temporary file: %w", removeErr))
			}
		}
	}()

	if _, err := io.Copy(temp, content); err != nil {
		return err
	}
	if preservePerm {
		if err := temp.Chmod(perm); err != nil {
			return err
		}
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	err = temp.Close()
	closed = true
	if err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

// EnsureDir creates the directory at path along with any necessary parents
// (equivalent to mkdir -p).
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// CleanDir removes all files in directory at path but leaves subdirectories
// intact. Returns nil if the directory does not exist.
func CleanDir(path string) error {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(path, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// FileExists reports whether the file at path exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// CreateSymlink creates a symbolic link at link pointing to target, or leaves an
// existing symlink to that target intact. It never replaces an existing path.
func CreateSymlink(target, link string) error {
	link, err := filepath.Abs(link)
	if err != nil {
		return err
	}
	if err := EnsureDir(filepath.Dir(link)); err != nil {
		return err
	}
	info, err := os.Lstat(link)
	if os.IsNotExist(err) {
		// Symlink fails if another writer creates the destination in the meantime.
		return os.Symlink(target, link)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		current, err := os.Readlink(link)
		if err != nil {
			return err
		}
		if current == target {
			return nil
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(link), target)
		}
		resolvedTarget, targetErr := filepath.EvalSymlinks(target)
		resolvedLink, linkErr := filepath.EvalSymlinks(link)
		if targetErr == nil && linkErr == nil && resolvedTarget == resolvedLink {
			return nil
		}
	}
	return fmt.Errorf("refusing to replace %q: %w", link, os.ErrExist)
}

// DeleteFile removes the file at path. Returns nil if the file does not exist.
func DeleteFile(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
