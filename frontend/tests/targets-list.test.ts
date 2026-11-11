import {beforeEach, expect, test, vi} from 'vitest';
import TemplateToggles from '../src/lib/components/sidebar/TemplateToggles.svelte';
import * as targets from '../src/lib/stores/targets.svelte';
import {ListTargets, SaveSettings} from '../wailsjs/go/main/App';
import {render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    ListTargets: vi.fn(),
    GetTemplateColors: vi.fn().mockResolvedValue({kitty: [], waybar: []}),
    GetSettings: vi.fn().mockResolvedValue({}),
    SaveSettings: vi.fn().mockResolvedValue(undefined),
}));

const LIST = [
    {
        id: 'kitty',
        label: 'Kitty',
        detected: true,
        path: '/h/.config/kitty/a.conf',
    },
    {
        id: 'swaync',
        label: 'SwayNC',
        detected: true,
        path: '/h/.config/swaync/s.css',
    },
    {id: 'btop', label: 'btop', detected: false, path: '/h/.config/btop/t'},
];

async function mountList() {
    const view = render(TemplateToggles, {});
    view.target
        .querySelector<HTMLButtonElement>('button[aria-expanded]')!
        .click();
    await settle();
    return view.target;
}

const row = (t: HTMLElement, id: string) =>
    t.querySelector<HTMLElement>(`[data-target="${id}"]`)!;

beforeEach(() => {
    vi.mocked(ListTargets)
        .mockReset()
        .mockResolvedValue(LIST as never);
    vi.mocked(SaveSettings).mockClear();
    targets.setLastApply(null);
});

test('renders a row per target; undetected rows are disabled and say not found', async () => {
    const t = await mountList();
    expect(t.querySelectorAll('[data-target]')).toHaveLength(3);
    expect(row(t, 'kitty').textContent).toContain('Kitty');
    expect(row(t, 'kitty').getAttribute('title')).toContain('kitty/a.conf');
    const btop = row(t, 'btop');
    expect(btop.textContent).toContain('not found');
    expect(btop.querySelector('button[role="switch"]')).toHaveProperty(
        'disabled',
        true
    );
});

test('a failed apply status shows failed with the error in the title', async () => {
    const t = await mountList();
    targets.setLastApply({
        success: true,
        themePath: '/t',
        targets: [
            {
                id: 'swaync',
                label: 'SwayNC',
                state: 'failed',
                path: '',
                error: 'css broken',
            },
            {id: 'kitty', label: 'Kitty', state: 'reloaded', path: ''},
        ],
    } as never);
    await settle();
    const sw = row(t, 'swaync');
    expect(sw.textContent).toContain('failed');
    expect(sw.getAttribute('title')).toContain('css broken');
    expect(sw.querySelector('.text-destructive')).not.toBeNull();
    expect(row(t, 'kitty').querySelector('.text-success')).not.toBeNull();
});

test('toggling a target off persists includedApps[id]=false explicitly', async () => {
    const t = await mountList();
    row(t, 'kitty')
        .querySelector<HTMLButtonElement>('button[role="switch"]')!
        .click();
    await settle();
    const saved = vi.mocked(SaveSettings).mock.calls.at(-1)![0] as {
        includedApps: Record<string, boolean>;
    };
    expect(saved.includedApps.kitty).toBe(false);
    expect(targets.isTargetEnabled('kitty')).toBe(false);
});

test('target apps are not duplicated in the legacy Apps list', async () => {
    const t = await mountList();
    // GetTemplateColors returns kitty + waybar; only waybar stays legacy.
    expect(t.textContent).toMatch(/Apps\s*1\b/);
    t.querySelectorAll<HTMLButtonElement>('button[aria-expanded]').forEach(
        b => b.getAttribute('aria-expanded') === 'false' && b.click()
    );
    await settle();
    expect(t.querySelectorAll('[aria-label="Toggle Kitty"]')).toHaveLength(1);
});
