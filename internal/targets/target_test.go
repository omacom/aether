package targets

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type recorder struct {
	calls  [][]string
	err    error
	stderr string
}

func (r *recorder) run(name string, args ...string) (string, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return r.stderr, r.err
}

func newCtx(t *testing.T) (Context, *recorder) {
	home := t.TempDir()
	r := &recorder{}
	return Context{Home: home, Run: r.run, Backups: NewBackups(home, time.Unix(3000, 0))}, r
}

func simpleTarget(content string) Target {
	return Target{
		ID: "t", Label: "T",
		Detect: func(Context) bool { return true },
		Files:  []File{{Path: ".config/t/out.conf", Render: func(Context) (string, error) { return content, nil }}},
		Reload: func(Context) []string { return []string{"reloadcmd", "-x"} },
	}
}

func all(string) bool { return true }

func TestApplyDisabledIsSkipped(t *testing.T) {
	ctx, r := newCtx(t)
	st := Apply(ctx, []Target{simpleTarget("a")}, func(string) bool { return false })
	if st[0].State != "skipped" || len(r.calls) != 0 {
		t.Fatal(st, r.calls)
	}
	if _, err := os.Stat(filepath.Join(ctx.Home, ".config/t/out.conf")); err == nil {
		t.Fatal("file written")
	}
}

func TestApplyNotDetected(t *testing.T) {
	ctx, _ := newCtx(t)
	tg := simpleTarget("a")
	tg.Detect = func(Context) bool { return false }
	if st := Apply(ctx, []Target{tg}, all); st[0].State != "not-found" {
		t.Fatal(st)
	}
}

func TestApplyReloadOnlyOnChange(t *testing.T) {
	ctx, r := newCtx(t)
	st := Apply(ctx, []Target{simpleTarget("a")}, all)
	if st[0].State != "reloaded" || len(r.calls) != 1 {
		t.Fatal(st, r.calls)
	}
	if !reflect.DeepEqual(r.calls[0], []string{"reloadcmd", "-x"}) {
		t.Fatal(r.calls)
	}
	if st[0].Path != filepath.Join(ctx.Home, ".config/t/out.conf") {
		t.Fatal(st[0].Path)
	}
	st = Apply(ctx, []Target{simpleTarget("a")}, all)
	if st[0].State != "unchanged" || len(r.calls) != 1 {
		t.Fatal(st, r.calls)
	}
}

func TestApplyWrittenWithoutReload(t *testing.T) {
	ctx, _ := newCtx(t)
	tg := simpleTarget("a")
	tg.Reload = nil
	if st := Apply(ctx, []Target{tg}, all); st[0].State != "written" {
		t.Fatal(st)
	}
}

func TestApplyAlwaysReload(t *testing.T) {
	ctx, r := newCtx(t)
	tg := simpleTarget("a")
	tg.AlwaysReload = true
	Apply(ctx, []Target{tg}, all)
	st := Apply(ctx, []Target{tg}, all)
	if st[0].State != "reloaded" || len(r.calls) != 2 {
		t.Fatal(st, r.calls)
	}
}

func TestApplyReloadErrorCarriesStderr(t *testing.T) {
	ctx, r := newCtx(t)
	r.err = errors.New("exit 1")
	r.stderr = " boom\n"
	st := Apply(ctx, []Target{simpleTarget("a"), simpleTarget("b")}, all)
	if st[0].State != "failed" || st[0].Error != "boom" {
		t.Fatal(st)
	}
	r.stderr = ""
	st = Apply(ctx, []Target{simpleTarget("c")}, all)
	if st[0].Error != "exit 1" {
		t.Fatal(st)
	}
}

func TestApplyRenderErrorContinues(t *testing.T) {
	ctx, _ := newCtx(t)
	bad := simpleTarget("a")
	bad.ID = "bad"
	bad.Files[0].Render = func(Context) (string, error) { return "", errors.New("nope") }
	st := Apply(ctx, []Target{bad, simpleTarget("a")}, all)
	if st[0].State != "failed" || st[0].Error != "nope" || st[1].State != "reloaded" {
		t.Fatal(st)
	}
}

func TestApplyIncludesTriggerReload(t *testing.T) {
	ctx, r := newCtx(t)
	tg := simpleTarget("a")
	tg.Includes = []Include{{Path: ".config/t/main.conf", Line: "include out.conf"}}
	Apply(ctx, []Target{tg}, all)
	os.WriteFile(filepath.Join(ctx.Home, ".config/t/main.conf"), []byte("x\n"), 0o644)
	st := Apply(ctx, []Target{tg}, all)
	if st[0].State != "reloaded" || len(r.calls) != 2 {
		t.Fatal(st, r.calls)
	}
}

func TestApplyCleanupRemovesOldFiles(t *testing.T) {
	ctx, _ := newCtx(t)
	old := filepath.Join(ctx.Home, ".config/t/old.conf")
	os.MkdirAll(filepath.Dir(old), 0o755)
	os.WriteFile(old, []byte("o"), 0o644)
	tg := simpleTarget("a")
	tg.Cleanup = []string{".config/t/old.conf", ".config/t/missing.conf"}
	st := Apply(ctx, []Target{tg}, all)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old file remains", st)
	}
	if st[0].State == "failed" {
		t.Fatal(st)
	}
}

func TestApplyCleanupSkippedOnFailure(t *testing.T) {
	ctx, _ := newCtx(t)
	old := filepath.Join(ctx.Home, "old")
	os.WriteFile(old, []byte("o"), 0o644)
	tg := simpleTarget("a")
	tg.Files[0].Render = func(Context) (string, error) { return "", errors.New("x") }
	tg.Cleanup = []string{"old"}
	Apply(ctx, []Target{tg}, all)
	if _, err := os.Stat(old); err != nil {
		t.Fatal("cleanup ran after failure")
	}
}
