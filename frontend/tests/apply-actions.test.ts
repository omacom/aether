import {beforeEach, expect, test, vi} from 'vitest';
import {
    applyTheme,
    requestThemeApply,
    saveAndApplyTheme,
    applyWallpaperOnly,
} from '../src/lib/actions/themeActions';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import * as theme from '../src/lib/stores/theme.svelte';
import * as ui from '../src/lib/stores/ui.svelte';
import {setLiveApply} from '../src/lib/stores/ui.svelte';
import {getSettings, updateSettings} from '../src/lib/stores/settings.svelte';
import {initOmarchyCapabilities} from '../src/lib/stores/omarchy.svelte';
import {STORAGE_KEYS} from '../src/lib/constants/storage';
import {
    ApplyTheme,
    ApplyWallpaperOnly,
    SaveAndApplyTheme,
    DownloadWallpaper,
} from '../wailsjs/go/main/App';
import {deferred, settle} from './setup';

const success = {success: true, isOmarchy: false, themePath: '/theme'};

vi.mock('../wailsjs/go/main/App', () => ({
    ApplyTheme: vi.fn(),
    ApplyWallpaperOnly: vi.fn().mockResolvedValue(undefined),
    SaveAndApplyTheme: vi.fn(),
    DownloadWallpaper: vi.fn(),
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
    theme.markApplied('');
    theme.setIsApplying(false);
    setLiveApply(false);
    document.documentElement.classList.remove('light-mode');
    ui.setApplySaveDialogOpen(false);
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
    vi.mocked(ApplyTheme).mockReset().mockResolvedValue(success);
    vi.mocked(ApplyWallpaperOnly).mockReset().mockResolvedValue(undefined);
    vi.mocked(SaveAndApplyTheme).mockReset().mockResolvedValue(success);
    vi.mocked(DownloadWallpaper).mockReset();
    vi.mocked(initOmarchyCapabilities).mockReset().mockResolvedValue();
});

test.each(['apply', 'save'] as const)(
    '%s captures before preflight and never marks or associates newer editor state',
    async action => {
        theme.setWallpaperPath('/original.png');
        theme.setWallpaperBlur(true, true);
        theme.setLightMode(true);
        theme.setAdditionalImages(['/extra.png']);
        theme.setExtendedColor('accent', '#123456');
        theme.setNativeColors({outline: '#112233'});
        theme.setIconTheme({mode: 'explicit', id: 'Original-Icons'}, true);
        theme.setAppOverride('kitty', 'background', '#123456');
        updateSettings({includedApps: {kitty: true}});
        const original = theme.getThemeSnapshot();
        const signature = theme.getThemeSignature(original);
        const preflight = deferred<void>();
        const issued = deferred<typeof success>();
        vi.mocked(initOmarchyCapabilities).mockReturnValueOnce(
            preflight.promise
        );
        const backend = action === 'apply' ? ApplyTheme : SaveAndApplyTheme;
        vi.mocked(backend).mockReturnValueOnce(issued.promise);
        const operation =
            action === 'apply'
                ? applyTheme()
                : saveAndApplyTheme('Original folder', true);

        // Mutate every nested input while capability discovery is still pending.
        theme.setColor(1, '#abcdef');
        theme.setLightMode(false);
        theme.setWallpaperPath('/review.png');
        theme.setAdditionalImages(['/new-extra.png']);
        theme.setExtendedColor('accent', '#ffffff');
        theme.setNativeColors({outline: '#ffffff'});
        theme.setIconTheme({mode: 'explicit', id: 'Later-Icons'}, true);
        theme.setAppOverride('kitty', 'background', '#ffffff');
        getSettings().includedApps!.kitty = false;
        preflight.resolve();
        await settle();
        expect(backend).toHaveBeenCalledTimes(1);
        expect(backend).toHaveBeenCalledWith(
            expect.objectContaining({
                ...original,
                settings: expect.objectContaining({
                    includedApps: {kitty: true},
                }),
            })
        );
        if (action === 'save')
            expect(backend).toHaveBeenCalledWith(
                expect.objectContaining({
                    name: 'Original folder',
                    updateExisting: true,
                })
            );

        theme.setWallpaperPath('/even-newer.png');
        issued.resolve(success);
        await operation;
        expect(theme.getLastAppliedSignature()).toBe(signature);
        expect(theme.isDirty()).toBe(true);
        expect(document.documentElement.classList.contains('light-mode')).toBe(
            true
        );
        expect(theme.getIsApplying()).toBe(false);
        if (action === 'save') {
            expect(
                JSON.parse(
                    localStorage.getItem(STORAGE_KEYS.savedThemeFolders)!
                )
            ).toEqual({'/original.png': 'Original folder'});
        }
    }
);

test('a wallpaper download cannot replace a newly loaded blueprint', async () => {
    theme.setWallpaperPath('/original.png');
    theme.setLightMode(true);
    const download = deferred<string>();
    vi.mocked(DownloadWallpaper).mockReturnValue(download.promise);
    const operation = applyWallpaperOnly('https://example.com/wallpaper.png');
    await settle();
    expect(theme.getIsApplying()).toBe(true);
    await applyTheme();
    expect(ApplyTheme).not.toHaveBeenCalled();
    loadBlueprintIntoEditor({
        name: 'Review',
        timestamp: 0,
        palette: {
            colors: Array(16).fill('#abcdef'),
            wallpaper: '/review.png',
            lightMode: false,
        },
    });
    download.resolve('/downloaded.png');
    await operation;
    expect(ApplyTheme).not.toHaveBeenCalled();
    expect(ApplyWallpaperOnly).not.toHaveBeenCalled();
    expect(theme.getWallpaperPath()).toBe('/review.png');
    expect(theme.getIsApplying()).toBe(false);
});

test('a wallpaper-only change preserves colors and does not acknowledge a full theme apply', async () => {
    theme.setWallpaperPath('/original.png');
    theme.setWallpaperBlur(true, true);
    theme.markApplied();
    const original = theme.getThemeSnapshot();
    const applied = theme.getLastAppliedSignature();
    await applyWallpaperOnly('/local.png');
    expect(ApplyWallpaperOnly).toHaveBeenCalledExactlyOnceWith('/local.png');
    expect(ApplyTheme).not.toHaveBeenCalled();
    expect(theme.getThemeSnapshot()).toEqual({
        ...original,
        wallpaperPath: '/local.png',
        wallpaperBlur: false,
    });
    expect(theme.getLastAppliedSignature()).toBe(applied);
});

test('a failed wallpaper-only change preserves editor state', async () => {
    theme.setWallpaperPath('/original.png');
    const before = theme.getThemeSnapshot();
    vi.mocked(ApplyWallpaperOnly).mockRejectedValueOnce(
        new Error('background failed')
    );
    await applyWallpaperOnly('/missing.png');
    expect(theme.getThemeSnapshot()).toEqual(before);
    expect(theme.getIsApplying()).toBe(false);
});

test('failed apply/save requests do not acknowledge editor state or save a folder association', async () => {
    theme.markApplied();
    const applied = theme.getLastAppliedSignature();
    theme.setColor(0, '#abcdef');
    theme.setWallpaperPath('/new.png');
    vi.mocked(ApplyTheme).mockRejectedValueOnce(new Error('failed'));
    await applyTheme();
    vi.mocked(SaveAndApplyTheme).mockRejectedValueOnce(new Error('failed'));
    await saveAndApplyTheme('Missing folder');
    expect(theme.getLastAppliedSignature()).toBe(applied);
    expect(theme.isDirty()).toBe(true);
    expect(localStorage.getItem(STORAGE_KEYS.savedThemeFolders)).toBe(null);
    expect(theme.getIsApplying()).toBe(false);
});

test('the primary apply opens the naming dialog and Ctrl+Enter applies only (AC-02.4)', async () => {
    theme.setWallpaperPath('/new.png');
    requestThemeApply();
    expect(ui.getApplySaveDialogOpen()).toBe(true);
    expect(ApplyTheme).not.toHaveBeenCalled();
    ui.setApplySaveDialogOpen(false);
    await applyTheme();
    expect(ApplyTheme).toHaveBeenCalledTimes(1);
    expect(ui.getApplySaveDialogOpen()).toBe(false);
});

test('a rejected apply shows the backend cause (AC-07.1)', async () => {
    vi.mocked(ApplyTheme).mockRejectedValueOnce('template kitty: unknown role');
    await applyTheme();
    expect(ui.getToastMessage()).toContain('template kitty: unknown role');
    expect(ui.getToastMessage()).not.toContain('see logs');
});

test('apply with success false is a failure (AC-07.2)', async () => {
    theme.markApplied();
    theme.setColor(0, '#abcdef');
    const applied = theme.getLastAppliedSignature();
    vi.mocked(ApplyTheme).mockResolvedValueOnce({
        success: false,
        isOmarchy: false,
        themePath: '',
    });
    await applyTheme();
    const message = ui.getToastMessage();
    expect(message.startsWith('Couldn’t apply theme')).toBe(true);
    expect(message).not.toContain('Theme applied');
    expect(message).not.toContain('Theme files generated');
    expect(theme.getLastAppliedSignature()).toBe(applied);
});

test('save and apply with success false is a failure and saves no folder (AC-07.12)', async () => {
    theme.setWallpaperPath('/a.png');
    theme.markApplied();
    const applied = theme.getLastAppliedSignature();
    vi.mocked(SaveAndApplyTheme).mockResolvedValueOnce({
        success: false,
        isOmarchy: false,
        themePath: '',
    });
    expect(await saveAndApplyTheme('test-theme')).toBe(false);
    const message = ui.getToastMessage();
    expect(message.startsWith('Couldn’t apply theme')).toBe(true);
    expect(message).not.toContain('Saved and applied');
    expect(message).not.toContain('Applied:');
    expect(theme.getLastAppliedSignature()).toBe(applied);
    expect(localStorage.getItem(STORAGE_KEYS.savedThemeFolders)).toBe(null);
});

test('the primary apply with a saved folder reports success false (AC-07.13)', async () => {
    theme.setWallpaperPath('/a.png');
    localStorage.setItem(
        STORAGE_KEYS.savedThemeFolders,
        JSON.stringify({'/a.png': 'test-theme'})
    );
    theme.markApplied();
    const applied = theme.getLastAppliedSignature();
    vi.mocked(SaveAndApplyTheme).mockResolvedValueOnce({
        success: false,
        isOmarchy: false,
        themePath: '',
    });
    requestThemeApply();
    await settle();
    expect(ui.getToastMessage().startsWith('Couldn’t apply theme')).toBe(true);
    expect(theme.getLastAppliedSignature()).toBe(applied);
});
