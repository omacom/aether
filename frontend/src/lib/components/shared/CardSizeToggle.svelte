<script lang="ts">
    import Segmented from './Segmented.svelte';
    import {
        getCardSize,
        setCardSize,
        type CardSize,
    } from '$lib/stores/cardsize.svelte';

    // Shared S / M / L segmented control for the grid card size. Writes the
    // single shared preference, so every grid tab stays in sync.
    const sizeOptions: {value: CardSize; label: string; title: string}[] = [
        {value: 'small', label: 'S', title: 'Small cards'},
        {value: 'medium', label: 'M', title: 'Medium cards'},
        {value: 'large', label: 'L', title: 'Large cards'},
    ];
</script>

<Segmented
    options={sizeOptions}
    value={getCardSize()}
    onchange={setCardSize}
    size="sm"
    mono
    label="Card size"
    itemClass="w-[26px] !px-0"
/>
