<script lang="ts">
    import ColorSwatch from './ColorSwatch.svelte';
    import SectionHeader from '$lib/components/shared/SectionHeader.svelte';
    import {
        getPalette,
        getLockedColors,
        setLockedColor,
        getSelectedColors,
        hasAnySelection,
        clearColorSelection,
        shufflePalette,
        setPalette,
        getExtractionMode,
        getSelectedExtColors,
    } from '$lib/stores/theme.svelte';
    import {
        openColorPicker,
        showToast,
        getColorPickerOpen,
        getColorPickerIndex,
        getColorPickerExtKey,
        getColorPickerOverrideApp,
    } from '$lib/stores/ui.svelte';
    import {
        ANSI_COLOR_NAMES,
        ANSI_SLOT_ROLES,
        EXTRACTION_MODES,
    } from '$lib/constants/colors';
    import {hslToHex, copyColor} from '$lib/utils/color';

    let palette = $derived(getPalette());
    let locked = $derived(getLockedColors());
    let selected = $derived(getSelectedColors());
    let selectedCount = $derived(
        Object.values(selected).filter(Boolean).length +
            Object.values(getSelectedExtColors()).filter(Boolean).length
    );
    let hasSelect = $derived(hasAnySelection());
    let modeLabel = $derived(
        EXTRACTION_MODES.find(m => m.value === getExtractionMode())?.label ?? ''
    );
    // The palette slot that the color picker edits, or -1.
    let activeIndex = $derived(
        getColorPickerOpen() &&
            !getColorPickerExtKey() &&
            !getColorPickerOverrideApp()
            ? getColorPickerIndex()
            : -1
    );

    let gridEl = $state<HTMLDivElement | null>(null);
    let focusedIndex = $state(0);

    // 8 cols × 2 rows; top row 0..7, bottom row 8..15.
    function nextIndex(idx: number, key: string): number | null {
        if (key === 'ArrowRight')
            return idx === 7 ? 0 : idx === 15 ? 8 : idx + 1;
        if (key === 'ArrowLeft')
            return idx === 0 ? 7 : idx === 8 ? 15 : idx - 1;
        if (key === 'ArrowDown') return idx < 8 ? idx + 8 : idx - 8;
        if (key === 'ArrowUp') return idx >= 8 ? idx - 8 : idx + 8;
        if (key === 'Home') return idx < 8 ? 0 : 8;
        if (key === 'End') return idx < 8 ? 7 : 15;
        return null;
    }

    function focusSwatch(idx: number) {
        focusedIndex = idx;
        queueMicrotask(() => {
            gridEl
                ?.querySelector<HTMLElement>(`[data-swatch-idx="${idx}"]`)
                ?.focus();
        });
    }

    function handleGridKey(e: KeyboardEvent) {
        const tgt = e.target as HTMLElement;
        const idxAttr = tgt.dataset?.swatchIdx;
        if (idxAttr === undefined) return;
        const idx = Number(idxAttr);

        const next = nextIndex(idx, e.key);
        if (next !== null) {
            e.preventDefault();
            focusSwatch(next);
            return;
        }

        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            if (!locked[idx]) openColorPicker(idx);
        } else if (
            (e.key === 'l' || e.key === 'L') &&
            !e.ctrlKey &&
            !e.metaKey
        ) {
            e.preventDefault();
            setLockedColor(idx, !locked[idx]);
        } else if (
            (e.key === 'c' || e.key === 'C') &&
            !e.ctrlKey &&
            !e.metaKey
        ) {
            e.preventDefault();
            copyColor(palette[idx]);
        }
    }

    const labels = [
        'BG',
        'Red',
        'Green',
        'Yellow',
        'Blue',
        'Magenta',
        'Cyan',
        'FG',
    ];

    async function randomPalette() {
        const hue = Math.random() * 360;
        const sat = 40 + Math.random() * 40;
        const lit = 45 + Math.random() * 20;
        const seed = hslToHex(hue, sat, lit);
        try {
            const {GeneratePaletteFromColor} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await GeneratePaletteFromColor(seed);
            setPalette(result);
            showToast(`Random palette from ${seed}`);
        } catch {
            showToast('Couldn’t generate a random palette');
        }
    }
</script>

<section>
    <SectionHeader
        title="Palette"
        suffix={hasSelect
            ? `${selectedCount} selected`
            : `16 colors${modeLabel ? ` · ${modeLabel}` : ''}`}
    >
        {#if hasSelect}
            <button
                class="text-accent hover:text-accent-hover h-[26px] px-2 text-[11.5px] transition-colors"
                onclick={clearColorSelection}>Clear selection</button
            >
        {/if}
        <button
            class="border-border text-fg-secondary hover:border-border-focus hover:text-fg-primary h-[26px] border px-2.5 text-[11.5px] transition-colors"
            onclick={randomPalette}
            title="Generate palette from a random shade (experimental)"
            >Random</button
        >
        <button
            class="border-border text-fg-secondary hover:border-border-focus hover:text-fg-primary h-[26px] border px-2.5 text-[11.5px] transition-colors"
            onclick={shufflePalette}
            title="Shuffle ANSI color roles (experimental)">Shuffle</button
        >
    </SectionHeader>

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <!-- 8 columns. Each column has a label, the normal color, and the bright color. -->
    <div
        bind:this={gridEl}
        class="grid grid-cols-8 gap-2"
        role="grid"
        tabindex={-1}
        aria-label="Palette colours"
        onkeydown={handleGridKey}
    >
        {#each Array(8) as _, i}
            <div class="flex min-w-0 flex-col gap-1.5">
                <span
                    class="text-fg-dimmed select-none pl-px text-[11px] font-medium"
                    >{labels[i]}</span
                >
                {#each [i, i + 8] as idx}
                    <ColorSwatch
                        color={palette[idx]}
                        index={idx}
                        label={ANSI_COLOR_NAMES[idx]}
                        role={ANSI_SLOT_ROLES[idx]}
                        contrastAgainst={palette[0]}
                        locked={locked[idx] || false}
                        selected={selected[idx] || false}
                        active={activeIndex === idx}
                        focused={focusedIndex === idx}
                        onclick={() => openColorPicker(idx)}
                    />
                {/each}
            </div>
        {/each}
    </div>
    <p class="text-fg-dimmed mt-2.5 text-[11px]">
        Click to edit · Shift-click to select · Ctrl-click to copy · Drag onto a
        template override · <span class="font-mono">L</span> lock ·
        <span class="font-mono">C</span> copy
    </p>
</section>
