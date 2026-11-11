import {beforeEach, expect, test, vi} from 'vitest';
import YozakuraPanel from '../src/lib/components/sidebar/YozakuraPanel.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import * as yoz from '../src/lib/stores/yozakura.svelte';
import {captureApplyRequest} from '../src/lib/actions/themeActions';
import {GetYozakuraRoles} from '../wailsjs/go/main/App';
import {button, render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    GetYozakuraRoles: vi.fn(),
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
}));

const ROLES: Record<string, string> = Object.fromEntries(
    yoz.YOZAKURA_ROLE_KEYS.map((k, i) => [k, `#${String(i + 10).repeat(3)}`])
);

function backendResult(overrides: Record<string, string> = {}) {
    return {
        roles: {...ROLES, ...overrides},
        explicit: Object.fromEntries(
            Object.keys(overrides).map(k => [k, true])
        ),
        glass: 0.92,
        clock: {digits: 'bold', colon: ':', glow: 0.35, ornament: 'arc'},
        contrast: [
            {pair: 'text/surface', ratio: 9.1, min: 7, state: 'ok'},
            {pair: 'muted/surface', ratio: 4.6, min: 4.5, state: 'nudged'},
            {pair: 'accent/surface', ratio: 3.1, min: 4.5, state: 'failed'},
        ],
        accentCandidates: ['#aa0000', '#00aa00', '#0000aa'],
    };
}

const wait = (ms = 160) => new Promise(r => setTimeout(r, ms));

async function mountPanel() {
    const view = render(YozakuraPanel, {});
    button(view.target, 'Yozakura').click();
    await settle();
    await wait();
    await settle();
    return view;
}

beforeEach(async () => {
    await settle();
    theme.reset();
    yoz.resetYozakuraResult();
    vi.mocked(GetYozakuraRoles)
        .mockReset()
        .mockImplementation(async req =>
            backendResult(
                (req as unknown as {overrides: Record<string, string>})
                    .overrides
            )
        );
});

test('pinning a role sends it in overrides and drops the auto mark; reset removes it', async () => {
    const {target} = await mountPanel();
    const swatch = () => target.querySelector('[data-role="accent"]')!;
    expect(swatch().textContent).toContain('auto');

    yoz.setYozakuraRole('accent', '#ff00aa');
    await wait();
    await settle();
    const request = vi.mocked(GetYozakuraRoles).mock
        .lastCall![0] as unknown as {
        overrides: Record<string, string>;
    };
    expect(request.overrides).toEqual({accent: '#ff00aa'});
    expect(swatch().textContent).not.toContain('auto');

    button(target, 'Reset accent').click();
    await wait();
    await settle();
    expect(yoz.getYozakuraOverrides()).toEqual({});
    expect(swatch().textContent).toContain('auto');
});

test('cycleAccent walks accentCandidates and wraps', async () => {
    await mountPanel();
    yoz.cycleAccent();
    expect(yoz.getYozakuraOverrides().accent).toBe('#aa0000');
    yoz.cycleAccent();
    yoz.cycleAccent();
    expect(yoz.getYozakuraOverrides().accent).toBe('#0000aa');
    yoz.cycleAccent();
    expect(yoz.getYozakuraOverrides().accent).toBe('#aa0000');
});

test('glass clamps to 0.80..1.00 in the store and the slider', async () => {
    const {target} = await mountPanel();
    yoz.setGlass(0.5);
    expect(yoz.getGlass()).toBe(0.8);
    yoz.setGlass(1.4);
    expect(yoz.getGlass()).toBe(1);
    yoz.setGlass(0.9);
    expect(yoz.getGlass()).toBe(0.9);
    const slider = target.querySelector<HTMLInputElement>(
        'input[aria-label="Glass"]'
    )!;
    expect(slider.min).toBe('0.8');
    expect(slider.max).toBe('1');
});

test('contrast badges show state and ratio', async () => {
    const {target} = await mountPanel();
    const text = target.textContent!;
    expect(text).toContain('nudged');
    expect(text).toContain('failed');
    expect(text).toContain('4.60:1');
    expect(text).toContain('3.10:1');
});

test('apply request carries yozakuraRoles, glass and clock', () => {
    yoz.setYozakuraRole('accent', '#ff00aa');
    yoz.setGlass(0.85);
    yoz.setClock({digits: 'thin', colon: '·', glow: 0.5, ornament: 'seal'});
    const request = captureApplyRequest();
    expect(request.yozakuraRoles).toEqual({accent: '#ff00aa'});
    expect(request.glass).toBe(0.85);
    expect(request.clock).toEqual({
        digits: 'thin',
        colon: '·',
        glow: 0.5,
        ornament: 'seal',
    });
});
