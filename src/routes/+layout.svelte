<script lang="ts">
  import '#src/app/styles.css';
  import { Toaster } from 'svelte-sonner';
  import type { Snippet } from 'svelte';
  let { children }: { children: Snippet } = $props();
</script>

<svelte:head><title>Garage Web Console</title></svelte:head>
{@render children()}
<Toaster richColors />
