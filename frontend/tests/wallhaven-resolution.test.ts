import {expect, test, vi} from 'vitest';
import {
    GetWallhavenConfig,
    SaveWallhavenConfig,
    SearchWallhaven,
} from '../wailsjs/go/main/App';
import type {wallhaven} from '../wailsjs/go/models';
import {deferred, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    GetWallhavenConfig: vi.fn(),
    SetWallhavenAPIKey: vi.fn(),
    SearchWallhaven: vi.fn(),
    SaveWallhavenConfig: vi.fn().mockResolvedValue(undefined),
}));

function result(): wallhaven.SearchResult {
    return {data: [], meta: {last_page: 0, total: 0}} as wallhaven.SearchResult;
}

async function freshStore(config: Record<string, unknown>) {
    vi.mocked(GetWallhavenConfig).mockResolvedValueOnce(config);
    vi.resetModules();
    return import('../src/lib/stores/wallhaven.svelte');
}

async function renderFilters(configPromise: Promise<Record<string, unknown>>) {
    vi.mocked(GetWallhavenConfig).mockReturnValueOnce(configPromise);
    vi.resetModules();
    const setup = await import('./setup');
    const {default: WallhavenFilters} = await import(
        '../src/lib/components/wallhaven/WallhavenFilters.svelte'
    );
    const view = setup.render(WallhavenFilters, {});
    return {
        ...view,
        settle: setup.settle,
    };
}

function resolutionSelect(target: HTMLElement): HTMLSelectElement {
    const select = target.querySelector<HTMLSelectElement>(
        'select[aria-label="Resolution filter"]'
    );
    if (!select) throw new Error('Missing resolution filter select');
    return select;
}

test('restores an exact resolution from config and sends it as resolutions', async () => {
    vi.mocked(SearchWallhaven).mockResolvedValue(result());
    const store = await freshStore({
        resolutions: '5120x1440',
        resolutionExact: true,
    });

    await store.search();
    expect(store.getResolutionSelection()).toBe('exact:5120x1440');
    expect(SearchWallhaven).toHaveBeenLastCalledWith(
        expect.objectContaining({atleast: '', resolutions: '5120x1440'})
    );
});

test('minimum and unrestricted filters keep using atleast', async () => {
    vi.mocked(SearchWallhaven).mockResolvedValue(result());

    const minimum = await freshStore({
        resolutions: '3840x2160',
        resolutionExact: false,
    });
    await minimum.search();
    expect(minimum.getResolutionSelection()).toBe('3840x2160');
    expect(SearchWallhaven).toHaveBeenLastCalledWith(
        expect.objectContaining({atleast: '3840x2160', resolutions: ''})
    );

    const any = await freshStore({resolutions: '', resolutionExact: false});
    await any.search();
    expect(any.getResolutionSelection()).toBe('');
    expect(SearchWallhaven).toHaveBeenLastCalledWith(
        expect.objectContaining({atleast: '', resolutions: ''})
    );
});

test('saving an exact choice persists the raw resolution plus the exact flag', async () => {
    const store = await freshStore({sorting: 'date_added'});

    store.setResolution('exact:3440x1440');
    await settle();

    expect(store.getResolutionSelection()).toBe('exact:3440x1440');
    expect(SaveWallhavenConfig).toHaveBeenLastCalledWith(
        expect.objectContaining({
            resolutions: '3440x1440',
            resolutionExact: true,
        })
    );
});

test('first launch defaults to the detected display resolution as a minimum filter', async () => {
    vi.mocked(SearchWallhaven).mockResolvedValue(result());
    const store = await freshStore({detectedResolution: '3440x1440'});

    await store.search();
    expect(store.getDetectedResolution()).toBe('3440x1440');
    expect(store.getResolutionSelection()).toBe('3440x1440');
    expect(SearchWallhaven).toHaveBeenLastCalledWith(
        expect.objectContaining({atleast: '3440x1440', resolutions: ''})
    );
});

test('saved resolutions including Any resolution are not overridden by display detection', async () => {
    vi.mocked(SearchWallhaven).mockResolvedValue(result());

    const any = await freshStore({
        resolutions: '',
        resolutionExact: false,
        detectedResolution: '3440x1440',
    });
    await any.search();
    expect(any.getResolutionSelection()).toBe('');
    expect(any.getDetectedResolution()).toBe('3440x1440');

    const saved = await freshStore({
        resolutions: '2560x1440',
        resolutionExact: false,
        detectedResolution: '3440x1440',
    });
    await saved.search();
    expect(saved.getResolutionSelection()).toBe('2560x1440');
});

test('dropdown adds a non-preset detected display resolution and selects it after async config load', async () => {
    const pendingConfig = deferred<Record<string, unknown>>();
    const view = await renderFilters(pendingConfig.promise);
    try {
        pendingConfig.resolve({detectedResolution: '2560x1600'});
        await view.settle();

        const select = resolutionSelect(view.target);
        const options = [...select.querySelectorAll('option')].map(o => ({
            value: o.value,
            label: o.textContent?.trim(),
        }));

        expect(select.value).toBe('2560x1600');
        expect(options).toContainEqual({
            value: '2560x1600',
            label: '2560×1600+ (Display)',
        });
        expect(options).toContainEqual({
            value: 'exact:2560x1600',
            label: '2560×1600 (Display)',
        });
    } finally {
        await view.destroy();
    }
});

test('dropdown preserves a custom saved resolution that is not a preset or current display', async () => {
    const view = await renderFilters(
        Promise.resolve({
            resolutions: '2560x1080',
            resolutionExact: false,
            detectedResolution: '3440x1440',
        })
    );
    try {
        await view.settle();

        const select = resolutionSelect(view.target);
        expect(select.value).toBe('2560x1080');
        const labels = [...select.querySelectorAll('option')].map(o =>
            o.textContent?.trim()
        );
        expect(labels).toContain('2560×1080+');
        expect(labels).toContain('3440×1440+ (Display)');
    } finally {
        await view.destroy();
    }
});
