<script lang="ts">
  import type { Bucket, Key } from '#lib/types/buckets.ts';
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import { handleError } from '#lib/utils.ts';
  import Button from './Button.svelte';
  import FormDialog from './FormDialog.svelte';
  import RequestState from './RequestState.svelte';
  let { bucket, reload }: { bucket: Bucket; reload: () => void } = $props();
  const keys = resource(() =>
    api.get<{ id: string; name: string }[]>('/v2/ListKeys')
  );
  let open = $state(false);
  let revoke = $state<Key | null>(null);
  let saving = $state(false);
  async function togglePermission(
    key: Key,
    permission: 'read' | 'write' | 'owner',
    checked: boolean
  ) {
    saving = true;
    try {
      await api.post(checked ? '/v2/AllowBucketKey' : '/v2/DenyBucketKey', {
        body: {
          bucketId: bucket.id,
          accessKeyId: key.accessKeyId,
          permissions: {
            read: false,
            write: false,
            owner: false,
            [permission]: true
          }
        }
      });
    } catch (error) {
      handleError(error);
    } finally {
      saving = false;
      reload();
    }
  }
  let fields = $derived([
    {
      name: 'keys',
      label: 'Access keys',
      type: 'multi' as const,
      options: (keys.data || []).map((k) => ({
        value: k.id,
        label: `${k.name || k.id} (${k.id})`
      }))
    },
    ...['read', 'write', 'owner'].map((name) => ({
      name,
      label: name,
      type: 'checkbox' as const
    }))
  ]);
</script>

<section class="card">
  <div class="toolbar">
    <h2 class="font-semibold">Bucket permissions</h2>
    <Button onclick={() => (open = true)}>Allow access key</Button>
  </div>
  <RequestState loading={keys.loading} error={keys.error} retry={keys.reload} />
  <div class="overflow-auto">
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>Key ID</th>
          <th>Read</th>
          <th>Write</th>
          <th>Owner</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each bucket.keys || [] as key (key.accessKeyId)}<tr>
            <td>{key.name}</td>
            <td><code>{key.accessKeyId}</code></td>
            {#each ['read', 'write', 'owner'] as permission (permission)}
              <td>
                <input
                  type="checkbox"
                  aria-label={`${permission} permission for ${key.name}`}
                  checked={key.permissions[
                    permission as 'read' | 'write' | 'owner'
                  ]}
                  disabled={saving || keys.loading}
                  onchange={(event) =>
                    togglePermission(
                      key,
                      permission as 'read' | 'write' | 'owner',
                      event.currentTarget.checked
                    )}
                />
              </td>
            {/each}
            <td>
              <Button variant="ghost" onclick={() => (revoke = key)}>
                Revoke
              </Button>
            </td>
          </tr>{:else}<tr>
            <td colspan="6" class="muted py-8 text-center">
              No keys have access.
            </td>
          </tr>{/each}
      </tbody>
    </table>
  </div>
</section>
<FormDialog
  bind:open
  title="Allow access keys"
  {fields}
  initial={{ keys: [], read: true, write: true, owner: false }}
  submit={async (values) => {
    if (!values.keys?.length) throw new Error('Select at least one key');
    for (const accessKeyId of values.keys)
      await api.post('/v2/AllowBucketKey', {
        body: {
          bucketId: bucket.id,
          accessKeyId,
          permissions: {
            read: !!values.read,
            write: !!values.write,
            owner: !!values.owner
          }
        }
      });
    reload();
  }}
/>
<FormDialog
  bind:open={
    () => revoke !== null,
    (value) => {
      if (!value) revoke = null;
    }
  }
  title="Revoke key access"
  description={`Remove all permissions for ${revoke?.name || revoke?.accessKeyId}?`}
  fields={[]}
  submitLabel="Revoke"
  submit={async () => {
    await api.post('/v2/DenyBucketKey', {
      body: {
        bucketId: bucket.id,
        accessKeyId: revoke?.accessKeyId,
        permissions: { read: true, write: true, owner: true }
      }
    });
    revoke = null;
    reload();
  }}
/>
