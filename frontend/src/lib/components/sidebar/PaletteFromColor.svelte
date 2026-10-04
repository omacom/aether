<script lang="ts">
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import {setPalette} from '$lib/stores/theme.svelte';
    import {showToast} from '$lib/stores/ui.svelte';

    let baseColor = $state('#89b4fa');
    let expanded = $state(false);

    async function generate() {
        try {
            const {GeneratePaletteFromColor} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await GeneratePaletteFromColor(baseColor);
            setPalette(result);
            showToast('Generated palette from color');
        } catch {
            showToast('Failed to generate palette');
        }
    }
</script>

<ExpandableSection
    title="Palette from color"
    contentClass="px-4 pb-3.5"
    bind:expanded
>
    <p class="text-fg-dimmed mb-2.5 text-[11.5px] leading-normal">
        Build a full 16-color palette from a single seed color.
    </p>
    <div class="flex items-center gap-2">
        <label
            class="border-border bg-bg-primary focus-within:border-accent flex h-8 min-w-0 flex-1 items-center gap-2 border pl-1 pr-2.5"
        >
            <input
                type="color"
                bind:value={baseColor}
                class="h-6 w-6 shrink-0 cursor-pointer border-none bg-transparent p-0"
                title="Base color"
                aria-label="Base color"
            />
            <span class="text-fg-secondary truncate font-mono text-[11px]"
                >{baseColor.toUpperCase()}</span
            >
        </label>
        <button
            type="button"
            class="bg-accent text-accent-fg hover:bg-accent-hover h-8 shrink-0 px-3.5 text-[12px] font-semibold transition-colors"
            onclick={generate}
        >
            Generate
        </button>
    </div>
</ExpandableSection>
