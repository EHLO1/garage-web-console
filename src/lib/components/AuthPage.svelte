<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { auth, refreshAuth } from '#lib/auth.ts';
  import api, { API_URL } from '#lib/api.ts';
  import { url } from '#lib/utils.ts';
  import Button from './Button.svelte';
  import AuthBackground from './AuthBackground.svelte';
  import { Eye, EyeOff, KeyRound, SunMoon } from '@lucide/svelte';
  import logo from '#src/assets/garage-logo.svg';
  let { register = false }: { register?: boolean } = $props();
  let username = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let showPassword = $state(false);
  let busy = $state(false);
  let error = $state('');
  onMount(async () => {
    error = page.url.searchParams.get('error') || '';
    try {
      const status = await refreshAuth();
      if (status.authenticated || (!status.enabled && !status.needsSetup))
        await goto(url('/'));
      else if (status.needsSetup !== register)
        await goto(url(status.needsSetup ? '/auth/register' : '/auth/login'));
    } catch (e) {
      error = (e as Error).message;
    }
  });
  function toggleTheme() {
    const dark = document.documentElement.classList.toggle('dark');
    try {
      localStorage.setItem('theme', dark ? 'dark' : 'light');
    } catch {
      /* Storage may be disabled. */
    }
  }
  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (busy) return;
    if (register && password !== confirmPassword) {
      error = 'Passwords do not match';
      return;
    }
    busy = true;
    error = '';
    try {
      await api.post(register ? '/auth/register' : '/auth/login', {
        body: { username, password }
      });
      await refreshAuth();
      await goto(url('/'));
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
    }
  }
</script>

<main
  class="auth-shell relative isolate h-dvh overflow-auto bg-background px-6 py-10 text-foreground sm:px-10"
>
  <AuthBackground />
  <div class="absolute right-5 top-5 z-10">
    <Button
      variant="ghost"
      aria-label="Toggle color theme"
      onclick={toggleTheme}
    >
      <SunMoon size={18} />
    </Button>
  </div>
  <div
    class="relative mx-auto flex min-h-full w-full max-w-md flex-col justify-center py-10"
  >
    <section aria-labelledby="auth-title" class="auth-panel w-full">
      <header class="mb-8 flex flex-col items-center text-center">
        <div
          class="garage-logo-tile mb-5 flex h-16 w-16 items-center justify-center rounded-2xl"
        >
          <img src={logo} alt="Garage" class="h-12 w-12" />
        </div>
        <p
          class="mb-3 text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground"
        >
          Garage Web Console
        </p>
        <h1 id="auth-title" class="text-2xl font-semibold tracking-tight">
          {register ? 'Welcome to Garage' : 'Welcome back'}
        </h1>
        <p class="mt-2 text-sm text-muted-foreground">
          {register
            ? 'Create the admin account to get started.'
            : 'Sign in to your storage console.'}
        </p>
      </header>
      <form onsubmit={submit} class="space-y-4">
        <label class="block text-sm font-medium">
          Username
          <input
            class="input mt-2"
            bind:value={username}
            autocomplete="username"
            required
          />
        </label>
        <div class="relative">
          <label class="block text-sm font-medium">
            Password
            <input
              class="input mt-2 pr-12"
              type={showPassword ? 'text' : 'password'}
              bind:value={password}
              autocomplete={register ? 'new-password' : 'current-password'}
              minlength={register ? 6 : 1}
              required
            />
          </label>
          <button
            type="button"
            aria-label={showPassword ? 'Hide password' : 'Show password'}
            class="absolute bottom-2 right-3 text-muted-foreground"
            onclick={() => (showPassword = !showPassword)}
          >
            {#if showPassword}<EyeOff size={18} />{:else}<Eye size={18} />{/if}
          </button>
        </div>
        {#if register}<label class="block text-sm font-medium">
            Confirm password
            <input
              class="input mt-2"
              type="password"
              bind:value={confirmPassword}
              autocomplete="new-password"
              required
            />
          </label>{/if}{#if error}<p
            role="alert"
            class="text-sm text-destructive"
          >
            {error}
          </p>{/if}<Button type="submit" class="w-full" disabled={busy}>
          {busy
            ? 'Please wait…'
            : register
              ? 'Create admin account'
              : 'Sign in'}
        </Button>
      </form>
      {#if !register && $auth?.oidcEnabled}
        <div class="my-6 flex items-center gap-4 text-xs text-muted-foreground">
          <span class="h-px flex-1 bg-border"></span>
          <span>Or continue with</span>
          <span class="h-px flex-1 bg-border"></span>
        </div>
        <a
          class="flex min-h-10 items-center justify-center gap-2 rounded-md border bg-card px-3 py-2 text-center text-sm font-medium transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          href={API_URL + '/v1/auth/oidc/login'}
        >
          {#if $auth.oidcButtonIconURL}
            <img
              src={$auth.oidcButtonIconURL}
              alt=""
              class="h-5 w-5 object-contain"
              referrerpolicy="no-referrer"
            />
          {:else}
            <KeyRound size={20} aria-hidden="true" />
          {/if}
          {$auth.oidcButtonText}
        </a>{/if}
    </section>
    <p class="mt-8 text-center text-xs text-muted-foreground">
      Your storage. Your infrastructure.
    </p>
  </div>
</main>
