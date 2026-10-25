// Real local collector-boundary integration. Synthetic metadata deliberately
// lacks a candidate, so no worker, appliance or provider can be dispatched.
import assert from 'node:assert/strict';
import {createHash, randomUUID} from 'node:crypto';
import {constants} from 'node:fs';
import {access, chmod, copyFile, lstat, mkdir, open, readFile, realpath, writeFile} from 'node:fs/promises';
import path from 'node:path';

process.umask(0o077);
assert.equal(process.geteuid?.(), 0, 'root required for protected local integration');
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--operator', '--out']
);
assert.equal(args.length, 4);
const operator = args[1]!,
  out = args[3]!;
assert(path.isAbsolute(operator) && path.resolve(operator) === operator);
assert.equal(await realpath(operator), operator);
assert.equal(path.dirname(out), '/home', 'fresh direct child of /home required');
assert.equal(path.resolve(out), out);
const source = await lstat(operator);
assert(source.isFile() && source.nlink === 1 && (source.mode & 0o111) !== 0);
const sha = (bytes: Uint8Array | string) => createHash('sha256').update(bytes).digest('hex');
const write = (file: string, value: unknown) =>
  writeFile(file, JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode: 0o600});
async function trusted(file: string, privateDirectory = false) {
  for (let current = file; ; current = path.dirname(current)) {
    const stat = await lstat(current);
    assert(stat.isDirectory() && stat.uid === 0 && (stat.mode & 0o022) === 0);
    if (current === file && privateDirectory) assert.equal(stat.mode & 0o7777, 0o700);
    if (current === '/') break;
  }
}
async function privateDirectory(file: string, fresh = false) {
  try {
    await mkdir(file, {mode: 0o700});
  } catch (error) {
    if (fresh || (error as NodeJS.ErrnoException).code !== 'EEXIST') throw error;
  }
  await trusted(file, true);
}
async function absent(file: string) {
  await assert.rejects(access(file), (e: unknown) => (e as NodeJS.ErrnoException).code === 'ENOENT');
}
async function execute(argv: string[]) {
  const child = Bun.spawn(argv, {stdin: 'ignore', stdout: 'pipe', stderr: 'pipe'});
  const [stdout, stderr, code] = await Promise.all([
    new Response(child.stdout).text(),
    new Response(child.stderr).text(),
    child.exited,
  ]);
  return {code, stdout, stderr};
}
await trusted('/home');
await privateDirectory(out, true);
const bin = path.join(out, 'bin'),
  outputs = path.join(out, 'outputs'),
  candidate = path.join(out, 'empty-candidate');
for (const directory of [bin, outputs, candidate]) await privateDirectory(directory, true);
const selected = path.join(bin, 'soda-release');
await copyFile(operator, selected, constants.COPYFILE_EXCL);
await chmod(selected, 0o700);
const operatorSHA = sha(await readFile(selected));
assert.equal(operatorSHA, sha(await readFile(operator)));
assert.equal((await lstat(selected)).mode & 0o077, 0);
const target = {distribution: 'docker', uuid: randomUUID()};
const resolved = path.join(out, 'resolved.json');
await write(resolved, {format: 1, scenario: {target}});
const authority = path.join(out, 'authority.json');
await write(authority, {
  controller: selected,
  controller_sha256: operatorSHA,
  collector: '/usr/bin/true',
  collector_sha256: sha(await readFile('/usr/bin/true')),
  output_parent: outputs,
  candidate_sha256: '1'.repeat(64),
  payload_sha256: '2'.repeat(64),
  build_sha256: '3'.repeat(64),
  source_revision: '4'.repeat(40),
  purpose: 'development',
  target: 'synthetic-target-lock-integration',
  timeout_seconds: 1,
  public_inputs: {},
  credentials: {},
  resolved_input: {path: resolved, sha256: sha(await readFile(resolved))},
});
const lockParent = '/home/.soda-qualification-targets';
await privateDirectory(lockParent);
const lockDirectory = path.join(lockParent, sha(target.distribution + ':' + target.uuid));
await privateDirectory(lockDirectory, true);
const lock = path.join(lockDirectory, 'controller.lock');
const handle = await open(lock, 'wx', 0o600);
await handle.sync();
await handle.close();
const holder = Bun.spawn(['/usr/bin/flock', '--no-fork', '-n', lock, '/usr/bin/sleep', '60'], {
  stdin: 'ignore',
  stdout: 'ignore',
  stderr: 'ignore',
});
const outcomes: {name: string; code: number; result: string}[] = [];
async function collect(name: string) {
  const output = path.join(outputs, name);
  const result = await execute([selected, 'collect', '--config', authority, '--candidate', candidate, '--out', output]);
  assert.notEqual(result.code, 0);
  assert.equal(result.stdout, '');
  await absent(output);
  return result;
}
try {
  const deadline = Date.now() + 3000;
  let acquired = false;
  while (Date.now() < deadline) {
    assert.equal(holder.exitCode, null, 'lock holder exited before acquisition');
    const probe = await execute(['/usr/bin/flock', '-n', lock, '/usr/bin/true']);
    assert(probe.code === 0 || probe.code === 1, 'lock probe failed');
    if (probe.code === 1) {
      acquired = true;
      break;
    }
    await Bun.sleep(20);
  }
  assert(acquired && holder.exitCode === null, 'live lock holder required');
  for (const name of ['first-output', 'second-output']) {
    const result = await collect(name);
    assert(result.stderr.includes('journey target collector already active'), 'shared target lock was bypassed');
    outcomes.push({name, code: result.code, result: 'same_target_lock_refused'});
  }
} finally {
  if (holder.exitCode === null) holder.kill('SIGTERM');
  await holder.exited;
}
const released = await collect('after-release');
assert(released.stderr.includes('lstat payload.json:'), 'released lock did not reach deliberate missing candidate');
assert(!released.stderr.includes('journey target collector already active'));
outcomes.push({name: 'after-release', code: released.code, result: 'missing_candidate_refused_before_dispatch'});
const result = {
  format: 1,
  operator_sha256: operatorSHA,
  integration_sha256: sha(await readFile(import.meta.path)),
  target,
  outcomes,
  collector_requests: 0,
  guest_effects: 0,
  model_calls: 0,
};
await write(path.join(out, 'result.json'), result);
console.log(JSON.stringify(result));
