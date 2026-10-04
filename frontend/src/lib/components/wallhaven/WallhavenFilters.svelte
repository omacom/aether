<script lang="ts">
    import {
        getQuery,
        setQuery,
        getCategories,
        toggleCategory,
        getPurity,
        togglePurity,
        getSorting,
        setSorting,
        getOrder,
        setOrder,
        getAtleast,
        setAtleast,
        getColorFilter,
        setColorFilter,
        getApiKey,
        setApiKey,
        getTotalResults,
        search,
    } from '$lib/stores/wallhaven.svelte';
    import SearchIcon from '$lib/components/shared/SearchIcon.svelte';

    let showAdvanced = $state(false);

    function handleSubmit(e: Event) {
        e.preventDefault();
        search();
    }

    const categoryLabels = ['General', 'Anime', 'People'];
    const purityLabels = ['SFW', 'Sketchy', 'NSFW'];
    // Active purity chips take the color of their content rating.
    const purityActive = [
        'text-success border-success bg-success/14',
        'text-warning border-warning bg-warning/14',
        'text-destructive border-destructive bg-destructive/14',
    ];
    const CHIP = 'h-6 border px-[9px] text-[11.5px] transition-colors';
    const CHIP_OFF = 'text-fg-dimmed border-border hover:text-fg-secondary';
    const colorPresets = [
        {hex: '660000', label: 'Red'},
        {hex: 'cc6633', label: 'Orange'},
        {hex: 'ffcc00', label: 'Yellow'},
        {hex: '006600', label: 'Green'},
        {hex: '336699', label: 'Blue'},
        {hex: '660066', label: 'Purple'},
        {hex: '000000', label: 'Black'},
        {hex: 'cccccc', label: 'Gray'},
        {hex: 'ffffff', label: 'White'},
    ];
</script>

{#snippet chevron(size: string)}
    <svg
        class="text-fg-dimmed pointer-events-none absolute top-1/2 -translate-y-1/2 {size}"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"><path d="m6 9 6 6 6-6"></path></svg
    >
{/snippet}

<form
    class="bg-bg-secondary border-border flex shrink-0 flex-col gap-2.5 border-b px-4 py-3"
    onsubmit={handleSubmit}
>
    <!-- Row 1: search and sort -->
    <div class="flex items-center gap-2">
        <label
            class="bg-bg-primary border-border focus-within:border-accent text-fg-dimmed flex h-[34px] min-w-[160px] flex-1 items-center gap-2 border px-2.5 transition-colors"
        >
            <SearchIcon size="h-3.5 w-3.5" />
            <input
                type="text"
                class="text-fg-primary min-w-0 flex-1 bg-transparent text-[13px] outline-none"
                placeholder="Search wallpapers…"
                aria-label="Search wallpapers"
                value={getQuery()}
                oninput={e => setQuery(e.currentTarget.value)}
            />
        </label>

        <div class="relative">
            <select
                class="!bg-bg-primary !border-border text-fg-secondary focus:!border-accent h-[34px] min-w-[112px] border pl-2.5 pr-8 text-[12px] outline-none"
                value={getSorting()}
                onchange={e => setSorting(e.currentTarget.value)}
                title="Sort order"
                aria-label="Sort order"
            >
                <option value="date_added">Latest</option>
                <option value="relevance">Relevance</option>
                <option value="random">Random</option>
                <option value="views">Views</option>
                <option value="favorites">Favorites</option>
                <option value="toplist">Top List</option>
            </select>
            {@render chevron('right-2.5 h-3 w-3')}
        </div>

        <button
            type="button"
            class="border-border bg-bg-primary text-fg-secondary hover:border-border-focus hover:text-fg-primary flex h-[34px] w-[34px] shrink-0 items-center justify-center border text-[14px] transition-colors"
            onclick={() => setOrder(getOrder() === 'desc' ? 'asc' : 'desc')}
            title={getOrder() === 'desc' ? 'Descending' : 'Ascending'}
            aria-label="Toggle sort direction"
        >
            {getOrder() === 'desc' ? '↓' : '↑'}
        </button>

        <button
            type="submit"
            class="bg-accent hover:bg-accent-hover text-accent-fg h-[34px] shrink-0 px-[18px] text-[12px] font-semibold transition-colors"
            >Search</button
        >
    </div>

    <!-- Row 2: filter chips and result count -->
    <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
        <div class="flex items-center gap-1" title="Category">
            {#each categoryLabels as label, i}
                <button
                    type="button"
                    class="{CHIP} {getCategories()[i] === '1'
                        ? 'text-accent border-accent bg-accent-muted'
                        : CHIP_OFF}"
                    onclick={() => toggleCategory(i)}
                    aria-pressed={getCategories()[i] === '1'}>{label}</button
                >
            {/each}
        </div>

        <span class="bg-border h-4 w-px"></span>

        <div class="flex items-center gap-1" title="Purity">
            {#each purityLabels as label, i}
                {@const locked = i === 2 && !getApiKey()}
                <button
                    type="button"
                    class="{CHIP} {locked
                        ? 'text-fg-dimmed/50 border-border cursor-not-allowed'
                        : getPurity()[i] === '1'
                          ? purityActive[i]
                          : CHIP_OFF}"
                    onclick={() => togglePurity(i)}
                    aria-pressed={getPurity()[i] === '1'}
                    title={locked ? 'NSFW requires an API key' : ''}
                    >{label}</button
                >
            {/each}
        </div>

        <span class="bg-border h-4 w-px"></span>

        <div class="relative">
            <select
                class="!border-border text-fg-secondary hover:!border-border-focus focus:!border-accent h-6 border !bg-transparent pl-2 pr-7 text-[11.5px] outline-none transition-colors"
                value={getAtleast()}
                onchange={e => setAtleast(e.currentTarget.value)}
                title="Minimum resolution"
                aria-label="Minimum resolution"
            >
                <option value="">Any resolution</option>
                <option value="1920x1080">1920×1080+</option>
                <option value="2560x1440">2560×1440+</option>
                <option value="3840x2160">3840×2160+</option>
            </select>
            {@render chevron('right-2 h-[11px] w-[11px]')}
        </div>

        <div class="ml-auto flex items-center gap-3">
            {#if getTotalResults() > 0}
                <span class="text-fg-dimmed text-[11.5px] tabular-nums"
                    >{getTotalResults().toLocaleString()} results</span
                >
            {/if}
            <button
                type="button"
                class="text-accent hover:text-accent-hover text-[11.5px] transition-colors"
                onclick={() => (showAdvanced = !showAdvanced)}
                aria-expanded={showAdvanced}
                >{showAdvanced ? 'Fewer filters' : 'More filters'}</button
            >
        </div>
    </div>

    {#if showAdvanced}
        <div
            class="border-border -mx-4 flex flex-wrap items-center gap-x-6 gap-y-2 border-t px-4 pt-3"
        >
            <div class="flex items-center gap-1.5">
                <span class="text-fg-dimmed mr-1 text-[11.5px]">Color</span>
                {#each colorPresets as cp}
                    <button
                        type="button"
                        class="h-5 w-5 border transition-shadow
                {getColorFilter() === cp.hex
                            ? 'border-fg-primary ring-accent ring-1'
                            : 'border-border hover:border-fg-dimmed'}"
                        style:background-color="#{cp.hex}"
                        onclick={() =>
                            setColorFilter(
                                getColorFilter() === cp.hex ? '' : cp.hex
                            )}
                        title={cp.label}
                        aria-label="Filter by {cp.label}"
                        aria-pressed={getColorFilter() === cp.hex}
                    ></button>
                {/each}
                {#if getColorFilter()}
                    <button
                        type="button"
                        class="text-fg-dimmed hover:text-fg-secondary ml-1 text-[11.5px] transition-colors"
                        onclick={() => setColorFilter('')}>Clear</button
                    >
                {/if}
            </div>

            <label class="flex min-w-[240px] flex-1 items-center gap-2">
                <span
                    class="text-fg-dimmed shrink-0 text-[11.5px]"
                    title="Required for NSFW and larger result pages"
                >
                    API key
                </span>
                <input
                    type="password"
                    class="bg-bg-primary border-border text-fg-primary focus:border-accent h-7 min-w-0 flex-1 border px-2.5 text-[12px] outline-none transition-colors"
                    placeholder="wallhaven API key…"
                    value={getApiKey()}
                    oninput={e => setApiKey(e.currentTarget.value)}
                />
            </label>
        </div>
    {/if}
</form>
