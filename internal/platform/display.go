package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// NativeMonitorResolution returns the unscaled physical pixel dimensions of the
// focused/primary enabled display. Returns (0, 0) if detection is unavailable.
func NativeMonitorResolution() (int, int) {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		if out, err := exec.Command("hyprctl", "monitors", "-j").Output(); err == nil {
			if w, h := parseHyprlandMonitors(out); w > 0 && h > 0 {
				return w, h
			}
		}
	}
	if os.Getenv("SWAYSOCK") != "" {
		if out, err := exec.Command("swaymsg", "-t", "get_outputs").Output(); err == nil {
			if w, h := parseSwayOutputs(out); w > 0 && h > 0 {
				return w, h
			}
		}
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if out, err := exec.Command("wlr-randr", "--json").Output(); err == nil {
			if w, h := parseWlrRandrOutputs(out); w > 0 && h > 0 {
				return w, h
			}
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if out, err := exec.Command("xrandr", "--current").Output(); err == nil {
			if w, h := parseXrandrOutput(out); w > 0 && h > 0 {
				return w, h
			}
		}
	}
	return 0, 0
}

// NativeMonitorResolutionString returns the native display resolution formatted
// as "WxH" (e.g. "3440x1440"), or "" if detection is unavailable.
func NativeMonitorResolutionString() string {
	w, h := NativeMonitorResolution()
	if w <= 0 || h <= 0 {
		return ""
	}
	return fmt.Sprintf("%dx%d", w, h)
}

func parseHyprlandMonitors(data []byte) (int, int) {
	var monitors []struct {
		Width     int  `json:"width"`
		Height    int  `json:"height"`
		Transform int  `json:"transform"`
		Focused   bool `json:"focused"`
		Disabled  bool `json:"disabled"`
	}
	if json.Unmarshal(data, &monitors) != nil {
		return 0, 0
	}
	chosen := -1
	for i, mon := range monitors {
		if mon.Disabled || mon.Width <= 0 || mon.Height <= 0 {
			continue
		}
		if chosen == -1 || mon.Focused {
			chosen = i
		}
		if mon.Focused {
			break
		}
	}
	if chosen == -1 {
		return 0, 0
	}
	m := monitors[chosen]
	if m.Transform%2 != 0 {
		return m.Height, m.Width
	}
	return m.Width, m.Height
}

func parseSwayOutputs(data []byte) (int, int) {
	var outputs []struct {
		Active      bool   `json:"active"`
		Focused     bool   `json:"focused"`
		Transform   string `json:"transform"`
		CurrentMode struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"current_mode"`
	}
	if json.Unmarshal(data, &outputs) != nil {
		return 0, 0
	}
	chosen := -1
	for i, out := range outputs {
		if !out.Active || out.CurrentMode.Width <= 0 || out.CurrentMode.Height <= 0 {
			continue
		}
		if chosen == -1 || out.Focused {
			chosen = i
		}
		if out.Focused {
			break
		}
	}
	if chosen == -1 {
		return 0, 0
	}
	out := outputs[chosen]
	if isQuarterTurn(out.Transform) {
		return out.CurrentMode.Height, out.CurrentMode.Width
	}
	return out.CurrentMode.Width, out.CurrentMode.Height
}

func parseWlrRandrOutputs(data []byte) (int, int) {
	var outputs []struct {
		Enabled   bool   `json:"enabled"`
		Transform string `json:"transform"`
		Modes     []struct {
			Width   int  `json:"width"`
			Height  int  `json:"height"`
			Current bool `json:"current"`
		} `json:"modes"`
	}
	if json.Unmarshal(data, &outputs) != nil {
		return 0, 0
	}
	for _, out := range outputs {
		if !out.Enabled {
			continue
		}
		for _, mode := range out.Modes {
			if !mode.Current || mode.Width <= 0 || mode.Height <= 0 {
				continue
			}
			if isQuarterTurn(out.Transform) {
				return mode.Height, mode.Width
			}
			return mode.Width, mode.Height
		}
	}
	return 0, 0
}

func isQuarterTurn(transform string) bool {
	switch strings.ToLower(strings.TrimSpace(transform)) {
	case "90", "270", "flipped-90", "flipped-270":
		return true
	default:
		return false
	}
}

var xrandrGeomRe = regexp.MustCompile(`\b(\d+)x(\d+)\+\d+\+\d+\b`)

func parseXrandrOutput(data []byte) (int, int) {
	var fallbackW, fallbackH int
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, " connected ") {
			continue
		}
		match := xrandrGeomRe.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		w, errW := strconv.Atoi(match[1])
		h, errH := strconv.Atoi(match[2])
		if errW != nil || errH != nil || w <= 0 || h <= 0 {
			continue
		}
		if strings.Contains(line, " primary ") {
			return w, h
		}
		if fallbackW == 0 {
			fallbackW, fallbackH = w, h
		}
	}
	return fallbackW, fallbackH
}
