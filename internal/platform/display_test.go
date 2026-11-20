package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHyprlandMonitors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		wantW int
		wantH int
	}{
		{
			name:  "ignores fractional scale and returns physical pixels",
			input: `[{"width":3440,"height":1440,"scale":1.25,"transform":0,"focused":true,"disabled":false}]`,
			wantW: 3440,
			wantH: 1440,
		},
		{
			name: "prefers focused enabled monitor",
			input: `[
				{"width":1920,"height":1080,"transform":0,"focused":false,"disabled":false},
				{"width":3440,"height":1440,"transform":0,"focused":true,"disabled":false}
			]`,
			wantW: 3440,
			wantH: 1440,
		},
		{
			name: "skips disabled monitors",
			input: `[
				{"width":3840,"height":2160,"transform":0,"focused":true,"disabled":true},
				{"width":2560,"height":1600,"transform":0,"focused":false,"disabled":false}
			]`,
			wantW: 2560,
			wantH: 1600,
		},
		{
			name:  "swaps dimensions for 90 and 270 degree rotations",
			input: `[{"width":2560,"height":1440,"transform":1,"focused":true,"disabled":false}]`,
			wantW: 1440,
			wantH: 2560,
		},
		{
			name:  "handles empty monitor list",
			input: `[]`,
			wantW: 0,
			wantH: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, h := parseHyprlandMonitors([]byte(tc.input))
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("parseHyprlandMonitors = (%d, %d); want (%d, %d)", w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestParseSwayOutputs(t *testing.T) {
	input := `[
		{"active":false,"focused":true,"transform":"normal","current_mode":{"width":1920,"height":1080}},
		{"active":true,"focused":false,"transform":"normal","current_mode":{"width":1920,"height":1080}},
		{"active":true,"focused":true,"transform":"270","current_mode":{"width":2560,"height":1440}}
	]`
	w, h := parseSwayOutputs([]byte(input))
	if w != 1440 || h != 2560 {
		t.Fatalf("parseSwayOutputs = (%d, %d); want (1440, 2560)", w, h)
	}
}

func TestParseWlrRandrOutputs(t *testing.T) {
	input := `[
		{"enabled":false,"transform":"normal","modes":[{"width":1920,"height":1080,"current":true}]},
		{"enabled":true,"transform":"normal","modes":[
			{"width":1920,"height":1080,"current":false},
			{"width":3440,"height":1440,"current":true}
		]}
	]`
	w, h := parseWlrRandrOutputs([]byte(input))
	if w != 3440 || h != 1440 {
		t.Fatalf("parseWlrRandrOutputs = (%d, %d); want (3440, 1440)", w, h)
	}
}

func TestParseXrandrOutput(t *testing.T) {
	input := `Screen 0: minimum 8 x 8, current 5360 x 1440, maximum 32767 x 32767
HDMI-1 connected 1920x1080+0+0 (normal left inverted right x axis y axis) 527mm x 296mm
DP-1 connected primary 3440x1440+1920+0 (normal left inverted right x axis y axis) 800mm x 340mm
DP-2 disconnected (normal left inverted right x axis y axis)
`
	w, h := parseXrandrOutput([]byte(input))
	if w != 3440 || h != 1440 {
		t.Fatalf("parseXrandrOutput = (%d, %d); want (3440, 1440)", w, h)
	}

	noPrimary := "HDMI-1 connected 2560x1600+0+0 (normal)\n"
	w, h = parseXrandrOutput([]byte(noPrimary))
	if w != 2560 || h != 1600 {
		t.Fatalf("parseXrandrOutput fallback = (%d, %d); want (2560, 1600)", w, h)
	}
}

func TestNativeMonitorResolutionStringUsesHyprctl(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '[{\"width\":3440,\"height\":1440,\"scale\":1.25,\"transform\":0,\"focused\":true,\"disabled\":false}]'\n"
	if err := os.WriteFile(filepath.Join(dir, "hyprctl"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "test-instance")
	t.Setenv("SWAYSOCK", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")

	if got := NativeMonitorResolutionString(); got != "3440x1440" {
		t.Fatalf("NativeMonitorResolutionString() = %q; want %q", got, "3440x1440")
	}
}
