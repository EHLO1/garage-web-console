<script lang="ts">
  import { auth } from '#lib/auth.ts';
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import { copyToClipboard, handleError } from '#lib/utils.ts';
  import type { Bucket } from '#lib/types/buckets.ts';
  import Button from '#lib/components/Button.svelte';
  import FormDialog from '#lib/components/FormDialog.svelte';
  import RequestState from '#lib/components/RequestState.svelte';
  let manager = $derived(
    $auth?.user?.role === 'owner' || $auth?.user?.role === 'admin'
  );
  type KeyRow = { id: string; name: string; access?: string[] };
  const keys = resource(async (): Promise<KeyRow[]> => {
    if (manager) return api.get('/v2/ListKeys');
    const buckets = await api.get<Bucket[]>('/buckets');
    const result = new Map<string, KeyRow>();
    for (const bucket of buckets)
      for (const key of bucket.keys || []) {
        const row = result.get(key.accessKeyId) || {
          id: key.accessKeyId,
          name: key.name,
          access: []
        };
        row.access?.push(
          `${bucket.globalAliases?.[0] || bucket.id}: ${Object.entries(
            key.permissions
          )
            .filter(([, enabled]) => enabled)
            .map(([name]) => name)
            .join(', ')}`
        );
        result.set(row.id, row);
      }
    return [...result.values()];
  });
  let secrets = $state<Record<string, string>>({});
  let create = $state(false);
  let importing = $state(false);
  let remove = $state<KeyRow | null>(null);
  let search = $state('');
  let deleteOpen = $derived(remove !== null);
  async function reveal(id: string) {
    try {
      const result = await api.get('/v2/GetKeyInfo', {
        params: { id, showSecretKey: true }
      });
      if (!result.secretAccessKey) throw new Error('Secret key unavailable');
      secrets[id] = result.secretAccessKey;
    } catch (e) {
      handleError(e);
    }
  }
</script>

<div class="toolbar">
  <div>
    <h1 class="page-title">Access Keys</h1>
    <p class="muted mt-1">S3 credentials for your buckets</p>
  </div>
  {#if manager}<div class="flex gap-2">
      <Button
        variant="outline"
        onclick={() => {
          importing = true;
          create = true;
        }}
      >
        Import key
      </Button><Button
        onclick={() => {
          importing = false;
          create = true;
        }}
      >
        Create key
      </Button>
    </div>{/if}
</div>
<input
  class="input mb-5 max-w-sm"
  aria-label="Search keys"
  placeholder="Search keys…"
  bind:value={search}
/>
<RequestState loading={keys.loading} error={keys.error} retry={keys.reload} />
<section class="card overflow-auto">
  <table>
    <thead>
      <tr>
        <th>Name</th>
        <th>Key ID</th>
        <th>Secret key</th>
        {#if !manager}<th>Access</th>{/if}
        <th>Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each (keys.data || []).filter((key) => (key.name + key.id)
          .toLowerCase()
          .includes(search.toLowerCase())) as key (key.id)}<tr>
          <td>{key.name || '—'}</td>
          <td>
            <button title="Copy key ID" onclick={() => copyToClipboard(key.id)}>
              <code>{key.id}</code>
            </button>
          </td>
          <td>
            {#if secrets[key.id]}<div class="flex gap-2">
                <Button
                  variant="ghost"
                  onclick={() => copyToClipboard(secrets[key.id])}
                >
                  Copy secret
                </Button><Button
                  variant="ghost"
                  onclick={() => delete secrets[key.id]}
                >
                  Hide
                </Button>
              </div>
              <code>{secrets[key.id]}</code>{:else}<Button
                variant="outline"
                onclick={() => reveal(key.id)}
              >
                Reveal secret
              </Button>{/if}
          </td>
          {#if !manager}<td>{key.access?.join('; ')}</td>{/if}
          <td>
            {#if manager}<Button variant="ghost" onclick={() => (remove = key)}>
                Delete
              </Button>{/if}
          </td>
        </tr>{:else}<tr>
          <td colspan="5" class="muted py-8 text-center">No keys found.</td>
        </tr>{/each}
    </tbody>
  </table>
</section>
<FormDialog
  bind:open={create}
  title={importing ? 'Import key' : 'Create key'}
  fields={[
    { name: 'name', label: 'Name', required: true },
    ...(importing
      ? [
          { name: 'accessKeyId', label: 'Access key ID', required: true },
          {
            name: 'secretAccessKey',
            label: 'Secret access key',
            type: 'password' as const,
            required: true
          }
        ]
      : [])
  ]}
  submit={async (values) => {
    const result = await api.post(
      importing ? '/v2/ImportKey' : '/v2/CreateKey',
      { body: values }
    );
    if (result.secretAccessKey)
      secrets[result.accessKeyId] = result.secretAccessKey;
    keys.reload();
  }}
/>
<FormDialog
  title="Delete access key"
  description={`Permanently delete ${remove?.name || remove?.id}? Applications using this key will lose access.`}
  fields={[]}
  submitLabel="Delete key"
  submit={async () => {
    await api.post('/v2/DeleteKey', { params: { id: remove?.id } });
    if (remove) delete secrets[remove.id];
    remove = null;
    keys.reload();
  }}
  bind:open={
    () => deleteOpen,
    (value) => {
      if (!value) remove = null;
    }
  }
/>
