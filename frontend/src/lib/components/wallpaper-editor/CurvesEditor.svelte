<script lang="ts">
    import {buildCurveLUT} from '$lib/utils/canvas-filters';
    import {getLightMode} from '$lib/stores/theme.svelte';
    import {isLightColor} from '$lib/utils/color';

    const W = 256;
    const PAD = 0;

    let {
        points = $bindable<[number, number][]>([]),
        histogram = [] as number[],
        onchange = () => {},
        height = 192,
        label = '',
        compact = false,
    }: {
        points: [number, number][];
        histogram: number[];
        onchange: () => void;
        // Canvas height in pixels. The width stays 256 and scales to fit.
        height?: number;
        // Small caption in the top-left corner of the canvas.
        label?: string;
        // Show the hint row only while a point is selected, and omit the
        // curve Reset button. The caller supplies its own reset.
        compact?: boolean;
    } = $props();

    let H = $derived(height);

    let canvasEl = $state<HTMLCanvasElement | null>(null);
    let dragging = $state<number | null>(null);
    let selected = $state<number | null>(null);

    // Clamp selection whenever points change length. Avoids a stale index
    // pointing past the end after a deletion or reset.
    $effect(() => {
        if (selected !== null && selected >= points.length) selected = null;
    });

    // Redraw whenever points, histogram, selection, or theme mode change.
    $effect(() => {
        const _ = JSON.stringify(points);
        const __ = histogram.length;
        const ___ = getLightMode();
        const ____ = selected;
        const _____ = H;
        draw();
    });

    function toCanvas(px: number, py: number): [number, number] {
        return [PAD + px * W, PAD + (1 - py) * H];
    }

    function fromCanvas(cx: number, cy: number): [number, number] {
        return [
            Math.max(0, Math.min(1, (cx - PAD) / W)),
            Math.max(0, Math.min(1, 1 - (cy - PAD) / H)),
        ];
    }

    function draw() {
        if (!canvasEl) return;
        const ctx = canvasEl.getContext('2d');
        if (!ctx) return;
        const cw = canvasEl.width;
        const ch = canvasEl.height;
        ctx.clearRect(0, 0, cw, ch);

        // Pick ink against the actually-painted bg, not just the user's
        // light/dark toggle — a system Omarchy theme can apply a light bg
        // while the toggle is still off, leaving white ink invisible.
        const bgVar = getComputedStyle(document.documentElement)
            .getPropertyValue('--color-bg-primary')
            .trim();
        const light = bgVar ? isLightColor(bgVar) : getLightMode();
        const ink = (alpha: number) =>
            light ? `rgba(0,0,0,${alpha})` : `rgba(255,255,255,${alpha})`;
        const rootStyle = getComputedStyle(document.documentElement);
        const accent =
            rootStyle.getPropertyValue('--color-accent').trim() ||
            (light ? '#2563eb' : '#89b4fa');
        const knobFill =
            rootStyle.getPropertyValue('--color-bg-secondary').trim() ||
            (light ? '#ececef' : '#16161b');

        // Histogram bars
        if (histogram.length === 256) {
            const max = Math.max(...histogram);
            if (max > 0) {
                ctx.fillStyle = ink(0.055);
                for (let i = 0; i < 256; i++) {
                    const barH = (histogram[i] / max) * H;
                    ctx.fillRect(PAD + i, PAD + H - barH, 1, barH);
                }
            }
        }

        // Grid lines
        ctx.strokeStyle = ink(light ? 0.11 : 0.075);
        ctx.lineWidth = 1;
        for (let i = 1; i < 4; i++) {
            const x = PAD + (W * i) / 4;
            const y = PAD + (H * i) / 4;
            ctx.beginPath();
            ctx.moveTo(x, PAD);
            ctx.lineTo(x, PAD + H);
            ctx.stroke();
            ctx.beginPath();
            ctx.moveTo(PAD, y);
            ctx.lineTo(PAD + W, y);
            ctx.stroke();
        }

        // Diagonal reference (identity)
        ctx.strokeStyle = ink(0.14);
        ctx.setLineDash([4, 4]);
        ctx.beginPath();
        ctx.moveTo(...toCanvas(0, 0));
        ctx.lineTo(...toCanvas(1, 1));
        ctx.stroke();
        ctx.setLineDash([]);

        // Curve from LUT
        const lut = buildCurveLUT(points);
        ctx.strokeStyle = accent;
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        for (let i = 0; i < 256; i++) {
            const x = PAD + i;
            const val = lut ? lut[i] / 255 : i / 255;
            const y = PAD + (1 - val) * H;
            if (i === 0) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
        }
        ctx.stroke();

        // Endpoints (always drawn small + dimmed)
        for (const pt of [
            {x: 0, y: 0},
            {x: 1, y: 1},
        ]) {
            const [cx, cy] = toCanvas(pt.x, pt.y);
            ctx.fillStyle = ink(0.35);
            ctx.fillRect(cx - 2.5, cy - 2.5, 5, 5);
        }

        // User points: square knobs with an accent outline. The selected
        // point fills with the accent.
        for (let i = 0; i < points.length; i++) {
            const [x, y] = points[i];
            const [cx, cy] = toCanvas(x, y);
            const isSel = selected === i;
            const half = isSel ? 5 : 4;
            ctx.fillStyle = isSel ? accent : knobFill;
            ctx.fillRect(cx - half, cy - half, half * 2, half * 2);
            ctx.strokeStyle = accent;
            ctx.lineWidth = 1.5;
            ctx.strokeRect(cx - half, cy - half, half * 2, half * 2);
        }
    }

    function getMousePos(e: MouseEvent): [number, number] {
        const rect = canvasEl!.getBoundingClientRect();
        const scaleX = canvasEl!.width / rect.width;
        const scaleY = canvasEl!.height / rect.height;
        return [
            (e.clientX - rect.left) * scaleX,
            (e.clientY - rect.top) * scaleY,
        ];
    }

    function findNearestPoint(cx: number, cy: number): number | null {
        let best = -1;
        let bestDist = 15; // max grab distance in canvas pixels
        for (let i = 0; i < points.length; i++) {
            const [px, py] = toCanvas(points[i][0], points[i][1]);
            const dist = Math.sqrt((cx - px) ** 2 + (cy - py) ** 2);
            if (dist < bestDist) {
                bestDist = dist;
                best = i;
            }
        }
        return best >= 0 ? best : null;
    }

    function handleMouseDown(e: MouseEvent) {
        if (!canvasEl) return;
        canvasEl.focus();
        const [cx, cy] = getMousePos(e);

        // Right-click or ctrl+click removes nearest point
        if (e.button === 2 || (e.button === 0 && e.ctrlKey)) {
            e.preventDefault();
            const idx = findNearestPoint(cx, cy);
            if (idx !== null) {
                points = points.filter((_, i) => i !== idx);
                selected = null;
                onchange();
            }
            return;
        }

        // Left-click: try to grab existing point, else add new
        const idx = findNearestPoint(cx, cy);
        if (idx !== null) {
            dragging = idx;
            selected = idx;
        } else {
            const [x, y] = fromCanvas(cx, cy);
            points = [...points, [x, y]];
            dragging = points.length - 1;
            selected = points.length - 1;
            onchange();
        }
    }

    function handleMouseMove(e: MouseEvent) {
        if (dragging === null || !canvasEl) return;
        const [cx, cy] = getMousePos(e);
        const [x, y] = fromCanvas(cx, cy);
        // Clamp x to avoid overlapping endpoints
        const clampedX = Math.max(0.01, Math.min(0.99, x));
        points = points.map((p, i) =>
            i === dragging ? ([clampedX, y] as [number, number]) : p
        );
        onchange();
    }

    function handleMouseUp() {
        dragging = null;
    }

    function handleKeyDown(e: KeyboardEvent) {
        if (selected === null || selected >= points.length) return;

        // Delete / Backspace removes the selected user point
        if (e.key === 'Delete' || e.key === 'Backspace') {
            points = points.filter((_, i) => i !== selected);
            selected = null;
            onchange();
            e.preventDefault();
            return;
        }

        // Arrow keys nudge. Shift increases step.
        const step = e.shiftKey ? 10 / 255 : 1 / 255;
        let dx = 0;
        let dy = 0;
        switch (e.key) {
            case 'ArrowLeft':
                dx = -step;
                break;
            case 'ArrowRight':
                dx = step;
                break;
            case 'ArrowUp':
                dy = step;
                break;
            case 'ArrowDown':
                dy = -step;
                break;
            default:
                return;
        }
        e.preventDefault();
        const [px, py] = points[selected];
        const nx = Math.max(0.01, Math.min(0.99, px + dx));
        const ny = Math.max(0, Math.min(1, py + dy));
        points = points.map((p, i) =>
            i === selected ? ([nx, ny] as [number, number]) : p
        );
        onchange();
    }
</script>

<div class="space-y-1.5">
    <div class="relative">
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <canvas
            bind:this={canvasEl}
            width={W}
            height={H}
            tabindex="0"
            class="border-border bg-bg-primary focus:border-accent block w-full cursor-crosshair border outline-none"
            style="height: {H}px; image-rendering: auto;"
            title="Click to add a point. Drag to adjust. Right-click to remove."
            onmousedown={handleMouseDown}
            onmousemove={handleMouseMove}
            onmouseup={handleMouseUp}
            onmouseleave={handleMouseUp}
            onkeydown={handleKeyDown}
            oncontextmenu={e => e.preventDefault()}
        ></canvas>
        {#if label}
            <span
                class="text-fg-dimmed pointer-events-none absolute left-1.5 top-1 font-mono text-[9.5px] font-medium"
                >{label}</span
            >
        {/if}
    </div>
    {#if !compact || selected !== null}
        <div class="flex items-center justify-between gap-2">
            <span class="text-fg-dimmed text-[10px] leading-snug">
                {#if selected !== null}
                    Point {selected + 1} selected · ← ↑ ↓ → nudge · Shift×10 · Del
                    remove
                {:else}
                    Click to add · drag to adjust · click a point then arrow
                    keys to nudge
                {/if}
            </span>
            {#if points.length > 0 && !compact}
                <button
                    type="button"
                    class="text-fg-secondary hover:text-fg-primary border-border hover:border-border-focus shrink-0 border px-2 py-0.5 text-[10.5px] transition-colors"
                    onclick={() => {
                        points = [];
                        selected = null;
                        onchange();
                    }}
                    title="Clear all curve points"
                    aria-label="Reset curve">Reset</button
                >
            {/if}
        </div>
    {/if}
</div>
