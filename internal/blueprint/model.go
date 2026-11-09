package blueprint

import (
	"encoding/json"
	"regexp"

	"aether/internal/icontheme"
	"aether/internal/yozakura"
)

// Blueprint represents a saved theme configuration.
type Blueprint struct {
	Name string `json:"name"`
	// DisplayName is the human-readable theme name ("Ichō"); Name is the slug.
	DisplayName  string                       `json:"displayName,omitempty"`
	Palette      PaletteData                  `json:"palette"`
	Adjustments  map[string]float64           `json:"adjustments,omitempty"`
	AppOverrides map[string]map[string]string `json:"appOverrides,omitempty"`
	Settings     Settings                     `json:"settings,omitempty"`
	IconTheme    *icontheme.Selection         `json:"iconTheme,omitempty"`
	Timestamp    int64                        `json:"timestamp"`
	// MatchWallpapers lists case-insensitive filename substrings; a wallpaper
	// whose base name contains one of them selects this blueprint.
	MatchWallpapers []string `json:"matchWallpapers,omitempty"`
	// Metadata (not persisted in the JSON, populated on load)
	Path     string `json:"-"`
	Filename string `json:"-"`
}

// IconThemeSelection returns the normalized blueprint choice. A missing field
// is the backward-compatible Automatic selection.
func (b *Blueprint) IconThemeSelection() (icontheme.Selection, error) {
	if b == nil || b.IconTheme == nil {
		return icontheme.Automatic(), nil
	}
	return icontheme.NormalizeSelection(*b.IconTheme)
}

// SetIconThemeSelection validates and stores the canonical blueprint encoding:
// Automatic is omitted, while Explicit is written as theme content.
func (b *Blueprint) SetIconThemeSelection(selection icontheme.Selection) error {
	normalized, err := icontheme.NormalizeSelection(selection)
	if err != nil {
		return err
	}
	if normalized.Mode == icontheme.SelectionAutomatic {
		b.IconTheme = nil
		return nil
	}
	b.IconTheme = &normalized
	return nil
}

// UnmarshalJSON removes state for integrations no longer supported by Aether.
func (b *Blueprint) UnmarshalJSON(data []byte) error {
	type Alias Blueprint
	var decoded Alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*b = Blueprint(decoded)
	delete(b.AppOverrides, "gtk")
	return nil
}

// PaletteData holds the palette colors and associated data.
type PaletteData struct {
	Colors           []string          `json:"colors"`
	Wallpaper        string            `json:"wallpaper,omitempty"`
	WallpaperBlur    bool              `json:"wallpaperBlur,omitempty"`
	WallpaperURL     string            `json:"wallpaperUrl,omitempty"`
	LightMode        bool              `json:"lightMode,omitempty"`
	Mode             string            `json:"mode,omitempty"` // "light"/"dark"/"" — preserves the three-valued mode LightMode collapses
	LockedColors     []int             `json:"lockedColors,omitempty"`
	ExtendedColors   map[string]string `json:"extendedColors,omitempty"`
	YozakuraRoles    map[string]string `json:"yozakuraRoles,omitempty"`
	Glass            *float64          `json:"glass,omitempty"`
	Clock            *yozakura.Clock   `json:"clock,omitempty"`
	AdditionalImages []string          `json:"additionalImages,omitempty"`
	WallpaperSource  string            `json:"wallpaperSource,omitempty"`
}

// UnmarshalJSON handles both formats for lockedColors:
//   - []int  — indices of locked colors, e.g. [0, 15]
//   - []bool — positional booleans, e.g. [false, true, ...] → [1]
func (p *PaletteData) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid infinite recursion.
	type Alias PaletteData
	raw := struct {
		Alias
		RawLocked    json.RawMessage   `json:"lockedColors,omitempty"`
		LegacyNative map[string]string `json:"nativeColors,omitempty"`
	}{}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = PaletteData(raw.Alias)

	// Legacy blueprints stored roles as nativeColors; honour them only when
	// yozakuraRoles is absent, keeping valid Yozakura keys.
	if len(p.YozakuraRoles) == 0 && len(raw.LegacyNative) > 0 {
		p.YozakuraRoles = FilterYozakuraRoles(raw.LegacyNative)
		if len(p.YozakuraRoles) == 0 {
			p.YozakuraRoles = nil
		}
	}

	if len(raw.RawLocked) == 0 || string(raw.RawLocked) == "null" {
		return nil
	}

	// Try []int first (the canonical format).
	var ints []int
	if err := json.Unmarshal(raw.RawLocked, &ints); err == nil {
		p.LockedColors = ints
		return nil
	}

	// Fall back to []bool (exported blueprint format).
	var bools []bool
	if err := json.Unmarshal(raw.RawLocked, &bools); err == nil {
		p.LockedColors = nil
		for i, locked := range bools {
			if locked {
				p.LockedColors = append(p.LockedColors, i)
			}
		}
		return nil
	}

	// Neither format — ignore the field rather than failing the whole import.
	p.LockedColors = nil
	return nil
}

// Settings holds theme generation settings.
type Settings struct {
	IncludeNeovim        *bool  `json:"includeNeovim,omitempty"`
	IncludeZed           *bool  `json:"includeZed,omitempty"`
	IncludeVscode        *bool  `json:"includeVscode,omitempty"`
	SelectedNeovimConfig string `json:"selectedNeovimConfig,omitempty"`
}

var hexRolePattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// FilterYozakuraRoles keeps only Yozakura role keys with valid #rrggbb values.
func FilterYozakuraRoles(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for _, k := range yozakura.Keys {
		if v, ok := in[k]; ok && hexRolePattern.MatchString(v) {
			out[k] = v
		}
	}
	return out
}
