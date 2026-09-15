<script lang="ts">
    import CommandPalette from '../src/lib/components/shared/CommandPalette.svelte';
    import type {Command} from '../src/lib/commands/commands.svelte';

    let {commands = []}: {commands?: Command[]} = $props();
    let open = $state(false);
</script>

<button onclick={() => (open = true)}>Open commands</button>
<button>Outside</button>
<CommandPalette {open} {commands} onclose={() => (open = false)} />
