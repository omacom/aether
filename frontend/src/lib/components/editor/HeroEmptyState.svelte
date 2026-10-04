<script lang="ts">
    import {showToast, setActiveTab} from '$lib/stores/ui.svelte';
    import {setWallpaperPath} from '$lib/stores/theme.svelte';
    import Kbd from '$lib/components/shared/Kbd.svelte';

    let isBrowsing = $state(false);

    const steps = [
        {
            lead: 'Pick a wallpaper.',
            body: 'Browse your local files or search Wallhaven for an image.',
        },
        {
            lead: 'Extract the palette.',
            body: 'Click Extract to generate 16 cohesive colors from the image. Change the extraction mode in the sidebar for a different feel.',
        },
        {lead: 'Apply the theme.', body: ''},
    ];

    async function handleBrowse() {
        if (isBrowsing) return;
        isBrowsing = true;
        try {
            const {OpenFileDialog} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const path = await OpenFileDialog();
            if (path) {
                setWallpaperPath(path);
                showToast(
                    'Wallpaper selected — click Extract to generate palette'
                );
            }
        } catch (e: any) {
            showToast('Couldn’t open that image file');
        } finally {
            isBrowsing = false;
        }
    }
</script>

<div class="flex min-h-full items-center justify-center px-6 py-10">
    <div class="flex w-full max-w-[520px] flex-col items-center text-center">
        <div
            class="bg-bg-surface border-border-focus flex w-full flex-col items-center border border-dashed px-8 pb-8 pt-10"
        >
            <div
                class="border-border bg-bg-secondary mb-[18px] flex h-14 w-14 items-center justify-center border"
            >
                <svg
                    class="text-fg-dimmed h-[26px] w-[26px]"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    aria-hidden="true"
                >
                    <path
                        stroke-linecap="square"
                        stroke-width="1.5"
                        d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                    />
                </svg>
            </div>
            <h2
                class="text-fg-primary mb-1.5 text-[17px] font-semibold tracking-[-0.005em]"
            >
                Pick a wallpaper to begin
            </h2>
            <p class="text-fg-secondary mb-[22px] text-[12.5px]">
                Drop an image here, browse your files, or find one on Wallhaven.
            </p>
            <div class="flex flex-wrap justify-center gap-2">
                <button
                    class="bg-accent text-accent-fg hover:bg-accent-hover h-8 px-4 text-[12px] font-semibold transition-colors disabled:opacity-50"
                    onclick={handleBrowse}
                    disabled={isBrowsing}
                >
                    {isBrowsing ? 'Opening…' : 'Browse files'}
                </button>
                <button
                    class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 border px-3.5 text-[12px] font-medium transition-colors"
                    onclick={() => setActiveTab('wallhaven')}
                >
                    Search Wallhaven
                </button>
                <button
                    class="border-border text-fg-secondary hover:bg-bg-hover hover:text-fg-primary h-8 border px-3.5 text-[12px] font-medium transition-colors"
                    onclick={() => setActiveTab('local')}
                >
                    Browse local
                </button>
            </div>
        </div>

        <div class="w-full pt-7 text-left">
            <h3
                class="text-fg-dimmed mb-3.5 text-[10px] font-semibold uppercase tracking-[0.14em]"
            >
                How it works
            </h3>
            <ol class="flex flex-col gap-3.5">
                {#each steps as step, i}
                    <li class="flex gap-3">
                        <span
                            class="border-border text-fg-dimmed flex h-[22px] w-[22px] shrink-0 items-center justify-center border font-mono text-[11px] font-medium"
                        >
                            {i + 1}
                        </span>
                        <span
                            class="text-fg-secondary text-[12.5px] leading-[1.55]"
                        >
                            <strong class="text-fg-primary font-semibold"
                                >{step.lead}</strong
                            >
                            {step.body}
                            {#if i === 2}
                                Click Apply theme or press <Kbd
                                    class="text-fg-primary text-[10.5px]"
                                    >Ctrl ↵</Kbd
                                > to push colors and the wallpaper to your apps.
                            {/if}
                        </span>
                    </li>
                {/each}
            </ol>
            <p
                class="border-border text-fg-dimmed mt-[22px] border-t pt-4 text-[11.5px]"
            >
                Save snapshots in
                <button
                    class="text-fg-secondary hover:text-fg-primary underline underline-offset-2 transition-colors"
                    onclick={() => setActiveTab('blueprints')}
                    >Blueprints</button
                >. Press <Kbd class="text-[10.5px]">?</Kbd> for keyboard shortcuts.
            </p>
        </div>
    </div>
</div>
