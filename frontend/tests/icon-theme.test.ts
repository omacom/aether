import {beforeEach, expect, test, vi} from 'vitest';
import IconThemePicker from '../src/lib/components/sidebar/IconThemePicker.svelte';
import OmarchyThemes from '../src/lib/components/blueprints/OmarchyThemes.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {isAppIncluded, setAppIncluded} from '../src/lib/stores/settings.svelte';
import {
    normalizeIconThemeSelection,
    DEFAULT_PALETTE,
} from '../src/lib/types/theme';
import {undoAction, redoAction} from '../src/lib/actions/themeActions';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import {
    GetIconThemePreview,
    ListInstalledIconThemes,
    RefreshInstalledIconThemes,
    LoadOmarchyThemes,
} from '../wailsjs/go/main/App';
import {render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
    ListInstalledIconThemes: vi.fn(),
    RefreshInstalledIconThemes: vi.fn(),
    GetIconThemePreview: vi.fn().mockResolvedValue({samples: []}),
    LoadOmarchyThemes: vi.fn().mockResolvedValue([]),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    getOmarchyAvailable: () => true,
    getOmarchyCapabilities: () => ({overrideApps: ['kitty']}),
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
    refreshOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
}));

const ocean = {id: 'Ocean', name: 'Ocean', origin: 'user', hasPreview: true};

beforeEach(async () => {
    await settle();
    theme.reset();
    theme.markApplied();
    setAppIncluded('icons', true);
    vi.mocked(ListInstalledIconThemes).mockReset().mockResolvedValue([ocean]);
    vi.mocked(RefreshInstalledIconThemes)
        .mockReset()
        .mockResolvedValue([ocean]);
    vi.mocked(GetIconThemePreview)
        .mockReset()
        .mockResolvedValue({themeId: 'Ocean', samples: []});
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

test('legacy selection defaults to automatic and missing explicit IDs survive', () => {
    expect(normalizeIconThemeSelection(undefined)).toEqual({mode: 'automatic'});
    expect(normalizeIconThemeSelection({mode: 'automatic'})).toEqual({
        mode: 'automatic',
    });
    expect(
        normalizeIconThemeSelection({mode: 'explicit', id: 'Missing-Icons'})
    ).toEqual({mode: 'explicit', id: 'Missing-Icons'});
});

test('icon undo and redo preserve adjusted colors and their original baselines', () => {
    theme.setPalette(Array(16).fill('#123456'), true);
    theme.setAdjustedPalette(Array(16).fill('#456789'));
    theme.markApplied();
    const before = theme.getHistorySnapshot();
    theme.setIconTheme({mode: 'explicit', id: 'Ocean'});
    expect(theme.isDirty()).toBe(true);
    undoAction();
    expect(theme.getHistorySnapshot()).toEqual(before);
    expect(theme.isDirty()).toBe(false);
    redoAction();
    expect(theme.getBasePalette()).toEqual(before.basePalette);
    expect(theme.getPalette()).toEqual(before.palette);
    expect(theme.getIconTheme()).toEqual({mode: 'explicit', id: 'Ocean'});
});

test('blueprint loads restore the icon selection and clear it for legacy blueprints', () => {
    const blueprint = {
        name: 'Saved',
        timestamp: 0,
        palette: {colors: [...DEFAULT_PALETTE]},
    };
    loadBlueprintIntoEditor({
        ...blueprint,
        iconTheme: {mode: 'explicit', id: 'Missing-Icons'},
    });
    expect(theme.getIconTheme()).toEqual({
        mode: 'explicit',
        id: 'Missing-Icons',
    });
    loadBlueprintIntoEditor(blueprint);
    expect(theme.getIconTheme()).toEqual({mode: 'automatic'});
});

test('extraction preserves the icon choice and reset restores automatic', () => {
    theme.setIconTheme({mode: 'explicit', id: 'Ocean'}, true);
    theme.setPaletteFromExtraction('/new.png', [...DEFAULT_PALETTE]);
    expect(theme.getIconTheme()).toEqual({mode: 'explicit', id: 'Ocean'});
    theme.reset();
    expect(theme.getIconTheme()).toEqual({mode: 'automatic'});
});

test('an Omarchy theme import replaces the previous icon selection', async () => {
    theme.setIconTheme({mode: 'explicit', id: 'Previous'}, true);
    vi.mocked(LoadOmarchyThemes).mockResolvedValue([
        {
            name: 'Native',
            colors: [...DEFAULT_PALETTE],
            wallpapers: [],
            iconTheme: {mode: 'explicit', id: 'Native-Icons'},
        },
    ]);
    const {target} = render(OmarchyThemes, {});
    await settle();
    [...target.querySelectorAll('button')]
        .find(button => button.textContent?.trim() === 'Edit')!
        .click();
    expect(theme.getIconTheme()).toEqual({
        mode: 'explicit',
        id: 'Native-Icons',
    });
});

test('the chooser selects an installed theme and preserves it while Icons is disabled', async () => {
    const {target} = render(IconThemePicker, {});
    await settle();
    target
        .querySelector<HTMLButtonElement>(
            'button[aria-label^="Choose icon theme"]'
        )!
        .click();
    await settle();
    const option = target.querySelector<HTMLButtonElement>(
        'button[aria-label="Use icon theme Ocean"]'
    )!;
    expect(option.getAttribute('aria-pressed')).toBe('false');
    option.focus();
    option.click();
    await settle();
    expect(theme.getIconTheme()).toEqual({mode: 'explicit', id: 'Ocean'});
    expect(target.querySelector('[role="dialog"]')).toBeNull();
    target
        .querySelector<HTMLButtonElement>('[aria-label="Toggle Icons"]')!
        .click();
    await settle();
    expect(isAppIncluded('icons')).toBe(false);
    expect(
        target.querySelector<HTMLButtonElement>(
            'button[aria-label^="Choose icon theme"]'
        )!.disabled
    ).toBe(true);
    expect(theme.getIconTheme()).toEqual({mode: 'explicit', id: 'Ocean'});
});

test('catalog refresh reloads previews for unchanged theme IDs', async () => {
    const {target} = render(IconThemePicker, {});
    await settle();
    target
        .querySelector<HTMLButtonElement>(
            'button[aria-label^="Choose icon theme"]'
        )!
        .click();
    await settle();
    expect(GetIconThemePreview).toHaveBeenCalledTimes(1);
    [...target.querySelectorAll('button')]
        .find(button => button.textContent?.trim() === 'Refresh')!
        .click();
    await settle();
    expect(GetIconThemePreview).toHaveBeenCalledTimes(2);
});

test('a failed catalog refresh preserves the explicit selection', async () => {
    theme.setIconTheme({mode: 'explicit', id: 'Missing-Icons'}, true);
    const {target} = render(IconThemePicker, {});
    await settle();
    target
        .querySelector<HTMLButtonElement>(
            'button[aria-label^="Choose icon theme"]'
        )!
        .click();
    await settle();
    expect(target.textContent).toContain('Missing-Icons');
    vi.mocked(RefreshInstalledIconThemes).mockRejectedValueOnce(
        new Error('unavailable')
    );
    [...target.querySelectorAll('button')]
        .find(button => button.textContent?.trim() === 'Refresh')!
        .click();
    await settle();
    expect(target.querySelector('[role="status"]')?.textContent).toContain(
        'Could not refresh'
    );
    expect(theme.getIconTheme()).toEqual({
        mode: 'explicit',
        id: 'Missing-Icons',
    });
});
