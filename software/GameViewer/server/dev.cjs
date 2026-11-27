// Runs the Vite dev server and the vision server together. If either exits,
// the other is stopped and this process exits with the same status, so a
// crashed backend is never left half-running behind a live frontend.
const { spawn } = require('child_process');
const path = require('path');

const root = path.join(__dirname, '..');
const processes = [
  { name: 'vite', cmd: process.execPath, args: [path.join(root, 'node_modules', 'vite', 'bin', 'vite.js')] },
  { name: 'vision', cmd: process.execPath, args: [path.join(__dirname, 'visionServer.cjs')] },
];

let stopping = false;
const children = processes.map(({ name, cmd, args }) => {
  const child = spawn(cmd, args, { cwd: root, stdio: 'inherit', env: process.env });
  child.on('exit', (code, signal) => {
    if (stopping) return;
    console.error(`[dev] ${name} exited (${signal ?? code}); stopping the rest`);
    shutdown(code ?? 1);
  });
  return child;
});

function shutdown(code) {
  stopping = true;
  for (const child of children) {
    if (child.exitCode === null && child.signalCode === null) child.kill('SIGTERM');
  }
  process.exitCode = code;
}

for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => shutdown(0));
