<script lang="ts">
    import {onMount} from 'svelte';
    import {
        setPalette,
        setExtendedColors,
        setNativeColors,
        setIconTheme,
        setWallpaperPath,
        setAdditionalImages,
        setAppOverrides,
        setLightMode,
    } from '$lib/stores/theme.svelte';
    import {setActiveTab, showToast} from '$lib/stores/ui.svelte';
    import {
        getCachedThumbnail,
        loadThumbnail,
    } from '$lib/stores/imagecache.svelte';
    import type {omarchy} from '../../../../wailsjs/go/models';
    import EmptyState from '$lib/components/shared/EmptyState.svelte';
    import LoadingState from '$lib/components/shared/LoadingState.svelte';
    import ViewHeader from '$lib/components/shared/ViewHeader.svelte';
    import CardSizeToggle from '$lib/components/shared/CardSizeToggle.svelte';
    import {getCardSize} from '$lib/stores/cardsize.svelte';
    import {THEME_CARD_MIN_WIDTH} from './BlueprintCard.svelte';
    import {
        getOmarchyCapabilities,
        refreshOmarchyCapabilities,
    } from '$lib/stores/omarchy.svelte';

    type Theme = omarchy.Theme;

    let themes = $state<Theme[]>([]);
    let isLoading = $state(true);
    let applyingName = $state('');
    let capabilities = $derived(getOmarchyCapabilities());

    onMount(() => {
        loadThemes();
    });

    async function loadThemes() {
        isLoading = true;
        try {
            const {LoadOmarchyThemes} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await LoadOmarchyThemes();
            themes = Array.isArray(result) ? result : [];
            // Load wallpaper previews into global cache
            for (const theme of themes) {
                const preview = theme.preview || theme.wallpapers?.[0];
                if (preview) {
                    loadThumbnail(preview);
                }
            }
        } catch {
            themes = [];
        } finally {
            isLoading = false;
        }
    }

    function loadThemeIntoState(theme: Theme): boolean {
        if (theme.colors?.length < 16) return false;
        setPalette(theme.colors);
        setExtendedColors(theme.extendedColors ?? {});
        setNativeColors(theme.nativeColors ?? {});
        setIconTheme(theme.iconTheme, true);
        if (theme.mode) setLightMode(theme.mode === 'light');
        setWallpaperPath(theme.wallpapers?.[0] ?? '');
        setAdditionalImages(theme.wallpapers?.slice(1) ?? []);
        setAppOverrides({});
        return true;
    }

    function handleEdit(theme: Theme) {
        if (loadThemeIntoState(theme)) {
            setActiveTab('editor');
            showToast(`Loaded theme: ${theme.name}`);
        }
    }

    async function handleApply(theme: Theme) {
        if (applyingName || !theme.canApply) return;
        applyingName = theme.name;
        try {
            const {ApplyOmarchyThemeByName} = await import(
                '../../../../wailsjs/go/main/App'
            );
            await ApplyOmarchyThemeByName(theme.name);
            await refreshOmarchyCapabilities();
            await loadThemes();
            showToast(`Applied theme: ${theme.name}`);
        } catch (error: unknown) {
            const message =
                typeof error === 'string'
                    ? error
                    : error instanceof Error
                      ? error.message
                      : 'Failed to apply theme';
            showToast(message);
        } finally {
            applyingName = '';
        }
    }
</script>

{#snippet tag(label: string, accent = false)}
    <span
        class="whitespace-nowrap border px-1.5 py-0.5 text-[10.5px] {accent
            ? 'border-accent text-accent'
            : 'border-border text-fg-secondary'}">{label}</span
    >
{/snippet}

<div class="flex h-full flex-col">
    <ViewHeader>
        <h2 class="text-fg-primary text-[13.5px] font-semibold">
            Omarchy themes
        </h2>
        {#if !isLoading}
            <span class="text-fg-dimmed text-[12px]"
                >{themes.length} installed{#if capabilities.version}
                    · Omarchy {capabilities.version}{/if}</span
            >
        {/if}
        <div class="ml-auto flex items-center gap-2">
            <CardSizeToggle />
        </div>
    </ViewHeader>

    <div class="flex-1 overflow-y-auto p-4">
        {#if isLoading}
            <LoadingState message="Loading system themes…" />
        {:else if themes.length === 0}
            <EmptyState
                title="No Omarchy themes found"
                body="Create a theme in the editor or install one with Omarchy."
            >
                {#snippet icon()}
                    <svg
                        class="h-6 w-6"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                    >
                        <path
                            d="M18.37 2.63a2.12 2.12 0 0 1 3 3L14 13l-4 1 1-4z"
                        ></path>
                        <path
                            d="M9 14.5A3.5 3.5 0 0 0 5.5 18c-1.2 0-2.5.7-2.5 2 2 0 4.5-1 5.5-3.5"
                        ></path>
                    </svg>
                {/snippet}
            </EmptyState>
        {:else}
            <div
                class="grid gap-3"
                style:grid-template-columns="repeat(auto-fill, minmax({THEME_CARD_MIN_WIDTH[
                    getCardSize()
                ]}px, 1fr))"
            >
                {#each themes as theme, i (theme.name + '_' + i)}
                    {@const preview = theme.preview || theme.wallpapers?.[0]}
                    <div
                        class="bg-bg-secondary border-border hover:border-border-focus flex flex-col border transition-colors"
                    >
                        <div
                            class="bg-bg-primary flex aspect-video overflow-hidden"
                        >
                            {#if preview && getCachedThumbnail(preview)}
                                <img
                                    src={getCachedThumbnail(preview)}
                                    alt={theme.name}
                                    class="h-full w-full object-cover"
                                />
                            {:else}
                                {#each (theme.colors || []).slice(0, 8) as c}
                                    <span
                                        class="flex-1"
                                        style:background-color={c}
                                    ></span>
                                {/each}
                            {/if}
                        </div>

                        <div class="flex h-1.5" aria-hidden="true">
                            {#each (theme.colors || []).slice(0, 16) as c}
                                <span class="flex-1" style:background-color={c}
                                ></span>
                            {/each}
                        </div>

                        <div class="flex flex-col gap-2.5 p-3">
                            <span
                                class="text-fg-primary truncate text-[13px] font-semibold"
                                >{theme.name}</span
                            >
                            <div class="flex items-center gap-1.5">
                                <div
                                    class="flex min-w-0 flex-1 flex-wrap gap-1.5"
                                >
                                    {#if theme.isCurrentTheme}
                                        {@render tag('Current', true)}
                                    {/if}
                                    {#if theme.isOverlay}
                                        {@render tag('Overlay')}
                                    {/if}
                                    {#if theme.isAetherGenerated}
                                        {@render tag('Aether')}
                                    {/if}
                                </div>
                                <button
                                    type="button"
                                    class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-7 border px-2.5 text-[12px] font-medium transition-colors"
                                    onclick={() => handleEdit(theme)}
                                    title="Import colors, icons, and wallpapers into the editor"
                                    >Import</button
                                >
                                <button
                                    type="button"
                                    class="bg-accent hover:bg-accent-hover text-accent-fg disabled:bg-bg-elevated disabled:text-fg-dimmed h-7 px-3 text-[12px] font-semibold transition-colors disabled:cursor-default"
                                    onclick={() => handleApply(theme)}
                                    disabled={!!applyingName || !theme.canApply}
                                    title={theme.canApply
                                        ? 'Apply with Omarchy'
                                        : 'This theme is available for editing only'}
                                    >{applyingName === theme.name
                                        ? 'Applying...'
                                        : 'Apply'}</button
                                >
                            </div>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>
