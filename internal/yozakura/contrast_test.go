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
