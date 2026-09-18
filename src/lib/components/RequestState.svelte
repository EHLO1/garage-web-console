<script lang="ts">
  import Button from './Button.svelte';
  let {
    loading,
    error,
    retry
  }: { loading: boolean; error: string; retry: () => void } = $props();
</script>

{#if error}<div
    role="alert"
    class="mb-5 flex items-center justify-between gap-4 rounded-lg border border-destructive p-4"
  >
    <p>{error}</p>
    <Button variant="outline" onclick={retry}>Retry</Button>
  </div>
{:else if loading}<p role="status" class="py-6 text-sm text-muted-foreground">
    Loading…
  </p>{/if}
