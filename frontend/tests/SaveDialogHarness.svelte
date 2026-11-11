<script lang="ts">
    import SaveDialog from '../src/lib/components/blueprints/SaveDialog.svelte';

    let open = $state(false);
</script>

<button onclick={() => (open = true)}>Open save</button>
<button onclick={() => (open = false)}>Close save</button>
<SaveDialog {open} onclose={() => (open = false)} onsave={() => {}} />
