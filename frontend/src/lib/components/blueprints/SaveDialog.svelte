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

<Modal {open} onclose={handleClose} onenter={handleEnter} z="z-40">
    {#if showOverrideConfirm}
        <h3 class="text-fg-primary mb-3 text-[12px] font-medium">
            Override existing theme?
        </h3>
        <p class="text-fg-dimmed mb-3 text-[11px]">
            A theme named "{pendingSave?.name}" already exists.
        </p>
        <div class="flex justify-end gap-2">
            <button
                class="text-fg-dimmed hover:text-fg-secondary px-3 py-1.5 text-[11px] transition-colors"
                disabled={isSaving}
                onclick={handleClose}>Cancel</button
            >
            <button
                class="bg-accent hover:bg-accent-hover text-accent-fg px-3 py-1.5 text-[11px] font-medium transition-colors disabled:opacity-50"
                onclick={handleOverride}
                disabled={isSaving}
                >{isSaving ? 'Saving...' : 'Override'}</button
            >
        </div>
    {:else}
        <h3 class="text-fg-primary mb-3 text-[12px] font-medium">Save Theme</h3>
        <input
            bind:this={nameInput}
            type="text"
            class="bg-bg-surface text-fg-primary focus:border-border-focus w-full border px-2 py-1.5 text-[12px] outline-none {nameError
                ? 'border-destructive'
                : 'border-border'}"
            placeholder="Theme name..."
            bind:value={name}
            disabled={isSaving}
            aria-invalid={!!nameError}
            aria-describedby={nameError ? 'save-name-error' : undefined}
        />
        {#if nameError}
            <p id="save-name-error" class="text-destructive mt-1 text-[10px]">
                {nameError}
            </p>
        {/if}
        <div class="mt-3 flex justify-end gap-2">
            <button
                class="text-fg-dimmed hover:text-fg-secondary px-3 py-1.5 text-[11px] transition-colors"
                disabled={isSaving}
                onclick={handleClose}>Cancel</button
            >
            <button
                class="bg-accent hover:bg-accent-hover text-accent-fg px-3 py-1.5 text-[11px] font-medium transition-colors disabled:opacity-50"
                onclick={handleSave}
                disabled={!name.trim() || isSaving}
                >{isSaving ? 'Saving...' : 'Save'}</button
            >
        </div>
    {/if}
</Modal>
