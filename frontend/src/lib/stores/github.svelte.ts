import type {githubsource} from '../../../wailsjs/go/models';

let url = $state('');
let results = $state<githubsource.ImageInfo[]>([]);
let isLoading = $state(false);
let error = $state('');
let requestSequence = 0;

export function getURL(): string {
    return url;
}
export function getResults(): githubsource.ImageInfo[] {
    return results;
}
export function getIsLoading(): boolean {
    return isLoading;
}
export function getError(): string {
    return error;
}

export function setURL(value: string): void {
    requestSequence++;
    url = value.trim();
    results = [];
    error = '';
    isLoading = false;
}

function navigationURL(): URL | null {
    try {
        const parsed = new URL(url);
        const parts = parsed.pathname.split('/').filter(Boolean);
        if (parsed.hostname === 'raw.githubusercontent.com') {
            const index = parts[2] === 'refs' && parts[3] === 'heads' ? 4 : 2;
            parsed.hostname = 'github.com';
            parsed.pathname =
                '/' +
                [...parts.slice(0, 2), 'tree', ...parts.slice(index)].join('/');
        }
        if (parsed.hostname !== 'github.com') return null;
        parsed.search = '';
        parsed.hash = '';
        return parsed;
    } catch {
        return null;
    }
}

export function getCanGoUp(): boolean {
    const parsed = navigationURL();
    if (!parsed) return false;
    const parts = parsed.pathname.split('/').filter(Boolean);
    return parts.length > (parts[2] === 'tree' || parts[2] === 'blob' ? 4 : 2);
}

export function navigateToDir(name: string): void {
    const parsed = navigationURL();
    if (!parsed) return;
    parsed.pathname =
        parsed.pathname.replace(/\/+$/, '') + '/' + encodeURIComponent(name);
    setURL(parsed.toString());
    void fetchImages();
}

export function goUp(): void {
    if (!getCanGoUp()) return;
    const parsed = navigationURL()!;
    const parts = parsed.pathname.split('/').filter(Boolean);
    parts.pop();
    parsed.pathname = '/' + parts.join('/');
    setURL(parsed.toString());
    void fetchImages();
}

export async function fetchImages(): Promise<void> {
    const source = url.trim();
    const sequence = ++requestSequence;
    if (!source) {
        isLoading = false;
        results = [];
        return;
    }
    isLoading = true;
    error = '';
    results = [];
    try {
        const {ListGitHubImages} = await import('../../../wailsjs/go/main/App');
        if (sequence !== requestSequence) return;
        const result = await ListGitHubImages(source);
        if (sequence === requestSequence)
            results = Array.isArray(result?.items) ? result.items : [];
    } catch (failure: unknown) {
        if (sequence === requestSequence)
            error =
                typeof failure === 'string'
                    ? failure
                    : failure instanceof Error
                      ? failure.message
                      : 'Could not fetch repository images';
    } finally {
        if (sequence === requestSequence) isLoading = false;
    }
}
