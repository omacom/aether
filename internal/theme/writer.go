package theme

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"aether/internal/color"
	"aether/internal/icontheme"
	"aether/internal/platform"
	"aether/internal/targets"
	"aether/internal/template"
	"aether/internal/yozakura"
)

// templateAppNameMap maps template file names to standardised app names,
// used for per-app override lookups.
var templateAppNameMap = map[string]string{
	"alacritty.toml":    "alacritty",
	"aether.zed.json":   "zed",
	"btop.theme":        "btop",
	"chromium.theme":    "chromium",
	"foot.ini":          "foot",
	"ghostty.conf":      "ghostty",
	"hyprland.conf":     "hyprland",
	"hyprlock.conf":     "hyprlock",
	"icons.theme":       "icons",
	"kitty.conf":        "kitty",
	"mako.ini":          "mako",
	"neovim.lua":        "neovim",
	"swayosd.css":       "swayosd",
	"walker.css":        "walker",
	"waybar.css":        "waybar",
	"wofi.css":          "wofi",
	"vencord.theme.css": "vencord",
	"warp.yaml":         "warp",
	"colors.toml":       "colors",
}

// getAppNameFromFileName returns the app name for a given template file name.
func getAppNameFromFileName(fileName string) string {
	if name, ok := templateAppNameMap[fileName]; ok {
		return name
	}
	parts := strings.SplitN(fileName, ".", 2)
	return parts[0]
}

// GetAppNameFromFileName is the exported variant of getAppNameFromFileName.
func GetAppNameFromFileName(fileName string) string {
	return getAppNameFromFileName(fileName)
}

// Settings holds the toggles that control which optional templates are
// processed and applied.
type Settings struct {
	IncludeZed           bool            `json:"includeZed"`
	IncludeVscode        bool            `json:"includeVscode"`
	IncludeNeovim        bool            `json:"includeNeovim"`
	SelectedNeovimConfig string          `json:"selectedNeovimConfig"`
	IncludedApps         map[string]bool `json:"includedApps,omitempty"`
	ExcludedApps         map[string]bool `json:"excludedApps,omitempty"`
}

// includesApp supports explicit opt-in targets while preserving callers that
// still use the legacy special flags and exclusion map.
func (s Settings) includesApp(app string) bool {
	if app == "icons" {
		if enabled, ok := s.IncludedApps[app]; ok {
			return enabled
		}
		return !s.ExcludedApps[app]
	}
	if s.IncludedApps != nil {
		return s.IncludedApps[app]
	}
	if s.ExcludedApps[app] {
		return false
	}

	switch app {
	case "zed":
		return s.IncludeZed
	case "vscode":
		return s.IncludeVscode
	case "neovim":
		return s.IncludeNeovim
	default:
		return !s.ExcludedApps[app]
	}
}

// TargetEnabled reports whether a desktop target (see targets.All) should run.
// Every target is enabled unless IncludedApps explicitly sets it to false.
func (s Settings) TargetEnabled(id string) bool {
	if enabled, ok := s.IncludedApps[id]; ok {
		return enabled
	}
	return true
}

// DefaultApplySettings returns the same defaults `aether --generate` uses.
// Use this anywhere the user is applying a theme without picking per-app
// toggles explicitly so a bare `Settings{}` does not silently drop editor
// integrations from the output.
func DefaultApplySettings() Settings {
	return Settings{
		IncludeZed:    true,
		IncludeVscode: true,
		IncludeNeovim: true,
	}
}

// ApplyResult is returned by ApplyTheme with the outcome of theme application.
type ApplyResult struct {
	Success   bool             `json:"success"`
	ThemePath string           `json:"themePath"`
	Targets   []targets.Status `json:"targets"`
}

// Writer processes templates from an embed.FS and generates theme files.
type Writer struct {
	templatesFS  embed.FS
	templatesDir string // root directory name inside the embed.FS (e.g. "templates")

	// Run executes external commands (reloads, queries) for desktop targets.
	// NewWriter sets a real exec-based runner; tests must replace it.
	Run targets.Runner
	// Have reports whether a binary is on PATH; nil uses exec.LookPath.
	Have func(bin string) bool
}

// execRunner is the production targets.Runner: it returns combined output.
// pkill exits 1 when nothing matched, which is not a failure for a reload.
func execRunner(name string, args ...string) (string, error) {
	out, err := platform.Command(name, args...).CombinedOutput()
	if err != nil && name == "pkill" {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return string(out), nil
		}
	}
	return string(out), err
}

// NewWriter creates a Writer that reads templates from the given embed.FS.
// dir is the root directory name inside the FS (typically "templates").
func NewWriter(fsys embed.FS, dir string) *Writer {
	return &Writer{
		templatesFS:  fsys,
		templatesDir: dir,
		Run:          execRunner,
	}
}

// prepareThemeDir stages all media before replacing the backgrounds subdir.
// Returns the wallpaper destination path.
// If no wallpaper or additional images are provided, existing backgrounds are preserved.
func prepareThemeDir(targetDir string, state *ThemeState) (string, error) {
	bgDir := filepath.Join(targetDir, "backgrounds")
	if err := platform.EnsureDir(bgDir); err != nil {
		return "", fmt.Errorf("create theme backgrounds directory: %w", err)
	}
	// Remove Aether-owned output left by versions that supported GTK styling.
	if err := removeLegacyGTKStylesheet(filepath.Join(targetDir, "gtk.css")); err != nil {
		return "", err
	}

	if state.WallpaperPath == "" && state.OriginalWallpaperPath == "" && len(state.AdditionalImages) == 0 {
		return "", nil
	}

	mediaDir, err := os.MkdirTemp(targetDir, ".backgrounds-")
	if err != nil {
		return "", fmt.Errorf("stage theme backgrounds: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			if err := os.RemoveAll(mediaDir); err != nil {
				log.Printf("Warning: could not remove background staging directory %s: %v", mediaDir, err)
			}
		}
	}()
	staging := filepath.Join(mediaDir, "backgrounds")
	if err := platform.EnsureDir(staging); err != nil {
		return "", fmt.Errorf("stage theme backgrounds: %w", err)
	}

	// Sources may be inside the live backgrounds directory, including symlinks.
	sources := append([]string{state.WallpaperPath}, state.AdditionalImages...)
	if original := state.OriginalWallpaperPath; original != "" && original != state.WallpaperPath {
		found := false
		for _, source := range sources {
			if source == original {
				found = true
				break
			}
		}
		if !found {
			sources = append(sources, original)
		}
	}
	seen := make(map[string]string, len(sources))
	for i, src := range sources {
		if i == 0 && src == "" {
			continue
		}
		name := filepath.Base(src)
		if previous, ok := seen[name]; ok {
			// A repeated path is the same file, so stage it once.
			if previous == src {
				continue
			}
			return "", fmt.Errorf("background basename collision %q: %q and %q", name, previous, src)
		}
		seen[name] = src
		if err := platform.CopyFile(src, filepath.Join(staging, name)); err != nil {
			if i == 0 {
				return "", fmt.Errorf("copy wallpaper %q: %w", src, err)
			}
			return "", fmt.Errorf("copy additional image %d %q: %w", i, src, err)
		}
	}
	if err := preserveThemeMedia(targetDir, mediaDir, false); err != nil {
		return "", err
	}

	backup := filepath.Join(mediaDir, "previous")
	if err := os.Rename(bgDir, backup); err != nil {
		return "", fmt.Errorf("backup theme backgrounds: %w", err)
	}
	if err := os.Rename(staging, bgDir); err != nil {
		if restoreErr := os.Rename(backup, bgDir); restoreErr != nil {
			cleanup = false // Keep the only remaining copy available for recovery.
			return "", fmt.Errorf("install theme backgrounds: %w; restore backgrounds from %s: %v", err, backup, restoreErr)
		}
		return "", fmt.Errorf("install theme backgrounds: %w", err)
	}

	wallpaperDest := ""
	if state.WallpaperPath != "" {
		wallpaperDest = filepath.Join(bgDir, filepath.Base(state.WallpaperPath))
	}
	return wallpaperDest, nil
}

// applyEditorThemes applies optional editor themes (Zed). VS Code is written
// by its desktop target (targets.vscodeTarget).
func (w *Writer) applyEditorThemes(
	themeDir string,
	appOverrides map[string]map[string]string,
	includeZed bool,
) error {
	if includeZed || len(appOverrides["zed"]) > 0 {
		if err := ApplyZedTheme(themeDir); err != nil {
			return fmt.Errorf("apply Zed theme: %w", err)
		}
	}
	return nil
}

// ApplyTheme generates all theme files and applies the theme to the system.
func (w *Writer) ApplyTheme(state *ThemeState, settings Settings) (*ApplyResult, error) {
	state, err := materializeWallpaper(state)
	if err != nil {
		return nil, err
	}
	if err := validateIconTheme(state.IconTheme, settings.includesApp("icons")); err != nil {
		return nil, err
	}
	variables := template.BuildVariables(state.ColorRoles, state.LightMode, state.ExtendedColors)
	if err := validateTemplateInputs(variables, state.AppOverrides); err != nil {
		return &ApplyResult{Success: false, ThemePath: platform.ThemeDir()}, err
	}
	if err := RetireLegacyGTKStylesheets(); err != nil {
		log.Printf("Warning: legacy GTK stylesheet cleanup failed: %v", err)
	}

	themeDir := platform.ThemeDir()
	if _, err := prepareThemeDir(themeDir, state); err != nil {
		return &ApplyResult{Success: false, ThemePath: themeDir}, err
	}
	// chromium.theme is written by its desktop target, so apply skips it here.
	if err := w.processTemplatesSkipping(targetOwnedTemplates, variables, themeDir, settings, state.AppOverrides, state.ExtendedColors, state.IconTheme); err != nil {
		return &ApplyResult{Success: false, ThemePath: themeDir}, err
	}
	if err := HandleLightModeMarker(themeDir, state.LightMode); err != nil {
		return &ApplyResult{Success: false, ThemePath: themeDir}, fmt.Errorf("update light mode marker: %w", err)
	}
	if err := w.applyEditorThemes(
		themeDir,
		state.AppOverrides,
		settings.includesApp("zed"),
	); err != nil {
		return &ApplyResult{Success: false, ThemePath: themeDir}, err
	}
	if err := template.ProcessCustomApps(themeDir, variables); err != nil {
		return &ApplyResult{Success: false, ThemePath: themeDir}, fmt.Errorf("process custom apps: %w", err)
	}
	statuses := w.applyTargets(state, variables, settings)
	success := true
	for _, st := range statuses {
		if st.State == "failed" && (st.ID == "yozakura" || st.ID == "hyprland") {
			success = false
		}
	}
	return &ApplyResult{Success: success, ThemePath: themeDir, Targets: statuses}, nil
}

// applyTargets writes and reloads the desktop targets from the Yozakura-derived
// theme. Failures are reported per target and never abort the apply.
func (w *Writer) applyTargets(state *ThemeState, variables map[string]string, settings Settings) []targets.Status {
	home, err := os.UserHomeDir()
	if err != nil {
		return []targets.Status{{ID: "yozakura", Label: "Yozakura shell", State: "failed", Error: err.Error()}}
	}
	yoz := yozakura.Derive(state.ColorRoles, state.NativeColors, nil, nil)
	yozakura.Enforce(&yoz)
	ctx := targets.Context{
		Name:      "Aether",
		ID:        "auto",
		Wallpaper: state.WallpaperPath,
		Vars:      variables,
		Yoz:       yoz,
		Home:      home,
		Run:       w.Run,
		Have:      w.Have,
	}
	if sub, err := fs.Sub(w.templatesFS, w.templatesDir); err == nil {
		ctx.Templates = sub
	}
	return targets.Apply(ctx, targets.All(), settings.TargetEnabled)
}

// GenerateOnly generates theme files to the specified output path without
// applying them (no symlinks, no service restarts, no activation).
func (w *Writer) GenerateOnly(state *ThemeState, settings Settings, outputPath string) error {
	state, err := materializeWallpaper(state)
	if err != nil {
		return err
	}
	if err := validateIconTheme(state.IconTheme, settings.includesApp("icons")); err != nil {
		return err
	}
	variables := template.BuildVariables(state.ColorRoles, state.LightMode, state.ExtendedColors)
	if err := validateTemplateInputs(variables, state.AppOverrides); err != nil {
		return err
	}
	targetDir := outputPath
	if targetDir == "" {
		targetDir = platform.ThemeDir()
	}
	if err := platform.EnsureDir(targetDir); err != nil {
		return fmt.Errorf("create theme output directory: %w", err)
	}

	if _, err := prepareThemeDir(targetDir, state); err != nil {
		return err
	}

	if err := w.processTemplates(variables, targetDir, settings, state.AppOverrides, state.ExtendedColors, state.IconTheme); err != nil {
		return err
	}

	// Generate VSCode extension into the export directory
	vscodeDir := filepath.Join(targetDir, "vscode-extension")
	if settings.includesApp("vscode") || len(state.AppOverrides["vscode"]) > 0 {
		if err := processVSCodeExtension(w.templatesFS, w.templatesDir, vscodeDir, variables); err != nil {
			return fmt.Errorf("export VSCode extension: %w", err)
		}
	} else if err := os.RemoveAll(vscodeDir); err != nil {
		return fmt.Errorf("remove stale VSCode extension: %w", err)
	}

	if err := HandleLightModeMarker(targetDir, state.LightMode); err != nil {
		return fmt.Errorf("update light mode marker: %w", err)
	}

	log.Printf("Theme files generated to: %s", targetDir)
	return nil
}

func validateTemplateInputs(variables map[string]string, appOverrides map[string]map[string]string) error {
	for key, value := range variables {
		switch key {
		case "mode", "theme_type":
			if value != "light" && value != "dark" {
				return fmt.Errorf("template variable %q has an invalid mode", key)
			}
		default:
			if !color.IsHexColor(value) {
				return fmt.Errorf("template variable %q is not a hex color", key)
			}
		}
	}
	for app, overrides := range appOverrides {
		for key, value := range overrides {
			if key == "mode" || key == "theme_type" || !color.IsHexColor(value) {
				return fmt.Errorf("%s override %q is not a hex color", app, key)
			}
		}
	}
	return nil
}

// Subdirectories are user content, not flat generated backgrounds. Preserve
// them even when replacing media, without changing the old tree during staging.
func preserveThemeMedia(existing, staging string, preserveFiles bool) error {
	if _, err := os.Stat(existing); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect existing theme: %w", err)
	}

	sourceBackgrounds := filepath.Join(existing, "backgrounds")
	entries, err := os.ReadDir(sourceBackgrounds)
	if err == nil {
		targetBackgrounds := filepath.Join(staging, "backgrounds")
		if err := platform.EnsureDir(targetBackgrounds); err != nil {
			return err
		}
		for _, entry := range entries {
			source := filepath.Join(sourceBackgrounds, entry.Name())
			target := filepath.Join(targetBackgrounds, entry.Name())
			if entry.IsDir() {
				if err := copyBackgroundDir(source, target); err != nil {
					return fmt.Errorf("preserve background directory %q: %w", entry.Name(), err)
				}
				continue
			}
			if !preserveFiles {
				continue
			}
			if err := platform.CopyFile(source, target); err != nil {
				return fmt.Errorf("preserve background %q: %w", entry.Name(), err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read existing backgrounds: %w", err)
	}
	if !preserveFiles {
		return nil
	}

	for _, name := range []string{"preview.png", "preview.jpg", "preview.jpeg", "preview.webp", "preview.gif", "preview.bmp"} {
		source := filepath.Join(existing, name)
		if _, err := os.Stat(source); err != nil {
			continue
		}
		if err := platform.CopyFile(source, filepath.Join(staging, name)); err != nil {
			return fmt.Errorf("preserve preview: %w", err)
		}
		break
	}
	return nil
}

func copyBackgroundDir(source, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("background directory changed during staging: %s", source)
	}
	if err := os.Mkdir(target, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src, dst := filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name())
		switch {
		case entry.IsDir():
			err = copyBackgroundDir(src, dst)
		case entry.Type()&os.ModeSymlink != 0:
			var link string
			link, err = os.Readlink(src)
			if err == nil {
				err = os.Symlink(link, dst)
			}
		case entry.Type().IsRegular():
			err = platform.CopyFile(src, dst)
		default:
			err = fmt.Errorf("cannot preserve non-regular background entry: %s", src)
		}
		if err != nil {
			return err
		}
	}
	return os.Chmod(target, info.Mode())
}

// processTemplates reads each template from the embedded FS, applies variable
// substitution, and writes the result to outputDir. Certain templates are
// skipped based on settings.
func (w *Writer) processTemplates(
	variables map[string]string,
	outputDir string,
	settings Settings,
	appOverrides map[string]map[string]string,
	globalOverrides map[string]string,
	iconThemes ...icontheme.Selection,
) error {
	return w.processTemplatesSkipping(nil, variables, outputDir, settings, appOverrides, globalOverrides, iconThemes...)
}

// targetOwnedTemplates are written by a desktop target during ApplyTheme.
var targetOwnedTemplates = map[string]bool{"chromium.theme": true}

func (w *Writer) processTemplatesSkipping(
	skip map[string]bool,
	variables map[string]string,
	outputDir string,
	settings Settings,
	appOverrides map[string]map[string]string,
	globalOverrides map[string]string,
	iconThemes ...icontheme.Selection,
) error {
	names, err := template.ListTemplates(w.templatesFS, w.templatesDir)
	if err != nil {
		return fmt.Errorf("list templates: %w", err)
	}

	for _, fileName := range names {
		// Skip copy.json (config file).
		if fileName == "copy.json" || skip[fileName] {
			continue
		}

		outputPath := filepath.Join(outputDir, fileName)
		appName := getAppNameFromFileName(fileName)
		if appName != "colors" && !settings.includesApp(appName) && (appName == "icons" || len(appOverrides[appName]) == 0) {
			if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove stale template %s: %w", fileName, err)
			}
			continue
		}

		if fileName == "icons.theme" {
			if err := w.writeIconTheme(outputPath, variables, appOverrides, globalOverrides, iconThemes...); err != nil {
				return err
			}
			continue
		}

		// Handle neovim.lua with custom config selection
		if fileName == "neovim.lua" && settings.SelectedNeovimConfig != "" && len(appOverrides[appName]) == 0 {
			if err := platform.WriteText(outputPath, settings.SelectedNeovimConfig); err != nil {
				return fmt.Errorf("write custom neovim.lua: %w", err)
			} else {
				log.Printf("Applied selected Neovim theme to %s", outputPath)
			}
			continue
		}

		if err := w.processTemplate(fileName, outputPath, variables, appOverrides, globalOverrides); err != nil {
			return err
		}
	}
	return nil
}

// processTemplate reads a single template, applies variable substitution
// (including per-app overrides), and writes the result.
func (w *Writer) processTemplate(
	fileName string,
	outputPath string,
	variables map[string]string,
	appOverrides map[string]map[string]string,
	globalOverrides map[string]string,
) error {
	// Check for custom override in ~/.config/aether/custom/ first
	content, isCustom := template.ReadCustomOverride(fileName)
	if !isCustom {
		var err error
		content, err = template.ReadTemplate(w.templatesFS, w.templatesDir, fileName)
		if err != nil {
			return fmt.Errorf("read template %s: %w", fileName, err)
		}
	}

	// Check for app-specific overrides
	appName := getAppNameFromFileName(fileName)
	overrides := appOverrides[appName]
	mergedVars := mergeTemplateVariables(variables, overrides, globalOverrides)
	if len(overrides) > 0 {
		log.Printf("Applied %d override(s) to %s", len(overrides), fileName)
	}

	processed := template.ProcessTemplate(content, mergedVars)

	if err := platform.WriteText(outputPath, processed); err != nil {
		return fmt.Errorf("write processed template %s: %w", fileName, err)
	}
	return nil
}

func mergeTemplateVariables(
	variables map[string]string,
	overrides map[string]string,
	globalOverrides map[string]string,
) map[string]string {
	if len(overrides) == 0 {
		return variables
	}

	merged := make(map[string]string, len(variables)+len(overrides))
	for k, v := range variables {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}

	// Recompute aliases and derived colors while preserving explicit pins.
	explicit := make(map[string]string, len(overrides)+len(globalOverrides))
	for k, v := range globalOverrides {
		explicit[k] = v
	}
	for k, v := range overrides {
		explicit[k] = v
	}
	template.RecomputeDerived(merged, explicit)
	return merged
}

// processVSCodeExtension recursively reads all files from the embedded
// vscode-extension directory, applies variable substitution, and writes
// them to destDir.
func processVSCodeExtension(fsys embed.FS, templatesDir string, destDir string, variables map[string]string) error {
	root := path.Join(templatesDir, "vscode-extension")

	return fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute the relative path from the vscode-extension root.
		// Use path (not filepath) because embed.FS always uses forward slashes.
		rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		if rel == "" {
			rel = "."
		}
		destPath := filepath.Join(destDir, filepath.FromSlash(rel))

		if d.IsDir() {
			return platform.EnsureDir(destPath)
		}

		// Read file from embedded FS
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}

		// Apply variable substitution (only {key} format for VSCode extension)
		content := string(data)
		for key, value := range variables {
			content = strings.ReplaceAll(content, "{"+key+"}", value)
		}

		return platform.WriteText(destPath, content)
	})
}
