<script lang="ts">
  import { Dialog } from 'bits-ui';
  import type { Snippet } from 'svelte';
  let {
    open = $bindable(false),
    title,
    description = '',
    children
  }: {
    open?: boolean;
    title: string;
    description?: string;
    children: Snippet;
  } = $props();
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 z-50 bg-black/60" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 z-50 max-h-[90dvh] w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-auto rounded-lg border bg-background p-6 shadow-lg"
    >
      <Dialog.Title class="mb-2 pr-8 text-lg font-semibold">
        {title}
      </Dialog.Title>
      <Dialog.Description class="mb-5 text-sm text-muted-foreground">
        {description}
      </Dialog.Description>
      <Dialog.Close
        class="absolute right-4 top-4 rounded px-2 py-1 hover:bg-accent"
        aria-label="Close dialog"
      >
        ✕
      </Dialog.Close>
      {@render children()}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
