import { beforeEach, afterEach, expect, it, vi } from 'vitest';

vi.mock('#src/lib/api.ts', () => ({ API_URL: '/console/api' }));

class FakeXHR {
  static requests: FakeXHR[] = [];
  upload = { onprogress: () => {} };
  status = 200;
  statusText = 'OK';
  responseText = '';
  withCredentials = false;
  url = '';
  onload = () => {};
  onerror = () => {};
  onabort = () => {};
  open(_method: string, url: string) {
    this.url = url;
  }
  send() {
    FakeXHR.requests.push(this);
  }
  abort() {
    this.onabort();
  }
}

beforeEach(() => {
  vi.resetModules();
  FakeXHR.requests = [];
  vi.stubGlobal('XMLHttpRequest', FakeXHR);
});
afterEach(() => vi.unstubAllGlobals());

it('limits concurrent uploads, deduplicates keys, cancels pending tasks, and retries failures', async () => {
  const { uploadStore } = await import('../../src/stores/upload-store');
  uploadStore.enqueue(
    ['a', 'b', 'c', 'd', 'a'].map((key) => ({
      bucket: 'photos',
      key,
      file: null
    }))
  );
  expect(uploadStore.getState().tasks).toHaveLength(4);
  expect(FakeXHR.requests).toHaveLength(3);
  const pending = uploadStore
    .getState()
    .tasks.find((task) => task.key === 'd')!;
  uploadStore.cancel(pending.id);
  expect(
    uploadStore.getState().tasks.find((task) => task.key === 'd')?.status
  ).toBe('error');
  FakeXHR.requests[0].onload();
  await vi.waitFor(() =>
    expect(uploadStore.getState().tasks[0].status).toBe('success')
  );
  expect(FakeXHR.requests).toHaveLength(3);
  uploadStore.retry(pending.id);
  expect(FakeXHR.requests).toHaveLength(4);
  expect(
    uploadStore.getState().tasks.find((task) => task.key === 'd')?.status
  ).toBe('uploading');
  uploadStore.clearAll();
  await Promise.resolve();
  expect(uploadStore.getState().tasks).toHaveLength(0);
});

it('preserves base paths and escapes reserved characters in upload keys', async () => {
  const { uploadStore } = await import('../../src/stores/upload-store');
  uploadStore.enqueue([
    { bucket: 'my bucket', key: 'nested/a #?.txt', file: null }
  ]);
  expect(FakeXHR.requests[0].url).toBe(
    '/console/api/browse/my%20bucket/nested/a%20%23%3F.txt'
  );
  expect(FakeXHR.requests[0].withCredentials).toBe(true);
  uploadStore.clearAll();
});
