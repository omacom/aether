<script lang="ts">
    import {onMount} from 'svelte';
    import WallhavenFilters from './WallhavenFilters.svelte';
    import WallpaperGrid from './WallpaperGrid.svelte';
    import {
        getResults,
        getIsSearching,
        getIsLoadingMore,
        getHasMore,
        getSearchError,
        getQuery,
        setQuery,
        search,
        loadMore,
        initializeSearch,
    } from '$lib/stores/wallhaven.svelte';
    import {observeIntersection} from '$lib/utils/intersection';
    import EmptyState from '$lib/components/shared/EmptyState.svelte';
    import LoadingState from '$lib/components/shared/LoadingState.svelte';
    import SearchIcon from '$lib/components/shared/SearchIcon.svelte';

    let scrollContainer = $state<HTMLDivElement | null>(null);
    let sentinel = $state<HTMLDivElement | null>(null);

    onMount(() => {
        void initializeSearch();
    });

    // Re-observe on every results change so the sentinel fires again even if it
    // stays intersecting — IntersectionObserver only reports *transitions*, so
    // a sentinel that's permanently in-view (first page smaller than viewport)
    // would otherwise trigger only once. Re-observing pumps loadMore until the
    // viewport fills or hasMore becomes false.
    $effect(() => {
        if (!sentinel || !scrollContainer) return;
        getResults().length;
        return observeIntersection(
            sentinel,
            entry => {
                if (
                    entry.isIntersecting &&
                    getHasMore() &&
                    !getIsSearching() &&
                    !getIsLoadingMore() &&
                    !getSearchError()
                ) {
                    loadMore();
                }
            },
            {root: scrollContainer, rootMargin: '400px 0px'}
        );
    });
</script>

<div class="flex h-full flex-col">
    <WallhavenFilters />

    <div class="flex-1 overflow-y-auto p-3" bind:this={scrollContainer}>
        {#if getIsSearching() && getResults().length === 0}
            <LoadingState message="Searching wallhaven…" />
        {:else if getSearchError() && getResults().length === 0}
            <EmptyState
                title="Search failed"
                body={getSearchError()}
                actionLabel="Retry"
                onaction={search}
            />
        {:else if getResults().length === 0}
            <EmptyState
                title={getQuery()
                    ? 'No matches found'
                    : 'No wallpapers to show'}
                body={getQuery()
                    ? `Nothing on wallhaven matches "${getQuery()}" with the current filters. Try a different query or reset filters.`
                    : 'Try a search term, or browse by category and purity to discover wallpapers.'}
                actionLabel={getQuery() ? 'Clear search' : undefined}
                onaction={getQuery()
                    ? () => {
                          setQuery('');
                          search();
                      }
                    : undefined}
            >
                {#snippet icon()}
                    <SearchIcon size="h-12 w-12" strokeWidth={1.5} />
                {/snippet}
            </EmptyState>
        {:else}
            <WallpaperGrid wallpapers={getResults()} />

            <div bind:this={sentinel} class="h-1 w-full"></div>

            {#if getIsLoadingMore()}
                <div class="flex h-12 items-center justify-center">
                    <span class="text-fg-dimmed text-[12px]"
                        >Loading more...</span
                    >
                </div>
            {:else if getSearchError()}
                <div
                    class="text-fg-dimmed flex items-center justify-center gap-3 py-3 text-[11px]"
                >
                    <span>{getSearchError()}</span>
                    <button
                        class="text-accent hover:text-accent-hover"
                        onclick={loadMore}>Retry</button
                    >
                </div>
            {:else if !getHasMore()}
                <div class="flex h-12 items-center justify-center">
                    <span class="text-fg-dimmed text-[12px]"
                        >End of results</span
                    >
                </div>
            {/if}
        {/if}
    </div>
</div>
