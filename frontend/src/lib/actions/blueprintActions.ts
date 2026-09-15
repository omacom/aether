import type {Blueprint} from '$lib/types/theme';
import {setActiveTab, setLiveApply, showToast} from '$lib/stores/ui.svelte';
import {
    setPalette,
    setExtendedColors,
    setNativeColors,
    setIconTheme,
    setLightMode,
    setWallpaperPath,
    setWallpaperBlur,
    setAppOverrides,
    setAdditionalImages,
    setLastExtractedPath,
    setLockedColor,
} from '$lib/stores/theme.svelte';

export function loadBlueprintIntoEditor(bp: Blueprint): void {
    const colors = bp.palette?.colors;
    if (
        colors?.length !== 16 ||
        colors.some(color => !/^#[0-9a-f]{6}$/i.test(color))
    ) {
        throw new Error('Blueprint must contain 16 valid hex colors');
    }

    // Loading is a review action, never an implicit desktop apply. The saved
    // colors are already adjusted; without saved bases they become a fresh base.
    setLiveApply(false);
    setPalette(colors);
    setExtendedColors(bp.palette.extendedColors ?? {});
    setNativeColors(bp.palette.nativeColors ?? {});
    setIconTheme(bp.iconTheme, true);
    setLightMode(
        bp.palette.mode ? bp.palette.mode === 'light' : !!bp.palette.lightMode
    );
    setWallpaperPath(bp.palette.wallpaper ?? '');
    setWallpaperBlur(!!bp.palette.wallpaperBlur, true);
    setAppOverrides(bp.appOverrides ?? {});
    setAdditionalImages(bp.palette.additionalImages ?? []);
    setLastExtractedPath(bp.palette.wallpaper ?? '');
    for (let i = 0; i < 16; i++) {
        setLockedColor(i, bp.palette.lockedColors?.includes(i) ?? false);
    }
    setActiveTab('editor');
    showToast(`Loaded: ${bp.name}. Review, then apply when ready.`);
}
