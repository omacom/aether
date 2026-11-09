package blueprint

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aether/internal/platform"
)

// Service manages blueprint persistence.
type Service struct {
	dir          string
	settingsPath string
}

// NewService creates a new blueprint service.
func NewService() *Service {
	dir := platform.BlueprintDir()
	if err := platform.EnsureDir(dir); err != nil {
		log.Printf("[blueprint] ensure dir %s: %v", dir, err)
	}
	return &Service{dir: dir, settingsPath: filepath.Join(platform.ConfigDir(), "settings.json")}
}

// LoadAll reads all blueprints from the blueprints directory.
func (s *Service) LoadAll() ([]Blueprint, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read blueprints dir: %w", err)
	}

	var blueprints []Blueprint
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.dir, entry.Name())
		bp, err := s.loadFromFile(path, entry.Name())
		if err != nil {
			continue
		}
		blueprints = append(blueprints, bp)
	}
	return blueprints, nil
}

// FindByName finds a blueprint by name (case-insensitive).
func (s *Service) FindByName(name string) (*Blueprint, error) {
	blueprints, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	lower := strings.ToLower(name)

	// Exact name match first
	for i := range blueprints {
		if strings.ToLower(blueprints[i].Name) == lower {
			return &blueprints[i], nil
		}
	}

	// Partial filename match
	for i := range blueprints {
		fn := strings.ToLower(strings.TrimSuffix(blueprints[i].Filename, ".json"))
		if strings.Contains(fn, lower) {
			return &blueprints[i], nil
		}
	}

	return nil, nil
}

// Save persists a blueprint to disk.
func (s *Service) Save(name string, bp Blueprint) error {
	bp.Name = name
	bp.Timestamp = time.Now().UnixMilli()
	if err := validateBlueprint(&bp); err != nil {
		return fmt.Errorf("validate blueprint: %w", err)
	}

	// Sanitize filename
	safeName := strings.ReplaceAll(name, "/", "-")
	safeName = strings.ReplaceAll(safeName, " ", "-")
	filename := safeName + ".json"
	path := filepath.Join(s.dir, filename)

	return platform.WriteJSON(path, bp)
}

// Delete removes a blueprint by its full, case-insensitive display name.
// Empty or duplicate names are rejected; fuzzy lookup is never used for deletion.
func (s *Service) Delete(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("blueprint name must not be empty")
	}
	blueprints, err := s.LoadAll()
	if err != nil {
		return err
	}
	var match *Blueprint
	for i := range blueprints {
		if strings.EqualFold(blueprints[i].Name, name) {
			if match != nil {
				return fmt.Errorf("blueprint name %q is ambiguous", name)
			}
			match = &blueprints[i]
		}
	}
	if match == nil {
		return fmt.Errorf("blueprint %q not found", name)
	}
	if err := os.Remove(match.Path); err != nil {
		return fmt.Errorf("delete blueprint %q: %w", name, err)
	}
	return nil
}

// Validate checks a blueprint's structure and color values.
func (s *Service) Validate(bp *Blueprint) bool {
	return validateBlueprint(bp) == nil
}

func (s *Service) loadFromFile(path, filename string) (Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Blueprint{}, err
	}

	var bp Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return Blueprint{}, err
	}

	bp.Path = path
	bp.Filename = filename
	if bp.Name == "" {
		bp.Name = strings.TrimSuffix(filename, ".json")
	}
	if err := validateBlueprint(&bp); err != nil {
		return Blueprint{}, err
	}

	return bp, nil
}

// FindForWallpaper returns the first blueprint (name ascending, then filename)
// whose matchWallpapers pattern is a case-insensitive substring of the
// wallpaper's base name. It returns nil, nil when none match.
func (s *Service) FindForWallpaper(path string) (*Blueprint, error) {
	blueprints, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(blueprints, func(i, j int) bool {
		a, b := strings.ToLower(blueprints[i].Name), strings.ToLower(blueprints[j].Name)
		if a != b {
			return a < b
		}
		return blueprints[i].Filename < blueprints[j].Filename
	})
	base := strings.ToLower(filepath.Base(path))
	for i := range blueprints {
		for _, pat := range blueprints[i].MatchWallpapers {
			pat = strings.ToLower(strings.TrimSpace(pat))
			if pat != "" && strings.Contains(base, pat) {
				return &blueprints[i], nil
			}
		}
	}
	return nil, nil
}

// Default returns the blueprint named by "defaultBlueprint" in settings.json,
// or nil, nil when unset or when the named blueprint does not exist.
func (s *Service) Default() (*Blueprint, error) {
	cfg, err := platform.ReadJSON[map[string]interface{}](s.settingsPath)
	if err != nil {
		return nil, nil
	}
	name, _ := cfg["defaultBlueprint"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	blueprints, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(name)
	for i := range blueprints {
		if strings.ToLower(blueprints[i].Name) == lower {
			return &blueprints[i], nil
		}
	}
	log.Printf("[blueprint] defaultBlueprint %q not found", name)
	return nil, nil
}
