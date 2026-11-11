<script lang="ts">
    import type {YozakuraClock} from '$lib/stores/theme.svelte';

    // A sample of the shell, drawn from the derived roles. It shows theme
    // colours, not app chrome, so the inline styles are intentional.
    let {
        roles,
        glass,
        clock,
    }: {
        roles: Record<string, string>;
        glass: number;
        clock: YozakuraClock;
    } = $props();

    let pct = $derived(Math.round(glass * 100));
    const glassy = (key: string) =>
        `color-mix(in srgb, ${roles[key]} ${pct}%, transparent)`;
    const TERM = ['danger', 'ok', 'warn', 'info', 'accent', 'accent2'];
</script>

<div
    class="outline-border flex flex-col gap-2 p-2.5 outline outline-1"
    style:background-color={roles.ground}
    style:color={roles.text}
    aria-hidden="true"
    data-testid="yozakura-preview"
>
    <div class="flex items-center justify-between gap-2">
        <div
            class="flex items-center gap-1.5 px-2.5 py-1 font-mono text-[11px]"
            style:background-color={glassy('surface')}
            style:border="1px solid {roles.border}"
            style:border-radius="999px"
        >
            <span style:color={roles.accent}>●</span>
            <span style:font-weight={clock.digits === 'thin' ? 300 : 700}
                >12{clock.colon}34</span
            >
        </div>
        <span class="text-[10px]" style:color={roles.muted}
            >{clock.ornament}</span
        >
    </div>

    <div
        class="flex flex-col gap-1 p-2 text-[11px]"
        style:background-color={glassy('surface')}
        style:border="1px solid {roles.border}"
    >
        <span class="font-semibold">Card title</span>
        <span style:color={roles.muted}>Secondary text on a surface</span>
        <span
            class="mt-0.5 w-fit px-2 py-0.5 text-[10.5px] font-medium"
            style:background-color={roles.accent}
            style:color={roles.accentInk}>Action</span
        >
    </div>

    <div
        class="flex items-center gap-2 p-2 text-[11px]"
        style:background-color={glassy('surface2')}
        style:border-left="3px solid {roles.accent2}"
    >
        <span style:color={roles.accent2}>◆</span>
        <span>Notification</span>
    </div>

    <div class="flex h-3.5">
        {#each TERM as key (key)}
            <span class="flex-1" style:background-color={roles[key]}></span>
        {/each}
    </div>
</div>
