import { describe, expect, it } from 'vitest';
import { objectPath, folderName, breadcrumbs } from '../../src/lib/objects';

describe('object paths', () => {
  it('encodes reserved characters without flattening nested folder paths', () => {
    expect(objectPath('my bucket', 'photos/a #?%+雪.jpg')).toBe(
      '/browse/my%20bucket/photos/a%20%23%3F%25%2B%E9%9B%AA.jpg'
    );
    expect(objectPath('bucket', 'empty/')).toBe('/browse/bucket/empty/');
  });
  it('keeps nested destination folders and rejects traversal', () => {
    expect(folderName('/parent/child/')).toBe('parent/child/');
    for (const path of ['', '/', '../x', 'x/../y', 'x//y', './x'])
      expect(() => folderName(path)).toThrow();
  });
  it('builds ancestor destinations for the breadcrumb folder picker', () => {
    expect(breadcrumbs('parent/child/')).toEqual([
      { label: 'parent', prefix: 'parent/' },
      { label: 'child', prefix: 'parent/child/' }
    ]);
  });
});
