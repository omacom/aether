<script lang="ts">
    import {tick} from 'svelte';
    import type {Command} from '$lib/commands/commands.svelte';
    import type {Blueprint} from '$lib/types/theme';
    import {loadBlueprintIntoEditor} from '$lib/actions/blueprintActions';
    import {showToast} from '$lib/stores/ui.svelte';
    import Kbd from './Kbd.svelte';
    import SearchIcon from './SearchIcon.svelte';

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

    // Group adjacent commands that share a category. The flat index stays
    // the same, so keyboard navigation follows the visible order.
    let groups = $derived.by(() => {
        const out: {category: string; items: {cmd: Command; idx: number}[]}[] =
            [];
        filtered.forEach((cmd, idx) => {
            const last = out[out.length - 1];
            if (last?.category === cmd.category) last.items.push({cmd, idx});
            else out.push({category: cmd.category, items: [{cmd, idx}]});
        });
        return out;
    });

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
        class="bg-scrim fixed inset-0 z-50 flex items-start justify-center pt-[12vh]"
        onclick={e => {
            if (e.target === e.currentTarget) onclose();
        }}
        onkeydown={handleKeydown}
    >
        <div
            class="border-border bg-bg-secondary shadow-(--shadow-panel) flex max-h-[72vh] w-[580px] max-w-[90vw] flex-col border"
            role="dialog"
            aria-modal="true"
            aria-label="Command palette"
        >
            <div
                class="border-border text-fg-dimmed flex h-[50px] shrink-0 items-center gap-2.5 border-b px-4"
            >
                <SearchIcon size="h-4 w-4" />
                <input
                    bind:this={inputEl}
                    bind:value={query}
                    type="text"
                    class="text-fg-primary min-w-0 flex-1 bg-transparent text-[14px] outline-none focus-visible:outline-none"
                    placeholder="Search commands and blueprints…"
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
                <Kbd class="text-fg-dimmed">Esc</Kbd>
            </div>
            <div
                bind:this={listEl}
                id={listId}
                class="min-h-0 flex-1 overflow-y-auto py-1.5"
                role="listbox"
                aria-label="Commands and blueprints"
            >
                {#if filtered.length === 0}
                    <div
                        class="text-fg-dimmed px-4 py-7 text-center text-[12px]"
                    >
                        No matching commands or blueprints
                    </div>
                {:else}
                    {#each groups as group, g (g + group.category)}
                        <div role="group" aria-label={group.category}>
                            <div
                                class="text-fg-dimmed px-4 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-[0.14em]"
                                aria-hidden="true"
                            >
                                {group.category}
                            </div>
                            {#each group.items as { cmd, idx: i } (cmd.id)}
                                {@const disabledReason = cmd.disabled?.()}
                                {@const active = i === activeIdx}
                                <button
                                    type="button"
                                    id={`${listId}-${i}`}
                                    tabindex="-1"
                                    data-cmd-idx={i}
                                    class="flex h-9 w-full items-center gap-3 px-4 text-left text-[13px] transition-colors
                                        {active
                                        ? 'bg-accent-muted text-fg-primary shadow-[inset_2px_0_0_var(--color-accent)]'
                                        : 'text-fg-secondary'}"
                                    onclick={() => run(cmd)}
                                    onmousedown={e => e.preventDefault()}
                                    onmouseenter={() => (activeIdx = i)}
                                    role="option"
                                    aria-selected={active}
                                    aria-disabled={!!disabledReason}
                                    title={disabledReason}
                                    class:opacity-50={!!disabledReason}
                                >
                                    <span class="min-w-0 flex-1 truncate"
                                        >{cmd.label}</span
                                    >
                                    {#if disabledReason}
                                        <span
                                            class="text-fg-dimmed shrink-0 text-[11px]"
                                            >{disabledReason}</span
                                        >
                                    {:else if cmd.colors?.length}
                                        <span
                                            class="flex shrink-0"
                                            aria-hidden="true"
                                        >
                                            {#each cmd.colors.slice(0, 8) as color}
                                                <span
                                                    class="h-3 w-2.5"
                                                    style:background={color}
                                                ></span>
                                            {/each}
                                        </span>
                                    {:else if cmd.shortcut}
                                        <kbd
                                            class="text-fg-dimmed shrink-0 font-mono text-[10.5px] font-medium"
                                            >{cmd.shortcut}</kbd
                                        >
                                    {/if}
                                </button>
                            {/each}
                        </div>
                    {/each}
                {/if}
            </div>
            {#if loadingBlueprints || blueprintError}
                <p
                    class="border-border text-fg-dimmed border-t px-4 py-2 text-[11px]"
                    role="status"
                >
                    {loadingBlueprints
                        ? 'Loading saved blueprints…'
                        : 'Blueprints unavailable. Reopen to retry. Commands still work.'}
                </p>
            {/if}
            <div
                class="border-border text-fg-dimmed flex shrink-0 items-center justify-between gap-2 border-t px-4 py-2 text-[11px]"
            >
                <span>↑↓ navigate · ↵ run · Esc close</span>
                <span class="font-mono tabular-nums"
                    >{filtered.length} / {visible.length}</span
                >
            </div>
        </div>
    </div>
{/if}
