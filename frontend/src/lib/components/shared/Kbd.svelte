<script lang="ts">
    let {
        children,
        class: className = '',
    }: {children: import('svelte').Snippet; class?: string} = $props();
</script>

<kbd
    class="border-border bg-bg-secondary text-fg-secondary inline-flex items-center border px-[5px] py-px font-mono text-[10px] font-medium leading-[1.3] {className}"
>
    {@render children()}
</kbd>
