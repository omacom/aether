<script lang="ts">
    import {setWallpaperPath} from '$lib/stores/theme.svelte';
    import {showToast, setActiveTab} from '$lib/stores/ui.svelte';

    async function handleBrowse() {
        try {
            const {OpenFileDialog} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const path = await OpenFileDialog();
            if (path) {
                setWallpaperPath(path);
                showToast('Wallpaper selected');
            }
        } catch {}
    }
</script>

<div
    class="bg-bg-surface border-border-focus flex items-center gap-2 border border-dashed px-4 py-3"
>
    <span class="text-fg-secondary flex-1 text-[12.5px]"
        >No wallpaper selected</span
    >
    <button
        class="bg-accent text-accent-fg hover:bg-accent-hover h-8 px-4 text-[12px] font-semibold transition-colors"
        onclick={handleBrowse}>Browse files</button
    >
    <button
        class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 border px-3.5 text-[12px] font-medium transition-colors"
        onclick={() => setActiveTab('wallhaven')}>Search Wallhaven</button
    >
    <button
        class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 border px-3.5 text-[12px] font-medium transition-colors"
        onclick={() => setActiveTab('local')}>Browse local</button
    >
</div>
