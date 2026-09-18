<script lang="ts">
  import type { Bucket } from '#lib/types/buckets.ts';
  import api from '#lib/api.ts';
  import { goto } from '$app/navigation';
  import { readableBytes, url, copyToClipboard } from '#lib/utils.ts';
  import Button from './Button.svelte';
  import FormDialog from './FormDialog.svelte';
  let {
    bucket,
    manager,
    reload
  }: { bucket: Bucket; manager: boolean; reload: () => void } = $props();
  let addAlias = $state(false);
  let removeAlias = $state<string | null>(null);
  let quota = $state(false);
  let website = $state(false);
  let deleting = $state(false);
  let quotaInitial = $derived({
    maxSize: bucket.quotas?.maxSize ? bucket.quotas.maxSize / 1024 ** 3 : 0,
    maxObjects: bucket.quotas?.maxObjects || 0
  });
  let websiteInitial = $derived({
    enabled: bucket.websiteAccess,
    indexDocument: bucket.websiteConfig?.indexDocument || 'index.html',
    errorDocument: bucket.websiteConfig?.errorDocument || 'error/400.html'
  });
</script>

<div class="grid gap-5 md:grid-cols-2">
  <section class="card">
    <h2 class="mb-4 font-semibold">Bucket information</h2>
    <button onclick={() => copyToClipboard(bucket.id)} title="Copy bucket ID">
      <code>{bucket.id}</code>
    </button>
    <dl class="mt-5 grid grid-cols-2 gap-3 text-sm">
      <dt class="text-muted-foreground">Objects</dt>
      <dd>{bucket.objects.toLocaleString()}</dd>
      <dt class="text-muted-foreground">Storage used</dt>
      <dd>{readableBytes(bucket.bytes)}</dd>
      <dt class="text-muted-foreground">Unfinished uploads</dt>
      <dd>{bucket.unfinishedUploads || 0}</dd>
      <dt class="text-muted-foreground">Multipart uploads</dt>
      <dd>{bucket.unfinishedMultipartUploads || 0}</dd>
      <dt class="text-muted-foreground">Multipart data</dt>
      <dd>{readableBytes(bucket.unfinishedMultipartUploadBytes || 0)}</dd>
    </dl>
  </section>
  <section class="card">
    <div class="toolbar">
      <h2 class="font-semibold">Aliases</h2>
      {#if manager}<Button variant="outline" onclick={() => (addAlias = true)}>
          Add alias
        </Button>{/if}
    </div>
    {#each bucket.globalAliases || [] as alias (alias)}<div
        class="flex items-center justify-between border-b py-2"
      >
        <span>{alias}</span>
        {#if manager}<Button
            variant="ghost"
            onclick={() => (removeAlias = alias)}
          >
            Remove
          </Button>{/if}
      </div>{:else}<p class="muted">
        No global aliases.
      </p>{/each}{#each bucket.localAliases || [] as alias (alias.accessKeyId + alias.alias)}<p
        class="muted mt-3"
      >
        {alias.alias} · key {alias.accessKeyId}
      </p>{/each}
  </section>
  <section class="card">
    <div class="toolbar">
      <h2 class="font-semibold">Quotas</h2>
      {#if manager}<Button variant="outline" onclick={() => (quota = true)}>
          Edit quotas
        </Button>{/if}
    </div>
    <p class="muted">Max objects: {bucket.quotas?.maxObjects || 'Unlimited'}</p>
    <p class="muted mt-2">
      Max size: {bucket.quotas?.maxSize
        ? readableBytes(bucket.quotas.maxSize)
        : 'Unlimited'}
    </p>
  </section>
  <section class="card">
    <div class="toolbar">
      <h2 class="font-semibold">Website access</h2>
      {#if manager}<Button variant="outline" onclick={() => (website = true)}>
          Configure
        </Button>{/if}
    </div>
    <span class="badge">{bucket.websiteAccess ? 'Enabled' : 'Disabled'}</span>
    {#if bucket.websiteAccess}<p class="muted mt-3">
        Index: {bucket.websiteConfig?.indexDocument}
      </p>
      <p class="muted mt-2">
        Error: {bucket.websiteConfig?.errorDocument}
      </p>{/if}
  </section>
</div>
{#if manager}<section class="card mt-5 border-destructive/40">
    <h2 class="font-semibold">Delete bucket</h2>
    <p class="muted my-3">
      The bucket must be empty. Deleting it permanently removes its
      configuration.
    </p>
    <Button variant="destructive" onclick={() => (deleting = true)}>
      Delete bucket
    </Button>
  </section>{/if}
<FormDialog
  bind:open={addAlias}
  title="Add global alias"
  fields={[{ name: 'alias', label: 'Alias', required: true }]}
  submit={async (values) => {
    await api.post('/v2/AddBucketAlias', {
      body: { bucketId: bucket.id, globalAlias: values.alias }
    });
    reload();
  }}
/>
<FormDialog
  bind:open={
    () => removeAlias !== null,
    (value) => {
      if (!value) removeAlias = null;
    }
  }
  title="Remove alias"
  description={`Remove ${removeAlias}? Clients using this name will lose access.`}
  fields={[]}
  submitLabel="Remove"
  submit={async () => {
    await api.post('/v2/RemoveBucketAlias', {
      body: { bucketId: bucket.id, globalAlias: removeAlias }
    });
    removeAlias = null;
    reload();
  }}
/>
<FormDialog
  bind:open={quota}
  title="Bucket quotas"
  description="Set a limit to zero for unlimited usage."
  initial={quotaInitial}
  fields={[
    {
      name: 'maxObjects',
      label: 'Maximum objects',
      type: 'number',
      min: 0,
      step: '1'
    },
    { name: 'maxSize', label: 'Maximum size (GiB)', type: 'number', min: 0 }
  ]}
  submit={async (values) => {
    await api.post('/v2/UpdateBucket', {
      params: { id: bucket.id },
      body: {
        quotas: {
          maxObjects: Number(values.maxObjects) || null,
          maxSize: Number(values.maxSize)
            ? Math.round(Number(values.maxSize) * 1024 ** 3)
            : null
        }
      }
    });
    reload();
  }}
/>
<FormDialog
  bind:open={website}
  title="Website access"
  initial={websiteInitial}
  fields={[
    { name: 'enabled', label: 'Enable public website', type: 'checkbox' },
    { name: 'indexDocument', label: 'Index document' },
    { name: 'errorDocument', label: 'Error document' }
  ]}
  submit={async (values) => {
    await api.post('/v2/UpdateBucket', {
      params: { id: bucket.id },
      body: { websiteAccess: values }
    });
    reload();
  }}
/>
<FormDialog
  bind:open={deleting}
  title="Delete bucket"
  description="This cannot be undone. Type the bucket ID to confirm."
  fields={[{ name: 'confirmation', label: bucket.id, required: true }]}
  submitLabel="Delete permanently"
  submit={async (values) => {
    if (values.confirmation !== bucket.id)
      throw new Error('Bucket ID does not match');
    await api.post('/v2/DeleteBucket', { params: { id: bucket.id } });
    await goto(url('/buckets'));
  }}
/>
