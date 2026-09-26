import {beforeEach, expect, test, vi} from 'vitest';
import ReleaseIndicator from '../src/lib/components/layout/ReleaseIndicator.svelte';
import {GetReleaseStatus, StartUpgrade} from '../wailsjs/go/main/App';
import {getToastMessage} from '../src/lib/stores/ui.svelte';
import {render, settle} from './setup';

vi.mock('../wailsjs/go/main/App', () => ({
    GetReleaseStatus: vi.fn(),
    StartUpgrade: vi.fn(),
}));

beforeEach(() => {
    vi.mocked(GetReleaseStatus).mockReset().mockResolvedValue({
        currentVersion: '4.30.0',
        latestVersion: '',
        releaseURL: '',
        updateAvailable: false,
        updateCommand: 'omarchy-update',
    });
    vi.mocked(StartUpgrade).mockReset();
});

test('managed status gives neutral guidance without refresh or upgrade on click', async () => {
    const {target} = render(ReleaseIndicator, {});
    await settle();
    const indicator = target.querySelector('button')!;
    expect(indicator.title).toContain('Run: omarchy-update');
    expect(indicator.title).not.toContain('up to date');
    expect(indicator.querySelector('.bg-accent')).not.toBeNull();
    indicator.click();
    await settle();
    expect(getToastMessage()).toContain('Run: omarchy-update');
    expect(StartUpgrade).not.toHaveBeenCalled();
    expect(GetReleaseStatus).toHaveBeenCalledTimes(1);
});
