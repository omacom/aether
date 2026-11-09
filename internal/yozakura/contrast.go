package yozakura

import (
	"math"

	"aether/internal/color"
)

// PairResult is the contrast outcome for one foreground/background pair.
type PairResult struct {
	Pair  string  `json:"pair"`
	Ratio float64 `json:"ratio"`
	Min   float64 `json:"min"`
	State string  `json:"state"` // "ok" | "nudged" | "failed"
}

const (
	nudgeStep     = 0.02
	nudgeMaxSteps = 20
	hueTolerance  = 1.0  // degrees
	minHueChroma  = 0.02 // below this, hue is dominated by hex quantisation
)

type contrastPair struct {
	name, fg, bg string
	min          float64
}

// contrastPairs is the spec §3 table, in order.
var contrastPairs = []contrastPair{
	{"text/surface", "text", "surface", 7.0},
	{"muted/surface", "muted", "surface", 4.5},
	{"muted/surface2", "muted", "surface2", 4.0},
	{"accent/surface", "accent", "surface", 4.5},
	{"accent2/surface", "accent2", "surface", 4.5},
	{"danger/surface", "danger", "surface", 4.5},
	{"accentInk/accent", "accentInk", "accent", 7.0},
}

// syncTerm recomputes the palette slots that mirror muted and text.
func syncTerm(t *Theme) {
	t.Term[8] = mixSRGB(t.Term[0], t.Roles["muted"], brightBlackMix)
	t.Term[15] = t.Roles["text"]
}

// inGamutKeepHue converts lch to hex; when sRGB clamping would visibly rotate
// the hue, chroma is reduced until the hue is within hueTolerance degrees.
func inGamutKeepHue(lch color.OKLCH) string {
	hex := color.OKLCHToHex(lch)
	if lch.C < minHueChroma {
		return hex
	}
	for c := lch.C; c > minHueChroma; c *= 0.9 {
		hex = color.OKLCHToHex(color.OKLCH{L: lch.L, C: c, H: lch.H})
		d := math.Abs(color.HexToOKLCH(hex).H - lch.H)
		if d > 180 {
			d = 360 - d
		}
		if d <= hueTolerance {
			return hex
		}
	}
	return hex
}

// nudge steps the foreground's OKLab L away from the background's until the
// pair meets min, giving up after nudgeMaxSteps or once gamut clamping stops
// changing the colour. It returns the (possibly unchanged) foreground.
func nudge(fg, bg string, min float64) string {
	dir := -1.0
	if color.HexToOKLab(bg).L < 0.5 {
		dir = 1.0
	}
	cur := fg
	lch := color.HexToOKLCH(fg)
	for i := 0; i < nudgeMaxSteps && color.ContrastRatio(cur, bg) < min; i++ {
		lch.L += dir * nudgeStep
		next := inGamutKeepHue(lch)
		if next == cur {
			break
		}
		cur = next
	}
	return cur
}

// Enforce nudges derived (non-explicit) foreground roles until each pair meets
// its minimum contrast. Backgrounds and explicit roles are never moved.
func Enforce(t *Theme) []PairResult {
	orig := map[string]string{}
	for k, v := range t.Roles {
		orig[k] = v
	}
	for _, p := range contrastPairs {
		if t.Explicit[p.fg] {
			continue
		}
		if color.ContrastRatio(t.Roles[p.fg], t.Roles[p.bg]) < p.min {
			t.Roles[p.fg] = nudge(t.Roles[p.fg], t.Roles[p.bg], p.min)
		}
	}
	syncTerm(t)
	out := evaluate(*t)
	for i, p := range contrastPairs {
		if out[i].State == "ok" && t.Roles[p.fg] != orig[p.fg] {
			out[i].State = "nudged"
		}
	}
	return out
}

// Check reports contrast for every pair without modifying the theme.
func Check(t Theme) []PairResult { return evaluate(t) }

func evaluate(t Theme) []PairResult {
	out := make([]PairResult, len(contrastPairs))
	for i, p := range contrastPairs {
		r := color.ContrastRatio(t.Roles[p.fg], t.Roles[p.bg])
		st := "ok"
		if r < p.min {
			st = "failed"
		}
		out[i] = PairResult{Pair: p.name, Ratio: r, Min: p.min, State: st}
	}
	return out
}
