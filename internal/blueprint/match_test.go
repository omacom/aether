package blueprint

import (
	"os"
	"path/filepath"
	"testing"
)

func newMatchService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	return &Service{dir: dir, settingsPath: filepath.Join(t.TempDir(), "settings.json")}
}

func saveBP(t *testing.T, s *Service, name string, bp Blueprint) {
	t.Helper()
	bp.Name = name
	bp.Palette.Colors = make([]string, 16)
	for i := range bp.Palette.Colors {
		bp.Palette.Colors[i] = "#112233"
	}
	if err := s.Save(name, bp); err != nil {
		t.Fatal(err)
	}
}

func TestFindForWallpaperCaseInsensitiveSubstring(t *testing.T) {
	s := newMatchService(t)
	saveBP(t, s, "evan", Blueprint{MatchWallpapers: []string{"evanescia"}})
	got, err := s.FindForWallpaper("/x/Evanescia-02.PNG")
	if err != nil || got == nil || got.Name != "evan" {
		t.Fatalf("got %v err %v", got, err)
	}
	got, err = s.FindForWallpaper("/x/other.png")
	if err != nil || got != nil {
		t.Fatalf("want nil,nil got %v %v", got, err)
	}
}

func TestFindForWallpaperIgnoresEmptyPatternAndDir(t *testing.T) {
	s := newMatchService(t)
	saveBP(t, s, "empty", Blueprint{MatchWallpapers: []string{""}})
	got, err := s.FindForWallpaper("/evanescia/foo.png")
	if err != nil || got != nil {
		t.Fatalf("want nil,nil got %v %v", got, err)
	}
}

func TestFindForWallpaperStableOrder(t *testing.T) {
	s := newMatchService(t)
	saveBP(t, s, "b-two", Blueprint{MatchWallpapers: []string{"wall"}})
	saveBP(t, s, "a-one", Blueprint{MatchWallpapers: []string{"wall"}})
	for i := 0; i < 5; i++ {
		got, err := s.FindForWallpaper("/p/wall.png")
		if err != nil || got == nil || got.Name != "a-one" {
			t.Fatalf("got %v err %v", got, err)
		}
	}
}

func TestDefaultBlueprintFromSettings(t *testing.T) {
	s := newMatchService(t)
	saveBP(t, s, "icho", Blueprint{})
	got, err := s.Default()
	if err != nil || got != nil {
		t.Fatalf("unset: want nil,nil got %v %v", got, err)
	}
	if err := os.WriteFile(s.settingsPath, []byte(`{"defaultBlueprint":"icho"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = s.Default()
	if err != nil || got == nil || got.Name != "icho" {
		t.Fatalf("got %v err %v", got, err)
	}
	if err := os.WriteFile(s.settingsPath, []byte(`{"defaultBlueprint":"missing"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = s.Default()
	if err != nil || got != nil {
		t.Fatalf("missing: want nil,nil got %v %v", got, err)
	}
}
