import {beforeEach, expect, test, vi} from 'vitest';
import {flushSync} from 'svelte';
import Harness from './CommandPaletteHarness.svelte';
import {ListBlueprints, ApplyTheme} from '../wailsjs/go/main/App';
import * as theme from '../src/lib/stores/theme.svelte';
import * as ui from '../src/lib/stores/ui.svelte';
import {DEFAULT_ADJUSTMENTS, DEFAULT_PALETTE} from '../src/lib/types/theme';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import {buildCommands} from '../src/lib/commands/commands.svelte';
import {button, deferred, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    ListBlueprints: vi.fn(),
    ApplyTheme: vi.fn(),
}));

const blueprint = {
    name: 'Midnight Forest',
    timestamp: 100,
    palette: {
        colors: [...DEFAULT_PALETTE],
        wallpaper: '/forest.png',
        extendedColors: {accent: '#abcdef'},
        nativeColors: {orange: '#ed9121'},
        lockedColors: [1, 7],
        mode: 'light' as const,
    },
    adjustments: {brightness: 25},
    appOverrides: {kitty: {background: '#123456'}},
    iconTheme: {mode: 'explicit' as const, id: 'Forest-Icons'},
};

beforeEach(() => {
    theme.reset();
    ui.setLiveApply(false);
    while (ui.getToastQueueDepth()) ui.dismissCurrentToast();
    vi.mocked(ListBlueprints).mockReset().mockResolvedValue([]);
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
        configurable: true,
        value: vi.fn(),
    });
});

function open(target: HTMLElement) {
    const trigger = button(target, 'Open commands');
    trigger.focus();
    trigger.click();
    flushSync();
    return trigger;
}

function search(target: HTMLElement, query: string) {
    const input = target.querySelector('input')!;
    input.value = query;
    input.dispatchEvent(new Event('input', {bubbles: true}));
    flushSync();
    return input;
}

function key(input: HTMLElement, key: string, extra = {}) {
    const event = new KeyboardEvent('keydown', {
        key,
        bubbles: true,
        cancelable: true,
        ...extra,
    });
    input.dispatchEvent(event);
    flushSync();
    return event;
}

test('keyboard search loads a saved blueprint into the editor without applying it', async () => {
    vi.mocked(ListBlueprints).mockResolvedValue([blueprint]);
    ui.setLiveApply(true);
    ui.setLivePending(true);
    ui.setActiveTab('local');
    const {target} = render(Harness, {commands: buildCommands()});
    const trigger = open(target);
    await settle();
    const input = search(target, 'midnight');
    expect(document.activeElement).toBe(input);
    expect(target.querySelectorAll('[role="option"]')).toHaveLength(1);
    expect(
        document
            .getElementById(input.getAttribute('aria-activedescendant')!)
            ?.getAttribute('aria-selected')
    ).toBe('true');
    expect(target.textContent).toContain('Load into editor');
    key(input, 'Enter');
    await settle();
    expect(target.querySelector('[role="dialog"]')).toBeNull();
    expect(document.activeElement).toBe(trigger);
    expect(theme.getPalette()).toEqual(blueprint.palette.colors);
    expect(theme.getBasePalette()).toEqual(blueprint.palette.colors);
    expect(theme.getAdjustments()).toEqual(DEFAULT_ADJUSTMENTS);
    expect(theme.getExtendedColors().accent).toBe('#abcdef');
    expect(theme.getNativeColors()).toEqual(blueprint.palette.nativeColors);
    expect(theme.getAppOverrides()).toEqual(blueprint.appOverrides);
    expect(theme.getIconTheme()).toEqual(blueprint.iconTheme);
    expect(theme.getWallpaperPath()).toBe('/forest.png');
    expect(theme.getLockedColors()[1]).toBe(true);
    expect(theme.getLockedColors()[2]).toBe(false);
    expect(theme.getLightMode()).toBe(true);
    expect(ui.getActiveTab()).toBe('editor');
    expect(ui.getLiveApply()).toBe(false);
    expect(ui.getLivePending()).toBe(false);
    expect(ApplyTheme).not.toHaveBeenCalled();
});

test('focus stays in the combobox, navigation wraps, and Escape restores the trigger', async () => {
    const {target} = render(Harness, {
        commands: [
            {id: 'first', label: 'First', category: 'Test', run: vi.fn()},
            {id: 'last', label: 'Last', category: 'Test', run: vi.fn()},
        ],
    });
    const trigger = open(target);
    await settle();
    const input = target.querySelector('input')!;
    key(input, 'Tab');
    expect(document.activeElement).toBe(input);
    key(input, 'Tab', {shiftKey: true});
    expect(document.activeElement).toBe(input);
    key(input, 'ArrowUp');
    expect(
        target.querySelector('[aria-selected="true"]')?.textContent
    ).toContain('Last');
    key(input, 'ArrowDown');
    expect(
        target.querySelector('[aria-selected="true"]')?.textContent
    ).toContain('First');
    // Home/End without modifiers retain normal text-editing behavior.
    expect(key(input, 'Home').defaultPrevented).toBe(false);
    search(target, 'no such command');
    key(input, 'ArrowDown');
    expect(input.hasAttribute('aria-activedescendant')).toBe(false);
    key(input, 'Escape');
    await settle();
    expect(document.activeElement).toBe(trigger);
    expect(target.querySelector('[role="dialog"]')).toBeNull();
});

test('an old blueprint request cannot replace results after closing and reopening', async () => {
    const first = deferred<Record<string, unknown>[]>();
    const second = deferred<Record<string, unknown>[]>();
    vi.mocked(ListBlueprints)
        .mockReturnValueOnce(first.promise)
        .mockReturnValueOnce(second.promise);
    const {target} = render(Harness, {});
    open(target);
    await settle();
    key(target.querySelector('input')!, 'Escape');
    open(target);
    await settle();
    second.resolve([{...blueprint, name: 'Current'}]);
    await settle();
    first.resolve([{...blueprint, name: 'Stale'}]);
    await settle();
    expect(target.textContent).toContain('Current');
    expect(target.textContent).not.toContain('Stale');
});

test('blueprint failure leaves commands usable and reports async command failures', async () => {
    vi.mocked(ListBlueprints).mockRejectedValue(new Error('offline'));
    const run = vi.fn().mockRejectedValue(new Error('Could not run action'));
    const {target} = render(Harness, {
        commands: [{id: 'test', label: 'Test', category: 'Test', run}],
    });
    open(target);
    await settle();
    expect(target.textContent).toContain('Blueprints unavailable');
    key(target.querySelector('input')!, 'Enter');
    await settle();
    expect(run).toHaveBeenCalledTimes(1);
    expect(ui.getToastMessage()).toBe('Could not run action');
});

test('disabled commands explain why they are unavailable and cannot run', async () => {
    const run = vi.fn();
    const {target} = render(Harness, {
        commands: [
            {
                id: 'test',
                label: 'Undo',
                category: 'Edit',
                run,
                disabled: () => 'Nothing to undo',
            },
        ],
    });
    open(target);
    await settle();
    key(target.querySelector('input')!, 'Enter');
    await settle();
    expect(run).not.toHaveBeenCalled();
    expect(target.textContent).toContain('Nothing to undo');
    expect(target.querySelector('[role="dialog"]')).not.toBeNull();
});

test('invalid blueprints cannot mutate the current theme or disable Live Apply', () => {
    ui.setLiveApply(true);
    expect(() =>
        loadBlueprintIntoEditor({...blueprint, palette: {colors: ['invalid']}})
    ).toThrow('16 valid hex colors');
    expect(theme.getWallpaperPath()).toBe('');
    expect(theme.getPalette()).toEqual(DEFAULT_PALETTE);
    expect(ui.getLiveApply()).toBe(true);
});

test('reset-adjustments command restores the base and remains undoable', async () => {
    theme.setAdjustedPalette(Array(16).fill('#abcdef'));
    theme.setAdjustments({...DEFAULT_ADJUSTMENTS, brightness: 20});
    theme.setPaletteCurvePoints([
        [0, 0],
        [128, 100],
        [255, 255],
    ]);
    const commands = buildCommands();
    await commands.find(c => c.id === 'theme.resetAdjustments')!.run();
    expect(theme.getPalette()).toEqual(DEFAULT_PALETTE);
    expect(theme.getAdjustments()).toEqual(DEFAULT_ADJUSTMENTS);
    expect(theme.getPaletteCurvePoints()).toEqual([]);
    await commands.find(c => c.id === 'edit.undo')!.run();
    expect(theme.getPalette()).toEqual(Array(16).fill('#abcdef'));
    expect(theme.getAdjustments().brightness).toBe(20);
    expect(theme.getPaletteCurvePoints()).toHaveLength(3);
});

test('reset-adjustments command cancels unfinished calculations', async () => {
    vi.useFakeTimers();
    theme.adjustPalette({...DEFAULT_ADJUSTMENTS, brightness: 25});
    expect(theme.getIsAdjusting()).toBe(true);
    await buildCommands()
        .find(c => c.id === 'theme.resetAdjustments')!
        .run();
    expect(theme.getIsAdjusting()).toBe(false);
    expect(theme.getAdjustments()).toEqual(DEFAULT_ADJUSTMENTS);
    await vi.advanceTimersByTimeAsync(100);
    expect(theme.getPalette()).toEqual(DEFAULT_PALETTE);
});
