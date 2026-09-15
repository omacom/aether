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
      Sits directly above the ActionBar footer (h-10). App chrome, not an image
      overlay, so it uses theme tokens and stays legible in light mode.
    -->
    <div
        class="bg-bg-secondary border-border fixed bottom-10 left-0 right-0 z-[90] border-t"
    >
        <div class="flex items-center gap-3 px-3 py-1.5">
            <span class="text-fg-secondary shrink-0 text-[11px]">
                {label}
                {#if state.total > 0}{state.index}/{state.total}{/if}
            </span>
            {#if state.name}
                <span class="text-fg-dimmed min-w-0 flex-1 truncate text-[11px]"
                    >{state.name}</span
                >
            {:else}
                <span class="min-w-0 flex-1"></span>
            {/if}
            <button
                class="text-destructive hover:bg-bg-hover shrink-0 px-2 py-1 text-[11px] transition-colors duration-100"
                onclick={cancelExport}>Cancel</button
            >
        </div>
        <div
            class="bg-bg-surface h-1 w-full"
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
        class="bg-bg-secondary border-border fixed bottom-10 left-0 right-0 z-[90] border-t px-3 py-2 text-xs"
    >
        <div class="flex items-center gap-3">
            <p class="text-fg-primary min-w-0 flex-1" role="status">
                Exported {result.exported} of {result.total} favorites
            </p>
            <button
                class="text-accent shrink-0 px-2 py-1"
                onclick={openExportFolder}>Open folder</button
            >
            <button
                class="text-fg-secondary shrink-0 px-2 py-1"
                onclick={dismissExportResult}>Dismiss</button
            >
        </div>
        {#if result.skipped?.length}
            <details class="text-fg-secondary mt-1">
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
