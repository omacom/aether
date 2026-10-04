<script lang="ts">
    import Modal from './Modal.svelte';
    import DialogFooter from './DialogFooter.svelte';

    let {
        open,
        title,
        body,
        confirmLabel = 'Confirm',
        cancelLabel = 'Cancel',
        danger = false,
        onconfirm,
        oncancel,
    }: {
        open: boolean;
        title: string;
        body?: string;
        confirmLabel?: string;
        cancelLabel?: string;
        danger?: boolean;
        onconfirm: () => void;
        oncancel: () => void;
    } = $props();

    let confirmEl = $state<HTMLButtonElement | null>(null);

    $effect(() => {
        if (open) confirmEl?.focus();
    });
</script>

<Modal
    {open}
    onclose={oncancel}
    onenter={onconfirm}
    bare
    panelClass="w-[400px]"
    label={title}
>
    <div class="px-5 pb-5 pt-[18px]">
        <h3 class="text-fg-primary mb-1 text-[14px] font-semibold">
            {title}
        </h3>
        {#if body}
            <p class="text-fg-dimmed text-[12px] leading-relaxed">
                {body}
            </p>
        {/if}
    </div>
    <DialogFooter>
        <button
            type="button"
            class="text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 px-3.5 text-[12px] transition-colors"
            onclick={oncancel}>{cancelLabel}</button
        >
        <button
            type="button"
            bind:this={confirmEl}
            class="h-8 px-4 text-[12px] font-semibold transition-colors {danger
                ? 'bg-destructive hover:bg-destructive/85 text-destructive-fg'
                : 'bg-accent hover:bg-accent-hover text-accent-fg'}"
            onclick={onconfirm}>{confirmLabel}</button
        >
    </DialogFooter>
</Modal>
