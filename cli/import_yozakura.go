package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aether/internal/blueprint"
	"aether/internal/platform"
)

func runImportYozakura(args []string) int {
	force, args := hasFlag(args, "--force")
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: aether --import-yozakura <themes.json> [--force]")
		return 1
	}
	bps, def, err := blueprint.ImportYozakura(expandHome(args[0]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	svc := blueprint.NewService()
	existing, err := svc.LoadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if !force {
		var clash []string
		for _, bp := range bps {
			for _, e := range existing {
				if strings.EqualFold(e.Name, bp.Name) {
					clash = append(clash, bp.Name)
					break
				}
			}
		}
		if len(clash) > 0 {
			fmt.Fprintf(os.Stderr, "Error: blueprints already exist: %s (use --force to overwrite)\n", strings.Join(clash, ", "))
			return 1
		}
	}
	for _, bp := range bps {
		if err := svc.Save(bp.Name, bp); err != nil {
			fmt.Fprintf(os.Stderr, "Error: save %s: %v\n", bp.Name, err)
			return 1
		}
		fmt.Printf("Imported blueprint: %s\n", bp.Name)
	}
	if def != "" {
		if err := setDefaultBlueprint(def); err != nil {
			fmt.Fprintf(os.Stderr, "Error: set default blueprint: %v\n", err)
			return 1
		}
		fmt.Printf("Default blueprint: %s\n", def)
	}
	return 0
}

// setDefaultBlueprint writes defaultBlueprint into settings.json, preserving
// every other key.
func setDefaultBlueprint(name string) error {
	path := filepath.Join(platform.ConfigDir(), "settings.json")
	cfg, err := platform.ReadJSON[map[string]interface{}](path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read settings %s: %w", path, err)
		}
		cfg = nil
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	cfg["defaultBlueprint"] = name
	if err := platform.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return platform.WriteJSON(path, cfg)
}
