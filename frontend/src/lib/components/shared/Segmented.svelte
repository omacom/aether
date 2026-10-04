<script lang="ts" generics="T extends string">
    // Bordered segmented control: palette mode, color model, card size.
    // The active option gets the elevated surface; the rest stay dimmed.
    let {
        options,
        value,
        onchange,
        size = 'md',
        mono = false,
        label,
        itemClass = '',
    }: {
        options: readonly {value: T; label: string; title?: string}[];
        value: T;
        onchange: (value: T) => void;
        size?: 'sm' | 'md';
        mono?: boolean;
        label?: string;
        itemClass?: string;
    } = $props();
</script>

<div
    class="border-border flex shrink-0 gap-[2px] border p-[2px]"
    role="group"
    aria-label={label}
    title={label}
>
    {#each options as opt}
        <button
            type="button"
            class="flex items-center justify-center transition-colors duration-100
                {size === 'sm' ? 'h-5 px-2' : 'h-[22px] px-3'}
                {mono
                ? 'font-mono text-[10px] font-semibold'
                : 'text-[11.5px] font-medium'}
                {value === opt.value
                ? 'bg-bg-elevated text-fg-primary'
                : 'text-fg-dimmed hover:text-fg-secondary'} {itemClass}"
            aria-pressed={value === opt.value}
            title={opt.title}
            onclick={() => onchange(opt.value)}>{opt.label}</button
        >
    {/each}
</div>
