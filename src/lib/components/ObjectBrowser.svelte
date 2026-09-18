<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { Folder, File, Upload, Download } from '@lucide/svelte';
  import { toast } from 'svelte-sonner';
  import api, { API_URL } from '#lib/api.ts';
  import { auth } from '#lib/auth.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import { readableBytes, handleError, copyToClipboard } from '#lib/utils.ts';
  import {
    objectPath,
    breadcrumbs,
    folderName,
    encodeObjectKey
  } from '#lib/objects.ts';
  import { readDataTransferItems } from '#lib/file-drop.ts';
  import {
    uploadStore,
    setUploadOnComplete
  } from '#src/stores/upload-store.ts';
  import type { Bucket } from '#lib/types/buckets.ts';
  import type { GetObjectsResult } from '#lib/types/objects.ts';
  import type { Config } from '#lib/types/garage.ts';
  import Button from './Button.svelte';
  import Modal from './Modal.svelte';
  import FormDialog from './FormDialog.svelte';
  import RequestState from './RequestState.svelte';
  let { bucket, reloadBucket }: { bucket: Bucket; reloadBucket: () => void } =
    $props();
  let bucketName = $derived(bucket.globalAliases[0]);
  let prefix = $derived(page.url.searchParams.get('prefix') || '');
  let next = $derived(page.url.searchParams.get('next') || '');
  const objects = resource(() =>
    api.get<GetObjectsResult>(`/browse/${encodeURIComponent(bucketName)}`, {
      params: { prefix, next, limit: 1000 }
    })
  );
  const config = resource(() =>
    $auth?.user?.role !== 'admin'
      ? Promise.resolve<Config | null>(null)
      : api.get<Config>('/config')
  );
  let writable = $derived(
    $auth?.user?.role === 'admin' || $auth?.user?.role === 'user'
  );
  let selected = $state<string[]>([]);
  let deleting = $state<string[]>([]);
  let createFolder = $state(false);
  let dragDepth = $state(0);
  let fileInput = $state<HTMLInputElement>();
  let folderInput = $state<HTMLInputElement>();
  let sharing = $state(false);
  let shareKeys = $state<string[]>([]);
  let domain = $state('');
  let protocol = $state('https');
  let moving = $state(false);
  let moveKeys = $state<string[]>([]);
  let destination = $state('');
  let destinationNext = $state('');
  let newFolder = $state('');
  let moveBusy = $state(false);
  let moveError = $state('');
  const folders = resource(() =>
    moving
      ? api.get<GetObjectsResult>(`/browse/${encodeURIComponent(bucketName)}`, {
          params: { prefix: destination, next: destinationNext, limit: 1000 }
        })
      : Promise.resolve<GetObjectsResult>({
          prefixes: [],
          objects: [],
          prefix: '',
          nextToken: null
        })
  );
  let allKeys = $derived([
    ...(objects.data?.prefixes || []),
    ...(objects.data?.objects || []).map(
      (object) => (objects.data?.prefix || '') + object.objectKey
    )
  ]);
  let domains = $derived([
    ...new Set([
      bucketName,
      ...(config.data?.s3_web?.root_domain
        ? [
            bucketName + config.data.s3_web.root_domain,
            bucketName +
              config.data.s3_web.root_domain +
              ':' +
              (config.data.s3_web.bind_addr?.split(':').pop() || '80')
          ]
        : [])
    ])
  ]);
  let shareUrls = $derived(
    shareKeys.map((key) => `${protocol}://${domain}/${encodeObjectKey(key)}`)
  );
  $effect(() => {
    void prefix;
    void next;
    selected = [];
  });
  onMount(() =>
    setUploadOnComplete((name) => {
      if (name === bucketName) {
        objects.reload();
        reloadBucket();
      }
    })
  );
  function navigate(value: string, token = '') {
    const target = new URL(page.url.href);
    target.searchParams.set('prefix', value);
    target.searchParams.delete('next');
    if (token) target.searchParams.set('next', token);
    void goto(target, { reset: false });
  }
  function toggle(key: string) {
    selected = selected.includes(key)
      ? selected.filter((item) => item !== key)
      : [...selected, key];
  }
  function uploadFiles(event: Event) {
    if (!writable) return;
    const input = event.currentTarget as HTMLInputElement;
    uploadStore.enqueue(
      Array.from(input.files || []).map((file) => ({
        bucket: bucketName,
        key: prefix + (file.webkitRelativePath || file.name),
        file
      }))
    );
    input.value = '';
  }
  async function drop(event: DragEvent) {
    event.preventDefault();
    dragDepth = 0;
    if (!writable) return;
    const targetBucket = bucketName;
    const targetPrefix = prefix;
    try {
      if (event.dataTransfer) {
        const items = await readDataTransferItems(event.dataTransfer);
        uploadStore.enqueue(
          items.map((item) => ({
            bucket: targetBucket,
            key: targetPrefix + item.path,
            file: item.file
          }))
        );
      }
    } catch (e) {
      handleError(e);
    }
  }
  function share(keys: string[]) {
    shareKeys = keys.filter((key) => !key.endsWith('/'));
    domain = domains[0];
    sharing = true;
  }
  function move(keys: string[]) {
    moveKeys = [...keys];
    destination = prefix;
    destinationNext = '';
    moveError = '';
    newFolder = '';
    moving = true;
  }
  function navigateDestination(value: string) {
    destination = value;
    destinationNext = '';
  }
  async function makeDestinationFolder() {
    moveBusy = true;
    moveError = '';
    try {
      const path = destination + folderName(newFolder);
      await api.put(objectPath(bucketName, path), { body: new FormData() });
      navigateDestination(path);
      newFolder = '';
      folders.reload();
      objects.reload();
    } catch (e) {
      moveError = (e as Error).message;
    } finally {
      moveBusy = false;
    }
  }
  async function moveObjects() {
    moveBusy = true;
    moveError = '';
    try {
      await api.post(`/browse/${encodeURIComponent(bucketName)}`, {
        body: { items: moveKeys, destination }
      });
      moving = false;
      selected = [];
      objects.reload();
      reloadBucket();
      toast.success('Objects moved');
    } catch (e) {
      moveError = (e as Error).message;
      objects.reload();
    } finally {
      moveBusy = false;
    }
  }
</script>

<section
  aria-label="Object browser"
  class="card relative"
  ondragenter={(event) => {
    if (writable && event.dataTransfer?.types.includes('Files')) {
      event.preventDefault();
      dragDepth++;
    }
  }}
  ondragover={(event) => event.preventDefault()}
  ondragleave={() => (dragDepth = Math.max(0, dragDepth - 1))}
  ondrop={drop}
>
  <div class="toolbar">
    <nav
      aria-label="Object folders"
      class="flex flex-wrap items-center gap-2 text-sm"
    >
      <button class="hover:underline" onclick={() => navigate('')}>Root</button>
      {#each breadcrumbs(prefix) as crumb (crumb.prefix)}<span
          class="text-muted-foreground"
        >
          /
        </span>
        <button class="hover:underline" onclick={() => navigate(crumb.prefix)}>
          {crumb.label}
        </button>{/each}
    </nav>
    <div class="flex flex-wrap gap-2">
      <Button
        variant="outline"
        onclick={() => {
          objects.reload();
          reloadBucket();
        }}
      >
        Refresh
      </Button>{#if writable}<Button
          variant="outline"
          onclick={() => (createFolder = true)}
        >
          New folder
        </Button><Button variant="outline" onclick={() => folderInput?.click()}>
          Upload folder
        </Button><Button onclick={() => fileInput?.click()}>
          <Upload size={16} />Upload files
        </Button>{/if}
    </div>
  </div>
  {#if writable}<input
      bind:this={fileInput}
      class="hidden"
      type="file"
      multiple
      onchange={uploadFiles}
      aria-label="Upload files"
    />
    <input
      bind:this={folderInput}
      class="hidden"
      type="file"
      multiple
      webkitdirectory
      onchange={uploadFiles}
      aria-label="Upload folder"
    />{/if}
  {#if selected.length}<div
      class="mb-4 flex flex-wrap items-center gap-2 rounded-md bg-muted p-3"
    >
      <span class="mr-auto text-sm">{selected.length} selected</span>
      <Button variant="outline" onclick={() => (selected = [])}>
        Clear
      </Button><Button
        variant="outline"
        onclick={() => share(selected)}
        disabled={!selected.some((key) => !key.endsWith('/'))}
      >
        Share files
      </Button>{#if writable}<Button
          variant="outline"
          onclick={() => move(selected)}
        >
          Move
        </Button><Button
          variant="destructive"
          onclick={() => (deleting = [...selected])}
        >
          Delete
        </Button>
      {/if}
    </div>{/if}
  <RequestState
    loading={objects.loading}
    error={objects.error}
    retry={objects.reload}
  />
  <div class="overflow-auto">
    <table>
      <thead>
        <tr>
          <th>
            <input
              type="checkbox"
              aria-label="Select all objects"
              checked={allKeys.length > 0 &&
                allKeys.every((key) => selected.includes(key))}
              onchange={(event) =>
                (selected = event.currentTarget.checked ? [...allKeys] : [])}
              disabled={objects.loading}
            />
          </th>
          <th>Name</th>
          <th>Size</th>
          <th>Last modified</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#if !objects.loading && !objects.error}
          {#each objects.data?.prefixes || [] as folder (folder)}<tr>
              <td>
                <input
                  type="checkbox"
                  aria-label={`Select ${folder}`}
                  checked={selected.includes(folder)}
                  onchange={() => toggle(folder)}
                />
              </td>
              <td>
                <button
                  class="flex items-center gap-2 hover:underline"
                  onclick={() => navigate(folder)}
                >
                  <Folder size={18} />{folder
                    .slice(prefix.length)
                    .replace(/\/$/, '')}
                </button>
              </td>
              <td>—</td>
              <td>—</td>
              <td>
                <div class="flex gap-1">
                  {#if writable}<Button
                      variant="ghost"
                      onclick={() => move([folder])}
                    >
                      Move
                    </Button><Button
                      variant="ghost"
                      onclick={() => (deleting = [folder])}
                    >
                      Delete
                    </Button>
                  {/if}
                </div>
              </td>
            </tr>{/each}
          {#each objects.data?.objects || [] as object (object.objectKey)}{@const key =
              (objects.data?.prefix || '') + object.objectKey}{@const path =
              API_URL + objectPath(bucketName, key)}
            <tr>
              <td>
                <input
                  type="checkbox"
                  aria-label={`Select ${object.objectKey}`}
                  checked={selected.includes(key)}
                  onchange={() => toggle(key)}
                />
              </td>
              <td>
                <a
                  class="flex items-center gap-2 hover:underline"
                  href={path + '?view=1'}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {#if /\.(jpe?g|png|gif)$/i.test(object.objectKey)}<img
                      src={path + '?thumb=1'}
                      alt=""
                      class="h-6 w-6 object-cover"
                      loading="lazy"
                    />{:else}<File size={18} />{/if}
                  <span class="max-w-sm truncate">{object.objectKey}</span>
                </a>
              </td>
              <td class="whitespace-nowrap">{readableBytes(object.size)}</td>
              <td class="whitespace-nowrap">
                {new Date(object.lastModified).toLocaleString()}
              </td>
              <td>
                <div class="flex items-center gap-1">
                  <a
                    class="rounded p-2 hover:bg-accent"
                    aria-label={`Download ${object.objectKey}`}
                    href={path + '?dl=1'}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Download size={16} />
                  </a>
                  <Button variant="ghost" onclick={() => share([key])}>
                    Share
                  </Button>{#if writable}<Button
                      variant="ghost"
                      onclick={() => move([key])}
                    >
                      Move
                    </Button><Button
                      variant="ghost"
                      onclick={() => (deleting = [key])}
                    >
                      Delete
                    </Button>{/if}
                </div>
              </td>
            </tr>{/each}
          {#if !allKeys.length}<tr>
              <td colspan="5" class="py-16 text-center text-muted-foreground">
                {writable
                  ? 'This folder is empty. Upload files or drop them here.'
                  : 'This folder is empty.'}
              </td>
            </tr>{/if}
        {/if}
      </tbody>
    </table>
  </div>
  <div class="mt-4 flex justify-end gap-2">
    {#if next}<Button variant="outline" onclick={() => navigate(prefix)}>
        First page
      </Button>{/if}{#if objects.data?.nextToken}<Button
        variant="outline"
        disabled={objects.loading}
        onclick={() => navigate(prefix, objects.data?.nextToken || '')}
      >
        Next page
      </Button>{/if}
  </div>
  {#if dragDepth}<div
      class="pointer-events-none absolute inset-0 flex items-center justify-center rounded-xl border-2 border-dashed border-primary bg-background/90"
    >
      <div class="text-center">
        <Upload class="mx-auto mb-2" size={36} />
        <p class="font-medium">Drop files or folders to upload</p>
        <p class="muted">/{prefix}</p>
      </div>
    </div>{/if}
</section>
<FormDialog
  bind:open={createFolder}
  title="Create folder"
  fields={[{ name: 'name', label: 'Folder path', required: true }]}
  submit={async (values) => {
    await api.put(objectPath(bucketName, prefix + folderName(values.name)), {
      body: new FormData()
    });
    objects.reload();
    reloadBucket();
  }}
/>
<FormDialog
  bind:open={
    () => deleting.length > 0,
    (value) => {
      if (!value) deleting = [];
    }
  }
  title="Delete objects"
  description={`Permanently delete ${deleting.length} selected item(s)? Folders and their contents will be deleted.`}
  fields={[]}
  submitLabel="Delete permanently"
  submit={async () => {
    const failed: string[] = [];
    const errors: string[] = [];
    for (const key of deleting) {
      try {
        await api.delete(objectPath(bucketName, key), {
          params: { recursive: key.endsWith('/') }
        });
      } catch (e) {
        failed.push(key);
        errors.push((e as Error).message);
      }
    }
    selected = failed;
    deleting = failed;
    objects.reload();
    reloadBucket();
    if (failed.length)
      throw new Error(
        `${failed.length} item(s) could not be deleted: ${errors.join('; ')}`
      );
  }}
/>
<Modal
  bind:open={sharing}
  title="Share files"
  description="Links use the bucket’s public website endpoint."
>
  {#if !bucket.websiteAccess}<p
      role="alert"
      class="mb-4 rounded border border-amber-500 p-3 text-sm"
    >
      Website access is disabled. These links will only work once an
      administrator enables it.
    </p>{/if}
  <div class="mb-4 flex gap-2">
    <label class="text-sm">
      Protocol
      <select class="input mt-1" bind:value={protocol}>
        <option value="https">HTTPS</option>
        <option value="http">HTTP</option>
      </select>
    </label>
    <label class="flex-1 text-sm">
      Domain
      <input class="input mt-1" list="share-domains" bind:value={domain} />
      <datalist id="share-domains">
        {#each domains as item (item)}<option value={item}></option>{/each}
      </datalist>
    </label>
  </div>
  <div class="max-h-64 space-y-2 overflow-auto">
    {#each shareUrls as shareUrl (shareUrl)}<div class="flex gap-2">
        <input class="input" aria-label="Share URL" readonly value={shareUrl} />
        <Button variant="outline" onclick={() => copyToClipboard(shareUrl)}>
          Copy
        </Button>
      </div>{/each}
  </div>
  <div class="mt-5 flex justify-end">
    <Button onclick={() => copyToClipboard(shareUrls.join('\n'))}>
      Copy all links
    </Button>
  </div>
</Modal>
<Modal
  bind:open={moving}
  title={`Move ${moveKeys.length} item(s)`}
  description="Choose a destination folder. Existing objects with the same names may be overwritten."
>
  <nav
    aria-label="Destination folders"
    class="mb-4 flex flex-wrap gap-2 text-sm"
  >
    <button onclick={() => navigateDestination('')}>Root</button>
    {#each breadcrumbs(destination) as crumb (crumb.prefix)}<span>/</span>
      <button onclick={() => navigateDestination(crumb.prefix)}>
        {crumb.label}
      </button>{/each}
  </nav>
  <RequestState
    loading={folders.loading}
    error={folders.error}
    retry={folders.reload}
  />
  <div class="max-h-48 overflow-auto rounded border">
    {#each folders.data?.prefixes || [] as folder (folder)}<button
        class="flex w-full items-center gap-2 border-b p-3 text-left text-sm hover:bg-accent"
        disabled={moveKeys.some(
          (key) => key.endsWith('/') && folder.startsWith(key)
        )}
        onclick={() => navigateDestination(folder)}
      >
        <Folder size={16} />{folder
          .slice(destination.length)
          .replace(/\/$/, '')}
      </button>{:else}<p class="muted p-4">No subfolders.</p>{/each}
  </div>
  {#if folders.data?.nextToken}<Button
      variant="ghost"
      onclick={() => (destinationNext = folders.data?.nextToken || '')}
    >
      More folders
    </Button>{/if}
  <form
    class="mt-4 flex gap-2"
    onsubmit={(event) => {
      event.preventDefault();
      void makeDestinationFolder();
    }}
  >
    <input
      class="input"
      aria-label="New destination folder"
      placeholder="New folder or nested/path"
      bind:value={newFolder}
      required
    />
    <Button type="submit" variant="outline" disabled={moveBusy}>Create</Button>
  </form>
  <p class="muted mt-4">Destination: /{destination}</p>
  {#if moveError}<p role="alert" class="mt-3 text-sm text-destructive">
      {moveError}
    </p>{/if}
  <div class="mt-5 flex justify-end gap-2">
    <Button
      variant="outline"
      disabled={moveBusy}
      onclick={() => (moving = false)}
    >
      Cancel
    </Button><Button
      disabled={moveBusy ||
        destination === prefix ||
        moveKeys.some(
          (key) => key.endsWith('/') && destination.startsWith(key)
        )}
      onclick={moveObjects}
    >
      {moveBusy ? 'Moving…' : 'Move here'}
    </Button>
  </div>
</Modal>
