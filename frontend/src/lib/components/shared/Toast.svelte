<script lang="ts">
    import {
        getToastMessage,
        getToastVisible,
        getToastAction,
        getToastQueueDepth,
        dismissCurrentToast,
    } from '$lib/stores/ui.svelte';
    import CloseIcon from '$lib/components/shared/CloseIcon.svelte';

    let visible = $derived(getToastVisible());
    let message = $derived(getToastMessage());
    let action = $derived(getToastAction());
    let queueDepth = $derived(getToastQueueDepth());

    function runAction() {
        action?.run();
        dismissCurrentToast();
    }
</script>

{#if visible}
    <div
        class="bg-fg-primary text-bg-primary shadow-(--shadow-panel) fixed bottom-16 left-1/2 z-[100] flex -translate-x-1/2 items-center gap-3 py-[9px] pl-3.5 pr-2.5 text-[12px] font-medium"
        style="animation: toast-in 150ms ease-out"
        role="status"
        aria-live="polite"
    >
        <span>{message}</span>
        {#if queueDepth > 1}
            <span class="font-mono text-[10px] tabular-nums opacity-60"
                >+{queueDepth - 1}</span
            >
        {/if}
        {#if action}
            <button
                type="button"
                class="-my-1 px-1.5 py-1 text-[12px] font-semibold underline underline-offset-2 opacity-90 transition-opacity hover:opacity-100"
                onclick={runAction}
            >
                {action.label}
            </button>
        {/if}
        <button
            type="button"
            class="-my-1 flex h-6 w-6 items-center justify-center opacity-55 transition-opacity hover:opacity-100"
            onclick={dismissCurrentToast}
            aria-label="Dismiss"
            title="Dismiss"
        >
            <CloseIcon size="h-3.5 w-3.5" />
        </button>
    </div>
{/if}

<style>
    @keyframes toast-in {
        from {
            opacity: 0;
            transform: translateX(-50%) translateY(8px);
        }
        to {
            opacity: 1;
            transform: translateX(-50%) translateY(0);
        }
    }
</style>
