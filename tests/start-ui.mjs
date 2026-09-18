import { cpSync, rmSync } from 'node:fs';
import { spawn } from 'node:child_process';
const embeddedAssets = new URL('../backend/ui/dist/', import.meta.url);
rmSync(embeddedAssets, { recursive: true, force: true });
cpSync(new URL('../dist/', import.meta.url), embeddedAssets, {
  recursive: true
});
const server = spawn('go', ['run', '-tags=prod', '../tests/serve-ui.go'], {
  cwd: 'backend',
  stdio: 'inherit',
  windowsHide: true
});
server.on('exit', (code) => process.exit(code ?? 1));
for (const signal of ['SIGINT', 'SIGTERM'])
  process.on(signal, () => server.kill(signal));
