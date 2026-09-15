package theme

import (
	"fmt"

	"aether/internal/platform"
	"aether/internal/wallpaper"
)

// materializeWallpaper keeps the editor source intact and resolves a derived output copy.
func materializeWallpaper(state *ThemeState) (*ThemeState, error) {
	if !state.WallpaperBlur {
		return state, nil
	}
	path, err := wallpaper.CreateBlurredVariant(state.WallpaperPath, platform.BlurDir())
	if err != nil {
		return nil, fmt.Errorf("create wallpaper variant: %w", err)
	}
	rendered := *state
	rendered.WallpaperPath = path
	rendered.OriginalWallpaperPath = state.WallpaperPath
	rendered.WallpaperBlur = false
	return &rendered, nil
}
