import {beforeEach, expect, test, vi} from 'vitest';
import GitHubCard from '../src/lib/components/github/GitHubCard.svelte';
import GitHubBrowser from '../src/lib/components/github/GitHubBrowser.svelte';
import HeaderBar from '../src/lib/components/layout/HeaderBar.svelte';
import * as source from '../src/lib/stores/github.svelte';
import * as favorites from '../src/lib/stores/favorites.svelte';
import * as theme from '../src/lib/stores/theme.svelte';
import {getActiveTab, setActiveTab} from '../src/lib/stores/ui.svelte';
import {
    ListGitHubImages,
    GetGitHubThumbnail,
    DownloadWallpaper,
    ToggleFavorite,
    GetFavorites,
} from '../wailsjs/go/main/App';
import {deferred, render, settle} from './setup';

vi.mock('../src/lib/components/layout/ReleaseIndicator.svelte', async () => ({
    default: (await import('./Empty.svelte')).default,
}));
vi.mock('../src/lib/stores/omarchy.svelte', () => ({
    getOmarchyAvailable: () => true,
    getOmarchyCapabilities: () => ({overrideApps: ['kitty']}),
    initOmarchyCapabilities: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('../wailsjs/go/main/App', () => ({
    ListGitHubImages: vi.fn(),
    GetGitHubThumbnail: vi.fn(),
    DownloadWallpaper: vi.fn(),
    ToggleFavorite: vi.fn(),
    GetFavorites: vi.fn().mockResolvedValue([]),
    GetWallpaperTags: vi.fn().mockResolvedValue({labels: [], assignments: {}}),
    IsMacOS: vi.fn().mockResolvedValue(false),
}));

const image = {
    name: 'wall.png',
    path: 'wall.png',
    type: 'file',
    size: 1024,
    url: 'https://raw.githubusercontent.com/owner/repo/main/wall.png',
};
const thumbnail = {
    dataURL: 'data:image/png;base64,AA==',
    width: 20,
    height: 10,
};

beforeEach(async () => {
    await settle();
    source.setURL('');
    theme.reset();
    setActiveTab('github');
    vi.mocked(ListGitHubImages).mockReset().mockResolvedValue({items: []});
    vi.mocked(GetGitHubThumbnail).mockReset().mockResolvedValue(thumbnail);
    vi.mocked(DownloadWallpaper)
        .mockReset()
        .mockResolvedValue('/downloaded.png');
    vi.mocked(ToggleFavorite).mockReset().mockResolvedValue(true);
    vi.mocked(GetFavorites).mockReset().mockResolvedValue([]);
    await favorites.refreshFavorites();
    vi.stubGlobal(
        'IntersectionObserver',
        class {
            constructor(
                private callback: (entries: {isIntersecting: boolean}[]) => void
            ) {}
            observe() {
                this.callback([{isIntersecting: true}]);
            }
            disconnect() {}
        }
    );
});

test('the newest repository response wins', async () => {
    const old = deferred<{items: (typeof image)[]}>();
    const next = deferred<{items: (typeof image)[]}>();
    vi.mocked(ListGitHubImages)
        .mockReturnValueOnce(old.promise)
        .mockReturnValueOnce(next.promise);
    source.setURL('https://github.com/owner/old');
    const first = source.fetchImages();
    await settle();
    source.setURL('https://github.com/owner/new');
    const second = source.fetchImages();
    await settle();
    next.resolve({items: [{...image, name: 'new.png'}]});
    await second;
    old.resolve({items: [image]});
    await first;
    expect(source.getURL()).toBe('https://github.com/owner/new');
    expect(source.getResults()[0].name).toBe('new.png');
});

test('directory navigation escapes names and stops at the repository root', async () => {
    source.setURL('https://github.com/owner/repo/tree/main?tab=readme#readme');
    expect(source.getCanGoUp()).toBe(false);
    source.navigateToDir('a #b');
    await settle();
    expect(source.getURL()).toBe(
        'https://github.com/owner/repo/tree/main/a%20%23b'
    );
    source.goUp();
    await settle();
    expect(source.getCanGoUp()).toBe(false);
    expect(source.getURL()).toBe('https://github.com/owner/repo/tree/main');
});

test('failed thumbnails expose a retry', async () => {
    vi.mocked(GetGitHubThumbnail).mockRejectedValueOnce(new Error('offline'));
    const {target} = render(GitHubCard, {image, onpreview: vi.fn()});
    await settle();
    const retry = [...target.querySelectorAll('button')].find(
        button => button.textContent === 'Retry preview'
    )!;
    expect(retry).not.toBeUndefined();
    retry.click();
    await settle();
    expect(GetGitHubThumbnail).toHaveBeenCalledTimes(2);
    expect(target.querySelector('img')?.getAttribute('src')).toBe(
        thumbnail.dataURL
    );
});

test('repository hearts use the shared favorites store', async () => {
    const {target} = render(GitHubCard, {image, onpreview: vi.fn()});
    await settle();
    target
        .querySelector<HTMLButtonElement>('[aria-label="Add to favorites"]')!
        .click();
    await settle();
    expect(ToggleFavorite).toHaveBeenCalledExactlyOnceWith(
        image.url,
        'github',
        {name: image.name}
    );
    expect(favorites.isFavorite(image.url)).toBe(true);
});

test('a late download does not replace a newer wallpaper choice', async () => {
    const pending = deferred<string>();
    vi.mocked(DownloadWallpaper).mockReturnValue(pending.promise);
    const {target} = render(GitHubCard, {image, onpreview: vi.fn()});
    await settle();
    target
        .querySelector<HTMLButtonElement>(
            'button[title="Set as wallpaper and open in editor"]'
        )!
        .click();
    await settle();
    theme.setWallpaperPath('/newer.png');
    pending.resolve('/old.png');
    await settle();
    expect(theme.getWallpaperPath()).toBe('/newer.png');
    expect(getActiveTab()).toBe('github');
});

test('GitHub navigation remains a standard focusable button', async () => {
    const {target} = render(HeaderBar, {});
    await settle();
    setActiveTab('editor');
    const button = [...target.querySelectorAll('nav button')].find(
        button => button.textContent?.trim() === 'GitHub'
    ) as HTMLButtonElement;
    button.focus();
    button.click();
    await settle();
    expect(document.activeElement).toBe(button);
    expect(getActiveTab()).toBe('github');
    expect(button.getAttribute('aria-current')).toBe('page');
});

test('malformed saved repository preferences do not break the browser', () => {
    localStorage.setItem('aether-saved-repos', '{"invalid":true}');
    const {target} = render(GitHubBrowser, {});
    expect(
        target.querySelector('[aria-label="GitHub repository URL"]')
    ).not.toBeNull();
});

const folder = (name: string, htmlURL = '') => ({
    name,
    path: name,
    type: 'dir',
    size: 0,
    url: '',
    htmlURL,
});

function openFolder(target: HTMLElement, name: string) {
    const button = target.querySelector<HTMLButtonElement>(
        `[aria-label="Open directory ${name}"]`
    );
    if (!button) throw new Error(`Missing folder: ${name}`);
    button.click();
}

test('folders with spaces, # and other reserved characters open and close', async () => {
    const names = ['My Folder', 'C#', 'a&b?c', '100%', 'æøå'];
    vi.mocked(ListGitHubImages).mockResolvedValue({
        items: names.map(name => folder(name)),
    } as never);
    const {target} = render(GitHubBrowser, {});
    source.setURL('https://github.com/owner/repo/tree/main');
    await source.fetchImages();
    await settle();

    for (const name of names) {
        openFolder(target, name);
        await settle();
    }
    const deepest =
        'https://github.com/owner/repo/tree/main/My%20Folder/C%23/a%26b%3Fc/100%25/%C3%A6%C3%B8%C3%A5';
    expect(source.getURL()).toBe(deepest);
    expect(vi.mocked(ListGitHubImages).mock.calls.at(-1)?.[0]).toBe(deepest);

    for (const _ of names) {
        target
            .querySelector<HTMLButtonElement>(
                '[aria-label="Go to parent directory"]'
            )!
            .click();
        await settle();
    }
    expect(source.getURL()).toBe('https://github.com/owner/repo/tree/main');
    expect(source.getCanGoUp()).toBe(false);
});

test('a folder named tree at a repository root opens through its page URL', async () => {
    source.setURL('https://github.com/owner/repo');
    source.navigateToDir(
        'tree',
        'https://github.com/owner/repo/tree/main/tree'
    );
    await settle();
    expect(source.getURL()).toBe(
        'https://github.com/owner/repo/tree/main/tree'
    );
    expect(source.getCanGoUp()).toBe(true);
    source.navigateToDir('blob', 'https://github.com/other/repo/tree/x/blob');
    await settle();
    expect(source.getURL()).toBe(
        'https://github.com/owner/repo/tree/main/tree/blob'
    );
});

test('a typed percent sign that starts no escape is escaped', () => {
    source.setURL(' https://github.com/owner/repo/tree/main/100%/a%20b ');
    expect(source.getURL()).toBe(
        'https://github.com/owner/repo/tree/main/100%25/a%20b'
    );
});

test('a new folder starts at the top without the previous filter', async () => {
    vi.mocked(ListGitHubImages).mockResolvedValue({
        items: [folder('My Folder'), image],
    } as never);
    const {target} = render(GitHubBrowser, {});
    source.setURL('https://github.com/owner/repo/tree/main');
    await source.fetchImages();
    await settle();

    const scroller = target.querySelector<HTMLElement>('.overflow-y-auto')!;
    let offset = 640;
    Object.defineProperty(scroller, 'scrollTop', {
        configurable: true,
        get: () => offset,
        set: value => (offset = value),
    });
    const filter = target.querySelector<HTMLInputElement>(
        '[aria-label="Filter repository files"]'
    )!;
    filter.value = 'my';
    filter.dispatchEvent(new Event('input', {bubbles: true}));
    await settle();

    openFolder(target, 'My Folder');
    await settle();
    expect(offset).toBe(0);
    expect(
        target.querySelector<HTMLInputElement>(
            '[aria-label="Filter repository files"]'
        )?.value
    ).toBe('');
});
