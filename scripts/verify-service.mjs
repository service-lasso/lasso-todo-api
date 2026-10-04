import { readFile, mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { spawn, spawnSync } from 'node:child_process';
import { once } from 'node:events';
import { createServer } from 'node:net';
import { createHash } from 'node:crypto';
import assert from 'node:assert/strict';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
await mkdir(path.join(root, '.tmp'), { recursive: true });
const consumer = await mkdtemp(path.join(root, '.tmp', 'consumer-'));
const platform = process.platform;
const archive = path.join(root, 'dist', `lasso-todo-api-${platform}.${platform === 'win32' ? 'zip' : 'tar.gz'}`);
const digest = createHash('sha256').update(await readFile(archive)).digest('hex');
const result = platform === 'win32'
  ? spawnSync('powershell.exe', ['-NoProfile', '-Command', 'Add-Type -AssemblyName System.IO.Compression.FileSystem; [IO.Compression.ZipFile]::ExtractToDirectory($env:LASSO_PACKAGE_ARCHIVE, $env:LASSO_PACKAGE_CONSUMER)'], { env: { ...process.env, LASSO_PACKAGE_ARCHIVE: archive, LASSO_PACKAGE_CONSUMER: consumer }, windowsHide: true })
  : spawnSync('tar', ['-xzf', archive, '-C', consumer]);
if (result.error || result.status !== 0) throw new Error('Archive extraction failed');
const binary = path.join(consumer, 'runtime', platform === 'win32' ? 'todo-api.exe' : 'todo-api');
assert.ok((await readFile(binary)).length > 1000, 'Expected actual packaged binary');
const statePath = process.env.TODO_VERIFY_DATABASE_STATE;
let boundary = 'native archive structure; PostgreSQL integration is a separate required job';
let child, closed;
if (statePath) {
  const probe = createServer(); probe.listen(0, '127.0.0.1'); await once(probe, 'listening');
  const port = probe.address().port; await new Promise(resolve => probe.close(resolve));
  const url = `http://127.0.0.1:${port}`;
  const start = async () => {
    child = spawn(binary, [], { cwd: consumer, env: { ...process.env, TODO_API_AUTH_MODE: 'anonymous', TODO_API_PORT: String(port), TODO_DATABASE_STATE: path.resolve(statePath) }, stdio: 'pipe', windowsHide: true });
    closed = once(child, 'close'); let logs = ''; child.stderr.on('data', bytes => { logs += bytes; });
    for (let i = 0; i < 200; i++) {
      if (child.exitCode !== null) throw new Error('Packaged API exited: ' + logs);
      try { if ((await fetch(url + '/healthz')).ok) return; } catch {}
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    throw new Error('API readiness timeout');
  };
  const stop = async () => { child.kill(); await closed; child = undefined; };
  try {
    await start();
    const post = title => fetch(url + '/todos', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ title }) });
    assert.equal((await post('')).status, 400);
    assert.equal((await fetch(url + '/todos', { method: 'POST', headers: { 'content-type': 'application/json', origin: 'https://other.example' }, body: '{"title":"foreign"}' })).status, 403);
    const created = await post('packaged API SQL persistence'); assert.equal(created.status, 201); const todo = await created.json();
    await stop(); await start();
    assert.ok((await (await fetch(url + '/todos')).json()).some(item => item.id === todo.id && item.title === todo.title));
    boundary = 'native archive runtime, real PostgreSQL create/list and restart persistence; Core managed lifecycle verified separately';
  } finally { if (child) await stop(); }
}
await mkdir(path.join(root, 'output'), { recursive: true });
await writeFile(path.join(root, 'output', 'package-verification.json'), JSON.stringify({ platform, archiveHash: digest, sqlIntegration: statePath ? 'passed' : 'deferred to required integration job', boundary }, null, 2) + '\n');
console.log(boundary);
