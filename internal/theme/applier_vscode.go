package theme

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"aether/internal/platform"
)

// vscodeExtensionID and vscodeExtensionVersion identify the generated
// extension in VS Code's own extensions.json. They must match the
// publisher/name/version baked into templates/vscode-extension/package.json.
const (
	vscodeExtensionID      = "local.theme-aether"
	vscodeExtensionVersion = "1.0.0"
)

// ApplyVSCodeTheme installs the VSCode extension theme by copying the
// vscode-extension template directory from the embedded FS to
// ~/.vscode/extensions/theme-aether/, processing template variables along
// the way.
func ApplyVSCodeTheme(fsys embed.FS, templatesDir string, variables map[string]string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	extensionsDir := filepath.Join(home, ".vscode", "extensions")
	extensionDir := filepath.Join(extensionsDir, "local.theme-aether-1.0.0")
	if err := platform.EnsureDir(extensionDir); err != nil {
		return err
	}

	if err := processVSCodeExtension(fsys, templatesDir, extensionDir, variables); err != nil {
		return err
	}

	// Recent VS Code releases only treat an extension as installed once it
	// has an entry in extensions.json; a well-formed folder sitting in the
	// extensions directory is no longer picked up on its own. Omarchy's own
	// VS Code theme hook (omarchy-theme-set-vscode) registers its bundled
	// "local.omarchy-theme" extension the same way for the same reason.
	if err := registerVSCodeExtension(extensionsDir, extensionDir); err != nil {
		log.Printf("Failed to register VS Code extension in extensions.json: %v", err)
	}

	log.Printf("VS Code theme extension installed to: %s", extensionDir)
	return nil
}

// registerVSCodeExtension records extensionDir as an installed extension in
// extensionsDir/extensions.json, the manifest VS Code consults to decide
// what is installed. Unrelated entries are preserved byte-for-byte via
// json.RawMessage so extensions installed from the Marketplace keep their
// metadata intact.
func registerVSCodeExtension(extensionsDir, extensionDir string) error {
	extensionsFile := filepath.Join(extensionsDir, "extensions.json")

	var entries []json.RawMessage
	if platform.FileExists(extensionsFile) {
		data, err := os.ReadFile(extensionsFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", extensionsFile, err)
		}
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("parse %s: %w", extensionsFile, err)
		}
	}

	filtered := entries[:0]
	for _, raw := range entries {
		if extensionEntryID(raw) != vscodeExtensionID {
			filtered = append(filtered, raw)
		}
	}

	entry := map[string]interface{}{
		"identifier": map[string]string{"id": vscodeExtensionID},
		"version":    vscodeExtensionVersion,
		"location": map[string]interface{}{
			"$mid":     1,
			"fsPath":   extensionDir,
			"external": "file://" + extensionDir,
			"path":     extensionDir,
			"scheme":   "file",
		},
		"relativeLocation": filepath.Base(extensionDir),
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	filtered = append(filtered, raw)

	if err := platform.WriteJSON(extensionsFile, filtered); err != nil {
		return fmt.Errorf("write %s: %w", extensionsFile, err)
	}

	return unmarkVSCodeExtensionObsolete(extensionsDir)
}

// extensionEntryID extracts identifier.id from a raw extensions.json entry,
// returning "" if the entry isn't shaped as expected.
func extensionEntryID(raw json.RawMessage) string {
	var entry struct {
		Identifier struct {
			ID string `json:"id"`
		} `json:"identifier"`
	}
	if err := json.Unmarshal(raw, &entry); err != nil {
		return ""
	}
	return entry.Identifier.ID
}

// unmarkVSCodeExtensionObsolete removes any stale obsolete-extension marker
// left behind by a previous uninstall of this extension. VS Code will not
// surface an extension that's still listed there, even once it's back in
// extensions.json.
func unmarkVSCodeExtensionObsolete(extensionsDir string) error {
	obsoleteFile := filepath.Join(extensionsDir, ".obsolete")
	if !platform.FileExists(obsoleteFile) {
		return nil
	}

	data, err := os.ReadFile(obsoleteFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", obsoleteFile, err)
	}

	var obsolete map[string]json.RawMessage
	if err := json.Unmarshal(data, &obsolete); err != nil {
		return fmt.Errorf("parse %s: %w", obsoleteFile, err)
	}

	key := vscodeExtensionID + "-" + vscodeExtensionVersion
	if _, ok := obsolete[key]; !ok {
		return nil
	}
	delete(obsolete, key)

	if len(obsolete) == 0 {
		return platform.DeleteFile(obsoleteFile)
	}
	return platform.WriteJSON(obsoleteFile, obsolete)
}
