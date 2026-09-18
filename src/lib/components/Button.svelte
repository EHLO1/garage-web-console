<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';
  import { cn } from '#lib/utils.ts';
  let {
    children,
    variant = 'default',
    class: className,
    type = 'button',
    ...rest
  }: HTMLButtonAttributes & {
    children?: Snippet;
    variant?: 'default' | 'outline' | 'ghost' | 'destructive';
  } = $props();
</script>

<button
  {type}
  class={cn(
    'inline-flex items-center justify-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50',
    variant === 'default'
      ? 'bg-primary text-primary-foreground hover:bg-primary/90'
      : variant === 'destructive'
        ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
        : variant === 'outline'
          ? 'border bg-background hover:bg-accent'
          : 'hover:bg-accent',
    className
  )}
  {...rest}
>
  {@render children?.()}
</button>
