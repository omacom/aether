// Frontend-side undo/redo history

import type {Adjustments, IconThemeSelection} from '$lib/types/theme';

const MAX_HISTORY = 50;

// Keep unfinished intent's target mask and the metadata matching its displayed
// colors. Redo resumes that intent; failures roll back to the displayed metadata.
export interface PendingAdjustment {
    previousAdjustments: Adjustments;
    previousCurvePoints: [number, number][];
    lockedColors: Record<number, boolean>;
    selectedColors: Record<number, boolean>;
    selectedExtColors: Record<string, boolean>;
}

export interface Snapshot {
    wallpaperPath: string;
    wallpaperBlur: boolean;
    palette: string[];
    basePalette: string[];
    extendedColors: Record<string, string>;
    baseExtendedColors: Record<string, string>;
    appOverrides: Record<string, Record<string, string>>;
    iconTheme: IconThemeSelection;
    adjustments: Adjustments;
    paletteCurvePoints: [number, number][];
    extractionMode: string;
    pendingAdjustment: PendingAdjustment | null;
}

let undoStack = $state<Snapshot[]>([]);
let redoStack = $state<Snapshot[]>([]);

let canUndo = $state(false);
let canRedo = $state(false);

function updateFlags() {
    canUndo = undoStack.length > 0;
    canRedo = redoStack.length > 0;
}

export function getCanUndo(): boolean {
    return canUndo;
}
export function getCanRedo(): boolean {
    return canRedo;
}

export function copySnapshot(snapshot: Snapshot): Snapshot {
    const pending = snapshot.pendingAdjustment;
    return {
        wallpaperPath: snapshot.wallpaperPath,
        wallpaperBlur: snapshot.wallpaperBlur,
        palette: [...snapshot.palette],
        basePalette: [...snapshot.basePalette],
        extendedColors: {...snapshot.extendedColors},
        baseExtendedColors: {...snapshot.baseExtendedColors},
        appOverrides: Object.fromEntries(
            Object.entries(snapshot.appOverrides).map(([app, colors]) => [
                app,
                {...colors},
            ])
        ),
        adjustments: {...snapshot.adjustments},
        paletteCurvePoints: snapshot.paletteCurvePoints.map(([x, y]) => [x, y]),
        extractionMode: snapshot.extractionMode,
        iconTheme: {...snapshot.iconTheme},
        pendingAdjustment: pending
            ? {
                  previousAdjustments: {...pending.previousAdjustments},
                  previousCurvePoints: pending.previousCurvePoints.map(
                      ([x, y]) => [x, y]
                  ),
                  lockedColors: {...pending.lockedColors},
                  selectedColors: {...pending.selectedColors},
                  selectedExtColors: {...pending.selectedExtColors},
              }
            : null,
    };
}

export function pushState(snapshot: Snapshot): void {
    pushUndo(snapshot);
    redoStack = [];
    updateFlags();
}

// Push to undo without clearing redo (used during redo operations)
export function pushUndo(snapshot: Snapshot): void {
    undoStack = [
        ...undoStack.slice(-(MAX_HISTORY - 1)),
        copySnapshot(snapshot),
    ];
    updateFlags();
}

export function undo(): Snapshot | null {
    if (undoStack.length === 0) return null;
    const snapshot = undoStack[undoStack.length - 1];
    undoStack = undoStack.slice(0, -1);
    updateFlags();
    return snapshot;
}

export function pushRedo(snapshot: Snapshot): void {
    redoStack = [
        ...redoStack.slice(-(MAX_HISTORY - 1)),
        copySnapshot(snapshot),
    ];
    updateFlags();
}

export function redo(): Snapshot | null {
    if (redoStack.length === 0) return null;
    const snapshot = redoStack[redoStack.length - 1];
    redoStack = redoStack.slice(0, -1);
    updateFlags();
    return snapshot;
}

export function clearHistory(): void {
    undoStack = [];
    redoStack = [];
    updateFlags();
}
