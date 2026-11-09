package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aether/internal/blueprint"
)

const curatedFixture = "../internal/yozakura/testdata/curated.json"

func isolate(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	return filepath.Join(home, ".config", "aether")
}

func capture(t *testing.T, f func() int) (int, string) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := f()
	w.Close()
	os.Stdout = old
	var b bytes.Buffer
	io.Copy(&b, r)
	return code, b.String()
}

func TestImportYozakuraRefusesOverwrite(t *testing.T) {
	cfg := isolate(t)
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(cfg, "settings.json"), []byte(`{"keep":"me"}`), 0o644)
	if c, _ := capture(t, func() int { return runImportYozakura([]string{curatedFixture}) }); c != 0 {
		t.Fatalf("first import exit %d", c)
	}
	raw, _ := os.ReadFile(filepath.Join(cfg, "settings.json"))
	var s map[string]any
	json.Unmarshal(raw, &s)
	if s["defaultBlueprint"] != "icho" || s["keep"] != "me" {
		t.Fatalf("settings = %s", raw)
	}
	// Tamper with icho, then re-import.
	p := filepath.Join(cfg, "blueprints", "icho.json")
	os.WriteFile(p, []byte(strings.Replace(mustRead(t, p), `"#ff9ed2"`, `"#ff9ed3"`, -1)), 0o644)
	if c, _ := capture(t, func() int { return runImportYozakura([]string{curatedFixture}) }); c != 1 {
		t.Fatalf("overwrite without --force exit %d", c)
	}
	if !strings.Contains(mustRead(t, p), "#ff9ed3") {
		t.Fatal("file overwritten without --force")
	}
	if c, _ := capture(t, func() int { return runImportYozakura([]string{curatedFixture, "--force"}) }); c != 0 {
		t.Fatalf("--force exit %d", c)
	}
	if strings.Contains(mustRead(t, p), "#ff9ed3") {
		t.Fatal("--force did not overwrite")
	}
}

func mustRead(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRunCheckContrastExitCode(t *testing.T) {
	isolate(t)
	if c, _ := capture(t, func() int { return runImportYozakura([]string{curatedFixture}) }); c != 0 {
		t.Fatalf("import exit %d", c)
	}
	c, out := capture(t, func() int { return runCheckContrast([]string{"icho"}) })
	if c != 0 || !strings.Contains(out, "text/surface") {
		t.Fatalf("icho exit %d out:\n%s", c, out)
	}

	svc := blueprint.NewService()
	bp, _ := svc.FindByName("icho")
	bad := *bp
	bad.Palette.YozakuraRoles = map[string]string{}
	for k, v := range bp.Palette.YozakuraRoles {
		bad.Palette.YozakuraRoles[k] = v
	}
	bad.Palette.YozakuraRoles["text"] = "#2a2030" // near surface
	bad.Palette.YozakuraRoles["muted"] = "#2a2030"
	if err := svc.Save("bad", bad); err != nil {
		t.Fatal(err)
	}
	c, out = capture(t, func() int { return runCheckContrast([]string{"bad"}) })
	if c != 1 || !strings.Contains(out, "below 7.0") {
		t.Fatalf("bad exit %d out:\n%s", c, out)
	}
	if !strings.Contains(out, "below 4.5") {
		t.Fatalf("missing below 4.5:\n%s", out)
	}
}
