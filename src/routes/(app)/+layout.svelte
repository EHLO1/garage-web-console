<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { auth, refreshAuth } from '#lib/auth.ts';
  import api from '#lib/api.ts';
  import { url, handleError } from '#lib/utils.ts';
  import {
    LayoutDashboard,
    Network,
    Database,
    KeyRound,
    Users,
    ScrollText,
    SunMoon,
    Menu
  } from '@lucide/svelte';
  import logo from '#src/assets/garage-logo.svg';
  import Button from '#lib/components/Button.svelte';
  import FormDialog from '#lib/components/FormDialog.svelte';
  import UploadPanel from '#lib/components/UploadPanel.svelte';
  import { uploadStore } from '#src/stores/upload-store.ts';
  let { children }: { children: Snippet } = $props();
  let ready = $state(false);
  let error = $state('');
  let menu = $state(false);
  let passwordOpen = $state(false);
  const passwordFields = [
    {
      name: 'currentPassword',
      label: 'Current password',
      type: 'password' as const,
      required: true
    },
    {
      name: 'newPassword',
      label: 'New password',
      type: 'password' as const,
      required: true
    },
    {
      name: 'confirmPassword',
      label: 'Confirm password',
      type: 'password' as const,
      required: true
    }
  ];
  let manager = $derived(
    $auth?.user?.role === 'owner' || $auth?.user?.role === 'admin'
  );
  let links = $derived([
    { path: '/', label: 'Dashboard', icon: LayoutDashboard },
    ...(manager ? [{ path: '/cluster', label: 'Cluster', icon: Network }] : []),
    { path: '/buckets', label: 'Buckets', icon: Database },
    { path: '/keys', label: 'Access Keys', icon: KeyRound },
    ...(manager
      ? [
          { path: '/users', label: 'Users', icon: Users },
          { path: '/logs', label: 'Logs', icon: ScrollText }
        ]
      : [])
  ]);
  async function load() {
    error = '';
    try {
      const status = await refreshAuth();
      if (status.needsSetup) await goto(url('/auth/register'));
      else if (status.enabled && !status.authenticated)
        await goto(url('/auth/login'));
      else ready = true;
    } catch (e) {
      error = (e as Error).message;
    }
  }
  onMount(load);
  $effect(() => {
    if (
      ready &&
      !manager &&
      ['/cluster', '/users', '/logs'].some(
        (path) => page.url.pathname === url(path)
      )
    )
      void goto(url('/buckets'), { replace: true });
    void page.url.pathname;
    menu = false;
  });
  async function logout() {
    try {
      await api.post('/auth/logout');
      uploadStore.clearAll();
      auth.set(null);
      await goto(url('/auth/login'));
    } catch (e) {
      handleError(e);
    }
  }
  function toggleTheme() {
    const dark = document.documentElement.classList.toggle('dark');
    try {
      localStorage.setItem('theme', dark ? 'dark' : 'light');
    } catch {
      /* Storage may be disabled. */
    }
  }
</script>

{#if error}<main class="p-8">
    <p role="alert">{error}</p>
    <Button onclick={load}>Retry</Button>
  </main>
{:else if !ready}<p role="status" class="p-8">Loading your workspace…</p>
{:else}
  <div class="flex h-dvh">
    <aside
      class="fixed inset-y-0 left-0 z-30 w-60 flex-col border-r bg-card p-4 md:static md:flex {menu
        ? 'flex shadow-xl'
        : 'hidden'}"
    >
      <a
        href={url('/')}
        class="mb-8 flex items-center gap-3 px-2 py-3 text-lg font-semibold"
      >
        <span
          class="garage-logo-tile flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
        >
          <img src={logo} alt="" class="h-8 w-8" />
        </span>
        Garage
      </a>
      <nav aria-label="Main navigation" class="space-y-1">
        {#each links as link (link.path)}<a
            href={url(link.path)}
            aria-current={page.url.pathname === url(link.path)
              ? 'page'
              : undefined}
            class="flex items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium hover:bg-accent {page
              .url.pathname === url(link.path) ||
            (link.path !== '/' &&
              page.url.pathname.startsWith(url(link.path) + '/'))
              ? 'bg-accent text-accent-foreground'
              : 'text-muted-foreground'}"
          >
            <link.icon size={18} />{link.label}
          </a>{/each}
      </nav>
      <div class="mt-auto space-y-2 border-t pt-4">
        <p class="truncate px-3 text-sm font-medium">
          {$auth?.user?.username || 'Garage'}
        </p>
        <p class="px-3 text-xs text-muted-foreground">
          {$auth?.user?.role || ''}
        </p>
        {#if $auth?.user && $auth.user.id !== 'legacy'}<Button
            variant="ghost"
            onclick={() => (passwordOpen = true)}
          >
            Change password
          </Button>{/if}
        <div class="flex justify-between">
          <Button variant="ghost" onclick={logout}>Sign out</Button><Button
            variant="ghost"
            onclick={toggleTheme}
            aria-label="Toggle color theme"
          >
            <SunMoon size={18} />
          </Button>
        </div>
      </div>
    </aside>
    <div class="min-w-0 flex-1 overflow-auto">
      <header class="flex items-center gap-3 border-b px-5 py-4 md:hidden">
        <Button
          variant="ghost"
          aria-label="Toggle navigation"
          onclick={() => (menu = !menu)}
        >
          <Menu size={20} />
        </Button>
        <span class="font-semibold">Garage Web Console</span>
      </header>
      <main class="mx-auto max-w-7xl p-5 md:p-8">
        {#if manager || !['/cluster', '/users', '/logs'].some((path) => page.url.pathname === url(path))}{@render children()}{/if}
      </main>
    </div>
  </div>
  <UploadPanel />
  <FormDialog
    bind:open={passwordOpen}
    title="Change password"
    fields={passwordFields}
    submit={async (values) => {
      if (values.newPassword.length < 6)
        throw new Error('Use at least 6 characters');
      if (values.newPassword !== values.confirmPassword)
        throw new Error('Passwords do not match');
      await api.post('/auth/change-password', {
        body: {
          currentPassword: values.currentPassword,
          newPassword: values.newPassword
        }
      });
    }}
  />
{/if}
