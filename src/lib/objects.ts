export const encodeObjectKey = (key: string) =>
  key.split('/').map(encodeURIComponent).join('/');
export const objectPath = (bucket: string, key: string) =>
  `/browse/${encodeURIComponent(bucket)}/${encodeObjectKey(key)}`;
export function breadcrumbs(prefix: string) {
  const parts = prefix.split('/').filter(Boolean);
  return parts.map((label, index) => ({
    label,
    prefix: parts.slice(0, index + 1).join('/') + '/'
  }));
}
export function folderName(value: string) {
  const name = value.trim().replace(/^\/+|\/+$/g, '');
  if (
    !name ||
    name.split('/').some((part) => !part || part === '.' || part === '..')
  )
    throw new Error('Enter a valid folder path');
  return name + '/';
}
