<script lang="ts">
    import {onDestroy} from 'svelte';
    import {getWallpaperPath} from '$lib/stores/theme.svelte';
    import {
        captureApplyRequest,
        saveAndApplyTheme,
    } from '$lib/actions/themeActions';
    import Modal from '$lib/components/shared/Modal.svelte';
    import DialogFooter from '$lib/components/shared/DialogFooter.svelte';

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
    bare
    panelClass="w-[420px]"
    label={pending ? 'Update theme folder' : 'Save theme folder'}
>
    <div class="flex flex-col gap-3.5 px-5 pb-5 pt-[18px]">
        <div>
            <h3 class="text-fg-primary mb-1 text-[14px] font-semibold">
                {pending ? 'Update theme folder' : 'Save theme folder'}
            </h3>
            {#if pending}
                <p class="text-fg-secondary text-[12px] leading-relaxed">
                    A theme folder named <strong
                        class="text-fg-primary font-semibold"
                        >{pending.name}</strong
                    > already exists. Update this folder and apply the captured theme?
                </p>
            {:else}
                <p class="text-fg-dimmed text-[12px] leading-relaxed">
                    Use lowercase letters, digits, and hyphens.
                </p>
            {/if}
        </div>
        {#if !pending}
            <label class="flex flex-col gap-1.5">
                <span class="text-fg-secondary text-[12px] font-medium"
                    >Folder name</span
                >
                <input
                    bind:this={nameInput}
                    bind:value={name}
                    type="text"
                    maxlength="64"
                    class="bg-bg-primary text-fg-primary h-[34px] border px-2.5 font-mono text-[13px] outline-none disabled:opacity-60 {name &&
                    !validName(name)
                        ? 'border-destructive'
                        : 'border-border focus:border-accent'}"
                    oninput={() =>
                        (name = name
                            .replace(/[^a-zA-Z0-9-]/g, '')
                            .toLowerCase())}
                    aria-label="Theme folder name"
                    disabled={busy}
                />
            </label>
        {/if}
        {#if error}<p class="text-destructive text-[12px]" role="alert">
                {error}
            </p>{/if}
    </div>
    <DialogFooter>
        <button
            type="button"
            class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 px-3.5 text-[12px] transition-colors disabled:opacity-50"
            onclick={handleClose}
            disabled={busy}>{pending ? 'Back' : 'Cancel'}</button
        >
        <button
            type="button"
            class="bg-accent hover:bg-accent-hover text-accent-fg h-8 px-4 text-[12px] font-semibold transition-colors disabled:opacity-50"
            onclick={() => save(!!pending)}
            disabled={busy || (!pending && !validName(name))}
            >{busy
                ? 'Please wait…'
                : pending
                  ? 'Update and Apply'
                  : 'Save and Apply'}</button
        >
    </DialogFooter>
</Modal>
