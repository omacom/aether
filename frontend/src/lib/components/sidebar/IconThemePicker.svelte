<script lang="ts">
    import {onMount} from 'svelte';
    import Modal from '$lib/components/shared/Modal.svelte';
    import DialogHeader from '$lib/components/shared/DialogHeader.svelte';
    import Switch from '$lib/components/shared/Switch.svelte';
    import IconThemePreview from './IconThemePreview.svelte';
    import {getIconTheme, setIconTheme} from '$lib/stores/theme.svelte';
    import type {IconThemeSelection} from '$lib/types/theme';
    import {
        isAppIncluded,
        toggleAppInclusion,
    } from '$lib/stores/settings.svelte';
    import type {icontheme} from '../../../../wailsjs/go/models';

    type ThemeSummary = icontheme.ThemeSummary;

    let open = $state(false);
    let query = $state('');
    let themes = $state<ThemeSummary[]>([]);
    let loaded = $state(false);
    let loading = $state(false);
    let error = $state('');
    let catalogRevision = $state(0);
    let searchInput = $state<HTMLInputElement | null>(null);

    let selection = $derived(getIconTheme());
    let enabled = $derived(isAppIncluded('icons'));
    let selectedTheme = $derived(
        selection.mode === 'explicit'
            ? themes.find(theme => theme.id === selection.id)
            : undefined
    );
    let missing = $derived(
        loaded && selection.mode === 'explicit' && !selectedTheme
    );
    let summary = $derived(
        selection.mode === 'automatic'
            ? 'Automatic · Yaru match'
            : selectedTheme
              ? selectedTheme.name
              : missing
                ? `${selection.id} · Missing`
                : selection.id
    );
    // The picker button shows the theme name and a dimmed detail.
    let summaryName = $derived(
        selection.mode === 'automatic'
            ? 'Automatic'
            : (selectedTheme?.name ?? selection.id)
    );
    let summaryDetail = $derived(
        selection.mode === 'automatic'
            ? 'Yaru, palette-matched'
            : missing
              ? 'Missing'
              : selectedTheme
                ? selectedTheme.origin === 'user'
                    ? 'User'
                    : 'System'
                : ''
    );
    let filteredThemes = $derived.by(() => {
        const needle = query.trim().toLocaleLowerCase();
        if (!needle) return themes;
        return themes.filter(
            theme =>
                theme.name.toLocaleLowerCase().includes(needle) ||
                theme.id.toLocaleLowerCase().includes(needle)
        );
    });

    $effect(() => {
        if (open) queueMicrotask(() => searchInput?.focus());
    });

    async function loadThemes(refresh = false) {
        if (loading) return;
        loading = true;
        error = '';
        try {
            const api = await import('../../../../wailsjs/go/main/App');
            const result = refresh
                ? await api.RefreshInstalledIconThemes()
                : await api.ListInstalledIconThemes();
            themes = Array.isArray(result) ? result : [];
            catalogRevision += 1;
            loaded = true;
        } catch {
            error = refresh
                ? 'Could not refresh installed icon themes. Try again.'
                : 'Could not load installed icon themes. Try again.';
        } finally {
            loading = false;
        }
    }

    function showPicker() {
        open = true;
        if (!loaded) loadThemes();
    }

    function choose(next: IconThemeSelection) {
        setIconTheme(next);
        open = false;
    }

    onMount(() => {
        loadThemes();
    });
</script>

<section class="border-border flex flex-col gap-[9px] border-b px-4 py-3">
    <div class="flex items-center gap-2">
        <span class="text-fg-primary flex-1 text-[12.5px] font-medium"
            >Desktop icons</span
        >
        <Switch
            checked={enabled}
            onchange={() => toggleAppInclusion('icons')}
            label="Toggle Icons"
            title={enabled
                ? 'Icons are themed on apply'
                : 'Icons are not themed on apply'}
        />
    </div>
    <button
        type="button"
        class="border-border bg-bg-surface enabled:hover:border-border-focus flex h-[30px] min-w-0 items-center gap-2 border px-2.5 text-left text-[12px] transition-colors disabled:cursor-default disabled:opacity-45"
        class:border-warning={missing}
        onclick={showPicker}
        disabled={!enabled}
        aria-label={`Choose icon theme, current: ${summary}`}
        title={enabled
            ? 'Choose an installed desktop icon theme'
            : 'Enable Icons to choose a theme'}
    >
        <span class="text-fg-primary min-w-0 flex-1 truncate"
            >{summaryName}</span
        >
        {#if summaryDetail}
            <span
                class="shrink-0 truncate text-[11px]"
                class:text-warning={missing}
                class:text-fg-dimmed={!missing}>{summaryDetail}</span
            >
        {/if}
        <svg
            class="text-fg-dimmed h-3 w-3 shrink-0"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg
        >
    </button>
</section>

<Modal
    {open}
    onclose={() => (open = false)}
    bare
    label="Choose icon theme"
    panelClass="w-[560px] max-h-[82vh] flex flex-col"
>
    <DialogHeader title="Choose icon theme" onclose={() => (open = false)} />

    <div class="flex gap-2 px-5 pb-3 pt-4">
        <input
            bind:this={searchInput}
            bind:value={query}
            type="search"
            class="bg-bg-primary border-border text-fg-primary focus:border-accent h-[34px] min-w-0 flex-1 border px-2.5 text-[13px] outline-none"
            placeholder="Search installed themes…"
            aria-label="Search installed icon themes"
        />
        <button
            type="button"
            class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-[34px] border px-3.5 text-[12px] font-medium transition-colors disabled:opacity-50"
            onclick={() => loadThemes(true)}
            disabled={loading}
        >
            {loading ? 'Refreshing…' : 'Refresh'}
        </button>
    </div>

    {#if error}
        <p class="text-warning px-5 pb-2 text-[11px]" role="status">
            {error}
        </p>
    {/if}

    <div
        class="border-border mx-5 mb-5 min-h-0 flex-1 overflow-y-auto border"
        role="group"
        aria-label="Icon themes"
    >
        <button
            type="button"
            aria-pressed={selection.mode === 'automatic'}
            aria-label="Use automatic icon theme"
            class="border-border hover:bg-bg-hover flex w-full items-start gap-2.5 border-b px-3 py-2.5 text-left transition-colors"
            class:bg-accent-muted={selection.mode === 'automatic'}
            onclick={() => choose({mode: 'automatic'})}
        >
            <span
                class="border-border mt-0.5 h-3 w-3 shrink-0 border"
                class:bg-accent={selection.mode === 'automatic'}
                aria-hidden="true"
            ></span>
            <span>
                <span class="text-fg-primary block text-[12px] font-medium"
                    >Automatic · Color-matched Yaru</span
                >
                <span class="text-fg-dimmed mt-0.5 block text-[11px]">
                    Uses Aether’s palette-derived Yaru variant
                </span>
            </span>
        </button>

        {#if missing && selection.mode === 'explicit'}
            <button
                type="button"
                aria-pressed="true"
                aria-label="Keep missing icon theme {selection.id}"
                class="border-border bg-accent-muted hover:bg-bg-hover flex w-full items-start gap-2.5 border-b px-3 py-2.5 text-left transition-colors"
                onclick={() => (open = false)}
            >
                <span
                    class="bg-warning mt-0.5 h-3 w-3 shrink-0"
                    aria-hidden="true"
                ></span>
                <span>
                    <span class="text-warning block text-[12px] font-medium"
                        >{selection.id} · Missing</span
                    >
                    <span class="text-fg-dimmed mt-0.5 block text-[11px]">
                        This icon theme is not installed. Aether preserves its
                        ID.
                    </span>
                </span>
            </button>
        {/if}

        {#if loading && !loaded}
            <p
                class="text-fg-dimmed px-3 py-6 text-center text-[11.5px]"
                role="status"
            >
                Loading installed icon themes…
            </p>
        {:else if loaded && themes.length === 0}
            <p class="text-fg-dimmed px-3 py-6 text-center text-[11.5px]">
                No installed icon themes were found. Automatic Yaru is still
                available.
            </p>
        {:else if filteredThemes.length === 0}
            <p class="text-fg-dimmed px-3 py-6 text-center text-[11.5px]">
                No installed themes match this search.
            </p>
        {:else}
            {#each filteredThemes as theme (theme.id + ':' + catalogRevision)}
                <button
                    type="button"
                    aria-pressed={selection.mode === 'explicit' &&
                        selection.id === theme.id}
                    aria-label="Use icon theme {theme.name}"
                    class="border-border hover:bg-bg-hover flex w-full items-start gap-2.5 border-b px-3 py-2.5 text-left transition-colors last:border-b-0"
                    class:bg-accent-muted={selection.mode === 'explicit' &&
                        selection.id === theme.id}
                    onclick={() => choose({mode: 'explicit', id: theme.id})}
                >
                    <span
                        class="border-border mt-0.5 h-3 w-3 shrink-0 border"
                        class:bg-accent={selection.mode === 'explicit' &&
                            selection.id === theme.id}
                        aria-hidden="true"
                    ></span>
                    <span class="min-w-0 flex-1">
                        <span class="flex items-baseline gap-2">
                            <span
                                class="text-fg-primary truncate text-[12px] font-medium"
                                >{theme.name}</span
                            >
                            <span
                                class="text-fg-dimmed ml-auto shrink-0 text-[10px] font-semibold uppercase tracking-[0.12em]"
                            >
                                {theme.origin === 'user' ? 'User' : 'System'}
                            </span>
                        </span>
                        {#if theme.id.toLocaleLowerCase() !== theme.name.toLocaleLowerCase()}
                            <span
                                class="text-fg-dimmed mt-0.5 block truncate font-mono text-[10px]"
                            >
                                {theme.id}
                            </span>
                        {/if}
                        <IconThemePreview
                            themeId={theme.id}
                            themeName={theme.name}
                            hasPreview={theme.hasPreview}
                        />
                    </span>
                </button>
            {/each}
        {/if}
    </div>
</Modal>
