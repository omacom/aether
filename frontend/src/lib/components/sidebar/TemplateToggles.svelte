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
    import {
        getTargets,
        getTargetStatus,
        isTargetEnabled,
        loadTargets,
        setTargetEnabled,
    } from '$lib/stores/targets.svelte';
    import ExpandableSection from '$lib/components/shared/ExpandableSection.svelte';
    import Switch from '$lib/components/shared/Switch.svelte';

    let templatesOpen = $state(false);
    let appsOpen = $state(false);
    let targetsOpen = $state(true);

    const specialToggles = [{key: 'zed', label: 'Zed'}] as const;

    // Apps that are desktop targets get one row in the target list instead of
    // a second legacy toggle.
    let targetIds = $derived(new Set(getTargets().map(t => t.id)));

    function statusText(state: string): string {
        return state === 'not-found' ? 'not found' : state;
    }

    function statusClass(state: string): string {
        if (state === 'written' || state === 'reloaded') return 'text-success';
        if (state === 'failed') return 'text-destructive';
        return 'text-fg-dimmed';
    }

    let templateKeys = $state<string[]>([]);
    let appList = $derived(
        templateKeys
            .filter(
                k =>
                    !SPECIAL_APP_KEYS.has(k) &&
                    !ALWAYS_INCLUDED_APPS.has(k) &&
                    !targetIds.has(k) &&
                    k !== 'icons'
            )
            .sort()
    );

    onMount(async () => {
        loadTargets();
        try {
            const {GetTemplateColors} = await import(
                '../../../../wailsjs/go/main/App'
            );
            templateKeys = Object.keys((await GetTemplateColors()) || {});
        } catch {
            templateKeys = [];
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

{#snippet targetRow(t: {
    id: string;
    label: string;
    detected: boolean;
    path: string;
})}
    {@const st = t.detected ? getTargetStatus(t.id) : undefined}
    <div
        data-target={t.id}
        class="flex h-6 items-center justify-between gap-3 {t.detected
            ? ''
            : 'opacity-60'}"
        title={[t.path, st?.error].filter(Boolean).join('\n')}
    >
        <span class="text-fg-secondary text-[12px]">{t.label}</span>
        <span class="flex items-center gap-2">
            <span
                class="text-[11px] {t.detected
                    ? st
                        ? statusClass(st.state)
                        : 'text-fg-dimmed'
                    : 'text-fg-dimmed'}"
                >{t.detected
                    ? st
                        ? statusText(st.state)
                        : ''
                    : 'not found'}</span
            >
            <Switch
                checked={t.detected && isTargetEnabled(t.id)}
                disabled={!t.detected}
                onchange={on => setTargetEnabled(t.id, on)}
                label="Toggle {t.label}"
                size="sm"
            />
        </span>
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

        {#if getTargets().length > 0}
            <div class="-mx-1.5 mt-1">
                <ExpandableSection
                    variant="group"
                    title="Desktop targets"
                    suffix={String(getTargets().length)}
                    bind:expanded={targetsOpen}
                >
                    <div class="flex flex-col gap-1 px-1.5 pt-1">
                        {#each getTargets() as t (t.id)}
                            {@render targetRow(t)}
                        {/each}
                    </div>
                </ExpandableSection>
            </div>
        {/if}

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
