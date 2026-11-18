import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import ActionBar from '../src/lib/components/layout/ActionBar.svelte';
import BlueprintsView from '../src/lib/components/blueprints/BlueprintsView.svelte';
import WallpaperPicker from '../src/lib/components/editor/WallpaperPicker.svelte';
import Toast from '../src/lib/components/shared/Toast.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import * as ui from '../src/lib/stores/ui.svelte';
import {errorMessage, failureMessage} from '../src/lib/utils/errors';
import {
    ClearTheme,
    ListBlueprints,
    OpenFileDialog,
} from '../wailsjs/go/main/App';
import {button, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    ClearTheme: vi.fn(),
    ListBlueprints: vi.fn(),
    OpenFileDialog: vi.fn(),
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
    ApplyTheme: vi.fn(),
    SaveAndApplyTheme: vi.fn(),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
    getOmarchyAvailable: () => false,
    getOmarchyCapabilities: () => ({overrideApps: []}),
}));

beforeEach(() => {
    theme.reset();
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
    vi.mocked(ClearTheme).mockReset();
    vi.mocked(ListBlueprints).mockReset().mockResolvedValue([]);
    vi.mocked(OpenFileDialog).mockReset();
});

test('errorMessage reads strings and Error objects and strips the prefix', () => {
    expect(errorMessage('read blueprints: denied')).toBe(
        'read blueprints: denied'
    );
    expect(errorMessage(new Error('boom'))).toBe('boom');
    expect(errorMessage('Error: boom')).toBe('boom');
    expect(errorMessage(undefined)).toBe('');
    expect(errorMessage({code: 1})).toBe('');
    expect(failureMessage('Couldn’t x', 'why')).toBe('Couldn’t x: why');
    expect(failureMessage('Couldn’t x')).toBe('Couldn’t x');
});

test.each([
    ['a string', 'restore theme: permission denied'],
    ['an Error', new Error('restore theme: permission denied')],
])('a failed revert shows the cause from %s (AC-07.4)', async (_, failure) => {
    vi.mocked(ClearTheme).mockRejectedValueOnce(failure);
    const {target} = render(ActionBar, {});
    button(target, 'More actions').click();
    flushSync();
    [...target.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')]
        .find(el => el.textContent?.includes('Revert system theme'))!
        .click();
    flushSync();
    button(document.body, 'Revert').click();
    await settle();
    const message = ui.getToastMessage();
    expect(message).toContain('restore theme: permission denied');
    expect(message).not.toContain('see logs');
});

test('a failed blueprint list shows the cause and Retry recovers (AC-07.5, AC-07.6)', async () => {
    vi.mocked(ListBlueprints).mockRejectedValueOnce(
        'read blueprints: permission denied'
    );
    const {target} = render(BlueprintsView, {});
    await settle();
    expect(target.textContent).toContain('read blueprints: permission denied');
    expect(target.textContent).not.toContain('No themes saved yet');
    expect(target.textContent).not.toContain('saved');
    vi.mocked(ListBlueprints).mockResolvedValueOnce([
        {
            name: 'Recovered',
            timestamp: 1,
            palette: {colors: Array(16).fill('#123456')},
        },
    ] as never);
    button(target, 'Retry').click();
    await settle();
    expect(target.textContent).toContain('Recovered');
    expect(target.textContent).not.toContain('Retry');
});

test('a second failed load shows the error again with the focus on Retry', async () => {
    vi.mocked(ListBlueprints).mockRejectedValue('first');
    const {target} = render(BlueprintsView, {});
    await settle();
    vi.mocked(ListBlueprints).mockRejectedValue('second');
    button(target, 'Retry').click();
    await settle();
    expect(target.textContent).toContain('second');
    expect(document.activeElement).toBe(button(target, 'Retry'));
});

test('a failed load with no text still shows the error state', async () => {
    vi.mocked(ListBlueprints).mockRejectedValueOnce(undefined);
    const {target} = render(BlueprintsView, {});
    await settle();
    expect(target.textContent).toContain('Couldn’t load your themes');
    expect(target.textContent).not.toContain('No themes saved yet');
});

test('Browse files reports a dialog failure and stays silent on cancel (AC-07.7)', async () => {
    vi.mocked(OpenFileDialog).mockRejectedValueOnce('dialog: no portal');
    const {target} = render(WallpaperPicker, {});
    button(target, 'Browse files').click();
    await settle();
    expect(ui.getToastMessage()).toContain('dialog: no portal');
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
    vi.mocked(OpenFileDialog).mockResolvedValueOnce('');
    button(target, 'Browse files').click();
    await settle();
    expect(ui.getToastQueueDepth()).toBe(0);
});

test('an error toast stays for 6 seconds and Dismiss closes it at once (AC-07.8)', async () => {
    vi.useFakeTimers();
    const {target} = render(Toast, {});
    ui.showErrorToast('Couldn’t apply theme', 'template kitty: unknown role');
    flushSync();
    expect(target.textContent).toContain('template kitty: unknown role');
    vi.advanceTimersByTime(5900);
    flushSync();
    expect(target.textContent).toContain('template kitty: unknown role');
    vi.advanceTimersByTime(200);
    flushSync();
    expect(target.textContent).not.toContain('template kitty');

    ui.showErrorToast('Couldn’t apply theme', 'again');
    flushSync();
    button(target, 'Dismiss').click();
    flushSync();
    expect(target.textContent).not.toContain('again');
});
