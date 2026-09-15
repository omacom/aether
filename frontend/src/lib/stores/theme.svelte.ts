import {
    DEFAULT_PALETTE,
    DEFAULT_ADJUSTMENTS,
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
export function getAdditionalImages(): string[] {
    return additionalImages;
}
export function getExtendedColors(): Record<string, string> {
    return extendedColors;
}
export function getNativeColors(): Record<string, string> {
    return nativeColors;
}
export function getBaseExtendedColors(): Record<string, string> {
    return baseExtendedColors;
}
export function getAppOverrides(): Record<string, Record<string, string>> {
    return appOverrides;
}

export function getHistorySnapshot(): Snapshot {
    return copySnapshot({
        palette,
        basePalette,
        extendedColors,
        baseExtendedColors,
        appOverrides,
        adjustments,
        paletteCurvePoints,
        extractionMode,
        pendingAdjustment,
    });
}

export function restoreHistorySnapshot(snapshot: Snapshot): void {
    invalidateThemeRequests();
    endColorEditSessions();
    const restored = copySnapshot(snapshot);
    palette = restored.palette;
    basePalette = restored.basePalette;
    extendedColors = restored.extendedColors;
    baseExtendedColors = restored.baseExtendedColors;
    appOverrides = restored.appOverrides;
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
    lightMode: boolean;
    extendedColors: Record<string, string>;
    nativeColors: Record<string, string>;
    appOverrides: Record<string, Record<string, string>>;
    additionalImages: string[];
} {
    return {
        palette: [...palette],
        wallpaperPath,
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
    };
}

// Snapshot signature is used as a cheap dirty-state and live-apply trigger.
// Field order here is fixed so JSON.stringify is stable across calls.
export function getThemeSignature(snapshot = getThemeSnapshot()): string {
    return JSON.stringify([
        snapshot.palette,
        snapshot.wallpaperPath,
        snapshot.lightMode,
        snapshot.extendedColors,
        snapshot.nativeColors,
        snapshot.appOverrides,
        snapshot.additionalImages,
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
    invalidateThemeRequests(false);
    wallpaperPath = path;
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
    invalidateThemeRequests(false);
    additionalImages = [...images];
}

export function addAdditionalImage(path: string): void {
    if (!additionalImages.includes(path)) {
        invalidateThemeRequests(false);
        additionalImages = [...additionalImages, path];
    }
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
    wallpaperPath = path;
    const next = [...additionalImages];
    if (oldMain) {
        next[idx] = oldMain;
    } else {
        next.splice(idx, 1);
    }
    additionalImages = next;
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
    paletteCurvePoints = [];
    lastExtractedPath = '';
}
