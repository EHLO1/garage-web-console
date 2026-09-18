<script lang="ts">
  import { uploadStore } from '#src/stores/upload-store.ts';
  import Button from './Button.svelte';
  let active = $derived(
    $uploadStore.tasks.filter(
      (t) => t.status === 'pending' || t.status === 'uploading'
    ).length
  );
  let totalBytes = $derived(
    $uploadStore.tasks.reduce((sum, task) => sum + task.size, 0)
  );
  let loadedBytes = $derived(
    $uploadStore.tasks.reduce(
      (sum, task) =>
        sum +
        (task.status === 'success'
          ? task.size
          : Math.min(task.loaded, task.size)),
      0
    )
  );
  let progress = $derived(
    totalBytes ? Math.round((loadedBytes / totalBytes) * 100) : active ? 0 : 100
  );
</script>

{#if $uploadStore.tasks.length}
  <aside
    aria-label="Uploads"
    class="fixed bottom-4 right-4 z-40 max-h-[60vh] w-[calc(100%-2rem)] max-w-sm overflow-auto rounded-xl border bg-card p-4 shadow-xl"
  >
    <div class="flex items-center justify-between">
      <button class="font-semibold" onclick={uploadStore.toggleCollapsed}>
        Uploads ({active} active) {$uploadStore.collapsed ? '▴' : '▾'}
      </button>
      <Button variant="ghost" onclick={uploadStore.clearFinished}>
        Clear finished
      </Button>
    </div>
    <progress
      aria-label="Overall upload progress"
      class="mt-3 w-full"
      max="100"
      value={progress}
    ></progress>
    <p class="muted">{progress}% uploaded</p>
    {#if !$uploadStore.collapsed}
      {#each $uploadStore.tasks as task (task.id)}
        <div class="mt-3 border-t pt-3 text-sm">
          <p class="truncate" title={task.key}>{task.name}</p>
          <progress
            aria-label={task.name}
            class="w-full"
            max="100"
            value={task.progress}
          ></progress>
          <div class="flex items-center justify-between">
            <span class="muted">
              {task.error || task.status} · {task.progress}%
            </span>
            {#if task.status === 'uploading' || task.status === 'pending'}<Button
                variant="ghost"
                onclick={() => uploadStore.cancel(task.id)}
              >
                Cancel
              </Button>{:else if task.status === 'error'}<Button
                variant="ghost"
                onclick={() => uploadStore.retry(task.id)}
              >
                Retry
              </Button>{/if}
          </div>
        </div>
      {/each}
    {/if}
  </aside>
{/if}
