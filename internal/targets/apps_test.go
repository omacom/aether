package targets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func updateGolden(t *testing.T, name, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(filepath.Join("testdata", "golden", name), []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	updateGolden(t, name, got)
	if want := golden(t, name); got != want {
		t.Fatalf("%s differs:\n%s", name, got)
	}
}

func readHome(t *testing.T, ctx Context, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(ctx.Home, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeHome(t *testing.T, ctx Context, rel, body string) {
	t.Helper()
	p := filepath.Join(ctx.Home, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGoldenRofi(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "aether-colors.rasi", renderOf(t, rofiTarget(), 0, ctx))
}

func TestGoldenNeovim(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "neovim-theme.lua", renderOf(t, neovimTarget(), 0, ctx))
}

func TestGoldenBtop(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "btop-aether.theme", renderOf(t, btopTarget(), 0, ctx))
}

func TestGoldenVSCode(t *testing.T) {
	ctx, _ := ichoContext(t)
	tg := vscodeTarget()
	if len(tg.Files) != 2 {
		t.Fatalf("files = %d", len(tg.Files))
	}
	for i, f := range tg.Files {
		out := renderOf(t, tg, i, ctx)
		if strings.Contains(out, "{background}") {
			t.Errorf("%s has unreplaced variables", f.Path)
		}
		if strings.HasSuffix(f.Path, "aether-color-theme.json") {
			checkGolden(t, "vscode-color-theme.json", out)
		}
	}
	if !strings.HasPrefix(tg.Files[0].Path, ".vscode/extensions/local.theme-aether-1.0.0/") {
		t.Errorf("path = %s", tg.Files[0].Path)
	}
}

func TestGoldenKdeglobals(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "kdeglobals", renderOf(t, qtTarget(), 0, ctx))
}

func TestGoldenFish(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "aether-colors.fish", renderOf(t, fishTarget(), 0, ctx))
}

func TestGoldenStarship(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "starship.toml", renderOf(t, starshipTarget(), 0, ctx))
}

func TestGoldenChrome(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "chromium.theme", renderOf(t, chromeTarget(), 0, ctx))
}

func TestGoldenHyprlock(t *testing.T) {
	ctx, _ := ichoContext(t)
	checkGolden(t, "aether-hyprlock.conf", renderOf(t, hyprlockTarget(), 0, ctx))
}

func TestNeovimNotFoundWithoutConfigDir(t *testing.T) {
	ctx, _ := ichoContext(t)
	sts := Apply(ctx, []Target{neovimTarget()}, nil)
	if sts[0].State != "not-found" {
		t.Fatalf("state = %s", sts[0].State)
	}
	if _, err := os.Stat(filepath.Join(ctx.Home, ".config/nvim")); !os.IsNotExist(err) {
		t.Fatalf("~/.config/nvim was created: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(ctx.Home, ".config/nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	if s := Apply(ctx, []Target{neovimTarget()}, nil)[0]; s.State != "written" {
		t.Fatalf("state = %s (%s)", s.State, s.Error)
	}
	readHome(t, ctx, ".config/nvim/lua/aether/theme.lua")
}

func TestHyprlandNotFoundWithoutLuaConfig(t *testing.T) {
	ctx, rec := ichoContext(t)
	if s := Apply(ctx, []Target{hyprlandTarget()}, nil)[0]; s.State != "not-found" {
		t.Fatalf("state = %s", s.State)
	}
	if _, err := os.Stat(filepath.Join(ctx.Home, ".config/hypr")); !os.IsNotExist(err) {
		t.Fatalf("hypr dir created: %v", err)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("calls = %q", rec.calls)
	}
	writeHome(t, ctx, ".config/hypr/hyprland.lua", "-- user\n")
	if s := Apply(ctx, []Target{hyprlandTarget()}, nil)[0]; s.State != "reloaded" {
		t.Fatalf("state = %s (%s)", s.State, s.Error)
	}
}

func TestSwayncCcBgAlphaMatchesPythonRound(t *testing.T) {
	ctx, _ := ichoContext(t)
	ctx.Yoz.Glass = 0.905
	out := renderOf(t, swayncTarget(), 0, ctx)
	if !strings.Contains(out, "--cc-bg: rgba(") || !strings.Contains(out, ", 0.85);") {
		t.Fatalf("cc-bg alpha not 0.85:\n%s", out)
	}
}

const kdeSample = "[General]\nColorScheme=BreezeDark\nfont=Noto Sans,10,-1,5,50,0,0,0,0,0\n\n" +
	"[Colors:Window]\nBackgroundNormal=1,2,3\nForegroundActive=9,9,9\n\n" +
	"[KDE]\nwidgetStyle=Breeze\n"

func TestKdeglobalsPreservesOtherSections(t *testing.T) {
	ctx, _ := ichoContext(t)
	writeHome(t, ctx, ".config/kdeglobals", kdeSample)
	if s := Apply(ctx, []Target{qtTarget()}, nil)[0]; s.State != "written" {
		t.Fatalf("state = %s (%s)", s.State, s.Error)
	}
	got := readHome(t, ctx, ".config/kdeglobals")
	for _, keep := range []string{
		"[General]\nColorScheme=BreezeDark\nfont=Noto Sans,10,-1,5,50,0,0,0,0,0\n",
		"[KDE]\nwidgetStyle=Breeze\n",
		"ForegroundActive=9,9,9\n",
	} {
		if !strings.Contains(got, keep) {
			t.Errorf("lost %q in:\n%s", keep, got)
		}
	}
	if strings.Contains(got, "BackgroundNormal=1,2,3") {
		t.Error("old BackgroundNormal kept")
	}
	if strings.Count(got, "[Colors:Window]") != 1 {
		t.Errorf("duplicated section:\n%s", got)
	}
	for _, sec := range []string{"Window", "View", "Button", "Selection", "Tooltip", "Header"} {
		if !strings.Contains(got, "[Colors:"+sec+"]") {
			t.Errorf("missing %s", sec)
		}
	}
	if _, err := os.Stat(filepath.Join(ctx.Home, ".local/state/aether/backups/4000/.config/kdeglobals")); err != nil {
		t.Errorf("no backup: %v", err)
	}
	if s := Apply(ctx, []Target{qtTarget()}, nil)[0]; s.State != "unchanged" {
		t.Errorf("second run = %s", s.State)
	}
}

func TestSetINISections(t *testing.T) {
	got := setINISections("[A]\nx=1\n[B]\ny=2\n", map[string]map[string]string{
		"A": {"x": "9", "z": "3"}, "C": {"k": "v"},
	})
	want := "[A]\nx=9\nz=3\n[B]\ny=2\n\n[C]\nk=v\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQtDetect(t *testing.T) {
	ctx, _ := ichoContext(t)
	ctx.Have = func(b string) bool { return b == "dolphin" }
	if !qtTarget().Detect(ctx) {
		t.Error("dolphin should detect")
	}
	ctx.Have = func(string) bool { return false }
	if qtTarget().Detect(ctx) {
		t.Error("nothing present should not detect")
	}
	writeHome(t, ctx, ".config/kdeglobals", "")
	if !qtTarget().Detect(ctx) {
		t.Error("kdeglobals should detect")
	}
}

func TestStarshipAddsPaletteToEmptyFile(t *testing.T) {
	ctx, _ := ichoContext(t)
	if s := Apply(ctx, []Target{starshipTarget()}, nil)[0]; s.State != "written" {
		t.Fatalf("state = %s (%s)", s.State, s.Error)
	}
	got := readHome(t, ctx, ".config/starship.toml")
	if !strings.HasPrefix(got, "palette = \"aether\"\n") {
		t.Fatalf("got:\n%s", got)
	}
	if strings.Index(got, "palette = ") > strings.Index(got, "# >>> aether palette >>>") {
		t.Fatal("palette line must precede the block")
	}
	if !strings.Contains(got, "[palettes.aether]") || !strings.HasSuffix(got, "# <<< aether palette <<<\n") {
		t.Fatalf("got:\n%s", got)
	}
}

const starshipUser = "\"$schema\" = 'https://starship.rs/config-schema.json'\nadd_newline = true\n\n" +
	"format = \"\"\"\n[╭─](#6b5b7c) $custom$directory\n[╰─](#6b5b7c)$character\"\"\"\n\n" +
	"[character]\nsuccess_symbol = \"[❯](bold #ea8cfb)\"\n"

func TestStarshipInsertsPaletteBeforeFirstTable(t *testing.T) {
	ctx, _ := ichoContext(t)
	writeHome(t, ctx, ".config/starship.toml", starshipUser)
	Apply(ctx, []Target{starshipTarget()}, nil)
	got := readHome(t, ctx, ".config/starship.toml")
	i := strings.Index(got, "palette = \"aether\"")
	if i < 0 || i > strings.Index(got, "[character]") || i < strings.Index(got, "\"\"\"\n\n") {
		t.Fatalf("palette misplaced (multi-line string headers must not count):\n%s", got)
	}
	if !strings.Contains(got, "format = \"\"\"\n[╭─](#6b5b7c) $custom$directory\n[╰─](#6b5b7c)$character\"\"\"\n") {
		t.Fatalf("user content changed:\n%s", got)
	}
}

func TestStarshipReplacesExistingPaletteLine(t *testing.T) {
	ctx, _ := ichoContext(t)
	writeHome(t, ctx, ".config/starship.toml", "palette = \"gruvbox\"\nadd_newline = false\n\n[palettes.gruvbox]\nfg = \"#fff\"\n")
	Apply(ctx, []Target{starshipTarget()}, nil)
	got := readHome(t, ctx, ".config/starship.toml")
	if strings.Contains(got, "gruvbox\"\nadd") || strings.Count(got, "palette = ") != 1 {
		t.Fatalf("got:\n%s", got)
	}
	if !strings.HasPrefix(got, "palette = \"aether\"\nadd_newline = false\n\n[palettes.gruvbox]\nfg = \"#fff\"\n") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestStarshipBlockIdempotent(t *testing.T) {
	ctx, _ := ichoContext(t)
	writeHome(t, ctx, ".config/starship.toml", starshipUser)
	Apply(ctx, []Target{starshipTarget()}, nil)
	first := readHome(t, ctx, ".config/starship.toml")
	if s := Apply(ctx, []Target{starshipTarget()}, nil)[0]; s.State != "unchanged" {
		t.Fatalf("second = %s", s.State)
	}
	if readHome(t, ctx, ".config/starship.toml") != first || strings.Count(first, "# >>> aether palette >>>") != 1 {
		t.Fatalf("not idempotent:\n%s", first)
	}
	// Stale block contents are replaced in place; text after the block stays.
	edited := strings.Replace(first, "[palettes.aether]\n", "[palettes.aether]\nstale = \"x\"\n", 1) + "[after]\nk = 1\n"
	writeHome(t, ctx, ".config/starship.toml", edited)
	Apply(ctx, []Target{starshipTarget()}, nil)
	got := readHome(t, ctx, ".config/starship.toml")
	if strings.Contains(got, "stale") || !strings.HasSuffix(got, "[after]\nk = 1\n") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestStarshipBackupBeforeModify(t *testing.T) {
	ctx, _ := ichoContext(t)
	writeHome(t, ctx, ".config/starship.toml", starshipUser)
	Apply(ctx, []Target{starshipTarget()}, nil)
	b, err := os.ReadFile(filepath.Join(ctx.Home, ".local/state/aether/backups/4000/.config/starship.toml"))
	if err != nil || string(b) != starshipUser {
		t.Fatalf("backup = %q, %v", b, err)
	}
}

func TestBtopReplacesColorTheme(t *testing.T) {
	ctx, _ := ichoContext(t)
	conf := "#? Config\ncolor_theme = \"Default\"\ntheme_background = True\n"
	writeHome(t, ctx, ".config/btop/btop.conf", conf)
	Apply(ctx, []Target{btopTarget()}, nil)
	got := readHome(t, ctx, ".config/btop/btop.conf")
	if got != "#? Config\ncolor_theme = \"aether\"\ntheme_background = True\n" {
		t.Fatalf("got %q", got)
	}
	b, _ := os.ReadFile(filepath.Join(ctx.Home, ".local/state/aether/backups/4000/.config/btop/btop.conf"))
	if string(b) != conf {
		t.Errorf("backup = %q", b)
	}
	if s := Apply(ctx, []Target{btopTarget()}, nil)[0]; s.State != "unchanged" {
		t.Errorf("second = %s", s.State)
	}
}

func TestBtopCreatesAndAppendsColorTheme(t *testing.T) {
	ctx, _ := ichoContext(t)
	Apply(ctx, []Target{btopTarget()}, nil)
	if got := readHome(t, ctx, ".config/btop/btop.conf"); got != "color_theme = \"aether\"\n" {
		t.Fatalf("created = %q", got)
	}
	writeHome(t, ctx, ".config/btop/btop.conf", "update_ms = 2000\n")
	ctx.Backups = NewBackups(ctx.Home, time.Unix(4005, 0))
	Apply(ctx, []Target{btopTarget()}, nil)
	if got := readHome(t, ctx, ".config/btop/btop.conf"); got != "update_ms = 2000\ncolor_theme = \"aether\"\n" {
		t.Fatalf("appended = %q", got)
	}
}

func TestFishReloadArgv(t *testing.T) {
	ctx, _ := ichoContext(t)
	argv := fishTarget().Reload(ctx)
	want := []string{"fish", "-c", "source " + filepath.Join(ctx.Home, ".config/fish/conf.d/aether-colors.fish")}
	if strings.Join(argv, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("argv = %q", argv)
	}
}

func TestIncludesForRofiAndHyprlock(t *testing.T) {
	ctx, _ := ichoContext(t)
	Apply(ctx, []Target{rofiTarget(), hyprlockTarget()}, nil)
	if got := readHome(t, ctx, ".config/rofi/config.rasi"); got != "@import \"aether-colors.rasi\"\n" {
		t.Errorf("rasi = %q", got)
	}
	if got := readHome(t, ctx, ".config/hypr/hyprlock.conf"); got != "source = ~/.config/hypr/aether-hyprlock.conf\n" {
		t.Errorf("hyprlock = %q", got)
	}
}

func TestAppTargetsDetectUsesInjectedLookup(t *testing.T) {
	ctx, rec := ichoContext(t)
	ctx.Have = func(string) bool { return false }
	for _, tg := range []Target{rofiTarget(), btopTarget(), vscodeTarget(), qtTarget(), fishTarget(), starshipTarget(), chromeTarget(), hyprlockTarget()} {
		if s := Apply(ctx, []Target{tg}, nil)[0]; s.State != "not-found" {
			t.Errorf("%s = %s", tg.ID, s.State)
		}
	}
	if len(rec.calls) != 0 {
		t.Errorf("calls %q", rec.calls)
	}
}
