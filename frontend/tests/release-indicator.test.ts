import {beforeEach, expect, test, vi} from 'vitest';
import ReleaseIndicator from '../src/lib/components/layout/ReleaseIndicator.svelte';
import {GetReleaseStatus, StartUpgrade} from '../wailsjs/go/main/App';
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

test('managed installs hide the indicator and never start an upgrade', async () => {
    const {target} = render(ReleaseIndicator, {});
    await settle();
    expect(target.querySelector('button')).toBeNull();
    expect(StartUpgrade).not.toHaveBeenCalled();
    expect(GetReleaseStatus).toHaveBeenCalledTimes(1);
});

test('standalone installs keep the indicator', async () => {
    vi.mocked(GetReleaseStatus).mockResolvedValue({
        currentVersion: '4.30.0',
        latestVersion: '4.30.0',
        releaseURL: '',
        updateAvailable: false,
        updateCommand: '',
    });
    const {target} = render(ReleaseIndicator, {});
    await settle();
    expect(target.querySelector('button')?.title).toContain('up to date');
});
