<script lang="ts">
    import {hslToHex} from '$lib/utils/color';

    // One row of tones: the current hue and saturation at 11 lightness
    // steps. The tone nearest to the current lightness gets a ring.
    let {
        hue,
        saturation,
        lightness,
        disabled = false,
        onselect,
    }: {
        hue: number;
        saturation: number;
        lightness: number;
        disabled?: boolean;
        onselect: (color: string) => void;
    } = $props();

    const STEPS = 11;
    const L_MIN = 8;
    const L_STEP = 8.4;

    let tones = $derived(
        Array.from({length: STEPS}, (_, i) => {
            const l = L_MIN + i * L_STEP;
            return {
                hex: hslToHex(hue, saturation, l),
                current: Math.abs(l - lightness) < L_STEP / 2,
            };
        })
    );
</script>

<div class="flex gap-[2px]">
    {#each tones as tone}
        <button
            type="button"
            class="h-[26px] flex-1 transition-transform hover:-translate-y-px disabled:opacity-40
                {tone.current
                ? 'shadow-[inset_0_0_0_2px_var(--color-fg-primary)]'
                : ''}"
            style:background-color={tone.hex}
            onclick={() => onselect(tone.hex)}
            {disabled}
            title={tone.hex}
            aria-label="Apply {tone.hex}"
        ></button>
    {/each}
</div>
