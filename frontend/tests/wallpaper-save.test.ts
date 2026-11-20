import {beforeEach, expect, test, vi} from 'vitest';
import {createRawSnippet} from 'svelte';
import WallpaperTile from '../src/lib/components/shared/WallpaperTile.svelte';
import {
    saveWallpaperToFolder,
    addWallpaperToCurrentTheme,
} from '../src/lib/actions/themeActions';
import {
    getToastMessage,
    dismissCurrentToast,
} from '../src/lib/stores/ui.svelte';
import {
    SaveWallpaperToFolder,
    AddWallpaperToCurrentTheme,
} from '../wailsjs/go/main/App';
import {button, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    SaveWallpaperToFolder: vi.fn(),
    AddWallpaperToCurrentTheme: vi.fn(),
}));

const emptyThumb = createRawSnippet(() => ({
    render: () => '<span></span>',
}));

const baseProps = {
    path: 'https://w.wallhaven.cc/full/ab/wallhaven-abcdef.jpg',
    name: 'abcdef',
    onuse: vi.fn(),
    onwallpaperonly: vi.fn(),
    onpreview: vi.fn(),
    onaddextra: vi.fn(),
    thumb: emptyThumb,
};

beforeEach(() => {
    vi.mocked(SaveWallpaperToFolder).mockReset();
    vi.mocked(AddWallpaperToCurrentTheme).mockReset();
    while (getToastMessage()) dismissCurrentToast();
});

test('tile save button opens a two-option menu when both folder and theme actions exist', async () => {
    const onsavefolder = vi.fn().mockResolvedValue(true);
    const onaddtotheme = vi.fn().mockResolvedValue(true);
    const {target} = render(WallpaperTile, {
        ...baseProps,
        onsavefolder,
        onaddtotheme,
    });

    button(target, 'Save wallpaper').click();
    await settle();

    const items = [...target.querySelectorAll('[role="menuitem"]')].map(el =>
        el.textContent?.trim()
    );
    expect(items).toEqual([
        'Save to wallpaper folder',
        'Add to current theme backgrounds',
    ]);

    button(target, 'Add to current theme backgrounds').click();
    await settle();
    expect(onaddtotheme).toHaveBeenCalledTimes(1);
    expect(onsavefolder).not.toHaveBeenCalled();

    button(target, 'Save wallpaper').click();
    await settle();
    button(target, 'Save to wallpaper folder').click();
    await settle();
    expect(onsavefolder).toHaveBeenCalledTimes(1);
});

test('tile save button runs directly without a menu when only one target is available', async () => {
    const onsavefolder = vi.fn().mockResolvedValue(true);
    const folderOnly = render(WallpaperTile, {
        ...baseProps,
        onsavefolder,
    });

    button(folderOnly.target, 'Save to wallpaper folder').click();
    await settle();
    expect(onsavefolder).toHaveBeenCalledTimes(1);
    expect(folderOnly.target.querySelector('[role="menu"]')).toBeNull();

    const onaddtotheme = vi.fn().mockResolvedValue(true);
    const themeOnly = render(WallpaperTile, {
        ...baseProps,
        path: '/home/user/Wallpapers/wall.png',
        onaddtotheme,
    });

    button(themeOnly.target, 'Add to current theme backgrounds').click();
    await settle();
    expect(onaddtotheme).toHaveBeenCalledTimes(1);
    expect(themeOnly.target.querySelector('[role="menu"]')).toBeNull();

    const neither = render(WallpaperTile, {...baseProps});
    expect(
        neither.target.querySelector('button[aria-label^="Save"]')
    ).toBeNull();
    expect(
        neither.target.querySelector('button[aria-label$="backgrounds"]')
    ).toBeNull();
});

test('saveWallpaperToFolder and addWallpaperToCurrentTheme invoke backend and toast results', async () => {
    vi.mocked(SaveWallpaperToFolder).mockResolvedValueOnce(
        '/home/user/Wallpapers/wall.png'
    );
    expect(
        await saveWallpaperToFolder(
            'https://example.com/wall.png',
            'custom.png'
        )
    ).toBe(true);
    expect(SaveWallpaperToFolder).toHaveBeenCalledWith(
        'https://example.com/wall.png',
        'custom.png'
    );
    expect(getToastMessage()).toBe('Downloading wallpaper…');
    dismissCurrentToast();
    expect(getToastMessage()).toBe('Saved to wallpaper folder');
    dismissCurrentToast();

    vi.mocked(AddWallpaperToCurrentTheme).mockResolvedValueOnce('ethereal');
    expect(
        await addWallpaperToCurrentTheme('/home/user/Wallpapers/wall.png')
    ).toBe(true);
    expect(AddWallpaperToCurrentTheme).toHaveBeenCalledWith(
        '/home/user/Wallpapers/wall.png',
        ''
    );
    expect(getToastMessage()).toBe('Added to ethereal backgrounds');
    dismissCurrentToast();

    vi.mocked(AddWallpaperToCurrentTheme).mockRejectedValueOnce(
        new Error('no active Omarchy theme')
    );
    expect(
        await addWallpaperToCurrentTheme('/home/user/Wallpapers/wall.png')
    ).toBe(false);
    expect(getToastMessage()).toBe('no active Omarchy theme');
});
