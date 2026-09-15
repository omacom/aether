package favorites

import "testing"

func TestGitHubFavoritePreservesNameAndTypeAcrossReload(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	service := NewService()
	path := "https://raw.githubusercontent.com/owner/repo/main/wall.png"
	if !service.Toggle(path, "github", map[string]interface{}{"name": "wall.png"}) {
		t.Fatal("favorite is not added")
	}
	loaded := NewService().GetAll()
	if len(loaded) != 1 || loaded[0].Path != path || loaded[0].Type != "github" || loaded[0].Data["name"] != "wall.png" {
		t.Fatalf("favorite = %+v", loaded)
	}
}
