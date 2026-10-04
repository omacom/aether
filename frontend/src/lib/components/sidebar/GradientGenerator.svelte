<script lang="ts">
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import {setPalette} from '$lib/stores/theme.svelte';
    import {showToast} from '$lib/stores/ui.svelte';

    let startColor = $state('#1e1e2e');
    let endColor = $state('#89b4fa');
    let preview = $state<string[]>([]);
    let expanded = $state(false);

    async function generatePreview() {
        try {
            const {GenerateGradient} = await import(
                '../../../../wailsjs/go/main/App'
            );
            preview = await GenerateGradient(startColor, endColor);
        } catch {
            preview = Array.from({length: 16}, (_, i) => {
                const r = i / 15;
                return `hsl(${220 + r * 40}, 50%, ${10 + r * 80}%)`;
            });
        }
    }

    function applyGradient() {
        if (preview.length === 16) {
            setPalette(preview);
            showToast('Applied gradient palette');
        }
    }
</script>

{#snippet colorField(label: string, value: string, set: (v: string) => void)}
    <label
        class="border-border bg-bg-primary focus-within:border-accent flex h-8 min-w-0 flex-1 items-center gap-1.5 border pl-1 pr-2"
    >
        <input
            type="color"
            {value}
            oninput={e => set(e.currentTarget.value)}
            class="h-6 w-6 shrink-0 cursor-pointer border-none bg-transparent p-0"
            title={label}
            aria-label={label}
        />
        <span class="text-fg-secondary truncate font-mono text-[10.5px]"
            >{value.toUpperCase()}</span
        >
    </label>
{/snippet}

<ExpandableSection title="Gradient" contentClass="px-4 pb-3.5" bind:expanded>
    <p class="text-fg-dimmed mb-2.5 text-[11.5px] leading-normal">
        Generate a palette as a smooth ramp between two colors.
    </p>
    <div class="flex flex-col gap-2">
        <div class="flex items-center gap-1.5">
            {@render colorField(
                'Gradient start color',
                startColor,
                v => (startColor = v)
            )}
            <span class="text-fg-dimmed text-[11px]" aria-hidden="true"
                >&rarr;</span
            >
            {@render colorField(
                'Gradient end color',
                endColor,
                v => (endColor = v)
            )}
        </div>

        {#if preview.length > 0}
            <div class="outline-border flex h-5 outline outline-1">
                {#each preview as color}
                    <div class="flex-1" style:background-color={color}></div>
                {/each}
            </div>
        {/if}
        <div class="flex gap-1.5">
            <button
                type="button"
                class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 flex-1 border px-3.5 text-[12px] font-medium transition-colors"
                onclick={generatePreview}>Preview</button
            >
            {#if preview.length > 0}
                <button
                    type="button"
                    class="bg-accent text-accent-fg hover:bg-accent-hover h-8 flex-1 px-3.5 text-[12px] font-semibold transition-colors"
                    onclick={applyGradient}>Apply</button
                >
            {/if}
        </div>
    </div>
</ExpandableSection>
