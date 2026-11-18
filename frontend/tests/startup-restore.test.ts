import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import App from '../src/App.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {
    getApplySaveDialogOpen,
    setLiveApply,
} from '../src/lib/stores/ui.svelte';
import {initOmarchyCapabilities} from '../src/lib/stores/omarchy.svelte';
import {registerShortcut} from '../src/lib/utils/keyboard';
import {
    ApplyTheme,
    ExtractColors,
    GetInitialState,
} from '../wailsjs/go/main/App';
import {theme as backendTheme} from '../wailsjs/go/models';
import {render, settle} from './setup';

const success = {success: true, isOmarchy: false, themePath: '/theme'};

// Keep the real App effects and stores; unrelated panels and all Wails IO are mocked.
vi.mock(
    '../src/lib/components/layout/HeaderBar.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/layout/ActionBar.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/layout/TargetAppsStrip.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/editor/ThemeEditor.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/wallhaven/WallhavenBrowser.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/local/LocalBrowser.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/favorites/FavoritesView.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/blueprints/BlueprintsView.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/blueprints/OmarchyThemes.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/settings/SettingsView.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/layout/AboutView.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/shared/Toast.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/shared/KeymapDialog.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/shared/CommandPalette.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/ExternalImportDialog.svelte',
    () => import('./Empty.svelte')
);
vi.mock(
    '../src/lib/components/layout/ApplySaveDialog.svelte',
    () => import('./Empty.svelte')
);
vi.mock('../src/lib/commands/commands.svelte', () => ({
    buildCommands: () => [],
}));
vi.mock('../src/lib/utils/browser', () => ({prefersReducedMotion: () => true}));
vi.mock('../src/lib/utils/keyboard', () => ({
    initKeyboardShortcuts: vi.fn(),
    registerShortcut: vi.fn(),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
    getOmarchyAvailable: () => false,
    getOmarchyCapabilities: () => ({overrideApps: []}),
}));
vi.mock('../wailsjs/go/main/App', () => ({
    GetInitialState: vi.fn().mockResolvedValue({}),
    GetFocusTab: vi.fn().mockResolvedValue(''),
    GetThemeColors: vi.fn().mockResolvedValue({}),
    SyncState: vi.fn().mockResolvedValue(undefined),
    GetSettings: vi.fn().mockResolvedValue({}),
    ApplyTheme: vi.fn(),
    SaveAndApplyTheme: vi.fn(),
    ExtractColors: vi.fn(),
}));
vi.mock('../wailsjs/runtime/runtime', () => ({
    WindowShow: vi.fn(),
    WindowSetBackgroundColour: vi.fn(),
    EventsOn: vi.fn(),
}));

beforeEach(() => {
    theme.reset();
    theme.markApplied('');
    theme.setIsApplying(false);
    theme.setIsExtracting(false);
    setLiveApply(false);
    vi.mocked(ApplyTheme).mockReset().mockResolvedValue(success);
    vi.mocked(ExtractColors).mockReset();
    vi.mocked(registerShortcut).mockClear();
    vi.mocked(initOmarchyCapabilities).mockReset().mockResolvedValue();
});

test('a restored wallpaper is not stale and startup does not extract (AC-01.6, AC-03.7)', async () => {
    vi.mocked(GetInitialState)
        .mockReset()
        .mockResolvedValue(
            new backendTheme.StateSnapshot({
                wallpaperPath: '/restored.png',
                palette: Array(16).fill('#456789'),
            })
        );
    render(App, {});
    await settle();
    expect(theme.getWallpaperPath()).toBe('/restored.png');
    expect(theme.isPaletteStale()).toBe(false);
    expect(ExtractColors).not.toHaveBeenCalled();
});

test('the ctrl+enter shortcut applies only and opens no dialog (AC-02.4)', async () => {
    vi.mocked(GetInitialState)
        .mockReset()
        .mockResolvedValue(
            new backendTheme.StateSnapshot({wallpaperPath: '/restored.png'})
        );
    render(App, {});
    await settle();
    const call = vi
        .mocked(registerShortcut)
        .mock.calls.find(([combo]) => combo === 'ctrl+enter');
    expect(call).toBeDefined();
    call![1]();
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    expect(getApplySaveDialogOpen()).toBe(false);
});
