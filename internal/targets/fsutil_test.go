package targets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteIfChangedSkipsIdentical(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a", "b.txt")
	if ch, err := WriteIfChanged(p, "x"); err != nil || !ch {
		t.Fatalf("first write: %v %v", ch, err)
	}
	old := time.Now().Add(-time.Hour)
	os.Chtimes(p, old, old)
	st1, _ := os.Stat(p)
	if ch, err := WriteIfChanged(p, "x"); err != nil || ch {
		t.Fatalf("second write: %v %v", ch, err)
	}
	st2, _ := os.Stat(p)
	if !st1.ModTime().Equal(st2.ModTime()) {
		t.Fatal("mtime changed")
	}
}

func TestWriteIfChangedAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.conf")
	WriteIfChanged(p, "one")
	WriteIfChanged(p, "two")
	m, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	hidden, _ := filepath.Glob(filepath.Join(dir, ".*.tmp"))
	if len(m)+len(hidden) != 0 {
		t.Fatalf("tmp left: %v %v", m, hidden)
	}
	if b, _ := os.ReadFile(p); string(b) != "two" {
		t.Fatalf("content %q", b)
	}
}

func TestWriteIfChangedPreservesMode(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	os.WriteFile(p, []byte("a"), 0o600)
	WriteIfChanged(p, "b")
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode())
	}
}

func TestBackupsSaveOncePerRun(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, ".config", "x.conf")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("orig"), 0o644)
	b := NewBackups(home, time.Unix(1000, 0))
	if err := b.Save(p); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(p, []byte("changed"), 0o644)
	b.Save(p)
	got, _ := os.ReadFile(filepath.Join(home, ".local/state/aether/backups/1000/.config/x.conf"))
	if string(got) != "orig" {
		t.Fatalf("backup %q", got)
	}
	if err := b.Save(filepath.Join(home, "missing")); err != nil {
		t.Fatal(err)
	}
}

func ctxFor(home string) Context {
	return Context{Home: home, Backups: NewBackups(home, time.Unix(2000, 0))}
}

func backupExists(home, rel string) bool {
	_, err := os.Stat(filepath.Join(home, ".local/state/aether/backups/2000", rel))
	return err == nil
}

func TestEnsureIncludeCreatesMissingFile(t *testing.T) {
	home := t.TempDir()
	ch, err := EnsureInclude(ctxFor(home), Include{Path: ".config/a/b.conf", Line: "include x"})
	if err != nil || !ch {
		t.Fatal(ch, err)
	}
	b, _ := os.ReadFile(filepath.Join(home, ".config/a/b.conf"))
	if string(b) != "include x\n" {
		t.Fatalf("%q", b)
	}
}

func TestEnsureIncludeReplacesOldLine(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "h.conf")
	os.WriteFile(p, []byte("a=1\ninclude yozakura-theme.conf  \nb=2\n"), 0o644)
	ch, err := EnsureInclude(ctxFor(home), Include{Path: "h.conf", Line: "include aether-theme.conf", Replaces: []string{"include yozakura-theme.conf"}})
	if err != nil || !ch {
		t.Fatal(ch, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "a=1\ninclude aether-theme.conf\nb=2\n" {
		t.Fatalf("%q", b)
	}
	if !backupExists(home, "h.conf") {
		t.Fatal("no backup")
	}
}

func TestEnsureIncludeIdempotent(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "h.conf")
	os.WriteFile(p, []byte("a\n"), 0o644)
	inc := Include{Path: "h.conf", Line: "include z"}
	ctx := ctxFor(home)
	EnsureInclude(ctx, inc)
	os.RemoveAll(filepath.Join(home, ".local"))
	ch, err := EnsureInclude(ctx, inc)
	if err != nil || ch {
		t.Fatal(ch, err)
	}
	if backupExists(home, "h.conf") {
		t.Fatal("second backup")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "include z\na\n" {
		t.Fatalf("%q", b)
	}
}

func TestEnsureIncludeWholeLineNotSubstring(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "h.conf")
	os.WriteFile(p, []byte("# include z\n"), 0o644)
	ch, _ := EnsureInclude(ctxFor(home), Include{Path: "h.conf", Line: "include z"})
	if !ch {
		t.Fatal("substring treated as present")
	}
}

func TestEnsureIncludeMultiLineAndAppend(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "h.conf")
	os.WriteFile(p, []byte("one\nmid"), 0o644)
	ch, err := EnsureInclude(ctxFor(home), Include{Path: "h.conf", Line: "one\ntwo\nthree", Append: true})
	if err != nil || !ch {
		t.Fatal(ch, err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "one\nmid\ntwo\nthree\n" {
		t.Fatalf("%q", b)
	}
	os.WriteFile(p, []byte("one\nmid\n"), 0o644)
	EnsureInclude(ctxFor(home), Include{Path: "h.conf", Line: "one\ntwo\nthree"})
	b, _ = os.ReadFile(p)
	if !strings.HasPrefix(string(b), "two\nthree\none\n") {
		t.Fatalf("prepend order: %q", b)
	}
}
