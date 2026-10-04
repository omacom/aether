<script lang="ts">
    import {onMount} from 'svelte';
    import {
        isAppIncluded,
        toggleAppInclusion,
    } from '$lib/stores/settings.svelte';
    import {
        SPECIAL_APP_KEYS,
        ALWAYS_INCLUDED_APPS,
        appLabel,
    } from '$lib/constants/apps';
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import Switch from '$lib/components/shared/Switch.svelte';

    let templatesOpen = $state(false);
    let appsOpen = $state(false);

    const specialToggles = [
        {key: 'neovim', label: 'Neovim'},
        {key: 'zed', label: 'Zed'},
        {key: 'vscode', label: 'VS Code'},
    ] as const;

    let appList = $state<string[]>([]);

    onMount(async () => {
        try {
            const {GetTemplateColors} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await GetTemplateColors();
            appList = Object.keys(result || {})
                .filter(
                    k =>
                        !SPECIAL_APP_KEYS.has(k) &&
                        !ALWAYS_INCLUDED_APPS.has(k) &&
                        k !== 'icons'
                )
                .sort();
        } catch {
            appList = [];
        }
    });
</script>

{#snippet toggleRow(label: string, on: boolean, onflip: () => void)}
    <div class="flex h-6 items-center justify-between gap-3">
        <span class="text-fg-secondary text-[12px]">{label}</span>
        <Switch
            checked={on}
            onchange={onflip}
            label="Toggle {label}"
            size="sm"
        />
    </div>
{/snippet}

<ExpandableSection
    title="App templates"
    contentClass="px-4 pb-3.5"
    bind:expanded={templatesOpen}
>
    <p class="text-fg-dimmed mb-2 text-[11.5px] leading-normal">
        Pick which apps receive generated config files when you apply.
    </p>
    <div class="flex flex-col gap-1">
        {#each specialToggles as toggle}
            {@render toggleRow(toggle.label, isAppIncluded(toggle.key), () =>
                toggleAppInclusion(toggle.key)
            )}
        {/each}

        {#if appList.length > 0}
            <div class="-mx-1.5 mt-1">
                <ExpandableSection
                    variant="group"
                    title="Apps"
                    suffix={String(appList.length)}
                    bind:expanded={appsOpen}
                >
                    <div class="flex flex-col gap-1 px-1.5 pt-1">
                        {#each appList as app}
                            {@render toggleRow(
                                appLabel(app),
                                isAppIncluded(app),
                                () => toggleAppInclusion(app)
                            )}
                        {/each}
                    </div>
                </ExpandableSection>
            </div>
        {/if}
    </div>
</ExpandableSection>
