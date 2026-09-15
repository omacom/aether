import type {main} from '../../../wailsjs/go/models';
import {
    showToast,
    getLiveApply,
    getLiveApplySession,
    setApplySaveDialogOpen,
} from '$lib/stores/ui.svelte';
import {
    getIsApplying,
    setIsApplying,
    getIsExtracting,
    getIsAdjusting,
    setIsExtracting,
    getWallpaperPath,
    setWallpaperPath,
    setPaletteFromExtraction,
    getLightMode,
    getAdditionalImages,
    getAppOverrides,
    getThemeSnapshot,
    getThemeSignature,
    getHistorySnapshot,
    restoreHistorySnapshot,
    getThemeRevision,
    invalidateThemeRequests,
    getExtractionMode,
    setExtractionMode,
    setPendingExtractionMode,
    markApplied,
} from '$lib/stores/theme.svelte';
import {getSettings} from '$lib/stores/settings.svelte';
import type {Settings} from '$lib/types/theme';
import {
    undo as historyUndo,
    redo as historyRedo,
    pushRedo,
    pushUndo,
} from '$lib/stores/history.svelte';
import {STORAGE_KEYS} from '$lib/constants/storage';
import {
    getOmarchyCapabilities,
    getOmarchyAvailable,
    initOmarchyCapabilities,
} from '$lib/stores/omarchy.svelte';

export function getNativeAppOverrides(): Record<
    string,
    Record<string, string>
> {
    const overrides = getAppOverrides();
    return filterNativeAppOverrides(overrides);
}

function filterNativeAppOverrides(
    overrides: Record<string, Record<string, string>>
) {
    const supported = new Set(getOmarchyCapabilities().overrideApps);
    return Object.fromEntries(
        Object.entries(overrides).filter(([app]) => supported.has(app))
    );
}

function captureApplyRequest() {
    const settings = getSettings();
    return {
        ...getThemeSnapshot(),
        settings: {...settings, includedApps: {...settings.includedApps}},
    };
}

function countTargetedApps(
    settings: Settings,
    appOverrides: Record<string, Record<string, string>>
): number {
    if (getOmarchyAvailable()) {
        return Object.values(appOverrides).filter(
            overrides => Object.keys(overrides).length > 0
        ).length;
    }
    const targeted = new Set(
        Object.entries(settings.includedApps ?? {})
            .filter(([, included]) => included)
            .map(([app]) => app)
    );
    for (const [app, overrides] of Object.entries(appOverrides)) {
        if (Object.keys(overrides).length > 0) targeted.add(app);
    }
    return targeted.size;
}

async function runApply(
    request: ReturnType<typeof captureApplyRequest>,
    options: {
        name?: string;
        updateExisting?: boolean;
        isCurrent?: () => boolean;
    } = {}
): Promise<{success: boolean; count: number} | null> {
    const signature = getThemeSignature(request);
    await initOmarchyCapabilities();
    const {ApplyTheme, SaveAndApplyTheme} = await import(
        '../../../wailsjs/go/main/App'
    );
    if (options.isCurrent && !options.isCurrent()) return null;
    const appOverrides = getOmarchyAvailable()
        ? filterNativeAppOverrides(request.appOverrides)
        : request.appOverrides;
    const payload = {...request, appOverrides};
    const count = countTargetedApps(request.settings, appOverrides);
    const result =
        options.name !== undefined
            ? await SaveAndApplyTheme({
                  ...payload,
                  name: options.name,
                  updateExisting: !!options.updateExisting,
              } as unknown as main.SaveAndApplyThemeRequest)
            : await ApplyTheme(payload as unknown as main.ApplyThemeRequest);
    if (result.success) {
        markApplied(signature);
        document.documentElement.classList.toggle(
            'light-mode',
            request.lightMode
        );
    }
    return {success: !!result.success, count};
}

export async function applyTheme(): Promise<void> {
    if (getIsApplying()) return;
    const request = captureApplyRequest();
    setIsApplying(true);
    try {
        const result = await runApply(request);
        if (!result) return;
        const suffix = result.count
            ? ` with ${result.count} app override${result.count === 1 ? '' : 's'}`
            : '';
        if (result.success) {
            showToast(`Theme applied${suffix}`);
        } else {
            showToast(`Theme files generated${suffix}`);
        }
    } catch {
        showToast('Couldn’t apply theme — see logs for details');
    } finally {
        setIsApplying(false);
    }
}

function getSavedThemeFolder(): string {
    const wallpaper = getWallpaperPath();
    if (!wallpaper) return '';
    try {
        const folders = JSON.parse(
            localStorage.getItem(STORAGE_KEYS.savedThemeFolders) ?? '{}'
        ) as Record<string, string>;
        return folders[wallpaper] ?? '';
    } catch {
        return '';
    }
}

function saveThemeFolder(name: string, wallpaper: string): void {
    if (!wallpaper) return;
    try {
        const folders = JSON.parse(
            localStorage.getItem(STORAGE_KEYS.savedThemeFolders) ?? '{}'
        ) as Record<string, string>;
        localStorage.setItem(
            STORAGE_KEYS.savedThemeFolders,
            JSON.stringify({...folders, [wallpaper]: name})
        );
    } catch {}
}

// The primary Apply action updates the folder saved for this wallpaper.
// Ctrl+Enter remains an intentional bypass for quickly applying editor state.
export function requestThemeApply(): void {
    if (getIsApplying()) return;
    const savedFolder = getSavedThemeFolder();
    if (savedFolder) {
        saveAndApplyTheme(savedFolder, true);
    } else {
        saveThemeAsNew();
    }
}

export function saveThemeAsNew(): void {
    if (!getIsApplying()) setApplySaveDialogOpen(true);
}

export async function saveAndApplyTheme(
    name: string,
    updateExisting = false
): Promise<void> {
    if (getIsApplying()) return;
    const request = captureApplyRequest();
    setIsApplying(true);
    try {
        await runApply(request, {name, updateExisting});
        saveThemeFolder(name, request.wallpaperPath);
        showToast(
            updateExisting ? `Applied: ${name}` : `Saved and applied: ${name}`
        );
    } catch (e: unknown) {
        showToast(
            e instanceof Error ? e.message : 'Couldn’t save and apply theme'
        );
    } finally {
        setIsApplying(false);
    }
}

// Swap the wallpaper without re-extracting colors. Resolves remote URLs
// (Wallhaven) by downloading first, then runs the standard apply path so
// the new wallpaper goes out together with the current palette.
export async function applyWallpaperOnly(originalPath: string): Promise<void> {
    if (getIsApplying() || !originalPath) return;
    const request = captureApplyRequest();
    const originalSignature = getThemeSignature(request);
    request.wallpaperPath = originalPath;
    setIsApplying(true);
    try {
        if (
            originalPath.startsWith('http://') ||
            originalPath.startsWith('https://')
        ) {
            showToast('Downloading wallpaper…');
            const {DownloadWallpaper} = await import(
                '../../../wailsjs/go/main/App'
            );
            request.wallpaperPath = await DownloadWallpaper(originalPath);
        }
        // A slow download must not replace the wallpaper of a newly loaded theme.
        if (getThemeSignature() === originalSignature)
            setWallpaperPath(request.wallpaperPath);
        const result = await runApply(request);
        if (!result) return;
        showToast(
            result.success ? 'Wallpaper applied' : 'Wallpaper files generated'
        );
    } catch {
        showToast('Couldn’t apply wallpaper — see logs for details');
    } finally {
        setIsApplying(false);
    }
}

// Quieter undo-offering toast for live preview; long enough to react,
// short enough not to pile up during rapid edits.
const LIVE_APPLY_TOAST_MS = 2200;

// Same backend call as applyTheme(), but with a quieter toast that offers
// Undo. Used by the live-preview effect when the user flips on Live Apply.
export async function applyThemeLive(): Promise<
    'applied' | 'canceled' | 'failed'
> {
    if (
        !getLiveApply() ||
        getIsApplying() ||
        getIsAdjusting() ||
        getIsExtracting()
    )
        return 'canceled';
    const session = getLiveApplySession();
    const request = captureApplyRequest();
    const signature = getThemeSignature(request);
    const isCurrent = () =>
        getLiveApply() &&
        session === getLiveApplySession() &&
        signature === getThemeSignature() &&
        !getIsAdjusting() &&
        !getIsExtracting();
    setIsApplying(true);
    try {
        const result = await runApply(request, {isCurrent});
        if (!result) return 'canceled';
        if (result.success && isCurrent()) {
            const suffix = result.count
                ? ` with ${result.count} app override${result.count === 1 ? '' : 's'}`
                : '';
            showToast(`Live preview applied${suffix}`, {
                duration: LIVE_APPLY_TOAST_MS,
                action: {label: 'Undo', run: undoAction},
            });
        }
        return result.success ? 'applied' : 'failed';
    } catch {
        // Stay quiet on transient live-apply failures; the user can hit
        // Apply manually if something is wrong.
        return 'failed';
    } finally {
        setIsApplying(false);
    }
}

export function undoAction(): void {
    const snapshot = historyUndo();
    if (!snapshot) return;
    pushRedo(getHistorySnapshot());
    restoreHistorySnapshot(snapshot);
}

export function redoAction(): void {
    const snapshot = historyRedo();
    if (!snapshot) return;
    pushUndo(getHistorySnapshot());
    restoreHistorySnapshot(snapshot);
}

export async function changeWallpaper(): Promise<void> {
    try {
        const {OpenFileDialog} = await import('../../../wailsjs/go/main/App');
        const path = await OpenFileDialog();
        if (path) {
            setWallpaperPath(path);
            showToast('Wallpaper changed — click Extract to generate palette');
        }
    } catch {
        showToast('Couldn’t open the wallpaper picker');
    }
}

let extractionRequest = 0;

export async function extractColors(
    options: {
        mode?: string;
        allImages?: boolean;
    } = {}
): Promise<void> {
    const path = getWallpaperPath();
    const paths = [path, ...getAdditionalImages()].filter(Boolean);
    if (!options.mode && (!path || getIsExtracting())) return;
    const mode = options.mode ?? getExtractionMode();
    const lightMode = getLightMode();
    const request = ++extractionRequest;
    const revision = invalidateThemeRequests();
    const isCurrent = () =>
        request === extractionRequest && revision === getThemeRevision();
    setPendingExtractionMode(options.mode ?? null);
    setIsExtracting(!!path);
    try {
        const {ExtractColors, ExtractColorsFromImages, SetExtractionMode} =
            await import('../../../wailsjs/go/main/App');
        if (!isCurrent()) return;
        if (options.mode) {
            await SetExtractionMode(mode);
            if (!isCurrent()) return;
        }
        if (!path) {
            setExtractionMode(mode);
            return;
        }
        const result = options.allImages
            ? await ExtractColorsFromImages(paths, lightMode, mode)
            : {palette: await ExtractColors(path, lightMode, mode), skipped: 0};
        if (!isCurrent()) return;
        setPaletteFromExtraction(path, result.palette);
        setExtractionMode(mode);
        if (options.allImages) {
            const used = paths.length - result.skipped;
            const suffix =
                result.skipped > 0 ? ` (${result.skipped} skipped)` : '';
            showToast(
                `Blended palette from ${used} image${used === 1 ? '' : 's'}${suffix}`
            );
            return;
        }
        showToast(
            options.mode ? `Re-extracted with ${mode} mode` : 'Colors extracted'
        );
    } catch {
        if (isCurrent()) showToast('Couldn’t extract colors from that image');
    } finally {
        if (request === extractionRequest) {
            setIsExtracting(false);
            setPendingExtractionMode(null);
        }
    }
}
