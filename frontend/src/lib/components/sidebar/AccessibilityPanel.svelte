<script lang="ts">
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import {getPalette} from '$lib/stores/theme.svelte';
    import {
        contrastLevel,
        contrastRatio,
        isValidHex,
        type ContrastLevel,
    } from '$lib/utils/color';

    let expanded = $state(false);

    // Foreground and the six ANSI hues, each checked against the background.
    const ROWS = [
        {index: 7, name: 'Foreground'},
        {index: 1, name: 'Red'},
        {index: 2, name: 'Green'},
        {index: 3, name: 'Yellow'},
        {index: 4, name: 'Blue'},
        {index: 5, name: 'Magenta'},
        {index: 6, name: 'Cyan'},
    ] as const;

    let palette = $derived(getPalette());
    let rows = $derived.by(() => {
        const bg = palette[0];
        return ROWS.map(row => {
            const hex = palette[row.index];
            const ratio =
                isValidHex(bg) && isValidHex(hex) ? contrastRatio(hex, bg) : 0;
            return {...row, hex, ratio, level: contrastLevel(ratio)};
        });
    });
    let passing = $derived(
        rows.filter(r => r.level === 'AA' || r.level === 'AAA').length
    );

    function levelLabel(level: ContrastLevel): string {
        return level === 'fail' ? 'Fail' : level;
    }

    function levelClass(level: ContrastLevel): string {
        if (level === 'AAA') return 'text-success';
        if (level === 'AA') return 'text-accent';
        if (level === 'AA-L') return 'text-warning';
        return 'text-destructive';
    }
</script>

<ExpandableSection
    title="Accessibility"
    suffix="{passing} of {rows.length} pass AA"
    contentClass="px-4 pb-3.5"
    bind:expanded
>
    <div class="flex flex-col gap-1">
        {#each rows as row (row.index)}
            <div
                class="flex h-6 items-center gap-2 text-[11.5px]"
                title="{row.name} on background: {row.ratio.toFixed(2)}:1"
            >
                <span
                    class="outline-border h-3 w-3 shrink-0 outline outline-1"
                    style:background-color={row.hex}
                ></span>
                <span class="text-fg-secondary flex-1">{row.name}</span>
                <span class="text-fg-dimmed font-mono text-[11px] font-medium"
                    >{row.ratio.toFixed(1)}:1</span
                >
                <span
                    class="w-[34px] text-right font-mono text-[10.5px] font-semibold {levelClass(
                        row.level
                    )}">{levelLabel(row.level)}</span
                >
            </div>
        {/each}
    </div>
    <div class="mt-2.5 flex gap-1" aria-hidden="true">
        <div
            class="outline-border flex h-6 flex-1 items-center justify-center text-[11px] outline outline-1"
            style:background-color={palette[0]}
            style:color={palette[7]}
        >
            Sample
        </div>
        <div
            class="outline-border flex h-6 flex-1 items-center justify-center text-[11px] outline outline-1"
            style:background-color={palette[7]}
            style:color={palette[0]}
        >
            Sample
        </div>
    </div>
</ExpandableSection>
