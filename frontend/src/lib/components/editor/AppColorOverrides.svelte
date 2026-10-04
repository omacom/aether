<script lang="ts">
    import {
        getPalette,
        getExtendedColors,
        getLightMode,
        getAppOverrides,
        clearAppOverridesForApp,
        removeAppOverride,
        setAppOverride,
    } from '$lib/stores/theme.svelte';
    import {
        openOverrideColorPicker,
        getColorDrag,
        setColorDrag,
    } from '$lib/stores/ui.svelte';
    import {isLightColor, copyColor} from '$lib/utils/color';
    import ContextMenu from '$lib/components/shared/ContextMenu.svelte';
    import {appLabel} from '$lib/constants/apps';
    import {getNativeAppOverrides} from '$lib/actions/themeActions';
    import {getOmarchyAvailable} from '$lib/stores/omarchy.svelte';

    let selectedApp = $state('');
    let templateColors = $state<Record<string, string[]>>({});
    let computedVars = $state<Record<string, string>>({});

    // Short labels for the few ANSI colours that don't fit in a 60px swatch.
    // Anything not listed falls back to the raw role name (truncated by CSS).
    const SHORT_LABELS: Record<string, string> = {
        bright_black: 'br black',
        bright_red: 'br red',
        bright_green: 'br green',
        bright_yellow: 'br yellow',
        bright_blue: 'br blue',
        bright_magenta: 'br magenta',
        bright_cyan: 'br cyan',
        bright_white: 'br white',
        selection_foreground: 'sel fg',
        selection_background: 'sel bg',
        bright_fg: 'br fg',
    };

    let palette = $derived(getPalette());
    let extColors = $derived(getExtendedColors());
    let lightMode = $derived(getLightMode());
    let overrides = $derived(
        getOmarchyAvailable() ? getNativeAppOverrides() : getAppOverrides()
    );
    let appOverrides = $derived(
        selectedApp ? overrides[selectedApp] || {} : {}
    );

    let apps = $derived(Object.keys(templateColors).sort());

    let totalOverrideCount = $derived(
        Object.values(overrides).reduce(
            (sum, o) => sum + Object.keys(o).length,
            0
        )
    );
    let appOverrideCount = $derived(Object.keys(appOverrides).length);
    let appColors = $derived(templateColors[selectedApp] || []);

    let paletteKey = $derived(
        palette.join(',') + JSON.stringify(extColors) + lightMode
    );
    let lastPaletteKey = $state('');

    async function loadTemplateColors() {
        try {
            const {GetTemplateColors} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await GetTemplateColors();
            templateColors = result || {};
            if (!selectedApp && Object.keys(templateColors).length > 0) {
                selectedApp = Object.keys(templateColors).sort()[0];
            }
        } catch {
            // ignore
        }
    }

    async function loadComputedVars() {
        try {
            const {ComputeVariables} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const result = await ComputeVariables(
                palette,
                extColors,
                lightMode
            );
            computedVars = result || {};
            lastPaletteKey = paletteKey;
        } catch {
            // ignore
        }
    }

    // Request the template list once. An empty result must not start the
    // request again on every state change.
    let templatesRequested = false;
    $effect(() => {
        if (!templatesRequested) {
            templatesRequested = true;
            loadTemplateColors();
        }
        if (paletteKey !== lastPaletteKey) {
            loadComputedVars();
        }
    });

    function getDisplayColor(role: string): string {
        return appOverrides[role] || computedVars[role] || '#000000';
    }

    function getRoleLabel(role: string): string {
        return SHORT_LABELS[role] || role.replace(/_/g, ' ');
    }

    let dragOverRole = $state('');

    $effect(() => {
        if (!getColorDrag()) dragOverRole = '';
    });

    function onButtonMouseEnter(role: string) {
        if (getColorDrag()) dragOverRole = role;
    }

    function onButtonMouseUp(e: MouseEvent, role: string) {
        const drag = getColorDrag();
        if (!drag || e.button !== 0) return;
        setColorDrag(null);
        setAppOverride(selectedApp, role, drag.color, true);
        dragOverRole = '';
    }

    let menu = $state({open: false, x: 0, y: 0, role: ''});

    function openMenu(e: MouseEvent, role: string) {
        e.preventDefault();
        menu = {open: true, x: e.clientX, y: e.clientY, role};
    }

    let menuItems = $derived.by(() => {
        const role = menu.role;
        if (!role) return [];
        const isOverridden = !!appOverrides[role];
        return [
            {
                kind: 'item' as const,
                label: 'Edit override…',
                onSelect: () => openOverrideColorPicker(selectedApp, role),
                kbd: 'Click',
            },
            {
                kind: 'item' as const,
                label: 'Copy hex',
                onSelect: () => copyColor(getDisplayColor(role)),
            },
            ...(isOverridden
                ? [
                      {kind: 'divider' as const},
                      {
                          kind: 'item' as const,
                          label: 'Reset to computed',
                          onSelect: () => removeAppOverride(selectedApp, role),
                          danger: true,
                      },
                  ]
                : []),
        ];
    });
</script>

<section class="bg-bg-secondary border-border border">
    <div class="border-border flex items-center gap-2 border-b px-3.5 py-3">
        <h3 class="text-fg-primary text-[13px] font-semibold">
            Template overrides
        </h3>
        {#if totalOverrideCount > 0}
            <span
                class="bg-accent-muted text-accent px-[5px] py-px font-mono text-[10px] font-semibold"
                title="Active overrides across all apps"
                >{totalOverrideCount}</span
            >
        {/if}
        <span class="flex-1"></span>
        {#if appOverrideCount > 0}
            <button
                class="text-destructive text-[11.5px] hover:underline"
                onclick={() => clearAppOverridesForApp(selectedApp)}
                title="Reset all overrides for {appLabel(selectedApp)}"
            >
                Reset {appLabel(selectedApp)}
            </button>
        {/if}
    </div>

    <div class="flex flex-col gap-3 px-3.5 pb-3.5 pt-3">
        <!-- App chip picker -->
        <div class="flex flex-wrap gap-1">
            {#each apps as app}
                {@const count = overrides[app]
                    ? Object.keys(overrides[app]).length
                    : 0}
                {@const active = selectedApp === app}
                <button
                    type="button"
                    class="flex h-6 items-center gap-[5px] border px-[9px] text-[11.5px] transition-colors {active
                        ? 'bg-accent-muted border-accent text-accent'
                        : count > 0
                          ? 'border-accent/45 text-fg-secondary hover:border-border-focus'
                          : 'border-border text-fg-secondary hover:border-border-focus'}"
                    onclick={() => (selectedApp = app)}
                    aria-pressed={active}
                >
                    {appLabel(app)}
                    {#if count > 0}
                        <span
                            class="text-accent font-mono text-[9.5px] font-semibold"
                            >{count}</span
                        >
                    {/if}
                </button>
            {/each}
        </div>

        <!-- Color swatches for this app's template variables -->
        {#if appColors.length > 0}
            <div
                class="grid gap-[5px] [grid-template-columns:repeat(auto-fill,minmax(68px,1fr))]"
            >
                {#each appColors as role}
                    {@const display = getDisplayColor(role)}
                    {@const isOverridden = !!appOverrides[role]}
                    {@const light = isLightColor(display)}
                    <button
                        class="flex h-[38px] cursor-pointer items-end overflow-hidden px-1.5 pb-1 transition-transform duration-100 hover:-translate-y-px
                        {dragOverRole === role ? 'z-[1] scale-[1.06]' : ''}"
                        style:background-color={display}
                        style:color={light
                            ? 'rgba(10,10,16,0.86)'
                            : 'rgba(255,255,255,0.92)'}
                        style:box-shadow={isOverridden || dragOverRole === role
                            ? 'inset 0 0 0 2px var(--color-accent)'
                            : 'inset 0 0 0 1px rgba(128,128,128,0.2)'}
                        onclick={() =>
                            openOverrideColorPicker(selectedApp, role)}
                        oncontextmenu={e => openMenu(e, role)}
                        onmouseenter={() => onButtonMouseEnter(role)}
                        onmouseleave={() => (dragOverRole = '')}
                        onmouseup={e => onButtonMouseUp(e, role)}
                        title="{role}{isOverridden
                            ? ` · override ${appOverrides[role]}`
                            : ` · computed ${display}`}\nClick edit · Right-click for menu · Drag palette color to override"
                    >
                        <span
                            class="block w-full select-none truncate text-left text-[9.5px] leading-none opacity-85"
                            >{getRoleLabel(role)}</span
                        >
                    </button>
                {/each}
            </div>
        {:else if selectedApp}
            <p class="text-fg-dimmed text-[11.5px]">
                No color variables in this template.
            </p>
        {:else}
            <p class="text-fg-dimmed text-[11.5px]">Loading templates…</p>
        {/if}
    </div>
</section>

<ContextMenu
    open={menu.open}
    x={menu.x}
    y={menu.y}
    items={menuItems}
    onclose={() => (menu = {...menu, open: false})}
/>
