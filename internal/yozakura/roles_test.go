package yozakura

import (
	"encoding/json"
	"os"
	"testing"

	"aether/internal/color"
	"aether/internal/template"
)

// Copy of theme.DefaultPalette (cannot import internal/theme: import cycle).
var defaultPalette = [16]string{
	"#1e1e2e", "#f38ba8", "#a6e3a1", "#f9e2af", "#89b4fa", "#cba6f7", "#94e2d5", "#cdd6f4",
	"#45475a", "#f38ba8", "#a6e3a1", "#f9e2af", "#89b4fa", "#cba6f7", "#94e2d5", "#ffffff",
}

// mapRoles mirrors cli.MapColorsToRoles.
func mapRoles(p [16]string) template.ColorRoles {
	return template.ColorRoles{
		Background: p[0], Foreground: p[7],
		Black: p[0], Red: p[1], Green: p[2], Yellow: p[3], Blue: p[4], Magenta: p[5], Cyan: p[6], White: p[7],
		BrightBlack: p[8], BrightRed: p[9], BrightGreen: p[10], BrightYellow: p[11],
		BrightBlue: p[12], BrightMagenta: p[13], BrightCyan: p[14], BrightWhite: p[15],
		Accent: p[4], Cursor: p[7], SelectionForeground: p[0], SelectionBackground: p[7],
	}
}

func TestDeriveFillsAllKeys(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	for _, k := range Keys {
		if !color.IsHexColor(th.Roles[k]) {
			t.Errorf("role %s = %q is not a hex colour", k, th.Roles[k])
		}
	}
}

func TestDeriveGroundTextAreBackgroundForeground(t *testing.T) {
	r := mapRoles(defaultPalette)
	th := Derive(r, nil, nil, nil)
	checks := map[string]string{
		"ground": r.Background, "text": r.Foreground,
		"info": r.Blue, "danger": r.Red, "ok": r.Green, "warn": r.Yellow,
	}
	for k, want := range checks {
		if th.Roles[k] != want {
			t.Errorf("%s = %s, want %s", k, th.Roles[k], want)
		}
	}
}

func TestDeriveSurfaceLadder(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	prev := -1.0
	for _, k := range []string{"ground", "surface", "surface2", "border"} {
		l := color.HexToOKLab(th.Roles[k]).L
		if l <= prev {
			t.Errorf("%s L=%f not above previous %f", k, l, prev)
		}
		prev = l
	}
}

func TestDeriveAccent2HueSeparation(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	if g := hueGap(th.Roles["accent"], th.Roles["accent2"]); g < 40 {
		t.Errorf("hue gap %f < 40", g)
	}
}

func TestDeriveOverridesWinAndAreMarked(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), map[string]string{"accent": "#ff9ed2"}, nil, nil)
	if th.Roles["accent"] != "#ff9ed2" || !th.Explicit["accent"] || th.Explicit["muted"] {
		t.Errorf("override handling wrong: %v %v", th.Roles["accent"], th.Explicit)
	}
}

func TestDeriveTermBrights(t *testing.T) {
	r := mapRoles(defaultPalette)
	th := Derive(r, nil, nil, nil)
	if th.Term[8] != mixSRGB(defaultPalette[0], th.Roles["muted"], 0.55) {
		t.Errorf("Term[8] = %s", th.Term[8])
	}
	for i := 1; i <= 6; i++ {
		if want := mixSRGB(defaultPalette[i], "#ffffff", 0.18); th.Term[8+i] != want {
			t.Errorf("Term[%d] = %s, want %s", 8+i, th.Term[8+i], want)
		}
	}
	if th.Term[15] != th.Roles["text"] {
		t.Errorf("Term[15] = %s", th.Term[15])
	}
	for i := 0; i < 8; i++ {
		if th.Term[i] != defaultPalette[i] {
			t.Errorf("Term[%d] = %s", i, th.Term[i])
		}
	}
}

type curatedTheme struct {
	Ground, Surface, Surface2, Border, Text, Muted string
	Term                                           [8]string
}

func TestDeriveCuratedSurfaceTolerance(t *testing.T) {
	raw, err := os.ReadFile("testdata/curated.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Themes map[string]curatedTheme `json:"themes"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Themes) == 0 {
		t.Fatal("no curated themes")
	}
	// Border cannot reach 0.04: curated borders are more chromatic than any
	// background->foreground mix (best single-ratio worst case is 0.0465).
	tol := map[string]float64{"surface": 0.04, "surface2": 0.04, "border": 0.05}
	for id, c := range f.Themes {
		var p [16]string
		copy(p[:8], c.Term[:])
		for i := 1; i <= 6; i++ {
			p[8+i] = mixSRGB(c.Term[i], "#ffffff", 0.18)
		}
		p[8] = mixSRGB(c.Term[0], c.Muted, 0.55)
		p[15] = c.Text
		r := mapRoles(p)
		r.Background, r.Foreground, r.Accent = c.Ground, c.Text, ""
		th := Derive(r, nil, nil, nil)
		for _, k := range []string{"surface", "surface2", "border"} {
			want := map[string]string{"surface": c.Surface, "surface2": c.Surface2, "border": c.Border}[k]
			d := color.OKLabDistance(color.HexToOKLab(th.Roles[k]), color.HexToOKLab(want))
			t.Logf("%s %s: derived %s curated %s dist %.4f", id, k, th.Roles[k], want, d)
			if d > tol[k] {
				t.Errorf("%s %s distance %.4f > %.2f", id, k, d, tol[k])
			}
		}
	}
}

func TestDeriveDefaultsGlassClock(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	if th.Glass != 0.92 || th.Clock != DefaultClock() {
		t.Errorf("glass=%v clock=%+v", th.Glass, th.Clock)
	}
	g := 0.8
	c := Clock{Digits: "medium", Colon: "·", Glow: 0.5, Ornament: "arc"}
	th = Derive(mapRoles(defaultPalette), nil, &g, &c)
	if th.Glass != 0.8 || th.Clock != c {
		t.Errorf("explicit glass/clock not honoured")
	}
}
