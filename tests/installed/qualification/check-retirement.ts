// Real native/filesystem guard integration. The scratch is a newly generated,
// explicitly incomplete development fixture, never a Factory execution receipt.
// No --execute is sent for a retained clone and no successful deletion is faked.
import assert from 'node:assert/strict';
import {access, chmod, lstat, mkdir, readFile, realpath, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {sha256} from './validate';

process.umask(0o077);
assert.equal(process.getuid?.(), 0, 'local root integration required');
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--operator', '--cold-restored', '--cold-result', '--out']
);
assert.equal(args.length, 8);
const [operator, restored, coldResult, out] = [args[1]!, args[3]!, args[5]!, args[7]!];
for (const p of [operator, restored, coldResult]) assert(p.startsWith('/home/') && (await realpath(p)) === p);
assert(out.startsWith('/home/') && path.dirname(out) === '/home' && (await realpath('/home')) === '/home');
const operatorSHA = sha256(await readFile(operator)),
  resultSHA = sha256(await readFile(coldResult));
await mkdir(out, {mode: 0o700});
async function save(name: string, value: unknown, mode = 0o600) {
  await writeFile(path.join(out, name), JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode});
}
async function execute(argv: string[], success = true) {
  const child = Bun.spawn(argv, {stdin: 'ignore', stdout: 'pipe', stderr: 'pipe'});
  const [stdout] = await Promise.all([new Response(child.stdout).text(), new Response(child.stderr).text()]);
  const code = await child.exited;
  if (!success) {
    assert.notEqual(code, 0);
    return null;
  }
  assert.equal(code, 0, 'selected retirement command failed');
  return JSON.parse(stdout);
}
const scratch = path.join(out, 'scratch-incomplete'),
  work = path.join(scratch, 'consumed-work');
await mkdir(scratch, {mode: 0o700});
await mkdir(work, {mode: 0o700});
for (const name of ['home', 'tmp', 'profile']) await mkdir(path.join(work, name), {mode: 0o700});
const witness = path.join(work, 'tmp', 'local-witness'),
  witnessBytes = Buffer.from('Retirement guard integration; preserve incomplete work.\n');
await writeFile(witness, witnessBytes, {flag: 'wx', mode: 0o600});
const write = (name: string, value: unknown, mode = 0o600) =>
  writeFile(path.join(scratch, name), JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode});
// A real unpaid native unit proves the terminal observation. It is unrelated
// to product/Factory success and cannot qualify any release.
const unit = 'soda-qualify-' + crypto.randomUUID().replaceAll('-', '') + '.service';
const native = Bun.spawnSync(
  ['/usr/bin/systemd-run', '--quiet', '--wait', '--collect', '--unit', unit, '/usr/bin/true'],
  {stdin: 'ignore', stdout: 'pipe', stderr: 'pipe', timeout: 30000}
);
assert.equal(native.exitCode, 0, 'actual unpaid native unit must finish');
const resolved = {
  format: 1,
  fixture: 'incomplete local filesystem guard; no paid authority or product outcome',
  scenario: {authority: {held: true}, permitted_effects: {retire_consumed_scratch: false}},
};
await write('resolved-input.json', resolved, 0o444);
await chmod(path.join(scratch, 'resolved-input.json'), 0o444);
const binding = {
  format: 1,
  purpose: 'development',
  source_revision: '0'.repeat(40),
  architecture: 'x86_64',
  candidate_sha256: '0'.repeat(64),
  payload_sha256: '0'.repeat(64),
  build_sha256: '0'.repeat(64),
  target: 'local-incomplete-scratch-guard',
  controller_sha256: operatorSHA,
  collector_sha256: operatorSHA,
  authority_sha256: '0'.repeat(64),
  unit,
  exit_code: -1,
  native_retirement: 'pending',
  files: {},
};
await mkdir(path.join(scratch, 'attempt'), {mode: 0o700});
await write('request.json', {
  authority: path.join(scratch, 'not-an-authority.json'),
  candidate: path.join(scratch, 'not-a-candidate'),
  authority_sha256: binding.authority_sha256,
  unit,
  attempt: path.join(scratch, 'attempt'),
  binding,
  resolved_input: {
    path: path.join(scratch, 'resolved-input.json'),
    sha256: sha256(await readFile(path.join(scratch, 'resolved-input.json'))),
  },
});
const meta = await lstat(work);
await write('work-origin.json', {format: 1, device: meta.dev, inode: meta.ino, uid: meta.uid, gid: meta.gid});
await write('work-preserved.json', {
  format: 1,
  state: 'retained',
  resource: 'collector-scratch',
  path: work,
  origin_sha256: sha256(await readFile(path.join(scratch, 'work-origin.json'))),
  request_sha256: sha256(await readFile(path.join(scratch, 'request.json'))),
  outside_sealed_evidence: true,
  automatic_retry: false,
});
const plan = await execute([operator, 'journey', 'retire-scratch', '--out', scratch]);
assert(
  plan.state === 'retained' &&
    plan.completed_evidence === false &&
    plan.scope_permitted === false &&
    plan.worker_terminal === true
);
await execute([operator, 'journey', 'retire-scratch', '--out', scratch, '--execute'], false);
await assert.rejects(access(path.join(scratch, 'scratch-retirement-intent.json')));
assert.deepEqual(await readFile(witness), witnessBytes);
const holder = Bun.spawn(
  [
    '/usr/bin/node',
    '-e',
    'require("fs").openSync(process.argv[1],"r");process.stdout.write("ready");setInterval(()=>{},1000)',
    witness,
  ],
  {stdin: 'ignore', stdout: 'pipe', stderr: 'pipe'}
);
try {
  const reader = holder.stdout.getReader();
  const ready = await reader.read();
  reader.releaseLock();
  assert(!ready.done && Buffer.from(ready.value).toString() === 'ready');
  await execute([operator, 'journey', 'retire-scratch', '--out', scratch], false);
} finally {
  holder.kill();
  await holder.exited;
}
await mkdir(path.join(work, 'unique-retained-data'), {mode: 0o700});
await execute([operator, 'journey', 'retire-scratch', '--out', scratch], false);
assert.deepEqual(await readFile(witness), witnessBytes);
const completeBefore = sha256(await readFile(path.join(restored, 'complete.json')));
const cold = await execute([
  operator,
  'journey',
  'cold',
  'retire',
  '--restored',
  restored,
  '--result-sha256',
  resultSHA,
]);
assert(cold.action === 'isolated-cold-retirement-plan' && cold.source_preserved && cold.private_settings_preserved);
for (const tree of cold.trees) assert((await lstat(tree)).isDirectory());
assert.equal(sha256(await readFile(path.join(restored, 'complete.json'))), completeBefore);
assert.equal(sha256(await readFile(coldResult)), resultSHA);
assert.equal(sha256(await readFile(operator)), operatorSHA);
await save('scratch-plan.json', plan);
await save('cold-plan.json', cold);
const result = {
  format: 1,
  passed: true,
  check: 'actual-local-retirement-guards',
  operator_sha256: operatorSHA,
  actual_unpaid_unit_terminal: true,
  incomplete_scratch_retained: true,
  held_scope_execute_refused: true,
  live_file_reference_refused: true,
  unrecognized_resource_refused: true,
  retained_clone_plan_only: true,
  source_and_copies_preserved: true,
  resources_deleted: 0,
  provider_calls: 0,
  guest_mutations: 0,
  qualification: null,
};
await save('result.json', result);
console.log(JSON.stringify(result));
