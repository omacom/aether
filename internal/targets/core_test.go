package targets

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aether/internal/template"
	"aether/internal/yozakura"
)

const testWallpaper = "/home/test/Pictures/evanescia.png"

type curatedIcho struct {
	Name  string         `json:"name"`
	Glass float64        `json:"glass"`
	Clock yozakura.Clock `json:"clock"`
	Term  [8]string      `json:"term"`
	Rest  map[string]any `json:"-"`
}

// ichoContext builds a Context from the curated Ichō theme with every role
// supplied as an explicit override, so Derive reproduces the curated values.
func ichoContext(t *testing.T) (Context, *recorder) {
	t.Helper()
	raw, err := os.ReadFile("../yozakura/testdata/curated.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Themes map[string]json.RawMessage `json:"themes"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	var c curatedIcho
	if err := json.Unmarshal(f.Themes["icho"], &c); err != nil {
		t.Fatal(err)
	}
	var roles map[string]any
	if err := json.Unmarshal(f.Themes["icho"], &roles); err != nil {
		t.Fatal(err)
	}
	over := map[string]string{}
	for _, k := range yozakura.Keys {
		over[k] = roles[k].(string)
	}
	cr := template.ColorRoles{
		Black: c.Term[0], Red: c.Term[1], Green: c.Term[2], Yellow: c.Term[3],
		Blue: c.Term[4], Magenta: c.Term[5], Cyan: c.Term[6], White: c.Term[7],
	}
	th := yozakura.Derive(cr, over, &c.Glass, &c.Clock)
	home := t.TempDir()
	rec := &recorder{}
	run := func(name string, args ...string) (string, error) {
		if name == "gsettings" {
			rec.calls = append(rec.calls, append([]string{name}, args...))
			return "true\n", nil
		}
		return rec.run(name, args...)
	}
	return Context{
		Name: c.Name, ID: "icho", Wallpaper: testWallpaper, Yoz: th, Home: home,
		Run: run, Have: func(string) bool { return true },
		Backups: NewBackups(home, time.Unix(4000, 0)),
	}, rec
}

func golden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func renderOf(t *testing.T, tg Target, i int, ctx Context) string {
	t.Helper()
	out, err := tg.Files[i].Render(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGoldenYozakura(t *testing.T) {
	ctx, _ := ichoContext(t)
	if got := renderOf(t, yozakuraTarget(), 0, ctx); got != golden(t, "theme.json") {
		t.Fatalf("theme.json differs:\n%s", got)
	}
}

func TestGoldenHyprland(t *testing.T) {
	ctx, _ := ichoContext(t)
	if got := renderOf(t, hyprlandTarget(), 0, ctx); got != golden(t, "aether-colors.lua") {
		t.Fatalf("hypr differs:\n%s", got)
	}
}

func TestGoldenKitty(t *testing.T) {
	ctx, _ := ichoContext(t)
	if got := renderOf(t, kittyTarget(), 0, ctx); got != golden(t, "aether-theme.conf") {
		t.Fatalf("kitty differs:\n%s", got)
	}
}

func TestGoldenGTK(t *testing.T) {
	ctx, _ := ichoContext(t)
	g := gtkTarget()
	for i := range g.Files {
		if got := renderOf(t, g, i, ctx); got != golden(t, "aether-colors.css") {
			t.Fatalf("gtk file %d differs:\n%s", i, got)
		}
	}
}

func TestGoldenSwaync(t *testing.T) {
	ctx, _ := ichoContext(t)
	if got := renderOf(t, swayncTarget(), 0, ctx); got != golden(t, "swaync-colors.css") {
		t.Fatalf("swaync differs:\n%s", got)
	}
}

func TestKittyReduceMotion(t *testing.T) {
	ctx, _ := ichoContext(t)
	ctx.Run = func(string, ...string) (string, error) { return "false\n", nil }
	if got := renderOf(t, kittyTarget(), 0, ctx); !strings.Contains(got, "cursor_trail 0\n") {
		t.Fatal("gsettings false should disable the cursor trail")
	}
	ctx.Run = func(string, ...string) (string, error) { return "true\n", nil }
	flag := filepath.Join(ctx.Home, ".local/state/yozakura/reduce-motion")
	if err := os.MkdirAll(filepath.Dir(flag), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(flag, nil, 0o644)
	if got := renderOf(t, kittyTarget(), 0, ctx); !strings.Contains(got, "cursor_trail 0\n") {
		t.Fatal("flag file should disable the cursor trail")
	}
}

// luaString parses a single-quoted Lua string literal, as inside dofile('...').
func luaString(t *testing.T, lit string) string {
	t.Helper()
	if !strings.HasPrefix(lit, "dofile('") || !strings.HasSuffix(lit, "')") {
		t.Fatalf("not a dofile call: %q", lit)
	}
	body := lit[len("dofile('") : len(lit)-2]
	var out strings.Builder
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '\\':
			i++
			if i >= len(body) {
				t.Fatalf("dangling backslash in %q", lit)
			}
			out.WriteByte(body[i])
		case '\'':
			t.Fatalf("unescaped quote in %q", lit)
		default:
			out.WriteByte(body[i])
		}
	}
	return out.String()
}

func TestHyprlandReloadQuotesPath(t *testing.T) {
	ctx, _ := ichoContext(t)
	ctx.Home = filepath.Join(t.TempDir(), `it's a \home`)
	argv := hyprlandTarget().Reload(ctx)
	if len(argv) != 3 || argv[0] != "hyprctl" || argv[1] != "eval" {
		t.Fatalf("argv = %q", argv)
	}
	want := filepath.Join(ctx.Home, ".config/hypr/aether-colors.lua")
	if got := luaString(t, argv[2]); got != want {
		t.Fatalf("lua path = %q, want %q", got, want)
	}
}

func TestApplyTwiceIsIdempotent(t *testing.T) {
	ctx, rec := ichoContext(t)
	first := Apply(ctx, All(), nil)
	for _, s := range first {
		if s.State == "failed" || s.State == "not-found" {
			t.Fatalf("first run: %+v", s)
		}
	}
	rec.calls = nil
	ctx.Backups = NewBackups(ctx.Home, time.Unix(4001, 0))
	for _, s := range Apply(ctx, All(), nil) {
		switch s.ID {
		case "hyprland", "awww":
			if s.State != "reloaded" {
				t.Errorf("%s = %s, want reloaded", s.ID, s.State)
			}
		default:
			if s.State != "unchanged" {
				t.Errorf("%s = %s (%s), want unchanged", s.ID, s.State, s.Error)
			}
		}
	}
	for _, c := range rec.calls {
		if c[0] != "hyprctl" && c[0] != "awww" && c[0] != "gsettings" {
			t.Errorf("unexpected call on run 2: %q", c)
		}
	}
	if _, err := os.Stat(filepath.Join(ctx.Home, ".local/state/aether/backups/4001")); err == nil {
		t.Error("second run created backups")
	}
}

func TestReloadFailureIsolated(t *testing.T) {
	ctx, rec := ichoContext(t)
	inner := ctx.Run
	ctx.Run = func(name string, args ...string) (string, error) {
		if name == "swaync-client" {
			rec.calls = append(rec.calls, append([]string{name}, args...))
			return "Failed to connect", errors.New("exit status 1")
		}
		return inner(name, args...)
	}
	byID := map[string]Status{}
	for _, s := range Apply(ctx, All(), nil) {
		byID[s.ID] = s
	}
	if s := byID["swaync"]; s.State != "failed" || s.Error != "Failed to connect" {
		t.Errorf("swaync = %+v", s)
	}
	if s := byID["kitty"]; s.State != "reloaded" {
		t.Errorf("kitty = %+v", s)
	}
	if byID["yozakura"].State == "failed" || byID["hyprland"].State == "failed" {
		t.Error("core targets must not fail")
	}
}

func TestMigrationReplacesOldIncludesAndCleansUp(t *testing.T) {
	ctx, _ := ichoContext(t)
	write := func(rel, body string) {
		p := filepath.Join(ctx.Home, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".config/kitty/kitty.conf", "font_size 12\ninclude yozakura-theme.conf\n")
	write(".config/kitty/yozakura-theme.conf", "old")
	write(".config/gtk-4.0/gtk.css", "@import url(\"yozakura-colors.css\");\n")
	write(".config/gtk-4.0/yozakura-colors.css", "old")
	write(".config/hypr/yozakura-colors.lua", "old")
	Apply(ctx, All(), nil)
	b, _ := os.ReadFile(filepath.Join(ctx.Home, ".config/kitty/kitty.conf"))
	if string(b) != "font_size 12\ninclude aether-theme.conf\n" {
		t.Errorf("kitty.conf = %q", b)
	}
	b, _ = os.ReadFile(filepath.Join(ctx.Home, ".config/gtk-4.0/gtk.css"))
	if string(b) != "@import url(\"aether-colors.css\");\n" {
		t.Errorf("gtk.css = %q", b)
	}
	for _, old := range []string{".config/kitty/yozakura-theme.conf", ".config/gtk-4.0/yozakura-colors.css", ".config/hypr/yozakura-colors.lua"} {
		if _, err := os.Stat(filepath.Join(ctx.Home, old)); err == nil {
			t.Errorf("%s not cleaned up", old)
		}
	}
	b, _ = os.ReadFile(filepath.Join(ctx.Home, ".config/swaync/style.css"))
	if string(b) != "@import url(\"file:///etc/xdg/swaync/style.css\");\n@import url(\"colors.css\");\n" {
		t.Errorf("swaync style.css = %q", b)
	}
	b, _ = os.ReadFile(filepath.Join(ctx.Home, ".config/hypr/hyprland.lua"))
	if !strings.HasSuffix(string(b), `aether-colors.lua")`+"\n") {
		t.Errorf("hyprland.lua = %q", b)
	}
}

func TestYozakuraJSONShape(t *testing.T) {
	ctx, _ := ichoContext(t)
	out := renderOf(t, yozakuraTarget(), 0, ctx)
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatal(err)
	}
	want := append([]string{"glass", "clock", "term", "id", "name", "wallpaper"}, yozakura.Keys[:]...)
	if len(m) != len(want) {
		t.Errorf("got %d keys, want %d", len(m), len(want))
	}
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
	var term []string
	json.Unmarshal(m["term"], &term)
	if len(term) != 8 {
		t.Errorf("term length %d", len(term))
	}
	var name string
	json.Unmarshal(m["name"], &name)
	if name != "Ichō" || !strings.Contains(out, `"Ichō"`) {
		t.Errorf("name = %q, raw output escaped", name)
	}
}

func TestDetectUsesInjectedLookup(t *testing.T) {
	ctx, rec := ichoContext(t)
	ctx.Have = func(bin string) bool { return false }
	byID := map[string]Status{}
	for _, s := range Apply(ctx, All(), nil) {
		byID[s.ID] = s
	}
	for _, id := range []string{"hyprland", "awww", "kitty", "swaync"} {
		if byID[id].State != "not-found" {
			t.Errorf("%s = %s", id, byID[id].State)
		}
	}
	if byID["yozakura"].State != "written" || byID["gtk"].State != "written" {
		t.Errorf("%+v", byID)
	}
	for _, c := range rec.calls {
		t.Errorf("unexpected command %q", c)
	}
}
