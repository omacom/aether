package platform

import (
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestPortalRequestPath(t *testing.T) {
	got := portalRequestPath(":1.42", "aether7")
	want := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/1_42/aether7")
	if got != want {
		t.Fatalf("portalRequestPath = %q, want %q", got, want)
	}
}

func TestToPortalFiltersSplitsWailsPatterns(t *testing.T) {
	got := toPortalFilters([]FileFilter{
		{Name: "Images", Patterns: []string{"*.jpg; *.png;"}},
		{Name: "Empty", Patterns: []string{""}},
	})
	want := []portalFilter{{
		Name:  "Images",
		Rules: []portalFilterRule{{Type: 0, Pattern: "*.jpg"}, {Type: 0, Pattern: "*.png"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("toPortalFilters = %#v, want %#v", got, want)
	}
}

func TestPortalFilterSignature(t *testing.T) {
	sig := dbus.SignatureOf(toPortalFilters([]FileFilter{{Name: "Images", Patterns: []string{"*.png"}}}))
	if sig.String() != "a(sa(us))" {
		t.Fatalf("filter signature = %s, want a(sa(us))", sig)
	}
}

func TestParsePortalResponse(t *testing.T) {
	results := map[string]dbus.Variant{
		"uris": dbus.MakeVariant([]string{"file:///home/me/My%20Pictures/a+b.png"}),
	}
	tests := []struct {
		name string
		body []interface{}
		want string
	}{
		{"selected", []interface{}{uint32(0), results}, "/home/me/My Pictures/a+b.png"},
		{"cancelled", []interface{}{uint32(1), map[string]dbus.Variant{}}, ""},
		{"ended", []interface{}{uint32(2), map[string]dbus.Variant{}}, ""},
		{"no uris", []interface{}{uint32(0), map[string]dbus.Variant{}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortalResponse(tt.body)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePortalResponseRejectsMalformedBody(t *testing.T) {
	if _, err := parsePortalResponse([]interface{}{uint32(0)}); err == nil {
		t.Fatal("expected error for short body")
	}
	if _, err := fileURIToPath("https://example.com/a.png"); err == nil {
		t.Fatal("expected error for non-file uri")
	}
}
