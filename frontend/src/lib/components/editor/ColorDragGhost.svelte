<script lang="ts">
    import {getColorDrag} from '$lib/stores/ui.svelte';

    let drag = $derived(getColorDrag());
    let active = $derived(!!drag);

    $effect(() => {
        if (!active) return;
        const previous = document.body.style.cursor;
        document.body.style.cursor = 'copy';
        return () => {
            if (document.body.style.cursor === 'copy')
                document.body.style.cursor = previous;
        };
    });
</script>

{#if drag}
    <div
        class="pointer-events-none fixed z-[9999] h-7 w-7 border-2 border-white shadow-lg"
        style:background-color={drag.color}
        style:left="{drag.x + 12}px"
        style:top="{drag.y + 12}px"
    ></div>
{/if}
