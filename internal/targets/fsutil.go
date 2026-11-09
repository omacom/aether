package targets

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WriteIfChanged writes content to path atomically (temp file + rename) and
// reports whether the file changed. Parent directories are created.
func WriteIfChanged(path, content string) (bool, error) {
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
		if old, err := os.ReadFile(path); err == nil && string(old) == content {
			return false, nil
		}
	}
	if err := writeAtomic(path, content, mode); err != nil {
		return false, err
	}
	return true, nil
}

func writeAtomic(path, content string, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.WriteString(content); err != nil {
		return fail(err)
	}
	if err := tmp.Chmod(mode); err != nil {
		return fail(err)
	}
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// Backups copies files into a timestamped directory, at most once per run.
type Backups struct {
	home string
	root string
	mu   sync.Mutex
	done map[string]bool
}

// NewBackups creates the backup set for one apply run:
// <home>/.local/state/aether/backups/<unix-ts>/.
func NewBackups(home string, now time.Time) *Backups {
	return &Backups{
		home: home,
		root: filepath.Join(home, ".local", "state", "aether", "backups", strconv.FormatInt(now.Unix(), 10)),
		done: map[string]bool{},
	}
}

// Save copies path into the backup directory before its first modification.
// It is a no-op if the file does not exist or was already saved this run.
func (b *Backups) Save(path string) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.done[path] {
		return nil
	}
	src, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer src.Close()
	st, err := src.Stat()
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(b.home, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = strings.TrimPrefix(path, string(filepath.Separator))
	}
	dst := filepath.Join(b.root, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	b.done[path] = true
	return nil
}

// EnsureInclude makes sure the user-owned file inc.Path contains every line of
// inc.Line (whole-line match, trailing whitespace ignored). Lines listed in
// inc.Replaces are swapped in place for Line's first line. Missing lines are
// prepended (in order) or appended when inc.Append is set. The file is backed
// up before its first modification and created if absent.
func EnsureInclude(ctx Context, inc Include) (bool, error) {
	path := filepath.Join(ctx.Home, inc.Path)
	var body string
	exists := true
	if raw, err := os.ReadFile(path); err == nil {
		body = string(raw)
	} else if os.IsNotExist(err) {
		exists = false
	} else {
		return false, err
	}

	want := splitLines(inc.Line)
	if len(want) == 0 {
		return false, nil
	}
	var lines []string
	if body != "" {
		lines = strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	}
	has := func(l string) bool {
		for _, x := range lines {
			if strings.TrimRight(x, " \t\r") == l {
				return true
			}
		}
		return false
	}

	changed := false
	if len(inc.Replaces) > 0 {
		repl := map[string]bool{}
		for _, r := range inc.Replaces {
			repl[strings.TrimRight(r, " \t\r")] = true
		}
		haveFirst := has(want[0])
		var out []string
		for _, x := range lines {
			if repl[strings.TrimRight(x, " \t\r")] {
				changed = true
				if !haveFirst {
					out = append(out, want[0])
					haveFirst = true
				}
				continue
			}
			out = append(out, x)
		}
		lines = out
	}

	var missing []string
	for _, l := range want {
		if !has(l) {
			missing = append(missing, l)
		}
	}
	if len(missing) > 0 {
		changed = true
		if inc.Append {
			lines = append(lines, missing...)
		} else {
			lines = append(append([]string{}, missing...), lines...)
		}
	}
	if !changed && exists {
		return false, nil
	}
	if err := ctx.Backups.Save(path); err != nil {
		return false, err
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	if err := writeAtomic(path, strings.Join(lines, "\n")+"\n", mode); err != nil {
		return false, err
	}
	return true, nil
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimRight(l, " \t\r"); l != "" {
			out = append(out, l)
		}
	}
	return out
}
