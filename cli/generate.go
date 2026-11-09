package cli

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"aether/internal/blueprint"
	"aether/internal/extraction"
	"aether/internal/icontheme"
	"aether/internal/platform"
	"aether/internal/targets"
	"aether/internal/theme"
)

func parseIconThemeOption(args []string) (icontheme.Selection, []string, error) {
	selection := icontheme.Automatic()
	remaining := make([]string, 0, len(args))
	found := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg != "--icon-theme" {
			remaining = append(remaining, arg)
			continue
		}
		if found {
			return icontheme.Selection{}, args, fmt.Errorf("--icon-theme may only be specified once")
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			return icontheme.Selection{}, args, fmt.Errorf("--icon-theme requires automatic or a theme ID")
		}

		value := args[i+1]
		if value != "automatic" {
			var err error
			selection, err = icontheme.NormalizeSelection(icontheme.Selection{
				Mode: icontheme.SelectionExplicit,
				ID:   value,
			})
			if err != nil {
				return icontheme.Selection{}, args, err
			}
		}

		found = true
		i++
	}

	return selection, remaining, nil
}

// generateDeps are the injectable collaborators of the generate pipeline.
// Zero values select the production behaviour.
type generateDeps struct {
	Extract  func(path string, lightMode bool, mode string) ([16]string, error)
	Run      targets.Runner
	Have     func(bin string) bool
	LockPath string
	// TemplatesDir is the root inside the template FS (default "templates").
	TemplatesDir string
	// OnState, if set, observes the theme state just before it is applied.
	OnState func(*theme.ThemeState)
	Stdout  io.Writer
	Stderr  io.Writer
}

func runGenerate(args []string, templatesFS embed.FS) int {
	return runGenerateWith(args, templatesFS, generateDeps{})
}

func runGenerateWith(args []string, templatesFS embed.FS, deps generateDeps) int {
	stdout, stderr := deps.Stdout, deps.Stderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	extract := deps.Extract
	if extract == nil {
		extract = extraction.ExtractColors
	}

	for _, arg := range args {
		if arg == "--gtk" || arg == "--no-gtk" {
			fmt.Fprintf(stderr, "Error: Unknown option: %s\n", arg)
			return 1
		}
	}
	iconThemeSet := slices.Contains(args, "--icon-theme")
	iconTheme, args, err := parseIconThemeOption(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Invalid icon theme: %v\n", err)
		return 1
	}

	// Parse flags
	mode, args := parseFlag(args, "--extract-mode")
	if mode == "" {
		mode = "normal"
	}
	lightMode, args := hasFlag(args, "--light-mode")
	noApply, args := hasFlag(args, "--no-apply")
	jsonOut, args := hasFlag(args, "--json")
	outputPath, args := parseFlag(args, "--output")

	// Editor integrations are opt-out and match the GUI defaults.
	noZed, args := hasFlag(args, "--no-zed")
	noVscode, args := hasFlag(args, "--no-vscode")
	noNeovim, args := hasFlag(args, "--no-neovim")

	// Human-readable progress goes to stdout, or to stderr when stdout is
	// reserved for the --json object.
	var out io.Writer = stdout
	if jsonOut {
		out = stderr
	}

	if len(args) == 0 {
		fmt.Fprintln(stderr, "Error: Wallpaper path is required")
		fmt.Fprintln(stderr, "Usage: aether --generate <wallpaper> [--extract-mode <mode>] [--light-mode] [--no-apply] [--json] [--output <path>] [--icon-theme automatic|<ID>] [--no-zed] [--no-vscode] [--no-neovim]")
		return 1
	}
	wallpaperPath := args[0]

	// Validate mode
	if !validModes[mode] {
		fmt.Fprintf(stderr, "Error: Invalid extraction mode: %s\n", mode)
		fmt.Fprintf(stderr, "Valid modes: normal, monochromatic, analogous, pastel, material, colorful, muted, bright\n")
		return 1
	}
	if jsonOut && noApply {
		fmt.Fprintln(stderr, "Error: --json requires the theme to be applied (incompatible with --no-apply)")
		return 1
	}

	// Expand paths
	wallpaperPath = expandHome(wallpaperPath)
	if outputPath != "" {
		outputPath = expandHome(outputPath)
	}

	// Validate file exists
	if _, err := os.Stat(wallpaperPath); os.IsNotExist(err) {
		fmt.Fprintf(stderr, "Error: Wallpaper file not found: %s\n", wallpaperPath)
		return 1
	}
	if !theme.IsImageFile(wallpaperPath) {
		fmt.Fprintf(stderr, "Error: Unsupported wallpaper file: %s\n", wallpaperPath)
		fmt.Fprintln(stderr, "Aether can extract colors from image files only")
		return 1
	}

	// Serialise applying runs; a newer request supersedes waiting older ones.
	if !noApply {
		lockPath := deps.LockPath
		if lockPath == "" {
			lockPath = platform.GenerateLockPath()
		}
		release, err := platform.AcquireGenerateLock(context.Background(), lockPath)
		switch {
		case errors.Is(err, platform.ErrSuperseded):
			fmt.Fprintf(stderr, "Superseded by a newer --generate request; skipping %s\n", wallpaperPath)
			return 0
		case err != nil:
			fmt.Fprintf(stderr, "Warning: generate lock unavailable, continuing unlocked: %v\n", err)
		default:
			defer release()
		}
	}

	settings := theme.Settings{
		IncludeZed:    !noZed,
		IncludeVscode: !noVscode,
		IncludeNeovim: !noNeovim,
	}
	svc := blueprint.NewService()

	// Lookup order: wallpaper-matched blueprint, extraction, default blueprint.
	var state *theme.ThemeState
	bp, err := svc.FindForWallpaper(wallpaperPath)
	if err != nil {
		fmt.Fprintf(stderr, "Warning: blueprint lookup failed: %v\n", err)
	}
	if bp != nil {
		if state, err = stateFromBlueprint(bp, wallpaperPath); err != nil {
			fmt.Fprintf(stderr, "Warning: blueprint %s unusable (%v); extracting instead\n", bp.Name, err)
			state = nil
		} else {
			fmt.Fprintf(out, "Using blueprint %s for %s\n", bp.Name, wallpaperPath)
		}
	}
	if state == nil {
		msg := fmt.Sprintf("Extracting colors from: %s", wallpaperPath)
		var details []string
		if mode != "normal" {
			details = append(details, mode)
		}
		if lightMode {
			details = append(details, "light mode")
		}
		if noApply {
			details = append(details, "generate only")
		}
		if len(details) > 0 {
			msg += fmt.Sprintf(" (%s)", joinStrings(details, ", "))
		}
		fmt.Fprintln(out, msg)

		palette, err := extract(wallpaperPath, lightMode, mode)
		if err == nil {
			fmt.Fprintln(out, "Extracted 16 colors successfully")
			state = &theme.ThemeState{
				Palette:       palette,
				WallpaperPath: wallpaperPath,
				LightMode:     lightMode,
				ColorRoles:    MapColorsToRoles(palette),
				ThemeID:       "auto",
				ThemeName:     "Auto",
			}
		} else {
			def, derr := svc.Default()
			if derr == nil && def != nil {
				state, derr = stateFromBlueprint(def, wallpaperPath)
			}
			if derr != nil || state == nil {
				fmt.Fprintf(stderr, "Error: Failed to extract colors: %v\n", err)
				return 1
			}
			fmt.Fprintf(stderr, "Warning: extraction failed (%v); falling back to default blueprint %s\n", err, def.Name)
		}
	}
	// A blueprint keeps its own icon theme unless --icon-theme was given.
	if iconThemeSet || state.ThemeID == "auto" {
		state.IconTheme = iconTheme
	}
	if deps.OnState != nil {
		deps.OnState(state)
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

	if noApply {
		fmt.Fprintln(out, "Generating theme files...")
		if err := writer.GenerateOnly(state, settings, outputPath); err != nil {
			fmt.Fprintf(stderr, "Error: Failed to generate theme: %v\n", err)
			return 1
		}
		fmt.Fprintln(out, "Theme files generated successfully")
		return 0
	}

	fmt.Fprintln(out, "Applying theme...")
	result, err := writer.ApplyTheme(state, settings)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Failed to apply theme: %v\n", err)
		return 1
	}
	for _, st := range result.Targets {
		if st.State == "failed" {
			fmt.Fprintf(stderr, "Warning: target %s failed: %s\n", st.ID, st.Error)
		}
	}
	if result.Success {
		fmt.Fprintln(out, "Theme applied successfully")
	}

	if jsonOut {
		if err := printThemeJSON(stdout, result); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
	}
	if !result.Success {
		return 1
	}
	return 0
}

// printThemeJSON writes the yozakura theme.json the apply just produced to w
// as a single compact line, preserving key order and unescaped Unicode.
func printThemeJSON(w io.Writer, result *theme.ApplyResult) error {
	for _, st := range result.Targets {
		if st.ID != "yozakura" {
			continue
		}
		if st.State == "failed" || st.Path == "" {
			return fmt.Errorf("yozakura theme.json was not written: %s", st.Error)
		}
		data, err := os.ReadFile(st.Path)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := json.Compact(&buf, data); err != nil {
			return fmt.Errorf("compact %s: %w", st.Path, err)
		}
		buf.WriteByte('\n')
		_, err = w.Write(buf.Bytes())
		return err
	}
	return fmt.Errorf("yozakura target did not run")
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}
