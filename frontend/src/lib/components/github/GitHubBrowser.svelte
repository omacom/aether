<script lang="ts">
    import {onDestroy} from 'svelte';
    import GitHubCard from './GitHubCard.svelte';
    import {getCardSize, CARD_MIN_WIDTH} from '$lib/stores/cardsize.svelte';
    import CardSizeToggle from '$lib/components/shared/CardSizeToggle.svelte';
    import EmptyState from '$lib/components/shared/EmptyState.svelte';
    import LoadingState from '$lib/components/shared/LoadingState.svelte';
    import ViewHeader from '$lib/components/shared/ViewHeader.svelte';
    import ImagePreview from '$lib/components/shared/ImagePreview.svelte';
    import type {githubsource} from '../../../../wailsjs/go/models';
    import {showToast} from '$lib/stores/ui.svelte';
    import {loadFullImage} from '$lib/stores/imagecache.svelte';
    import {
        getURL,
        getResults,
        getIsLoading,
        getError,
        getCanGoUp,
        setURL,
        fetchImages,
        navigateToDir as storeNavigateToDir,
        goUp as storeGoUp,
    } from '$lib/stores/github.svelte';

    type ImageInfo = githubsource.ImageInfo;

    const SAVED_REPOS_KEY = 'aether-saved-repos';

    type SavedRepo = {name: string; url: string};

    function loadSavedRepos(): SavedRepo[] {
        try {
            const saved = JSON.parse(
                localStorage.getItem(SAVED_REPOS_KEY) || '[]'
            );
            return Array.isArray(saved)
                ? saved
                      .filter(
                          repo =>
                              typeof repo?.name === 'string' &&
                              typeof repo?.url === 'string'
                      )
                      .slice(0, 32)
                : [];
        } catch {
            return [];
        }
    }

    function saveRepos(repos: SavedRepo[]) {
        try {
            localStorage.setItem(
                SAVED_REPOS_KEY,
                JSON.stringify(repos.slice(0, 32))
            );
        } catch {
            showToast('Could not save repository preferences');
        }
    }

    let urlInput = $state(getURL());
    let previewIndex = $state(-1);
    let previewSrc = $state('');
    let previewSequence = 0;
    onDestroy(() => previewSequence++);
    let savedRepos = $state<SavedRepo[]>(loadSavedRepos());
    let savedReposOpen = $state(false);
    let savedReposRef = $state<HTMLDivElement | null>(null);

    let nameFilter = $state('');
    let scrollEl = $state<HTMLDivElement | null>(null);

    // Start every folder at the top, with no filter from the previous one.
    // A fast response can replace the list before the browser shrinks the
    // scroll area, so the old offset stays unless it is reset here.
    $effect(() => {
        getURL();
        nameFilter = '';
        if (scrollEl) scrollEl.scrollTop = 0;
    });

    let results = $derived(getResults());
    let isLoading = $derived(getIsLoading());
    let error = $derived(getError());
    let filteredResults = $derived(
        nameFilter
            ? results.filter(i =>
                  i.name.toLowerCase().includes(nameFilter.toLowerCase())
              )
            : results
    );
    let fileResults = $derived(filteredResults.filter(i => i.type === 'file'));
    let dirResults = $derived(filteredResults.filter(i => i.type === 'dir'));

    let canGoUp = $derived(getCanGoUp());

    let currentIsSaved = $derived(savedRepos.some(r => r.url === getURL()));

    // Close saved-repos dropdown on outside click
    $effect(() => {
        if (!savedReposOpen || !savedReposRef) return;
        function onPointerDown(e: PointerEvent) {
            if (savedReposRef && !savedReposRef.contains(e.target as Node)) {
                savedReposOpen = false;
            }
        }
        window.addEventListener('pointerdown', onPointerDown);
        return () => window.removeEventListener('pointerdown', onPointerDown);
    });

    function handleSubmit() {
        closePreview();
        setURL(urlInput);
        fetchImages();
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === 'Enter') handleSubmit();
    }

    function handleNavigate(dir: ImageInfo) {
        closePreview();
        storeNavigateToDir(dir.name, dir.htmlURL);
        urlInput = getURL();
    }

    function handleGoUp() {
        closePreview();
        storeGoUp();
        urlInput = getURL();
    }

    function handlePreview(img: ImageInfo) {
        void showPreview(fileResults.findIndex(f => f.path === img.path));
    }

    function closePreview() {
        previewSequence++;
        previewIndex = -1;
        previewSrc = '';
    }

    async function showPreview(index: number) {
        const image = fileResults[index];
        if (!image) return;
        const sequence = ++previewSequence;
        previewIndex = index;
        previewSrc = '';
        try {
            const {DownloadWallpaper} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const path = await DownloadWallpaper(image.url);
            const source = await loadFullImage(path);
            if (
                sequence === previewSequence &&
                fileResults[index]?.url === image.url
            )
                previewSrc = source;
        } catch {
            if (sequence === previewSequence) {
                closePreview();
                showToast('Could not load the wallpaper preview');
            }
        }
    }

    function toggleSaveRepo() {
        const current = getURL();
        if (!current.trim()) return;
        let repos = loadSavedRepos();
        const idx = repos.findIndex(r => r.url === current);
        if (idx >= 0) {
            repos.splice(idx, 1);
            showToast('Removed from saved repos');
        } else {
            let name = current.replace(/\/+$/, '');
            try {
                const parsed = new URL(current);
                const segs = parsed.pathname
                    .replace(/\/+$/, '')
                    .split('/')
                    .filter(Boolean);
                if (segs.length >= 2) {
                    name = `${segs[0]}/${segs[1]}`;
                } else {
                    name = segs[segs.length - 1] || current;
                }
            } catch {}
            repos.push({name, url: current});
            showToast('Repo saved');
        }
        saveRepos(repos);
        savedRepos = repos;
    }

    function loadSavedRepo(url: string) {
        closePreview();
        setURL(url);
        urlInput = url;
        savedReposOpen = false;
        fetchImages();
    }

    function removeSavedRepo(event: MouseEvent, url: string) {
        event.stopPropagation();
        let repos = loadSavedRepos();
        repos = repos.filter(r => r.url !== url);
        saveRepos(repos);
        savedRepos = repos;
    }
</script>

<div class="flex h-full flex-col">
    <ViewHeader>
        <h2 class="text-fg-primary shrink-0 text-[13.5px] font-semibold">
            GitHub
        </h2>
        <span class="text-fg-dimmed mr-2 hidden shrink-0 text-[12px] xl:inline"
            >Wallpapers from public repositories</span
        >
        <button
            class="border-border text-fg-secondary hover:border-border-focus hover:text-fg-primary flex h-8 w-8 shrink-0 items-center justify-center border transition-colors disabled:opacity-35"
            onclick={handleGoUp}
            disabled={!canGoUp}
            title="Go to parent directory"
            aria-label="Go to parent directory"
            ><svg
                class="h-3.5 w-3.5"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"><path d="m6 12 6-6 6 6M12 6v14" /></svg
            ></button
        >
        <!-- Saved repos dropdown -->
        <div bind:this={savedReposRef} class="relative">
            <button
                class="border-border hover:border-border-focus flex h-8 w-8 items-center justify-center border transition-colors {currentIsSaved
                    ? 'text-accent'
                    : 'text-fg-secondary hover:text-fg-primary'}"
                onclick={() => (savedReposOpen = !savedReposOpen)}
                title="Saved repos"
                aria-label="Saved repositories"
                aria-expanded={savedReposOpen}
            >
                <svg
                    class="h-3.5 w-3.5"
                    viewBox="0 0 24 24"
                    fill={currentIsSaved ? 'currentColor' : 'none'}
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <polygon
                        points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"
                    />
                </svg>
            </button>
            {#if savedReposOpen}
                <div
                    class="bg-bg-secondary border-border shadow-(--shadow-panel) absolute left-0 top-full z-50 mt-1 min-w-[240px] border py-1"
                >
                    <div
                        class="flex items-center justify-between px-3 pb-1 pt-1.5"
                    >
                        <span
                            class="text-fg-dimmed text-[10px] font-semibold uppercase tracking-[0.14em]"
                            >Saved repos</span
                        >
                        <button
                            class="text-accent hover:text-accent-hover text-[11px] transition-colors"
                            onclick={toggleSaveRepo}
                            title={currentIsSaved
                                ? 'Remove current from saved'
                                : 'Save current repo'}
                        >
                            {currentIsSaved ? 'Remove' : '+ Save current'}
                        </button>
                    </div>
                    {#if savedRepos.length === 0}
                        <div class="text-fg-dimmed px-3 py-2.5 text-[11.5px]">
                            No saved repos yet
                        </div>
                    {:else}
                        {#each savedRepos as repo}
                            <div
                                class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary flex h-[30px] w-full cursor-pointer items-center gap-2 px-3 text-left text-[12px] transition-colors"
                                role="button"
                                tabindex="0"
                                onclick={() => loadSavedRepo(repo.url)}
                                onkeydown={e => {
                                    if (
                                        e.target === e.currentTarget &&
                                        (e.key === 'Enter' || e.key === ' ')
                                    ) {
                                        e.preventDefault();
                                        loadSavedRepo(repo.url);
                                    }
                                }}
                            >
                                <span class="min-w-0 flex-1 truncate"
                                    >{repo.name}</span
                                >
                                <button
                                    class="text-fg-dimmed hover:text-destructive flex h-5 w-5 shrink-0 items-center justify-center transition-colors"
                                    onclick={e => removeSavedRepo(e, repo.url)}
                                    aria-label="Remove {repo.name}"
                                    title="Remove"
                                    ><svg
                                        class="h-3 w-3"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2.4"
                                        stroke-linecap="round"
                                        aria-hidden="true"
                                        ><path d="M18 6L6 18M6 6l12 12"
                                        ></path></svg
                                    ></button
                                >
                            </div>
                        {/each}
                    {/if}
                </div>
            {/if}
        </div>
        <input
            type="url"
            aria-label="GitHub repository URL"
            bind:value={urlInput}
            placeholder="https://github.com/owner/repo"
            class="bg-bg-primary text-fg-primary border-border focus:border-accent h-8 min-w-[200px] flex-[2] border px-2.5 text-[12px] outline-none transition-colors"
            onkeydown={handleKeydown}
        />
        <button
            class="bg-accent hover:bg-accent-hover text-accent-fg h-8 shrink-0 px-4 text-[12px] font-semibold transition-colors disabled:opacity-50"
            onclick={handleSubmit}
            disabled={isLoading || !urlInput.trim()}
        >
            {isLoading ? 'Loading…' : 'Fetch'}
        </button>
        {#if results.length > 0}
            <input
                type="text"
                bind:value={nameFilter}
                placeholder="Filter by name…"
                aria-label="Filter repository files"
                class="bg-bg-primary text-fg-primary border-border focus:border-accent h-8 min-w-[140px] flex-1 border px-2.5 text-[12px] outline-none transition-colors"
            />
        {/if}
        <div class="ml-auto flex shrink-0 items-center gap-3">
            {#if results.length > 0}
                <span class="text-fg-dimmed text-[11.5px] tabular-nums"
                    >{fileResults.length} / {results.length}{dirResults.length >
                    0
                        ? `, ${dirResults.length} dir${dirResults.length === 1 ? '' : 's'}`
                        : ''}</span
                >
            {/if}
            <CardSizeToggle />
        </div>
    </ViewHeader>

    <div bind:this={scrollEl} class="flex-1 overflow-y-auto p-4">
        {#if isLoading}
            <LoadingState message="Fetching from GitHub…" />
        {:else if error}
            <EmptyState
                title="Failed to load"
                body={error}
                actionLabel="Retry"
                onaction={handleSubmit}
            />
        {:else if filteredResults.length === 0 && results.length === 0}
            <EmptyState
                title="Nothing to show"
                body="Enter a GitHub repository URL above and click Fetch to browse wallpapers and directories."
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
                        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
                        <path
                            d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"
                        ></path>
                    </svg>
                {/snippet}
            </EmptyState>
        {:else if filteredResults.length === 0 && results.length > 0}
            <EmptyState
                title="No images match filter"
                body="Try a different name or clear the filter."
                actionLabel="Clear filter"
                onaction={() => (nameFilter = '')}
            />
        {:else}
            <div
                class="grid gap-3"
                style:grid-template-columns="repeat(auto-fill, minmax({CARD_MIN_WIDTH[
                    getCardSize()
                ]}px, 1fr))"
            >
                {#each filteredResults as item, i (item.path)}
                    {#if item.type === 'dir'}
                        <button
                            class="bg-bg-secondary border-border hover:border-border-focus group flex cursor-pointer flex-col border text-left transition-colors duration-100"
                            onclick={() => handleNavigate(item)}
                            type="button"
                            aria-label="Open directory {item.name}"
                        >
                            <div
                                class="bg-bg-primary text-fg-dimmed group-hover:text-fg-secondary flex aspect-video w-full items-center justify-center transition-colors"
                            >
                                <svg
                                    class="h-9 w-9"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="1.5"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                >
                                    <path
                                        d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                                    />
                                </svg>
                            </div>
                            <div
                                class="flex h-8 w-full items-center gap-2 px-2.5 text-[11px]"
                            >
                                <span
                                    class="text-fg-secondary min-w-0 flex-1 truncate font-mono"
                                    >{item.name}/</span
                                >
                                <span class="text-fg-dimmed shrink-0"
                                    >Folder</span
                                >
                            </div>
                        </button>
                    {:else}
                        <GitHubCard
                            image={item}
                            onpreview={() => handlePreview(item)}
                        />
                    {/if}
                {/each}
            </div>
        {/if}
    </div>
</div>

<ImagePreview
    src={previewSrc}
    alt={previewIndex >= 0 ? fileResults[previewIndex]?.name : ''}
    open={previewIndex >= 0}
    onclose={closePreview}
    hasPrev={previewIndex > 0}
    hasNext={previewIndex < fileResults.length - 1}
    onprev={() => showPreview(previewIndex - 1)}
    onnext={() => showPreview(previewIndex + 1)}
/>
