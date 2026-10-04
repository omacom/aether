<script lang="ts">
    import {onMount} from 'svelte';
    import BlueprintCard, {THEME_CARD_MIN_WIDTH} from './BlueprintCard.svelte';
    import SaveDialog from './SaveDialog.svelte';
    import {showToast} from '$lib/stores/ui.svelte';
    import {loadBlueprintIntoEditor} from '$lib/actions/blueprintActions';
    import {importThemeFile} from '$lib/actions/importActions';
    import type {Blueprint} from '$lib/types/theme';
    import EmptyState from '$lib/components/shared/EmptyState.svelte';
    import LoadingState from '$lib/components/shared/LoadingState.svelte';
    import ViewHeader from '$lib/components/shared/ViewHeader.svelte';
    import CardSizeToggle from '$lib/components/shared/CardSizeToggle.svelte';
    import {getCardSize} from '$lib/stores/cardsize.svelte';

    let blueprints = $state<Blueprint[]>([]);
    let isLoading = $state(true);
    let showSaveDialog = $state(false);

    onMount(() => {
        loadBlueprints();
    });

    async function loadBlueprints() {
        isLoading = true;
        try {
            const {ListBlueprints} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await ListBlueprints();
            blueprints = (
                Array.isArray(result) ? (result as unknown as Blueprint[]) : []
            ).sort((a, b) => (b.timestamp || 0) - (a.timestamp || 0));
        } catch {
            blueprints = [];
        } finally {
            isLoading = false;
        }
    }

    async function handleDelete(name: string) {
        try {
            const {DeleteBlueprint} = await import(
                '../../../../wailsjs/go/main/App'
            );
            await DeleteBlueprint(name);
            blueprints = blueprints.filter(b => b.name !== name);
            showToast(`Deleted: ${name}`);
        } catch {
            showToast('Couldn’t delete that theme');
        }
    }

    function handleLoad(bp: Blueprint) {
        try {
            loadBlueprintIntoEditor(bp);
        } catch {
            showToast('Couldn’t load that blueprint');
        }
    }
</script>

<div class="flex h-full flex-col">
    <ViewHeader>
        <h2 class="text-fg-primary text-[13.5px] font-semibold">My themes</h2>
        {#if !isLoading}
            <span class="text-fg-dimmed text-[12px]"
                >{blueprints.length} saved</span
            >
        {/if}
        <div class="ml-auto flex items-center gap-2">
            <CardSizeToggle />
            <button
                type="button"
                class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-7 border px-3 text-[12px] font-medium transition-colors"
                onclick={() => importThemeFile('blueprint')}
                title="Load a blueprint .json file into the editor"
                >Import</button
            >
            <button
                type="button"
                class="bg-accent text-accent-fg hover:bg-accent-hover h-7 px-3 text-[12px] font-semibold transition-colors"
                onclick={() => (showSaveDialog = true)}>Save current</button
            >
        </div>
    </ViewHeader>

    <div class="flex-1 overflow-y-auto p-4">
        {#if isLoading}
            <LoadingState message="Loading themes…" />
        {:else if blueprints.length === 0}
            <EmptyState
                title="No themes saved yet"
                body="Save the current palette, adjustments, overrides, and wallpaper as a Blueprint to revisit later."
                actionLabel="Save current theme"
                onaction={() => (showSaveDialog = true)}
            >
                {#snippet icon()}
                    <svg
                        class="h-6 w-6"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                    >
                        <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
                        <polyline points="2 17 12 22 22 17"></polyline>
                        <polyline points="2 12 12 17 22 12"></polyline>
                    </svg>
                {/snippet}
            </EmptyState>
        {:else}
            <div
                class="grid gap-3"
                style:grid-template-columns="repeat(auto-fill, minmax({THEME_CARD_MIN_WIDTH[
                    getCardSize()
                ]}px, 1fr))"
            >
                {#each blueprints as bp, i (bp.name + '_' + i)}
                    <BlueprintCard
                        blueprint={bp}
                        onload={() => handleLoad(bp)}
                        ondelete={() => handleDelete(bp.name)}
                    />
                {/each}
            </div>
        {/if}
    </div>

    <SaveDialog
        open={showSaveDialog}
        onclose={() => (showSaveDialog = false)}
        onsave={() => {
            showSaveDialog = false;
            loadBlueprints();
        }}
    />
</div>
