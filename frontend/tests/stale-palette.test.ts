import {beforeEach, expect, test, vi} from 'vitest';
import OmarchyThemes from '../src/lib/components/blueprints/OmarchyThemes.svelte';
import WallpaperHero from '../src/lib/components/editor/WallpaperHero.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import * as ui from '../src/lib/stores/ui.svelte';
import {applyTheme, extractColors} from '../src/lib/actions/themeActions';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import {importThemeFile} from '../src/lib/actions/importActions';
import {DEFAULT_PALETTE} from '../src/lib/types/theme';
import {
    ApplyTheme,
    ExtractColors,
    ExtractColorsFromImages,
    ImportFileDialog,
    LoadOmarchyThemes,
} from '../wailsjs/go/main/App';
import {render, settle} from './setup';

const NOTICE =
    'This palette is not from this wallpaper. Click Extract to update it.';
const colorsA = Array(16).fill('#111111');
const colorsB = Array(16).fill('#222222');

vi.mock('../wailsjs/go/main/App', () => ({
    ApplyTheme: vi.fn(),
    ExtractColors: vi.fn(),
    ExtractColorsFromImages: vi.fn(),
    SetExtractionMode: vi.fn(),
    SaveAndApplyTheme: vi.fn(),
    ImportFileDialog: vi.fn(),
    LoadOmarchyThemes: vi.fn(),
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
    getOmarchyAvailable: () => false,
    getOmarchyCapabilities: () => ({overrideApps: ['kitty']}),
    refreshOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('../src/lib/stores/imagecache.svelte', () => ({
    getCachedFullImage: (path: string) => path,
    getCachedThumbnail: () => '',
    loadThumbnail: vi.fn(),
    loadFullImage: vi.fn(),
    isPending: () => false,
}));

beforeEach(async () => {
    await settle();
    theme.reset();
    theme.setIsExtracting(false);
    theme.setIsApplying(false);
    ui.setEyedropperActive(false);
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
    vi.mocked(ApplyTheme)
        .mockReset()
        .mockResolvedValue({success: true, isOmarchy: false, themePath: '/t'});
    vi.mocked(ExtractColors).mockReset().mockResolvedValue(colorsB);
    vi.mocked(ExtractColorsFromImages)
        .mockReset()
        .mockResolvedValue({palette: colorsB, skipped: 0});
    vi.mocked(ImportFileDialog).mockReset();
    vi.mocked(LoadOmarchyThemes).mockReset().mockResolvedValue([]);
    vi.stubGlobal(
        'IntersectionObserver',
        class {
            constructor(
                private callback: (entries: {isIntersecting: boolean}[]) => void
            ) {}
            observe() {
                this.callback([{isIntersecting: true}]);
            }
            disconnect() {}
        }
    );
});

async function extractedOn(path: string) {
    theme.setWallpaperPath(path);
    await extractColors();
    expect(theme.isPaletteStale()).toBe(false);
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
}

test('a new wallpaper makes the palette stale and keeps colors and overrides (AC-01.2)', async () => {
    await extractedOn('/a.png');
    theme.setAppOverride('kitty', 'background', '#123456');
    vi.mocked(ExtractColors).mockClear();
    const palette = [...theme.getPalette()];
    const extended = {...theme.getExtendedColors()};
    theme.setWallpaperPath('/b.png');
    expect(theme.isPaletteStale()).toBe(true);
    expect(theme.getPalette()).toEqual(palette);
    expect(theme.getExtendedColors()).toEqual(extended);
    expect(theme.getAppOverrides()).toEqual({kitty: {background: '#123456'}});
    expect(ExtractColors).not.toHaveBeenCalled();
});

test('no wallpaper is never stale', () => {
    expect(theme.isPaletteStale()).toBe(false);
});

test('Extract and Extract all clear the stale state (AC-01.4)', async () => {
    await extractedOn('/a.png');
    theme.setWallpaperPath('/b.png');
    expect(theme.isPaletteStale()).toBe(true);
    await extractColors();
    expect(theme.isPaletteStale()).toBe(false);
    theme.setWallpaperPath('/c.png');
    await extractColors({allImages: true});
    expect(theme.isPaletteStale()).toBe(false);
});

test('a failed extraction keeps the palette stale (AC-01.5)', async () => {
    await extractedOn('/a.png');
    theme.setWallpaperPath('/b.png');
    vi.mocked(ExtractColors).mockRejectedValueOnce('decode failed');
    await extractColors();
    expect(theme.isPaletteStale()).toBe(true);
});

const blueprint = {
    name: 'Saved',
    timestamp: 0,
    palette: {colors: [...DEFAULT_PALETTE], wallpaper: '/bp.png'},
};

test('a blueprint load clears the stale state (AC-01.6)', async () => {
    await extractedOn('/a.png');
    theme.setWallpaperPath('/b.png');
    loadBlueprintIntoEditor(blueprint);
    expect(theme.getWallpaperPath()).toBe('/bp.png');
    expect(theme.isPaletteStale()).toBe(false);
});

test.each(['base16', 'toml', 'blueprint'] as const)(
    'a %s import clears the stale state (AC-01.6)',
    async type => {
        await extractedOn('/a.png');
        theme.setWallpaperPath('/b.png');
        vi.mocked(ImportFileDialog).mockResolvedValue({
            name: 'Imported',
            colors: [...DEFAULT_PALETTE],
            wallpaperPath: '/imported.png',
        } as never);
        expect(await importThemeFile(type)).toBe(true);
        expect(theme.getWallpaperPath()).toBe('/imported.png');
        expect(theme.isPaletteStale()).toBe(false);
    }
);

test('an import with no wallpaper keeps the current wallpaper fresh (AC-01.6)', async () => {
    await extractedOn('/a.png');
    vi.mocked(ImportFileDialog).mockResolvedValue({
        name: 'Imported',
        colors: [...DEFAULT_PALETTE],
    } as never);
    await importThemeFile('toml');
    expect(theme.isPaletteStale()).toBe(false);
});

test('an Omarchy theme Import clears the stale state (AC-01.6)', async () => {
    await extractedOn('/a.png');
    theme.setWallpaperPath('/b.png');
    vi.mocked(LoadOmarchyThemes).mockResolvedValue([
        {
            name: 'Native',
            colors: [...DEFAULT_PALETTE],
            wallpapers: ['/native.png'],
        },
    ] as never);
    const {target} = render(OmarchyThemes, {});
    await settle();
    [...target.querySelectorAll('button')]
        .find(b => b.textContent?.trim() === 'Import')!
        .click();
    expect(theme.getWallpaperPath()).toBe('/native.png');
    expect(theme.isPaletteStale()).toBe(false);
});

test('the hero shows the notice only while stale and not extracting or picking (AC-01.7)', async () => {
    await extractedOn('/a.png');
    const {target} = render(WallpaperHero, {});
    await settle();
    expect(target.textContent).not.toContain(NOTICE);
    theme.setWallpaperPath('/b.png');
    await settle();
    const notice = target.querySelector('[role="status"]');
    expect(notice?.textContent?.replace(/\s+/g, ' ').trim()).toBe(NOTICE);
    expect(notice?.className).toContain('text-white');
    ui.setEyedropperActive(true);
    await settle();
    expect(target.textContent).not.toContain('This palette is not from');
    expect(target.textContent).toContain('Click to pick a color');
    ui.setEyedropperActive(false);
    await settle();
    expect(target.textContent).toContain('This palette is not from');
    theme.setWallpaperPath('/b.png');
    await extractColors();
    await settle();
    expect(target.textContent).not.toContain('This palette is not from');
});

test('a stale palette still applies (AC-01.8)', async () => {
    await extractedOn('/a.png');
    theme.setWallpaperPath('/b.png');
    const palette = [...theme.getPalette()];
    await applyTheme();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    expect(ApplyTheme).toHaveBeenCalledWith(
        expect.objectContaining({palette, wallpaperPath: '/b.png'})
    );
});
