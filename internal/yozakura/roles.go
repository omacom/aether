// Package yozakura derives the colour roles of the Yozakura Quickshell shell
// from Aether's 16-colour palette.
package yozakura

import (
	"math"
	"sort"

	"aether/internal/color"
	"aether/internal/template"
)

// Keys lists the 13 Yozakura colour roles in canonical order.
var Keys = [13]string{"ground", "surface", "surface2", "border", "accent", "accent2", "info", "danger", "ok", "warn", "text", "muted", "accentInk"}

// Surface ladder mix ratios (background -> foreground, OKLab), tuned against
// the curated fixtures in testdata/curated.json.
const (
	surfaceRatio  = 0.05
	surface2Ratio = 0.125
	borderRatio   = 0.205

	mutedRatio       = 0.40 // text -> border
	minAccent2Hue    = 40.0 // degrees
	defaultGlass     = 0.92
	brightBlackMix   = 0.55
	brightTowardMix  = 0.18
	inkContrastGoal  = 7.0
	inkDarkLimit     = 0.15
	inkLightLimit    = 0.97
	inkChromaScale   = 0.5
	inkDarkThreshold = 0.6
)

// Clock holds the shell clock styling.
type Clock struct {
	Digits   string  `json:"digits"`
	Colon    string  `json:"colon"`
	Glow     float64 `json:"glow"`
	Ornament string  `json:"ornament"`
}

// Theme is the full derived Yozakura theme.
type Theme struct {
	Roles    map[string]string // all 13 keys, "#rrggbb"
	Explicit map[string]bool   // keys supplied by overrides
	Glass    float64
	Clock    Clock
	Term     [16]string
}

// DefaultClock returns the default clock styling.
func DefaultClock() Clock {
	return Clock{Digits: "bold", Colon: ":", Glow: 0.35, Ornament: "arc"}
}

// Mix blends a toward b by t in OKLab space.
func Mix(a, b string, t float64) string {
	la, lb := color.HexToOKLab(a), color.HexToOKLab(b)
	return color.OKLabToHex(color.OKLab{
		L: la.L + (lb.L-la.L)*t,
		A: la.A + (lb.A-la.A)*t,
		B: la.B + (lb.B-la.B)*t,
	})
}

// mixSRGB blends a toward b by t in sRGB space (parity with the old term16).
func mixSRGB(a, b string, t float64) string {
	ra, rb := color.HexToRGB(a), color.HexToRGB(b)
	return color.RGBToHex(ra.R+(rb.R-ra.R)*t, ra.G+(rb.G-ra.G)*t, ra.B+(rb.B-ra.B)*t)
}

// AccentCandidates returns ANSI 1-6 and 9-14 ranked by OKLCH chroma
// (descending), deduplicated by hex.
func AccentCandidates(roles template.ColorRoles) []string {
	all := []string{
		roles.Red, roles.Green, roles.Yellow, roles.Blue, roles.Magenta, roles.Cyan,
		roles.BrightRed, roles.BrightGreen, roles.BrightYellow, roles.BrightBlue, roles.BrightMagenta, roles.BrightCyan,
	}
	seen := map[string]bool{}
	var out []string
	for _, h := range all {
		if !color.IsHexColor(h) || seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return color.HexToOKLCH(out[i]).C > color.HexToOKLCH(out[j]).C
	})
	return out
}

func hueGap(a, b string) float64 {
	d := math.Abs(color.HexToOKLCH(a).H - color.HexToOKLCH(b).H)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func accentInk(accent string) string {
	lch := color.HexToOKLCH(accent)
	target, extreme := inkLightLimit, 1.0
	if lch.L > inkDarkThreshold {
		target, extreme = inkDarkLimit, 0.0
	}
	out := color.OKLCHToHex(color.OKLCH{L: target, C: lch.C * inkChromaScale, H: lch.H})
	if color.ContrastRatio(out, accent) < inkContrastGoal {
		out = color.OKLCHToHex(color.OKLCH{L: extreme, C: 0, H: lch.H})
	}
	return out
}

func firstValid(vals ...string) string {
	for _, v := range vals {
		if color.IsHexColor(v) {
			return v
		}
	}
	return ""
}

// Derive computes the Yozakura theme from Aether roles. Overrides (role key ->
// hex) always win and are recorded in Explicit. nil glass/clock use defaults.
func Derive(roles template.ColorRoles, overrides map[string]string, glass *float64, clock *Clock) Theme {
	r := map[string]string{}
	explicit := map[string]bool{}
	for _, k := range Keys {
		if v, ok := overrides[k]; ok && color.IsHexColor(v) {
			r[k] = v
			explicit[k] = true
		}
	}
	set := func(k, v string) {
		if !explicit[k] {
			r[k] = v
		}
	}

	set("ground", roles.Background)
	set("text", roles.Foreground)
	set("info", roles.Blue)
	set("danger", roles.Red)
	set("ok", roles.Green)
	set("warn", roles.Yellow)

	set("surface", Mix(r["ground"], r["text"], surfaceRatio))
	set("surface2", Mix(r["ground"], r["text"], surface2Ratio))
	set("border", Mix(r["ground"], r["text"], borderRatio))

	cands := AccentCandidates(roles)
	if !explicit["accent"] {
		acc := roles.Accent
		if !color.IsHexColor(acc) && len(cands) > 0 {
			acc = cands[0]
		}
		r["accent"] = acc
	}
	if !explicit["accent2"] {
		a2 := ""
		for _, c := range cands {
			if c != r["accent"] && hueGap(c, r["accent"]) >= minAccent2Hue {
				a2 = c
				break
			}
		}
		if a2 == "" {
			a2 = firstValid(roles.Magenta, roles.Blue)
		}
		r["accent2"] = a2
	}

	set("muted", Mix(r["text"], r["border"], mutedRatio))
	set("accentInk", accentInk(r["accent"]))

	th := Theme{Roles: r, Explicit: explicit, Glass: defaultGlass, Clock: DefaultClock()}
	if glass != nil {
		th.Glass = *glass
	}
	if clock != nil {
		th.Clock = *clock
	}

	base := [8]string{roles.Black, roles.Red, roles.Green, roles.Yellow, roles.Blue, roles.Magenta, roles.Cyan, roles.White}
	copy(th.Term[:8], base[:])
	th.Term[8] = mixSRGB(base[0], r["muted"], brightBlackMix)
	for i := 1; i <= 6; i++ {
		th.Term[8+i] = mixSRGB(base[i], "#ffffff", brightTowardMix)
	}
	th.Term[15] = r["text"]
	return th
}
