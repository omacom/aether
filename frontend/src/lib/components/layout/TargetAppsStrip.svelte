<script lang="ts">
    import {onMount} from 'svelte';
    import {
        isAppIncluded,
        toggleAppInclusion,
    } from '$lib/stores/settings.svelte';
    import {setTargetsVisible} from '$lib/stores/ui.svelte';
    import CloseIcon from '$lib/components/shared/CloseIcon.svelte';
    import {
        SPECIAL_APP_FLAGS,
        SPECIAL_APP_ORDER,
        ALWAYS_INCLUDED_APPS,
        appLabel,
    } from '$lib/constants/apps';

    let appList = $state<string[]>([]);

    onMount(async () => {
        // Specials always appear: GetTemplateColors filters out apps whose
        // templates have no color variables (vscode.json is just a pointer).
        // Inject all specials unconditionally so the target list is stable.
        try {
            const {GetTemplateColors} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await GetTemplateColors();
            const all = Object.keys(result || {});
            const rest = all
                .filter(
                    k =>
                        !(k in SPECIAL_APP_FLAGS) &&
                        !ALWAYS_INCLUDED_APPS.has(k)
                )
                .sort();
            appList = [...SPECIAL_APP_ORDER, ...rest];
        } catch {
            appList = [...SPECIAL_APP_ORDER];
        }
    });
</script>

<div
    class="bg-bg-secondary border-border flex shrink-0 items-center gap-3 border-t px-4 py-2"
>
    <span
        class="text-fg-dimmed shrink-0 text-[10px] font-semibold uppercase tracking-[0.14em]"
        title="Aether app templates included as overrides"
    >
        Targets
    </span>
    <div class="flex min-w-0 flex-1 flex-wrap items-center gap-1">
        {#each appList as app}
            {@const on = isAppIncluded(app)}
            <button
                type="button"
                onclick={() => toggleAppInclusion(app)}
                class="h-6 border px-[9px] text-[11.5px] transition-colors duration-100 {on
                    ? 'bg-accent-muted border-accent text-accent'
                    : 'border-border text-fg-secondary hover:text-fg-primary hover:border-border-focus'}"
                title={on
                    ? `${appLabel(app)} will use Aether's template override`
                    : `${appLabel(app)} will use the default generated template unless it has color overrides`}
                aria-pressed={on}
            >
                {appLabel(app)}
            </button>
        {/each}
    </div>
    <button
        type="button"
        class="text-fg-dimmed hover:text-fg-primary hover:bg-bg-hover flex h-6 w-6 shrink-0 items-center justify-center transition-colors"
        onclick={() => setTargetsVisible(false)}
        title="Hide targets"
        aria-label="Hide targets"
    >
        <CloseIcon size="h-3.5 w-3.5" />
    </button>
</div>
