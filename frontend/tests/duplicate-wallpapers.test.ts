import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import App from '../src/App.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {
    getLivePending,
    setLiveApply,
    showToast,
} from '../src/lib/stores/ui.svelte';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import {initOmarchyCapabilities} from '$lib/stores/omarchy.svelte';
import {ApplyTheme, GetInitialState, SyncState} from '../wailsjs/go/main/App';
import {theme as backendTheme} from '../wailsjs/go/models';
import {render, settle} from './setup';

const success = {success: true, isOmarchy: false, themePath: '/theme'};

// App installs the IPC listener on mount, so the harness records the callbacks
// to replay a backend `ipc-state-changed` echo.
const {eventHandlers} = vi.hoisted(() => ({
    eventHandlers: new Map<string, (payload: never) => void>(),
}));

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
}));
vi.mock('../src/lib/stores/ui.svelte', async importOriginal => ({
    ...(await importOriginal<typeof import('../src/lib/stores/ui.svelte')>()),
    showToast: vi.fn(),
}));
vi.mock('../wailsjs/runtime/runtime', () => ({
    WindowShow: vi.fn(),
    WindowSetBackgroundColour: vi.fn(),
    EventsOn: vi.fn((name: string, handler: (payload: never) => void) => {
        eventHandlers.set(name, handler);
    }),
}));

beforeEach(() => {
    vi.useFakeTimers();
    eventHandlers.clear();
    theme.reset();
    theme.markApplied('');
    theme.setIsApplying(false);
    theme.setIsExtracting(false);
    setLiveApply(false);
    vi.mocked(showToast).mockClear();
    vi.mocked(ApplyTheme).mockReset().mockResolvedValue(success);
    vi.mocked(SyncState).mockReset().mockResolvedValue(undefined);
    vi.mocked(GetInitialState)
        .mockReset()
        .mockResolvedValue(new backendTheme.StateSnapshot());
    vi.mocked(initOmarchyCapabilities).mockReset().mockResolvedValue();
});

test('importing a blueprint drops duplicated paths and shared basenames', () => {
    loadBlueprintIntoEditor({
        name: 'Dupes',
        timestamp: 0,
        palette: {
            colors: Array(16).fill('#abcdef'),
            wallpaper: '/wallpapers/main.png',
            additionalImages: [
                '/wallpapers/main.png', // same file as the main wallpaper
                '/extra/a.png',
                '/extra/a.png', // exact path duplicate
                '/other/a.png', // same basename as the entry above
                '/other/b.png',
            ],
        },
    });

    expect(theme.getWallpaperPath()).toBe('/wallpapers/main.png');
    expect(theme.getAdditionalImages()).toEqual([
        '/extra/a.png',
        '/other/b.png',
    ]);

    const sent = theme.getThemeSnapshot().additionalImages;
    const basenames = sent.map(theme.imageBasename);
    expect(new Set(basenames).size).toBe(basenames.length);
    expect(basenames).not.toContain('main.png');
});

test('adding a wallpaper whose filename is already staged is rejected', () => {
    theme.setWallpaperPath('/w/main.png');
    expect(theme.addAdditionalImage('/other/main.png')).toBe(false);
    expect(theme.addAdditionalImage('/other/extra.png')).toBe(true);
    expect(theme.addAdditionalImage('/third/extra.png')).toBe(false);
    expect(theme.addAdditionalImage('/third/extra2.png')).toBe(true);
    expect(theme.getAdditionalImages()).toEqual([
        '/other/extra.png',
        '/third/extra2.png',
    ]);
});

test('changing the main wallpaper drops additional images that would collide', () => {
    theme.setWallpaperPath('/w/first.png');
    expect(theme.addAdditionalImage('/other/second.png')).toBe(true);
    expect(theme.addAdditionalImage('/other/keep.png')).toBe(true);

    theme.setWallpaperPath('/w/second.png');

    expect(theme.getAdditionalImages()).toEqual(['/other/keep.png']);
    expect(showToast).toHaveBeenCalledWith(
        'Removed 1 additional image with the same filename as the main wallpaper'
    );
});

test('swapping keeps the previous main wallpaper as an additional image', () => {
    theme.setWallpaperPath('/w/main.png');
    expect(theme.addAdditionalImage('/other/extra.png')).toBe(true);
    expect(theme.addAdditionalImage('/other/source.png')).toBe(true);

    theme.swapMainWithAdditional('/other/source.png');

    expect(theme.getWallpaperPath()).toBe('/other/source.png');
    expect(theme.getAdditionalImages()).toEqual([
        '/other/extra.png',
        '/w/main.png',
    ]);
    expect(showToast).not.toHaveBeenCalled();
});

test('a duplicate-basename import applies once with unique names', async () => {
    const view = render(App, {});
    await settle();

    loadBlueprintIntoEditor({
        name: 'Dupes',
        timestamp: 0,
        palette: {
            colors: Array(16).fill('#abcdef'),
            wallpaper: '/wallpapers/main.png',
            additionalImages: ['/extra/a.png', '/other/a.png', '/extra/a.png'],
        },
    });
    setLiveApply(true);
    flushSync();
    theme.setColor(1, '#123456');
    flushSync();
    expect(getLivePending()).toBe(true);

    vi.mocked(ApplyTheme).mockRejectedValueOnce(
        new Error(
            'background basename collision "a.png": "/extra/a.png" and "/other/a.png"'
        )
    );
    await vi.advanceTimersByTimeAsync(1500);
    await settle();

    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    const request = vi.mocked(ApplyTheme).mock.calls[0][0];
    expect(request.wallpaperPath).toBe('/wallpapers/main.png');
    // The GUI never sends the colliding set the backend would reject.
    expect(request.additionalImages).toEqual(['/extra/a.png']);

    // The failure clears the apply lock and is not retried on its own.
    expect(theme.getIsApplying()).toBe(false);
    await vi.advanceTimersByTimeAsync(15000);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    expect(theme.getIsApplying()).toBe(false);
    expect(getLivePending()).toBe(false);
    await view.destroy();
});

test('an identical backend state echo does not re-dispatch SyncState', async () => {
    render(App, {});
    await settle();

    theme.setAdditionalImages(['/w/a.png']);
    flushSync();
    await vi.advanceTimersByTimeAsync(500);
    await settle();

    const dispatched = vi.mocked(SyncState).mock.calls.length;
    expect(dispatched).toBeGreaterThan(0);

    const echo = eventHandlers.get('ipc-state-changed');
    expect(echo).toBeDefined();
    (echo as (payload: unknown) => void)({
        palette: theme.getPalette(),
        extendedColors: theme.getExtendedColors(),
        nativeColors: theme.getNativeColors(),
        lightMode: theme.getLightMode(),
        mode: theme.getExtractionMode(),
        wallpaper: theme.getWallpaperPath(),
        wallpaperBlur: theme.getWallpaperBlur(),
        appOverrides: theme.getAppOverrides(),
        additionalImages: theme.getAdditionalImages(),
        adjustments: theme.getAdjustments(),
        iconTheme: theme.getIconTheme(),
    });

    flushSync();
    await vi.advanceTimersByTimeAsync(1000);
    await settle();

    expect(vi.mocked(SyncState).mock.calls.length).toBe(dispatched);
});
