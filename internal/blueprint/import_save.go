package blueprint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aether/internal/platform"
)

// ImportResult lists what ImportYozakuraFile saved.
type ImportResult struct {
	Imported []string `json:"imported"`
	Default  string   `json:"default"`
}

// ImportYozakuraFile converts a curated Yozakura themes.json into blueprints,
// saves them, and records the file's default theme in settings.json. Existing
// blueprints with the same name are refused unless force is set. It backs both
// `aether --import-yozakura` and the GUI.
func ImportYozakuraFile(path string, force bool) (*ImportResult, error) {
	bps, def, err := ImportYozakura(path)
	if err != nil {
		return nil, err
	}
	svc := NewService()
	existing, err := svc.LoadAll()
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("blueprints already exist: %s (use --force to overwrite)", strings.Join(clash, ", "))
		}
	}
	res := &ImportResult{Imported: []string{}}
	for _, bp := range bps {
		if err := svc.Save(bp.Name, bp); err != nil {
			return res, fmt.Errorf("save %s: %w", bp.Name, err)
		}
		res.Imported = append(res.Imported, bp.Name)
	}
	if def != "" {
		if err := SetDefaultBlueprint(def); err != nil {
			return res, fmt.Errorf("set default blueprint: %w", err)
		}
		res.Default = def
	}
	return res, nil
}

// SetDefaultBlueprint writes defaultBlueprint into settings.json, preserving
// every other key.
func SetDefaultBlueprint(name string) error {
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
