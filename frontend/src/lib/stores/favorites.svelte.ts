// One shared list keeps favorite controls in sync without per-card IPC calls.

import type {favorites as favoritesNs} from '../../../wailsjs/go/models';

export type Favorite = favoritesNs.Favorite;

let favorites = $state<Favorite[]>([]);
let pathSet = $derived(new Set(favorites.map(f => f.path)));
let loadError = $state('');
let refreshSequence = 0;
let mutationRevision = 0;
const modifiedPaths = new Map<string, number>();
const pendingToggles = new Map<string, Promise<boolean>>();

refreshFavorites();

export async function refreshFavorites(): Promise<void> {
    const sequence = ++refreshSequence;
    const revision = mutationRevision;
    try {
        const {GetFavorites} = await import('../../../wailsjs/go/main/App');
        const result = await GetFavorites();
        if (sequence !== refreshSequence) return;

        // Keep newer local mutations while accepting unrelated backend entries.
        const changed = (path: string) =>
            pendingToggles.has(path) ||
            (modifiedPaths.get(path) ?? 0) > revision;
        favorites = [
            ...(Array.isArray(result) ? result : []).filter(
                f => !changed(f.path)
            ),
            ...favorites.filter(f => changed(f.path)),
        ];
        loadError = '';
        for (const path of modifiedPaths.keys()) {
            if (!pendingToggles.has(path)) modifiedPaths.delete(path);
        }
    } catch {
        if (sequence === refreshSequence)
            loadError = 'Could not load favorites. Try again.';
    }
}

export function getFavorites(): Favorite[] {
    return favorites;
}

export function getFavoritesError(): string {
    return loadError;
}

export function isFavorite(path: string): boolean {
    return pathSet.has(path);
}

// Adds or removes `path`; resolves to the new favorited state. `data` is
// only used when adding (see favorites.Service.buildEntry for the shape
// each type accepts).
export function toggleFavorite(
    path: string,
    type: string,
    data: Record<string, unknown> = {}
): Promise<boolean> {
    modifiedPaths.set(path, ++mutationRevision);
    const previous = pendingToggles.get(path) ?? Promise.resolve(false);
    const metadata = {...data};
    const operation = previous
        .catch(() => false)
        .then(async () => {
            const {ToggleFavorite} = await import(
                '../../../wailsjs/go/main/App'
            );
            const nowFavorited = await ToggleFavorite(path, type, metadata);
            if (nowFavorited) {
                if (!pathSet.has(path))
                    favorites = [...favorites, {path, type, data: metadata}];
            } else {
                favorites = favorites.filter(f => f.path !== path);
            }
            return nowFavorited;
        });
    const queued = operation.finally(() => {
        modifiedPaths.set(path, ++mutationRevision);
        if (pendingToggles.get(path) === queued) pendingToggles.delete(path);
    });
    pendingToggles.set(path, queued);
    return queued;
}
