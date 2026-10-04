<script lang="ts">
    import {onMount} from 'svelte';
    import {
        setWallpaperPath,
        addAdditionalImage,
        getAdditionalImages,
    } from '$lib/stores/theme.svelte';
    import {setActiveTab, showToast} from '$lib/stores/ui.svelte';
    import {applyWallpaperOnly} from '$lib/actions/themeActions';
    import {getIsApplying} from '$lib/stores/theme.svelte';
    import {
        getLabels,
        getLabelForPath,
        getAssignments,
    } from '$lib/stores/tags.svelte';
    import {
        loadFullImage,
        getCachedFullImage,
    } from '$lib/stores/imagecache.svelte';
    import {isFavorite, toggleFavorite} from '$lib/stores/favorites.svelte';
    import LazyImage from '$lib/components/shared/LazyImage.svelte';
    import WallpaperTile from '$lib/components/shared/WallpaperTile.svelte';
    import ImagePreview from '$lib/components/shared/ImagePreview.svelte';
    import EmptyState from '$lib/components/shared/EmptyState.svelte';
    import LoadingState from '$lib/components/shared/LoadingState.svelte';
    import ViewHeader from '$lib/components/shared/ViewHeader.svelte';
    import SearchIcon from '$lib/components/shared/SearchIcon.svelte';
    import CardSizeToggle from '$lib/components/shared/CardSizeToggle.svelte';
    import Segmented from '$lib/components/shared/Segmented.svelte';
    import {formatFileSize} from '$lib/utils/format';
    import {getCardSize, CARD_MIN_WIDTH} from '$lib/stores/cardsize.svelte';
    import {getSettings} from '$lib/stores/settings.svelte';
    import type {wallpaper} from '../../../../wailsjs/go/models';

    type Wallpaper = wallpaper.WallpaperInfo;

    let wallpapers = $state<Wallpaper[]>([]);
    let isLoading = $state(true);
    let sortBy = $state<'name' | 'date' | 'size'>('date');
    let filterTag = $state<string>('');
    let query = $state<string>('');
    let previewIndex = $state(-1);
    let previewSrc = $state<string>('');
    let loadError = $state('');
    let wallpaperFolder = $derived(
        getSettings().wallpaperFolder || '~/Wallpapers'
    );

    // Viewport windowing: at 10k+ wallpapers, mounting every card freezes the
    // renderer. We measure the scroll container, compute how many rows fit,
    // and only mount the visible slice (plus a small buffer). Spacer divs at
    // top/bottom preserve scrollbar position so scrolling feels native.
    const CARD_GAP = 12; // matches gap-3
    const GRID_PADDING = 16; // matches p-4 on the scroll container
    const CARD_BORDER = 1; // 1px card border on each side
    const META_ROW_HEIGHT = 32; // h-8 row with the label picker and name
    const BUFFER_ROWS = 3; // rows rendered above + below the viewport

    const SORT_OPTIONS = [
        {value: 'date', label: 'Date', title: 'Newest first'},
        {value: 'name', label: 'Name', title: 'A to Z'},
        {value: 'size', label: 'Size', title: 'Largest first'},
    ] as const;

    let scrollContainer = $state<HTMLDivElement | null>(null);
    let scrollTop = $state(0);
    let containerHeight = $state(600);
    let containerWidth = $state(800);

    onMount(() => {
        loadWallpapers();
    });

    $effect(() => {
        if (!scrollContainer) return;
        const measure = () => {
            if (!scrollContainer) return;
            containerHeight = scrollContainer.clientHeight;
            containerWidth = scrollContainer.clientWidth;
        };
        measure();
        const ro = new ResizeObserver(measure);
        ro.observe(scrollContainer);
        return () => ro.disconnect();
    });

    function handleScroll(e: Event) {
        scrollTop = (e.currentTarget as HTMLDivElement).scrollTop;
    }

    async function loadWallpapers() {
        isLoading = true;
        loadError = '';
        try {
            const {ScanLocalWallpapers} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await ScanLocalWallpapers();
            wallpapers = Array.isArray(result) ? result : [];
        } catch (err) {
            wallpapers = [];
            loadError =
                err instanceof Error
                    ? err.message
                    : 'Wallpaper folder unavailable';
        } finally {
            isLoading = false;
        }
    }

    let allLabels = $derived(getLabels());
    let allAssignments = $derived(getAssignments());

    let filtered = $derived(
        (() => {
            let wps = [...wallpapers];

            if (filterTag) {
                wps = wps.filter(wp => allAssignments[wp.path] === filterTag);
            }

            const q = query.trim().toLowerCase();
            if (q) {
                wps = wps.filter(wp => wp.name.toLowerCase().includes(q));
            }

            wps.sort((a, b) => {
                if (sortBy === 'name') return a.name.localeCompare(b.name);
                if (sortBy === 'size') return b.size - a.size;
                return b.modTime - a.modTime;
            });

            return wps;
        })()
    );

    // Available width is the container minus the horizontal padding.
    let innerWidth = $derived(Math.max(0, containerWidth - GRID_PADDING * 2));
    let minCardWidth = $derived(CARD_MIN_WIDTH[getCardSize()]);
    let columns = $derived(
        Math.max(
            1,
            Math.floor((innerWidth + CARD_GAP) / (minCardWidth + CARD_GAP))
        )
    );
    let cardWidth = $derived(
        columns > 0
            ? (innerWidth - (columns - 1) * CARD_GAP) / columns
            : minCardWidth
    );
    // One row's pitch: the 16:9 thumb inside the border, the meta row, the
    // top and bottom border, and the gap below.
    let rowHeight = $derived(
        Math.round(((cardWidth - CARD_BORDER * 2) * 9) / 16) +
            META_ROW_HEIGHT +
            CARD_BORDER * 2 +
            CARD_GAP
    );
    let totalRows = $derived(Math.ceil(filtered.length / columns));
    let firstVisibleRow = $derived(
        Math.max(0, Math.floor(scrollTop / rowHeight) - BUFFER_ROWS)
    );
    let lastVisibleRow = $derived(
        Math.min(
            totalRows,
            Math.ceil((scrollTop + containerHeight) / rowHeight) + BUFFER_ROWS
        )
    );
    let visibleStart = $derived(firstVisibleRow * columns);
    let visibleEnd = $derived(
        Math.min(filtered.length, lastVisibleRow * columns)
    );
    let visibleSlice = $derived(filtered.slice(visibleStart, visibleEnd));
    let topSpacer = $derived(firstVisibleRow * rowHeight);
    let bottomSpacer = $derived(
        Math.max(0, (totalRows - lastVisibleRow) * rowHeight)
    );

    function selectWallpaper(path: string) {
        setWallpaperPath(path);
        setActiveTab('editor');
        showToast('Wallpaper selected — click Extract to generate palette');
    }

    async function handleBrowse() {
        try {
            const {OpenFileDialog} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const path = await OpenFileDialog();
            if (path) selectWallpaper(path);
        } catch (err) {
            console.error('OpenFileDialog failed', err);
        }
    }

    function handleAddExtra(path: string) {
        if (getAdditionalImages().includes(path)) {
            showToast('Already in additional images');
            return;
        }
        showToast(
            addAdditionalImage(path)
                ? 'Added to additional images'
                : 'Skipped: the theme already has a wallpaper with that filename'
        );
    }

    async function handleFavorite(wp: Wallpaper) {
        try {
            const nowFavorited = await toggleFavorite(wp.path, 'local', {
                name: wp.name,
            });
            showToast(
                nowFavorited ? 'Added to favorites' : 'Removed from favorites'
            );
        } catch (err) {
            console.error('ToggleFavorite failed', err);
            showToast('Could not update favorites');
        }
    }

    async function handlePreview(index: number) {
        const wp = filtered[index];
        const cached = getCachedFullImage(wp.path);
        previewSrc = cached || (await loadFullImage(wp.path));
        previewIndex = index;
    }

    async function navigatePreview(index: number) {
        const wp = filtered[index];
        const cached = getCachedFullImage(wp.path);
        previewSrc = cached || (await loadFullImage(wp.path));
        previewIndex = index;
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
            Local
        </h2>
        <span
            class="text-fg-dimmed mr-2 min-w-0 max-w-[280px] truncate text-[12px]"
            title={wallpaperFolder}
            >{wallpaperFolder} · {filterTag || query
                ? `${filtered.length.toLocaleString()} of ${wallpapers.length.toLocaleString()}`
                : wallpapers.length.toLocaleString()} images</span
        >

        <label
            class="bg-bg-primary border-border focus-within:border-accent text-fg-dimmed flex h-8 w-56 items-center gap-2 border px-2.5 transition-colors"
        >
            <SearchIcon size="h-3.5 w-3.5" />
            <input
                type="search"
                bind:value={query}
                placeholder="Search name or folder…"
                aria-label="Search local wallpapers"
                class="text-fg-primary min-w-0 flex-1 bg-transparent text-[12px] outline-none"
            />
            {#if query}
                <button
                    class="hover:text-fg-primary -mr-1 flex h-5 w-5 items-center justify-center transition-colors"
                    onclick={() => (query = '')}
                    title="Clear search"
                    aria-label="Clear search"
                    ><svg
                        class="h-3 w-3"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2.4"
                        stroke-linecap="round"
                        aria-hidden="true"
                        ><path d="M18 6L6 18M6 6l12 12"></path></svg
                    ></button
                >
            {/if}
        </label>

        <Segmented
            options={SORT_OPTIONS}
            value={sortBy}
            onchange={v => (sortBy = v)}
            size="sm"
            label="Sort wallpapers"
        />

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
                class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary flex h-8 items-center gap-1.5 border px-3 text-[12px] font-medium transition-colors"
                onclick={handleBrowse}
                title="Browse local files"
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
                    ><path
                        d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                    ></path></svg
                >
                Browse…
            </button>
        </div>
    </ViewHeader>

    <div
        class="flex-1 overflow-y-auto p-4"
        bind:this={scrollContainer}
        onscroll={handleScroll}
    >
        {#if isLoading}
            <LoadingState message={`Scanning ${wallpaperFolder}...`} />
        {:else if loadError}
            <EmptyState
                title="Wallpaper folder unavailable"
                body={`${wallpaperFolder} could not be read. Choose another folder in Settings.`}
                actionLabel="Open settings"
                onaction={() => setActiveTab('settings')}
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
                            d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                        ></path>
                        <path d="M9 12h6M12 9v6"></path>
                    </svg>
                {/snippet}
            </EmptyState>
        {:else if filtered.length === 0}
            {#if query || filterTag}
                <EmptyState
                    title={query
                        ? `No matches for "${query}"`
                        : 'No wallpapers with this label'}
                    body="Adjust your search or filters to see more wallpapers."
                    actionLabel="Clear filters"
                    onaction={() => {
                        query = '';
                        filterTag = '';
                    }}
                >
                    {#snippet icon()}
                        <SearchIcon
                            size="h-[26px] w-[26px]"
                            strokeWidth={1.5}
                        />
                    {/snippet}
                </EmptyState>
            {:else}
                <EmptyState
                    title="No wallpapers found"
                    body={`Add images to ${wallpaperFolder} (or any subfolder), or browse to pick a single file.`}
                    actionLabel="Browse for a file…"
                    onaction={handleBrowse}
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
                                d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                            ></path>
                        </svg>
                    {/snippet}
                </EmptyState>
            {/if}
        {:else}
            {#if topSpacer > 0}
                <div aria-hidden="true" style="height: {topSpacer}px"></div>
            {/if}
            <div
                class="grid gap-3"
                style:grid-template-columns="repeat(auto-fill, minmax({minCardWidth}px,
                1fr))"
            >
                {#each visibleSlice as wp, sliceIdx (wp.path)}
                    {@const i = visibleStart + sliceIdx}
                    <WallpaperTile
                        path={wp.path}
                        name={wp.name}
                        detail={formatFileSize(wp.size)}
                        isAdded={getAdditionalImages().includes(wp.path)}
                        isFavorited={isFavorite(wp.path)}
                        applying={getIsApplying()}
                        onuse={() => selectWallpaper(wp.path)}
                        onwallpaperonly={() => applyWallpaperOnly(wp.path)}
                        onpreview={() => handlePreview(i)}
                        onaddextra={() => handleAddExtra(wp.path)}
                        onfavorite={() => handleFavorite(wp)}
                    >
                        {#snippet thumb()}
                            <LazyImage path={wp.path} alt={wp.name} />
                        {/snippet}
                    </WallpaperTile>
                {/each}
            </div>
            {#if bottomSpacer > 0}
                <div aria-hidden="true" style="height: {bottomSpacer}px"></div>
            {/if}
        {/if}
    </div>
</div>

<ImagePreview
    src={previewSrc}
    alt={previewIndex >= 0 ? filtered[previewIndex]?.name : ''}
    open={previewIndex >= 0}
    onclose={() => {
        previewIndex = -1;
        previewSrc = '';
    }}
    hasPrev={previewIndex > 0}
    hasNext={previewIndex < filtered.length - 1}
    onprev={() => navigatePreview(previewIndex - 1)}
    onnext={() => navigatePreview(previewIndex + 1)}
/>
