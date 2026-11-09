package blueprint

import (
	"reflect"
	"testing"

	"aether/internal/template"
	"aether/internal/yozakura"
)

const curatedFixture = "../yozakura/testdata/curated.json"

func TestImportYozakuraSixThemes(t *testing.T) {
	bps, def, err := ImportYozakura(curatedFixture)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, bp := range bps {
		names = append(names, bp.Name)
	}
	want := []string{"icho", "kocho", "seiya", "sumi", "tsukiyo", "ushio"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	if def != "icho" {
		t.Fatalf("default = %q", def)
	}
}

func findBP(t *testing.T, bps []Blueprint, name string) Blueprint {
	for _, bp := range bps {
		if bp.Name == name {
			return bp
		}
	}
	t.Fatalf("no blueprint %q", name)
	return Blueprint{}
}

func TestImportYozakuraRolesExplicit(t *testing.T) {
	bps, _, _ := ImportYozakura(curatedFixture)
	bp := findBP(t, bps, "icho")
	want := map[string]string{
		"ground": "#120b18", "surface": "#1b1222", "surface2": "#281a31", "border": "#3c2a49",
		"accent": "#ff9ed2", "accent2": "#c39cec", "info": "#8ba2f1", "danger": "#ff5d6c",
		"ok": "#8fd6a8", "warn": "#f2c14e", "text": "#f2e7f0", "muted": "#a08fa8", "accentInk": "#1d0c17",
	}
	if !reflect.DeepEqual(bp.Palette.YozakuraRoles, want) {
		t.Fatalf("roles = %v", bp.Palette.YozakuraRoles)
	}
	if bp.Palette.Glass == nil || *bp.Palette.Glass != 0.9 {
		t.Fatalf("glass = %v", bp.Palette.Glass)
	}
	wc := yozakura.Clock{Digits: "bold", Colon: ":", Glow: 0.35, Ornament: "arc"}
	if bp.Palette.Clock == nil || *bp.Palette.Clock != wc {
		t.Fatalf("clock = %v", bp.Palette.Clock)
	}
	if !reflect.DeepEqual(bp.MatchWallpapers, []string{"evanescia"}) {
		t.Fatalf("match = %v", bp.MatchWallpapers)
	}
}

func TestImportYozakuraPalette(t *testing.T) {
	bps, _, _ := ImportYozakura(curatedFixture)
	bp := findBP(t, bps, "icho")
	term := []string{"#281a31", "#ff5d6c", "#8fd6a8", "#f2c14e", "#8ba2f1", "#ff9ed2", "#86d4e6", "#dacfdc"}
	if len(bp.Palette.Colors) != 16 {
		t.Fatalf("%d colors", len(bp.Palette.Colors))
	}
	if !reflect.DeepEqual(bp.Palette.Colors[:8], term) {
		t.Fatalf("base = %v", bp.Palette.Colors[:8])
	}
	roles := template.ColorRoles{Black: term[0], Red: term[1], Green: term[2], Yellow: term[3], Blue: term[4], Magenta: term[5], Cyan: term[6], White: term[7]}
	th := yozakura.Derive(roles, bp.Palette.YozakuraRoles, bp.Palette.Glass, bp.Palette.Clock)
	if !reflect.DeepEqual(bp.Palette.Colors[8:], th.Term[8:]) {
		t.Fatalf("brights = %v, want %v", bp.Palette.Colors[8:], th.Term[8:])
	}
	if bp.Palette.ExtendedColors["background"] != "#120b18" || bp.Palette.ExtendedColors["foreground"] != "#f2e7f0" {
		t.Fatalf("extended = %v", bp.Palette.ExtendedColors)
	}
}
