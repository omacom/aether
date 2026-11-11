<script lang="ts">
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import Segmented from '$lib/components/shared/Segmented.svelte';
    import YozakuraPreview from './YozakuraPreview.svelte';
    import {
        getExtendedColors,
        getLightMode,
        getPalette,
    } from '$lib/stores/theme.svelte';
    import {openYozakuraColorPicker} from '$lib/stores/ui.svelte';
    import {
        CLOCK_DIGITS,
        CLOCK_ORNAMENTS,
        GLASS_MAX,
        GLASS_MIN,
        YOZAKURA_ROLE_KEYS,
        cycleAccent,
        getClock,
        getEffectiveRole,
        getGlass,
        getYozakuraOverrides,
        getYozakuraResult,
        refreshYozakura,
        resetYozakuraRole,
        setClock,
        setGlass,
    } from '$lib/stores/yozakura.svelte';

    let expanded = $state(false);

    let overrides = $derived(getYozakuraOverrides());
    let result = $derived(getYozakuraResult());
    let glass = $derived(getGlass());
    let clock = $derived(getClock());
    let pinned = $derived(Object.keys(overrides).length);
    let contrast = $derived(result?.contrast ?? []);

    // Touch every input of the derivation so any edit re-queries the backend.
    $effect(() => {
        const _ = [
            getPalette(),
            getExtendedColors(),
            getLightMode(),
            overrides,
            glass,
            clock,
        ].map(v => JSON.stringify(v));
        refreshYozakura();
    });

    const DIGIT_OPTIONS = CLOCK_DIGITS.map(v => ({value: v, label: v}));
    const ORNAMENT_OPTIONS = CLOCK_ORNAMENTS.map(v => ({value: v, label: v}));

    function stateClass(state: string): string {
        if (state === 'ok') return 'text-success';
        if (state === 'nudged') return 'text-warning';
        return 'text-destructive';
    }
</script>

<ExpandableSection
    title="Yozakura"
    suffix={pinned ? `${pinned} pinned` : ''}
    contentClass="flex flex-col gap-3.5 px-4 pb-3.5"
    bind:expanded
>
    {#if result}
        <YozakuraPreview roles={result.roles} {glass} {clock} />
    {/if}

    <div class="grid grid-cols-2 gap-x-3 gap-y-1">
        {#each YOZAKURA_ROLE_KEYS as key (key)}
            {@const isPinned = key in overrides}
            <div
                class="flex h-6 items-center gap-1.5 text-[11.5px]"
                data-role={key}
            >
                <button
                    type="button"
                    class="hover:bg-bg-hover flex min-w-0 flex-1 items-center gap-2 text-left"
                    onclick={() => openYozakuraColorPicker(key)}
                    aria-label="Edit {key}"
                    title="{key}: {getEffectiveRole(key) ||
                        '…'} · click to pin a colour"
                >
                    <span
                        class="outline-border h-3 w-3 shrink-0 outline outline-1"
                        style:background-color={getEffectiveRole(key)}
                    ></span>
                    <span class="text-fg-secondary truncate">{key}</span>
                    {#if !isPinned}
                        <span
                            class="text-fg-dimmed text-[10px] italic"
                            title="Derived from the palette">auto</span
                        >
                    {/if}
                </button>
                {#if isPinned}
                    <button
                        type="button"
                        class="text-fg-dimmed hover:text-fg-primary text-[11px]"
                        onclick={() => resetYozakuraRole(key)}
                        aria-label="Reset {key}"
                        title="Unpin {key} and use the derived value">↺</button
                    >
                {/if}
            </div>
        {/each}
    </div>

    <button
        type="button"
        class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-6 border text-[11.5px]"
        onclick={cycleAccent}
        title="Pin the next chromatic accent candidate from the palette"
    >
        Cycle accent
    </button>

    <label class="flex flex-col gap-1 text-[12px]">
        <span class="text-fg-secondary flex justify-between"
            >Glass
            <span class="text-fg-dimmed font-mono text-[11px]"
                >{glass.toFixed(2)}</span
            ></span
        >
        <input
            type="range"
            class="w-full cursor-pointer"
            min={GLASS_MIN}
            max={GLASS_MAX}
            step="0.01"
            value={glass}
            oninput={e => setGlass(parseFloat(e.currentTarget.value))}
            aria-label="Glass"
            title="Surface opacity, {GLASS_MIN.toFixed(
                2
            )} to {GLASS_MAX.toFixed(2)}"
        />
    </label>

    <div class="flex flex-col gap-2">
        <span class="text-fg-dimmed text-[10px] font-semibold uppercase"
            >Clock</span
        >
        <Segmented
            options={DIGIT_OPTIONS}
            value={clock.digits as (typeof CLOCK_DIGITS)[number]}
            onchange={digits => setClock({...clock, digits})}
            size="sm"
            label="Clock digits"
        />
        <Segmented
            options={ORNAMENT_OPTIONS}
            value={clock.ornament as (typeof CLOCK_ORNAMENTS)[number]}
            onchange={ornament => setClock({...clock, ornament})}
            size="sm"
            itemClass="flex-1 px-1"
            label="Clock ornament"
        />
        <label class="flex items-center justify-between gap-2 text-[12px]">
            <span class="text-fg-secondary">Colon</span>
            <input
                type="text"
                maxlength="3"
                class="text-fg-primary bg-bg-primary border-border w-12 border px-1 text-center font-mono text-[12px]"
                value={clock.colon}
                oninput={e =>
                    setClock({...clock, colon: e.currentTarget.value})}
                aria-label="Clock colon"
            />
        </label>
        <label class="flex flex-col gap-1 text-[12px]">
            <span class="text-fg-secondary flex justify-between"
                >Glow
                <span class="text-fg-dimmed font-mono text-[11px]"
                    >{clock.glow.toFixed(2)}</span
                ></span
            >
            <input
                type="range"
                class="w-full cursor-pointer"
                min="0"
                max="1"
                step="0.05"
                value={clock.glow}
                oninput={e =>
                    setClock({
                        ...clock,
                        glow: parseFloat(e.currentTarget.value),
                    })}
                aria-label="Clock glow"
            />
        </label>
    </div>

    <div class="flex flex-col gap-1">
        {#each contrast as row (row.pair)}
            <div
                class="flex h-5 items-center gap-2 text-[11px]"
                title="{row.pair}: {row.ratio.toFixed(2)}:1, needs {row.min}:1"
            >
                <span class="text-fg-secondary flex-1 truncate">{row.pair}</span
                >
                <span class="text-fg-dimmed font-mono"
                    >{row.ratio.toFixed(2)}:1</span
                >
                <span
                    class="w-[44px] text-right font-mono text-[10.5px] font-semibold {stateClass(
                        row.state
                    )}">{row.state}</span
                >
            </div>
        {/each}
    </div>
</ExpandableSection>
