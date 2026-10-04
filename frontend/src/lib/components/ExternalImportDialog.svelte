<script lang="ts">
    import Modal from '$lib/components/shared/Modal.svelte';
    import DialogFooter from '$lib/components/shared/DialogFooter.svelte';
    import {showToast, setActiveTab} from '$lib/stores/ui.svelte';
    import {
        loadFullImage,
        getCachedFullImage,
    } from '$lib/stores/imagecache.svelte';
    import {
        ConfirmExternalImport,
        OpenExternalImportInEditor,
        CancelExternalImport,
        GetPendingExternalImport,
    } from '../../../wailsjs/go/main/App';
    import {EventsOn} from '../../../wailsjs/runtime/runtime';
    import {onMount} from 'svelte';

    type Preview = {
        has_external_theme: boolean;
        has_colors: boolean;
        has_wallpaper: boolean;
        source_url: string;
        palette?: string[];
        wallpaper?: string;
        theme_name?: string;
        mode?: string;
        edit?: boolean;
        omarchy_theme_name?: string;
    };

    let preview = $state<Preview | null>(null);
    let wallpaperDataUrl = $state('');
    let isApplying = $state(false);
    // edit=true imports load into the editor instead of applying.
    let isEdit = $derived(!!preview?.edit);
    let isOmarchyInstall = $derived(!isEdit && !!preview?.omarchy_theme_name);

    onMount(() => {
        // The backend emits this on startup when a staged file is present,
        // and on each IPC pending-import while the GUI is already running.
        const cleanup = EventsOn('external-import-requested', (p: Preview) => {
            preview = p;
            primeWallpaper(p);
        });
        // Also pull on mount in case the event fired before we subscribed.
        GetPendingExternalImport().then(p => {
            if (p) {
                preview = p as Preview;
                primeWallpaper(preview);
            }
        });
        return cleanup;
    });

    async function primeWallpaper(p: Preview | null) {
        wallpaperDataUrl = '';
        if (!p?.wallpaper) return;
        const cached = getCachedFullImage(p.wallpaper);
        if (cached) {
            wallpaperDataUrl = cached;
            return;
        }
        try {
            wallpaperDataUrl = await loadFullImage(p.wallpaper);
        } catch {
            wallpaperDataUrl = '';
        }
    }

    let assetKind = $derived(() => {
        if (!preview) return '';
        const parts: string[] = [];
        if (preview.has_external_theme) parts.push('External theme');
        else if (preview.has_colors && preview.has_wallpaper)
            parts.push('Colors + wallpaper');
        else if (preview.has_colors) parts.push('Colors');
        else if (preview.has_wallpaper) parts.push('Wallpaper');
        else parts.push('Theme');
        if (preview.mode === 'light') parts.push('light mode');
        else if (preview.mode === 'dark') parts.push('dark mode');
        return parts.join(' · ');
    });

    let displayHost = $derived(() => {
        if (!preview?.source_url) return '';
        try {
            const u = new URL(preview.source_url);
            return u.searchParams
                .toString()
                .replace(/&/g, '\n')
                .replace(/=/g, ' = ');
        } catch {
            return preview.source_url;
        }
    });

    // edit=true links load the colors + wallpaper into the editor without
    // applying, so the user can tweak before hitting Apply themselves. Any
    // other link applies the theme immediately on confirm.
    async function handleConfirm() {
        if (!preview) return;
        const edit = isEdit; // snapshot before the await nulls preview
        const omarchyInstall = isOmarchyInstall;
        const sourceUrl = preview.source_url;
        isApplying = true;
        try {
            if (edit) {
                await OpenExternalImportInEditor(sourceUrl);
                setActiveTab('editor');
                showToast('Loaded into editor');
            } else {
                await ConfirmExternalImport(sourceUrl);
                showToast(
                    omarchyInstall ? 'Omarchy theme installed' : 'Applied'
                );
            }
            if (preview?.source_url === sourceUrl) preview = null;
        } catch (err) {
            showToast(edit ? 'Failed to load' : 'Failed to apply');
            // eslint-disable-next-line no-console
            console.error('external-import confirm:', err);
        } finally {
            isApplying = false;
        }
    }

    async function handleCancel() {
        const sourceUrl = preview?.source_url ?? '';
        try {
            await CancelExternalImport(sourceUrl);
        } catch {
            // best-effort
        }
        if (preview?.source_url === sourceUrl) preview = null;
    }
</script>

<Modal
    open={preview !== null}
    onclose={handleCancel}
    onenter={handleConfirm}
    panelClass="w-[440px]"
    z="z-50"
    bare
    label="Theme from web"
>
    {#if preview}
        <div class="flex flex-col gap-3.5 px-5 pb-5 pt-[18px]">
            <div>
                <p
                    class="text-fg-dimmed mb-1.5 text-[10px] font-semibold uppercase tracking-[0.14em]"
                >
                    {assetKind()}
                </p>
                <h3 class="text-fg-primary text-[14px] font-semibold">
                    {isEdit
                        ? 'Open theme from web in editor?'
                        : isOmarchyInstall
                          ? 'Install Omarchy theme from web?'
                          : 'Apply theme from web?'}
                </h3>
            </div>

            {#if preview.palette && preview.palette.length > 0}
                <div class="border-border grid grid-cols-8 border">
                    {#each preview.palette.slice(0, 16) as color, i (i)}
                        <div
                            class="h-6"
                            style="background:{color || 'transparent'}"
                            title={color}
                        ></div>
                    {/each}
                </div>
            {/if}

            {#if preview.has_wallpaper && wallpaperDataUrl}
                <div class="border-border border">
                    <img
                        src={wallpaperDataUrl}
                        alt="Wallpaper preview"
                        class="block h-32 w-full object-cover"
                    />
                </div>
            {/if}

            {#if preview.theme_name || isOmarchyInstall}
                <dl
                    class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-[12px]"
                >
                    {#if preview.theme_name}
                        <dt class="text-fg-dimmed">Name</dt>
                        <dd class="text-fg-primary truncate">
                            {preview.theme_name}
                        </dd>
                    {/if}
                    {#if isOmarchyInstall}
                        <dt class="text-fg-dimmed">Install as</dt>
                        <dd class="text-fg-primary truncate">
                            {preview.omarchy_theme_name}
                        </dd>
                    {/if}
                </dl>
            {/if}

            <p
                class="bg-bg-primary border-border text-fg-secondary whitespace-pre-line break-all border px-2.5 py-2 font-mono text-[11px]"
            >
                {displayHost()}
            </p>

            <p class="text-fg-dimmed text-[12px] leading-relaxed">
                {#if isEdit}
                    Aether will load these colors and wallpaper into the editor.
                    Nothing is applied until you click Apply.
                {:else if isOmarchyInstall}
                    Aether will create and activate this named Omarchy theme. An
                    existing theme with the same name will not be overwritten.
                {:else}
                    Aether will replace your palette and background. Only apply
                    from sources you trust.
                {/if}
            </p>
        </div>

        <DialogFooter>
            <button
                type="button"
                class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 px-3.5 text-[12px] transition-colors disabled:opacity-50"
                onclick={handleCancel}
                disabled={isApplying}
            >
                Cancel
            </button>
            <button
                type="button"
                class="bg-accent hover:bg-accent-hover text-accent-fg h-8 px-4 text-[12px] font-semibold transition-colors disabled:opacity-50"
                onclick={handleConfirm}
                disabled={isApplying}
            >
                {#if isEdit}
                    {isApplying ? 'Opening...' : 'Open in editor'}
                {:else if isOmarchyInstall}
                    {isApplying ? 'Installing...' : 'Install and apply'}
                {:else}
                    {isApplying ? 'Applying...' : 'Apply'}
                {/if}
            </button>
        </DialogFooter>
    {/if}
</Modal>
