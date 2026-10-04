<script lang="ts">
    let {
        src,
        alt = '',
        open = false,
        onclose,
        onprev,
        onnext,
        hasPrev = false,
        hasNext = false,
    }: {
        src: string;
        alt?: string;
        open: boolean;
        onclose: () => void;
        onprev?: () => void;
        onnext?: () => void;
        hasPrev?: boolean;
        hasNext?: boolean;
    } = $props();

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === 'Escape') onclose();
        if (e.key === 'ArrowLeft' && hasPrev) onprev?.();
        if (e.key === 'ArrowRight' && hasNext) onnext?.();
    }

    // The backdrop div is never focused, so its onkeydown never fires. Listen
    // on window while open so Escape and arrow paging actually work.
    $effect(() => {
        if (!open) return;
        window.addEventListener('keydown', handleKeydown);
        return () => window.removeEventListener('keydown', handleKeydown);
    });
</script>

{#if open}
    <!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(5,5,9,0.86)] p-12"
        onclick={e => {
            if (e.target === e.currentTarget) onclose();
        }}
        role="presentation"
    >
        <!-- Close button -->
        <button
            class="border-white/14 absolute right-4 top-4 z-10 flex h-8 w-8 items-center justify-center border bg-[rgba(12,12,16,0.5)] text-white/80 backdrop-blur-[8px] transition-colors hover:bg-[rgba(12,12,16,0.78)] hover:text-white"
            onclick={onclose}
            aria-label="Close preview"
        >
            <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
        </button>

        <!-- Previous arrow -->
        {#if hasPrev}
            <button
                class="border-white/14 absolute left-4 top-1/2 z-10 flex h-10 w-10 -translate-y-1/2 items-center justify-center border bg-[rgba(12,12,16,0.5)] text-white/80 backdrop-blur-[8px] transition-colors hover:bg-[rgba(12,12,16,0.78)] hover:text-white"
                onclick={onprev}
                aria-label="Previous image"
            >
                <svg
                    class="h-5 w-5"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <polyline points="15 18 9 12 15 6"></polyline>
                </svg>
            </button>
        {/if}

        <!-- Next arrow -->
        {#if hasNext}
            <button
                class="border-white/14 absolute right-4 top-1/2 z-10 flex h-10 w-10 -translate-y-1/2 items-center justify-center border bg-[rgba(12,12,16,0.5)] text-white/80 backdrop-blur-[8px] transition-colors hover:bg-[rgba(12,12,16,0.78)] hover:text-white"
                onclick={onnext}
                aria-label="Next image"
            >
                <svg
                    class="h-5 w-5"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <polyline points="9 6 15 12 9 18"></polyline>
                </svg>
            </button>
        {/if}

        <img
            {src}
            {alt}
            class="max-h-full max-w-full object-contain shadow-[0_24px_64px_rgba(0,0,0,0.55)]"
        />
    </div>
{/if}
