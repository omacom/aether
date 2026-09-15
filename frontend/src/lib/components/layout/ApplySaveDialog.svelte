<script lang="ts">
    import {onDestroy} from 'svelte';
    import {getWallpaperPath} from '$lib/stores/theme.svelte';
    import {
        captureApplyRequest,
        saveAndApplyTheme,
    } from '$lib/actions/themeActions';
    import Modal from '$lib/components/shared/Modal.svelte';

    let {open, onclose}: {open: boolean; onclose: () => void} = $props();
    let name = $state('');
    let busy = $state(false);
    let error = $state('');
    let nameInput = $state<HTMLInputElement | null>(null);
    let pending = $state.raw<{
        name: string;
        request: ReturnType<typeof captureApplyRequest>;
    } | null>(null);
    let revision = 0;
    let opened = false;
    const validName = (value: string) =>
        /^[a-z0-9][a-z0-9-]{0,63}$/.test(value);

    function suggestedName() {
        const base = (getWallpaperPath().split(/[\\/]/).pop() ?? '').replace(
            /\.[^.]+$/,
            ''
        );
        const suggestion = base
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '')
            .slice(0, 64);
        return validName(suggestion) ? suggestion : 'theme';
    }

    $effect(() => {
        if (!open) {
            opened = false;
            revision++;
            name = '';
            pending = null;
            busy = false;
            error = '';
        } else {
            if (!opened) {
                opened = true;
                name = suggestedName();
            }
            if (!pending) nameInput?.focus();
        }
    });
    onDestroy(() => revision++);

    function handleClose() {
        if (busy) return;
        if (pending) {
            pending = null;
            error = '';
        } else {
            onclose();
        }
    }

    async function save(updateExisting = false) {
        if (!open || busy || (!updateExisting && pending)) return;
        const candidate = pending ?? {
            name: name.trim().toLowerCase(),
            request: captureApplyRequest(),
        };
        if (!validName(candidate.name)) {
            nameInput?.focus();
            return;
        }
        const id = ++revision;
        busy = true;
        error = '';
        try {
            if (!updateExisting) {
                const {ThemeFolderExists} = await import(
                    '../../../../wailsjs/go/main/App'
                );
                const exists = await ThemeFolderExists(candidate.name);
                if (!open || id !== revision) return;
                if (exists) {
                    pending = candidate;
                    return;
                }
            }
            if (!open || id !== revision) return;
            const saved = await saveAndApplyTheme(
                candidate.name,
                updateExisting,
                candidate.request
            );
            if (!open || id !== revision) return;
            if (saved) onclose();
            else
                error =
                    'Could not save the theme. Check the error message and try again.';
        } catch (failure: unknown) {
            if (id === revision)
                error =
                    failure instanceof Error
                        ? failure.message
                        : String(failure);
        } finally {
            if (id === revision) busy = false;
        }
    }
</script>

<Modal
    {open}
    onclose={handleClose}
    onenter={() => {
        if (!pending) void save();
    }}
>
    <h3 class="text-fg-primary mb-2 text-[12px] font-medium">
        {pending ? 'Update Theme Folder' : 'Save Theme Folder'}
    </h3>
    {#if pending}
        <p class="text-fg-secondary mb-3 text-[11px]">
            A theme folder named <strong>{pending.name}</strong> already exists.
            Update this folder and apply the captured theme?
        </p>
    {:else}
        <p class="text-fg-dimmed mb-3 text-[11px] leading-relaxed">
            Use lowercase letters, digits, and hyphens.
        </p>
        <input
            bind:this={nameInput}
            bind:value={name}
            type="text"
            maxlength="64"
            class="bg-bg-surface text-fg-primary focus:border-border-focus w-full border px-2 py-1.5 text-[12px] outline-none {name &&
            !validName(name)
                ? 'border-destructive'
                : 'border-border'}"
            oninput={() =>
                (name = name.replace(/[^a-zA-Z0-9-]/g, '').toLowerCase())}
            aria-label="Theme folder name"
            disabled={busy}
        />
    {/if}
    {#if error}<p class="text-destructive mt-2 text-[11px]" role="alert">
            {error}
        </p>{/if}
    <div class="mt-3 flex justify-end gap-2">
        <button
            type="button"
            class="text-fg-secondary px-3 py-1.5 text-[11px] disabled:opacity-50"
            onclick={handleClose}
            disabled={busy}>{pending ? 'Back' : 'Cancel'}</button
        >
        <button
            type="button"
            class="bg-accent hover:bg-accent-hover text-accent-fg px-3 py-1.5 text-[11px] font-medium disabled:opacity-50"
            onclick={() => save(!!pending)}
            disabled={busy || (!pending && !validName(name))}
            >{busy
                ? 'Please wait…'
                : pending
                  ? 'Update and Apply'
                  : 'Save and Apply'}</button
        >
    </div>
</Modal>
