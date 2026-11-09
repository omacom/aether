package yozakura

import (
	"encoding/json"
	"os"
	"testing"

	"aether/internal/color"
)

func pairByName(t *testing.T, rs []PairResult, name string) PairResult {
	t.Helper()
	for _, r := range rs {
		if r.Pair == name {
			return r
		}
	}
	t.Fatalf("pair %s missing", name)
	return PairResult{}
}

func TestEnforceNudgesDerivedMuted(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	th.Roles["muted"] = Mix(th.Roles["text"], th.Roles["surface"], 0.8)
	before := th.Roles["muted"]
	if cr := color.ContrastRatio(before, th.Roles["surface"]); cr >= 4.5 {
		t.Fatalf("setup: muted already %.2f", cr)
	}
	res := Enforce(&th)
	if len(res) != 7 {
		t.Fatalf("want 7 pairs, got %d", len(res))
	}
	if cr := color.ContrastRatio(th.Roles["muted"], th.Roles["surface"]); cr < 4.5 {
		t.Errorf("muted/surface %.2f < 4.5", cr)
	}
	if cr := color.ContrastRatio(th.Roles["muted"], th.Roles["surface2"]); cr < 4.0 {
		t.Errorf("muted/surface2 %.2f < 4.0", cr)
	}
	if s := pairByName(t, res, "muted/surface").State; s != "nudged" {
		t.Errorf("state %s", s)
	}
	if d := hueGap(before, th.Roles["muted"]); d > 2 {
		t.Errorf("hue moved %.2f %s -> %s", d, before, th.Roles["muted"])
	}
	if th.Term[8] != mixSRGB(th.Term[0], th.Roles["muted"], brightBlackMix) || th.Term[15] != th.Roles["text"] {
		t.Errorf("Term not synced")
	}
}

func TestEnforceNeverTouchesExplicit(t *testing.T) {
	bad := "#2a2a3a"
	th := Derive(mapRoles(defaultPalette), map[string]string{"accent": bad}, nil, nil)
	res := Enforce(&th)
	if th.Roles["accent"] != bad {
		t.Errorf("explicit accent changed to %s", th.Roles["accent"])
	}
	if s := pairByName(t, res, "accent/surface").State; s != "failed" {
		t.Errorf("state %s", s)
	}
}

func TestEnforceBackgroundNotMoved(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	th.Roles["muted"] = th.Roles["surface"]
	g, s, s2 := th.Roles["ground"], th.Roles["surface"], th.Roles["surface2"]
	Enforce(&th)
	if th.Roles["ground"] != g || th.Roles["surface"] != s || th.Roles["surface2"] != s2 {
		t.Errorf("background roles moved")
	}
}

func TestEnforceGivesUpAfter20Steps(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	th.Roles["text"] = th.Roles["surface"]
	res := Enforce(&th)
	if s := pairByName(t, res, "text/surface").State; s != "failed" {
		t.Errorf("state %s", s)
	}
}

func TestCheckReadOnly(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	th.Roles["muted"] = th.Roles["surface"]
	before := th.Roles["muted"]
	res := Check(th)
	if th.Roles["muted"] != before {
		t.Error("Check mutated theme")
	}
	if s := pairByName(t, res, "muted/surface").State; s != "failed" {
		t.Errorf("state %s", s)
	}
}

// Curated themes that fail the Python check() today (none expected).
var knownCuratedFailures = map[string]bool{}

func TestCheckCuratedThemes(t *testing.T) {
	raw, err := os.ReadFile("testdata/curated.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Themes map[string]map[string]any `json:"themes"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for id, m := range f.Themes {
		th := Theme{Roles: map[string]string{}}
		for _, k := range Keys {
			s, _ := m[k].(string)
			th.Roles[k] = s
		}
		for _, r := range Check(th) {
			t.Logf("%s %-16s %5.2f min %.1f %s", id, r.Pair, r.Ratio, r.Min, r.State)
			if r.State == "failed" && !knownCuratedFailures[id] {
				t.Errorf("%s %s %.2f < %.1f", id, r.Pair, r.Ratio, r.Min)
			}
		}
	}
}

func TestEnforceRederivesInkAfterAccentNudge(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	// Mid-tone surface: a light accent is nudged darker, so the dark ink
	// derived from the pre-nudge accent must flip to a light one.
	th.Roles["surface"] = color.OKLabToHex(color.OKLab{L: 0.62})
	acc := color.OKLCHToHex(color.OKLCH{L: 0.66, C: 0.1, H: 280})
	th.Roles["accent"] = acc
	th.Roles["accentInk"] = accentInk(acc)
	origInk := th.Roles["accentInk"]
	res := Enforce(&th)
	if th.Roles["accent"] == acc {
		t.Fatal("setup: accent not nudged")
	}
	if color.ContrastRatio(origInk, th.Roles["accent"]) >= 7.0 {
		t.Fatal("setup: original ink still passes")
	}
	r := pairByName(t, res, "accentInk/accent")
	if r.State != "nudged" || color.ContrastRatio(th.Roles["accentInk"], th.Roles["accent"]) < 7.0 {
		t.Errorf("accent %s->%s ink %s->%s: %+v", acc, th.Roles["accent"], origInk, th.Roles["accentInk"], r)
	}
}

func TestNudgeDirectionByContrastGain(t *testing.T) {
	th := Derive(mapRoles(defaultPalette), nil, nil, nil)
	bg := color.OKLabToHex(color.OKLab{L: 0.62})
	if nudgeDirection(bg) != -1 {
		t.Fatalf("setup: expected darker direction for %s", bg)
	}
	th.Roles["surface"] = bg
	th.Roles["danger"] = Mix(bg, "#000000", 0.1)
	fg0 := th.Roles["danger"]
	res := Enforce(&th)
	r := pairByName(t, res, "danger/surface")
	if r.State != "nudged" {
		t.Errorf("state %s ratio %.2f", r.State, r.Ratio)
	}
	if color.HexToOKLab(th.Roles["danger"]).L >= color.HexToOKLab(fg0).L {
		t.Errorf("did not move darker: %s -> %s", fg0, th.Roles["danger"])
	}
}
