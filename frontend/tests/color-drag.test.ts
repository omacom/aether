import {afterEach, beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import ColorSwatch from '../src/lib/components/editor/ColorSwatch.svelte';
import AppColorOverrides from '../src/lib/components/editor/AppColorOverrides.svelte';
import ColorDragGhost from '../src/lib/components/editor/ColorDragGhost.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import * as ui from '../src/lib/stores/ui.svelte';
import * as history from '../src/lib/stores/history.svelte';
import {undoAction, redoAction} from '../src/lib/actions/themeActions';
import {render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    GetTemplateColors: vi
        .fn()
        .mockResolvedValue({kitty: ['background', 'foreground']}),
    ComputeVariables: vi
        .fn()
        .mockResolvedValue({background: '#000000', foreground: '#ffffff'}),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    getOmarchyAvailable: () => true,
    getOmarchyCapabilities: () => ({overrideApps: ['kitty']}),
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
}));

beforeEach(() => {
    theme.reset();
    ui.setColorDrag(null);
    ui.closeColorPicker();
    document.body.style.cursor = '';
});

afterEach(() => {
    window.dispatchEvent(new Event('blur'));
    ui.setColorDrag(null);
    document.body.style.cursor = '';
});

function mouse(target: EventTarget, type: string, x = 20, y = 20, buttons = 1) {
    target.dispatchEvent(
        new MouseEvent(type, {
            bubbles: true,
            button: 0,
            buttons,
            clientX: x,
            clientY: y,
        })
    );
    flushSync();
}

function swatch() {
    const onclick = vi.fn();
    const rendered = render(ColorSwatch, {
        color: '#123456',
        index: 1,
        label: 'Red',
        locked: false,
        selected: false,
        focused: true,
        onclick,
    });
    const element = rendered.target.querySelector<HTMLElement>(
        '[data-swatch-idx="1"]'
    )!;
    return {...rendered, element, onclick};
}

function beginDrag(element: HTMLElement) {
    mouse(element, 'mousedown');
    mouse(window, 'mousemove', 30, 30);
    expect(ui.getColorDrag()?.color).toBe('#123456');
}

async function overrideTarget() {
    const target = render(AppColorOverrides, {});
    target.target.querySelector<HTMLButtonElement>('button')!.click();
    await settle();
    return target.target.querySelector<HTMLButtonElement>(
        'button[title^="background"]'
    )!;
}

test('a subthreshold movement remains a normal picker click', () => {
    const source = swatch();
    mouse(source.element, 'mousedown');
    mouse(window, 'mousemove', 21, 21);
    expect(ui.getColorDrag()).toBeNull();
    mouse(source.element, 'mouseup', 21, 21, 0);
    source.element.click();
    expect(source.onclick).toHaveBeenCalledTimes(1);
});

test('a drop creates one reversible Omarchy app override', async () => {
    theme.setAppOverride('kitty', 'background', '#abcdef');
    const before = theme.getHistorySnapshot();
    const source = swatch();
    const role = await overrideTarget();
    beginDrag(source.element);
    mouse(role, 'mouseup', 50, 50, 0);
    expect(theme.getAppOverrides()).toEqual({kitty: {background: '#123456'}});
    expect(ui.getColorDrag()).toBeNull();
    expect(ui.getColorPickerOpen()).toBe(false);
    undoAction();
    expect(theme.getHistorySnapshot()).toEqual(before);
    expect(history.getCanUndo()).toBe(false);
    redoAction();
    expect(theme.getAppOverrides()).toEqual({kitty: {background: '#123456'}});
});

test('the first source click after an outside drop opens the picker', () => {
    const source = swatch();
    beginDrag(source.element);
    mouse(document.body, 'mouseup', 50, 50, 0);
    mouse(source.element, 'mousedown');
    mouse(source.element, 'mouseup', 20, 20, 0);
    source.element.click();
    expect(source.onclick).toHaveBeenCalledTimes(1);
});

test('a drag released over its source does not open the picker', () => {
    const source = swatch();
    beginDrag(source.element);
    mouse(source.element, 'mouseup', 30, 30, 0);
    source.element.click();
    expect(source.onclick).not.toHaveBeenCalled();
});

test.each(['blur', 'escape', 'released'] as const)(
    '%s cancels an abandoned drag',
    async action => {
        const source = swatch();
        const role = await overrideTarget();
        beginDrag(source.element);
        if (action === 'blur') window.dispatchEvent(new Event('blur'));
        if (action === 'escape')
            window.dispatchEvent(new KeyboardEvent('keydown', {key: 'Escape'}));
        if (action === 'released') mouse(window, 'mousemove', 40, 40, 0);
        mouse(role, 'mousedown', 50, 50);
        mouse(role, 'mouseup', 50, 50, 0);
        expect(ui.getColorDrag()).toBeNull();
        expect(theme.getAppOverrides()).toEqual({});
        expect(history.getCanUndo()).toBe(false);
    }
);

test('source destruction clears drag state and restores the cursor', async () => {
    document.body.style.cursor = 'crosshair';
    const source = swatch();
    render(ColorDragGhost, {});
    beginDrag(source.element);
    expect(document.body.style.cursor).toBe('copy');
    await source.destroy();
    flushSync();
    expect(ui.getColorDrag()).toBeNull();
    expect(document.body.style.cursor).toBe('crosshair');
});

test('destroying an unrelated swatch does not cancel the drag', async () => {
    const source = swatch();
    const other = swatch();
    beginDrag(source.element);
    await other.destroy();
    expect(ui.getColorDrag()?.color).toBe('#123456');
});

test('the lock control does not start a drag', () => {
    const source = swatch();
    mouse(
        source.target.querySelector<HTMLElement>('[aria-label="Lock color"]')!,
        'mousedown'
    );
    mouse(window, 'mousemove', 40, 40);
    expect(ui.getColorDrag()).toBeNull();
});
