<script lang="ts">
    import type {Snippet} from 'svelte';
    import TagPicker from './TagPicker.svelte';

    // Shared card for the Wallhaven, GitHub, Local, and Favorites grids. The
    // thumbnail source differs per view, so it arrives as a `thumb` snippet.
    // Hover or keyboard focus shows the actions over the image. The favourite
    // heart renders only when `onfavorite` is supplied, and the source link
    // only when `onvisit` is supplied. Clicking the thumbnail or the Use
    // button selects the wallpaper. `meta` replaces the default metadata row
    // (label picker, name, and `detail`).
    let {
        path,
        name,
        detail = '',
        isAdded = false,
        isFavorited = false,
        applying = false,
        busy = false,
        useLabel = 'Use',
        useTitle = 'Set as wallpaper and open in editor',
        visitTitle = 'Open source page',
        onuse,
        onwallpaperonly,
        onpreview,
        onaddextra,
        onfavorite,
        onvisit,
        thumb,
        meta,
    }: {
        path: string;
        name: string;
        detail?: string;
        isAdded?: boolean;
        isFavorited?: boolean;
        applying?: boolean;
        busy?: boolean;
        useLabel?: string;
        useTitle?: string;
        visitTitle?: string;
        onuse: () => void;
        onwallpaperonly: () => void;
        onpreview: () => void;
        onaddextra: () => void;
        onfavorite?: () => void;
        onvisit?: () => void;
        thumb: Snippet;
        meta?: Snippet;
    } = $props();

    const HEART =
        'M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z';

    // Fixed scrim colors: these controls sit on arbitrary image content.
    const SCRIM =
        'border border-white/14 bg-[rgba(12,12,16,0.5)] backdrop-blur-[8px]';
    const TOOL =
        'flex h-7 w-[30px] items-center justify-center text-white/85 transition-colors hover:bg-white/16 hover:text-white disabled:opacity-40';
</script>

{#snippet icon(d: string, fill = 'none', size = 'h-3.5 w-3.5')}
    <svg
        class={size}
        viewBox="0 0 24 24"
        {fill}
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"><path {d}></path></svg
    >
{/snippet}

<div
    class="bg-bg-secondary border-border hover:border-border-focus group flex flex-col border transition-colors duration-100"
>
    <div class="bg-bg-primary relative aspect-video overflow-hidden">
        <button
            data-wallpaper-tile
            class="flex h-full w-full items-center justify-center"
            onclick={onuse}
            disabled={busy}
            title={useTitle}
        >
            {@render thumb()}
        </button>

        {#if isAdded}
            <span
                class="{SCRIM} pointer-events-none absolute left-2 top-2 z-[1] flex h-7 items-center gap-1.5 pl-[7px] pr-[9px] text-[11px] font-medium text-white"
                aria-hidden="true"
            >
                {@render icon('M20 6 9 17l-5-5', 'none', 'h-3 w-3')}
                Extra
            </span>
        {/if}
        {#if isFavorited}
            <span
                class="{SCRIM} text-destructive pointer-events-none absolute right-2 top-2 z-[1] flex h-7 w-7 items-center justify-center"
                aria-hidden="true"
            >
                {@render icon(HEART, 'currentColor')}
            </span>
        {/if}

        <div
            class="pointer-events-none absolute inset-0 z-[2] flex flex-col justify-between p-2 opacity-0 transition-opacity duration-150 ease-out group-focus-within:opacity-100 group-hover:opacity-100"
            style="background: linear-gradient(to bottom, rgba(0,0,0,0.42) 0%, rgba(0,0,0,0) 34%, rgba(0,0,0,0) 48%, rgba(0,0,0,0.72) 100%)"
        >
            <div class="flex items-start gap-2">
                <button
                    class="{SCRIM} pointer-events-auto flex h-7 items-center gap-1.5 whitespace-nowrap pl-[7px] pr-[9px] text-[11px] font-medium text-white transition-colors hover:bg-[rgba(12,12,16,0.78)] disabled:opacity-40"
                    onclick={e => {
                        e.stopPropagation();
                        onaddextra();
                    }}
                    disabled={busy}
                    aria-label={isAdded
                        ? 'Already in additional images'
                        : 'Add to additional images'}
                    title={isAdded
                        ? 'Already in additional images'
                        : 'Add to additional images (blend into extraction)'}
                >
                    {#if isAdded}
                        {@render icon('M20 6 9 17l-5-5', 'none', 'h-3 w-3')}
                    {:else}
                        {@render icon('M12 5v14M5 12h14', 'none', 'h-3 w-3')}
                    {/if}
                    Extra
                </button>
                <span class="flex-1"></span>
                {#if onfavorite}
                    <button
                        class="{SCRIM} pointer-events-auto flex h-7 w-7 items-center justify-center transition-colors hover:bg-[rgba(12,12,16,0.78)] {isFavorited
                            ? 'text-destructive'
                            : 'text-white'}"
                        onclick={e => {
                            e.stopPropagation();
                            onfavorite();
                        }}
                        aria-label={isFavorited
                            ? 'Remove from favorites'
                            : 'Add to favorites'}
                        title={isFavorited
                            ? 'Remove from favorites'
                            : 'Add to favorites'}
                    >
                        {@render icon(
                            HEART,
                            isFavorited ? 'currentColor' : 'none'
                        )}
                    </button>
                {/if}
            </div>

            <div class="flex items-center gap-1.5">
                <button
                    class="bg-accent text-accent-fg hover:bg-accent-hover pointer-events-auto flex h-[30px] min-w-0 flex-1 items-center justify-center gap-1.5 whitespace-nowrap px-3 text-[12px] font-semibold transition-colors disabled:opacity-60"
                    onclick={onuse}
                    disabled={busy}
                    title={useTitle}
                >
                    {useLabel}
                    {#if !busy}
                        {@render icon(
                            'M5 12h14 M13 6l6 6-6 6',
                            'none',
                            'h-[13px] w-[13px]'
                        )}
                    {/if}
                </button>
                <div
                    class="border-white/14 pointer-events-auto flex shrink-0 border bg-[rgba(12,12,16,0.55)] backdrop-blur-[8px]"
                >
                    <button
                        class={TOOL}
                        onclick={onwallpaperonly}
                        disabled={applying || busy}
                        aria-label="Apply as wallpaper only"
                        title="Apply this wallpaper without changing the current palette"
                    >
                        {@render icon('M3 4h18v12H3z M8 20h8 M12 16v4')}
                    </button>
                    <button
                        class="{TOOL} shadow-[inset_1px_0_0_rgba(255,255,255,0.1)]"
                        onclick={onpreview}
                        aria-label="Preview full-size"
                        title="Preview wallpaper full-size"
                    >
                        {@render icon(
                            'M15 3h6v6 M9 21H3v-6 M21 3l-7 7 M3 21l7-7'
                        )}
                    </button>
                    {#if onvisit}
                        <button
                            class="{TOOL} shadow-[inset_1px_0_0_rgba(255,255,255,0.1)]"
                            onclick={onvisit}
                            aria-label={visitTitle}
                            title={visitTitle}
                        >
                            {@render icon(
                                'M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6 M15 3h6v6 M10 14 21 3'
                            )}
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    </div>

    <div class="flex h-8 min-w-0 items-center gap-2 px-2.5 text-[11px]">
        {#if meta}
            {@render meta()}
        {:else}
            <TagPicker {path} />
            <span class="text-fg-secondary min-w-0 flex-1 truncate" title={name}
                >{name}</span
            >
            {#if detail}
                <span class="text-fg-dimmed shrink-0">{detail}</span>
            {/if}
        {/if}
    </div>
</div>
