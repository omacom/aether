<script lang="ts">
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import {setPalette} from '$lib/stores/theme.svelte';
    import {showToast} from '$lib/stores/ui.svelte';
    import {PRESET_THEMES} from '$lib/constants/colors';

    let expanded = $state(false);
    const presetNames = Object.keys(PRESET_THEMES);

    function applyPreset(name: string) {
        setPalette(PRESET_THEMES[name]);
        showToast(`Applied ${name} preset`);
    }
</script>

<ExpandableSection
    title="Presets"
    suffix={String(presetNames.length)}
    contentClass="px-4 pb-3.5"
    bind:expanded
>
    <div class="grid grid-cols-2 gap-1.5">
        {#each presetNames as name}
            <button
                type="button"
                class="bg-bg-surface border-border hover:border-accent flex flex-col gap-1.5 border p-[7px] text-left transition-colors duration-100"
                onclick={() => applyPreset(name)}
                title="Apply the {name} preset"
            >
                <span class="text-fg-secondary truncate text-[11px]"
                    >{name}</span
                >
                <span class="flex h-2.5 w-full" aria-hidden="true">
                    {#each PRESET_THEMES[name].slice(0, 8) as color}
                        <span class="flex-1" style:background-color={color}
                        ></span>
                    {/each}
                </span>
            </button>
        {/each}
    </div>
</ExpandableSection>
