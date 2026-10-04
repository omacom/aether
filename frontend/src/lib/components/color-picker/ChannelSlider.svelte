<script lang="ts">
    let {
        label,
        value,
        min = 0,
        max,
        step = 1,
        display,
        gradient,
        disabled = false,
        onchange,
        oncommit,
    }: {
        label: string;
        value: number;
        min?: number;
        max: number;
        step?: number;
        display: string;
        gradient: string;
        disabled?: boolean;
        onchange: (value: number) => void;
        oncommit: (raw: string) => void;
    } = $props();

    let editing = $state(false);
    let editValue = $state('');
    let inputEl = $state<HTMLInputElement | null>(null);

    $effect(() => {
        if (editing && inputEl) {
            inputEl.focus();
            inputEl.select();
        }
    });

    function startEdit() {
        if (disabled) return;
        editValue = display;
        editing = true;
    }

    // Guarded so Enter's commit() doesn't double-fire when the subsequent
    // onblur (from the unmounting input) would otherwise run commit again.
    function commit() {
        if (!editing) return;
        const raw = editValue;
        editing = false;
        oncommit(raw);
    }

    function handleKey(e: KeyboardEvent) {
        if (e.key === 'Enter') {
            e.preventDefault();
            commit();
        } else if (e.key === 'Escape') {
            e.preventDefault();
            editing = false;
        }
    }
</script>

<!-- The row is as tall as the 14px track. The input and its 20px handle
     extend 3px above and below the track. -->
<div class="flex h-[14px] items-center gap-2.5">
    <span
        class="text-fg-dimmed w-3 shrink-0 font-mono text-[11px] font-semibold"
        >{label}</span
    >

    <div class="group relative h-full flex-1 {disabled ? 'opacity-50' : ''}">
        <div
            class="absolute inset-0 shadow-[inset_0_0_0_1px_rgba(128,128,128,0.2)] group-focus-within:shadow-[inset_0_0_0_1px_var(--color-accent)]"
            style:background={gradient}
        ></div>
        <input
            type="range"
            class="channel-input absolute inset-x-0 -top-[3px] h-5 w-full cursor-pointer disabled:cursor-not-allowed"
            {min}
            {max}
            {step}
            {value}
            oninput={e => onchange(parseFloat(e.currentTarget.value))}
            {disabled}
            aria-label={label}
            aria-valuetext={display}
        />
    </div>

    {#if editing}
        <input
            type="text"
            bind:this={inputEl}
            bind:value={editValue}
            onblur={commit}
            onkeydown={handleKey}
            spellcheck={false}
            inputmode="decimal"
            aria-label="Edit {label} value"
            class="text-fg-primary bg-bg-primary border-accent -my-[3px] h-5 w-12 shrink-0 border px-1 text-right font-mono text-[11px] font-medium tabular-nums outline-none"
        />
    {:else}
        <button
            type="button"
            class="text-fg-secondary -my-[3px] h-5 w-12 shrink-0 text-right font-mono text-[11px] font-medium tabular-nums transition-colors
                {disabled ? 'cursor-default' : 'hover:text-fg-primary'}"
            onclick={startEdit}
            {disabled}
            title={disabled ? undefined : 'Click to type a value'}
            aria-label="Edit {label} value">{display}</button
        >
    {/if}
</div>

<style>
    .channel-input {
        -webkit-appearance: none;
        appearance: none;
        margin: 0;
        touch-action: none;
        background: transparent;
        outline: none;
    }

    .channel-input::-webkit-slider-runnable-track {
        height: 100%;
        border: 0;
        background: transparent;
    }

    .channel-input::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 6px;
        height: 20px;
        border: 0;
        background: #fff;
        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.55),
            0 1px 3px rgba(0, 0, 0, 0.35);
    }

    .channel-input:hover::-webkit-slider-thumb,
    .channel-input:focus-visible::-webkit-slider-thumb {
        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.55),
            0 0 0 3px rgba(255, 255, 255, 0.55),
            0 1px 4px rgba(0, 0, 0, 0.45);
    }

    .channel-input::-moz-range-track {
        height: 100%;
        border: 0;
        background: transparent;
    }

    .channel-input::-moz-range-thumb {
        width: 6px;
        height: 20px;
        border: 0;
        border-radius: 0;
        background: #fff;
        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.55),
            0 1px 3px rgba(0, 0, 0, 0.35);
    }
</style>
