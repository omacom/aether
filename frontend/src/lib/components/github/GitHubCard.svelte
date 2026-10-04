<script lang="ts">
    import {onDestroy} from 'svelte';
    import WallpaperTile from '$lib/components/shared/WallpaperTile.svelte';
    import {
        setWallpaperPath,
        addAdditionalImage,
        getAdditionalImages,
        getIsApplying,
        getWallpaperRevision,
    } from '$lib/stores/theme.svelte';
    import {setActiveTab, showToast} from '$lib/stores/ui.svelte';
    import {applyWallpaperOnly} from '$lib/actions/themeActions';
    import {isFavorite, toggleFavorite} from '$lib/stores/favorites.svelte';
    import {observeIntersection} from '$lib/utils/intersection';
    import {formatFileSize} from '$lib/utils/format';
    import type {githubsource} from '../../../../wailsjs/go/models';

    let {
        image,
        onpreview,
    }: {image: githubsource.ImageInfo; onpreview: () => void} = $props();
    let card = $state<HTMLDivElement | null>(null);
    let thumbSrc = $state('');
    let localPath = $state('');
    let dimensions = $state('');
    let thumbError = $state(false);
    let loading = $state(false);
    let busy = $state(false);
    let current = true;
    onDestroy(() => {
        current = false;
    });

    $effect(() => {
        if (!card) return;
        return observeIntersection(
            card,
            entry => {
                if (entry.isIntersecting && !thumbSrc && !thumbError)
                    void loadThumbnail();
            },
            {rootMargin: '300px 0px'}
        );
    });

    async function loadThumbnail() {
        if (loading) return;
        loading = true;
        thumbError = false;
        try {
            const {GetGitHubThumbnail} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await GetGitHubThumbnail(image.url);
            if (!current) return;
            thumbSrc = result.dataURL;
            dimensions = `${result.width}×${result.height}`;
        } catch {
            if (current) thumbError = true;
        } finally {
            if (current) loading = false;
        }
    }

    async function useImage(additional = false) {
        if (busy) return;
        busy = true;
        const revision = getWallpaperRevision();
        const source = image.url;
        try {
            const {DownloadWallpaper} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const path = await DownloadWallpaper(source);
            if (
                !current ||
                source !== image.url ||
                revision !== getWallpaperRevision()
            )
                return;
            localPath = path;
            if (additional) {
                if (getAdditionalImages().includes(path)) {
                    showToast('Already in additional images');
                    return;
                }
                showToast(
                    addAdditionalImage(path)
                        ? 'Added to additional images'
                        : 'Skipped: the theme already has a wallpaper with that filename'
                );
            } else {
                setWallpaperPath(path);
                setActiveTab('editor');
                showToast('Wallpaper selected. Extract colors when ready.');
            }
        } catch {
            if (current)
                showToast('Could not download the wallpaper. Try again.');
        } finally {
            if (current) busy = false;
        }
    }

    async function favorite() {
        try {
            await toggleFavorite(image.url, 'github', {name: image.name});
        } catch {
            showToast('Could not update favorites');
        }
    }
</script>

<div bind:this={card}>
    <WallpaperTile
        path={image.url}
        name={image.name}
        {busy}
        applying={getIsApplying()}
        isFavorited={isFavorite(image.url)}
        isAdded={!!localPath && getAdditionalImages().includes(localPath)}
        useLabel={busy ? 'Loading…' : 'Use'}
        onuse={() => useImage()}
        onaddextra={() => useImage(true)}
        onfavorite={favorite}
        onwallpaperonly={() => applyWallpaperOnly(image.url)}
        {onpreview}
    >
        {#snippet thumb()}
            {#if thumbSrc}<img
                    src={thumbSrc}
                    alt={image.name}
                    class="h-full w-full object-cover"
                    loading="lazy"
                />
            {:else}<span class="text-fg-dimmed font-mono text-[10px]"
                    >{thumbError
                        ? 'Preview unavailable'
                        : 'Loading preview…'}</span
                >{/if}
        {/snippet}
        {#snippet meta()}
            <span
                class="text-fg-secondary shrink-0 font-mono"
                title={image.name}>{dimensions || '—'}</span
            >
            <span
                class="text-fg-dimmed min-w-0 flex-1 truncate"
                title={image.name}>{image.name}</span
            >
            {#if thumbError}<button
                    class="text-accent hover:text-accent-hover shrink-0"
                    onclick={loadThumbnail}>Retry preview</button
                >{/if}
            <span class="text-fg-dimmed shrink-0"
                >{formatFileSize(image.size)}</span
            >
        {/snippet}
    </WallpaperTile>
</div>
