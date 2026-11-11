package cli

import (
	"embed"
	"fmt"
	"os"
	"sort"

	"aether/internal/blueprint"
	"aether/internal/icontheme"
	"aether/internal/platform"
	"aether/internal/theme"
	"aether/internal/wallhaven"
)

func runListBlueprints(args []string) int {
	jsonOut, _ := stripJSON(args)

	svc := blueprint.NewService()
	blueprints, err := svc.LoadAll()
	if err != nil {
		msg := fmt.Sprintf("Error listing blueprints: %v", err)
		if jsonOut {
			return printErrorJSON(msg)
		}
		fmt.Fprintln(os.Stderr, msg)
		return 1
	}

	// Sort by timestamp (newest first)
	sort.Slice(blueprints, func(i, j int) bool {
		return blueprints[i].Timestamp > blueprints[j].Timestamp
	})

	if jsonOut {
		type entry struct {
			Name          string              `json:"name"`
			Colors        []string            `json:"colors"`
			LightMode     bool                `json:"lightMode"`
			Wallpaper     string              `json:"wallpaper,omitempty"`
			WallpaperBlur bool                `json:"wallpaperBlur,omitempty"`
			Timestamp     int64               `json:"timestamp"`
			IconTheme     icontheme.Selection `json:"iconTheme"`
		}
		out := make([]entry, len(blueprints))
		for i, bp := range blueprints {
			iconTheme, err := bp.IconThemeSelection()
			if err != nil {
				return printErrorJSON(fmt.Sprintf("Blueprint %q has invalid iconTheme: %v", bp.Name, err))
			}
			out[i] = entry{
				Name:          bp.Name,
				Colors:        bp.Palette.Colors,
				LightMode:     bp.Palette.LightMode,
				Wallpaper:     bp.Palette.Wallpaper,
				WallpaperBlur: bp.Palette.WallpaperBlur,
				Timestamp:     bp.Timestamp,
				IconTheme:     iconTheme,
			}
		}
		return printJSON(map[string]interface{}{
			"blueprints": out,
			"count":      len(out),
		})
	}

	if len(blueprints) == 0 {
		fmt.Println("No blueprints found.")
		return 0
	}

	for _, bp := range blueprints {
		fmt.Println(bp.Name)
	}
	return 0
}

func runApplyBlueprint(args []string, templatesFS embed.FS) int {
	return runApplyBlueprintWith(args, templatesFS, generateDeps{})
}

func runApplyBlueprintWith(args []string, templatesFS embed.FS, deps generateDeps) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: Blueprint name is required")
		fmt.Fprintln(os.Stderr, "Usage: aether --apply-blueprint <name>")
		return 1
	}

	name := args[0]
	svc := blueprint.NewService()

	bp, err := svc.FindByName(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if bp == nil {
		fmt.Fprintf(os.Stderr, "Error: Blueprint %q not found\n", name)
		fmt.Fprintln(os.Stderr, "\nAvailable blueprints:")
		blueprints, _ := svc.LoadAll()
		if len(blueprints) == 0 {
			fmt.Fprintln(os.Stderr, "  (none)")
		} else {
			for _, b := range blueprints {
				fmt.Fprintf(os.Stderr, "  - %s\n", b.Name)
			}
		}
		return 1
	}

	if !svc.Validate(bp) {
		fmt.Fprintln(os.Stderr, "Error: Blueprint has invalid structure")
		return 1
	}

	wallpaperPath := resolveWallpaperCLI(bp.Palette)
	state, err := stateFromBlueprint(bp, wallpaperPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Blueprint iconTheme: %v\n", err)
		return 1
	}
	tplDir := deps.TemplatesDir
	if tplDir == "" {
		tplDir = "templates"
	}
	writer := theme.NewWriter(templatesFS, tplDir)
	if deps.Run != nil {
		writer.Run = deps.Run
	}
	if deps.Have != nil {
		writer.Have = deps.Have
	}

	// DefaultApplySettings carries the GUI's saved per-target opt-outs.
	settings := theme.DefaultApplySettings()
	fmt.Printf("Applying blueprint: %s\n", bp.Name)
	result, err := writer.ApplyTheme(state, settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to apply blueprint: %v\n", err)
		return 1
	}

	if result.Success {
		fmt.Printf("Applied blueprint: %s\n", bp.Name)
	}
	return 0
}

// resolveWallpaperCLI returns a local wallpaper path from palette data. If the
// local path exists it is returned directly. Otherwise, if a URL is available,
// the wallpaper is downloaded.
func resolveWallpaperCLI(palette blueprint.PaletteData) string {
	if palette.Wallpaper != "" && platform.FileExists(palette.Wallpaper) {
		return palette.Wallpaper
	}
	if palette.WallpaperURL != "" {
		client := wallhaven.NewClient()
		localPath, err := client.DownloadFromURL(palette.WallpaperURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not download wallpaper from %s: %v\n", palette.WallpaperURL, err)
			return ""
		}
		fmt.Printf("Downloaded wallpaper: %s\n", localPath)
		return localPath
	}
	return ""
}

// stateFromBlueprint builds the theme state a blueprint applies, using
// wallpaperPath as the wallpaper (callers decide: the blueprint's own or the
// one being matched). The blueprint slug becomes the theme id and its display
// name (falling back to the slug) the theme name.
func stateFromBlueprint(bp *blueprint.Blueprint, wallpaperPath string) (*theme.ThemeState, error) {
	var palette [16]string
	for i := 0; i < 16 && i < len(bp.Palette.Colors); i++ {
		palette[i] = bp.Palette.Colors[i]
	}

	colorRoles := MapColorsToRoles(palette)
	if v := bp.Palette.ExtendedColors["accent"]; v != "" {
		colorRoles.Accent = v
	}
	if v := bp.Palette.ExtendedColors["cursor"]; v != "" {
		colorRoles.Cursor = v
	}
	if v := bp.Palette.ExtendedColors["selection_foreground"]; v != "" {
		colorRoles.SelectionForeground = v
	}
	if v := bp.Palette.ExtendedColors["selection_background"]; v != "" {
		colorRoles.SelectionBackground = v
	}

	iconTheme, err := bp.IconThemeSelection()
	if err != nil {
		return nil, err
	}
	name := bp.DisplayName
	if name == "" {
		name = bp.Name
	}
	return &theme.ThemeState{
		Palette:          palette,
		WallpaperPath:    wallpaperPath,
		WallpaperBlur:    bp.Palette.WallpaperBlur,
		LightMode:        bp.Palette.LightMode,
		ColorRoles:       colorRoles,
		ExtendedColors:   bp.Palette.ExtendedColors,
		YozakuraRoles:    bp.Palette.YozakuraRoles,
		Glass:            bp.Palette.Glass,
		Clock:            bp.Palette.Clock,
		AppOverrides:     bp.AppOverrides,
		AdditionalImages: bp.Palette.AdditionalImages,
		IconTheme:        iconTheme,
		ThemeName:        name,
		ThemeID:          bp.Name,
	}, nil
}
