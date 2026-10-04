<script lang="ts">
    let {message}: {message: string} = $props();
</script>

<div
    class="text-fg-dimmed flex h-full min-h-32 items-center justify-center gap-2.5 text-[12px]"
    role="status"
>
    <span class="bg-accent h-1.5 w-1.5 animate-pulse" aria-hidden="true"></span>
    {message}
</div>
