<script lang="ts">
    import {onDestroy} from 'svelte';
    import AdjustmentSlider from './AdjustmentSlider.svelte';
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import CurvesEditor from '$lib/components/wallpaper-editor/CurvesEditor.svelte';
    import {
        getAdjustments,
        adjustPalette,
        setAdjustments,
        getPalette,
        getBasePalette,
        setAdjustedPalette,
        getBaseExtendedColors,
        setAdjustedExtendedColors,
        getPaletteCurvePoints,
        setPaletteCurvePoints,
        getHistorySnapshot,
        getThemeRevision,
    } from '$lib/stores/theme.svelte';
    import {pushState} from '$lib/stores/history.svelte';
    import {ADJUSTMENT_LIMITS} from '$lib/constants/colors';
    import {DEFAULT_ADJUSTMENTS} from '$lib/types/theme';
    import {debounce} from '$lib/utils/debounce';
    import {hexToRgb} from '$lib/utils/color';

    let adj = $derived(getAdjustments());
    let expanded = $state(true);
    let curvePoints = $state<[number, number][]>([]);

    // Sync store → local on external changes (undo/redo)
    $effect(() => {
        const stored = getPaletteCurvePoints();
        if (JSON.stringify(stored) !== JSON.stringify(curvePoints)) {
            curvePoints = stored.map(([x, y]) => [x, y]);
        }
    });

    // Build a palette luminance histogram (Gaussian bumps at each color's L)
    let paletteHistogram = $derived.by(() => {
        const pal = getPalette();
        const hist = new Array(256).fill(0);
        const sigma = 8;
        for (const hex of pal) {
            if (!hex || hex.length < 7) continue;
            const {r, g, b} = hexToRgb(hex);
            const lum = Math.round((Math.max(r, g, b) + Math.min(r, g, b)) / 2);
            for (let i = 0; i < 256; i++) {
                const d = i - lum;
                hist[i] += Math.exp((-d * d) / (2 * sigma * sigma));
            }
        }
        return hist;
    });

    function handleCurveChange() {
        beginEdit('curve');
        adjustPalette(getAdjustments(), curvePoints);
        editRevision = getThemeRevision();
        endEdit();
    }

    const sliderDefs = [
        {key: 'vibrance', label: 'Vibrance'},
        {key: 'saturation', label: 'Saturation'},
        {key: 'contrast', label: 'Contrast'},
        {key: 'brightness', label: 'Brightness'},
        {key: 'shadows', label: 'Shadows'},
        {key: 'highlights', label: 'Highlights'},
        {key: 'hueShift', label: 'Hue Shift'},
        {key: 'temperature', label: 'Temperature'},
        {key: 'tint', label: 'Tint'},
        {key: 'blackPoint', label: 'Black Point'},
        {key: 'whitePoint', label: 'White Point'},
        {key: 'gamma', label: 'Gamma'},
    ] as const;

    // Push before editing, so Undo works even while a debounce/RPC is pending.
    let editKind: 'slider' | 'nudge' | 'curve' | null = null;
    let editRevision = -1;
    const endEdit = debounce(() => {
        editKind = null;
    }, 500);

    function beginEdit(kind: typeof editKind) {
        if (editKind !== kind || editRevision !== getThemeRevision()) {
            pushState(getHistorySnapshot());
        }
        endEdit.cancel();
        editKind = kind;
    }

    function handleSliderInput(key: string, value: number) {
        beginEdit('slider');
        const newAdj = {...getAdjustments(), [key]: value};
        adjustPalette(newAdj);
        editRevision = getThemeRevision();
    }

    function handleSliderCommit() {
        editKind = null;
    }

    function resetAll() {
        endEdit.cancel();
        editKind = null;
        pushState(getHistorySnapshot());
        setAdjustments({...DEFAULT_ADJUSTMENTS});
        curvePoints = [];
        setPaletteCurvePoints([]);
        setAdjustedPalette(getBasePalette());
        setAdjustedExtendedColors(getBaseExtendedColors());
    }

    type AdjustmentKey = (typeof sliderDefs)[number]['key'];

    function nudge(key: AdjustmentKey, delta: number) {
        const current = getAdjustments();
        const limits = ADJUSTMENT_LIMITS[key];
        const next = Math.max(
            limits.min,
            Math.min(limits.max, current[key] + delta)
        );
        if (next === current[key]) return;

        beginEdit('nudge');
        const newAdj = {...current, [key]: next};
        adjustPalette(newAdj);
        editRevision = getThemeRevision();
        endEdit();
    }

    onDestroy(() => {
        endEdit.cancel();
    });

    const VARIANTS: {
        label: string;
        title: string;
        key: AdjustmentKey;
        delta: number;
    }[] = [
        {label: '−H', title: 'Shift hue −15°', key: 'hueShift', delta: -15},
        {label: '+H', title: 'Shift hue +15°', key: 'hueShift', delta: 15},
        {label: '−S', title: 'Desaturate 15%', key: 'saturation', delta: -15},
        {label: '+S', title: 'Saturate 15%', key: 'saturation', delta: 15},
        {
            label: '❄',
            title: 'Cooler (temperature −15)',
            key: 'temperature',
            delta: -15,
        },
        {
            label: '☀',
            title: 'Warmer (temperature +15)',
            key: 'temperature',
            delta: 15,
        },
    ];
</script>

<ExpandableSection title="Color Adjustments" bind:expanded>
    <div class="mb-3">
        <CurvesEditor
            bind:points={curvePoints}
            histogram={paletteHistogram}
            onchange={handleCurveChange}
        />
    </div>

    <div class="mb-2 flex items-center justify-between gap-2">
        <div class="flex gap-1">
            {#each VARIANTS as v}
                <button
                    type="button"
                    class="text-fg-secondary hover:text-fg-primary border-border hover:bg-bg-surface border px-1.5 py-0.5 text-[10px] tabular-nums transition-colors"
                    onclick={() => nudge(v.key, v.delta)}
                    title={v.title}
                    aria-label={v.title}>{v.label}</button
                >
            {/each}
        </div>
        <button
            class="text-fg-dimmed hover:text-fg-secondary text-[10px]"
            onclick={resetAll}
        >
            Reset All
        </button>
    </div>
    <div class="flex flex-col gap-1.5">
        {#each sliderDefs as def}
            <AdjustmentSlider
                label={def.label}
                value={adj[def.key]}
                min={ADJUSTMENT_LIMITS[def.key].min}
                max={ADJUSTMENT_LIMITS[def.key].max}
                step={ADJUSTMENT_LIMITS[def.key].step}
                defaultValue={ADJUSTMENT_LIMITS[def.key].default}
                oninput={v => handleSliderInput(def.key, v)}
                oncommit={handleSliderCommit}
            />
        {/each}
    </div>
</ExpandableSection>
