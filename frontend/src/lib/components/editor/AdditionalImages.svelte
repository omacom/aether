<script lang="ts">
    import {
        getAdditionalImages,
        addAdditionalImage,
        removeAdditionalImage,
        swapMainWithAdditional,
    } from '$lib/stores/theme.svelte';
    import {showToast} from '$lib/stores/ui.svelte';
    import CloseIcon from '$lib/components/shared/CloseIcon.svelte';

    let thumbnails = $state<Record<string, string>>({});

    $effect(() => {
        const images = getAdditionalImages();
        for (const img of images) {
            if (!thumbnails[img]) loadThumb(img);
        }
    });

    async function loadThumb(path: string) {
        try {
            const {GetThumbnail, ReadImageAsDataURL} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const thumbPath = await GetThumbnail(path);
            const dataUrl = await ReadImageAsDataURL(thumbPath);
            thumbnails = {...thumbnails, [path]: dataUrl};
        } catch {
            try {
                const {ReadImageAsDataURL} = await import(
                    '../../../../wailsjs/go/main/App'
                );
                const dataUrl = await ReadImageAsDataURL(path);
                thumbnails = {...thumbnails, [path]: dataUrl};
            } catch {}
        }
    }

    async function handleAdd() {
        try {
            const {OpenFileDialog} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const path = await OpenFileDialog();
            if (path) {
                showToast(
                    addAdditionalImage(path)
                        ? 'Image added'
                        : 'Skipped: the theme already has a wallpaper with that filename'
                );
            }
        } catch {}
    }

    function handleRemove(path: string) {
        removeAdditionalImage(path);
        const copy = {...thumbnails};
        delete copy[path];
        thumbnails = copy;
    }

    function handleSetAsMain(path: string) {
        swapMainWithAdditional(path);
        showToast('Set as main wallpaper');
    }
</script>

<section class="bg-bg-secondary border-border border">
    <div class="border-border flex items-center gap-2 border-b px-3.5 py-3">
        <h3 class="text-fg-primary text-[13px] font-semibold">
            Additional images
        </h3>
        <span class="text-fg-dimmed truncate text-[11.5px]"
            >Blended by Extract all</span
        >
    </div>

    <div class="grid grid-cols-2 gap-2 px-3.5 pb-3.5 pt-3">
        <!-- Unkeyed on purpose. A keyed each throws on a repeated path,
             and that error stopped every later effect (issue #130). -->
        {#each getAdditionalImages() as img}
            <div
                class="border-border group relative aspect-video overflow-hidden border bg-black"
            >
                {#if thumbnails[img]}
                    <img
                        src={thumbnails[img]}
                        alt=""
                        class="block h-full w-full object-cover"
                    />
                {:else}
                    <div class="flex h-full w-full items-center justify-center">
                        <span class="text-[11px] text-white/60">Loading…</span>
                    </div>
                {/if}
                <button
                    class="absolute left-1 top-1 flex h-[22px] items-center bg-black/55 px-1.5 text-[10.5px] font-medium text-white opacity-0 transition-opacity hover:bg-black/80 focus-visible:opacity-100 group-hover:opacity-100"
                    onclick={() => handleSetAsMain(img)}
                    aria-label="Set as main wallpaper"
                    title="Set as main wallpaper">Set main</button
                >
                <button
                    class="absolute right-1 top-1 flex h-[22px] w-[22px] items-center justify-center bg-black/55 text-white transition-colors hover:bg-black/80"
                    onclick={() => handleRemove(img)}
                    aria-label="Remove image"
                    title="Remove image"
                >
                    <CloseIcon size="h-[11px] w-[11px]" />
                </button>
            </div>
        {/each}
        <button
            class="border-border-focus text-fg-dimmed hover:bg-bg-hover hover:text-fg-primary flex aspect-video flex-col items-center justify-center gap-[5px] border border-dashed text-[11.5px] transition-colors"
            onclick={handleAdd}
            title="Add an image to blend into Extract all"
        >
            <svg
                class="h-4 w-4"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg
            >
            Add image
        </button>
    </div>
</section>
