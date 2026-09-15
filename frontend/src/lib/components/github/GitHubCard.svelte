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
                addAdditionalImage(path);
                showToast('Added to additional images');
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
            {:else}<span class="text-fg-dimmed text-[10px]"
                    >{thumbError
                        ? 'Preview unavailable'
                        : 'Loading preview…'}</span
                >{/if}
        {/snippet}
    </WallpaperTile>
    <div
        class="text-fg-dimmed mt-1 flex items-center justify-between gap-2 text-[10px]"
    >
        <span>{dimensions}</span>
        {#if thumbError}<button
                class="text-accent px-1 py-0.5"
                onclick={loadThumbnail}>Retry preview</button
            >{/if}
        <span
            >{image.size > 0
                ? `${(image.size / 1024 / 1024).toFixed(1)} MB`
                : ''}</span
        >
    </div>
</div>
