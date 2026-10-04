<script lang="ts">
    import type {Snippet} from 'svelte';

    // Shared toolbar shell pinned to the top of a tab view (Local, Favorites,
    // Blueprints, System). Fixes the surface, bottom border, padding, and
    // wrap behaviour so every tab has an identical top edge; callers fill it
    // with a title, filters, result counts, or actions.
    let {children}: {children: Snippet} = $props();
</script>

<div
    class="bg-bg-secondary border-border flex min-h-12 shrink-0 flex-wrap items-center gap-2.5 border-b px-4 py-2"
>
    {@render children()}
</div>
