<script lang="ts">
    import type {Snippet} from 'svelte';

    // Header row for an always-open content section in the editor (Palette,
    // Semantic colors, Additional images). Keeps the title typography
    // identical across sections. `suffix` is a dimmed note after the title.
    // The optional `children` snippet holds right-aligned actions. Distinct
    // from `SectionLabel`, the uppercase group label in the sidebar.
    let {
        title,
        suffix = '',
        children,
    }: {
        title: string;
        suffix?: string;
        children?: Snippet;
    } = $props();
</script>

<div class="mb-3 flex items-end justify-between gap-3">
    <div class="flex min-w-0 items-baseline gap-2.5">
        <h3 class="text-fg-primary shrink-0 text-[13.5px] font-semibold">
            {title}
        </h3>
        {#if suffix}
            <span class="text-fg-dimmed truncate text-[12px]">{suffix}</span>
        {/if}
    </div>
    {#if children}
        <div class="flex shrink-0 items-center gap-1.5">
            {@render children()}
        </div>
    {/if}
</div>
