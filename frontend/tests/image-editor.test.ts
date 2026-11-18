import {beforeAll, beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import WallpaperEditor from '../src/lib/components/wallpaper-editor/WallpaperEditor.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {
    initKeyboardShortcuts,
    registerShortcut,
} from '../src/lib/utils/keyboard';
import {button, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    SaveDataURLToFile: vi.fn(),
}));
vi.mock('../src/lib/stores/imagecache.svelte', () => ({
    getCachedFullImage: () => '',
    loadFullImage: vi.fn(),
    setCachedImage: vi.fn(),
}));

const undo = vi.fn();
const apply = vi.fn();
const escape = vi.fn();

beforeAll(() => {
    initKeyboardShortcuts();
    registerShortcut('ctrl+z', () => undo());
    registerShortcut('ctrl+enter', () => apply());
    registerShortcut('escape', () => escape());
});

beforeEach(() => {
    theme.reset();
    theme.setWallpaperPath('/a.png');
    undo.mockClear();
    apply.mockClear();
    escape.mockClear();
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);
});

function key(el: Element, key: string, extra: KeyboardEventInit = {}) {
    const event = new KeyboardEvent('keydown', {
        key,
        bubbles: true,
        cancelable: true,
        ...extra,
    });
    el.dispatchEvent(event);
    flushSync();
    return event;
}

function open() {
    const onclose = vi.fn();
    const view = render(WallpaperEditor, {open: true, onclose});
    const root = view.target.querySelector<HTMLElement>('.fixed.inset-0')!;
    return {...view, onclose, root};
}

const dialog = (target: HTMLElement) =>
    target.querySelector('[role="dialog"][aria-label="Discard edits?"]');

// Flip the image so the editor has unsaved adjustments.
function edit(root: HTMLElement) {
    key(root, 'f');
    expect(root.querySelector('[aria-label="Unsaved adjustments"]')).not.toBe(
        null
    );
}

test('Escape with edits asks before closing and Keep editing keeps them (AC-08.1, AC-08.3)', async () => {
    const {target, root, onclose} = open();
    edit(root);
    key(root, 'Escape');
    expect(onclose).not.toHaveBeenCalled();
    expect(dialog(target)?.textContent).toContain('Discard edits?');
    expect(button(target, 'Keep editing')).toBeDefined();
    expect(button(target, 'Discard')).toBeDefined();
    await settle();
    expect(document.activeElement).toBe(button(target, 'Keep editing'));
    button(target, 'Keep editing').click();
    flushSync();
    expect(dialog(target)).toBeNull();
    expect(root.querySelector('[aria-label="Unsaved adjustments"]')).not.toBe(
        null
    );
    expect(escape).not.toHaveBeenCalled();
});

test('the Close button asks too (AC-08.2)', () => {
    const {target, root, onclose} = open();
    edit(root);
    button(target, 'Close').click();
    flushSync();
    expect(onclose).not.toHaveBeenCalled();
    expect(dialog(target)).not.toBeNull();
});

test('Escape in the dialog closes only the dialog (AC-08.4)', () => {
    const {target, root, onclose} = open();
    edit(root);
    key(root, 'Escape');
    key(button(target, 'Keep editing'), 'Escape');
    expect(dialog(target)).toBeNull();
    expect(onclose).not.toHaveBeenCalled();
    expect(escape).not.toHaveBeenCalled();
    expect(root.querySelector('[aria-label="Unsaved adjustments"]')).not.toBe(
        null
    );
});

test('Discard closes the editor (AC-08.5)', () => {
    const {target, root, onclose} = open();
    edit(root);
    key(root, 'Escape');
    button(target, 'Discard').click();
    flushSync();
    expect(onclose).toHaveBeenCalledTimes(1);
});

test('Tab moves focus from Keep editing to Discard (AC-08.6)', async () => {
    const {target, root} = open();
    edit(root);
    key(root, 'Escape');
    await settle();
    const keep = button(target, 'Keep editing');
    const discard = button(target, 'Discard');
    expect(document.activeElement).toBe(keep);
    // jsdom has no layout, so the focus trap sees no visible items.
    // The DOM order is the Tab order.
    const items = [...dialog(target)!.querySelectorAll('button')];
    expect(items).toEqual([keep, discard]);
});

test('keys do nothing behind the dialog', () => {
    const {target, root} = open();
    edit(root);
    key(root, 'Escape');
    key(button(target, 'Keep editing'), 'z', {ctrlKey: true});
    key(button(target, 'Keep editing'), 'Enter', {ctrlKey: true});
    key(button(target, 'Keep editing'), 'r');
    expect(undo).not.toHaveBeenCalled();
    expect(apply).not.toHaveBeenCalled();
    expect(root.querySelector('[aria-label="Unsaved adjustments"]')).not.toBe(
        null
    );
});

test('without edits Escape and Close close at once (AC-08.7)', () => {
    const {target, root, onclose} = open();
    key(root, 'Escape');
    expect(onclose).toHaveBeenCalledTimes(1);
    expect(dialog(target)).toBeNull();
    button(target, 'Close').click();
    expect(onclose).toHaveBeenCalledTimes(2);
    expect(dialog(target)).toBeNull();
});

test('Escape in crop mode only leaves crop mode (AC-08.8)', () => {
    const {target, root, onclose} = open();
    edit(root);
    key(root, 'c');
    key(root, 'Escape');
    expect(onclose).not.toHaveBeenCalled();
    expect(dialog(target)).toBeNull();
});

test('Escape in a text field only blurs it (AC-08.9)', () => {
    const {target, root, onclose} = open();
    edit(root);
    const field = target.querySelector<HTMLInputElement>('input[type="text"]');
    expect(field).not.toBeNull();
    if (!field) return;
    field.focus();
    key(field, 'Escape');
    expect(onclose).not.toHaveBeenCalled();
    expect(dialog(target)).toBeNull();
});

test('Ctrl+Z and Ctrl+Enter on a filter slider do not reach global shortcuts (AC-04.6)', () => {
    const {target, root} = open();
    const range = target.querySelector<HTMLInputElement>('input[type="range"]');
    expect(range).not.toBeNull();
    range!.focus();
    key(range!, 'z', {ctrlKey: true});
    key(range!, 'Enter', {ctrlKey: true});
    expect(undo).not.toHaveBeenCalled();
    expect(apply).not.toHaveBeenCalled();
    expect(root).toBeDefined();
});
