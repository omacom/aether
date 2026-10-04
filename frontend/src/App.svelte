<script lang="ts">
    import {onMount, onDestroy} from 'svelte';
    import {fade} from 'svelte/transition';
    import HeaderBar from '$lib/components/layout/HeaderBar.svelte';
    import ActionBar from '$lib/components/layout/ActionBar.svelte';
    import TargetAppsStrip from '$lib/components/layout/TargetAppsStrip.svelte';
    import ThemeEditor from '$lib/components/editor/ThemeEditor.svelte';
    import WallhavenBrowser from '$lib/components/wallhaven/WallhavenBrowser.svelte';
    import GitHubBrowser from '$lib/components/github/GitHubBrowser.svelte';
    import LocalBrowser from '$lib/components/local/LocalBrowser.svelte';
    import FavoritesView from '$lib/components/favorites/FavoritesView.svelte';
    import BlueprintsView from '$lib/components/blueprints/BlueprintsView.svelte';
    import OmarchyThemes from '$lib/components/blueprints/OmarchyThemes.svelte';
    import SettingsView from '$lib/components/settings/SettingsView.svelte';
    import AboutView from '$lib/components/layout/AboutView.svelte';
    import ExportProgress from '$lib/components/favorites/ExportProgress.svelte';
    import {initExportEvents} from '$lib/stores/favoritesExport.svelte';
    import {
        getActiveTab,
        setActiveTab,
        showToast,
        closeColorPicker,
        getColorPickerOpen,
        getColorPickerIndex,
        getColorPickerExtKey,
        getColorPickerOverrideApp,
        getColorPickerOverrideRole,
        getEyedropperActive,
        setEyedropperActive,
        getCommandPaletteOpen,
        openCommandPalette,
        closeCommandPalette,
        getKeymapOpen,
        setKeymapOpen,
        toggleKeymap,
        toggleSidebar,
        getLiveApply,
        getLiveApplySession,
        invalidateLiveApplySession,
        setLivePending,
        getTargetsVisible,
        getApplySaveDialogOpen,
        setApplySaveDialogOpen,
        type Tab,
    } from '$lib/stores/ui.svelte';

    const VALID_TABS: readonly Tab[] = [
        'editor',
        'wallhaven',
        'github',
        'local',
        'favorites',
        'blueprints',
        'system',
        'settings',
        'about',
    ] as const;
    const isValidTab = (t: string): t is Tab =>
        (VALID_TABS as readonly string[]).includes(t);
    import {
        setWallpaperPath,
        setWallpaperBlur,
        setPalette,
        setExtendedColors,
        setNativeColors,
        setIconTheme,
        setColor,
        setExtendedColor,
        setAppOverride,
        applyBackendState,
        getThemeSnapshot,
        getThemeSignature,
        getLastAppliedSignature,
        getIsApplying,
        getIsAdjusting,
        getIsExtracting,
        setLastExtractedPath,
        type BackendStatePayload,
    } from '$lib/stores/theme.svelte';
    import {debounce} from '$lib/utils/debounce';
    import {STORAGE_KEYS} from '$lib/constants/storage';
    import {
        applyTheme,
        applyThemeLive,
        requestThemeApply,
        saveThemeAsNew,
        undoAction,
        redoAction,
    } from '$lib/actions/themeActions';
    import {
        zoomIn,
        zoomOut,
        resetZoom,
        setZoom,
        getZoom,
    } from '$lib/utils/zoom';
    import Toast from '$lib/components/shared/Toast.svelte';
    import KeymapDialog from '$lib/components/shared/KeymapDialog.svelte';
    import CommandPalette from '$lib/components/shared/CommandPalette.svelte';
    import ExternalImportDialog from '$lib/components/ExternalImportDialog.svelte';
    import ApplySaveDialog from '$lib/components/layout/ApplySaveDialog.svelte';
    import {initKeyboardShortcuts, registerShortcut} from '$lib/utils/keyboard';
    import {
        hexToRgb,
        isLightRgb,
        wcagAlphaForContrast,
        contrastInk,
    } from '$lib/utils/color';
    import {prefersReducedMotion} from '$lib/utils/browser';
    import {buildCommands} from '$lib/commands/commands.svelte';
    import type {main} from '../wailsjs/go/models';
    import {
        getOmarchyAvailable,
        initOmarchyCapabilities,
    } from '$lib/stores/omarchy.svelte';

    let activeTab = $derived(getActiveTab());
    let commands = $derived(buildCommands());
    let omarchyAvailable = $derived(getOmarchyAvailable());

    void initOmarchyCapabilities();

    const TAB_FADE_DURATION = prefersReducedMotion() ? 0 : 100;

    // RGB step between bg-primary and bg-secondary so panels stay
    // visually layered on any theme bg.
    const BG_SECONDARY_SHIFT = 10;
    // CSS tokens layered over bg-primary, paired with their alpha values
    // for light and dark backgrounds. Order: [token, lightBgAlpha,
    // darkBgAlpha]. The values match the tokens in app.css. Borders need
    // more alpha on a light bg to separate panels from the page.
    const OVERLAY_TOKENS: ReadonlyArray<[string, number, number]> = [
        ['--color-bg-surface', 0.03, 0.035],
        ['--color-bg-elevated', 0.065, 0.075],
        ['--color-bg-hover', 0.05, 0.055],
        ['--color-border', 0.11, 0.075],
        ['--color-border-focus', 0.28, 0.2],
    ];
    // Design alpha for fg-secondary/dimmed; bumped at apply-time when needed
    // to satisfy WCAG against the actual theme bg/fg pair.
    const FG_SECONDARY_DESIGN_ALPHA = 0.74;
    const FG_DIMMED_DESIGN_ALPHA = 0.52;
    const WCAG_AA_RATIO = 4.5;
    const WCAG_AA_LARGE_RATIO = 3;

    // Mirror editor state into Go (debounced) so `aether status` and other IPC
    // readers reflect live edits without waiting for an Apply. Long enough
    // to cover slider drags; first sync on mount posts the initial snapshot.
    const STATE_SYNC_DEBOUNCE_MS = 300;
    const syncStateToBackend = debounce((snapshot: main.SyncStateRequest) => {
        import('../wailsjs/go/main/App')
            .then(({SyncState}) => SyncState(snapshot))
            .catch(() => {});
    }, STATE_SYNC_DEBOUNCE_MS);

    // Send only snapshots that differ from the last mirrored state, so a
    // backend push of equal values does not cause another SyncState call.
    let lastSyncedSignature = '';
    $effect(() => {
        const snapshot = getThemeSnapshot();
        const signature = getThemeSignature(snapshot);
        if (signature === lastSyncedSignature) return;
        lastSyncedSignature = signature;
        syncStateToBackend(snapshot as unknown as main.SyncStateRequest);
    });

    // Arming records the initial state without applying it. Queuing is not an
    // acknowledgement: only a completed apply advances the applied signature.
    const LIVE_APPLY_DEBOUNCE_MS = 1500;
    let initialStateLoaded = $state(false);
    let initialLiveSignature = '';
    type LiveAttempt = {signature: string; session: number};
    let queuedLive: LiveAttempt | null = null;
    let lastLiveAttempt = $state.raw<LiveAttempt | null>(null);
    const debouncedLiveApply = debounce(async () => {
        const attempt = queuedLive;
        queuedLive = null;
        if (
            !attempt ||
            !getLiveApply() ||
            attempt.session !== getLiveApplySession() ||
            attempt.signature !== getThemeSignature() ||
            getIsApplying() ||
            getIsAdjusting() ||
            getIsExtracting()
        )
            return;
        lastLiveAttempt = attempt;
        const result = await applyThemeLive();
        if (result !== 'failed' && lastLiveAttempt === attempt)
            lastLiveAttempt = null;
    }, LIVE_APPLY_DEBOUNCE_MS);
    $effect(() => {
        const enabled = getLiveApply();
        const session = getLiveApplySession();
        const signature = getThemeSignature();
        const applied = getLastAppliedSignature();
        if (
            lastLiveAttempt &&
            (lastLiveAttempt.signature !== signature ||
                lastLiveAttempt.session !== session ||
                lastLiveAttempt.signature === applied)
        ) {
            lastLiveAttempt = null;
        }
        if (!initialStateLoaded) initialLiveSignature = signature;
        if (
            !initialStateLoaded ||
            !enabled ||
            signature === (applied || initialLiveSignature)
        ) {
            debouncedLiveApply.cancel();
            queuedLive = null;
            setLivePending(false);
            return;
        }
        if (getIsApplying() || getIsAdjusting() || getIsExtracting()) {
            debouncedLiveApply.cancel();
            queuedLive = null;
            setLivePending(true);
            return;
        }
        // Don't loop on a failed request. A new edit or OFF/ON session retries it.
        if (
            lastLiveAttempt?.signature === signature &&
            lastLiveAttempt.session === session
        ) {
            setLivePending(false);
            return;
        }
        queuedLive = {signature, session};
        setLivePending(true);
        debouncedLiveApply();
    });

    onDestroy(() => {
        cleanupFileDrop?.();
        invalidateLiveApplySession();
        debouncedLiveApply.cancel();
        syncStateToBackend.cancel();
        setLivePending(false);
    });

    // Dropped files arrive from Wails as absolute paths. HandleDroppedFiles
    // picks the first supported image and rejects everything else.
    let cleanupFileDrop: (() => void) | null = null;

    async function handleWallpaperDrop(paths: string[]): Promise<void> {
        try {
            const {HandleDroppedFiles} = await import('../wailsjs/go/main/App');
            const path = await HandleDroppedFiles(paths);
            setActiveTab('editor');
            setWallpaperPath(path);
            showToast('Wallpaper changed — click Extract to generate palette');
        } catch {
            showToast('Drop an image file to use as wallpaper');
        }
    }

    onMount(async () => {
        // Register file drops before showing the window. The runtime installs
        // preventDefault handlers for file drags, which stops WebKit from
        // navigating to the dropped file instead of keeping the UI.
        try {
            const {OnFileDrop, OnFileDropOff} = await import(
                '../wailsjs/runtime/runtime'
            );
            const guardEvents: string[] = [
                'drag',
                'dragenter',
                'dragstart',
                'dragend',
            ];
            const preventFileNavigation = (event: Event) => {
                const dragEvent = event as DragEvent;
                if (dragEvent.dataTransfer?.types.includes('Files')) {
                    dragEvent.preventDefault();
                }
            };
            for (const name of guardEvents) {
                window.addEventListener(name, preventFileNavigation);
            }
            OnFileDrop((_x, _y, paths) => {
                void handleWallpaperDrop(paths);
            }, false);
            cleanupFileDrop = () => {
                OnFileDropOff();
                for (const name of guardEvents) {
                    window.removeEventListener(name, preventFileNavigation);
                }
            };
        } catch (e) {
            console.warn('File drop setup failed:', e);
        }

        // Show window now that the DOM is ready (started hidden to avoid white flash)
        try {
            const {WindowShow} = await import('../wailsjs/runtime/runtime');
            WindowShow();
        } catch {}

        await initOmarchyCapabilities();

        // Focus a specific tab if requested via --tab flag
        try {
            const {GetFocusTab} = await import('../wailsjs/go/main/App');
            const tab = await GetFocusTab();
            if (tab && isValidTab(tab)) {
                setActiveTab(
                    tab === 'system' && !getOmarchyAvailable() ? 'editor' : tab
                );
            }
        } catch {}

        // Overwrite module-load DEFAULT_PALETTE with backend defaults
        // before any user interaction so history starts clean.
        try {
            const {GetInitialState} = await import('../wailsjs/go/main/App');
            const s = await GetInitialState();
            if (s?.palette?.length >= 16) {
                setPalette(s.palette, true /* skipHistory */);
            }
            if (s?.extendedColors) {
                setExtendedColors(s.extendedColors);
            }
            if (s?.nativeColors) {
                setNativeColors(s.nativeColors);
            }
            setIconTheme(s?.iconTheme, true);
            if (s?.wallpaperPath) {
                setWallpaperPath(s.wallpaperPath);
                // Treat the restored wallpaper as already-extracted so a
                // re-extract on the same image doesn't clear overrides.
                setLastExtractedPath(s.wallpaperPath);
            }
            setWallpaperBlur(!!s?.wallpaperBlur, true);
        } catch (e) {
            console.warn('GetInitialState failed:', e);
        }
        initialLiveSignature = getThemeSignature();
        initialStateLoaded = true;

        initKeyboardShortcuts();

        registerShortcut('ctrl+z', undoAction);
        registerShortcut('ctrl+shift+z', redoAction);
        registerShortcut('ctrl+enter', applyTheme);
        registerShortcut('ctrl+j', saveThemeAsNew);

        // Ctrl+S - Save blueprint
        registerShortcut('ctrl+s', () => {
            showToast('Use the Blueprints tab to save');
        });

        // Shift+C - Copy hex from active color picker
        registerShortcut('shift+c', () => {
            if (!getColorPickerOpen()) return;
            // Read the hex value straight from the picker's input field
            const input = document.querySelector<HTMLInputElement>(
                '[data-color-hex-input]'
            );
            const hex = input?.value || '';
            if (hex) {
                navigator.clipboard.writeText(hex).then(() => {
                    showToast(`Copied ${hex}`);
                });
            }
        });

        // Shift+V - Paste hex into active swatch
        registerShortcut('shift+v', () => {
            if (!getColorPickerOpen()) return;
            navigator.clipboard.readText().then(text => {
                let hex = text.trim();
                // Accept with or without # prefix
                if (/^[0-9a-fA-F]{6}$/.test(hex)) {
                    hex = '#' + hex;
                }
                if (!/^#[0-9a-fA-F]{6}$/.test(hex)) {
                    showToast('Clipboard is not a valid hex color');
                    return;
                }
                const overrideApp = getColorPickerOverrideApp();
                const overrideRole = getColorPickerOverrideRole();
                const extKey = getColorPickerExtKey();
                if (overrideApp && overrideRole) {
                    setAppOverride(overrideApp, overrideRole, hex);
                } else if (extKey) {
                    setExtendedColor(extKey, hex);
                } else {
                    setColor(getColorPickerIndex(), hex);
                }
                showToast(`Pasted ${hex}`);
            });
        });

        setZoom(getZoom());
        registerShortcut('ctrl+shift++', zoomIn);
        registerShortcut('ctrl++', zoomIn);
        registerShortcut('ctrl+-', zoomOut);
        registerShortcut('ctrl+0', resetZoom);

        registerShortcut('ctrl+b', toggleSidebar);
        registerShortcut('ctrl+k', toggleKeymap);
        // ? mirrors Ctrl+K — universal convention for "show shortcuts".
        registerShortcut('shift+?', toggleKeymap);
        registerShortcut('ctrl+p', () => {
            if (getCommandPaletteOpen()) closeCommandPalette();
            else openCommandPalette();
        });

        registerShortcut('escape', () => {
            if (getCommandPaletteOpen()) closeCommandPalette();
            else if (getEyedropperActive()) setEyedropperActive(false);
            else if (getColorPickerOpen()) closeColorPicker();
            else if (getKeymapOpen()) setKeymapOpen(false);
        });

        // Favorites export progress. Wired here rather than in FavoritesView
        // so an export keeps reporting after the user switches tabs.
        initExportEvents().catch(error =>
            console.error('Favorites export events unavailable:', error)
        );

        // Listen for events from Go
        (async () => {
            try {
                const {EventsOn, WindowSetBackgroundColour} = await import(
                    '../wailsjs/runtime/runtime'
                );

                const applyThemeColors = (colors: Record<string, string>) => {
                    const root = document.documentElement;
                    for (const [name, value] of Object.entries(colors)) {
                        root.style.setProperty(`--aether-${name}`, value);
                    }
                    if (colors.background) {
                        const bg = hexToRgb(colors.background);
                        const isLightBg = isLightRgb(bg.r, bg.g, bg.b);
                        const shift = isLightBg
                            ? -BG_SECONDARY_SHIFT
                            : BG_SECONDARY_SHIFT;
                        const clamp = (n: number) =>
                            Math.max(0, Math.min(255, n));
                        const overlay = isLightBg ? '0, 0, 0' : '255, 255, 255';
                        root.style.setProperty(
                            '--color-bg-primary',
                            colors.background
                        );
                        root.style.setProperty(
                            '--color-bg-secondary',
                            `rgb(${clamp(bg.r + shift)}, ${clamp(bg.g + shift)}, ${clamp(bg.b + shift)})`
                        );
                        for (const [token, lightA, darkA] of OVERLAY_TOKENS) {
                            root.style.setProperty(
                                token,
                                `rgba(${overlay}, ${isLightBg ? lightA : darkA})`
                            );
                        }
                        document.body.style.background = colors.background;
                        WindowSetBackgroundColour(bg.r, bg.g, bg.b, 255);
                    }
                    if (colors.foreground) {
                        root.style.setProperty(
                            '--color-fg-primary',
                            colors.foreground
                        );
                        // Pick alpha that satisfies WCAG AA / AA-Large
                        // against the actual theme bg, falling back to the
                        // designed dim levels when contrast budget allows.
                        const fg = hexToRgb(colors.foreground);
                        const bg = colors.background
                            ? hexToRgb(colors.background)
                            : null;
                        const secondaryAlpha = bg
                            ? wcagAlphaForContrast(
                                  fg,
                                  bg,
                                  FG_SECONDARY_DESIGN_ALPHA,
                                  WCAG_AA_RATIO
                              )
                            : FG_SECONDARY_DESIGN_ALPHA;
                        const dimmedAlpha = bg
                            ? wcagAlphaForContrast(
                                  fg,
                                  bg,
                                  FG_DIMMED_DESIGN_ALPHA,
                                  WCAG_AA_LARGE_RATIO
                              )
                            : FG_DIMMED_DESIGN_ALPHA;
                        root.style.setProperty(
                            '--color-fg-secondary',
                            `rgba(${fg.r}, ${fg.g}, ${fg.b}, ${secondaryAlpha})`
                        );
                        root.style.setProperty(
                            '--color-fg-dimmed',
                            `rgba(${fg.r}, ${fg.g}, ${fg.b}, ${dimmedAlpha})`
                        );
                        document.body.style.color = colors.foreground;
                    }
                    const accent = colors.accent || colors.blue;
                    if (accent) {
                        root.style.setProperty('--color-accent', accent);
                        // Pick black or white text for content sitting on
                        // the accent button, based on accent luminance.
                        root.style.setProperty(
                            '--color-accent-fg',
                            contrastInk(accent)
                        );
                    }
                    if (colors.red) {
                        root.style.setProperty(
                            '--color-destructive',
                            colors.red
                        );
                        // Same black/white-by-luminance choice as accent-fg,
                        // so text on a destructive button stays legible on an
                        // arbitrary extracted red (light pink needs dark text,
                        // deep red needs white).
                        root.style.setProperty(
                            '--color-destructive-fg',
                            contrastInk(colors.red)
                        );
                    }
                    if (colors.green) {
                        root.style.setProperty('--color-success', colors.green);
                    }
                    if (colors.yellow) {
                        root.style.setProperty(
                            '--color-warning',
                            colors.yellow
                        );
                    }
                    // Read synchronously by index.html at first paint to
                    // avoid a flash before the Wails bridge is ready.
                    try {
                        localStorage.setItem(
                            STORAGE_KEYS.themeColors,
                            JSON.stringify(colors)
                        );
                    } catch {}
                };

                EventsOn('theme-colors-changed', applyThemeColors);

                // Pull before subscribing-is-too-late: EventsOn attaches
                // after the watcher's startup emit has already fired.
                try {
                    const {GetThemeColors} = await import(
                        '../wailsjs/go/main/App'
                    );
                    const colors = await GetThemeColors();
                    if (colors && Object.keys(colors).length > 0) {
                        applyThemeColors(colors);
                    }
                } catch {}

                // Listen for IPC remote control state changes. The applier
                // skips fields that already match, so an echo of our own
                // SyncState cannot re-arm the sync effect above.
                EventsOn('ipc-state-changed', (state: BackendStatePayload) => {
                    applyBackendState(state);
                });
            } catch {}
        })();
    });
</script>

<div class="bg-bg-primary flex h-screen flex-col">
    <HeaderBar />
    <main class="flex-1 overflow-hidden">
        {#key activeTab}
            <div class="h-full" in:fade={{duration: TAB_FADE_DURATION}}>
                {#if activeTab === 'editor'}
                    <ThemeEditor />
                {:else if activeTab === 'wallhaven'}
                    <WallhavenBrowser />
                {:else if activeTab === 'github'}
                    <GitHubBrowser />
                {:else if activeTab === 'local'}
                    <LocalBrowser />
                {:else if activeTab === 'favorites'}
                    <FavoritesView />
                {:else if activeTab === 'blueprints'}
                    <BlueprintsView />
                {:else if activeTab === 'system'}
                    <OmarchyThemes />
                {:else if activeTab === 'settings'}
                    <SettingsView />
                {:else if activeTab === 'about'}
                    <AboutView />
                {/if}
            </div>
        {/key}
    </main>
    {#if activeTab === 'editor' && getTargetsVisible() && !omarchyAvailable}
        <TargetAppsStrip />
    {/if}
    <ActionBar />
    <ExportProgress />
    <Toast />
    <KeymapDialog open={getKeymapOpen()} onclose={() => setKeymapOpen(false)} />
    <CommandPalette
        open={getCommandPaletteOpen()}
        {commands}
        onclose={closeCommandPalette}
    />
    <ExternalImportDialog />
    <ApplySaveDialog
        open={getApplySaveDialogOpen()}
        onclose={() => setApplySaveDialogOpen(false)}
    />
</div>
