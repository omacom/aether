package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aether/internal/theme"
	"aether/internal/yozakura"
	"aether/ipc"
)

func yozTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local/state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local/share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	return home
}

func yozPalette() []string {
	p := make([]string, 16)
	for i := range p {
		p[i] = fmt.Sprintf("#%02x%02x%02x", 16+i*10, 32+i*5, 64+i*8)
	}
	return p
}

func saveYozBlueprint(t *testing.T, app *App) (float64, yozakura.Clock) {
	t.Helper()
	glass := 0.81
	clock := yozakura.Clock{Digits: "thin", Colon: "·", Glow: 0.5, Ornament: "none"}
	err := app.SaveBlueprint(SaveBlueprintRequest{
		Name:            "yoz",
		Palette:         yozPalette(),
		YozakuraRoles:   map[string]string{"accent": "#ff0000"},
		Glass:           &glass,
		Clock:           &clock,
		MatchWallpapers: []string{"evanescia", "icho"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return glass, clock
}

func TestSaveBlueprintRoundTripsYozakuraFields(t *testing.T) {
	yozTestHome(t)
	app := NewApp()
	glass, clock := saveYozBlueprint(t, app)

	list, err := app.ListBlueprints()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v, %v", list, err)
	}
	bp := list[0]
	pal := bp["palette"].(map[string]interface{})
	if !reflect.DeepEqual(pal["yozakuraRoles"], map[string]string{"accent": "#ff0000"}) {
		t.Errorf("roles = %v", pal["yozakuraRoles"])
	}
	if g, _ := pal["glass"].(*float64); g == nil || *g != glass {
		t.Errorf("glass = %v", pal["glass"])
	}
	if c, _ := pal["clock"].(*yozakura.Clock); c == nil || *c != clock {
		t.Errorf("clock = %v", pal["clock"])
	}
	if !reflect.DeepEqual(bp["matchWallpapers"], []string{"evanescia", "icho"}) {
		t.Errorf("matchWallpapers = %v", bp["matchWallpapers"])
	}

	if err := app.LoadBlueprint("yoz"); err != nil {
		t.Fatal(err)
	}
	if a := app.state.YozakuraRoles["accent"]; a != "#ff0000" {
		t.Errorf("state roles = %v", app.state.YozakuraRoles)
	}
	if app.state.Glass == nil || *app.state.Glass != glass || app.state.Clock == nil || *app.state.Clock != clock {
		t.Errorf("state glass/clock = %v %v", app.state.Glass, app.state.Clock)
	}
}

func TestImportFileReturnsGlassAndClock(t *testing.T) {
	yozTestHome(t)
	app := NewApp()
	glass := 0.7
	data, _ := json.Marshal(map[string]interface{}{
		"name": "imp",
		"palette": map[string]interface{}{
			"colors": yozPalette(), "glass": glass,
			"clock": map[string]interface{}{"digits": "bold", "colon": ":", "glow": 0.2, "ornament": "arc"},
		},
	})
	path := filepath.Join(t.TempDir(), "imp.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := app.importFile(path, "blueprint")
	if err != nil {
		t.Fatal(err)
	}
	if res.Glass == nil || *res.Glass != glass {
		t.Errorf("glass = %v", res.Glass)
	}
	if res.Clock == nil || res.Clock.Glow != 0.2 || res.Clock.Ornament != "arc" {
		t.Errorf("clock = %v", res.Clock)
	}
	if app.state.Glass == nil || app.state.Clock == nil {
		t.Error("state did not receive glass/clock")
	}
}

func TestIPCStatePayloadAfterLoadBlueprintHasGlassClock(t *testing.T) {
	yozTestHome(t)
	app := NewApp()
	glass, clock := saveYozBlueprint(t, app)
	if resp := app.HandleIPC(ipc.Request{Cmd: "load-blueprint", Name: "yoz"}); !resp.OK {
		t.Fatalf("load-blueprint: %s", resp.Error)
	}
	p := app.ipcStatePayload()
	if g, _ := p["glass"].(*float64); g == nil || *g != glass {
		t.Errorf("payload glass = %v", p["glass"])
	}
	if c, _ := p["clock"].(*yozakura.Clock); c == nil || *c != clock {
		t.Errorf("payload clock = %v", p["clock"])
	}
	if r, _ := p["yozakuraRoles"].(map[string]string); r["accent"] != "#ff0000" {
		t.Errorf("payload roles = %v", p["yozakuraRoles"])
	}
}

func TestIPCApplyUsesStateGlassAndClock(t *testing.T) {
	home := yozTestHome(t)
	app := NewApp()
	w := theme.NewWriter(EmbeddedTemplates, "templates")
	w.Run = func(string, ...string) (string, error) { return "", nil }
	w.Have = func(string) bool { return true }
	app.writer = w
	saveYozBlueprint(t, app)
	if resp := app.HandleIPC(ipc.Request{Cmd: "load-blueprint", Name: "yoz"}); !resp.OK {
		t.Fatal(resp.Error)
	}
	if resp := app.HandleIPC(ipc.Request{Cmd: "apply"}); !resp.OK {
		t.Fatalf("apply: %s", resp.Error)
	}
	data, err := os.ReadFile(filepath.Join(home, ".local/state/yozakura/theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Glass float64        `json:"glass"`
		Clock yozakura.Clock `json:"clock"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Glass != 0.81 || got.Clock.Digits != "thin" || !strings.Contains(string(data), `"ornament": "none"`) {
		t.Errorf("theme.json glass/clock not from state: %s", data)
	}
}
