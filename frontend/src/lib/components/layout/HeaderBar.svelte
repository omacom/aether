<script lang="ts">
    import {onMount} from 'svelte';
    import {
        getActiveTab,
        setActiveTab,
        toggleKeymap,
        openCommandPalette,
        type Tab,
    } from '$lib/stores/ui.svelte';
    import SearchIcon from '$lib/components/shared/SearchIcon.svelte';
    import Kbd from '$lib/components/shared/Kbd.svelte';
    import ReleaseIndicator from '$lib/components/layout/ReleaseIndicator.svelte';
    import aetherLogo from '../../../assets/aether-logo.png';
    import {
        getOmarchyAvailable,
        initOmarchyCapabilities,
    } from '$lib/stores/omarchy.svelte';

    let activeTab = $derived(getActiveTab());
    let isMac = $state(false);
    let omarchyAvailable = $derived(getOmarchyAvailable());

    initOmarchyCapabilities();

    onMount(async () => {
        try {
            const {IsMacOS} = await import('../../../../wailsjs/go/main/App');
            isMac = await IsMacOS();
        } catch {}
    });

    const tabs: {id: Tab; label: string; icon: string}[] = [
        {
            id: 'editor',
            label: 'Editor',
            // Sliders (adjustments)
            icon: '<line x1="4" y1="21" x2="4" y2="14"/><line x1="4" y1="10" x2="4" y2="3"/><line x1="12" y1="21" x2="12" y2="12"/><line x1="12" y1="8" x2="12" y2="3"/><line x1="20" y1="21" x2="20" y2="16"/><line x1="20" y1="12" x2="20" y2="3"/><line x1="1" y1="14" x2="7" y2="14"/><line x1="9" y1="8" x2="15" y2="8"/><line x1="17" y1="16" x2="23" y2="16"/>',
        },
        {
            id: 'wallhaven',
            label: 'Wallhaven',
            // Globe
            icon: '<circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>',
        },
        {
            id: 'github',
            label: 'GitHub',
            icon: '<circle cx="6" cy="5" r="3"/><circle cx="6" cy="19" r="3"/><circle cx="18" cy="5" r="3"/><path d="M6 8v8M18 8a11 11 0 0 1-9 11"/>',
        },
        {
            id: 'local',
            label: 'Local',
            // Folder
            icon: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>',
        },
        {
            id: 'favorites',
            label: 'Favorites',
            // Heart
            icon: '<path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/>',
        },
        {
            id: 'blueprints',
            label: 'Blueprints',
            // Layers
            icon: '<polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/>',
        },
        {
            id: 'system',
            label: 'Omarchy',
            // Paintbrush / Brush
            icon: '<path d="M18.37 2.63a2.12 2.12 0 0 1 3 3L14 13l-4 1 1-4z"/><path d="M9 14.5A3.5 3.5 0 0 0 5.5 18c-1.2 0-2.5.7-2.5 2 2 0 4.5-1 5.5-3.5"/>',
        },
    ];
    let visibleTabs = $derived(
        tabs.filter(tab => tab.id !== 'system' || omarchyAvailable)
    );
</script>

{#snippet iconButton(
    tab: Tab | null,
    label: string,
    title: string,
    onclick: () => void,
    icon: import('svelte').Snippet
)}
    <button
        type="button"
        class="flex h-[30px] w-[30px] items-center justify-center transition-colors
            {tab && activeTab === tab
            ? 'text-accent bg-accent-muted'
            : 'text-fg-dimmed hover:text-fg-primary hover:bg-bg-hover'}"
        {onclick}
        aria-label={label}
        aria-current={tab && activeTab === tab ? 'page' : undefined}
        {title}
    >
        {@render icon()}
    </button>
{/snippet}

<header
    class="bg-bg-secondary border-border flex shrink-0 items-stretch border-b pr-2"
    class:h-11={!isMac}
    class:h-[50px]={isMac}
    class:pl-3.5={!isMac}
    class:pl-[84px]={isMac}
    class:pt-1.5={isMac}
    style="--wails-draggable:drag"
>
    <button
        type="button"
        class="text-fg-primary hover:text-accent flex items-center gap-[9px] pr-3.5 text-[11px] font-semibold tracking-[0.18em] transition-colors duration-100"
        style="--wails-draggable:no-drag"
        onclick={() => setActiveTab('editor')}
        title="Editor"
    >
        <img src={aetherLogo} alt="" class="h-[18px] w-[18px] object-contain" />
        <span>AETHER</span>
    </button>
    <div class="bg-border my-3 mr-1 w-px shrink-0" aria-hidden="true"></div>
    <nav
        class="flex min-w-0 items-stretch overflow-x-auto"
        aria-label="Main navigation"
        style="--wails-draggable:no-drag"
    >
        {#each visibleTabs as tab}
            <button
                type="button"
                class="flex shrink-0 items-center gap-[7px] px-3 text-[12px] font-medium transition-colors duration-100
                    {activeTab === tab.id
                    ? 'text-fg-primary shadow-[inset_0_-2px_0_var(--color-accent)]'
                    : 'text-fg-dimmed hover:text-fg-primary'}"
                onclick={() => setActiveTab(tab.id)}
                aria-current={activeTab === tab.id ? 'page' : undefined}
            >
                <svg
                    class="h-3.5 w-3.5 shrink-0"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    aria-hidden="true"
                >
                    {@html tab.icon}
                </svg>
                {tab.label}
            </button>
        {/each}
    </nav>

    <div class="min-w-4 flex-1"></div>

    <div class="flex items-center gap-1" style="--wails-draggable:no-drag">
        <ReleaseIndicator {isMac} />
        <button
            type="button"
            class="bg-bg-surface border-border text-fg-dimmed hover:border-border-focus mr-1.5 flex h-7 min-w-0 items-center gap-2 border pl-2.5 pr-1.5 text-[12px] transition-colors xl:w-[232px]"
            onclick={openCommandPalette}
            aria-label="Open command palette"
            title="Command palette (Ctrl+P)"
        >
            <SearchIcon size="h-[13px] w-[13px]" />
            <span class="hidden flex-1 truncate text-left xl:inline"
                >Search commands…</span
            >
            <Kbd class="text-fg-dimmed">Ctrl P</Kbd>
        </button>
        {#snippet settingsIcon()}
            <svg
                class="h-[15px] w-[15px]"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
            >
                <path d="M12 2l8.66 5v10L12 22l-8.66-5V7z"></path>
                <circle cx="12" cy="12" r="3"></circle>
            </svg>
        {/snippet}
        {#snippet aboutIcon()}
            <svg
                class="h-[15px] w-[15px]"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
            >
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="16" x2="12" y2="12"></line>
                <line x1="12" y1="8" x2="12.01" y2="8"></line>
            </svg>
        {/snippet}
        {#snippet keymapIcon()}
            <span class="font-mono text-[12px] font-semibold leading-none"
                >?</span
            >
        {/snippet}
        {@render iconButton(
            'settings',
            'Settings',
            'Settings',
            () => setActiveTab('settings'),
            settingsIcon
        )}
        {@render iconButton(
            'about',
            'About',
            'About',
            () => setActiveTab('about'),
            aboutIcon
        )}
        {@render iconButton(
            null,
            'Show keyboard shortcuts',
            'Keyboard shortcuts (Ctrl+K or ?)',
            toggleKeymap,
            keymapIcon
        )}
    </div>
</header>
