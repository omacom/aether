package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aether/internal/platform"
)

func TestApplyVSCodeThemeRegistersExtension(t *testing.T) {
	home := setupWriterTestEnv(t)

	variables := map[string]string{"theme_type": "dark"}
	if err := ApplyVSCodeTheme(omarchyV4TestTemplates, "testdata/v4", variables); err != nil {
		t.Fatalf("ApplyVSCodeTheme() error = %v", err)
	}

	extensionsFile := filepath.Join(home, ".vscode", "extensions", "extensions.json")
	entries := readExtensionEntries(t, extensionsFile)
	entry := findExtensionEntry(t, entries, vscodeExtensionID)

	wantDir := filepath.Join(home, ".vscode", "extensions", "local.theme-aether-1.0.0")
	location, ok := entry["location"].(map[string]interface{})
	if !ok {
		t.Fatalf("entry[location] = %#v, want an object", entry["location"])
	}
	if got := location["path"]; got != wantDir {
		t.Errorf("location.path = %v, want %q", got, wantDir)
	}
	if got := entry["relativeLocation"]; got != "local.theme-aether-1.0.0" {
		t.Errorf("relativeLocation = %v, want local.theme-aether-1.0.0", got)
	}
}

func TestRegisterVSCodeExtensionPreservesExistingEntries(t *testing.T) {
	extensionsDir := t.TempDir()
	extensionDir := filepath.Join(extensionsDir, "local.theme-aether-1.0.0")

	marketplaceEntry := `{"identifier":{"id":"ms-python.python","uuid":"abc"},"version":"1.2.3","location":{"$mid":1,"path":"/home/u/.vscode/extensions/ms-python.python-1.2.3","scheme":"file"},"relativeLocation":"ms-python.python-1.2.3","metadata":{"source":"gallery","installedTimestamp":123}}`
	writeExtensionsJSON(t, extensionsDir, "["+marketplaceEntry+"]")

	if err := registerVSCodeExtension(extensionsDir, extensionDir); err != nil {
		t.Fatalf("registerVSCodeExtension() error = %v", err)
	}

	entries := readExtensionEntries(t, filepath.Join(extensionsDir, "extensions.json"))
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2: %#v", len(entries), entries)
	}

	preserved := findExtensionEntry(t, entries, "ms-python.python")
	metadata, ok := preserved["metadata"].(map[string]interface{})
	if !ok || metadata["source"] != "gallery" {
		t.Errorf("marketplace entry lost its metadata: %#v", preserved)
	}

	findExtensionEntry(t, entries, vscodeExtensionID)
}

func TestRegisterVSCodeExtensionIsIdempotent(t *testing.T) {
	extensionsDir := t.TempDir()
	extensionDir := filepath.Join(extensionsDir, "local.theme-aether-1.0.0")

	for i := 0; i < 2; i++ {
		if err := registerVSCodeExtension(extensionsDir, extensionDir); err != nil {
			t.Fatalf("registerVSCodeExtension() error = %v", err)
		}
	}

	entries := readExtensionEntries(t, filepath.Join(extensionsDir, "extensions.json"))
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1 (no duplicates): %#v", len(entries), entries)
	}
}

func TestRegisterVSCodeExtensionClearsObsoleteMarker(t *testing.T) {
	extensionsDir := t.TempDir()
	extensionDir := filepath.Join(extensionsDir, "local.theme-aether-1.0.0")

	obsoleteFile := filepath.Join(extensionsDir, ".obsolete")
	key := vscodeExtensionID + "-" + vscodeExtensionVersion
	writeFile(t, obsoleteFile, `{"`+key+`":true,"other.ext-1.0.0":true}`)

	if err := registerVSCodeExtension(extensionsDir, extensionDir); err != nil {
		t.Fatalf("registerVSCodeExtension() error = %v", err)
	}

	data, err := os.ReadFile(obsoleteFile)
	if err != nil {
		t.Fatalf("read .obsolete: %v", err)
	}
	var obsolete map[string]interface{}
	if err := json.Unmarshal(data, &obsolete); err != nil {
		t.Fatalf("parse .obsolete: %v", err)
	}
	if _, ok := obsolete[key]; ok {
		t.Errorf(".obsolete still lists %q", key)
	}
	if _, ok := obsolete["other.ext-1.0.0"]; !ok {
		t.Errorf(".obsolete lost unrelated key: %#v", obsolete)
	}
}

func TestRegisterVSCodeExtensionRemovesObsoleteFileWhenEmpty(t *testing.T) {
	extensionsDir := t.TempDir()
	extensionDir := filepath.Join(extensionsDir, "local.theme-aether-1.0.0")

	obsoleteFile := filepath.Join(extensionsDir, ".obsolete")
	key := vscodeExtensionID + "-" + vscodeExtensionVersion
	writeFile(t, obsoleteFile, `{"`+key+`":true}`)

	if err := registerVSCodeExtension(extensionsDir, extensionDir); err != nil {
		t.Fatalf("registerVSCodeExtension() error = %v", err)
	}

	if platform.FileExists(obsoleteFile) {
		t.Error(".obsolete still exists after its only entry was cleared")
	}
}

func writeExtensionsJSON(t *testing.T, extensionsDir, content string) {
	t.Helper()
	writeFile(t, filepath.Join(extensionsDir, "extensions.json"), content)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readExtensionEntries(t *testing.T, path string) []map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return entries
}

func findExtensionEntry(t *testing.T, entries []map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	for _, entry := range entries {
		identifier, ok := entry["identifier"].(map[string]interface{})
		if ok && identifier["id"] == id {
			return entry
		}
	}
	t.Fatalf("no entry with identifier.id = %q in %#v", id, entries)
	return nil
}
