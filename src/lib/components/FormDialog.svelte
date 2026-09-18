<script module lang="ts">
  export type Field = {
    name: string;
    label: string;
    type?:
      | 'text'
      | 'password'
      | 'email'
      | 'number'
      | 'checkbox'
      | 'select'
      | 'multi';
    required?: boolean;
    min?: number;
    step?: string;
    options?: { value: string; label: string }[];
    hint?: string;
  };
  export type Values = Record<string, any>;
</script>

<script lang="ts">
  import Modal from './Modal.svelte';
  import Button from './Button.svelte';
  import { toast } from 'svelte-sonner';
  let {
    open = $bindable(false),
    title,
    fields,
    initial = {},
    submit,
    submitLabel = 'Save',
    description = ''
  }: {
    open?: boolean;
    title: string;
    fields: Field[];
    initial?: Values;
    submit: (values: Values) => Promise<unknown>;
    submitLabel?: string;
    description?: string;
  } = $props();
  let values = $state<Values>({});
  let busy = $state(false);
  let error = $state('');
  $effect(() => {
    if (open) {
      values = { ...initial };
      error = '';
    }
  });
  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (busy) return;
    busy = true;
    error = '';
    try {
      await submit(values);
      open = false;
      toast.success('Saved successfully');
    } catch (reason) {
      error = reason instanceof Error ? reason.message : 'Request failed';
    } finally {
      busy = false;
    }
  }
</script>

<Modal bind:open {title} {description}>
  <form onsubmit={save} class="space-y-4">
    {#each fields as field (field.name)}
      <label class="block space-y-1.5 text-sm font-medium">
        <span>{field.label}</span>
        {#if field.type === 'checkbox'}
          <input
            type="checkbox"
            bind:checked={values[field.name]}
            class="ml-3"
          />
        {:else if field.type === 'select'}
          <select
            class="input"
            bind:value={values[field.name]}
            required={field.required}
          >
            {#each field.options || [] as option (option.value)}<option
                value={option.value}
              >
                {option.label}
              </option>{/each}
          </select>
        {:else if field.type === 'multi'}
          <select
            class="input min-h-28"
            multiple
            bind:value={values[field.name]}
          >
            {#each field.options || [] as option (option.value)}<option
                value={option.value}
              >
                {option.label}
              </option>{/each}
          </select>
        {:else if field.type === 'number'}
          <input
            class="input"
            type="number"
            bind:value={values[field.name]}
            min={field.min}
            step={field.step || 'any'}
            required={field.required}
          />
        {:else}
          <input
            class="input"
            type={field.type || 'text'}
            bind:value={values[field.name]}
            required={field.required}
            autocomplete={field.type === 'password' ? 'new-password' : 'off'}
          />
        {/if}
        {#if field.hint}<span class="block text-xs text-muted-foreground">
            {field.hint}
          </span>{/if}
      </label>
    {/each}
    {#if error}<p role="alert" class="text-sm text-destructive">{error}</p>{/if}
    <div class="flex justify-end gap-2 pt-3">
      <Button variant="outline" onclick={() => (open = false)} disabled={busy}>
        Cancel
      </Button><Button type="submit" disabled={busy}>
        {busy ? 'Saving…' : submitLabel}
      </Button>
    </div>
  </form>
</Modal>
