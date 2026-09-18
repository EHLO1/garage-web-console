<script lang="ts">
  import { auth } from '#lib/auth.ts';
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import { readableBytes, url } from '#lib/utils.ts';
  import type { Bucket } from '#lib/types/buckets.ts';
  import Button from '#lib/components/Button.svelte';
  import FormDialog from '#lib/components/FormDialog.svelte';
  import RequestState from '#lib/components/RequestState.svelte';
  import { Database } from '@lucide/svelte';
  const buckets = resource(() => api.get<Bucket[]>('/buckets'));
  let search = $state('');
  let create = $state(false);
  let manager = $derived(
    $auth?.user?.role === 'owner' || $auth?.user?.role === 'admin'
  );
  let filtered = $derived(
    (buckets.data || []).filter((b) =>
      `${b.id} ${b.globalAliases?.join(' ')}`
        .toLowerCase()
        .includes(search.toLowerCase())
    )
  );
</script>

<div class="toolbar">
  <div>
    <h1 class="page-title">Buckets</h1>
    <p class="muted mt-1">Manage your object storage</p>
  </div>
  {#if manager}<Button onclick={() => (create = true)}>
      Create bucket
    </Button>{/if}
</div>
<input
  aria-label="Search buckets"
  class="input mb-5 max-w-sm"
  placeholder="Search buckets…"
  bind:value={search}
/>
<RequestState
  loading={buckets.loading}
  error={buckets.error}
  retry={buckets.reload}
/>
<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
  {#each filtered as bucket (bucket.id)}<a
      class="card transition-colors hover:bg-accent/50"
      href={url('/buckets/' + bucket.id)}
    >
      <Database size={24} class="mb-4 text-muted-foreground" />
      <h2 class="truncate font-semibold">
        {bucket.globalAliases?.[0] || bucket.id}
      </h2>
      <p class="muted mt-1 truncate">{bucket.id}</p>
      <div class="mt-5 flex justify-between text-sm">
        <span>{bucket.objects.toLocaleString()} objects</span>
        <span>{readableBytes(bucket.bytes)}</span>
      </div>
      {#if bucket.websiteAccess}<span class="badge mt-3">
          Website enabled
        </span>{/if}
    </a>{:else}{#if !buckets.loading}<p class="muted">
        No buckets found.
      </p>{/if}{/each}
</div>
<FormDialog
  bind:open={create}
  title="Create bucket"
  fields={[{ name: 'globalAlias', label: 'Bucket name', required: true }]}
  submit={async (values) => {
    await api.post('/v2/CreateBucket', { body: values });
    buckets.reload();
  }}
  submitLabel="Create"
/>
