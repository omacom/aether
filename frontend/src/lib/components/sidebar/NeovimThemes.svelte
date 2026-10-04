<script lang="ts">
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import {
        getSettings,
        setAppIncluded,
        updateSettings,
    } from '$lib/stores/settings.svelte';
    import {showToast} from '$lib/stores/ui.svelte';
    import {NEOVIM_PRESETS} from '$lib/constants/neovim-presets';

    let expanded = $state(false);
    let selected = $state('');

    $effect(() => {
        selected = getSettings().selectedNeovimConfig ? findSelectedName() : '';
    });

    function findSelectedName(): string {
        const current = getSettings().selectedNeovimConfig;
        const preset = NEOVIM_PRESETS.find(p => p.config === current);
        return preset?.name || '';
    }

    function handleSelect(preset: (typeof NEOVIM_PRESETS)[0]) {
        updateSettings({selectedNeovimConfig: preset.config});
        setAppIncluded('neovim', true);
        selected = preset.name;
        showToast(`Neovim theme: ${preset.name}`);
    }

    function handleClear() {
        updateSettings({selectedNeovimConfig: ''});
        selected = '';
        showToast('Neovim theme cleared (using default)');
    }
</script>

<ExpandableSection
    title="Neovim theme"
    suffix={selected || 'Default'}
    contentClass="px-2.5 pb-3"
    bind:expanded
>
    <p class="text-fg-dimmed mb-2 px-1.5 text-[11.5px] leading-normal">
        Choose which Neovim colorscheme is written with the theme.
    </p>
    <div class="flex max-h-56 flex-col gap-px overflow-y-auto">
        {@render option('Default (template)', '', !selected, handleClear)}
        {#each NEOVIM_PRESETS as preset}
            {@render option(
                preset.name,
                preset.author,
                selected === preset.name,
                () => handleSelect(preset)
            )}
        {/each}
    </div>
</ExpandableSection>

{#snippet option(
    name: string,
    author: string,
    active: boolean,
    onpick: () => void
)}
    <button
        type="button"
        class="flex h-7 w-full shrink-0 items-center gap-2 pl-2.5 pr-2 text-left text-[12px] transition-colors duration-100 {active
            ? 'bg-bg-elevated text-fg-primary shadow-[inset_2px_0_0_var(--color-accent)]'
            : 'text-fg-secondary hover:bg-bg-hover hover:text-fg-primary'}"
        aria-pressed={active}
        onclick={onpick}
    >
        <span class="min-w-0 flex-1 truncate">{name}</span>
        {#if author}
            <span class="text-fg-dimmed shrink-0 truncate text-[11px]"
                >{author}</span
            >
        {/if}
    </button>
{/snippet}
