import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import ColorAdjustments from '../src/lib/components/sidebar/ColorAdjustments.svelte';
import ActionBar from '../src/lib/components/layout/ActionBar.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import * as history from '../src/lib/stores/history.svelte';
import {undoAction, redoAction} from '../src/lib/actions/themeActions';
import {AdjustPaletteColors} from '../wailsjs/go/main/App';
import {DEFAULT_ADJUSTMENTS} from '../src/lib/types/theme';
import {button, deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    AdjustPaletteColors: vi.fn(),
    GetWallhavenConfig: vi.fn().mockResolvedValue({}),
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
    getOmarchyAvailable: () => false,
    getOmarchyCapabilities: () => ({overrideApps: []}),
}));

const base = Array(16).fill('#123456');
const adjusted = Array(16).fill('#345678');
const ext = {
    accent: '#112233',
    cursor: '#445566',
    selection_foreground: '#000000',
    selection_background: '#ffffff',
};

beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);
    vi.mocked(AdjustPaletteColors).mockReset();
    theme.reset();
    theme.setPalette(base, true);
    theme.setExtendedColors(ext);
});

function input(target: HTMLElement, value: number) {
    const slider = target.querySelector<HTMLInputElement>(
        'input[aria-label="Brightness"]'
    )!;
    slider.value = String(value);
    slider.dispatchEvent(new Event('input', {bubbles: true}));
    flushSync();
}

test('undo/redo from both the action bar and shared actions restores complete, copied baselines', async () => {
    theme.setAdjustments({...DEFAULT_ADJUSTMENTS, brightness: 20});
    theme.setAdjustedPalette(adjusted);
    theme.setAdjustedExtendedColors({...ext, accent: '#778899'});
    const points: [number, number][] = [[0.3, 0.6]];
    theme.setPaletteCurvePoints(points);
    const before = theme.getHistorySnapshot();
    points[0][1] = 0.1;
    theme.setColor(1, '#abcdef');
    const after = theme.getHistorySnapshot();
    const {target} = render(ActionBar, {});

    button(target, 'Undo').click();
    flushSync();
    expect(theme.getHistorySnapshot()).toEqual(before);
    redoAction();
    flushSync();
    expect(theme.getHistorySnapshot()).toEqual(after);
    undoAction();
    flushSync();
    button(target, 'Redo').click();
    flushSync();
    expect(theme.getHistorySnapshot()).toEqual(after);

    // An edit immediately after Undo must start a new history session.
    undoAction();
    theme.setExtendedColor('accent', '#fedcba');
    undoAction();
    expect(theme.getHistorySnapshot()).toEqual(before);
});

test('adjustments after undo use the original palette and semantic baseline, not the displayed result', async () => {
    theme.setAdjustments({...DEFAULT_ADJUSTMENTS, brightness: 20});
    theme.setAdjustedPalette(adjusted);
    theme.setAdjustedExtendedColors({...ext, accent: '#778899'});
    theme.setColor(1, '#abcdef');
    undoAction();
    vi.mocked(AdjustPaletteColors).mockImplementation(async colors => colors);
    const {target} = render(ColorAdjustments, {});
    input(target, 25);
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    expect(AdjustPaletteColors).toHaveBeenNthCalledWith(
        1,
        base,
        expect.objectContaining({brightness: 25})
    );
    expect(AdjustPaletteColors).toHaveBeenNthCalledWith(
        2,
        Object.values(ext),
        expect.objectContaining({brightness: 25})
    );
});

test('palette and semantic adjustments publish together only after both requests succeed', async () => {
    const paletteResult = deferred<string[]>();
    const extResult = deferred<string[]>();
    vi.mocked(AdjustPaletteColors)
        .mockReturnValueOnce(paletteResult.promise)
        .mockReturnValueOnce(extResult.promise);
    const {target} = render(ColorAdjustments, {});
    input(target, 20);
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    paletteResult.resolve(adjusted);
    await settle();
    expect(theme.getPalette()).toEqual(base);
    expect(theme.getExtendedColors()).toEqual(ext);
    extResult.resolve(Object.values(ext).map(() => '#abcdef'));
    await settle();
    expect(theme.getPalette()).toEqual(adjusted);
    expect(theme.getExtendedColors().accent).toBe('#abcdef');
});

test('a failed semantic request cannot leave a partially adjusted palette', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(AdjustPaletteColors)
        .mockResolvedValueOnce(adjusted)
        .mockRejectedValueOnce(new Error('failed'));
    const {target} = render(ColorAdjustments, {});
    input(target, 20);
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    expect(theme.getPalette()).toEqual(base);
    expect(theme.getExtendedColors()).toEqual(ext);
    expect(theme.getAdjustments()).toEqual(DEFAULT_ADJUSTMENTS);
    expect(theme.getIsAdjusting()).toBe(false);
    expect(error).toHaveBeenCalledTimes(1);
});

test('a newer input invalidates older responses before its debounce fires', async () => {
    const old = deferred<string[]>();
    vi.mocked(AdjustPaletteColors).mockImplementation(colors =>
        colors.length === 16 ? old.promise : Promise.resolve(Object.values(ext))
    );
    const {target} = render(ColorAdjustments, {});
    input(target, 10);
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    input(target, 20);
    old.resolve(adjusted);
    await settle();
    expect(theme.getPalette()).toEqual(base);
    vi.mocked(AdjustPaletteColors).mockImplementation(async colors =>
        colors.map(() => '#abcdef')
    );
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    expect(theme.getPalette()).toEqual(Array(16).fill('#abcdef'));
    expect(theme.getAdjustments().brightness).toBe(20);
});

test.each([
    ['undo', () => undoAction()],
    ['new palette', () => theme.setPalette(Array(16).fill('#aaaaaa'))],
    ['color selection', () => theme.toggleColorSelection(2)],
] as const)('%s invalidates in-flight adjustments', async (_, change) => {
    const pending = deferred<string[]>();
    vi.mocked(AdjustPaletteColors).mockImplementation(colors =>
        colors.length === 16
            ? pending.promise
            : Promise.resolve(Object.values(ext))
    );
    const {target} = render(ColorAdjustments, {});
    input(target, 20);
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    change();
    const expected = theme.getHistorySnapshot();
    pending.resolve(adjusted);
    await settle();
    expect(theme.getHistorySnapshot()).toEqual(expected);
});

test('pending nudge undo does not later push stale history or clear redo', async () => {
    const {target} = render(ColorAdjustments, {});
    target
        .querySelector<HTMLButtonElement>(
            'button[aria-label^="Shift hue +15"]'
        )!
        .click();
    undoAction();
    expect(history.getCanRedo()).toBe(true);
    await vi.advanceTimersByTimeAsync(500);
    await settle();
    expect(AdjustPaletteColors).not.toHaveBeenCalled();
    expect(history.getCanRedo()).toBe(true);
    expect(theme.getAdjustments()).toEqual(DEFAULT_ADJUSTMENTS);
});

test('Reset All restores semantic colors and curves as well as the palette, and is undoable', () => {
    theme.setAdjustments({...DEFAULT_ADJUSTMENTS, brightness: 20});
    theme.setAdjustedPalette(adjusted);
    theme.setAdjustedExtendedColors({...ext, accent: '#778899'});
    theme.setPaletteCurvePoints([[0.3, 0.6]]);
    const before = theme.getHistorySnapshot();
    const {target} = render(ColorAdjustments, {});
    button(target, 'Reset All').click();
    expect(theme.getPalette()).toEqual(base);
    expect(theme.getExtendedColors()).toEqual(ext);
    expect(theme.getPaletteCurvePoints()).toEqual([]);
    undoAction();
    expect(theme.getHistorySnapshot()).toEqual(before);
});

test('curves respect locks and color selections, and curve edits have their own undo snapshot', async () => {
    vi.mocked(AdjustPaletteColors).mockImplementation(async colors => colors);
    theme.setLockedColor(0, true);
    theme.toggleColorSelection(1);
    theme.toggleExtColorSelection('accent');
    const {target} = render(ColorAdjustments, {});
    const canvas = target.querySelector('canvas')!;
    vi.spyOn(canvas, 'getBoundingClientRect').mockReturnValue({
        left: 0,
        top: 0,
        width: 256,
        height: 192,
    } as DOMRect);
    canvas.dispatchEvent(
        new MouseEvent('mousedown', {
            button: 0,
            clientX: 128,
            clientY: 30,
            bubbles: true,
        })
    );
    window.dispatchEvent(new MouseEvent('mouseup'));
    flushSync();
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    expect(theme.getPaletteCurvePoints()).toHaveLength(1);
    expect(theme.getPalette()[0]).toBe(base[0]);
    expect(theme.getPalette()[1]).not.toBe(base[1]);
    expect(theme.getPalette()[2]).toBe(base[2]);
    expect(theme.getExtendedColors().accent).not.toBe(ext.accent);
    expect(theme.getExtendedColors().cursor).toBe(ext.cursor);
    undoAction();
    expect(theme.getPaletteCurvePoints()).toEqual([]);
    expect(theme.getPalette()).toEqual(base);
    redoAction();
    expect(theme.getPaletteCurvePoints()).toHaveLength(1);
});

test.each([false, true])(
    'pending adjustments survive destruction and remount (started: %s)',
    async started => {
        const pending = deferred<string[]>();
        vi.mocked(AdjustPaletteColors).mockImplementation(colors =>
            colors.length === 16
                ? pending.promise
                : Promise.resolve(Object.values(ext))
        );
        const {target, destroy} = render(ColorAdjustments, {});
        input(target, 20);
        if (started) {
            await vi.advanceTimersByTimeAsync(75);
            await settle();
        }
        await destroy();
        pending.resolve(adjusted);
        await vi.advanceTimersByTimeAsync(500);
        await settle();
        expect(theme.getPalette()).toEqual(adjusted);
        expect(theme.getAdjustments().brightness).toBe(20);
        expect(theme.getIsAdjusting()).toBe(false);
        expect(AdjustPaletteColors).toHaveBeenCalledTimes(2);
        const remounted = render(ColorAdjustments, {});
        expect(
            remounted.target.querySelector<HTMLInputElement>(
                'input[aria-label="Brightness"]'
            )!.value
        ).toBe('20');
    }
);

test.each([
    ['light mode', () => theme.setLightMode(true)],
    ['wallpaper', () => theme.setWallpaperPath('/new.png')],
] as const)(
    '%s does not invalidate unrelated adjustment inputs',
    async (_, change) => {
        const pending = deferred<string[]>();
        vi.mocked(AdjustPaletteColors).mockImplementation(colors =>
            colors.length === 16
                ? pending.promise
                : Promise.resolve(Object.values(ext))
        );
        const {target} = render(ColorAdjustments, {});
        input(target, 20);
        await vi.advanceTimersByTimeAsync(75);
        await settle();
        change();
        pending.resolve(adjusted);
        await settle();
        expect(theme.getPalette()).toEqual(adjusted);
        expect(theme.getAdjustments().brightness).toBe(20);
    }
);

test.each([false, true])(
    'Undo then Redo resumes unfinished adjustment colors (started: %s)',
    async started => {
        const old = deferred<string[]>();
        vi.mocked(AdjustPaletteColors).mockImplementation(colors =>
            colors.length === 16
                ? old.promise
                : Promise.resolve(Object.values(ext))
        );
        const {target, destroy} = render(ColorAdjustments, {});
        input(target, 20);
        if (started) {
            await vi.advanceTimersByTimeAsync(75);
            await settle();
        }
        undoAction();
        expect(theme.getPalette()).toEqual(base);
        expect(theme.getAdjustments()).toEqual(DEFAULT_ADJUSTMENTS);
        await destroy();
        vi.mocked(AdjustPaletteColors).mockImplementation(async (colors, adj) =>
            colors.map(() => (adj.brightness === 20 ? '#456789' : '#000000'))
        );
        redoAction();
        expect(theme.getIsAdjusting()).toBe(true);
        await vi.advanceTimersByTimeAsync(75);
        await settle();
        old.resolve(adjusted);
        await settle();
        expect(theme.getAdjustments().brightness).toBe(20);
        expect(theme.getPalette()).toEqual(Array(16).fill('#456789'));
        expect(theme.getExtendedColors().accent).toBe('#456789');
        expect(theme.getBasePalette()).toEqual(base);
        expect(theme.getIsAdjusting()).toBe(false);
    }
);

test('a failed replacement adjustment restores the last successful metadata, not defaults', async () => {
    vi.mocked(AdjustPaletteColors).mockImplementation(async colors =>
        colors.map(() => '#345678')
    );
    const {target} = render(ColorAdjustments, {});
    input(target, 10);
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(AdjustPaletteColors).mockRejectedValue(new Error('offline'));
    input(target, 20);
    await vi.advanceTimersByTimeAsync(75);
    await settle();
    expect(theme.getPalette()).toEqual(adjusted);
    expect(theme.getAdjustments().brightness).toBe(10);
    expect(theme.getIsAdjusting()).toBe(false);
});

test('settled history does not recalculate over manual colors', async () => {
    theme.setAdjustments({...DEFAULT_ADJUSTMENTS, brightness: 20});
    theme.setAdjustedPalette(adjusted);
    theme.setColor(1, '#abcdef');
    const manual = theme.getHistorySnapshot();
    undoAction();
    redoAction();
    await vi.advanceTimersByTimeAsync(500);
    await settle();
    expect(AdjustPaletteColors).not.toHaveBeenCalled();
    expect(theme.getHistorySnapshot()).toEqual(manual);
});
