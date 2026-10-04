<script lang="ts">
    import {getSettings, updateSettings} from '$lib/stores/settings.svelte';
    import {showToast} from '$lib/stores/ui.svelte';

    let choosingFolder = $state(false);
    let wallpaperFolder = $derived(
        getSettings().wallpaperFolder || '~/Wallpapers'
    );

    async function chooseWallpaperFolder() {
        choosingFolder = true;
        try {
            const {ChooseWallpaperFolder} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const folder = await ChooseWallpaperFolder();
            if (!folder) return;

            updateSettings({wallpaperFolder: folder});
            showToast('Wallpaper folder updated');
        } catch (err) {
            console.error('ChooseWallpaperFolder failed', err);
            showToast('Failed to set wallpaper folder');
        } finally {
            choosingFolder = false;
        }
    }
</script>

<div class="h-full overflow-y-auto">
    <div class="mx-auto max-w-[760px] px-8 py-11">
        <header class="mb-9">
            <p
                class="text-accent mb-2 text-[10px] font-semibold uppercase tracking-[0.18em]"
            >
                Preferences
            </p>
            <h1
                class="text-fg-primary text-[26px] font-semibold tracking-[-0.01em]"
            >
                Settings
            </h1>
            <p class="text-fg-secondary mt-2 text-[13px] leading-relaxed">
                Configure where Aether finds local wallpapers.
            </p>
        </header>

        <section aria-labelledby="wallpaper-library-heading">
            <h2
                id="wallpaper-library-heading"
                class="text-fg-dimmed mb-2.5 text-[10px] font-semibold uppercase tracking-[0.14em]"
            >
                Wallpaper library
            </h2>

            <div
                class="border-border bg-bg-secondary grid grid-cols-[auto_minmax(0,1fr)] items-start gap-4 border p-[18px] sm:grid-cols-[auto_minmax(0,1fr)_auto]"
            >
                <div
                    class="bg-accent-muted text-accent flex h-9 w-9 shrink-0 items-center justify-center"
                >
                    <svg
                        class="h-4 w-4"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                    >
                        <path
                            d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                        ></path>
                    </svg>
                </div>

                <div class="min-w-0">
                    <div class="text-fg-primary text-[13px] font-semibold">
                        Wallpaper folder
                    </div>
                    <p class="text-fg-dimmed mb-3 mt-[3px] text-[12px]">
                        The Local page scans this folder and all of its
                        subfolders for images.
                    </p>
                    <div
                        class="bg-bg-primary border-border text-fg-secondary truncate border px-[11px] py-[9px] font-mono text-[12px] font-medium"
                        title={wallpaperFolder}
                    >
                        {wallpaperFolder}
                    </div>
                </div>

                <button
                    type="button"
                    class="bg-accent text-accent-fg hover:bg-accent-hover col-span-2 h-8 w-full shrink-0 px-3.5 text-[12px] font-semibold transition-colors disabled:cursor-wait disabled:opacity-60 sm:col-span-1 sm:w-auto"
                    onclick={chooseWallpaperFolder}
                    disabled={choosingFolder}
                >
                    {choosingFolder ? 'Choosing…' : 'Choose folder…'}
                </button>
            </div>
        </section>
    </div>
</div>
