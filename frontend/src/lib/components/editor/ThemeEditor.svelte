<script lang="ts">
    import WallpaperHero from './WallpaperHero.svelte';
    import WallpaperPicker from './WallpaperPicker.svelte';
    import ColorPaletteGrid from './ColorPaletteGrid.svelte';
    import HeroEmptyState from './HeroEmptyState.svelte';
    import SemanticColors from './SemanticColors.svelte';
    import AppColorOverrides from './AppColorOverrides.svelte';
    import AdditionalImages from './AdditionalImages.svelte';
    import SettingsSidebar from '../sidebar/SettingsSidebar.svelte';
    import ColorPickerDialog from '../color-picker/ColorPickerDialog.svelte';
    import WallpaperEditor from '../wallpaper-editor/WallpaperEditor.svelte';
    import ColorDragGhost from './ColorDragGhost.svelte';
    import {getWallpaperPath, getPalette} from '$lib/stores/theme.svelte';
    import {
        getSidebarVisible,
        getColorPickerOpen,
        getImageEditorOpen,
        setImageEditorOpen,
    } from '$lib/stores/ui.svelte';
    import {DEFAULT_PALETTE} from '$lib/types/theme';

    let wallpaper = $derived(getWallpaperPath());
    let palette = $derived(getPalette());
    let hasContent = $derived(
        !!wallpaper || palette.some((c, i) => c !== DEFAULT_PALETTE[i])
    );
    let sidebarVisible = $derived(getSidebarVisible());
    let colorPickerOpen = $derived(getColorPickerOpen());
</script>

<div class="flex h-full">
    {#if sidebarVisible}
        <aside
            class="bg-bg-secondary border-border w-[284px] shrink-0 border-r"
        >
            <SettingsSidebar />
        </aside>
    {/if}

    <div class="min-w-0 flex-1 overflow-y-auto">
        {#if hasContent}
            <div class="flex max-w-[1280px] flex-col gap-7 px-6 pb-8 pt-5">
                {#if wallpaper}
                    <WallpaperHero
                        expanded={colorPickerOpen}
                        onedit={() => setImageEditorOpen(true)}
                    />
                {:else}
                    <WallpaperPicker />
                {/if}

                <ColorPaletteGrid />
                <SemanticColors />
                <div
                    class="grid grid-cols-1 items-start gap-2.5 lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]"
                >
                    <AppColorOverrides />
                    <AdditionalImages />
                </div>
            </div>
        {:else}
            <HeroEmptyState />
        {/if}
    </div>

    {#if colorPickerOpen}
        <aside
            class="bg-bg-secondary border-border flex min-h-0 w-[304px] shrink-0 flex-col border-l"
        >
            <ColorPickerDialog />
        </aside>
    {/if}

    <WallpaperEditor
        open={getImageEditorOpen()}
        onclose={() => setImageEditorOpen(false)}
    />
    <ColorDragGhost />
</div>
