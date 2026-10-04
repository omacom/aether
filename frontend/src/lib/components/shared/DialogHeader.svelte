<script lang="ts">
    import CloseIcon from './CloseIcon.svelte';

    // Title row for a bare `Modal`. Pair it with `DialogFooter`.
    let {title, onclose}: {title: string; onclose?: () => void} = $props();
</script>

<div class="border-border flex items-center gap-2 border-b py-3.5 pl-5 pr-3">
    <h3 class="text-fg-primary flex-1 text-[14px] font-semibold">{title}</h3>
    {#if onclose}
        <button
            type="button"
            class="text-fg-dimmed hover:text-fg-primary hover:bg-bg-hover flex h-7 w-7 items-center justify-center transition-colors"
            onclick={onclose}
            aria-label="Close"
            title="Close"
        >
            <CloseIcon size="h-3.5 w-3.5" />
        </button>
    {/if}
</div>
