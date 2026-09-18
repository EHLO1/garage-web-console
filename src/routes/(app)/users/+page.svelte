<script lang="ts">
  import { auth } from '#lib/auth.ts';
  import api from '#lib/api.ts';
  import { resource } from '#lib/resource.svelte.ts';
  import type { User } from '#lib/types/users.ts';
  import type { Bucket } from '#lib/types/buckets.ts';
  import Button from '#lib/components/Button.svelte';
  import FormDialog from '#lib/components/FormDialog.svelte';
  import RequestState from '#lib/components/RequestState.svelte';
  const data = resource(async () => {
    const [users, buckets] = await Promise.all([
      api.get<User[]>('/users'),
      api.get<Bucket[]>('/buckets')
    ]);
    return { users, buckets };
  });
  let edit = $state<User | null>(null);
  let open = $state(false);
  let deleting = $state<User | null>(null);
  let search = $state('');
  let initial = $derived({
    username: edit?.username || '',
    email: edit?.email || '',
    password: '',
    role: edit?.role || 'user',
    buckets: edit?.buckets || []
  });
  let fields = $derived([
    { name: 'username', label: 'Username', required: true },
    {
      name: 'email',
      label: 'Email for OIDC sign-in',
      type: 'email' as const
    },
    {
      name: 'password',
      label: edit
        ? 'New password (leave blank to keep)'
        : 'Password (optional with email)',
      type: 'password' as const
    },
    {
      name: 'role',
      label: 'Role',
      type: 'select' as const,
      options: ['admin', 'user', 'viewer'].map((value) => ({
        value,
        label: value
      }))
    },
    {
      name: 'buckets',
      label: 'Assigned buckets (users and viewers)',
      type: 'multi' as const,
      options: (data.data?.buckets || []).map((b) => ({
        value: b.id,
        label: b.globalAliases?.[0] || b.id
      }))
    }
  ]);
</script>

<div class="toolbar">
  <div>
    <h1 class="page-title">Users</h1>
    <p class="muted mt-1">Manage accounts and access</p>
  </div>
  <Button
    onclick={() => {
      edit = null;
      open = true;
    }}
  >
    Add user
  </Button>
</div>
<p class="muted mb-5">
  Admins have full access. Users can manage assigned buckets and objects.
  Viewers can browse and download from assigned buckets. Keys are admin-only.
</p>
<input
  class="input mb-5 max-w-sm"
  aria-label="Search users"
  placeholder="Search users…"
  bind:value={search}
/>
<RequestState loading={data.loading} error={data.error} retry={data.reload} />
<section class="card overflow-auto">
  <table>
    <thead>
      <tr>
        <th>Username</th>
        <th>Email</th>
        <th>Role</th>
        <th>Buckets</th>
        <th>Created</th>
        <th>Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each (data.data?.users || []).filter( (user) => `${user.username} ${user.email}`
            .toLowerCase()
            .includes(search.toLowerCase()) ) as user (user.id)}<tr>
          <td>{user.username}</td>
          <td>{user.email || '—'}</td>
          <td><span class="badge">{user.role}</span></td>
          <td>
            {user.role !== 'admin'
              ? (user.buckets || [])
                  .map(
                    (id) =>
                      data.data?.buckets.find((b) => b.id === id)
                        ?.globalAliases?.[0] || id
                  )
                  .join(', ') || 'None'
              : 'All'}
          </td>
          <td>{new Date(user.createdAt).toLocaleDateString()}</td>
          <td>
            <Button
              variant="ghost"
              onclick={() => {
                edit = user;
                open = true;
              }}
            >
              Edit
            </Button>{#if user.id !== $auth?.user?.id}<Button
                variant="ghost"
                onclick={() => (deleting = user)}
              >
                Delete
              </Button>{/if}
          </td>
        </tr>{:else}<tr>
          <td colspan="6" class="muted py-8 text-center">No users found.</td>
        </tr>{/each}
    </tbody>
  </table>
</section>
<FormDialog
  bind:open
  title={edit ? 'Edit user' : 'Add user'}
  {initial}
  {fields}
  submit={async (values) => {
    if (values.password && values.password.length < 6)
      throw new Error('Password must have at least 6 characters');
    if (!edit && !values.password && !values.email)
      throw new Error('Provide a password or email');
    const body = {
      username: values.username,
      email: values.email || '',
      role: values.role,
      buckets: values.role !== 'admin' ? values.buckets || [] : [],
      ...(values.password ? { password: values.password } : {})
    };
    if (edit) await api.fetch('/users/' + edit.id, { method: 'PATCH', body });
    else await api.post('/users', { body });
    data.reload();
  }}
/>
<FormDialog
  bind:open={
    () => deleting !== null,
    (value) => {
      if (!value) deleting = null;
    }
  }
  title="Delete user"
  description={`Delete ${deleting?.username}? This cannot be undone.`}
  fields={[]}
  submitLabel="Delete user"
  submit={async () => {
    await api.delete('/users/' + deleting?.id);
    deleting = null;
    data.reload();
  }}
/>
