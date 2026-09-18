<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { auth, refreshAuth } from '#lib/auth.ts';
  import api, { API_URL } from '#lib/api.ts';
  import { url } from '#lib/utils.ts';
  import Button from './Button.svelte';
  import AuthPreview from './AuthPreview.svelte';
  import { Eye, EyeOff } from '@lucide/svelte';
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

<main class="grid h-dvh grid-cols-1 overflow-auto md:grid-cols-2">
  <AuthPreview />
  <div
    class="dark flex items-center justify-center bg-[#0b0f1a] p-6 text-slate-100"
  >
    <section class="w-full max-w-sm py-10">
      <img src={logo} alt="Garage" class="mb-6 h-12 w-12" />
      <h1 class="page-title">
        {register ? 'Welcome to Garage' : 'Welcome back'}
      </h1>
      <p class="mb-7 mt-2 muted">
        {register
          ? 'Create the owner account to get started.'
          : 'Sign in to your storage console.'}
      </p>
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
              ? 'Create owner account'
              : 'Sign in'}
        </Button>
      </form>
      {#if !register && $auth?.googleEnabled}<a
          class="mt-4 block rounded-md border p-2 text-center text-sm font-medium hover:bg-accent"
          href={API_URL + '/v1/auth/google/login'}
        >
          Continue with Google
        </a>{/if}
    </section>
  </div>
</main>
