import {showToast} from '$lib/stores/ui.svelte';
import type {main} from '../../../wailsjs/go/models';

export type ExportPhase = 'download' | 'archive';
export type ExportState = {
    active: boolean;
    phase: ExportPhase;
    index: number;
    total: number;
    name: string;
    zipPath: string;
};

export type ExportResult = {
    zipPath: string;
    total: number;
    exported: number;
    skipped: {path: string; reason: string}[] | null;
};

const IDLE: ExportState = {
    active: false,
    phase: 'download',
    index: 0,
    total: 0,
    name: '',
    zipPath: '',
};

let state = $state<ExportState>({...IDLE});
let starting = $state(false);
let result = $state<ExportResult | null>(null);
let eventsInitialization: Promise<void> | null = null;
let runSeq = 0;
let acceptsProgress = true;

export function getExportState(): ExportState {
    return state;
}

export function getExportBusy(): boolean {
    return starting || state.active;
}

export function getExportResult(): ExportResult | null {
    return result;
}

export function dismissExportResult(): void {
    result = null;
}

function folderURL(zipPath: string): string {
    const dir = zipPath.slice(0, zipPath.lastIndexOf('/'));
    return 'file://' + dir.split('/').map(encodeURIComponent).join('/');
}

export async function openExportFolder(): Promise<void> {
    if (!result) return;
    const url = folderURL(result.zipPath);
    try {
        const {BrowserOpenURL} = await import(
            '../../../wailsjs/runtime/runtime'
        );
        BrowserOpenURL(url);
    } catch {
        showToast('Could not open the export folder');
    }
}

export async function startExport(paths: string[]): Promise<void> {
    if (getExportBusy() || paths.length === 0) return;
    const selected = [...paths];
    const seq = ++runSeq;
    acceptsProgress = true;
    starting = true;
    result = null;
    try {
        await initExportEvents();
        const {ExportFavorites} = await import('../../../wailsjs/go/main/App');
        const zipPath = await ExportFavorites({
            paths: selected,
        } as unknown as main.ExportFavoritesRequest);
        if (runSeq !== seq) return;
        state = state.active
            ? {...state, zipPath}
            : {...IDLE, active: true, total: selected.length, zipPath};
    } catch (error: unknown) {
        if (!state.active) acceptsProgress = false;
        const message = error instanceof Error ? error.message : String(error);
        if (!message.includes('cancelled'))
            showToast(message || 'Export failed');
    } finally {
        starting = false;
    }
}

export async function cancelExport(): Promise<void> {
    if (!state.active) return;
    try {
        const {CancelFavoritesExport} = await import(
            '../../../wailsjs/go/main/App'
        );
        await CancelFavoritesExport();
    } catch {
        showToast('Could not cancel the export. Try again.');
    }
}

// Install listeners before a fast export can emit its completion event.
export function initExportEvents(): Promise<void> {
    if (!eventsInitialization) {
        eventsInitialization = subscribeExportEvents().catch(error => {
            eventsInitialization = null;
            throw error;
        });
    }
    return eventsInitialization;
}

async function subscribeExportEvents(): Promise<void> {
    const {EventsOn, BrowserOpenURL} = await import(
        '../../../wailsjs/runtime/runtime'
    );
    EventsOn(
        'favorites-export-progress',
        (progress: {
            phase: ExportPhase;
            index: number;
            total: number;
            name: string;
        }) => {
            if (!acceptsProgress) return;
            state = {...state, active: true, ...progress};
        }
    );
    EventsOn('favorites-export-completed', (completed: ExportResult) => {
        runSeq++;
        acceptsProgress = false;
        state = {...IDLE};
        result = {...completed, skipped: completed.skipped ?? []};
        showToast(
            `Exported ${completed.exported} of ${completed.total} favorites`,
            {
                duration: 8000,
                action: {
                    label: 'Open folder',
                    run: () => BrowserOpenURL(folderURL(completed.zipPath)),
                },
            }
        );
    });
    EventsOn('favorites-export-failed', (failure: {error: string}) => {
        runSeq++;
        acceptsProgress = false;
        state = {...IDLE};
        showToast(failure?.error || 'Export failed');
    });
    EventsOn('favorites-export-cancelled', () => {
        runSeq++;
        acceptsProgress = false;
        state = {...IDLE};
        showToast('Export cancelled');
    });
    void recoverExportState(runSeq);
}

async function recoverExportState(sequence: number): Promise<void> {
    try {
        const {IsFavoritesExportRunning} = await import(
            '../../../wailsjs/go/main/App'
        );
        const running = await IsFavoritesExportRunning();
        if (sequence === runSeq && running) state = {...state, active: true};
    } catch {
        // A later progress event can still recover an active export.
    }
}
