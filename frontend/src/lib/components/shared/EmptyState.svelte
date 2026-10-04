<script lang="ts">
    import type {Snippet} from 'svelte';

    let {
        icon,
        title,
        body,
        actionLabel,
        onaction,
        secondaryLabel,
        onsecondary,
    }: {
        icon?: Snippet;
        title: string;
        body?: string;
        actionLabel?: string;
        onaction?: () => void;
        secondaryLabel?: string;
        onsecondary?: () => void;
    } = $props();
</script>

<div class="flex h-full flex-col items-center justify-center p-10 text-center">
    {#if icon}
        <div
            class="border-border bg-bg-secondary text-fg-dimmed mb-4 flex h-14 w-14 items-center justify-center border"
        >
            {@render icon()}
        </div>
    {/if}
    <h3 class="text-fg-primary mb-1.5 text-[15px] font-semibold">{title}</h3>
    {#if body}
        <p
            class="text-fg-secondary mb-5 max-w-[380px] text-[12.5px] leading-relaxed"
        >
            {body}
        </p>
    {/if}
    {#if actionLabel && onaction}
        <div class="flex items-center gap-2">
            <button
                type="button"
                class="bg-accent hover:bg-accent-hover text-accent-fg h-8 px-4 text-[12px] font-semibold transition-colors"
                onclick={onaction}>{actionLabel}</button
            >
            {#if secondaryLabel && onsecondary}
                <button
                    type="button"
                    class="text-fg-secondary border-border hover:bg-bg-hover hover:text-fg-primary h-8 border px-3.5 text-[12px] font-medium transition-colors"
                    onclick={onsecondary}>{secondaryLabel}</button
                >
            {/if}
        </div>
    {/if}
</div>
