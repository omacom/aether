<script lang="ts">
    import {onDestroy} from 'svelte';
    import type {main} from '../../../../wailsjs/go/models';
    import {showToast} from '$lib/stores/ui.svelte';
    import {
        getPalette,
        getWallpaperPath,
        getWallpaperBlur,
        getLightMode,
        getAdditionalImages,
        getExtendedColors,
        getNativeColors,
        getIconTheme,
        getAppOverrides,
        getAdjustments,
    } from '$lib/stores/theme.svelte';
    import Modal from '$lib/components/shared/Modal.svelte';
    import DialogFooter from '$lib/components/shared/DialogFooter.svelte';

    let {
        open = true,
        onclose,
        onsave,
    }: {open?: boolean; onclose: () => void; onsave: () => void} = $props();
    let name = $state('');
    let isSaving = $state(false);
    let showOverrideConfirm = $state(false);
    let nameInput = $state<HTMLInputElement | null>(null);
    let attemptedSubmit = $state(false);
    let pendingSave = $state.raw<main.SaveBlueprintRequest | null>(null);
    let requestId = 0;

    $effect(() => {
        if (!open) {
            requestId++;
            pendingSave = null;
            showOverrideConfirm = false;
            isSaving = false;
        }
    });
    onDestroy(() => requestId++);

    $effect(() => {
        if (open && !showOverrideConfirm) nameInput?.focus();
    });

    let nameError = $derived(
        attemptedSubmit && !name.trim() ? 'Theme name is required' : ''
    );

    // Esc/backdrop dismissal goes through Modal's onclose, which we override
    // to step back from the override-confirm sub-panel before fully closing.
    function handleClose() {
        if (isSaving) return;
        if (showOverrideConfirm) {
            showOverrideConfirm = false;
            pendingSave = null;
        } else {
            onclose();
        }
    }

    function handleEnter() {
        if (!showOverrideConfirm) handleSave();
    }

    async function handleSave() {
        if (!open || isSaving || showOverrideConfirm) return;
        attemptedSubmit = true;
        if (!name.trim()) {
            nameInput?.focus();
            return;
        }
        isSaving = true;
        const id = ++requestId;
        // The existence check and any explicit override must use this exact payload.
        const request: main.SaveBlueprintRequest = {
            name: name.trim(),
            palette: [...getPalette()],
            wallpaperPath: getWallpaperPath(),
            wallpaperBlur: getWallpaperBlur(),
            lightMode: getLightMode(),
            additionalImages: [...getAdditionalImages()],
            lockedColors: [],
            extendedColors: {...getExtendedColors()},
            nativeColors: {...getNativeColors()},
            iconTheme: {...getIconTheme()},
            appOverrides: Object.fromEntries(
                Object.entries(getAppOverrides()).map(([app, colors]) => [
                    app,
                    {...colors},
                ])
            ),
            adjustments: {...getAdjustments()},
        } as unknown as main.SaveBlueprintRequest;
        pendingSave = request;
        try {
            const {BlueprintExists} = await import(
                '../../../../wailsjs/go/main/App'
            );
            if (id !== requestId || !open) return;
            const exists = await BlueprintExists(request.name);
            if (id !== requestId || !open) return;
            if (exists) {
                showOverrideConfirm = true;
                return;
            }
            await doSave(request, id);
        } catch {
            if (id === requestId) showToast('Failed to save');
        } finally {
            if (id === requestId) isSaving = false;
        }
    }

    async function handleOverride() {
        if (!open || isSaving || !showOverrideConfirm || !pendingSave) return;
        isSaving = true;
        await doSave(pendingSave, ++requestId);
    }

    async function doSave(request: main.SaveBlueprintRequest, id: number) {
        try {
            const {SaveBlueprint} = await import(
                '../../../../wailsjs/go/main/App'
            );
            if (id !== requestId || !open) return;
            await SaveBlueprint(request);
            if (id !== requestId || !open) return;
            pendingSave = null;
            showOverrideConfirm = false;
            showToast(`Saved: ${request.name}`);
            onsave();
        } catch {
            if (id === requestId) showToast('Failed to save');
        } finally {
            if (id === requestId) isSaving = false;
        }
    }
</script>

<Modal
    {open}
    onclose={handleClose}
    onenter={handleEnter}
    z="z-40"
    bare
    panelClass="w-[420px]"
    label={showOverrideConfirm ? 'Override existing theme' : 'Save blueprint'}
>
    <div class="flex h-1.5" aria-hidden="true">
        {#each getPalette() as color}
            <span class="flex-1" style:background-color={color}></span>
        {/each}
    </div>
    {#if showOverrideConfirm}
        <div class="px-5 pb-5 pt-[18px]">
            <h3 class="text-fg-primary mb-1 text-[14px] font-semibold">
                Override existing theme?
            </h3>
            <p class="text-fg-dimmed text-[12px] leading-relaxed">
                A theme named "{pendingSave?.name}" already exists.
            </p>
        </div>
        <DialogFooter>
            <button
                type="button"
                class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 px-3.5 text-[12px] transition-colors disabled:opacity-50"
                disabled={isSaving}
                onclick={handleClose}>Cancel</button
            >
            <button
                type="button"
                class="bg-accent hover:bg-accent-hover text-accent-fg h-8 px-4 text-[12px] font-semibold transition-colors disabled:opacity-50"
                onclick={handleOverride}
                disabled={isSaving}
                >{isSaving ? 'Saving...' : 'Override'}</button
            >
        </DialogFooter>
    {:else}
        <div class="flex flex-col gap-3.5 px-5 pb-5 pt-[18px]">
            <div>
                <h3 class="text-fg-primary mb-1 text-[14px] font-semibold">
                    Save blueprint
                </h3>
                <p class="text-fg-dimmed text-[12px] leading-relaxed">
                    Saves the current palette, adjustments, overrides, and
                    wallpaper so you can load them again later.
                </p>
            </div>
            <label class="flex flex-col gap-1.5">
                <span class="text-fg-secondary text-[12px] font-medium"
                    >Theme name</span
                >
                <input
                    bind:this={nameInput}
                    type="text"
                    class="bg-bg-primary text-fg-primary h-[34px] border px-2.5 text-[13px] outline-none disabled:opacity-60 {nameError
                        ? 'border-destructive'
                        : 'border-border focus:border-accent'}"
                    placeholder="e.g. Midnight Aurora"
                    bind:value={name}
                    disabled={isSaving}
                    aria-invalid={!!nameError}
                    aria-describedby={nameError ? 'save-name-error' : undefined}
                />
            </label>
            {#if nameError}
                <p
                    id="save-name-error"
                    class="text-destructive -mt-2 text-[11px]"
                >
                    {nameError}
                </p>
            {/if}
        </div>
        <DialogFooter>
            <button
                type="button"
                class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 px-3.5 text-[12px] transition-colors disabled:opacity-50"
                disabled={isSaving}
                onclick={handleClose}>Cancel</button
            >
            <button
                type="button"
                class="bg-accent hover:bg-accent-hover text-accent-fg h-8 px-4 text-[12px] font-semibold transition-colors disabled:opacity-50"
                onclick={handleSave}
                disabled={!name.trim() || isSaving}
                >{isSaving ? 'Saving...' : 'Save'}</button
            >
        </DialogFooter>
    {/if}
</Modal>
