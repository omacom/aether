<script lang="ts">
    import {onActivate} from '$lib/utils/keyboard';

    let {
        label,
        value,
        min,
        max,
        step,
        defaultValue,
        oninput,
        oncommit,
    }: {
        label: string;
        value: number;
        min: number;
        max: number;
        step: number;
        defaultValue: number;
        oninput: (value: number) => void;
        oncommit?: () => void;
    } = $props();

    let editing = $state(false);
    let editValue = $state('');
    let editInput = $state<HTMLInputElement | null>(null);

    $effect(() => {
        if (editing) {
            editInput?.focus();
            editInput?.select();
        }
    });

    let changed = $derived(value !== defaultValue);
    let display = $derived(
        step < 1
            ? value.toFixed(1)
            : `${value > 0 && min < 0 ? '+' : ''}${value}`
    );

    // The track paints the accent fill from the default value to the
    // current value. The thumb covers the ends of the fill.
    let trackBackground = $derived.by(() => {
        const pct = (v: number) => ((v - min) / (max - min)) * 100;
        const lo = Math.min(pct(value), pct(defaultValue));
        const hi = Math.max(pct(value), pct(defaultValue));
        const rest = 'var(--color-bg-elevated)';
        return `linear-gradient(to right, ${rest} ${lo}%, var(--color-accent) ${lo}% ${hi}%, ${rest} ${hi}%)`;
    });

    function handleDblClick() {
        oninput(defaultValue);
        oncommit?.();
    }

    function startEdit() {
        editValue = step < 1 ? value.toFixed(1) : String(value);
        editing = true;
    }

    function commitEdit() {
        editing = false;
        const num = parseFloat(editValue);
        if (!isNaN(num)) {
            oninput(Math.max(min, Math.min(max, num)));
            oncommit?.();
        }
    }

    function handleEditKeydown(e: KeyboardEvent) {
        if (e.key === 'Enter') commitEdit();
        if (e.key === 'Escape') editing = false;
    }
</script>

<div class="flex flex-col gap-[3px]">
    <!-- Label + value row -->
    <div class="flex items-baseline justify-between">
        <span class="text-fg-secondary text-[12px]">{label}</span>

        {#if editing}
            <div class="flex items-center gap-1">
                <input
                    bind:this={editInput}
                    type="text"
                    class="text-fg-primary bg-bg-primary border-accent w-11 border px-1 py-0 text-right font-mono text-[11px] outline-none"
                    bind:value={editValue}
                    onblur={commitEdit}
                    onkeydown={handleEditKeydown}
                    aria-label="{label} value"
                />
                <button
                    class="text-accent hover:text-accent-hover"
                    onclick={commitEdit}
                    title="Apply"
                    aria-label="Apply {label} value"
                >
                    <svg
                        class="h-3 w-3"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2.5"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                    >
                        <polyline points="20 6 9 17 4 12"></polyline>
                    </svg>
                </button>
            </div>
        {:else}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <span
                class="cursor-text font-mono text-[11px] font-medium tabular-nums transition-colors
          {changed
                    ? 'text-fg-primary hover:text-accent'
                    : 'text-fg-dimmed hover:text-fg-secondary'}"
                role="button"
                tabindex="-1"
                onclick={startEdit}
                ondblclick={handleDblClick}
                onkeydown={onActivate(startEdit)}
                title="Click to type · Double-click to reset"
            >
                {display}
            </span>
        {/if}
    </div>

    <!-- Slider. The inline background overrides the flat track from
         app.css with the accent fill. -->
    <div class="flex h-3.5 items-center">
        <input
            type="range"
            class="w-full cursor-pointer"
            style:background={trackBackground}
            {min}
            {max}
            {step}
            {value}
            oninput={e => oninput(parseFloat(e.currentTarget.value))}
            onchange={() => oncommit?.()}
            ondblclick={handleDblClick}
            title="Double-click to reset"
            aria-label={label}
        />
    </div>
</div>
