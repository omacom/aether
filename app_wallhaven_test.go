package main

import "testing"

func TestGetWallhavenConfigIncludesDetectedResolution(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	prev := nativeMonitorResolution
	nativeMonitorResolution = func() string { return "3440x1440" }
	t.Cleanup(func() { nativeMonitorResolution = prev })

	app := NewApp()
	config := app.GetWallhavenConfig()
	if got, _ := config["detectedResolution"].(string); got != "3440x1440" {
		t.Fatalf("detectedResolution = %q; want %q", got, "3440x1440")
	}

	if err := app.SaveWallhavenConfig(map[string]interface{}{
		"resolutions":     "2560x1440",
		"resolutionExact": false,
	}); err != nil {
		t.Fatal(err)
	}

	saved := app.GetWallhavenConfig()
	if got, _ := saved["resolutions"].(string); got != "2560x1440" {
		t.Errorf("resolutions = %q; want %q", got, "2560x1440")
	}
	if got, _ := saved["detectedResolution"].(string); got != "3440x1440" {
		t.Errorf("detectedResolution = %q; want %q", got, "3440x1440")
	}
}

func TestGetWallhavenConfigOmitsEmptyDetectedResolution(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	prev := nativeMonitorResolution
	nativeMonitorResolution = func() string { return "" }
	t.Cleanup(func() { nativeMonitorResolution = prev })

	config := NewApp().GetWallhavenConfig()
	if _, exists := config["detectedResolution"]; exists {
		t.Fatalf("unexpected detectedResolution key in %#v", config)
	}
}
