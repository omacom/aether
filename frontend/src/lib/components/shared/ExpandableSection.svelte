<script lang="ts">
    import type {Snippet} from 'svelte';

    // Collapsible block with a chevron that turns when the block opens.
    // `section`: a full-width sidebar row with a bottom border.
    // `group`: a small uppercase header nested inside a section.
    let {
        title,
        expanded = $bindable(false),
        suffix = '',
        suffixAccent = false,
        variant = 'section',
        contentClass = 'px-4 pb-4',
        children,
    }: {
        title: string;
        expanded?: boolean;
        suffix?: string;
        suffixAccent?: boolean;
        variant?: 'section' | 'group';
        contentClass?: string;
        children: Snippet;
    } = $props();
</script>

{#snippet chevron(sizeClass: string, strokeWidth: string)}
    <svg
        class="{sizeClass} shrink-0 transition-transform duration-150 {expanded
            ? 'rotate-90'
            : ''}"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width={strokeWidth}
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"><path d="M9 6l6 6-6 6"></path></svg
    >
{/snippet}

{#if variant === 'group'}
    <div>
        <button
            type="button"
            class="text-fg-dimmed hover:text-fg-secondary flex w-full items-center gap-1.5 px-1.5 pb-[5px] pt-2.5 text-[10px] font-semibold uppercase tracking-[0.12em] transition-colors"
            onclick={() => (expanded = !expanded)}
            aria-expanded={expanded}
        >
            {@render chevron('h-2.5 w-2.5', '2.6')}
            <span class="flex-1 text-left">{title}</span>
            {#if suffix}
                <span
                    class="truncate font-medium normal-case tracking-normal {suffixAccent
                        ? 'text-accent'
                        : ''}">{suffix}</span
                >
            {/if}
        </button>
        {#if expanded}
            {@render children()}
        {/if}
    </div>
{:else}
    <section class="border-border border-b">
        <button
            type="button"
            class="group flex w-full items-center gap-2 px-4 py-[11px] text-left"
            onclick={() => (expanded = !expanded)}
            aria-expanded={expanded}
        >
            <span class="text-fg-primary flex-1 text-[12.5px] font-medium"
                >{title}</span
            >
            {#if suffix}
                <span
                    class="max-w-[55%] truncate text-[11px] {suffixAccent
                        ? 'text-accent'
                        : 'text-fg-dimmed'}">{suffix}</span
                >
            {/if}
            <span
                class="text-fg-dimmed group-hover:text-fg-secondary transition-colors"
            >
                {@render chevron('h-3 w-3', '2.2')}
            </span>
        </button>
        {#if expanded}
            <div class={contentClass}>
                {@render children()}
            </div>
        {/if}
    </section>
{/if}
