import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import ApplySaveDialog from '../src/lib/components/layout/ApplySaveDialog.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {ThemeFolderExists, SaveAndApplyTheme} from '../wailsjs/go/main/App';
import {button, deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    ThemeFolderExists: vi.fn(),
    SaveAndApplyTheme: vi.fn(),
    ApplyTheme: vi.fn(),
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
    getOmarchyAvailable: () => true,
    getOmarchyCapabilities: () => ({overrideApps: ['kitty']}),
}));

beforeEach(() => {
    theme.reset();
    theme.setIsApplying(false);
    theme.setWallpaperPath('/original.png');
    vi.mocked(ThemeFolderExists).mockReset().mockResolvedValue(false);
    vi.mocked(SaveAndApplyTheme).mockReset().mockResolvedValue({
        success: true,
        isOmarchy: true,
        themePath: '/theme',
    });
});

function enter() {
    window.dispatchEvent(
        new KeyboardEvent('keydown', {key: 'Enter', bubbles: true})
    );
    flushSync();
}

test('an existing folder requires a distinct confirmation and uses the captured state', async () => {
    const check = deferred<boolean>();
    vi.mocked(ThemeFolderExists).mockReturnValue(check.promise);
    theme.setWallpaperBlur(true, true);
    const onclose = vi.fn();
    const {target} = render(ApplySaveDialog, {open: true, onclose});
    enter();
    await settle();
    theme.setWallpaperPath('/later.png');
    theme.setIconTheme({mode: 'explicit', id: 'Later'}, true);
    check.resolve(true);
    await settle();
    enter();
    expect(SaveAndApplyTheme).not.toHaveBeenCalled();
    button(target, 'Update and Apply').click();
    await settle();
    expect(SaveAndApplyTheme).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({
            name: 'original',
            updateExisting: true,
            wallpaperPath: '/original.png',
            wallpaperBlur: true,
            iconTheme: {mode: 'automatic'},
        })
    );
    expect(onclose).toHaveBeenCalledTimes(1);
});

test('a failed existence check permits a retry and never starts a save', async () => {
    vi.mocked(ThemeFolderExists).mockRejectedValueOnce(
        new Error('lookup failed')
    );
    const {target} = render(ApplySaveDialog, {open: true, onclose: vi.fn()});
    enter();
    await settle();
    expect(target.querySelector('[role="alert"]')?.textContent).toContain(
        'lookup failed'
    );
    expect(SaveAndApplyTheme).not.toHaveBeenCalled();
    button(target, 'Save and Apply').click();
    await settle();
    expect(SaveAndApplyTheme).toHaveBeenCalledTimes(1);
});

test('source changes after the dialog closes cannot complete an old lookup', async () => {
    const check = deferred<boolean>();
    vi.mocked(ThemeFolderExists).mockReturnValue(check.promise);
    const view = render(ApplySaveDialog, {open: true, onclose: vi.fn()});
    enter();
    await settle();
    await view.destroy();
    check.resolve(false);
    await settle();
    expect(SaveAndApplyTheme).not.toHaveBeenCalled();
});

test('the suggested name can be cleared and replaced', () => {
    const {target} = render(ApplySaveDialog, {open: true, onclose: vi.fn()});
    const input = target.querySelector('input')!;
    input.value = '';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    flushSync();
    expect(input.value).toBe('');
    expect(button(target, 'Save and Apply').disabled).toBe(true);
});
