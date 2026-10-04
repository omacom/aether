import {
    DEFAULT_PALETTE,
    DEFAULT_ADJUSTMENTS,
    AUTOMATIC_ICON_THEME,
    normalizeIconThemeSelection,
    type IconThemeSelection,
    type Adjustments,
    type ColorRoles,
} from '$lib/types/theme';
import {
    pushState,
    clearHistory,
    copySnapshot,
    type PendingAdjustment,
    type Snapshot,
} from '$lib/stores/history.svelte';
import {debounce} from '$lib/utils/debounce';
import {showToast} from '$lib/stores/ui.svelte';
import {buildCurveLUT, applyCurveToColors} from '$lib/utils/canvas-filters';

// Extraction follows this revision. Adjustment jobs are invalidated separately
// so unrelated wallpaper/light-mode edits do not discard valid calculations.
let themeRevision = 0;
export function getThemeRevision(): number {
    return themeRevision;
}
export function invalidateThemeRequests(cancelAdjustment = true): number {
    if (cancelAdjustment) cancelPendingAdjustment();
    return ++themeRevision;
}

// --- Reactive state ---
let palette = $state<string[]>([...DEFAULT_PALETTE]);
let basePalette = $state<string[]>([...DEFAULT_PALETTE]);
let wallpaperPath = $state<string>('');
let wallpaperBlur = $state(false);
let wallpaperRevision = $state(0);
let blurPreview = $state<{source: string; path: string} | null>(null);
let lightMode = $state<boolean>(false);
let lockedColors = $state<Record<number, boolean>>({});
let selectedColors = $state<Record<number, boolean>>({}); // empty = all selected
let selectedExtColors = $state<Record<string, boolean>>({}); // extended color selection
let adjustments = $state<Adjustments>({...DEFAULT_ADJUSTMENTS});
let extractionMode = $state<string>('normal');
let pendingExtractionMode = $state<string | null>(null);
let pendingAdjustment = $state.raw<PendingAdjustment | null>(null);
let isExtracting = $state<boolean>(false);
let isApplying = $state<boolean>(false);
let additionalImages = $state<string[]>([]);
let appOverrides = $state<Record<string, Record<string, string>>>({});
let nativeColors = $state<Record<string, string>>({});
let iconTheme = $state<IconThemeSelection>({...AUTOMATIC_ICON_THEME});
let paletteCurvePoints = $state<[number, number][]>([]);
// Source path of the most recently extracted palette. Used to decide
// whether per-app template overrides should be cleared on the next
// extraction (a fresh image invalidates color choices made for the
// previous palette).
let lastExtractedPath = $state<string>('');

// Extended colors — independent from palette, initialized from palette on extract/load
let extendedColors = $state<Record<string, string>>({
    accent: DEFAULT_PALETTE[4],
    cursor: DEFAULT_PALETTE[7],
    selection_foreground: DEFAULT_PALETTE[0],
    selection_background: DEFAULT_PALETTE[7],
});
let baseExtendedColors = $state<Record<string, string>>({
    accent: DEFAULT_PALETTE[4],
    cursor: DEFAULT_PALETTE[7],
    selection_foreground: DEFAULT_PALETTE[0],
    selection_background: DEFAULT_PALETTE[7],
});

// --- Derived ---
let colorRoles = $derived<ColorRoles>(buildColorRoles(palette, extendedColors));

function buildColorRoles(p: string[], ext: Record<string, string>): ColorRoles {
    return {
        background: p[0],
        foreground: p[7],
        black: p[0],
        red: p[1],
        green: p[2],
        yellow: p[3],
        blue: p[4],
        magenta: p[5],
        cyan: p[6],
        white: p[7],
        bright_black: p[8],
        bright_red: p[9],
        bright_green: p[10],
        bright_yellow: p[11],
        bright_blue: p[12],
        bright_magenta: p[13],
        bright_cyan: p[14],
        bright_white: p[15],
        accent: ext.accent,
        cursor: ext.cursor,
        selection_foreground: ext.selection_foreground,
        selection_background: ext.selection_background,
    };
}

// --- Getters ---
export function getPalette(): string[] {
    return palette;
}
export function getBasePalette(): string[] {
    return basePalette;
}
export function getWallpaperPath(): string {
    return wallpaperPath;
}
export function getWallpaperBlur(): boolean {
    return wallpaperBlur;
}
export function getWallpaperRevision(): number {
    return wallpaperRevision;
}
export function getBlurredWallpaperPath(): string {
    return wallpaperBlur && blurPreview?.source === wallpaperPath
        ? blurPreview.path
        : '';
}
export function setBlurredWallpaper(
    source: string,
    path: string,
    revision: number
): void {
    if (
        wallpaperBlur &&
        source === wallpaperPath &&
        revision === wallpaperRevision
    )
        blurPreview = {source, path};
}
export function setWallpaperBlur(enabled: boolean, skipHistory = false): void {
    if (wallpaperBlur === enabled) return;
    endColorEditSessions();
    if (!skipHistory) pushState(getHistorySnapshot());
    wallpaperBlur = enabled;
    wallpaperRevision++;
}
export function getLightMode(): boolean {
    return lightMode;
}
export function getLockedColors(): Record<number, boolean> {
    return lockedColors;
}
export function getSelectedColors(): Record<number, boolean> {
    return selectedColors;
}
export function hasColorSelection(): boolean {
    return Object.values(selectedColors).some(v => v);
}
export function toggleColorSelection(index: number): void {
    invalidateThemeRequests();
    selectedColors = {...selectedColors, [index]: !selectedColors[index]};
}
export function clearColorSelection(): void {
    invalidateThemeRequests();
    selectedColors = {};
    selectedExtColors = {};
}
export function getSelectedExtColors(): Record<string, boolean> {
    return selectedExtColors;
}
export function hasExtColorSelection(): boolean {
    return Object.values(selectedExtColors).some(v => v);
}
export function toggleExtColorSelection(key: string): void {
    invalidateThemeRequests();
    selectedExtColors = {...selectedExtColors, [key]: !selectedExtColors[key]};
}
export function hasAnySelection(): boolean {
    return hasColorSelection() || hasExtColorSelection();
}
export function getAdjustments(): Adjustments {
    return adjustments;
}
export function getExtractionMode(): string {
    return extractionMode;
}
export function getPendingExtractionMode(): string | null {
    return pendingExtractionMode;
}
export function setPendingExtractionMode(mode: string | null): void {
    pendingExtractionMode = mode;
}
export function getIsAdjusting(): boolean {
    return pendingAdjustment !== null;
}
export function getIsExtracting(): boolean {
    return isExtracting;
}
export function getIsApplying(): boolean {
    return isApplying;
}
// The backend copies every background into a single flat `backgrounds/`
// directory. Two different files with the same filename cannot coexist in one
// theme, and the backend rejects the apply with a basename collision. Dedupe
// by full path and by basename before the list reaches the backend. The first
// entry wins, and the primary wallpaper counts as the first entry.
export function imageBasename(path: string): string {
    return path.split(/[\\/]/).pop() ?? path;
}

export function dedupeAdditionalImages(
    images: readonly string[],
    primaryWallpaper: string
): string[] {
    const seenPaths = new Set<string>();
    const seenNames = new Set<string>();
    if (primaryWallpaper) {
        seenPaths.add(primaryWallpaper);
        seenNames.add(imageBasename(primaryWallpaper));
    }
    const unique: string[] = [];
    for (const image of images) {
        if (!image) continue;
        const name = imageBasename(image);
        if (seenPaths.has(image) || seenNames.has(name)) continue;
        seenPaths.add(image);
        seenNames.add(name);
        unique.push(image);
    }
    return unique;
}

function sameStringList(a: readonly string[], b: readonly string[]): boolean {
    return (
        a.length === b.length && a.every((value, index) => value === b[index])
    );
}

// Drops additional images that collide with a new main wallpaper. History
// snapshots do not hold additional images, so undo cannot bring them back.
// Tell the user about each different file that was dropped.
function dropImagesCollidingWith(primary: string): void {
    const kept = dedupeAdditionalImages(additionalImages, primary);
    if (sameStringList(kept, additionalImages)) return;
    const dropped = additionalImages.filter(
        image => image !== primary && !kept.includes(image)
    );
    additionalImages = kept;
    if (dropped.length > 0) {
        const noun = dropped.length === 1 ? 'image' : 'images';
        showToast(
            `Removed ${dropped.length} additional ${noun} with the same filename as the main wallpaper`
        );
    }
}

// Order-independent comparison for values pushed from the backend. Keeping
// `applyBackendState` from rewriting identical maps/slices stops it re-arming
// App's SyncState `$effect` with an unchanged snapshot.
function sameSerialized(a: unknown, b: unknown): boolean {
    return stableStringify(a) === stableStringify(b);
}

function stableStringify(value: unknown): string {
    return JSON.stringify(value, (_key, val: unknown) =>
        val && typeof val === 'object' && !Array.isArray(val)
            ? Object.fromEntries(
                  Object.entries(val as Record<string, unknown>).sort(
                      ([a], [b]) => a.localeCompare(b)
                  )
              )
            : val
    );
}

export function getAdditionalImages(): string[] {
    return additionalImages;
}
export function getExtendedColors(): Record<string, string> {
    return extendedColors;
}
export function getNativeColors(): Record<string, string> {
    return nativeColors;
}
export function getIconTheme(): IconThemeSelection {
    return iconTheme;
}
export function setIconTheme(
    value: {mode?: string; id?: string} | null | undefined,
    skipHistory = false
): void {
    const next = normalizeIconThemeSelection(value);
    if (iconTheme.mode === next.mode && iconTheme.id === next.id) return;
    endColorEditSessions();
    if (!skipHistory) pushState(getHistorySnapshot());
    iconTheme = next;
}
export function getBaseExtendedColors(): Record<string, string> {
    return baseExtendedColors;
}
export function getAppOverrides(): Record<string, Record<string, string>> {
    return appOverrides;
}

export function getHistorySnapshot(): Snapshot {
    return copySnapshot({
        wallpaperPath,
        wallpaperBlur,
        palette,
        basePalette,
        extendedColors,
        baseExtendedColors,
        appOverrides,
        adjustments,
        paletteCurvePoints,
        extractionMode,
        pendingAdjustment,
        iconTheme,
    });
}

export function restoreHistorySnapshot(snapshot: Snapshot): void {
    invalidateThemeRequests();
    endColorEditSessions();
    const restored = copySnapshot(snapshot);
    wallpaperPath = restored.wallpaperPath;
    wallpaperBlur = restored.wallpaperBlur;
    blurPreview = null;
    wallpaperRevision++;
    // An undo can bring back a main wallpaper whose filename matches an
    // additional image. Drop those entries so the restored state still applies.
    dropImagesCollidingWith(restored.wallpaperPath);
    palette = restored.palette;
    basePalette = restored.basePalette;
    extendedColors = restored.extendedColors;
    baseExtendedColors = restored.baseExtendedColors;
    appOverrides = restored.appOverrides;
    iconTheme = restored.iconTheme;
    adjustments = restored.adjustments;
    paletteCurvePoints = restored.paletteCurvePoints;
    extractionMode = restored.extractionMode;
    pendingAdjustment = restored.pendingAdjustment;
    // Only unfinished calculations resume. Recomputing settled snapshots would
    // overwrite manual color edits made after their adjustment was applied.
    if (pendingAdjustment)
        applyPendingAdjustment(getHistorySnapshot(), pendingAdjustment);
}

function cancelPendingAdjustment(): void {
    applyPendingAdjustment.cancel();
    if (!pendingAdjustment) return;
    adjustments = {...pendingAdjustment.previousAdjustments};
    paletteCurvePoints = pendingAdjustment.previousCurvePoints.map(([x, y]) => [
        x,
        y,
    ]);
    pendingAdjustment = null;
}

export function adjustPalette(
    next: Adjustments,
    points: [number, number][] = paletteCurvePoints
): void {
    const previous = pendingAdjustment;
    invalidateThemeRequests(false);
    pendingAdjustment = {
        previousAdjustments: {
            ...(previous?.previousAdjustments ?? adjustments),
        },
        previousCurvePoints: (
            previous?.previousCurvePoints ?? paletteCurvePoints
        ).map(([x, y]) => [x, y]),
        lockedColors: {...lockedColors},
        selectedColors: {...selectedColors},
        selectedExtColors: {...selectedExtColors},
    };
    adjustments = {...next};
    paletteCurvePoints = points.map(([x, y]) => [x, y]);
    applyPendingAdjustment(getHistorySnapshot(), pendingAdjustment);
}

// The calculation belongs to editor state, not the sidebar that initiated it.
const applyPendingAdjustment = debounce(
    async (snapshot: Snapshot, job: PendingAdjustment) => {
        const isCurrent = () => pendingAdjustment === job;
        if (!isCurrent()) return;
        const base = snapshot.basePalette;
        const baseExt = snapshot.baseExtendedColors;
        const extKeys = Object.keys(baseExt);
        const paletteSelection = Object.values(job.selectedColors).some(
            Boolean
        );
        const extSelection = Object.values(job.selectedExtColors).some(Boolean);
        const curveLUT = buildCurveLUT(snapshot.paletteCurvePoints);
        try {
            const {AdjustPaletteColors} = await import(
                '../../../wailsjs/go/main/App'
            );
            if (!isCurrent()) return;
            const [result, extResult] = await Promise.all([
                extSelection && !paletteSelection
                    ? null
                    : AdjustPaletteColors(base, snapshot.adjustments),
                paletteSelection && !extSelection
                    ? null
                    : AdjustPaletteColors(
                          Object.values(baseExt),
                          snapshot.adjustments
                      ),
            ]);
            if (!isCurrent()) return;
            if (
                (result !== null &&
                    (!Array.isArray(result) ||
                        result.length !== base.length)) ||
                (extResult !== null &&
                    (!Array.isArray(extResult) ||
                        extResult.length !== extKeys.length))
            ) {
                throw new Error('Incomplete adjustment result');
            }
            // Build both outputs before publishing, and apply selection masks after curves.
            let nextPalette = snapshot.palette;
            let nextExt = snapshot.extendedColors;
            if (result) {
                const curved = curveLUT
                    ? applyCurveToColors(result, curveLUT)
                    : result;
                nextPalette = curved.map((c, i) =>
                    job.lockedColors[i] ||
                    (paletteSelection && !job.selectedColors[i])
                        ? base[i]
                        : c
                );
            }
            if (extResult) {
                const curved = curveLUT
                    ? applyCurveToColors(extResult, curveLUT)
                    : extResult;
                nextExt = Object.fromEntries(
                    extKeys.map((key, i) => [
                        key,
                        extSelection && !job.selectedExtColors[key]
                            ? baseExt[key]
                            : curved[i],
                    ])
                );
            }
            palette = nextPalette;
            extendedColors = nextExt;
            pendingAdjustment = null;
        } catch (e) {
            if (isCurrent()) {
                cancelPendingAdjustment();
                console.error('AdjustPaletteColors failed:', e);
            }
        }
    },
    75
);

// Snapshot of the fields mirrored into Go for IPC reads (aether status) and
// for constructing ApplyThemeRequest/SaveBlueprintRequest payloads.
export function getThemeSnapshot(): {
    palette: string[];
    wallpaperPath: string;
    wallpaperBlur: boolean;
    lightMode: boolean;
    extendedColors: Record<string, string>;
    nativeColors: Record<string, string>;
    appOverrides: Record<string, Record<string, string>>;
    additionalImages: string[];
    iconTheme: IconThemeSelection;
} {
    return {
        palette: [...palette],
        wallpaperPath,
        wallpaperBlur,
        lightMode,
        extendedColors: {...extendedColors},
        nativeColors: {...nativeColors},
        appOverrides: Object.fromEntries(
            Object.entries(appOverrides).map(([app, colors]) => [
                app,
                {...colors},
            ])
        ),
        additionalImages: [...additionalImages],
        iconTheme: {...iconTheme},
    };
}

// Snapshot signature is used as a cheap dirty-state and live-apply trigger.
// Field order here is fixed so JSON.stringify is stable across calls.
export function getThemeSignature(snapshot = getThemeSnapshot()): string {
    return JSON.stringify([
        snapshot.palette,
        snapshot.wallpaperPath,
        snapshot.wallpaperBlur,
        snapshot.lightMode,
        snapshot.extendedColors,
        snapshot.nativeColors,
        snapshot.appOverrides,
        snapshot.additionalImages,
        snapshot.iconTheme,
    ]);
}

let lastAppliedSignature = $state<string>('');
export function markApplied(signature = getThemeSignature()): void {
    lastAppliedSignature = signature;
}
export function getLastAppliedSignature(): string {
    return lastAppliedSignature;
}
export function isDirty(): boolean {
    // Empty signature means nothing has been applied yet in this session,
    // so suppress the dirty indicator until the first apply.
    if (!lastAppliedSignature) return false;
    return getThemeSignature() !== lastAppliedSignature;
}
export function getPaletteCurvePoints(): [number, number][] {
    return paletteCurvePoints;
}
export function setPaletteCurvePoints(pts: [number, number][]): void {
    invalidateThemeRequests();
    paletteCurvePoints = pts.map(([x, y]) => [x, y]);
}
export function setAppOverride(
    app: string,
    role: string,
    hex: string,
    recordHistory = false
): void {
    const current = appOverrides[app] || {};
    if (current[role] === hex) return;
    if (recordHistory) {
        endColorEditSessions();
        pushState(getHistorySnapshot());
    }
    appOverrides = {...appOverrides, [app]: {...current, [role]: hex}};
}
export function removeAppOverride(app: string, role: string): void {
    if (!appOverrides[app]) return;
    const {[role]: _, ...rest} = appOverrides[app];
    if (Object.keys(rest).length === 0) {
        const {[app]: __, ...remaining} = appOverrides;
        appOverrides = remaining;
    } else {
        appOverrides = {...appOverrides, [app]: rest};
    }
}
export function clearAppOverridesForApp(app: string): void {
    const {[app]: _, ...remaining} = appOverrides;
    appOverrides = remaining;
}
export function setAppOverrides(
    overrides: Record<string, Record<string, string>>
): void {
    appOverrides = overrides ? {...overrides} : {};
}

// --- Setters ---

// setPalette sets both the display palette AND the base palette.
// Also initializes extended colors from the new palette.
// Pass skipHistory=true when initializing; undo/redo uses restoreHistorySnapshot.

export function setPalette(colors: string[], skipHistory = false): void {
    if (!skipHistory) {
        pushState(getHistorySnapshot());
    }
    invalidateThemeRequests();
    endColorEditSessions();
    basePalette = [...colors];
    palette = [...colors];
    adjustments = {...DEFAULT_ADJUSTMENTS};
    paletteCurvePoints = [];
    if (!skipHistory) {
        // Initial state supplies its own extended colors.
        const ext = {
            accent: colors[4] || extendedColors.accent,
            cursor: colors[7] || extendedColors.cursor,
            selection_foreground:
                colors[0] || extendedColors.selection_foreground,
            selection_background:
                colors[7] || extendedColors.selection_background,
        };
        baseExtendedColors = {...ext};
        extendedColors = {...ext};
    }
}

// setAdjustedPalette sets only the display palette (from adjustment results).
export function setAdjustedPalette(colors: string[]): void {
    palette = [...colors];
}

// setPaletteFromExtraction is the entry point used by extract flows. It
// drops per-app template overrides when the source image changes, since
// override hex values were chosen relative to the prior palette and tend
// to look out of place on a different wallpaper.
export function setPaletteFromExtraction(path: string, colors: string[]): void {
    if (path && lastExtractedPath && path !== lastExtractedPath) {
        appOverrides = {};
    }
    lastExtractedPath = path;
    nativeColors = {};
    setPalette(colors);
}

export function setLastExtractedPath(path: string): void {
    lastExtractedPath = path;
}

// setAdjustedExtendedColors sets only the display extended colors (from adjustment results).
export function setAdjustedExtendedColors(
    colors: Record<string, string>
): void {
    extendedColors = {...colors};
}

// setExtendedColors replaces both the display and base semantic colors when
// loading persisted/imported state. Keeping them aligned lets adjustments use
// explicit roles such as an imported accent instead of the ANSI blue fallback.
export function setExtendedColors(colors: Record<string, string>): void {
    invalidateThemeRequests();
    const next = {
        accent: colors.accent || palette[4],
        cursor: colors.cursor || palette[7],
        selection_foreground: colors.selection_foreground || palette[0],
        selection_background: colors.selection_background || palette[7],
        ...colors,
    };
    extendedColors = next;
    baseExtendedColors = {...next};
}

export function setNativeColors(colors: Record<string, string>): void {
    nativeColors = colors ? {...colors} : {};
}

// Window after the last edit during which subsequent edits are folded
// into the same history snapshot. Long enough for paint-then-tweak,
// short enough that a deliberate next edit gets its own undo step.
const EDIT_SESSION_TIMEOUT_MS = 1000;

// Debounced history push for individual color edits (picker drag)
let colorEditTimer: ReturnType<typeof setTimeout> | null = null;
let colorEditSnapshotPushed = false;

export function setColor(index: number, hex: string): void {
    // Push history once at the start of a color edit session, not on every drag tick
    if (!colorEditSnapshotPushed) {
        pushState(getHistorySnapshot());
        colorEditSnapshotPushed = true;
    }
    if (colorEditTimer) clearTimeout(colorEditTimer);
    colorEditTimer = setTimeout(() => {
        colorEditSnapshotPushed = false;
    }, EDIT_SESSION_TIMEOUT_MS);

    invalidateThemeRequests();
    palette[index] = hex;
    basePalette[index] = hex;
    palette = [...palette];
    basePalette = [...basePalette];
}

let extEditSnapshotPushed = false;
let extEditTimer: ReturnType<typeof setTimeout> | null = null;

function endColorEditSessions(): void {
    if (colorEditTimer) clearTimeout(colorEditTimer);
    if (extEditTimer) clearTimeout(extEditTimer);
    colorEditTimer = null;
    extEditTimer = null;
    colorEditSnapshotPushed = false;
    extEditSnapshotPushed = false;
}

export function setExtendedColor(key: string, hex: string): void {
    if (!extEditSnapshotPushed) {
        pushState(getHistorySnapshot());
        extEditSnapshotPushed = true;
    }
    if (extEditTimer) clearTimeout(extEditTimer);
    extEditTimer = setTimeout(() => {
        extEditSnapshotPushed = false;
    }, EDIT_SESSION_TIMEOUT_MS);

    invalidateThemeRequests();
    extendedColors = {...extendedColors, [key]: hex};
    baseExtendedColors = {...baseExtendedColors, [key]: hex};
}

// clearExtendedColors removes explicit extended/semantic colors so they revert
// to their derived defaults (the Shades editor's "reset to auto"). Pushes a
// single history entry for the whole batch, so "Reset all" is one undo step.
// No-op for keys that aren't pinned.
export function clearExtendedColors(keys: string[]): void {
    const present = keys.filter(
        k => k in extendedColors || k in baseExtendedColors
    );
    if (present.length === 0) return;
    pushState(getHistorySnapshot());
    invalidateThemeRequests();
    const next = {...extendedColors};
    const base = {...baseExtendedColors};
    for (const k of present) {
        delete next[k];
        delete base[k];
    }
    extendedColors = next;
    baseExtendedColors = base;
}

export function clearExtendedColor(key: string): void {
    clearExtendedColors([key]);
}

export function setWallpaperPath(path: string): void {
    wallpaperRevision++;
    wallpaperBlur = false;
    blurPreview = null;
    invalidateThemeRequests(false);
    wallpaperPath = path;
    // An additional image whose filename matches the new wallpaper can never be
    // staged alongside it, so drop it instead of failing the whole apply.
    dropImagesCollidingWith(path);
}
export function setLightMode(enabled: boolean): void {
    invalidateThemeRequests(false);
    lightMode = enabled;
}
export function setLockedColor(index: number, locked: boolean): void {
    invalidateThemeRequests();
    lockedColors = {...lockedColors, [index]: locked};
}
export function setAdjustments(adj: Adjustments): void {
    invalidateThemeRequests();
    adjustments = {...adj};
}
export function setExtractionMode(mode: string): void {
    invalidateThemeRequests();
    extractionMode = mode;
}
export function setIsExtracting(v: boolean): void {
    isExtracting = v;
}
export function setIsApplying(v: boolean): void {
    isApplying = v;
}

export function setAdditionalImages(images: string[]): void {
    const unique = dedupeAdditionalImages(images, wallpaperPath);
    if (sameStringList(unique, additionalImages)) return;
    invalidateThemeRequests(false);
    additionalImages = unique;
}

// Returns false when the image repeats a path or a filename that the theme
// already uses, including the main wallpaper.
export function addAdditionalImage(path: string): boolean {
    if (!path || additionalImages.includes(path)) return false;
    const next = dedupeAdditionalImages(
        [...additionalImages, path],
        wallpaperPath
    );
    if (!next.includes(path)) return false;
    invalidateThemeRequests(false);
    additionalImages = next;
    return true;
}

export function removeAdditionalImage(path: string): void {
    invalidateThemeRequests(false);
    additionalImages = additionalImages.filter(p => p !== path);
}

export function swapMainWithAdditional(path: string): void {
    const idx = additionalImages.indexOf(path);
    if (idx === -1) return;
    invalidateThemeRequests(false);
    const oldMain = wallpaperPath;
    const next = [...additionalImages];
    if (oldMain) {
        next[idx] = oldMain;
    } else {
        next.splice(idx, 1);
    }
    setWallpaperPath(path);
    const unique = dedupeAdditionalImages(next, path);
    if (!sameStringList(unique, additionalImages)) additionalImages = unique;
}

// Shape of the editor snapshot the Go backend pushes over the
// `ipc-state-changed` Wails event for IPC remote control.
export interface BackendStatePayload {
    palette?: string[];
    extendedColors?: Record<string, string>;
    nativeColors?: Record<string, string>;
    iconTheme?: {mode?: string; id?: string};
    lightMode?: boolean;
    mode?: string;
    wallpaper?: string;
    wallpaperBlur?: boolean;
    adjustments?: Adjustments;
    appOverrides?: Record<string, Record<string, string>>;
    additionalImages?: string[];
}

// Mirrors a backend-pushed snapshot into local state. Values that already
// match local state are skipped, so an equal push does not cancel pending
// theme requests or send the same snapshot back to Go.
export function applyBackendState(state: BackendStatePayload): void {
    if (
        state.palette &&
        state.palette.length >= 16 &&
        !sameSerialized(state.palette, palette)
    ) {
        setPalette(state.palette);
    }
    if (
        state.extendedColors &&
        !sameSerialized(state.extendedColors, extendedColors)
    ) {
        setExtendedColors(state.extendedColors);
    }
    if (
        state.nativeColors &&
        !sameSerialized(state.nativeColors, nativeColors)
    ) {
        setNativeColors(state.nativeColors);
    }
    if (state.iconTheme) setIconTheme(state.iconTheme, true);
    if (state.lightMode !== undefined) setLightMode(state.lightMode);
    if (state.mode) setExtractionMode(state.mode);
    if (state.wallpaper !== undefined && state.wallpaper !== wallpaperPath) {
        setWallpaperPath(state.wallpaper);
    }
    if (state.wallpaperBlur !== undefined) {
        setWallpaperBlur(state.wallpaperBlur, true);
    }
    if (state.adjustments && !sameSerialized(state.adjustments, adjustments)) {
        setAdjustments(state.adjustments);
    }
    if (
        state.appOverrides &&
        !sameSerialized(state.appOverrides, appOverrides)
    ) {
        setAppOverrides(state.appOverrides);
    }
    if (state.additionalImages) setAdditionalImages(state.additionalImages);
}

// --- Shuffle (experimental) ---
// Randomly reassigns ANSI color roles 1-6 (and their bright counterparts 9-14).
// Locked colors are excluded from the shuffle.
export function shufflePalette(): void {
    const indices = [1, 2, 3, 4, 5, 6];
    const unlocked = indices.filter(i => !lockedColors[i]);
    if (unlocked.length < 2) return; // nothing to shuffle
    pushState(getHistorySnapshot());
    invalidateThemeRequests();

    // Fisher-Yates shuffle on unlocked indices
    const colors = unlocked.map(i => palette[i]);
    const brights = unlocked.map(i => palette[i + 8]);
    for (let i = colors.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [colors[i], colors[j]] = [colors[j], colors[i]];
        [brights[i], brights[j]] = [brights[j], brights[i]];
    }

    const newPalette = [...palette];
    unlocked.forEach((idx, i) => {
        newPalette[idx] = colors[i];
        newPalette[idx + 8] = brights[i];
    });
    basePalette = [...newPalette];
    palette = [...newPalette];
}

// --- Reset ---
export function reset(): void {
    invalidateThemeRequests();
    endColorEditSessions();
    clearHistory();
    palette = [...DEFAULT_PALETTE];
    basePalette = [...DEFAULT_PALETTE];
    wallpaperPath = '';
    wallpaperBlur = false;
    wallpaperRevision++;
    blurPreview = null;
    lightMode = false;
    lockedColors = {};
    selectedColors = {};
    selectedExtColors = {};
    adjustments = {...DEFAULT_ADJUSTMENTS};
    extractionMode = 'normal';
    pendingExtractionMode = null;
    additionalImages = [];
    const ext = {
        accent: DEFAULT_PALETTE[4],
        cursor: DEFAULT_PALETTE[7],
        selection_foreground: DEFAULT_PALETTE[0],
        selection_background: DEFAULT_PALETTE[7],
    };
    extendedColors = {...ext};
    baseExtendedColors = {...ext};
    appOverrides = {};
    nativeColors = {};
    iconTheme = {...AUTOMATIC_ICON_THEME};
    paletteCurvePoints = [];
    lastExtractedPath = '';
}
