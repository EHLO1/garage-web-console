<script lang="ts">
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import { readableBytes } from '#lib/utils.ts';
  import type {
    GetStatusResult,
    GetClusterLayoutResult,
    Node,
    GetNodeInfoResult
  } from '#lib/types/cluster.ts';
  import Button from '#lib/components/Button.svelte';
  import FormDialog from '#lib/components/FormDialog.svelte';
  import Modal from '#lib/components/Modal.svelte';
  import RequestState from '#lib/components/RequestState.svelte';
  const data = resource(async () => {
    const [status, layout, info] = await Promise.all([
      api.get<GetStatusResult>('/v2/GetClusterStatus'),
      api.get<GetClusterLayoutResult>('/v2/GetClusterLayout'),
      api.get<GetNodeInfoResult>('/v2/GetNodeInfo', {
        params: { node: 'self' }
      })
    ]);
    return { status, layout, info: Object.values(info.success || {})[0] };
  });
  let connecting = $state(false);
  let assigning = $state(false);
  let selected = $state<Node | null>(null);
  let action = $state<'apply' | 'revert' | 'remove' | null>(null);
  let messages = $state<string[]>([]);
  let messagesOpen = $state(false);
  let role = $derived(
    data.data?.layout.stagedRoleChanges.find((r) => r.id === selected?.id) ||
      data.data?.layout.roles.find((r) => r.id === selected?.id) ||
      selected?.role
  );
  let initial = $derived({
    zone: role?.zone || '',
    capacity: role?.capacity ? role.capacity / 1024 ** 3 : 1,
    gateway: role?.capacity === null,
    tags: role?.tags?.join(', ') || ''
  });
</script>

<div class="toolbar">
  <div>
    <h1 class="page-title">Cluster</h1>
    <p class="muted mt-1">Nodes and storage layout</p>
  </div>
  <div class="flex gap-2">
    <Button variant="outline" onclick={data.reload}>Refresh</Button><Button
      onclick={() => (connecting = true)}
    >
      Connect node
    </Button>
  </div>
</div>
<RequestState loading={data.loading} error={data.error} retry={data.reload} />
{#if data.data}
  <section class="card mb-5">
    <div class="flex flex-wrap gap-5">
      <div>
        <p class="muted">Garage version</p>
        <p class="font-medium">
          {data.data.info?.garageVersion || data.data.status.garageVersion}
        </p>
      </div>
      <div>
        <p class="muted">Layout version</p>
        <p>{data.data.layout.version}</p>
      </div>
      <div>
        <p class="muted">Database</p>
        <p>{data.data.info?.dbEngine || data.data.status.dbEngine}</p>
      </div>
      <div>
        <p class="muted">Rust version</p>
        <p>{data.data.info?.rustVersion || data.data.status.rustVersion}</p>
      </div>
    </div>
  </section>
  {#if data.data.layout.stagedRoleChanges.length}<section
      class="card mb-5 border-amber-500"
    >
      <div class="toolbar">
        <div>
          <h2 class="font-semibold">Staged changes</h2>
          <p class="muted">Review changes before applying the layout.</p>
        </div>
        <div class="flex gap-2">
          <Button variant="outline" onclick={() => (action = 'revert')}>
            Revert changes
          </Button><Button onclick={() => (action = 'apply')}>
            Apply changes
          </Button>
        </div>
      </div>
      {#each data.data.layout.stagedRoleChanges as staged (staged.id)}<p
          class="mb-2 text-sm"
        >
          <code>{staged.id}</code>
          · {staged.remove
            ? 'Remove node'
            : `${staged.zone} · ${staged.capacity == null ? 'Gateway' : readableBytes(staged.capacity)} · ${staged.tags?.join(', ') || ''}`}
        </p>{/each}
    </section>{/if}
  <div class="grid gap-4 lg:grid-cols-2">
    {#each data.data.status.nodes || data.data.status.knownNodes || [] as node (node.id)}{@const nodeRole =
        data.data.layout.roles.find((r) => r.id === node.id) || node.role}
      <section class="card">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="font-semibold">{node.hostname || node.addr}</h2>
          <span
            class="badge {node.isUp ? 'text-green-600' : 'text-destructive'}"
          >
            {node.isUp ? 'Connected' : 'Offline'}
          </span>
        </div>
        <code>{node.id}</code>
        <p class="muted mt-3">{node.addr}</p>
        <div class="my-4 grid grid-cols-2 gap-3 text-sm">
          <p>Zone: {nodeRole?.zone || 'Unassigned'}</p>
          <p>
            Capacity: {nodeRole?.capacity === null
              ? 'Gateway'
              : readableBytes(nodeRole?.capacity)}
          </p>
          <p>Tags: {nodeRole?.tags?.join(', ') || '—'}</p>
          <p>{node.draining ? 'Draining' : ''}</p>
        </div>
        {#if node.dataPartition}<p class="muted mb-3">
            Data: {readableBytes(node.dataPartition.available)} free of {readableBytes(
              node.dataPartition.total
            )}
          </p>{/if}{#if node.metadataPartition}<p class="muted mb-3">
            Metadata: {readableBytes(node.metadataPartition.available)} free of {readableBytes(
              node.metadataPartition.total
            )}
          </p>{/if}
        <div class="flex gap-2">
          <Button
            variant="outline"
            onclick={() => {
              selected = node;
              assigning = true;
            }}
          >
            {nodeRole ? 'Edit assignment' : 'Assign node'}
          </Button>{#if nodeRole}<Button
              variant="ghost"
              onclick={() => {
                selected = node;
                action = 'remove';
              }}
            >
              Unassign
            </Button>{/if}
        </div>
      </section>{/each}
  </div>
{/if}
<FormDialog
  bind:open={connecting}
  title="Connect node"
  fields={[
    {
      name: 'node',
      label: 'Node ID and address',
      hint: 'Node ID@host:port',
      required: true
    }
  ]}
  submit={async (values) => {
    const result = await api.post('/v2/ConnectClusterNodes', {
      body: [values.node]
    });
    if (!result[0]?.success)
      throw new Error(result[0]?.error || 'Connection failed');
    data.reload();
  }}
/>
<FormDialog
  bind:open={assigning}
  title="Assign node"
  {initial}
  fields={[
    { name: 'zone', label: 'Zone', required: true },
    { name: 'gateway', label: 'Gateway (no storage)', type: 'checkbox' },
    {
      name: 'capacity',
      label: 'Storage capacity (GiB)',
      type: 'number',
      min: 0
    },
    { name: 'tags', label: 'Tags (comma separated)' }
  ]}
  submit={async (values) => {
    if (!values.gateway && !(Number(values.capacity) > 0))
      throw new Error('Storage capacity must be greater than zero');
    await api.post('/v2/UpdateClusterLayout', {
      body: {
        parameters: null,
        roles: [
          {
            id: selected?.id,
            zone: values.zone,
            capacity: values.gateway
              ? null
              : Math.round(Number(values.capacity) * 1024 ** 3),
            tags: String(values.tags || '')
              .split(',')
              .map((t) => t.trim())
              .filter(Boolean)
          }
        ]
      }
    });
    data.reload();
  }}
/>
<FormDialog
  bind:open={
    () => action !== null,
    (value) => {
      if (!value) action = null;
    }
  }
  title={action === 'apply'
    ? 'Apply layout'
    : action === 'revert'
      ? 'Revert staged changes'
      : 'Unassign node'}
  description={action === 'apply'
    ? 'Apply the staged changes to your cluster?'
    : action === 'revert'
      ? 'Discard all staged layout changes?'
      : 'Stage removal of this node from the layout?'}
  fields={[]}
  submitLabel="Confirm"
  submit={async () => {
    if (action === 'remove')
      await api.post('/v2/UpdateClusterLayout', {
        body: { parameters: null, roles: [{ id: selected?.id, remove: true }] }
      });
    else if (action === 'revert') await api.post('/v2/RevertClusterLayout');
    else {
      const result = await api.post('/v2/ApplyClusterLayout', {
        body: { version: (data.data?.layout.version || 0) + 1 }
      });
      messages = result.message || [];
      messagesOpen = true;
    }
    action = null;
    data.reload();
  }}
/>
<Modal bind:open={messagesOpen} title="Layout applied">
  <pre class="whitespace-pre-wrap text-sm">{messages.join('\n')}</pre>
</Modal>
