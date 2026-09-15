import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import ExtractionModeSelect from '../src/lib/components/sidebar/ExtractionModeSelect.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {extractColors, undoAction} from '../src/lib/actions/themeActions';
import {
    ExtractColors,
    ExtractColorsFromImages,
    PreviewExtractColors,
    SetExtractionMode,
} from '../wailsjs/go/main/App';
import {
    EXTRACTION_MODES,
    EXTRACTION_MODE_GROUPS,
} from '../src/lib/constants/colors';
import {deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    ExtractColors: vi.fn(),
    ExtractColorsFromImages: vi.fn(),
    SetExtractionMode: vi.fn(),
    PreviewExtractColors: vi.fn(),
}));

beforeEach(() => {
    theme.reset();
    theme.setIsExtracting(false);
    theme.setWallpaperPath('/original.png');
    vi.mocked(ExtractColors).mockReset();
    vi.mocked(ExtractColorsFromImages).mockReset();
    vi.mocked(SetExtractionMode).mockReset().mockResolvedValue();
    vi.mocked(PreviewExtractColors).mockReset().mockResolvedValue([]);
});

function selectMode(target: HTMLElement, mode: string) {
    const definition = EXTRACTION_MODES.find(item => item.value === mode)!;
    const findMode = () =>
        [...target.querySelectorAll('button')].find(
            el => el.title === definition.description
        );
    if (!findMode()) {
        const group = EXTRACTION_MODE_GROUPS.find(
            item => item.id === definition.group
        )!;
        [...target.querySelectorAll('button')]
            .find(el => el.textContent?.trim().startsWith(group.label))!
            .click();
        flushSync();
    }
    findMode()!.click();
    flushSync();
}

test('rapid mode changes keep the newest result and busy state, regardless of completion order', async () => {
    const first = deferred<string[]>();
    const second = deferred<string[]>();
    vi.mocked(ExtractColors)
        .mockReturnValueOnce(first.promise)
        .mockReturnValueOnce(second.promise);
    const {target} = render(ExtractionModeSelect, {});
    const modes = EXTRACTION_MODES.filter(
        mode => mode.value !== 'normal'
    ).slice(0, 2);
    selectMode(target, modes[0].value);
    await settle();
    selectMode(target, modes[1].value);
    await settle();
    first.resolve(Array(16).fill('#111111'));
    await settle();
    expect(theme.getIsExtracting()).toBe(true);
    expect(theme.getPalette()).not.toEqual(Array(16).fill('#111111'));
    second.resolve(Array(16).fill('#222222'));
    await settle();
    expect(theme.getIsExtracting()).toBe(false);
    expect(theme.getPalette()).toEqual(Array(16).fill('#222222'));
    expect(theme.getExtractionMode()).toBe(modes[1].value);
});

test('out-of-order mode persistence cannot start a stale extraction', async () => {
    const oldMode = deferred<void>();
    vi.mocked(SetExtractionMode)
        .mockReturnValueOnce(oldMode.promise)
        .mockResolvedValueOnce();
    vi.mocked(ExtractColors).mockResolvedValue(Array(16).fill('#222222'));
    const {target} = render(ExtractionModeSelect, {});
    const modes = EXTRACTION_MODES.filter(
        mode => mode.value !== 'normal'
    ).slice(0, 2);
    selectMode(target, modes[0].value);
    await settle();
    selectMode(target, modes[1].value);
    await settle();
    oldMode.resolve();
    await settle();
    expect(ExtractColors).toHaveBeenCalledExactlyOnceWith(
        '/original.png',
        false,
        modes[1].value
    );
});

test('an older extraction cannot overwrite a newer completed mode', async () => {
    const older = deferred<string[]>();
    vi.mocked(ExtractColors)
        .mockReturnValueOnce(older.promise)
        .mockResolvedValueOnce(Array(16).fill('#222222'));
    const {target} = render(ExtractionModeSelect, {});
    const modes = EXTRACTION_MODES.filter(
        mode => mode.value !== 'normal'
    ).slice(0, 2);
    selectMode(target, modes[0].value);
    await settle();
    selectMode(target, modes[1].value);
    await settle();
    const expected = theme.getHistorySnapshot();
    older.resolve(Array(16).fill('#111111'));
    await settle();
    expect(theme.getPalette()).toEqual(Array(16).fill('#222222'));
    expect(theme.getHistorySnapshot()).toEqual(expected);
    expect(theme.getIsExtracting()).toBe(false);
});

test.each([
    ['wallpaper change', () => theme.setWallpaperPath('/new.png')],
    [
        'undo',
        () => {
            theme.setColor(1, '#123456');
            undoAction();
        },
    ],
    ['light mode', () => theme.setLightMode(true)],
] as const)('%s invalidates a pending extraction', async (_, change) => {
    const pending = deferred<string[]>();
    vi.mocked(ExtractColors).mockReturnValue(pending.promise);
    const request = extractColors();
    await settle();
    change();
    const expected = theme.getHistorySnapshot();
    pending.resolve(Array(16).fill('#111111'));
    await request;
    expect(theme.getHistorySnapshot()).toEqual(expected);
    expect(theme.getIsExtracting()).toBe(false);
});

test('blend requests also discard stale results after the image list changes', async () => {
    const pending = deferred<{palette: string[]; skipped: number}>();
    vi.mocked(ExtractColorsFromImages).mockReturnValue(pending.promise);
    theme.setAdditionalImages(['/second.png']);
    const request = extractColors({allImages: true});
    await settle();
    expect(ExtractColorsFromImages).toHaveBeenCalledExactlyOnceWith(
        ['/original.png', '/second.png'],
        false,
        'normal'
    );
    const before = theme.getHistorySnapshot();
    theme.setAdditionalImages(['/third.png']);
    pending.resolve({palette: Array(16).fill('#333333'), skipped: 0});
    await request;
    expect(theme.getHistorySnapshot()).toEqual(before);
});

test('destroy stops preview prefetch but the intended mode extraction survives', async () => {
    const preview = deferred<string[]>();
    const pending = deferred<string[]>();
    vi.mocked(PreviewExtractColors).mockReturnValue(preview.promise);
    vi.mocked(ExtractColors).mockReturnValue(pending.promise);
    const {target, destroy} = render(ExtractionModeSelect, {});
    const mode = EXTRACTION_MODES.find(mode => mode.value !== 'normal')!.value;
    selectMode(target, mode);
    await settle();
    await destroy();
    preview.resolve(Array(16).fill('#111111'));
    pending.resolve(Array(16).fill('#222222'));
    await settle();
    expect(PreviewExtractColors).toHaveBeenCalledTimes(1);
    expect(theme.getPalette()).toEqual(Array(16).fill('#222222'));
    expect(theme.getExtractionMode()).toBe(mode);
    expect(theme.getPendingExtractionMode()).toBe(null);
    const remounted = render(ExtractionModeSelect, {});
    selectMode(remounted.target, mode);
    await settle();
    expect(ExtractColors).toHaveBeenCalledTimes(1);
});

test('a failed mode request restores the displayed selection without changing palette metadata', async () => {
    vi.mocked(ExtractColors).mockRejectedValue(new Error('offline'));
    const before = theme.getHistorySnapshot();
    const {target} = render(ExtractionModeSelect, {});
    selectMode(
        target,
        EXTRACTION_MODES.find(mode => mode.value !== 'normal')!.value
    );
    expect(theme.getPendingExtractionMode()).not.toBe(null);
    expect(theme.getExtractionMode()).toBe('normal');
    await settle();
    expect(theme.getHistorySnapshot()).toEqual(before);
    expect(theme.getPendingExtractionMode()).toBe(null);
    expect(theme.getIsExtracting()).toBe(false);
});
