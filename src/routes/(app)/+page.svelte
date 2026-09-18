<script lang="ts">
  import { auth } from '#lib/auth.ts';
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import { readableBytes, url } from '#lib/utils.ts';
  import type { Bucket } from '#lib/types/buckets.ts';
  import type { GetHealthResult } from '#lib/types/health.ts';
  import type { User } from '#lib/types/users.ts';
  import RequestState from '#lib/components/RequestState.svelte';
  import Button from '#lib/components/Button.svelte';
  let manager = $derived($auth?.user?.role === 'admin');
  const dashboard = resource(async () => {
    const isManager = manager;
    const [buckets, health, users] = await Promise.all([
      api.get<Bucket[]>('/buckets'),
      isManager ? api.get<GetHealthResult>('/v2/GetClusterHealth') : null,
      isManager ? api.get<User[]>('/users') : []
    ]);
    return { buckets, health, users };
  });
  let buckets = $derived(dashboard.data?.buckets || []);
  let totalBytes = $derived(
    buckets.reduce((sum, bucket) => sum + bucket.bytes, 0)
  );
</script>

<div class="toolbar">
  <div>
    <h1 class="page-title">Dashboard</h1>
    <p class="muted mt-1">Your storage at a glance</p>
  </div>
  <Button variant="outline" onclick={dashboard.reload}>Refresh</Button>
</div>
<RequestState
  loading={dashboard.loading}
  error={dashboard.error}
  retry={dashboard.reload}
/>
{#if dashboard.data}
  <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
    <section class="card">
      <p class="muted">Buckets</p>
      <p class="mt-2 text-3xl font-semibold">{buckets.length}</p>
    </section>
    <section class="card">
      <p class="muted">Objects</p>
      <p class="mt-2 text-3xl font-semibold">
        {buckets
          .reduce((sum, bucket) => sum + bucket.objects, 0)
          .toLocaleString()}
      </p>
    </section>
    <section class="card">
      <p class="muted">Storage used</p>
      <p class="mt-2 text-3xl font-semibold">{readableBytes(totalBytes)}</p>
    </section>
    <section class="card">
      <p class="muted">{manager ? 'Cluster health' : 'Your role'}</p>
      <p class="mt-2 text-2xl font-semibold capitalize">
        {dashboard.data.health?.status || $auth?.user?.role}
      </p>
      {#if dashboard.data.health}<p class="muted mt-1">
          {dashboard.data.health.connectedNodes} / {dashboard.data.health
            .knownNodes} nodes connected
        </p>{/if}
    </section>
  </div>
  <section class="card mt-6">
    <h2 class="mb-4 font-semibold">Bucket usage</h2>
    {#each buckets as bucket (bucket.id)}<div class="mb-4">
        <div class="mb-1 flex justify-between gap-4 text-sm">
          <a
            class="font-medium hover:underline"
            href={url('/buckets/' + bucket.id)}
          >
            {bucket.globalAliases?.[0] || bucket.id}
          </a>
          <span class="muted">
            {readableBytes(bucket.bytes)} · {bucket.objects.toLocaleString()} objects
          </span>
        </div>
        <div class="h-2 rounded bg-muted">
          <div
            class="h-2 rounded bg-primary"
            style:width={totalBytes
              ? `${(bucket.bytes / totalBytes) * 100}%`
              : '0%'}
          ></div>
        </div>
      </div>{:else}<p class="muted">No buckets yet.</p>{/each}
  </section>
  {#if manager}<section class="card mt-6">
      <h2 class="mb-4 font-semibold">Users & roles</h2>
      <div class="mb-4 flex flex-wrap gap-3">
        {#each ['admin', 'user', 'viewer'] as role (role)}<span class="badge">
            {role}: {dashboard.data.users.filter((user) => user.role === role)
              .length}
          </span>{/each}
      </div>
      <div class="overflow-auto">
        <table>
          <thead>
            <tr>
              <th>Username</th>
              <th>Role</th>
              <th>Assigned buckets</th>
            </tr>
          </thead>
          <tbody>
            {#each dashboard.data.users as user (user.id)}<tr>
                <td>{user.username}</td>
                <td><span class="badge">{user.role}</span></td>
                <td>
                  {user.role !== 'admin'
                    ? user.buckets?.length || 0
                    : 'All buckets'}
                </td>
              </tr>{/each}
          </tbody>
        </table>
      </div>
    </section>{/if}
{/if}
