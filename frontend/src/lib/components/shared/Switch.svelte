<script lang="ts">
    import type {Snippet} from 'svelte';

    // Square toggle switch. Pass `children` to render a text label inside
    // the same button, as the Live toggle in the action bar does.
    let {
        checked,
        onchange,
        label,
        title,
        size = 'md',
        disabled = false,
        class: className = '',
        children,
    }: {
        checked: boolean;
        onchange: (checked: boolean) => void;
        label?: string;
        title?: string;
        size?: 'sm' | 'md';
        disabled?: boolean;
        class?: string;
        children?: Snippet;
    } = $props();

    let knobLeft = $derived(checked ? (size === 'sm' ? 14 : 16) : 2);
</script>

<button
    type="button"
    role="switch"
    aria-checked={checked}
    aria-label={label}
    {title}
    {disabled}
    class="flex shrink-0 items-center gap-2 disabled:cursor-default disabled:opacity-45 {className}"
    onclick={() => onchange(!checked)}
>
    <span
        class="relative block shrink-0 border transition-colors duration-150
            {size === 'sm' ? 'h-3.5 w-[26px]' : 'h-4 w-[30px]'}
            {checked
            ? 'bg-accent border-accent'
            : 'bg-bg-surface border-border-focus'}"
    >
        <span
            class="absolute top-[2px] block transition-[left] duration-150
                {size === 'sm' ? 'h-2 w-2' : 'h-2.5 w-2.5'}
                {checked ? 'bg-accent-fg' : 'bg-fg-dimmed'}"
            style:left="{knobLeft}px"
        ></span>
    </span>
    {#if children}
        {@render children()}
    {/if}
</button>
