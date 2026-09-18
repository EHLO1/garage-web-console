<script lang="ts">
  import { onMount } from 'svelte';
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import type { LogsResponse } from '#lib/types/logs.ts';
  import Button from '#lib/components/Button.svelte';
  import RequestState from '#lib/components/RequestState.svelte';
  let search = $state('');
  let query = $state('');
  let level = $state('');
  let page = $state(1);
  let autoRefresh = $state(false);
  const limit = 50;
  $effect(() => {
    const value = search;
    const timer = setTimeout(() => {
      query = value;
      page = 1;
    }, 300);
    return () => clearTimeout(timer);
  });
  const logs = resource(() =>
    api.get<LogsResponse>('/logs', {
      params: { search: query, level, page, limit }
    })
  );
  onMount(() => {
    const timer = setInterval(() => {
      if (autoRefresh && !logs.loading) logs.reload();
    }, 3000);
    return () => clearInterval(timer);
  });
</script>

<div class="toolbar">
  <div>
    <h1 class="page-title">Audit Logs</h1>
    <p class="muted mt-1">Account and storage activity</p>
  </div>
  <div class="flex items-center gap-3">
    <label class="text-sm">
      <input type="checkbox" bind:checked={autoRefresh} />
       Auto refresh
    </label>
    <Button variant="outline" onclick={logs.reload}>Refresh</Button>
  </div>
</div>
<div class="mb-5 flex flex-wrap gap-3">
  <input
    class="input max-w-sm"
    aria-label="Search logs"
    placeholder="Search events…"
    bind:value={search}
  />
  <select
    class="input max-w-40"
    aria-label="Log level"
    bind:value={level}
    onchange={() => (page = 1)}
  >
    <option value="">All levels</option>
    {#each ['INFO', 'WARN', 'ERROR', 'DEBUG'] as value (value)}<option {value}>
        {value}
      </option>{/each}
  </select>
</div>
{#if logs.data}<div class="mb-4 flex flex-wrap gap-3">
    {#each Object.entries(logs.data.counts || {}) as [name, count] (name)}<span
        class="badge"
      >
        {name}: {count}
      </span>{/each}
  </div>{/if}
<RequestState loading={logs.loading} error={logs.error} retry={logs.reload} />
<section class="card">
  {#each logs.data?.entries || [] as entry, index (index)}<details
      class="border-b py-3"
    >
      <summary class="cursor-pointer text-sm">
        <span class="mr-3 text-muted-foreground">
          {new Date(entry.timestamp).toLocaleString()}
        </span>
        <span class="badge mr-3">{entry.level}</span>
        {entry.message}
      </summary>
      <pre
        class="mt-3 overflow-auto rounded bg-muted p-3 text-xs">{JSON.stringify(
          entry.context || {},
          null,
          2
        )}</pre>
    </details>{:else}{#if !logs.loading}<p class="muted py-8 text-center">
        No events found.
      </p>{/if}{/each}
</section>
<div class="mt-5 flex items-center justify-between">
  <p class="muted">
    Page {page} of {Math.max(1, Math.ceil((logs.data?.total || 0) / limit))} · {logs
      .data?.total || 0} events
  </p>
  <div class="flex gap-2">
    <Button
      variant="outline"
      disabled={page === 1 || logs.loading}
      onclick={() => page--}
    >
      Previous
    </Button><Button
      variant="outline"
      disabled={page * limit >= (logs.data?.total || 0) || logs.loading}
      onclick={() => page++}
    >
      Next
    </Button>
  </div>
</div>
