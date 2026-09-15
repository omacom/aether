import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import {SearchWallhaven} from '../wailsjs/go/main/App';
import type {wallhaven} from '../wailsjs/go/models';
import {button, deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    GetWallhavenConfig: vi.fn().mockResolvedValue({sorting: 'favorites'}),
    SetWallhavenAPIKey: vi.fn().mockResolvedValue(undefined),
    SearchWallhaven: vi.fn(),
}));
vi.mock(
    '../src/lib/components/wallhaven/WallhavenFilters.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/wallhaven/WallpaperGrid.svelte',
    () => import('./Empty.svelte')
);
vi.mock('../src/lib/utils/intersection', () => ({
    observeIntersection: vi.fn(() => () => {}),
}));

function result(ids: string[] = [], lastPage = 0): wallhaven.SearchResult {
    return {
        data: ids.map(id => ({id})),
        meta: {last_page: lastPage, total: ids.length},
    } as wallhaven.SearchResult;
}

beforeEach(() => {
    vi.mocked(SearchWallhaven).mockReset();
    vi.spyOn(console, 'error').mockImplementation(() => {});
});

test('an empty initial search stays empty across rerenders and remounts', async () => {
    vi.mocked(SearchWallhaven).mockResolvedValue(result());
    const {default: Browser} = await import(
        '../src/lib/components/wallhaven/WallhavenBrowser.svelte'
    );
    const store = await import('../src/lib/stores/wallhaven.svelte');
    const first = render(Browser, {});
    await settle();
    flushSync();
    await settle();
    expect(SearchWallhaven).toHaveBeenCalledTimes(1);
    expect(SearchWallhaven).toHaveBeenCalledWith(
        expect.objectContaining({sorting: 'favorites'})
    );
    expect(first.target.textContent).toContain('No wallpapers to show');
    await first.destroy();
    render(Browser, {});
    await settle();
    expect(SearchWallhaven).toHaveBeenCalledTimes(1);
    await store.search();
    expect(SearchWallhaven).toHaveBeenCalledTimes(2);
});

test('search failures remain stable until the user explicitly retries', async () => {
    vi.mocked(SearchWallhaven)
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValueOnce(result());
    const {default: Browser} = await import(
        '../src/lib/components/wallhaven/WallhavenBrowser.svelte'
    );
    const store = await import('../src/lib/stores/wallhaven.svelte');
    await store.search();
    const {target} = render(Browser, {});
    await settle();
    flushSync();
    await settle();
    expect(SearchWallhaven).toHaveBeenCalledTimes(1);
    expect(target.textContent).toContain('Search failed');
    button(target, 'Retry').click();
    await settle();
    expect(SearchWallhaven).toHaveBeenCalledTimes(2);
    expect(target.textContent).toContain('No wallpapers to show');
});

test('new searches supersede older responses without clearing their loading state', async () => {
    const old = deferred<wallhaven.SearchResult>();
    const current = deferred<wallhaven.SearchResult>();
    vi.mocked(SearchWallhaven)
        .mockReturnValueOnce(old.promise)
        .mockReturnValueOnce(current.promise);
    const store = await import('../src/lib/stores/wallhaven.svelte');
    store.setQuery('old');
    const first = store.search();
    await settle();
    store.setQuery('new');
    const second = store.search();
    await settle();
    old.resolve(result(['old']));
    await first;
    expect(store.getIsSearching()).toBe(true);
    expect(store.getResults()).toEqual([]);
    current.resolve(result(['new']));
    await second;
    expect(store.getIsSearching()).toBe(false);
    expect(store.getResults().map(item => item.id)).toEqual(['new']);
});

test('failed pagination retries the same page and query instead of mixing in unsubmitted filters', async () => {
    vi.mocked(SearchWallhaven)
        .mockResolvedValueOnce(result(['first'], 3))
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValueOnce(result(['first', 'second'], 3));
    const store = await import('../src/lib/stores/wallhaven.svelte');
    store.setQuery('original');
    await store.search();
    store.setQuery('not submitted');
    await store.loadMore();
    expect(store.getSearchError()).not.toBe('');
    expect(store.getResults().map(item => item.id)).toEqual(['first']);
    await store.loadMore();
    expect(SearchWallhaven).toHaveBeenNthCalledWith(
        2,
        expect.objectContaining({q: 'original', page: 2})
    );
    expect(SearchWallhaven).toHaveBeenNthCalledWith(
        3,
        expect.objectContaining({q: 'original', page: 2})
    );
    expect(store.getResults().map(item => item.id)).toEqual([
        'first',
        'second',
    ]);
    expect(store.getSearchError()).toBe('');
});

test('a new search discards an older pagination response', async () => {
    const page = deferred<wallhaven.SearchResult>();
    vi.mocked(SearchWallhaven)
        .mockResolvedValueOnce(result(['first'], 3))
        .mockReturnValueOnce(page.promise)
        .mockResolvedValueOnce(result(['new']));
    const store = await import('../src/lib/stores/wallhaven.svelte');
    await store.search();
    const append = store.loadMore();
    await settle();
    store.setQuery('new');
    await store.search();
    page.resolve(result(['stale'], 3));
    await append;
    expect(store.getResults().map(item => item.id)).toEqual(['new']);
    expect(store.getIsLoadingMore()).toBe(false);
});
