<script lang="ts">
    // Shortcut hint that sits on a filled accent button. It inherits the
    // button text color, so it stays legible on any accent.
    let {children}: {children: import('svelte').Snippet} = $props();
</script>

<kbd
    class="inline-flex items-center border border-current px-[5px] py-px font-mono text-[10px] font-medium leading-[1.3] opacity-55"
    aria-hidden="true"
>
    {@render children()}
</kbd>
