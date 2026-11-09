package blueprint

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"aether/internal/template"
	"aether/internal/yozakura"
)

type curatedTheme struct {
	Match []string        `json:"match"`
	Glass *float64        `json:"glass"`
	Clock *yozakura.Clock `json:"clock"`
	Term  []string        `json:"term"`
}

// ImportYozakura converts a curated Yozakura themes.json into blueprints
// (one per theme id, sorted by id) and returns the file's default theme id.
// The display name from the file is not stored; Blueprint.Name is the id.
func ImportYozakura(path string) (bps []Blueprint, defaultName string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var doc struct {
		Default string                     `json:"default"`
		Themes  map[string]json.RawMessage `json:"themes"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, "", fmt.Errorf("parse %s: %w", path, err)
	}
	if len(doc.Themes) == 0 {
		return nil, "", fmt.Errorf("%s has no themes", path)
	}
	ids := make([]string, 0, len(doc.Themes))
	for id := range doc.Themes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		var t curatedTheme
		if err := json.Unmarshal(doc.Themes[id], &t); err != nil {
			return nil, "", fmt.Errorf("theme %s: %w", id, err)
		}
		var flat map[string]any
		_ = json.Unmarshal(doc.Themes[id], &flat)
		raw := map[string]string{}
		for _, k := range yozakura.Keys {
			if v, ok := flat[k].(string); ok {
				raw[k] = v
			}
		}
		rolesMap := FilterYozakuraRoles(raw)
		if len(rolesMap) != len(yozakura.Keys) {
			return nil, "", fmt.Errorf("theme %s: want %d valid roles, got %d", id, len(yozakura.Keys), len(rolesMap))
		}
		if len(t.Term) < 8 {
			return nil, "", fmt.Errorf("theme %s: term has %d colors; want 8", id, len(t.Term))
		}
		roles := template.ColorRoles{
			Background: rolesMap["ground"], Foreground: rolesMap["text"],
			Black: t.Term[0], Red: t.Term[1], Green: t.Term[2], Yellow: t.Term[3],
			Blue: t.Term[4], Magenta: t.Term[5], Cyan: t.Term[6], White: t.Term[7],
		}
		th := yozakura.Derive(roles, rolesMap, t.Glass, t.Clock)
		colors := make([]string, 0, 16)
		colors = append(colors, t.Term[:8]...)
		colors = append(colors, th.Term[8:]...)

		bp := Blueprint{
			Name: id,
			Palette: PaletteData{
				Colors:         colors,
				ExtendedColors: map[string]string{"background": rolesMap["ground"], "foreground": rolesMap["text"]},
				YozakuraRoles:  rolesMap,
				Glass:          t.Glass,
				Clock:          t.Clock,
			},
			MatchWallpapers: t.Match,
		}
		if err := validateBlueprint(&bp); err != nil {
			return nil, "", fmt.Errorf("theme %s: %w", id, err)
		}
		bps = append(bps, bp)
	}
	return bps, doc.Default, nil
}
