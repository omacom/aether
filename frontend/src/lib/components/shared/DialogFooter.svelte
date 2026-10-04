<script lang="ts">
    import type {Snippet} from 'svelte';

    // Action row at the bottom of a bare `Modal`. Buttons align right.
    let {children}: {children: Snippet} = $props();
</script>

<div
    class="border-border bg-bg-surface flex items-center justify-end gap-2 border-t px-5 py-3"
>
    {@render children()}
</div>
