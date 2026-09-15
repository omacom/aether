import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import App from '../src/App.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {
    setLiveApply,
    getLiveApply,
    getLivePending,
} from '../src/lib/stores/ui.svelte';
import {applyTheme, applyThemeLive} from '../src/lib/actions/themeActions';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import {initOmarchyCapabilities} from '../src/lib/stores/omarchy.svelte';
import {ApplyTheme, GetInitialState} from '../wailsjs/go/main/App';
import {theme as backendTheme} from '../wailsjs/go/models';
import {deferred, render, settle} from './setup';

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
}));
vi.mock('../wailsjs/runtime/runtime', () => ({
    WindowShow: vi.fn(),
    WindowSetBackgroundColour: vi.fn(),
    EventsOn: vi.fn(),
}));

beforeEach(() => {
    vi.useFakeTimers();
    theme.reset();
    theme.markApplied('');
    theme.setIsApplying(false);
    theme.setIsExtracting(false);
    setLiveApply(false);
    vi.mocked(ApplyTheme).mockReset().mockResolvedValue(success);
    vi.mocked(GetInitialState)
        .mockReset()
        .mockResolvedValue(new backendTheme.StateSnapshot());
    vi.mocked(initOmarchyCapabilities).mockReset().mockResolvedValue();
});

async function queueLiveApply() {
    const view = render(App, {});
    await settle();
    setLiveApply(true);
    flushSync();
    theme.setColor(1, '#123456');
    flushSync();
    expect(getLivePending()).toBe(true);
    return view;
}

test('disabling Live Apply cancels pending work; subsequent enabled edits still apply', async () => {
    await queueLiveApply();
    await vi.advanceTimersByTimeAsync(500);
    setLiveApply(false);
    flushSync();
    expect(getLivePending()).toBe(false);
    await vi.advanceTimersByTimeAsync(2000);
    expect(ApplyTheme).not.toHaveBeenCalled();
    setLiveApply(true);
    flushSync();
    theme.setColor(1, '#abcdef');
    flushSync();
    await vi.advanceTimersByTimeAsync(1500);
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
});

test('the timer callback checks the toggle even before effect cleanup runs', async () => {
    await queueLiveApply();
    setLiveApply(false);
    // Synchronous timer advancement deliberately runs before the next Svelte flush.
    vi.advanceTimersByTime(1500);
    expect(ApplyTheme).not.toHaveBeenCalled();
    expect(getLivePending()).toBe(false);
});

test('destroying App cancels pending Live Apply', async () => {
    const view = await queueLiveApply();
    await view.destroy();
    await vi.advanceTimersByTimeAsync(2000);
    expect(ApplyTheme).not.toHaveBeenCalled();
    expect(getLivePending()).toBe(false);
});

test('the live action itself refuses to apply when disabled', async () => {
    await applyThemeLive();
    expect(ApplyTheme).not.toHaveBeenCalled();
    expect(initOmarchyCapabilities).not.toHaveBeenCalled();
});

test('mounting with Live Apply already enabled records the hydrated state without applying', async () => {
    setLiveApply(true);
    vi.mocked(GetInitialState).mockResolvedValue(
        new backendTheme.StateSnapshot({palette: Array(16).fill('#456789')})
    );
    render(App, {});
    await settle();
    await vi.advanceTimersByTimeAsync(3000);
    expect(ApplyTheme).not.toHaveBeenCalled();
});

test('OFF then ON retries a canceled queued edit without another edit', async () => {
    await queueLiveApply();
    const signature = theme.getThemeSignature();
    await vi.advanceTimersByTimeAsync(500);
    setLiveApply(false);
    flushSync();
    await vi.advanceTimersByTimeAsync(2000);
    setLiveApply(true);
    flushSync();
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    expect(theme.getLastAppliedSignature()).toBe(signature);
    expect(getLivePending()).toBe(false);
});

test('OFF/ON in the same tick rejects the old timer session and schedules the unsent edit again', async () => {
    await queueLiveApply();
    setLiveApply(false);
    setLiveApply(true);
    vi.advanceTimersByTime(1500);
    expect(ApplyTheme).not.toHaveBeenCalled();
    flushSync();
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
});

test.each(['disable', 'load blueprint'] as const)(
    '%s during preflight cannot dispatch a live apply',
    async change => {
        await queueLiveApply();
        const preflight = deferred<void>();
        vi.mocked(initOmarchyCapabilities).mockReturnValueOnce(
            preflight.promise
        );
        await vi.advanceTimersByTimeAsync(1500);
        await settle();
        expect(theme.getIsApplying()).toBe(true);
        if (change === 'disable') setLiveApply(false);
        else
            loadBlueprintIntoEditor({
                name: 'Review',
                timestamp: 0,
                palette: {
                    colors: Array(16).fill('#abcdef'),
                    wallpaper: '/review.png',
                },
            });
        flushSync();
        preflight.resolve();
        await settle();
        expect(ApplyTheme).not.toHaveBeenCalled();
        expect(theme.getLastAppliedSignature()).toBe('');
        expect(getLivePending()).toBe(false);
    }
);

test('OFF/ON during preflight rejects the old session but retries the unchanged edit', async () => {
    await queueLiveApply();
    const preflight = deferred<void>();
    vi.mocked(initOmarchyCapabilities).mockReturnValueOnce(preflight.promise);
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    setLiveApply(false);
    setLiveApply(true);
    flushSync();
    preflight.resolve();
    await settle();
    expect(ApplyTheme).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
});

test('an already-issued live apply marks only its old signature and uses its own CSS mode', async () => {
    await queueLiveApply();
    theme.setLightMode(true);
    flushSync();
    const signature = theme.getThemeSignature();
    const issued = deferred<typeof success>();
    vi.mocked(ApplyTheme).mockReturnValueOnce(issued.promise);
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    loadBlueprintIntoEditor({
        name: 'Review',
        timestamp: 0,
        palette: {
            colors: Array(16).fill('#abcdef'),
            lightMode: false,
            wallpaper: '/review.png',
        },
    });
    issued.resolve(success);
    await settle();
    expect(theme.getPalette()).toEqual(Array(16).fill('#abcdef'));
    expect(theme.getLastAppliedSignature()).toBe(signature);
    expect(theme.isDirty()).toBe(true);
    expect(theme.getLightMode()).toBe(false);
    expect(document.documentElement.classList.contains('light-mode')).toBe(
        true
    );
    await vi.advanceTimersByTimeAsync(3000);
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
});

test('a newer edit waits for an issued request, then applies its own snapshot', async () => {
    await queueLiveApply();
    const issued = deferred<typeof success>();
    vi.mocked(ApplyTheme).mockReturnValueOnce(issued.promise);
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    theme.setColor(1, '#abcdef');
    flushSync();
    await vi.advanceTimersByTimeAsync(3000);
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    issued.resolve(success);
    await settle();
    expect(theme.isDirty()).toBe(true);
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(2);
    expect(vi.mocked(ApplyTheme).mock.calls[1][0].palette[1]).toBe('#abcdef');
    expect(theme.isDirty()).toBe(false);
});

test('failed live applies do not loop, and OFF/ON explicitly retries the unsent signature', async () => {
    await queueLiveApply();
    vi.mocked(ApplyTheme).mockRejectedValueOnce(new Error('offline'));
    await vi.advanceTimersByTimeAsync(10000);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    expect(getLivePending()).toBe(false);
    setLiveApply(false);
    flushSync();
    setLiveApply(true);
    flushSync();
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(2);
});

test('destroying App also invalidates live preflight without changing the saved toggle preference', async () => {
    const view = await queueLiveApply();
    const preflight = deferred<void>();
    vi.mocked(initOmarchyCapabilities).mockReturnValueOnce(preflight.promise);
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    await view.destroy();
    preflight.resolve();
    await settle();
    expect(ApplyTheme).not.toHaveBeenCalled();
    expect(getLivePending()).toBe(false);
    expect(getLiveApply()).toBe(true);
});

test('returning to an earlier successful live signature after a manual apply still applies the edit', async () => {
    await queueLiveApply();
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    theme.setColor(1, '#abcdef');
    flushSync();
    await applyTheme();
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(2);
    theme.setColor(1, '#123456');
    flushSync();
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(3);
    expect(theme.isDirty()).toBe(false);
});

test('a new edit clears a failed attempt even when a later edit returns to the same signature', async () => {
    await queueLiveApply();
    vi.mocked(ApplyTheme).mockRejectedValueOnce(new Error('offline'));
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    theme.setColor(1, '#abcdef');
    flushSync();
    theme.setColor(1, '#123456');
    flushSync();
    await vi.advanceTimersByTimeAsync(1500);
    await settle();
    expect(ApplyTheme).toHaveBeenCalledTimes(2);
    expect(theme.isDirty()).toBe(false);
});
