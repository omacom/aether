import {beforeEach, expect, test, vi} from 'vitest';
import SettingsSidebar from '../src/lib/components/sidebar/SettingsSidebar.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import * as ui from '../src/lib/stores/ui.svelte';
import {
    applyTheme,
    extractColors,
    undoAction,
} from '../src/lib/actions/themeActions';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import {importThemeFile} from '../src/lib/actions/importActions';
import {buildCommands} from '../src/lib/commands/commands.svelte';
import {DEFAULT_PALETTE} from '../src/lib/types/theme';
import {
    ApplyTheme,
    ExtractColors,
    ImportFileDialog,
} from '../wailsjs/go/main/App';
import {button, deferred, render, settle} from './setup';

const HINT = 'Applies on the next extract.';
const dark = Array(16).fill('#111111');
const light = Array(16).fill('#eeeeee');

vi.mock(
    '../src/lib/components/sidebar/ExtractionModeSelect.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/ColorAdjustments.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/PresetsSection.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/GradientGenerator.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/PaletteFromColor.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/AccessibilityPanel.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/NeovimThemes.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/TemplateToggles.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/sidebar/IconThemePicker.svelte',
    () => import('./Empty.svelte')
);
vi.mock('../wailsjs/go/main/App', () => ({
    ApplyTheme: vi.fn(),
    SaveAndApplyTheme: vi.fn(),
    ExtractColors: vi.fn(),
    ExtractColorsFromImages: vi.fn(),
    SetExtractionMode: vi.fn(),
    ImportFileDialog: vi.fn(),
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
    getOmarchyAvailable: () => false,
    getOmarchyCapabilities: () => ({overrideApps: ['kitty']}),
}));

beforeEach(async () => {
    await settle();
    theme.reset();
    theme.setIsExtracting(false);
    theme.setIsApplying(false);
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
    vi.mocked(ExtractColors).mockReset().mockResolvedValue(light);
    vi.mocked(ApplyTheme)
        .mockReset()
        .mockResolvedValue({success: true, isOmarchy: false, themePath: '/t'});
    vi.mocked(ImportFileDialog).mockReset();
});

// Wallpaper A with a fresh palette from Extract, in dark mode, with 1 override.
async function freshDark() {
    vi.mocked(ExtractColors).mockResolvedValueOnce(dark);
    theme.setWallpaperPath('/a.png');
    await extractColors();
    theme.setAppOverride('kitty', 'background', '#123456');
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
    vi.mocked(ExtractColors).mockClear();
}

function pick(target: HTMLElement, label: 'Dark' | 'Light') {
    button(target, label).click();
}

function command() {
    return buildCommands().find(item => item.id === 'theme.toggleLight')!;
}

function shown(target: HTMLElement) {
    return [...target.querySelectorAll('button[aria-pressed="true"]')].map(el =>
        el.textContent?.trim()
    );
}

test('selecting Light extracts once with the current mode (AC-03.1)', async () => {
    await freshDark();
    const {target} = render(SettingsSidebar, {});
    pick(target, 'Light');
    await settle();
    expect(ExtractColors).toHaveBeenCalledExactlyOnceWith(
        '/a.png',
        true,
        'normal'
    );
    expect(theme.getPalette()).toEqual(light);
    expect(theme.getLightMode()).toBe(true);
    expect(ui.getToastMessage()).toBe('Palette extracted for light mode.');
});

test('the toggle command extracts once (AC-03.2)', async () => {
    await freshDark();
    await command().run();
    await settle();
    expect(ExtractColors).toHaveBeenCalledExactlyOnceWith(
        '/a.png',
        true,
        'normal'
    );
    expect(theme.getLightMode()).toBe(true);
});

test('Ctrl+Z restores palette, override and mode in one step (AC-03.3)', async () => {
    await freshDark();
    const before = theme.getHistorySnapshot();
    const {target} = render(SettingsSidebar, {});
    pick(target, 'Light');
    await settle();
    vi.mocked(ExtractColors).mockClear();
    undoAction();
    await settle();
    expect(theme.getHistorySnapshot()).toEqual(before);
    expect(theme.getLightMode()).toBe(false);
    expect(theme.getAppOverrides()).toEqual({kitty: {background: '#123456'}});
    expect(ExtractColors).not.toHaveBeenCalled();
});

test('a stale palette only sets the mode and shows the hint (AC-03.4, AC-03.6)', async () => {
    await freshDark();
    theme.setWallpaperPath('/b.png');
    const palette = [...theme.getPalette()];
    const {target} = render(SettingsSidebar, {});
    expect(target.textContent).toContain(HINT);
    pick(target, 'Light');
    await settle();
    expect(ExtractColors).not.toHaveBeenCalled();
    expect(theme.getLightMode()).toBe(true);
    expect(theme.getPalette()).toEqual(palette);
    expect(theme.getAppOverrides()).toEqual({kitty: {background: '#123456'}});
});

test('with no wallpaper the mode changes with no extraction and undo (AC-03.5, AC-03.6)', async () => {
    const {target} = render(SettingsSidebar, {});
    expect(target.textContent).toContain(HINT);
    pick(target, 'Light');
    await settle();
    expect(ExtractColors).not.toHaveBeenCalled();
    expect(theme.getLightMode()).toBe(true);
    undoAction();
    expect(theme.getLightMode()).toBe(false);
});

test('the hint hides for a wallpaper with a fresh palette (AC-03.6)', async () => {
    await freshDark();
    const {target} = render(SettingsSidebar, {});
    expect(target.textContent).not.toContain(HINT);
});

test('the toggle command explains itself when it does not extract', async () => {
    await command().run();
    expect(ui.getToastMessage()).toBe(
        'Palette mode set to light. Applies on the next extract.'
    );
});

test('loads, imports and light mode pushes never extract (AC-03.7)', async () => {
    await freshDark();
    loadBlueprintIntoEditor({
        name: 'Saved',
        timestamp: 0,
        palette: {
            colors: [...DEFAULT_PALETTE],
            wallpaper: '/bp.png',
            lightMode: true,
        },
    });
    vi.mocked(ImportFileDialog).mockResolvedValue({
        name: 'Imported',
        colors: [...DEFAULT_PALETTE],
        lightMode: false,
    } as never);
    await importThemeFile('toml');
    theme.setLightMode(true);
    await settle();
    expect(ExtractColors).not.toHaveBeenCalled();
});

test.each(['switch', 'command'] as const)(
    'a failed %s extraction reverts the mode and shows the cause (AC-03.10)',
    async via => {
        await freshDark();
        const before = theme.getHistorySnapshot();
        vi.mocked(ExtractColors).mockRejectedValueOnce(
            'decode image: unknown format'
        );
        const {target} = render(SettingsSidebar, {});
        if (via === 'switch') pick(target, 'Light');
        else await command().run();
        await settle();
        expect(shown(target)).toEqual(['Dark']);
        expect(theme.getLightMode()).toBe(false);
        expect(theme.getHistorySnapshot()).toEqual(before);
        expect(ui.getToastMessage()).toContain('Couldn’t extract colors');
        expect(ui.getToastMessage()).toContain('decode image: unknown format');
        await applyTheme();
        expect(ApplyTheme).toHaveBeenCalledWith(
            expect.objectContaining({lightMode: false})
        );
    }
);

test('switching back before the first result returns drops the older result', async () => {
    await freshDark();
    const first = deferred<string[]>();
    vi.mocked(ExtractColors).mockReturnValueOnce(first.promise);
    const {target} = render(SettingsSidebar, {});
    pick(target, 'Light');
    await settle();
    expect(shown(target)).toEqual(['Light']);
    pick(target, 'Dark');
    await settle();
    expect(shown(target)).toEqual(['Dark']);
    first.resolve(light);
    await settle();
    expect(theme.getLightMode()).toBe(false);
    expect(theme.getPalette()).toEqual(dark);
    expect(theme.getIsExtracting()).toBe(false);
});
