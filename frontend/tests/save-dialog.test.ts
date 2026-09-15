import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import SaveDialog from '../src/lib/components/blueprints/SaveDialog.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {BlueprintExists, SaveBlueprint} from '../wailsjs/go/main/App';
import {button, deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    BlueprintExists: vi.fn(),
    SaveBlueprint: vi.fn(),
}));

beforeEach(() => {
    theme.reset();
    vi.mocked(BlueprintExists).mockReset();
    vi.mocked(SaveBlueprint).mockReset();
});

function enter() {
    window.dispatchEvent(
        new KeyboardEvent('keydown', {key: 'Enter', bubbles: true})
    );
    flushSync();
}

function setName(target: HTMLElement, name: string) {
    const input = target.querySelector('input')!;
    input.value = name;
    input.dispatchEvent(new Event('input', {bubbles: true}));
    input.focus();
    flushSync();
}

test('repeated Enter cannot bypass overwrite confirmation, and Override saves the checked snapshot once', async () => {
    const exists = deferred<boolean>();
    const saved = deferred<void>();
    vi.mocked(BlueprintExists).mockReturnValue(exists.promise);
    vi.mocked(SaveBlueprint).mockReturnValue(saved.promise);
    theme.setWallpaperPath('/original.png');
    theme.setAppOverride('kitty', 'background', '#123456');
    theme.setIconTheme({mode: 'explicit', id: 'Original-Icons'}, true);
    const originalPalette = [...theme.getPalette()];
    const onsave = vi.fn();
    const {target} = render(SaveDialog, {open: true, onclose: vi.fn(), onsave});

    setName(target, '  Original  ');
    enter();
    enter();
    await settle();
    expect(BlueprintExists).toHaveBeenCalledExactlyOnceWith('Original');
    expect(target.querySelector('input')!.disabled).toBe(true);

    // Even programmatic edits while the name check is pending cannot retarget it.
    setName(target, 'Different');
    theme.setColor(0, '#abcdef');
    theme.setWallpaperPath('/different.png');
    theme.setAppOverride('kitty', 'background', '#ffffff');
    theme.setIconTheme({mode: 'explicit', id: 'Later-Icons'}, true);
    exists.resolve(true);
    await settle();
    expect(target.textContent).toContain('A theme named "Original"');
    enter();
    expect(SaveBlueprint).not.toHaveBeenCalled();

    const override = button(target, 'Override');
    override.click();
    override.click();
    await settle();
    expect(SaveBlueprint).toHaveBeenCalledTimes(1);
    expect(SaveBlueprint).toHaveBeenCalledWith(
        expect.objectContaining({
            name: 'Original',
            palette: originalPalette,
            wallpaperPath: '/original.png',
            appOverrides: {kitty: {background: '#123456'}},
            iconTheme: {mode: 'explicit', id: 'Original-Icons'},
        })
    );
    expect(onsave).not.toHaveBeenCalled();
    saved.resolve();
    await settle();
    expect(onsave).toHaveBeenCalledTimes(1);
});

test('new-name saves stay single-flight until the write completes', async () => {
    vi.mocked(BlueprintExists).mockResolvedValue(false);
    const saved = deferred<void>();
    vi.mocked(SaveBlueprint).mockReturnValue(saved.promise);
    const {target} = render(SaveDialog, {onclose: vi.fn(), onsave: vi.fn()});
    setName(target, 'New');
    enter();
    await settle();
    enter();
    enter();
    await settle();
    expect(BlueprintExists).toHaveBeenCalledTimes(1);
    expect(SaveBlueprint).toHaveBeenCalledTimes(1);
    saved.resolve();
    await settle();
});

test('a failed lookup can be retried without saving, and canceling confirmation checks the next name afresh', async () => {
    vi.mocked(BlueprintExists)
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValueOnce(true)
        .mockResolvedValueOnce(false);
    vi.mocked(SaveBlueprint).mockResolvedValue();
    const {target} = render(SaveDialog, {onclose: vi.fn(), onsave: vi.fn()});
    setName(target, 'Existing');
    enter();
    await settle();
    expect(SaveBlueprint).not.toHaveBeenCalled();
    expect(button(target, 'Save').disabled).toBe(false);
    enter();
    await settle();
    button(target, 'Cancel').click();
    flushSync();
    setName(target, 'Fresh');
    enter();
    await settle();
    expect(BlueprintExists).toHaveBeenLastCalledWith('Fresh');
    expect(SaveBlueprint).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({name: 'Fresh'})
    );
});

test('destroying the dialog invalidates an outstanding name check', async () => {
    const exists = deferred<boolean>();
    vi.mocked(BlueprintExists).mockReturnValue(exists.promise);
    const {target, destroy} = render(SaveDialog, {
        onclose: vi.fn(),
        onsave: vi.fn(),
    });
    setName(target, 'Abandoned');
    enter();
    await settle();
    await destroy();
    exists.resolve(false);
    await settle();
    expect(SaveBlueprint).not.toHaveBeenCalled();
});
