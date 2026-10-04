<script lang="ts">
    import ExtractionModeSelect from './ExtractionModeSelect.svelte';
    import ColorAdjustments from './ColorAdjustments.svelte';
    import PresetsSection from './PresetsSection.svelte';
    import GradientGenerator from './GradientGenerator.svelte';
    import PaletteFromColor from './PaletteFromColor.svelte';
    import AccessibilityPanel from './AccessibilityPanel.svelte';
    import NeovimThemes from './NeovimThemes.svelte';
    import TemplateToggles from './TemplateToggles.svelte';
    import IconThemePicker from './IconThemePicker.svelte';
    import SectionLabel from '$lib/components/shared/SectionLabel.svelte';
    import Segmented from '$lib/components/shared/Segmented.svelte';
    import {getLightMode, setLightMode} from '$lib/stores/theme.svelte';
    import {
        getOmarchyAvailable,
        initOmarchyCapabilities,
    } from '$lib/stores/omarchy.svelte';

    let lightMode = $derived(getLightMode());
    let isOmarchy = $derived(getOmarchyAvailable());

    const paletteModes = [
        {value: 'dark', label: 'Dark', title: 'Generate a dark palette'},
        {value: 'light', label: 'Light', title: 'Generate a light palette'},
    ] as const;

    void initOmarchyCapabilities();
</script>

<!-- Each section component renders its own bordered <section>. -->
<div class="flex h-full flex-col overflow-y-auto pb-6">
    <div
        class="border-border flex items-center justify-between gap-3 border-b px-4 py-3.5"
    >
        <span class="text-fg-secondary text-[12px] font-medium"
            >Palette mode</span
        >
        <Segmented
            options={paletteModes}
            value={lightMode ? 'light' : 'dark'}
            onchange={mode => setLightMode(mode === 'light')}
            label="Palette mode"
        />
    </div>

    <SectionLabel label="Generate" />
    <ExtractionModeSelect />
    <PresetsSection />
    <PaletteFromColor />
    <GradientGenerator />

    <SectionLabel label="Adjust" />
    <ColorAdjustments />
    <IconThemePicker />
    <AccessibilityPanel />

    <SectionLabel label="Targets" />
    <NeovimThemes />
    {#if !isOmarchy}
        <TemplateToggles />
    {/if}
</div>
