<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import { auth } from '#lib/auth.ts';
  import { url } from '#lib/utils.ts';
  import type { Bucket } from '#lib/types/buckets.ts';
  import RequestState from '#lib/components/RequestState.svelte';
  import BucketOverview from '#lib/components/BucketOverview.svelte';
  import BucketPermissions from '#lib/components/BucketPermissions.svelte';
  import ObjectBrowser from '#lib/components/ObjectBrowser.svelte';
  let id = $derived(page.params.id || '');
  const bucket = resource(() =>
    api.get<Bucket>('/v2/GetBucketInfo', { params: { id } })
  );
  let manager = $derived(
    $auth?.user?.role === 'owner' || $auth?.user?.role === 'admin'
  );
  let tab = $derived(page.url.searchParams.get('tab') || 'browse');
  function selectTab(value: string) {
    const next = new URL(page.url.href);
    next.searchParams.set('tab', value);
    void goto(next, { reset: false });
  }
</script>

<a class="muted hover:underline" href={url('/buckets')}>← Buckets</a>
<h1 class="page-title mb-2 mt-4 break-all">
  {bucket.data?.globalAliases?.[0] || id}
</h1>
<p class="muted mb-6 break-all">{id}</p>
<RequestState
  loading={bucket.loading}
  error={bucket.error}
  retry={bucket.reload}
/>
{#if bucket.data && bucket.data.id === id}
  <nav
    aria-label="Bucket sections"
    class="mb-5 flex gap-1 rounded-lg bg-muted p-1 w-fit"
  >
    {#each ['browse', 'overview', ...(manager ? ['permissions'] : [])] as value (value)}<button
        onclick={() => selectTab(value)}
        aria-current={tab === value ? 'page' : undefined}
        class="rounded-md px-4 py-2 text-sm font-medium capitalize {tab ===
        value
          ? 'bg-background shadow-sm'
          : 'text-muted-foreground hover:text-foreground'}"
      >
        {value}
      </button>{/each}
  </nav>
  {#key id}
    {#if tab === 'overview'}<BucketOverview
        bucket={bucket.data}
        {manager}
        reload={bucket.reload}
      />
    {:else if tab === 'permissions' && manager}<BucketPermissions
        bucket={bucket.data}
        reload={bucket.reload}
      />
    {:else if bucket.data.globalAliases?.length && bucket.data.keys?.some((key) => key.permissions.read && key.permissions.write)}<ObjectBrowser
        bucket={bucket.data}
        reloadBucket={bucket.reload}
      />
    {:else}<section class="card">
        <p>
          A global alias and an access key with read and write permissions are
          required to browse this bucket.
        </p>
      </section>{/if}
  {/key}
{/if}
