import {expect, test, vi} from 'vitest';
import {
    GetWallhavenConfig,
    SaveWallhavenConfig,
    SearchWallhaven,
} from '../wailsjs/go/main/App';
import type {wallhaven} from '../wailsjs/go/models';
import {settle} from './setup';

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
