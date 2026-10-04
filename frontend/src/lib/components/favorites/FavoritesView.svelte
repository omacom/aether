<script lang="ts">
    import {onMount, onDestroy} from 'svelte';
    import {
        setWallpaperPath,
        addAdditionalImage,
        getAdditionalImages,
        getWallpaperRevision,
    } from '$lib/stores/theme.svelte';
    import {setActiveTab, showToast} from '$lib/stores/ui.svelte';
    import {
        getCachedThumbnail,
        loadThumbnail,
        isThumbnailCached,
        setCachedImage,
        loadFullImage,
        getCachedFullImage,
    } from '$lib/stores/imagecache.svelte';
    import {getLabels, getAssignments} from '$lib/stores/tags.svelte';
    import {
        getExportBusy,
        startExport,
    } from '$lib/stores/favoritesExport.svelte';
    import WallpaperTile from '$lib/components/shared/WallpaperTile.svelte';
    import ImagePreview from '$lib/components/shared/ImagePreview.svelte';
    import EmptyState from '$lib/components/shared/EmptyState.svelte';
    import LoadingState from '$lib/components/shared/LoadingState.svelte';
    import ViewHeader from '$lib/components/shared/ViewHeader.svelte';
    import CardSizeToggle from '$lib/components/shared/CardSizeToggle.svelte';
    import {getCardSize, CARD_MIN_WIDTH} from '$lib/stores/cardsize.svelte';
    import {applyWallpaperOnly} from '$lib/actions/themeActions';
    import {openURL} from '$lib/utils/browser';
    import {getIsApplying} from '$lib/stores/theme.svelte';
    import {
        getFavorites,
        getFavoritesError,
        refreshFavorites,
        toggleFavorite,
        type Favorite,
    } from '$lib/stores/favorites.svelte';

    let favorites = $derived(getFavorites());
    let loadError = $derived(getFavoritesError());
    let isLoading = $state(true);
    let filterTag = $state<string>('');
    let previewIndex = $state(-1);
    let previewSrc = $state<string>('');
    let current = true;
    let previewSequence = 0;
    onDestroy(() => {
        current = false;
        previewSequence++;
    });

    let allLabels = $derived(getLabels());
    let allAssignments = $derived(getAssignments());

    let filtered = $derived(
        filterTag
            ? favorites.filter(f => allAssignments[f.path] === filterTag)
            : favorites
    );

    onMount(() => {
        loadFavorites();
    });

    // Re-sync with the backend on every mount so favourites added via the
    // CLI/IPC while this tab was closed show up.
    async function loadFavorites() {
        isLoading = true;
        try {
            await refreshFavorites();
            loadThumbnails();
        } finally {
            isLoading = false;
        }
    }

    async function loadRemoteThumb(path: string) {
        try {
            const {GetGitHubThumbnail} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await GetGitHubThumbnail(path);
            if (result?.dataURL)
                setCachedImage('thumb:' + path, result.dataURL);
        } catch {}
    }

    async function loadThumbnails() {
        for (const fav of favorites) {
            if (isThumbnailCached(fav.path)) continue;

            // Wallhaven thumbs are remote URLs — cache directly
            if (fav.data?.thumbUrl && fav.data.thumbUrl.startsWith('http')) {
                setCachedImage('thumb:' + fav.path, fav.data.thumbUrl);
                continue;
            }

            // Remote GitHub URLs — use Go thumbnail generator (download +
            // resize to 300px, cached on disk for subsequent loads). Fire and
            // forget so the loop doesn't block on HTTP downloads.
            if (
                fav.path.startsWith('http://') ||
                fav.path.startsWith('https://')
            ) {
                loadRemoteThumb(fav.path);
                continue;
            }

            // Local files — load thumbnail
            loadThumbnail(fav.path);
        }
    }

    async function handleSelect(fav: Favorite) {
        const revision = getWallpaperRevision();
        let localPath = fav.path;

        if (
            localPath &&
            (localPath.startsWith('http://') ||
                localPath.startsWith('https://'))
        ) {
            try {
                showToast('Downloading wallpaper...');
                const {DownloadWallpaper} = await import(
                    '../../../../wailsjs/go/main/App'
                );
                localPath = await DownloadWallpaper(localPath);
            } catch {
                showToast('Failed to download wallpaper');
                return;
            }
        }

        if (!current || revision !== getWallpaperRevision()) return;
        setWallpaperPath(localPath);
        setActiveTab('editor');
        showToast('Wallpaper selected — click Extract to generate palette');
    }

    async function handleRemove(fav: Favorite) {
        try {
            await toggleFavorite(fav.path, fav.type ?? '');
        } catch (err) {
            console.error('ToggleFavorite failed', err);
            showToast('Could not update favorites');
        }
    }

    async function handleAddExtra(fav: Favorite) {
        const revision = getWallpaperRevision();
        let localPath = fav.path;

        if (
            localPath &&
            (localPath.startsWith('http://') ||
                localPath.startsWith('https://'))
        ) {
            try {
                showToast('Downloading wallpaper...');
                const {DownloadWallpaper} = await import(
                    '../../../../wailsjs/go/main/App'
                );
                localPath = await DownloadWallpaper(localPath);
            } catch {
                showToast('Failed to download wallpaper');
                return;
            }
        }

        if (!current || revision !== getWallpaperRevision()) return;
        if (getAdditionalImages().includes(localPath)) {
            showToast('Already in additional images');
            return;
        }
        showToast(
            addAdditionalImage(localPath)
                ? 'Added to additional images'
                : 'Skipped: the theme already has a wallpaper with that filename'
        );
    }

    async function resolvePreviewSrc(fav: Favorite): Promise<string> {
        if (fav.type === 'github') {
            const {DownloadWallpaper} = await import(
                '../../../../wailsjs/go/main/App'
            );
            return loadFullImage(await DownloadWallpaper(fav.path));
        }
        if (fav.path?.startsWith('http')) return fav.path;
        const cached = getCachedFullImage(fav.path);
        return cached || (await loadFullImage(fav.path));
    }

    async function handlePreview(index: number) {
        const selected = filtered[index];
        if (!selected) return;
        const sequence = ++previewSequence;
        try {
            const source = await resolvePreviewSrc(selected);
            if (
                !current ||
                sequence !== previewSequence ||
                filtered[index]?.path !== selected.path
            )
                return;
            previewSrc = source;
            previewIndex = index;
        } catch {
            if (current && sequence === previewSequence)
                showToast('Could not load the wallpaper preview');
        }
    }

    async function navigatePreview(index: number) {
        await handlePreview(index);
    }
</script>

{#snippet chip(
    active: boolean,
    label: string,
    color: string,
    onclick: () => void
)}
    <button
        class="flex h-6 items-center gap-1.5 border px-[9px] text-[11.5px] transition-colors {active
            ? ''
            : 'text-fg-dimmed border-border hover:text-fg-secondary'}"
        style={active && color
            ? `background: ${color}24; border-color: ${color}; color: ${color};`
            : ''}
        class:text-accent={active && !color}
        class:border-accent={active && !color}
        class:bg-accent-muted={active && !color}
        aria-pressed={active}
        {onclick}
    >
        {#if color}
            <span class="h-2 w-2 shrink-0" style:background-color={color}
            ></span>
        {/if}
        {label}
    </button>
{/snippet}

<div class="flex h-full flex-col">
    <ViewHeader>
        <h2 class="text-fg-primary shrink-0 text-[13.5px] font-semibold">
            Favorites
        </h2>
        <span class="text-fg-dimmed mr-2 shrink-0 text-[12px] tabular-nums"
            >{filterTag
                ? `${filtered.length} of ${favorites.length}`
                : favorites.length}
            {favorites.length === 1 ? 'wallpaper' : 'wallpapers'}</span
        >

        {#if allLabels.length > 0}
            <span class="bg-border h-4 w-px"></span>
            <div class="flex flex-wrap items-center gap-1">
                {@render chip(!filterTag, 'All', '', () => (filterTag = ''))}
                {#each allLabels as label}
                    {@render chip(
                        filterTag === label.id,
                        label.name,
                        label.color,
                        () =>
                            (filterTag = filterTag === label.id ? '' : label.id)
                    )}
                {/each}
            </div>
        {/if}

        <div class="ml-auto flex shrink-0 items-center gap-2">
            <CardSizeToggle />
            <button
                class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 border px-3 text-[12px] font-medium transition-colors disabled:pointer-events-none disabled:opacity-45"
                disabled={filtered.length === 0 || getExportBusy()}
                onclick={() => startExport(filtered.map(f => f.path))}
                title="Export the listed favorites as a .zip archive"
                >Export .zip · {filtered.length}</button
            >
        </div>
    </ViewHeader>

    <div class="flex-1 overflow-y-auto p-4">
        {#if loadError}
            <div
                class="border-destructive/40 bg-destructive/8 text-fg-primary mb-4 flex items-center justify-between gap-3 border px-3.5 py-2.5 text-[12px]"
                role="alert"
            >
                <span>{loadError}</span>
                <button
                    class="text-accent hover:text-accent-hover shrink-0 px-2 py-1 text-[12px] font-medium transition-colors"
                    onclick={loadFavorites}
                    disabled={isLoading}>Retry</button
                >
            </div>
        {/if}
        {#if isLoading}
            <LoadingState message="Loading favorites…" />
        {:else if filtered.length === 0 && !loadError}
            {#if filterTag}
                <EmptyState
                    title="No favorites with this label"
                    body="Try a different label or remove the filter to see all favorites."
                    actionLabel="Clear filter"
                    onaction={() => (filterTag = '')}
                >
                    {#snippet icon()}
                        <svg
                            class="h-[26px] w-[26px]"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.5"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path
                                d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"
                            ></path>
                        </svg>
                    {/snippet}
                </EmptyState>
            {:else}
                <EmptyState
                    title="No favorites yet"
                    body="Tap the heart on any wallpaper in Wallhaven or Local to save it here for quick access."
                    actionLabel="Browse Wallhaven"
                    onaction={() => setActiveTab('wallhaven')}
                    secondaryLabel="Browse Local"
                    onsecondary={() => setActiveTab('local')}
                >
                    {#snippet icon()}
                        <svg
                            class="h-[26px] w-[26px]"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.5"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path
                                d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"
                            ></path>
                        </svg>
                    {/snippet}
                </EmptyState>
            {/if}
        {:else}
            <div
                class="grid gap-3"
                style:grid-template-columns="repeat(auto-fill, minmax({CARD_MIN_WIDTH[
                    getCardSize()
                ]}px, 1fr))"
            >
                {#each filtered as fav, i (fav.path)}
                    <WallpaperTile
                        path={fav.path}
                        name={fav.data?.name || fav.data?.id || 'Wallpaper'}
                        detail={fav.data?.resolution || ''}
                        isAdded={getAdditionalImages().includes(fav.path)}
                        isFavorited={true}
                        applying={getIsApplying()}
                        onuse={() => handleSelect(fav)}
                        onwallpaperonly={() => applyWallpaperOnly(fav.path)}
                        onpreview={() => handlePreview(i)}
                        onaddextra={() => handleAddExtra(fav)}
                        onfavorite={() => handleRemove(fav)}
                        onvisit={fav.type === 'wallhaven' && fav.data?.id
                            ? () =>
                                  openURL(
                                      `https://wallhaven.cc/w/${fav.data?.id}`
                                  )
                            : undefined}
                        visitTitle="Open on wallhaven.cc"
                    >
                        {#snippet thumb()}
                            {#if getCachedThumbnail(fav.path)}
                                <img
                                    src={getCachedThumbnail(fav.path)}
                                    alt=""
                                    class="h-full w-full object-cover"
                                />
                            {:else}
                                <span
                                    class="text-fg-dimmed font-mono text-[10px]"
                                    >Loading…</span
                                >
                            {/if}
                        {/snippet}
                    </WallpaperTile>
                {/each}
            </div>
        {/if}
    </div>
</div>

<ImagePreview
    src={previewSrc}
    alt={previewIndex >= 0
        ? filtered[previewIndex]?.data?.name || 'Favorite wallpaper'
        : ''}
    open={previewIndex >= 0}
    onclose={() => {
        previewSequence++;
        previewIndex = -1;
        previewSrc = '';
    }}
    hasPrev={previewIndex > 0}
    hasNext={previewIndex < filtered.length - 1}
    onprev={() => navigatePreview(previewIndex - 1)}
    onnext={() => navigatePreview(previewIndex + 1)}
/>
