<script lang="ts" module>
    import type {CardSize} from '$lib/stores/cardsize.svelte';

    // Minimum column width (px) per card size for theme grids (Blueprints,
    // Omarchy themes). Theme cards carry a text body, so they stay narrower
    // than wallpaper cards at M and L.
    export const THEME_CARD_MIN_WIDTH: Record<CardSize, number> = {
        small: 220,
        medium: 290,
        large: 380,
    };
</script>

<script lang="ts">
    import {
        getCachedThumbnail,
        loadThumbnail,
    } from '$lib/stores/imagecache.svelte';
    import type {Blueprint} from '$lib/types/theme';

    let {
        blueprint,
        onload,
        ondelete,
    }: {
        blueprint: Blueprint;
        onload: () => void;
        ondelete: () => void;
    } = $props();

    let confirmingDelete = $state(false);

    let wallpaperPath = $derived(blueprint.palette?.wallpaper ?? '');
    let colors = $derived(blueprint.palette?.colors || []);
    let isLight = $derived(
        blueprint.palette?.mode
            ? blueprint.palette.mode === 'light'
            : !!blueprint.palette?.lightMode
    );

    $effect(() => {
        if (wallpaperPath) loadThumbnail(wallpaperPath);
    });

    function formatDate(ts: number): string {
        if (!ts) return '';
        return new Date(ts).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
        });
    }
</script>

{#snippet tag(label: string)}
    <span
        class="border-border text-fg-secondary whitespace-nowrap border px-1.5 py-0.5 text-[10.5px]"
        >{label}</span
    >
{/snippet}

<div
    class="bg-bg-secondary border-border hover:border-border-focus flex flex-col border transition-colors"
>
    <div class="bg-bg-primary flex aspect-video overflow-hidden">
        {#if wallpaperPath && getCachedThumbnail(wallpaperPath)}
            <img
                src={getCachedThumbnail(wallpaperPath)}
                alt={blueprint.name}
                class="h-full w-full object-cover"
            />
        {:else}
            {#each colors.slice(0, 8) as color}
                <span class="flex-1" style:background-color={color}></span>
            {/each}
        {/if}
    </div>

    <div class="flex h-1.5" aria-hidden="true">
        {#each colors.slice(0, 16) as color}
            <span class="flex-1" style:background-color={color}></span>
        {/each}
    </div>

    <div class="flex flex-col gap-2.5 p-3">
        <div class="flex items-baseline justify-between gap-2">
            <span class="text-fg-primary truncate text-[13px] font-semibold"
                >{blueprint.name}</span
            >
            <span class="text-fg-dimmed shrink-0 text-[11px]"
                >{formatDate(blueprint.timestamp)}</span
            >
        </div>
        {#if confirmingDelete}
            <div class="flex h-7 items-center gap-1.5">
                <span class="text-fg-secondary flex-1 text-[11.5px]"
                    >Delete this theme?</span
                >
                <button
                    type="button"
                    class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-7 px-2.5 text-[12px] transition-colors"
                    onclick={() => (confirmingDelete = false)}>Cancel</button
                >
                <button
                    type="button"
                    class="border-destructive text-destructive hover:bg-destructive hover:text-destructive-fg h-7 border px-2.5 text-[12px] font-medium transition-colors"
                    onclick={ondelete}>Delete</button
                >
            </div>
        {:else}
            <div class="flex items-center gap-1.5">
                <div class="flex min-w-0 flex-1 flex-wrap gap-1.5">
                    {@render tag(isLight ? 'Light' : 'Dark')}
                    {@render tag(wallpaperPath ? 'Wallpaper' : 'Palette only')}
                </div>
                <button
                    type="button"
                    class="text-fg-dimmed hover:text-destructive hover:bg-bg-hover flex h-7 w-7 items-center justify-center transition-colors"
                    onclick={() => (confirmingDelete = true)}
                    aria-label="Delete theme"
                    title="Delete"
                >
                    <svg
                        class="h-3.5 w-3.5"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                    >
                        <path d="M3 6h18 M8 6V4h8v2 M19 6l-1 14H6L5 6"></path>
                    </svg>
                </button>
                <button
                    type="button"
                    class="border-border text-fg-primary hover:bg-accent hover:border-accent hover:text-accent-fg h-7 border px-3 text-[12px] font-medium transition-colors"
                    onclick={onload}
                    title="Load into the editor. Nothing is applied until you click Apply theme."
                    >Load</button
                >
            </div>
        {/if}
    </div>
</div>
