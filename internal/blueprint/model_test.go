package blueprint

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshalLegacyNativeColors(t *testing.T) {
	var bp Blueprint
	data := `{"name":"x","palette":{"colors":[],"nativeColors":{"accent":"#ff0000","foo":"#000","ground":"nothex"}}}`
	if err := json.Unmarshal([]byte(data), &bp); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"accent": "#ff0000"}
	if !reflect.DeepEqual(bp.Palette.YozakuraRoles, want) {
		t.Fatalf("got %v want %v", bp.Palette.YozakuraRoles, want)
	}
}

func TestYozakuraRolesWinOverLegacy(t *testing.T) {
	var bp Blueprint
	data := `{"palette":{"colors":[],"yozakuraRoles":{"ground":"#010203"},"nativeColors":{"accent":"#ff0000"}}}`
	if err := json.Unmarshal([]byte(data), &bp); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"ground": "#010203"}
	if !reflect.DeepEqual(bp.Palette.YozakuraRoles, want) {
		t.Fatalf("got %v want %v", bp.Palette.YozakuraRoles, want)
	}
}

func TestMarshalOmitsEmptyYozakuraFields(t *testing.T) {
	out, err := json.Marshal(Blueprint{Name: "x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"yozakuraRoles", "glass", "clock", "matchWallpapers", "nativeColors"} {
		if strings.Contains(string(out), k) {
			t.Errorf("output contains %s: %s", k, out)
		}
	}
}
