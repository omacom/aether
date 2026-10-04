<script lang="ts">
    import {
        getExportState,
        getExportResult,
        dismissExportResult,
        openExportFolder,
        cancelExport,
    } from '$lib/stores/favoritesExport.svelte';

    let state = $derived(getExportState());
    let result = $derived(getExportResult());
    let percent = $derived(
        state.total > 0
            ? Math.min(100, Math.round((state.index / state.total) * 100))
            : 0
    );
    let label = $derived(
        state.phase === 'archive' ? 'Archiving' : 'Downloading'
    );
</script>

{#if state.active}
    <!--
      Sits directly above the ActionBar footer (h-12). App chrome, not an image
      overlay, so it uses theme tokens and stays legible in light mode.
    -->
    <div
        class="bg-bg-secondary border-border fixed bottom-12 left-0 right-0 z-[90] border-t"
    >
        <div class="flex h-10 items-center gap-3 px-4">
            <span class="text-fg-secondary shrink-0 text-[12px]">
                {label}
                {#if state.total > 0}<span class="font-mono tabular-nums"
                        >{state.index}/{state.total}</span
                    >{/if}
            </span>
            {#if state.name}
                <span
                    class="text-fg-dimmed min-w-0 flex-1 truncate font-mono text-[11px]"
                    >{state.name}</span
                >
            {:else}
                <span class="min-w-0 flex-1"></span>
            {/if}
            <button
                type="button"
                class="text-destructive hover:bg-bg-hover h-7 shrink-0 px-2.5 text-[12px] transition-colors duration-100"
                onclick={cancelExport}>Cancel</button
            >
        </div>
        <div
            class="bg-bg-elevated h-0.5 w-full"
            role="progressbar"
            aria-label="Favorites export progress"
            aria-valuenow={percent}
            aria-valuemin={0}
            aria-valuemax={100}
        >
            <div
                class="bg-accent h-full transition-[width] duration-150"
                style:width="{percent}%"
            ></div>
        </div>
    </div>
{:else if result}
    <div
        class="bg-bg-secondary border-border fixed bottom-12 left-0 right-0 z-[90] border-t px-4 py-2 text-[12px]"
    >
        <div class="flex min-h-7 items-center gap-2">
            <p class="text-fg-primary min-w-0 flex-1" role="status">
                Exported {result.exported} of {result.total} favorites
            </p>
            <button
                type="button"
                class="text-accent hover:text-accent-hover hover:bg-bg-hover h-7 shrink-0 px-2.5 transition-colors"
                onclick={openExportFolder}>Open folder</button
            >
            <button
                type="button"
                class="text-fg-secondary hover:text-fg-primary hover:bg-bg-hover h-7 shrink-0 px-2.5 transition-colors"
                onclick={dismissExportResult}>Dismiss</button
            >
        </div>
        {#if result.skipped?.length}
            <details class="text-fg-secondary mt-1 text-[11.5px]">
                <summary class="cursor-pointer py-1"
                    >{result.skipped.length} skipped files</summary
                >
                <ul class="mt-1 max-h-32 space-y-1 overflow-y-auto">
                    {#each result.skipped as item}
                        <li class="break-words">
                            <span class="font-mono">{item.path}</span>: {item.reason}
                        </li>
                    {/each}
                </ul>
            </details>
        {/if}
    </div>
{/if}
