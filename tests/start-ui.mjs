import { cpSync } from 'node:fs';
import { spawn } from 'node:child_process';
cpSync('dist', 'backend/ui/dist', { recursive: true });
const server = spawn('go', ['run', '-tags=prod', '../tests/serve-ui.go'], {
  cwd: 'backend',
  stdio: 'inherit',
  windowsHide: true
});
server.on('exit', (code) => process.exit(code ?? 1));
for (const signal of ['SIGINT', 'SIGTERM'])
  process.on(signal, () => server.kill(signal));
