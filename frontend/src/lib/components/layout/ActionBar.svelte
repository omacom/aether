<script lang="ts">
    import type {main} from '../../../../wailsjs/go/models';
    import {
        getIsApplying,
        getPalette,
        getWallpaperPath,
        getWallpaperBlur,
        getLightMode,
        getAdditionalImages,
        getExtendedColors,
        getNativeColors,
        getIconTheme,
        getAppOverrides,
        isDirty,
        reset as resetTheme,
    } from '$lib/stores/theme.svelte';
    import {getCanUndo, getCanRedo} from '$lib/stores/history.svelte';
    import {getSettings} from '$lib/stores/settings.svelte';
    import {
        getActiveTab,
        setActiveTab,
        showToast,
        getLiveApply,
        setLiveApply,
        getLivePending,
        getTargetsVisible,
        toggleTargetsVisible,
        getSidebarVisible,
        toggleSidebar,
    } from '$lib/stores/ui.svelte';
    import {getApiKey, getTotalResults} from '$lib/stores/wallhaven.svelte';
    import {
        applyTheme,
        getNativeAppOverrides,
        requestThemeApply,
        saveThemeAsNew,
        undoAction,
        redoAction,
    } from '$lib/actions/themeActions';
    import {importThemeFile} from '$lib/actions/importActions';
    import SaveDialog from '$lib/components/blueprints/SaveDialog.svelte';
    import ConfirmDialog from '$lib/components/shared/ConfirmDialog.svelte';
    import KbdInverse from '$lib/components/shared/KbdInverse.svelte';
    import Modal from '$lib/components/shared/Modal.svelte';
    import DialogHeader from '$lib/components/shared/DialogHeader.svelte';
    import DialogFooter from '$lib/components/shared/DialogFooter.svelte';
    import Switch from '$lib/components/shared/Switch.svelte';
    import {
        getOmarchyAvailable,
        initOmarchyCapabilities,
    } from '$lib/stores/omarchy.svelte';

    let showImportMenu = $state(false);
    let showApplyMenu = $state(false);
    let showMoreMenu = $state(false);
    let showExportDialog = $state(false);
    let showSaveDialog = $state(false);
    let confirmKind = $state<'revert' | 'reset' | null>(null);

    const CONFIRM_CONFIG = {
        revert: {
            title: 'Revert system theme?',
            body: 'This will roll your desktop back to its default theme. Your editor state stays intact, so you can re-apply afterwards.',
            confirmLabel: 'Revert',
            run: () => handleClear(),
        },
        reset: {
            title: 'Reset editor?',
            body: 'Clears the current palette, wallpaper, adjustments, and overrides from the editor. This does not change anything on disk.',
            confirmLabel: 'Reset',
            run: () => handleReset(),
        },
    } as const;
    let exportName = $state('');
    let installToOmarchy = $state(false);
    let isOmarchy = $derived(getOmarchyAvailable());
    let exportNameInput = $state<HTMLInputElement | null>(null);

    initOmarchyCapabilities();

    $effect(() => {
        if (showExportDialog) exportNameInput?.focus();
    });

    let activeTab = $derived(getActiveTab());
    let wallpaperSelected = $derived(getWallpaperPath() !== '');
    let apiKeySet = $derived(getApiKey() !== '');
    let totalResults = $derived(getTotalResults());

    const exportAppGroups = [
        {
            label: 'Editor themes',
            apps: [
                {key: 'vscode', name: 'VS Code'},
                {key: 'zed', name: 'Zed'},
                {key: 'neovim', name: 'Neovim'},
            ],
        },
        {
            label: 'Terminals',
            apps: [
                {key: 'alacritty', name: 'Alacritty'},
                {key: 'foot', name: 'Foot'},
                {key: 'ghostty', name: 'Ghostty'},
                {key: 'kitty', name: 'Kitty'},
                {key: 'warp', name: 'Warp'},
            ],
        },
        {
            label: 'Desktop',
            apps: [
                {key: 'hyprland', name: 'Hyprland'},
                {key: 'hyprlock', name: 'Hyprlock'},
                {key: 'icons', name: 'Icons'},
                {key: 'mako', name: 'Mako'},
                {key: 'swayosd', name: 'SwayOSD'},
                {key: 'walker', name: 'Walker'},
                {key: 'waybar', name: 'Waybar'},
                {key: 'wofi', name: 'Wofi'},
            ],
        },
        {
            label: 'Apps',
            apps: [
                {key: 'btop', name: 'Btop'},
                {key: 'chromium', name: 'Chromium'},
                {key: 'vencord', name: 'Vencord'},
                {key: 'colors', name: 'Colors (.toml)'},
            ],
        },
    ] as const;

    function createDefaultExportApps(): Record<string, boolean> {
        const apps: Record<string, boolean> = {};
        for (const group of exportAppGroups) {
            for (const app of group.apps) {
                apps[app.key] = true;
            }
        }
        return apps;
    }

    let exportApps = $state<Record<string, boolean>>(createDefaultExportApps());
    let exportAppCount = $derived(
        exportAppGroups.reduce((sum, group) => sum + group.apps.length, 0)
    );
    let exportSelectedCount = $derived(
        Object.values(exportApps).filter(Boolean).length
    );

    function setAllExportApps(enabled: boolean) {
        for (const key of Object.keys(exportApps)) exportApps[key] = enabled;
    }

    function closeMenus() {
        showImportMenu = false;
        showApplyMenu = false;
        showMoreMenu = false;
    }

    let undoEnabled = $derived(getCanUndo());
    let redoEnabled = $derived(getCanRedo());
    let liveApply = $derived(getLiveApply());
    let livePending = $derived(getLivePending());
    let applying = $derived(getIsApplying());
    let dirty = $derived(isDirty());
    let targetsVisible = $derived(getTargetsVisible());
    let sidebarVisible = $derived(getSidebarVisible());
    let menuOpen = $derived(showImportMenu || showApplyMenu || showMoreMenu);
    let overrideCount = $derived(
        Object.values(
            isOmarchy ? getNativeAppOverrides() : getAppOverrides()
        ).reduce((sum, o) => sum + Object.keys(o).length, 0)
    );

    // --- Editor actions ---

    const handleApply = requestThemeApply;

    async function handleClear() {
        try {
            const {ClearTheme} = await import(
                '../../../../wailsjs/go/main/App'
            );
            await ClearTheme();
            showToast('Reverted to system theme');
        } catch {
            showToast('Couldn’t revert — see logs for details');
        }
    }

    async function handleReset() {
        try {
            const {ResetState} = await import(
                '../../../../wailsjs/go/main/App'
            );
            await ResetState();
            resetTheme();
            showToast('Editor reset');
        } catch {
            resetTheme();
            showToast('Editor reset');
        }
    }

    async function handleExport() {
        if (!exportName.trim()) return;
        try {
            await initOmarchyCapabilities();
            const {ExportTheme} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const nativeExport = getOmarchyAvailable();
            const includedApps = Object.entries(exportApps)
                .filter(([, enabled]) => enabled)
                .map(([key]) => key);
            const path = await ExportTheme({
                name: exportName.trim(),
                includedApps,
                palette: getPalette(),
                wallpaperPath: getWallpaperPath(),
                wallpaperBlur: getWallpaperBlur(),
                lightMode: getLightMode(),
                additionalImages: getAdditionalImages(),
                extendedColors: getExtendedColors(),
                nativeColors: getNativeColors(),
                iconTheme: {...getIconTheme()},
                installToOmarchy,
                appOverrides: nativeExport
                    ? getNativeAppOverrides()
                    : getAppOverrides(),
            } as unknown as main.ExportThemeRequest);
            // Path ends with .../omarchy-{slug}-theme — pull the slug so the
            // user can see what name actually went into Omarchy's menu.
            const slug =
                path.match(/omarchy-(.+)-theme$/)?.[1] ?? exportName.trim();
            showToast(
                installToOmarchy
                    ? `Installed as Omarchy theme: ${slug}`
                    : `Exported to ${path}`
            );
            showExportDialog = false;
            exportName = '';
            installToOmarchy = false;
        } catch (e: any) {
            showToast(e?.message || 'Export failed');
        }
    }

    const importOptions = [
        {type: 'base16', label: 'Base16', ext: '.yaml'},
        {type: 'toml', label: 'Colors', ext: '.toml'},
        {type: 'blueprint', label: 'Blueprint', ext: '.json'},
    ] as const;

    async function rescanLocal() {
        try {
            const {ScanLocalWallpapers} = await import(
                '../../../../wailsjs/go/main/App'
            );
            await ScanLocalWallpapers();
            showToast('Wallpapers rescanned');
            // Force LocalBrowser to remount so it re-reads the file list
            setActiveTab('editor');
            setTimeout(() => setActiveTab('local'), 0);
        } catch {
            showToast('Failed to rescan');
        }
    }
</script>

<svelte:window
    onkeydown={e => {
        if (e.key === 'Escape' && menuOpen) closeMenus();
    }}
/>

{#snippet divider()}
    <span class="bg-border mx-1 h-[18px] w-px shrink-0" aria-hidden="true"
    ></span>
{/snippet}

{#snippet menuPanel(
    align: 'left' | 'right',
    widthClass: string,
    items: import('svelte').Snippet
)}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
        class="fixed inset-0 z-30"
        onclick={closeMenus}
        role="presentation"
    ></div>
    <div
        class="bg-bg-secondary border-border shadow-(--shadow-panel) absolute bottom-[calc(100%+8px)] z-40 border py-1 {widthClass}"
        class:left-0={align === 'left'}
        class:right-0={align === 'right'}
        role="menu"
    >
        {@render items()}
    </div>
{/snippet}

{#snippet menuItem(label: string, hint: string, run: () => void)}
    <button
        type="button"
        class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary flex h-[30px] w-full items-center justify-between gap-4 px-3 text-left text-[12px] transition-colors"
        role="menuitem"
        onclick={() => {
            closeMenus();
            run();
        }}
    >
        <span>{label}</span>
        <span class="text-fg-dimmed font-mono text-[10.5px] font-medium"
            >{hint}</span
        >
    </button>
{/snippet}

{#snippet checkbox(label: string, checked: boolean, toggle: () => void)}
    <button
        type="button"
        role="checkbox"
        aria-checked={checked}
        class="text-fg-secondary hover:text-fg-primary flex h-[26px] items-center gap-2 text-left text-[12px] transition-colors"
        onclick={toggle}
    >
        <span
            class="text-accent-fg flex h-3.5 w-3.5 shrink-0 items-center justify-center border transition-colors
                {checked ? 'bg-accent border-accent' : 'border-border-focus'}"
        >
            {#if checked}
                <svg
                    class="h-2.5 w-2.5"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="3.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    aria-hidden="true"><path d="M20 6 9 17l-5-5"></path></svg
                >
            {/if}
        </span>
        <span class="truncate">{label}</span>
    </button>
{/snippet}

{#snippet chevronUp(sizeClass: string)}
    <svg
        class={sizeClass}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"><path d="m6 15 6-6 6 6"></path></svg
    >
{/snippet}

<footer
    class="bg-bg-secondary border-border relative z-[5] flex h-12 shrink-0 items-center gap-1 border-t px-2.5"
>
    {#if activeTab === 'editor'}
        <button
            type="button"
            class="text-fg-dimmed hover:text-fg-primary hover:bg-bg-hover flex h-[30px] w-[30px] items-center justify-center transition-colors"
            onclick={toggleSidebar}
            title={sidebarVisible
                ? 'Hide sidebar (Ctrl+B)'
                : 'Show sidebar (Ctrl+B)'}
            aria-label="Toggle sidebar"
            aria-pressed={sidebarVisible}
        >
            <svg
                class="h-[15px] w-[15px]"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
            >
                <rect x="3" y="3" width="18" height="18" rx="2" />
                <line x1="9" y1="3" x2="9" y2="21" />
            </svg>
        </button>
        {@render divider()}

        <div class="relative">
            <button
                type="button"
                class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary flex h-[30px] items-center gap-[5px] px-2.5 text-[12px] transition-colors"
                onclick={() => {
                    const open = !showImportMenu;
                    closeMenus();
                    showImportMenu = open;
                }}
                aria-haspopup="menu"
                aria-expanded={showImportMenu}
                >Import{@render chevronUp('h-[11px] w-[11px]')}</button
            >
            {#if showImportMenu}
                {#snippet importItems()}
                    {#each importOptions as option}
                        {@render menuItem(option.label, option.ext, () =>
                            importThemeFile(option.type)
                        )}
                    {/each}
                {/snippet}
                {@render menuPanel('left', 'min-w-[200px]', importItems)}
            {/if}
        </div>
        <button
            type="button"
            class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-[30px] px-2.5 text-[12px] transition-colors"
            onclick={() => {
                closeMenus();
                showExportDialog = true;
            }}>Export</button
        >
        <button
            type="button"
            class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-[30px] px-2.5 text-[12px] transition-colors"
            onclick={() => {
                closeMenus();
                showSaveDialog = true;
            }}>Save</button
        >
        {@render divider()}
        <button
            type="button"
            class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary disabled:text-fg-dimmed flex h-[30px] w-[30px] items-center justify-center transition-colors disabled:cursor-default disabled:bg-transparent disabled:opacity-45"
            onclick={undoAction}
            disabled={!undoEnabled}
            aria-label="Undo"
            title="Undo (Ctrl+Z)"
        >
            <svg
                class="h-[15px] w-[15px]"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
                ><path d="M9 14 4 9l5-5 M4 9h10.5a5.5 5.5 0 0 1 0 11H11"
                ></path></svg
            >
        </button>
        <button
            type="button"
            class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary disabled:text-fg-dimmed flex h-[30px] w-[30px] items-center justify-center transition-colors disabled:cursor-default disabled:bg-transparent disabled:opacity-45"
            onclick={redoAction}
            disabled={!redoEnabled}
            aria-label="Redo"
            title="Redo (Ctrl+Shift+Z)"
        >
            <svg
                class="h-[15px] w-[15px]"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
                ><path d="m15 14 5-5-5-5 M20 9H9.5a5.5 5.5 0 0 0 0 11H13"
                ></path></svg
            >
        </button>

        <div
            class="text-fg-dimmed flex min-w-0 flex-1 items-center justify-center gap-3.5 overflow-hidden whitespace-nowrap text-[11.5px]"
        >
            {#if dirty}
                <span class="text-fg-secondary flex items-center gap-1.5">
                    <span class="bg-warning h-1.5 w-1.5" aria-hidden="true"
                    ></span>
                    Unsaved changes
                </span>
            {/if}
            {#if overrideCount > 0}
                <span
                    class="text-accent"
                    title="Active per-app template overrides"
                >
                    {overrideCount} override{overrideCount === 1 ? '' : 's'}
                </span>
            {/if}
            {#if !isOmarchy}
                <button
                    type="button"
                    class="hover:text-fg-primary transition-colors {targetsVisible
                        ? 'text-fg-secondary'
                        : ''}"
                    onclick={toggleTargetsVisible}
                    title={targetsVisible
                        ? 'Hide the Targets strip'
                        : 'Show the Targets strip'}
                    aria-pressed={targetsVisible}
                >
                    Targets
                </button>
            {/if}
            <span title="Aether version">v{__APP_VERSION__}</span>
        </div>

        <div class="relative">
            <button
                type="button"
                class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary flex h-[30px] w-[30px] items-center justify-center transition-colors"
                onclick={() => {
                    const open = !showMoreMenu;
                    closeMenus();
                    showMoreMenu = open;
                }}
                aria-label="More actions"
                aria-haspopup="menu"
                aria-expanded={showMoreMenu}
                title="More actions"
            >
                <svg
                    class="h-4 w-4"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="3"
                    stroke-linecap="round"
                    aria-hidden="true"
                    ><path d="M5 12h.01M12 12h.01M19 12h.01"></path></svg
                >
            </button>
            {#if showMoreMenu}
                {#snippet moreItems()}
                    {#each [{kind: 'revert', label: 'Revert system theme…', hint: 'Roll the desktop back to its default theme'}, {kind: 'reset', label: 'Reset editor…', hint: 'Clear palette, wallpaper and overrides'}] as const as item}
                        <button
                            type="button"
                            class="hover:bg-bg-hover flex w-full flex-col gap-px px-3 py-2 text-left transition-colors"
                            role="menuitem"
                            onclick={() => {
                                closeMenus();
                                confirmKind = item.kind;
                            }}
                        >
                            <span class="text-destructive text-[12px]"
                                >{item.label}</span
                            >
                            <span class="text-fg-dimmed text-[11px]"
                                >{item.hint}</span
                            >
                        </button>
                    {/each}
                {/snippet}
                {@render menuPanel('right', 'w-[260px]', moreItems)}
            {/if}
        </div>
        {@render divider()}

        <Switch
            checked={liveApply}
            onchange={setLiveApply}
            size="sm"
            class="hover:bg-bg-hover h-[30px] pl-2 pr-2.5 text-[12px] transition-colors {liveApply
                ? 'text-fg-primary'
                : 'text-fg-dimmed'}"
            title={!liveApply
                ? 'Auto-apply theme on every edit (debounced)'
                : livePending
                  ? 'Live preview: the latest changes are syncing'
                  : 'Live preview on. Every edit applies automatically. Press Ctrl+Z to revert.'}
        >
            <span class="flex items-center gap-1.5">
                {liveApply && livePending ? 'Syncing' : 'Live'}
                {#if liveApply && livePending}
                    <span
                        class="bg-accent h-1.5 w-1.5 animate-pulse"
                        aria-hidden="true"
                    ></span>
                {/if}
            </span>
        </Switch>

        <div class="relative ml-1 flex">
            <button
                type="button"
                class="bg-accent text-accent-fg hover:bg-accent-hover flex h-8 items-center gap-2.5 px-3.5 text-[12px] font-semibold transition-colors disabled:opacity-50"
                onclick={handleApply}
                disabled={applying}
                title={dirty
                    ? 'Apply updates to the saved theme folder (Ctrl+Enter applies only)'
                    : 'Apply theme (Ctrl+Enter applies only)'}
            >
                <span>{applying ? 'Applying…' : 'Apply theme'}</span>
                {#if !applying}
                    <KbdInverse>Ctrl ↵</KbdInverse>
                {/if}
            </button>
            <button
                type="button"
                class="bg-accent text-accent-fg hover:bg-accent-hover flex h-8 w-7 items-center justify-center shadow-[inset_1px_0_0_rgba(0,0,0,0.18)] transition-colors disabled:opacity-50"
                onclick={() => {
                    const open = !showApplyMenu;
                    closeMenus();
                    showApplyMenu = open;
                }}
                disabled={applying}
                aria-label="More apply options"
                aria-haspopup="menu"
                aria-expanded={showApplyMenu}
                title="More apply options"
            >
                {@render chevronUp('h-3 w-3')}
            </button>
            {#if dirty && !applying}
                <span
                    class="bg-warning absolute -right-[3px] -top-[3px] h-2 w-2 shadow-[0_0_0_2px_var(--color-bg-secondary)]"
                    aria-label="Unsaved changes"
                ></span>
            {/if}
            {#if showApplyMenu}
                {#snippet applyItems()}
                    {@render menuItem(
                        'Save as new folder…',
                        'Ctrl J',
                        saveThemeAsNew
                    )}
                    {@render menuItem('Apply only', 'Ctrl ↵', applyTheme)}
                {/snippet}
                {@render menuPanel('right', 'min-w-[220px]', applyItems)}
            {/if}
        </div>
    {:else}
        <div
            class="text-fg-dimmed flex min-w-0 flex-1 items-center gap-3.5 pl-1 text-[11.5px]"
        >
            {#if activeTab === 'wallhaven'}
                {#if apiKeySet}
                    <span class="text-success">API key set</span>
                {:else}
                    <span>No API key. Add one under the search filters.</span>
                {/if}
                {#if totalResults > 0}
                    <span>{totalResults.toLocaleString()} results</span>
                {/if}
            {:else if activeTab === 'github'}
                <span>Public repositories</span>
            {:else if activeTab === 'local'}
                <button
                    type="button"
                    class="text-fg-secondary hover:text-fg-primary transition-colors"
                    onclick={rescanLocal}>Rescan</button
                >
            {:else if activeTab === 'favorites'}
                <span>Favorited wallpapers</span>
            {:else if activeTab === 'blueprints'}
                <span>Saved themes</span>
            {:else if activeTab === 'system'}
                <span>Native themes</span>
            {:else if activeTab === 'settings'}
                <span>App settings</span>
            {:else if activeTab === 'about'}
                <span>v{__APP_VERSION__}</span>
            {/if}
        </div>
        <button
            type="button"
            class="bg-accent text-accent-fg hover:bg-accent-hover h-8 px-3.5 text-[12px] font-semibold transition-colors"
            onclick={() => setActiveTab('editor')}>Go to editor</button
        >
    {/if}
</footer>

<Modal
    open={showExportDialog}
    onclose={() => (showExportDialog = false)}
    onenter={handleExport}
    bare
    label="Export theme"
    panelClass="flex w-[460px] max-h-[85vh] flex-col"
>
    <DialogHeader
        title="Export theme"
        onclose={() => (showExportDialog = false)}
    />
    <div
        class="flex min-h-0 flex-col gap-[18px] overflow-y-auto px-5 py-[18px]"
    >
        <label class="flex flex-col gap-1.5">
            <span class="text-fg-secondary text-[12px] font-medium"
                >Theme name</span
            >
            <input
                bind:this={exportNameInput}
                type="text"
                class="bg-bg-primary border-border text-fg-primary focus:border-accent h-[34px] border px-2.5 text-[13px] outline-none transition-colors"
                placeholder="e.g. Midnight Aurora"
                bind:value={exportName}
            />
        </label>
        {#if isOmarchy}
            <p class="text-fg-dimmed text-[12px] leading-relaxed">
                Exports a native colors.toml theme. Omarchy generates and
                reloads application themes when it is applied.
            </p>
            {@render checkbox(
                'Install as Omarchy theme',
                installToOmarchy,
                () => (installToOmarchy = !installToOmarchy)
            )}
        {:else}
            <div class="flex flex-col gap-3.5">
                <div class="flex items-baseline justify-between">
                    <span class="text-fg-secondary text-[12px] font-medium"
                        >Include apps</span
                    >
                    <span class="text-fg-dimmed text-[11.5px]">
                        {exportSelectedCount} of {exportAppCount} selected ·
                        <button
                            type="button"
                            class="text-accent hover:text-accent-hover"
                            onclick={() => setAllExportApps(true)}>All</button
                        >
                        ·
                        <button
                            type="button"
                            class="text-accent hover:text-accent-hover"
                            onclick={() => setAllExportApps(false)}>None</button
                        >
                    </span>
                </div>
                {#each exportAppGroups as group}
                    <div>
                        <div
                            class="text-fg-dimmed mb-1.5 text-[10px] font-semibold uppercase tracking-[0.14em]"
                        >
                            {group.label}
                        </div>
                        <div class="grid grid-cols-3 gap-x-2.5 gap-y-0.5">
                            {#each group.apps as app}
                                {@render checkbox(
                                    app.name,
                                    exportApps[app.key],
                                    () =>
                                        (exportApps[app.key] =
                                            !exportApps[app.key])
                                )}
                            {/each}
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
    <DialogFooter>
        <button
            type="button"
            class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 px-3.5 text-[12px] transition-colors"
            onclick={() => (showExportDialog = false)}>Cancel</button
        >
        <button
            type="button"
            class="bg-accent text-accent-fg hover:bg-accent-hover h-8 px-4 text-[12px] font-semibold transition-colors disabled:opacity-50"
            onclick={handleExport}
            disabled={!exportName.trim()}>Export</button
        >
    </DialogFooter>
</Modal>

<SaveDialog
    open={showSaveDialog}
    onclose={() => (showSaveDialog = false)}
    onsave={() => (showSaveDialog = false)}
/>

{#if confirmKind}
    {@const cfg = CONFIRM_CONFIG[confirmKind]}
    <ConfirmDialog
        open={true}
        title={cfg.title}
        body={cfg.body}
        confirmLabel={cfg.confirmLabel}
        danger={true}
        onconfirm={() => {
            const run = cfg.run;
            confirmKind = null;
            run();
        }}
        oncancel={() => (confirmKind = null)}
    />
{/if}
