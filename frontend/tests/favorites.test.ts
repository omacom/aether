import {beforeEach, expect, test, vi} from 'vitest';
import WallpaperCard from '../src/lib/components/wallhaven/WallpaperCard.svelte';
import FavoritesView from '../src/lib/components/favorites/FavoritesView.svelte';
import LocalBrowser from '../src/lib/components/local/LocalBrowser.svelte';
import * as favorites from '../src/lib/stores/favorites.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {
    GetFavorites,
    ToggleFavorite,
    ScanLocalWallpapers,
} from '../wailsjs/go/main/App';
import {deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    GetFavorites: vi.fn().mockResolvedValue([]),
    ToggleFavorite: vi.fn(),
    ScanLocalWallpapers: vi.fn().mockResolvedValue([]),
    GetWallpaperTags: vi.fn().mockResolvedValue({labels: [], assignments: {}}),
    GetThumbnail: vi.fn().mockResolvedValue('/thumbnail.png'),
    ReadImageAsDataURL: vi.fn().mockResolvedValue('data:image/png;base64,AA=='),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    getOmarchyAvailable: () => true,
    getOmarchyCapabilities: () => ({overrideApps: []}),
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
}));

const remote = 'https://example.com/wallpaper.png';
const wallpaper = {path: remote, id: 'sample', thumbs: {small: remote}};

beforeEach(async () => {
    vi.stubGlobal(
        'IntersectionObserver',
        class {
            observe() {}
            disconnect() {}
            unobserve() {}
        }
    );
    vi.stubGlobal(
        'ResizeObserver',
        class {
            observe() {}
            disconnect() {}
            unobserve() {}
        }
    );
    await settle();
    vi.mocked(GetFavorites).mockReset().mockResolvedValue([]);
    vi.mocked(ToggleFavorite).mockReset();
    vi.mocked(ScanLocalWallpapers).mockReset().mockResolvedValue([]);
    await favorites.refreshFavorites();
    theme.reset();
});

test('a favorite toggle updates both cards and the Favorites view', async () => {
    const one = render(WallpaperCard, {wallpaper, onpreview: vi.fn()});
    const two = render(WallpaperCard, {wallpaper, onpreview: vi.fn()});
    vi.mocked(ToggleFavorite).mockResolvedValue(true);
    one.target
        .querySelector<HTMLButtonElement>(
            'button[aria-label="Add to favorites"]'
        )!
        .click();
    await settle();
    expect(ToggleFavorite).toHaveBeenCalledTimes(1);
    expect(
        two.target.querySelector('button[aria-label="Remove from favorites"]')
    ).not.toBeNull();
    vi.mocked(GetFavorites).mockResolvedValue([
        {path: remote, type: 'wallhaven', data: {id: 'sample'}},
    ]);
    const view = render(FavoritesView, {});
    await settle();
    expect(
        view.target.querySelectorAll(
            'button[aria-label="Remove from favorites"]'
        )
    ).toHaveLength(1);
});

test('the local heart saves a favorite without selecting the wallpaper', async () => {
    vi.mocked(ScanLocalWallpapers).mockResolvedValue([
        {path: '/local.png', name: 'Local', size: 1, modTime: 1},
    ]);
    vi.mocked(ToggleFavorite).mockResolvedValue(true);
    const view = render(LocalBrowser, {});
    await settle();
    view.target
        .querySelector<HTMLButtonElement>(
            'button[aria-label="Add to favorites"]'
        )!
        .click();
    await settle();
    expect(favorites.isFavorite('/local.png')).toBe(true);
    expect(theme.getWallpaperPath()).toBe('');
    expect(ToggleFavorite).toHaveBeenCalledWith('/local.png', 'local', {
        name: 'Local',
    });
});

test('an old refresh preserves a newer toggle and loads unrelated favorites', async () => {
    const old = deferred<Awaited<ReturnType<typeof GetFavorites>>>();
    vi.mocked(GetFavorites).mockReturnValueOnce(old.promise);
    const refresh = favorites.refreshFavorites();
    await settle();
    vi.mocked(ToggleFavorite).mockResolvedValue(true);
    await favorites.toggleFavorite(remote, 'wallhaven');
    old.resolve([{path: '/existing.png', type: 'local'}]);
    await refresh;
    expect(favorites.isFavorite(remote)).toBe(true);
    expect(favorites.isFavorite('/existing.png')).toBe(true);
});

test('a refresh cannot restore a removed favorite', async () => {
    const entry = {path: remote, type: 'wallhaven'};
    vi.mocked(GetFavorites).mockResolvedValueOnce([entry]);
    await favorites.refreshFavorites();
    const old = deferred<Awaited<ReturnType<typeof GetFavorites>>>();
    vi.mocked(GetFavorites).mockReturnValueOnce(old.promise);
    const refresh = favorites.refreshFavorites();
    await settle();
    vi.mocked(ToggleFavorite).mockResolvedValue(false);
    await favorites.toggleFavorite(remote, 'wallhaven');
    old.resolve([entry]);
    await refresh;
    expect(favorites.isFavorite(remote)).toBe(false);
});

test('the newest refresh wins', async () => {
    const old = deferred<Awaited<ReturnType<typeof GetFavorites>>>();
    vi.mocked(GetFavorites)
        .mockReturnValueOnce(old.promise)
        .mockResolvedValueOnce([{path: '/new.png'}]);
    const first = favorites.refreshFavorites();
    await settle();
    await favorites.refreshFavorites();
    old.resolve([]);
    await first;
    expect(favorites.isFavorite('/new.png')).toBe(true);
});

test('same-path toggles reach the backend in click order', async () => {
    const added = deferred<boolean>();
    const removed = deferred<boolean>();
    vi.mocked(ToggleFavorite)
        .mockReturnValueOnce(added.promise)
        .mockReturnValueOnce(removed.promise);
    const first = favorites.toggleFavorite(remote, 'wallhaven');
    const second = favorites.toggleFavorite(remote, 'wallhaven');
    await settle();
    expect(ToggleFavorite).toHaveBeenCalledTimes(1);
    added.resolve(true);
    await first;
    await settle();
    expect(ToggleFavorite).toHaveBeenCalledTimes(2);
    removed.resolve(false);
    await second;
    expect(favorites.isFavorite(remote)).toBe(false);
});

test('a failed toggle does not block the next toggle', async () => {
    vi.mocked(ToggleFavorite)
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValueOnce(true);
    const first = favorites.toggleFavorite(remote, 'wallhaven');
    const second = favorites.toggleFavorite(remote, 'wallhaven');
    await expect(first).rejects.toThrow('offline');
    await expect(second).resolves.toBe(true);
    expect(favorites.isFavorite(remote)).toBe(true);
});

test('a failed refresh preserves favorites and offers a retry', async () => {
    vi.mocked(GetFavorites).mockResolvedValueOnce([{path: '/saved.png'}]);
    await favorites.refreshFavorites();
    vi.mocked(GetFavorites).mockRejectedValueOnce(new Error('offline'));
    const view = render(FavoritesView, {});
    await settle();
    expect(favorites.isFavorite('/saved.png')).toBe(true);
    expect(view.target.querySelector('[role="alert"]')?.textContent).toContain(
        'Could not load favorites'
    );
    vi.mocked(GetFavorites).mockResolvedValue([{path: '/saved.png'}]);
    view.target
        .querySelector<HTMLButtonElement>('[role="alert"] button')!
        .click();
    await settle();
    expect(view.target.querySelector('[role="alert"]')).toBeNull();
});
