<script lang="ts">
    import Modal from './Modal.svelte';
    import DialogHeader from './DialogHeader.svelte';
    import Kbd from './Kbd.svelte';

    let {open, onclose}: {open: boolean; onclose: () => void} = $props();

    const keybindings = [
        {
            group: 'General',
            binds: [
                {keys: 'Ctrl+P', desc: 'Command palette'},
                {keys: 'Ctrl+K', desc: 'Show keyboard shortcuts'},
                {keys: 'Ctrl+B', desc: 'Toggle sidebar'},
                {keys: 'Ctrl+Enter', desc: 'Apply theme'},
                {keys: 'Ctrl+J', desc: 'Save as new theme folder'},
                {keys: 'Ctrl+S', desc: 'Save blueprint'},
                {keys: 'Ctrl+Z', desc: 'Undo'},
                {keys: 'Ctrl+Shift+Z', desc: 'Redo'},
                {keys: 'Ctrl++', desc: 'Zoom in'},
                {keys: 'Ctrl+-', desc: 'Zoom out'},
                {keys: 'Ctrl+0', desc: 'Reset zoom'},
                {keys: 'Escape', desc: 'Close dialog / color picker'},
            ],
        },
        {
            group: 'Color Picker',
            binds: [
                {keys: 'Shift+C', desc: 'Copy hex value'},
                {keys: 'Shift+V', desc: 'Paste hex value'},
            ],
        },
        {
            group: 'Palette',
            binds: [
                {keys: 'Click', desc: 'Open color picker'},
                {keys: 'Ctrl+Click', desc: 'Copy hex value'},
                {keys: 'Shift+Click', desc: 'Toggle selection'},
            ],
        },
        {
            group: 'Image Editor',
            binds: [
                {keys: 'C', desc: 'Toggle crop mode'},
                {keys: 'R', desc: 'Reset all adjustments'},
                {keys: 'B', desc: 'Toggle before/after'},
                {keys: 'Space', desc: 'Toggle before/after'},
                {keys: '[', desc: 'Rotate 90° CCW'},
                {keys: ']', desc: 'Rotate 90° CW'},
                {keys: 'F', desc: 'Flip horizontal'},
                {keys: 'V', desc: 'Flip vertical'},
                {keys: 'Ctrl+Z', desc: 'Undo filter change'},
                {keys: 'Ctrl+Shift+Z', desc: 'Redo filter change'},
                {keys: 'Ctrl+Enter', desc: 'Apply and close'},
                {keys: 'Escape', desc: 'Exit crop / close'},
            ],
        },
        {
            group: 'Curves Editor',
            binds: [
                {keys: 'Click', desc: 'Add or select point'},
                {keys: 'Ctrl+Click', desc: 'Remove point'},
                {keys: 'Arrow Keys', desc: 'Nudge selected point'},
                {keys: 'Shift+Arrow', desc: 'Nudge by 10×'},
                {keys: 'Delete', desc: 'Remove selected point'},
            ],
        },
    ];
</script>

<Modal {open} {onclose} bare panelClass="w-[420px]" label="Keyboard shortcuts">
    <DialogHeader title="Keyboard shortcuts" {onclose} />
    <div class="max-h-[60vh] overflow-y-auto px-5 pb-5 pt-2">
        {#each keybindings as section}
            <h4
                class="text-fg-dimmed pb-1.5 pt-3.5 text-[10px] font-semibold uppercase tracking-[0.14em]"
            >
                {section.group}
            </h4>
            {#each section.binds as bind}
                <div class="flex h-7 items-center justify-between gap-3">
                    <span class="text-fg-secondary text-[12px]"
                        >{bind.desc}</span
                    >
                    <div class="flex shrink-0 gap-1">
                        <!-- Split on "+" only when a key follows it, so
                             "Ctrl++" shows Ctrl and +. -->
                        {#each bind.keys.split(/\+(?=.)/) as part}
                            <Kbd class="min-w-[22px] justify-center">{part}</Kbd
                            >
                        {/each}
                    </div>
                </div>
            {/each}
        {/each}
    </div>
</Modal>
