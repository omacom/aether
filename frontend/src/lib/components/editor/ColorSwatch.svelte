<script lang="ts">
    import {
        setLockedColor,
        toggleColorSelection,
    } from '$lib/stores/theme.svelte';
    import {
        isLightColor,
        copyColor,
        contrastRatio,
        contrastLevel,
    } from '$lib/utils/color';
    import {setEyedropperActive, setColorDrag} from '$lib/stores/ui.svelte';
    import {onActivate} from '$lib/utils/keyboard';
    import LockIcon from '$lib/components/shared/LockIcon.svelte';
    import ContextMenu from '$lib/components/shared/ContextMenu.svelte';

    let {
        color,
        index,
        label,
        role = '',
        contrastAgainst = '',
        locked,
        selected,
        active = false,
        focused = false,
        onclick,
    }: {
        color: string;
        index: number;
        label: string;
        role?: string;
        contrastAgainst?: string;
        locked: boolean;
        selected: boolean;
        // The color picker is open on this swatch.
        active?: boolean;
        focused?: boolean;
        onclick: () => void;
    } = $props();

    let menu = $state({open: false, x: 0, y: 0});

    function handleContextMenu(e: MouseEvent) {
        e.preventDefault();
        menu = {open: true, x: e.clientX, y: e.clientY};
    }

    let menuItems = $derived([
        {
            kind: 'item' as const,
            label: 'Edit color…',
            onSelect: onclick,
            kbd: 'Click',
        },
        {
            kind: 'item' as const,
            label: 'Copy hex',
            onSelect: () => copyColor(color),
            kbd: 'Ctrl+Click',
        },
        {kind: 'divider' as const},
        {
            kind: 'item' as const,
            label: locked ? 'Unlock' : 'Lock',
            onSelect: () => setLockedColor(index, !locked),
        },
        {
            kind: 'item' as const,
            label: selected ? 'Deselect' : 'Add to selection',
            onSelect: () => toggleColorSelection(index),
            kbd: 'Shift+Click',
        },
        {kind: 'divider' as const},
        {
            kind: 'item' as const,
            label: 'Pick from wallpaper',
            onSelect: () => {
                onclick();
                setEyedropperActive(true);
            },
        },
    ]);

    let ratio = $derived(
        contrastAgainst ? contrastRatio(color, contrastAgainst) : 0
    );
    let level = $derived(contrastAgainst ? contrastLevel(ratio) : 'fail');
    // Skip the badge for the background slot itself (BG vs BG = 1:1, useless).
    let showBadge = $derived(
        !!contrastAgainst &&
            color.toLowerCase() !== contrastAgainst.toLowerCase()
    );

    function toggleLock(event: MouseEvent) {
        event.stopPropagation();
        toggleLockBare();
    }

    function toggleLockBare() {
        setLockedColor(index, !locked);
    }

    function handleClick(event: MouseEvent) {
        if (didDrag) {
            didDrag = false;
            return;
        }
        if (event.ctrlKey || event.metaKey) {
            event.preventDefault();
            copyColor(color);
            return;
        }
        if (event.shiftKey) {
            event.preventDefault();
            toggleColorSelection(index);
            return;
        }
        if (!locked) onclick();
    }

    let light = $derived(isLightColor(color));
    let ink = $derived(
        light ? 'rgba(10,10,16,0.86)' : 'rgba(255,255,255,0.92)'
    );
    let ring = $derived(
        active
            ? '0 0 0 2px var(--color-bg-primary), 0 0 0 4px var(--color-accent)'
            : selected
              ? 'inset 0 0 0 2px var(--color-accent)'
              : 'inset 0 0 0 1px rgba(128,128,128,0.18)'
    );

    let isDragging = $state(false);
    let didDrag = $state(false);
    let pendingDrag: {startX: number; startY: number; color: string} | null =
        null;

    function onMouseDown(e: MouseEvent) {
        if (e.button !== 0) return;
        clearDrag();
        didDrag = false;
        if (e.ctrlKey || e.metaKey || e.shiftKey) return;
        if (
            (e.target as Element).closest('[role="button"]') !== e.currentTarget
        )
            return;
        pendingDrag = {startX: e.clientX, startY: e.clientY, color};
        window.addEventListener('mousemove', onDragMove);
        window.addEventListener('mouseup', onDragUp);
        window.addEventListener('blur', clearDrag);
        window.addEventListener('keydown', onDragKeydown);
    }

    function onDragMove(e: MouseEvent) {
        if (!(e.buttons & 1)) {
            clearDrag();
            return;
        }
        if (!pendingDrag) return;
        if (isDragging) {
            e.preventDefault();
            setColorDrag({
                color: pendingDrag.color,
                x: e.clientX,
                y: e.clientY,
            });
            return;
        }
        const dx = e.clientX - pendingDrag.startX;
        const dy = e.clientY - pendingDrag.startY;
        if (Math.hypot(dx, dy) >= 4) {
            didDrag = true;
            isDragging = true;
            e.preventDefault();
            setColorDrag({
                color: pendingDrag.color,
                x: e.clientX,
                y: e.clientY,
            });
        }
    }

    function onDragKeydown(e: KeyboardEvent) {
        if (e.key === 'Escape') {
            e.preventDefault();
            clearDrag();
        }
    }

    function onDragUp(e: MouseEvent) {
        if (e.button === 0) clearDrag();
    }

    function clearDrag() {
        window.removeEventListener('mousemove', onDragMove);
        window.removeEventListener('mouseup', onDragUp);
        window.removeEventListener('blur', clearDrag);
        window.removeEventListener('keydown', onDragKeydown);
        if (pendingDrag || isDragging) setColorDrag(null);
        pendingDrag = null;
        isDragging = false;
    }

    $effect(() => {
        const _ = color;
        return clearDrag;
    });
</script>

<!-- Keyboard handling lives on the parent ColorPaletteGrid (roving tabindex). -->
<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div
    class="focus-visible:outline-accent group relative h-[68px] w-full transition-transform duration-[120ms] focus:outline-none focus-visible:z-10 focus-visible:outline-2 focus-visible:outline-offset-2
    {locked
        ? 'cursor-default'
        : 'cursor-pointer hover:z-[2] hover:-translate-y-0.5'}
    {active ? 'z-[1]' : ''}
    {isDragging ? 'opacity-60' : ''}"
    style:background-color={color}
    style:box-shadow={ring}
    style:color={ink}
    role="button"
    tabindex={focused ? 0 : -1}
    data-swatch-idx={index}
    onclick={handleClick}
    onmousedown={onMouseDown}
    oncontextmenu={handleContextMenu}
    title={`${label}${role ? ` · ${role}` : ''}\n${color}${
        showBadge ? `\nContrast vs BG: ${ratio.toFixed(2)}:1 (${level})` : ''
    }\nClick edit · Ctrl+click copy · Shift+click select · Right-click for menu\nDrag to copy into a template override\n← → ↑ ↓ navigate · Enter open · L lock · C copy`}
>
    {#if role}
        <span
            class="opacity-72 pointer-events-none absolute left-[7px] top-1.5 select-none font-mono text-[9.5px] font-semibold leading-none"
        >
            {role}
        </span>
    {/if}

    {#if showBadge}
        <span
            class="pointer-events-none absolute right-[5px] top-[5px] select-none whitespace-nowrap px-[3px] py-px font-mono text-[9px] font-semibold leading-none
            {level === 'fail' ? 'bg-destructive text-white' : ''}"
            aria-label={`Contrast ${ratio.toFixed(2)} to 1, ${level}`}
        >
            {level === 'fail' ? '!' : level}
        </span>
    {/if}

    <span
        class="absolute left-[3px] top-4 z-10 flex h-5 w-5 cursor-pointer items-center justify-center
      {locked
            ? 'opacity-75 hover:opacity-100'
            : 'opacity-0 transition-opacity hover:!opacity-100 group-hover:opacity-40'}"
        onclick={toggleLock}
        onkeydown={onActivate(toggleLockBare)}
        role="button"
        tabindex="0"
        title={locked ? 'Unlock color' : 'Lock color'}
        aria-label={locked ? 'Unlock color' : 'Lock color'}
    >
        <LockIcon {locked} size="h-[11px] w-[11px]" strokeWidth={2.2} />
    </span>

    <span
        class="opacity-82 pointer-events-none absolute bottom-1.5 left-[7px] right-1 overflow-hidden whitespace-nowrap font-mono text-[10px] font-medium leading-none"
    >
        {color.toUpperCase()}
    </span>
</div>

<ContextMenu
    open={menu.open}
    x={menu.x}
    y={menu.y}
    items={menuItems}
    onclose={() => (menu = {...menu, open: false})}
/>
