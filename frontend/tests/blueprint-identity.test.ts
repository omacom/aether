import {beforeEach, expect, test, vi} from 'vitest';
import {applyTheme} from '../src/lib/actions/themeActions';
import {loadBlueprintIntoEditor} from '../src/lib/actions/blueprintActions';
import * as theme from '../src/lib/stores/theme.svelte';
import {setLiveApply} from '../src/lib/stores/ui.svelte';
import {ApplyTheme} from '../wailsjs/go/main/App';
import {settle} from './setup';
import type {Blueprint} from '../src/lib/types/theme';

vi.mock('../wailsjs/go/main/App', () => ({
    ApplyTheme: vi.fn(),
    SaveAndApplyTheme: vi.fn(),
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
}));

const colors = Array.from({length: 16}, (_, i) => `#${i.toString(16)}0a0b0`);
const blueprint = {
    name: 'sumi',
    displayName: 'Sumi Ink',
    palette: {
        colors,
        glass: 0.4,
        clock: {digits: 'a', colon: 'b', glow: 1, ornament: 'c'},
    },
    matchWallpapers: ['sumi', 'ink'],
    timestamp: 0,
} as unknown as Blueprint;

beforeEach(async () => {
    await settle();
    theme.reset();
    setLiveApply(false);
    theme.setIsApplying(false);
    vi.mocked(ApplyTheme)
        .mockReset()
        .mockResolvedValue({success: true, themePath: '/t'} as never);
});

test('extraction clears the previous blueprint match patterns', () => {
    loadBlueprintIntoEditor(blueprint);
    expect(theme.getMatchWallpapers()).toEqual(['sumi', 'ink']);
    theme.setPaletteFromExtraction('/new.png', colors);
    expect(theme.getMatchWallpapers()).toEqual([]);
});

test('reset clears glass, clock and match patterns', () => {
    loadBlueprintIntoEditor(blueprint);
    expect(theme.getGlass()).toBe(0.4);
    theme.reset();
    expect(theme.getMatchWallpapers()).toEqual([]);
    expect(theme.getGlass()).toBeNull();
    expect(theme.getClock()).toBeNull();
});

test('apply carries the loaded blueprint name and id, cleared by extraction', async () => {
    loadBlueprintIntoEditor(blueprint);
    await applyTheme();
    expect(ApplyTheme).toHaveBeenLastCalledWith(
        expect.objectContaining({themeName: 'Sumi Ink', themeID: 'sumi'})
    );
    theme.setPaletteFromExtraction('/new.png', colors);
    await applyTheme();
    expect(ApplyTheme).toHaveBeenLastCalledWith(
        expect.objectContaining({themeName: '', themeID: ''})
    );
});

test('reset clears the loaded blueprint identity', async () => {
    loadBlueprintIntoEditor(blueprint);
    theme.reset();
    await applyTheme();
    expect(ApplyTheme).toHaveBeenLastCalledWith(
        expect.objectContaining({themeName: '', themeID: ''})
    );
});
