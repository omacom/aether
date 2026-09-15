import {beforeEach, expect, test, vi} from 'vitest';
import WallpaperHero from '../src/lib/components/editor/WallpaperHero.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {BlurWallpaper} from '../wailsjs/go/main/App';
import {undoAction, redoAction} from '../src/lib/actions/themeActions';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import {DEFAULT_PALETTE} from '../src/lib/types/theme';
import {deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({BlurWallpaper: vi.fn()}));
vi.mock('../src/lib/stores/imagecache.svelte', () => ({
    getCachedFullImage: (path: string) => path,
    loadFullImage: vi.fn(),
    isPending: () => false,
}));

beforeEach(() => {
    theme.reset();
    theme.setWallpaperPath('/source-a.png');
    vi.mocked(BlurWallpaper).mockReset().mockResolvedValue('/blur-a.jpg');
});

test('the hero rejects an older blur result after another wallpaper is selected', async () => {
    const old = deferred<string>();
    vi.mocked(BlurWallpaper)
        .mockReturnValueOnce(old.promise)
        .mockResolvedValueOnce('/blur-b.jpg');
    const {target} = render(WallpaperHero, {});
    theme.setWallpaperBlur(true);
    await settle();
    theme.setWallpaperPath('/source-b.png');
    theme.setWallpaperBlur(true);
    await settle();
    expect(theme.getBlurredWallpaperPath()).toBe('/blur-b.jpg');
    old.resolve('/blur-a.jpg');
    await settle();
    expect(theme.getWallpaperPath()).toBe('/source-b.png');
    expect(theme.getBlurredWallpaperPath()).toBe('/blur-b.jpg');
    expect(target.querySelector('img')?.getAttribute('src')).toBe(
        '/blur-b.jpg'
    );
});

test('blur can be disabled before its preview completes', async () => {
    const pending = deferred<string>();
    vi.mocked(BlurWallpaper).mockReturnValue(pending.promise);
    const {target} = render(WallpaperHero, {});
    target
        .querySelector<HTMLButtonElement>(
            '[aria-label="Heavy blur wallpaper"]'
        )!
        .click();
    await settle();
    target
        .querySelector<HTMLButtonElement>('[aria-label="Remove blur"]')!
        .click();
    pending.resolve('/blur-a.jpg');
    await settle();
    expect(theme.getWallpaperBlur()).toBe(false);
    expect(theme.getBlurredWallpaperPath()).toBe('');
    expect(target.querySelector('img')?.getAttribute('src')).toBe(
        '/source-a.png'
    );
});

test('a repeated source path still rejects results from an older selection', () => {
    theme.setWallpaperBlur(true);
    const old = theme.getWallpaperRevision();
    theme.setWallpaperPath('/source-b.png');
    theme.setWallpaperPath('/source-a.png');
    theme.setWallpaperBlur(true);
    theme.setBlurredWallpaper('/source-a.png', '/obsolete.jpg', old);
    expect(theme.getBlurredWallpaperPath()).toBe('');
});

test('history and blueprints preserve source-based blur intent', () => {
    theme.setWallpaperBlur(true);
    expect(theme.getThemeSnapshot()).toEqual(
        expect.objectContaining({
            wallpaperPath: '/source-a.png',
            wallpaperBlur: true,
        })
    );
    const signature = theme.getThemeSignature();
    theme.setBlurredWallpaper(
        '/source-a.png',
        '/cached.jpg',
        theme.getWallpaperRevision()
    );
    expect(theme.getThemeSignature()).toBe(signature);
    undoAction();
    expect(theme.getWallpaperBlur()).toBe(false);
    redoAction();
    expect(theme.getWallpaperBlur()).toBe(true);
    const blueprint = {
        name: 'Saved',
        timestamp: 0,
        palette: {
            colors: [...DEFAULT_PALETTE],
            wallpaper: '/original.png',
            wallpaperBlur: true,
        },
    };
    loadBlueprintIntoEditor(blueprint);
    expect(theme.getWallpaperPath()).toBe('/original.png');
    expect(theme.getWallpaperBlur()).toBe(true);
    loadBlueprintIntoEditor({
        ...blueprint,
        palette: {...blueprint.palette, wallpaperBlur: false},
    });
    expect(theme.getWallpaperBlur()).toBe(false);
});

test('swapping the main image clears the previous blur choice', () => {
    theme.setWallpaperBlur(true);
    theme.addAdditionalImage('/source-b.png');
    theme.swapMainWithAdditional('/source-b.png');
    expect(theme.getWallpaperPath()).toBe('/source-b.png');
    expect(theme.getWallpaperBlur()).toBe(false);
});
