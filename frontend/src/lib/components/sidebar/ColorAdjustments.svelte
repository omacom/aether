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
        {key: 'hueShift', label: 'Hue shift'},
        {key: 'temperature', label: 'Temperature'},
        {key: 'tint', label: 'Tint'},
        {key: 'blackPoint', label: 'Black point'},
        {key: 'whitePoint', label: 'White point'},
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
            label: 'Cool',
            title: 'Cooler (temperature −15)',
            key: 'temperature',
            delta: -15,
        },
        {
            label: 'Warm',
            title: 'Warmer (temperature +15)',
            key: 'temperature',
            delta: 15,
        },
    ];

    // The first six sliders are always visible. The other sliders show when
    // the user expands the list or when their value is not the default, so
    // a nudge never changes a hidden slider.
    const PRIMARY_SLIDER_COUNT = 6;
    let showAllSliders = $state(false);
    function isChanged(key: AdjustmentKey): boolean {
        return adj[key] !== ADJUSTMENT_LIMITS[key].default;
    }
    let visibleSliders = $derived(
        sliderDefs.filter(
            (def, i) =>
                showAllSliders || i < PRIMARY_SLIDER_COUNT || isChanged(def.key)
        )
    );
    let changedCount = $derived(
        sliderDefs.filter(def => isChanged(def.key)).length +
            (curvePoints.length > 0 ? 1 : 0)
    );
</script>

<ExpandableSection
    title="Color adjustments"
    suffix={changedCount > 0 ? `${changedCount} changed` : ''}
    suffixAccent
    bind:expanded
>
    <div class="flex flex-col gap-3.5">
        <CurvesEditor
            bind:points={curvePoints}
            histogram={paletteHistogram}
            onchange={handleCurveChange}
            height={124}
            label="Curve"
            compact
        />

        <div class="flex items-center gap-1">
            {#each VARIANTS as v}
                <button
                    type="button"
                    class="border-border text-fg-secondary hover:border-border-focus hover:text-fg-primary h-6 border px-[7px] font-mono text-[10.5px] font-medium transition-colors"
                    onclick={() => nudge(v.key, v.delta)}
                    title={v.title}
                    aria-label={v.title}>{v.label}</button
                >
            {/each}
            <span class="flex-1"></span>
            <button
                type="button"
                class="text-fg-dimmed hover:text-fg-primary text-[11px] transition-colors"
                onclick={resetAll}
                title="Reset all adjustments and the curve"
            >
                Reset
            </button>
        </div>

        <div class="flex flex-col gap-2.5">
            {#each visibleSliders as def (def.key)}
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

        {#if showAllSliders || visibleSliders.length < sliderDefs.length}
            <button
                type="button"
                class="text-accent hover:text-accent-hover self-start text-[11.5px] transition-colors"
                onclick={() => (showAllSliders = !showAllSliders)}
                aria-expanded={showAllSliders}
            >
                {showAllSliders
                    ? 'Show fewer'
                    : `Show all ${sliderDefs.length} sliders`}
            </button>
        {/if}
    </div>
</ExpandableSection>
