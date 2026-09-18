import { expect, it } from 'vitest';
import { readDataTransferItems } from '../../src/lib/file-drop';

it('reads all directory batches and preserves empty folder markers', async () => {
  let batch = 0;
  const file = new File(['contents'], 'hello.txt');
  const entry = {
    isDirectory: true,
    name: 'folder',
    createReader: () => ({
      readEntries: (done: (entries: unknown[]) => void) =>
        done(
          batch++ === 0
            ? [
                {
                  isFile: true,
                  name: 'hello.txt',
                  file: (done: (file: File) => void) => done(file)
                }
              ]
            : []
        )
    })
  };
  const transfer = {
    items: [{ webkitGetAsEntry: () => entry }]
  } as unknown as DataTransfer;
  expect(await readDataTransferItems(transfer)).toEqual([
    { path: 'folder/', file: null },
    { path: 'folder/hello.txt', file }
  ]);
  expect(batch).toBe(2);
});

it('propagates errors reading a child entry instead of hanging', async () => {
  const failure = new Error('File unavailable');
  const entry = {
    isDirectory: true,
    name: 'folder',
    createReader: () => ({
      readEntries: (done: (entries: unknown[]) => void) =>
        done([
          {
            isFile: true,
            name: 'bad.txt',
            file: (_done: unknown, reject: (error: Error) => void) =>
              reject(failure)
          }
        ])
    })
  };
  const transfer = {
    items: [{ webkitGetAsEntry: () => entry }]
  } as unknown as DataTransfer;
  await expect(readDataTransferItems(transfer)).rejects.toThrow(
    'File unavailable'
  );
});
