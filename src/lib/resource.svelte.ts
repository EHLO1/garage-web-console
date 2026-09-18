// Requests are scoped to their component. Stale responses cannot overwrite a
// newer route/filter, and unmounted components stop receiving updates.
export function resource<T>(request: () => Promise<T>) {
  let data = $state<T>();
  let error = $state('');
  let loading = $state(true);
  let revision = $state(0);
  $effect(() => {
    void revision;
    let active = true;
    loading = true;
    error = '';
    request()
      .then((value) => {
        if (active) data = value;
      })
      .catch((reason: unknown) => {
        if (active)
          error = reason instanceof Error ? reason.message : 'Request failed';
      })
      .finally(() => {
        if (active) loading = false;
      });
    return () => {
      active = false;
    };
  });
  return {
    get data() {
      return data;
    },
    get error() {
      return error;
    },
    get loading() {
      return loading;
    },
    reload() {
      revision += 1;
    }
  };
}
