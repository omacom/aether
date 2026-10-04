<script lang="ts">
    import {
        getExtractionMode,
        getPendingExtractionMode,
        getWallpaperPath,
        getLightMode,
    } from '$lib/stores/theme.svelte';
    import {extractColors} from '$lib/actions/themeActions';
    import {
        EXTRACTION_MODES,
        EXTRACTION_MODE_GROUPS,
        type ExtractionMode,
        type ExtractionModeGroup,
    } from '$lib/constants/colors';
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';

    let expanded = $state(true);
    let selectedMode = $derived(
        getPendingExtractionMode() ?? getExtractionMode()
    );

    const openGroups: Record<ExtractionModeGroup, boolean> = $state(
        Object.fromEntries(
            EXTRACTION_MODE_GROUPS.map(g => [g.id, g.defaultOpen])
        ) as Record<ExtractionModeGroup, boolean>
    );

    const grouped = $derived.by(() => {
        const out: Record<string, ExtractionMode[]> = {};
        for (const g of EXTRACTION_MODE_GROUPS) out[g.id] = [];
        for (const m of EXTRACTION_MODES) out[m.group].push(m);
        return out;
    });

    let selectedLabel = $derived(
        EXTRACTION_MODES.find(m => m.value === selectedMode)?.label ?? ''
    );

    function activeInGroup(groupId: ExtractionModeGroup) {
        return grouped[groupId]?.find(m => m.value === selectedMode);
    }

    // Per-mode palette previews keyed by `${path}:${lightMode}:${mode}`.
    // Populated lazily by a serial prefetch keyed to the current wallpaper.
    let stripCache = $state<Record<string, string[]>>({});
    let prefetchToken = 0;

    function stripKey(path: string, lm: boolean, mode: string): string {
        return `${path}:${lm ? 'L' : 'D'}:${mode}`;
    }

    async function prefetchStrips(path: string, lm: boolean) {
        if (!path) return;
        const myToken = ++prefetchToken;
        try {
            const {PreviewExtractColors} = await import(
                '../../../../wailsjs/go/main/App'
            );
            for (const mode of EXTRACTION_MODES) {
                if (myToken !== prefetchToken) return;
                const key = stripKey(path, lm, mode.value);
                if (stripCache[key]) continue;
                try {
                    const colors = await PreviewExtractColors(
                        path,
                        lm,
                        mode.value
                    );
                    if (myToken !== prefetchToken) return;
                    if (Array.isArray(colors) && colors.length >= 8) {
                        stripCache = {...stripCache, [key]: colors};
                    }
                } catch {
                    // Leave the strip empty for this mode.
                }
            }
        } catch {}
    }

    $effect(() => {
        const path = getWallpaperPath();
        const lm = getLightMode();
        prefetchStrips(path, lm);
        return () => {
            prefetchToken++;
        };
    });

    function getStrip(mode: string): string[] | null {
        const key = stripKey(getWallpaperPath(), getLightMode(), mode);
        return stripCache[key] || null;
    }

    function handleModeChange(mode: string) {
        if (mode === selectedMode) return;
        void extractColors({mode});
    }
</script>

{#snippet modeList(items: ExtractionMode[])}
    <ul class="flex flex-col gap-px">
        {#each items as mode}
            {@const isActive = selectedMode === mode.value}
            {@const strip = getStrip(mode.value)}
            <li>
                <button
                    type="button"
                    onclick={() => handleModeChange(mode.value)}
                    title={mode.description}
                    aria-pressed={isActive}
                    class="flex h-[30px] w-full items-center gap-2.5 pl-2.5 pr-2 text-left text-[12px] transition-colors duration-100 {isActive
                        ? 'bg-bg-elevated text-fg-primary shadow-[inset_2px_0_0_var(--color-accent)]'
                        : 'text-fg-secondary hover:bg-bg-hover hover:text-fg-primary'}"
                >
                    <span class="min-w-0 flex-1 truncate">{mode.label}</span>
                    <span
                        class="outline-border flex h-2 w-14 shrink-0 overflow-hidden outline outline-1"
                        aria-hidden="true"
                    >
                        {#if strip}
                            {#each [0, 1, 2, 3, 4, 5, 6, 7] as i}
                                <span
                                    class="flex-1"
                                    style:background-color={strip[i]}
                                ></span>
                            {/each}
                        {/if}
                    </span>
                </button>
            </li>
        {/each}
    </ul>
{/snippet}

<ExpandableSection
    title="Extraction mode"
    suffix={selectedLabel}
    contentClass="px-2.5 pb-3"
    bind:expanded
>
    <div class="flex flex-col gap-px">
        {#if grouped.auto?.length}
            {@render modeList(grouped.auto)}
        {/if}

        {#each EXTRACTION_MODE_GROUPS as group}
            {#if group.id !== 'auto'}
                {@const items = grouped[group.id]}
                {#if items?.length}
                    {@const isOpen = openGroups[group.id]}
                    {@const active = activeInGroup(group.id)}
                    <ExpandableSection
                        variant="group"
                        title={group.label}
                        suffix={!isOpen && active
                            ? active.label
                            : String(items.length)}
                        suffixAccent={!isOpen && !!active}
                        bind:expanded={openGroups[group.id]}
                    >
                        {@render modeList(items)}
                    </ExpandableSection>
                {/if}
            {/if}
        {/each}
    </div>
</ExpandableSection>
