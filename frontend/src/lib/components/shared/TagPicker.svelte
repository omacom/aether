<script lang="ts">
    import {
        getLabels,
        getLabelForPath,
        assignLabel,
        removeAssignment,
        createLabel,
        deleteLabel,
        LABEL_COLORS,
    } from '$lib/stores/tags.svelte';
    import CloseIcon from './CloseIcon.svelte';

    let {path}: {path: string} = $props();

    let currentLabel = $derived(getLabelForPath(path));
    let allLabels = $derived(getLabels());
    let open = $state(false);
    let creating = $state(false);
    let newName = $state('');
    let newNameInput = $state<HTMLInputElement | null>(null);

    $effect(() => {
        if (creating) newNameInput?.focus();
    });
    let newColor = $state(LABEL_COLORS[0]);
    let anchorEl: HTMLElement;

    function toggle(e: MouseEvent) {
        e.stopPropagation();
        e.preventDefault();
        open = !open;
        creating = false;
        newName = '';
        newColor = LABEL_COLORS[allLabels.length % LABEL_COLORS.length];
    }

    function handleCreate() {
        if (!newName.trim()) return;
        const label = createLabel(newName.trim(), newColor);
        assignLabel(path, label.id);
        newName = '';
        creating = false;
        open = false;
    }

    function pick(e: MouseEvent, labelId: string) {
        e.stopPropagation();
        assignLabel(path, labelId);
        open = false;
    }

    function remove(e: MouseEvent) {
        e.stopPropagation();
        removeAssignment(path);
        open = false;
    }
</script>

<div class="relative z-20" bind:this={anchorEl}>
    {#if currentLabel}
        <button
            class="flex h-5 max-w-[120px] items-center gap-1 px-1.5 text-[10.5px] font-medium transition-colors hover:brightness-125"
            style="background-color: {currentLabel.color}24; color: {currentLabel.color}; border: 1px solid {currentLabel.color}55;"
            onclick={toggle}
            title="Change label"
        >
            <span
                class="h-2 w-2 shrink-0"
                style:background-color={currentLabel.color}
            ></span>
            <span class="truncate">{currentLabel.name}</span>
        </button>
    {:else}
        <button
            class="text-fg-dimmed hover:text-fg-secondary border-border hover:border-border-focus flex h-5 w-5 items-center justify-center border transition-colors"
            onclick={toggle}
            aria-label="Add label"
            title="Add label"
        >
            <svg
                class="h-3 w-3"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
            >
                <path
                    d="M20.59 13.41 13.42 20.58a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z M7 7h.01"
                ></path>
            </svg>
        </button>
    {/if}

    {#if open}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div
            class="fixed inset-0 z-40"
            onclick={e => {
                e.stopPropagation();
                open = false;
                creating = false;
            }}
            role="presentation"
        ></div>
        <div
            class="border-border bg-bg-secondary shadow-(--shadow-panel) absolute bottom-full left-0 z-50 mb-1 min-w-[200px] border"
        >
            {#if allLabels.length > 0}
                <div class="py-1">
                    {#each allLabels as label}
                        <div class="group/label flex items-center">
                            <button
                                class="hover:bg-bg-hover hover:text-fg-primary flex h-[30px] flex-1 items-center gap-2 px-3 text-left text-[12px] transition-colors
                  {currentLabel?.id === label.id
                                    ? 'text-fg-primary'
                                    : 'text-fg-secondary'}"
                                onclick={e => pick(e, label.id)}
                            >
                                <span
                                    class="h-3 w-3 shrink-0"
                                    style:background-color={label.color}
                                ></span>
                                <span class="flex-1">{label.name}</span>
                                {#if currentLabel?.id === label.id}
                                    <svg
                                        class="text-accent h-3.5 w-3.5 shrink-0"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2.5"
                                    >
                                        <polyline points="20 6 9 17 4 12"
                                        ></polyline>
                                    </svg>
                                {/if}
                            </button>
                            <button
                                class="text-fg-dimmed hover:text-destructive flex h-[30px] w-8 items-center justify-center opacity-0 transition-opacity focus-visible:opacity-100 group-hover/label:opacity-100"
                                aria-label="Delete label {label.name}"
                                onclick={e => {
                                    e.stopPropagation();
                                    deleteLabel(label.id);
                                }}
                                title="Delete label"
                            >
                                <CloseIcon size="h-3 w-3" />
                            </button>
                        </div>
                    {/each}
                </div>
            {/if}

            {#if currentLabel}
                <div class="border-border border-t py-1">
                    <button
                        class="text-fg-secondary hover:text-fg-primary hover:bg-bg-hover h-[30px] w-full px-3 text-left text-[12px] transition-colors"
                        onclick={remove}>Remove label</button
                    >
                </div>
            {/if}

            <div class="border-border border-t py-1">
                {#if creating}
                    <!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->
                    <div
                        class="space-y-2 px-3 py-1.5"
                        onclick={e => e.stopPropagation()}
                        role="presentation"
                    >
                        <!-- Color selection -->
                        <div class="flex flex-wrap gap-1">
                            {#each LABEL_COLORS as c}
                                <button
                                    class="h-4 w-4 transition-shadow
                    {newColor === c
                                        ? 'ring-offset-bg-secondary ring-fg-primary ring-1 ring-offset-1'
                                        : 'hover:ring-border-focus hover:ring-1'}"
                                    style:background-color={c}
                                    onclick={e => {
                                        e.stopPropagation();
                                        newColor = c;
                                    }}
                                    aria-label="Pick color {c}"
                                    aria-pressed={newColor === c}
                                ></button>
                            {/each}
                        </div>
                        <!-- Name input + confirm -->
                        <div class="flex items-center gap-1">
                            <span
                                class="h-3 w-3 shrink-0"
                                style:background-color={newColor}
                            ></span>
                            <input
                                bind:this={newNameInput}
                                type="text"
                                class="text-fg-primary focus:border-accent border-border bg-bg-primary h-7 min-w-0 flex-1 border px-2 text-[12px] outline-none"
                                placeholder="Name..."
                                bind:value={newName}
                                onkeydown={e => {
                                    e.stopPropagation();
                                    if (e.key === 'Enter') handleCreate();
                                    if (e.key === 'Escape') creating = false;
                                }}
                                aria-label="Label name"
                            />
                            <button
                                class="text-accent hover:text-accent-hover p-1"
                                onclick={e => {
                                    e.stopPropagation();
                                    handleCreate();
                                }}
                                aria-label="Create label"
                                title="Create label"
                            >
                                <svg
                                    class="h-4 w-4"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2.5"
                                    aria-hidden="true"
                                >
                                    <polyline points="20 6 9 17 4 12"
                                    ></polyline>
                                </svg>
                            </button>
                        </div>
                    </div>
                {:else}
                    <button
                        class="text-accent hover:bg-bg-hover h-[30px] w-full px-3 text-left text-[12px] transition-colors"
                        onclick={e => {
                            e.stopPropagation();
                            creating = true;
                        }}>+ New label</button
                    >
                {/if}
            </div>
        </div>
    {/if}
</div>
