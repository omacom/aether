package cli

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"aether/internal/blueprint"
)

//go:embed all:testdata/templates
var pipelineTemplates embed.FS

type pipelineEnv struct {
	home          string
	lock          string
	mu            sync.Mutex
	calls         [][]string
	fail          map[string]bool // binaries whose Run fails
	have          map[string]bool
	calledExtract int
}

// newPipelineEnv isolates HOME, imports the curated blueprints and fakes every
// external command.
func newPipelineEnv(t *testing.T) *pipelineEnv {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local/share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	e := &pipelineEnv{
		home: home, lock: filepath.Join(home, "generate.lock"),
		fail: map[string]bool{}, have: map[string]bool{},
	}
	bps, def, err := blueprint.ImportYozakura(curatedFixture)
	if err != nil {
		t.Fatal(err)
	}
	svc := blueprint.NewService()
	for _, bp := range bps {
		if err := svc.Save(bp.Name, bp); err != nil {
			t.Fatal(err)
		}
	}
	if err := setDefaultBlueprint(def); err != nil {
		t.Fatal(err)
	}
	return e
}

func (e *pipelineEnv) wallpaper(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(e.home, "Pictures", name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("not really a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func (e *pipelineEnv) deps(extract func(string, bool, string) ([16]string, error)) (generateDeps, *bytes.Buffer, *bytes.Buffer) {
	var so, se bytes.Buffer
	return generateDeps{
		Extract: func(p string, l bool, m string) ([16]string, error) {
			e.mu.Lock()
			e.calledExtract++
			e.mu.Unlock()
			return extract(p, l, m)
		},
		Run: func(name string, args ...string) (string, error) {
			e.mu.Lock()
			defer e.mu.Unlock()
			e.calls = append(e.calls, append([]string{name}, args...))
			if e.fail[name] {
				return "boom", errors.New("exit status 1")
			}
			return "", nil
		},
		Have:         func(bin string) bool { return e.have[bin] },
		LockPath:     e.lock,
		TemplatesDir: "testdata/templates",
		Stdout:       &so,
		Stderr:       &se,
	}, &so, &se
}

func extractOK(string, bool, string) ([16]string, error) {
	return [16]string{"#101010", "#cc3333", "#33cc33", "#cccc33", "#3333cc", "#cc33cc", "#33cccc", "#e0e0e0",
		"#404040", "#ff5555", "#55ff55", "#ffff55", "#5555ff", "#ff55ff", "#55ffff", "#ffffff"}, nil
}

func readThemeJSON(t *testing.T, e *pipelineEnv) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(e.home, ".local/state/yozakura/theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestGenerateUsesMatchedBlueprint(t *testing.T) {
	e := newPipelineEnv(t)
	wp := e.wallpaper(t, "evanescia.png")
	d, _, se := e.deps(extractOK)
	if code := runGenerateWith([]string{wp}, pipelineTemplates, d); code != 0 {
		t.Fatalf("exit %d: %s", code, se)
	}
	if e.calledExtract != 0 {
		t.Error("extractor called for a matched wallpaper")
	}
	m := readThemeJSON(t, e)
	if m["id"] != "icho" || m["name"] != "Ichō" || m["wallpaper"] != wp {
		t.Errorf("theme.json id/name/wallpaper = %v / %v / %v", m["id"], m["name"], m["wallpaper"])
	}
	want, _, _ := blueprint.ImportYozakura(curatedFixture)
	for _, bp := range want {
		if bp.Name != "icho" {
			continue
		}
		for role, hex := range bp.Palette.YozakuraRoles {
			if m[role] != hex {
				t.Errorf("role %s = %v, want %s", role, m[role], hex)
			}
		}
	}
}

func TestGenerateExtractionIsAuto(t *testing.T) {
	e := newPipelineEnv(t)
	wp := e.wallpaper(t, "unmatched.png")
	d, _, se := e.deps(extractOK)
	if code := runGenerateWith([]string{wp}, pipelineTemplates, d); code != 0 {
		t.Fatalf("exit %d: %s", code, se)
	}
	m := readThemeJSON(t, e)
	if m["id"] != "auto" || m["name"] != "Auto" || e.calledExtract != 1 {
		t.Errorf("id=%v name=%v extract=%d", m["id"], m["name"], e.calledExtract)
	}
}

func TestGenerateFallsBackToDefaultOnExtractError(t *testing.T) {
	e := newPipelineEnv(t)
	wp := e.wallpaper(t, "unmatched.png")
	d, _, se := e.deps(func(string, bool, string) ([16]string, error) {
		return [16]string{}, errors.New("decode failed")
	})
	if code := runGenerateWith([]string{wp}, pipelineTemplates, d); code != 0 {
		t.Fatalf("exit %d: %s", code, se)
	}
	if m := readThemeJSON(t, e); m["id"] != "icho" || m["wallpaper"] != wp {
		t.Errorf("id=%v wallpaper=%v", m["id"], m["wallpaper"])
	}
	if !strings.Contains(se.String(), "falling back to default blueprint icho") {
		t.Errorf("stderr = %q", se)
	}
}

func TestGenerateJSONStdoutOnly(t *testing.T) {
	e := newPipelineEnv(t)
	wp := e.wallpaper(t, "evanescia.png")
	d, so, se := e.deps(extractOK)
	if code := runGenerateWith([]string{wp, "--json"}, pipelineTemplates, d); code != 0 {
		t.Fatalf("exit %d: %s", code, se)
	}
	out := so.String()
	if !strings.HasSuffix(out, "\n") || strings.Count(out, "\n") != 1 {
		t.Fatalf("stdout must be one line, got %q", out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("stdout is not one JSON object: %v\n%q", err, out)
	}
	if m["wallpaper"] != wp {
		t.Errorf("wallpaper = %v", m["wallpaper"])
	}
	if !strings.Contains(out, `"name":"Ichō"`) {
		t.Errorf("unicode escaped or reordered: %s", out)
	}
	if !strings.HasPrefix(out, `{"name":`) {
		t.Errorf("key order not preserved: %s", out)
	}
	if !strings.Contains(se.String(), "Theme applied successfully") {
		t.Errorf("progress should go to stderr, got %q", se)
	}
}

func TestGenerateExitCodeOnlyForCoreTargets(t *testing.T) {
	e := newPipelineEnv(t)
	wp := e.wallpaper(t, "evanescia.png")
	if err := os.MkdirAll(filepath.Join(e.home, ".config/hypr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.home, ".config/hypr/hyprland.lua"), []byte("-- user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.have["swaync-client"], e.have["hyprctl"] = true, true

	e.fail["swaync-client"] = true
	d, _, se := e.deps(extractOK)
	if code := runGenerateWith([]string{wp}, pipelineTemplates, d); code != 0 {
		t.Fatalf("swaync failure: exit %d: %s", code, se)
	}
	if !strings.Contains(se.String(), "swaync") {
		t.Errorf("failed target not reported on stderr: %q", se)
	}

	e.fail["swaync-client"], e.fail["hyprctl"] = false, true
	d, so, se := e.deps(extractOK)
	if code := runGenerateWith([]string{wp, "--json"}, pipelineTemplates, d); code != 1 {
		t.Fatalf("hyprland failure: exit %d, want 1: %s", code, se)
	}
	_ = so
}

func TestGenerateLockLastWins(t *testing.T) {
	e := newPipelineEnv(t)
	wps := []string{e.wallpaper(t, "a.png"), e.wallpaper(t, "b.png"), e.wallpaper(t, "c.png")}
	slow := func(string, bool, string) ([16]string, error) {
		time.Sleep(300 * time.Millisecond)
		return extractOK("", false, "")
	}
	codes := make([]int, 3)
	outs := make([]*bytes.Buffer, 3)
	errs := make([]*bytes.Buffer, 3)
	var wg sync.WaitGroup
	for i, wp := range wps {
		d, so, se := e.deps(slow)
		outs[i], errs[i] = so, se
		wg.Add(1)
		go func(i int, wp string) {
			defer wg.Done()
			codes[i] = runGenerateWith([]string{wp, "--json"}, pipelineTemplates, d)
		}(i, wp)
		time.Sleep(80 * time.Millisecond) // A, B, C take tickets in order
	}
	wg.Wait()

	for i, c := range codes {
		if c != 0 {
			t.Errorf("run %d exit %d: %s", i, c, errs[i])
		}
	}
	if m := readThemeJSON(t, e); m["wallpaper"] != wps[2] {
		t.Errorf("final wallpaper = %v, want %s", m["wallpaper"], wps[2])
	}
	if outs[2].Len() == 0 {
		t.Error("newest run printed nothing")
	}
	if !strings.Contains(errs[1].String(), "Superseded") {
		t.Errorf("B should be superseded: %q", errs[1])
	}
	if outs[1].Len() != 0 {
		t.Errorf("superseded run wrote stdout: %q", outs[1])
	}
	filepath.Walk(e.home, func(p string, info os.FileInfo, err error) error {
		if err == nil && strings.Contains(info.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", p)
		}
		return nil
	})
}
