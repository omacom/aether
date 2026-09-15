package theme

import (
	"fmt"

	"aether/internal/icontheme"
	"aether/internal/platform"
)

func validateIconTheme(selection icontheme.Selection, enabled bool) error {
	if !enabled {
		return nil
	}
	if _, err := icontheme.NormalizeSelection(selection); err != nil {
		return fmt.Errorf("icon theme: %w", err)
	}
	return nil
}

func (w *Writer) writeIconTheme(outputPath string, variables map[string]string, appOverrides map[string]map[string]string, globalOverrides map[string]string, selections ...icontheme.Selection) error {
	selection := icontheme.Automatic()
	if len(selections) > 0 {
		selection = selections[0]
	}
	normalized, err := icontheme.NormalizeSelection(selection)
	if err != nil {
		return fmt.Errorf("icon theme: %w", err)
	}
	if normalized.Mode == icontheme.SelectionExplicit {
		if err := platform.WriteText(outputPath, normalized.ID+"\n"); err != nil {
			return fmt.Errorf("write icons.theme: %w", err)
		}
		return nil
	}
	return w.processTemplate("icons.theme", outputPath, variables, appOverrides, globalOverrides)
}
