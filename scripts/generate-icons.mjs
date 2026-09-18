// Regenerate browser icons from the canonical selfh.st Garage SVG.
// PLAYWRIGHT_CHANNEL=chrome node scripts/generate-icons.mjs
import { readFile, writeFile } from 'node:fs/promises';
import { chromium } from '@playwright/test';

const source = await readFile(
  new URL('../src/assets/garage-logo.svg', import.meta.url),
  'utf8'
);
const logo = source.replace(
  '<svg ',
  '<svg x="48" y="48" width="416" height="416" '
);
const icon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><rect width="512" height="512" rx="96" fill="#fffaf2"/>${logo}</svg>`;
await writeFile(new URL('../public/favicon.svg', import.meta.url), icon);
const browser = await chromium.launch({
  channel: process.env.PLAYWRIGHT_CHANNEL || undefined
});
try {
  const page = await browser.newPage();
  const pngs = new Map();
  for (const size of [16, 32, 48, 180]) {
    const data = await page.evaluate(
      async ({ icon, size }) => {
        const img = new Image();
        img.src = 'data:image/svg+xml;base64,' + btoa(icon);
        await img.decode();
        const canvas = document.createElement('canvas');
        canvas.width = canvas.height = size;
        canvas.getContext('2d').drawImage(img, 0, 0, size, size);
        return canvas.toDataURL('image/png').split(',')[1];
      },
      { icon, size }
    );
    pngs.set(size, Buffer.from(data, 'base64'));
  }
  await writeFile(
    new URL('../public/favicon-32x32.png', import.meta.url),
    pngs.get(32)
  );
  await writeFile(
    new URL('../public/apple-touch-icon.png', import.meta.url),
    pngs.get(180)
  );
  const sizes = [16, 32, 48];
  const header = Buffer.alloc(6 + 16 * sizes.length);
  header.writeUInt16LE(1, 2);
  header.writeUInt16LE(sizes.length, 4);
  let offset = header.length;
  sizes.forEach((size, i) => {
    const entry = 6 + i * 16;
    header[entry] = header[entry + 1] = size;
    header.writeUInt16LE(1, entry + 4);
    header.writeUInt16LE(32, entry + 6);
    header.writeUInt32LE(pngs.get(size).length, entry + 8);
    header.writeUInt32LE(offset, entry + 12);
    offset += pngs.get(size).length;
  });
  await writeFile(
    new URL('../public/favicon.ico', import.meta.url),
    Buffer.concat([header, ...sizes.map((size) => pngs.get(size))])
  );
} finally {
  await browser.close();
}
