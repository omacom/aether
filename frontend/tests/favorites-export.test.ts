import {beforeEach, expect, test, vi} from 'vitest';
import {deferred, settle} from './setup';

const events = vi.hoisted(() => new Map<string, (data?: unknown) => void>());
const api = vi.hoisted(() => ({
    ExportFavorites: vi.fn(),
    IsFavoritesExportRunning: vi.fn(),
    CancelFavoritesExport: vi.fn(),
}));
vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn((name: string, callback: (data?: unknown) => void) =>
        events.set(name, callback)
    ),
    BrowserOpenURL: vi.fn(),
}));

let exports: typeof import('../src/lib/stores/favoritesExport.svelte');
const complete = {
    zipPath: '/exports/favorites.zip',
    exported: 1,
    total: 2,
    skipped: [{path: '/missing.png', reason: 'file not found'}],
};

beforeEach(async () => {
    vi.resetModules();
    events.clear();
    vi.stubGlobal('go', {main: {App: api}});
    vi.mocked(api.ExportFavorites)
        .mockReset()
        .mockResolvedValue('/exports/favorites.zip');
    vi.mocked(api.IsFavoritesExportRunning)
        .mockReset()
        .mockResolvedValue(false);
    vi.mocked(api.CancelFavoritesExport)
        .mockReset()
        .mockResolvedValue(undefined);
    exports = await import('../src/lib/stores/favoritesExport.svelte');
    expect(exports.getExportBusy()).toBe(false);
    expect(exports.getExportResult()).toBeNull();
});

function emit(name: string, data?: unknown) {
    const handler = events.get('favorites-export-' + name);
    if (!handler) throw new Error('Export event listener is missing');
    handler(data);
}

test('a delayed recovery response cannot reactivate a completed export', async () => {
    const pending = deferred<boolean>();
    vi.mocked(api.IsFavoritesExportRunning).mockReturnValue(pending.promise);
    await exports.initExportEvents();
    await settle();
    expect(api.IsFavoritesExportRunning).toHaveBeenCalledTimes(1);
    emit('completed', complete);
    pending.resolve(true);
    await settle();
    expect(exports.getExportState().active).toBe(false);
    expect(exports.getExportResult()?.skipped).toEqual(complete.skipped);
});

test('start reserves the operation and captures the selected paths', async () => {
    const pending = deferred<string>();
    vi.mocked(api.ExportFavorites).mockReturnValue(pending.promise);
    const paths = ['/one.png'];
    const first = exports.startExport(paths);
    paths.push('/later.png');
    await exports.startExport(['/two.png']);
    await settle();
    expect(api.ExportFavorites).toHaveBeenCalledExactlyOnceWith({
        paths: ['/one.png'],
    });
    expect(exports.getExportBusy()).toBe(true);
    pending.resolve('/exports/favorites.zip');
    await first;
    expect(exports.getExportState().total).toBe(1);
});

test('completion before the start response leaves the export complete', async () => {
    const pending = deferred<string>();
    vi.mocked(api.ExportFavorites).mockReturnValue(pending.promise);
    const operation = exports.startExport(['/one.png']);
    await settle();
    emit('completed', complete);
    pending.resolve(complete.zipPath);
    await operation;
    emit('progress', {phase: 'archive', index: 1, total: 1, name: 'one.png'});
    expect(exports.getExportBusy()).toBe(false);
    expect(exports.getExportResult()).toEqual(complete);
});

test('cancel waits for the terminal event and permits the next export', async () => {
    await exports.startExport(['/one.png']);
    await exports.cancelExport();
    expect(api.CancelFavoritesExport).toHaveBeenCalledTimes(1);
    expect(exports.getExportState().active).toBe(true);
    emit('cancelled');
    await exports.startExport(['/two.png']);
    expect(api.ExportFavorites).toHaveBeenCalledTimes(2);
});

test('a dismissed directory dialog releases the start guard', async () => {
    vi.mocked(api.ExportFavorites).mockRejectedValueOnce(
        new Error('export cancelled')
    );
    await exports.startExport(['/one.png']);
    expect(exports.getExportBusy()).toBe(false);
    await exports.startExport(['/two.png']);
    expect(api.ExportFavorites).toHaveBeenCalledTimes(2);
});

test('the completion panel retains skipped-file details until dismissal', async () => {
    const {mount, unmount, flushSync} = await import('svelte');
    const {default: ExportProgress} = await import(
        '../src/lib/components/favorites/ExportProgress.svelte'
    );
    await exports.initExportEvents();
    const target = document.createElement('div');
    document.body.append(target);
    const view = mount(ExportProgress, {target});
    try {
        emit('completed', complete);
        flushSync();
        expect(target.querySelector('[role="status"]')?.textContent).toContain(
            'Exported 1 of 2'
        );
        expect(target.querySelector('details')?.textContent).toContain(
            '/missing.png'
        );
        expect(target.querySelector('details')?.textContent).toContain(
            'file not found'
        );
        [...target.querySelectorAll('button')]
            .find(button => button.textContent === 'Dismiss')!
            .click();
        flushSync();
        expect(exports.getExportResult()).toBeNull();
    } finally {
        await unmount(view);
        target.remove();
    }
});
