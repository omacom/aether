<script lang="ts">
    import {tick} from 'svelte';
    import type {Command} from '$lib/commands/commands.svelte';
    import type {Blueprint} from '$lib/types/theme';
    import {loadBlueprintIntoEditor} from '$lib/actions/blueprintActions';
    import {showToast} from '$lib/stores/ui.svelte';

    let {
        open,
        commands,
        onclose,
    }: {
        open: boolean;
        commands: Command[];
        onclose: () => void;
    } = $props();

    let query = $state('');
    let activeIdx = $state(0);
    let inputEl = $state<HTMLInputElement | null>(null);
    let listEl = $state<HTMLElement | null>(null);
    let blueprintCommands = $state<Command[]>([]);
    let loadingBlueprints = $state(false);
    let blueprintError = $state(false);
    const listId = $props.id();

    function matches(cmd: Command, q: string): boolean {
        if (!q) return true;
        const hay = (
            cmd.label +
            ' ' +
            cmd.category +
            ' ' +
            (cmd.keywords ?? '')
        ).toLowerCase();
        const needle = q.toLowerCase();
        if (hay.includes(needle)) return true;
        let i = 0;
        for (let j = 0; j < hay.length && i < needle.length; j++) {
            if (hay[j] === needle[i]) i++;
        }
        return i === needle.length;
    }

    let visible = $derived(
        [...commands, ...blueprintCommands].filter(c =>
            c.visible ? c.visible() : true
        )
    );
    let filtered = $derived(visible.filter(c => matches(c, query.trim())));

    $effect(() => {
        const _ = query;
        activeIdx = 0;
    });

    $effect(() => {
        if (activeIdx >= filtered.length) activeIdx = 0;
    });

    $effect(() => {
        if (!open) return;
        const trigger = document.activeElement as HTMLElement | null;
        let cancelled = false;
        query = '';
        activeIdx = 0;
        blueprintCommands = [];
        blueprintError = false;
        loadingBlueprints = true;
        tick().then(() => {
            if (!cancelled) inputEl?.focus();
        });
        import('../../../../wailsjs/go/main/App')
            .then(({ListBlueprints}) => ListBlueprints())
            .then(result => {
                if (cancelled) return;
                const blueprints = (Array.isArray(result)
                    ? result
                    : []) as unknown as Blueprint[];
                blueprintCommands = blueprints
                    .sort((a, b) => (b.timestamp || 0) - (a.timestamp || 0))
                    .map((bp, i) => ({
                        id: `blueprint.${i}`,
                        label: bp.name,
                        category: 'Blueprint / Load into editor',
                        keywords: 'saved theme palette',
                        colors: bp.palette?.colors,
                        run: () => loadBlueprintIntoEditor(bp),
                    }));
            })
            .catch(() => {
                if (!cancelled) blueprintError = true;
            })
            .finally(() => {
                if (!cancelled) loadingBlueprints = false;
            });
        return () => {
            cancelled = true;
            if (trigger?.isConnected) trigger.focus();
        };
    });

    async function run(cmd: Command) {
        if (cmd.disabled?.()) return;
        onclose();
        await tick();
        try {
            await cmd.run();
        } catch (e) {
            showToast(e instanceof Error ? e.message : 'Command failed');
        }
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === 'Escape') {
            onclose();
            e.preventDefault();
            e.stopPropagation();
        } else if (e.key === 'ArrowDown') {
            activeIdx = filtered.length ? (activeIdx + 1) % filtered.length : 0;
            scrollActiveIntoView();
            e.preventDefault();
        } else if (e.key === 'ArrowUp') {
            activeIdx = filtered.length
                ? (activeIdx - 1 + filtered.length) % filtered.length
                : 0;
            scrollActiveIntoView();
            e.preventDefault();
        } else if (e.key === 'Enter') {
            const cmd = filtered[activeIdx];
            if (cmd) run(cmd);
            e.preventDefault();
            e.stopPropagation();
        } else if (e.key === 'Home' && (e.ctrlKey || e.metaKey)) {
            activeIdx = 0;
            scrollActiveIntoView();
            e.preventDefault();
        } else if (e.key === 'End' && (e.ctrlKey || e.metaKey)) {
            activeIdx = Math.max(0, filtered.length - 1);
            scrollActiveIntoView();
            e.preventDefault();
        } else if (e.key === 'Tab') {
            e.preventDefault();
            inputEl?.focus();
        }
        // Do not let app-wide shortcuts act on the editor behind the dialog.
        e.stopPropagation();
    }

    function scrollActiveIntoView() {
        queueMicrotask(() => {
            const el = listEl?.querySelector<HTMLElement>(
                `[data-cmd-idx="${activeIdx}"]`
            );
            el?.scrollIntoView({block: 'nearest'});
        });
    }
</script>

{#if open}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
        class="fixed inset-0 z-50 flex items-start justify-center bg-black/50 pt-[12vh]"
        onclick={e => {
            if (e.target === e.currentTarget) onclose();
        }}
        onkeydown={handleKeydown}
    >
        <div
            class="border-border bg-bg-secondary flex max-h-[80vh] w-[520px] max-w-[90vw] flex-col border shadow-2xl"
            role="dialog"
            aria-modal="true"
            aria-label="Command palette"
        >
            <div class="border-border border-b">
                <input
                    bind:this={inputEl}
                    bind:value={query}
                    type="text"
                    class="text-fg-primary placeholder:text-fg-dimmed w-full bg-transparent px-4 py-3 text-[12px] outline-none"
                    placeholder="Search commands and blueprints..."
                    aria-label="Search commands and blueprints"
                    role="combobox"
                    aria-autocomplete="list"
                    aria-expanded={true}
                    aria-controls={listId}
                    aria-activedescendant={filtered[activeIdx]
                        ? `${listId}-${activeIdx}`
                        : undefined}
                    spellcheck={false}
                    autocomplete="off"
                />
            </div>
            <div
                bind:this={listEl}
                id={listId}
                class="max-h-[50vh] min-h-0 overflow-y-auto"
                role="listbox"
                aria-label="Commands and blueprints"
            >
                {#if filtered.length === 0}
                    <div
                        class="text-fg-dimmed px-4 py-6 text-center text-[11px]"
                    >
                        No matching commands or blueprints
                    </div>
                {:else}
                    {#each filtered as cmd, i (cmd.id)}
                        {@const disabledReason = cmd.disabled?.()}
                        <button
                            type="button"
                            id={`${listId}-${i}`}
                            tabindex="-1"
                            data-cmd-idx={i}
                            class="flex w-full items-center justify-between gap-3 px-4 py-2 text-left transition-colors
                                {i === activeIdx
                                ? 'bg-accent/10 text-fg-primary'
                                : 'text-fg-secondary hover:bg-bg-surface'}"
                            onclick={() => run(cmd)}
                            onmousedown={e => e.preventDefault()}
                            onmouseenter={() => (activeIdx = i)}
                            role="option"
                            aria-selected={i === activeIdx}
                            aria-disabled={!!disabledReason}
                            title={disabledReason}
                            class:opacity-50={!!disabledReason}
                        >
                            <div class="flex min-w-0 flex-col gap-0.5">
                                <span class="truncate text-[12px]"
                                    >{cmd.label}</span
                                >
                                <span
                                    class="text-fg-dimmed text-[9px] uppercase tracking-wider"
                                    >{disabledReason ?? cmd.category}</span
                                >
                            </div>
                            {#if cmd.colors?.length}
                                <span class="flex shrink-0" aria-hidden="true">
                                    {#each cmd.colors.slice(0, 8) as color}
                                        <span
                                            class="h-3 w-2 sm:w-3"
                                            style:background={color}
                                        ></span>
                                    {/each}
                                </span>
                            {:else if cmd.shortcut}
                                <span
                                    class="text-fg-dimmed shrink-0 font-mono text-[10px]"
                                    >{cmd.shortcut}</span
                                >
                            {/if}
                        </button>
                    {/each}
                {/if}
            </div>
            {#if loadingBlueprints || blueprintError}
                <p class="text-fg-dimmed px-4 py-2 text-[10px]" role="status">
                    {loadingBlueprints
                        ? 'Loading saved blueprints...'
                        : 'Blueprints unavailable. Reopen to retry; commands still work.'}
                </p>
            {/if}
            <div
                class="border-border text-fg-dimmed flex items-center justify-between gap-2 border-t px-4 py-1.5 text-[9px]"
            >
                <span>↑↓ navigate · ↵ run · Esc close</span>
                <span class="tabular-nums"
                    >{filtered.length} / {visible.length}</span
                >
            </div>
        </div>
    </div>
{/if}
