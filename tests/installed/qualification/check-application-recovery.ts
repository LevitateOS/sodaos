// Reuse a retained owner/data/record and actual native writer. No bootstrap,
// image build, enrollment, model call, guest mutation or replacement credential.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {mkdir, readFile, writeFile, realpath} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {startLocalApplication} from './local-application-runtime';

process.umask(0o077);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--storage', '--runtime', '--bundle', '--module', '--proof', '--out']
);
assert.equal(args.length, 12);
const storage = args[1]!,
  runtime = args[3]!,
  bundle = args[5]!,
  moduleFile = args[7]!,
  proof = args[9]!,
  out = args[11]!;
assert(out.startsWith('/home/') && path.resolve(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out));
await mkdir(out, {mode: 0o700});
const temporary = path.join(out, 'tmp');
await mkdir(temporary, {mode: 0o700});
const env = {...process.env, TMPDIR: temporary};
const prior = JSON.parse(await readFile(path.join(proof, 'result.json'), 'utf8'));
assert(
  prior.passed &&
    prior.same_owner_and_record_after_restart &&
    prior.retirements.every((r: any) => r.writer_lock_available)
);
const sourceData = path.join(proof, 'data'),
  sourceRecord = prior.record;
assert.equal(
  createHash('sha256')
    .update(await readFile(bundle))
    .digest('hex'),
  prior.bundle.sha256
);
const credentials = JSON.parse(await readFile(path.join(proof, 'private', 'owner.json'), 'utf8'));
const {openApplicationOwner, nativeHttpScript} = (await import(
  pathToFileURL(moduleFile).href
)) as typeof import('./phases/index');
const retirements: unknown[] = [];
const start = (data: string) =>
  startLocalApplication({
    runtime,
    bundle,
    data,
    owner_team: prior.owner_team,
    env,
    retired: (r) => retirements.push(r),
  });
const save = (relative: string, value: unknown) =>
  writeFile(path.join(out, relative), JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode: 0o600});
function snapshot(action: 'backup' | 'restore', source: string, destination: string) {
  return spawnSync(storage, [action, '--source', source, '--out', destination], {
    env,
    stdio: ['ignore', 'pipe', 'pipe'],
    timeout: 30000,
    maxBuffer: 65536,
  });
}
async function read(live: Awaited<ReturnType<typeof start>>) {
  const owner = await openApplicationOwner({
    origin: live.origin,
    owner_team: prior.owner_team,
    credentials,
    transport: async (request) => {
      const response = spawnSync('/usr/bin/node', ['-'], {
        input: nativeHttpScript(live.origin, '127.0.0.1', request),
        env,
        timeout: 30000,
        maxBuffer: 4 << 20,
        stdio: ['pipe', 'pipe', 'pipe'],
      });
      assert.equal(response.status, 0, 'real_namespace_http_unconfirmed');
      const value = JSON.parse(response.stdout.toString());
      return {
        status: value.status,
        text: Buffer.from(value.body_base64, 'base64').toString('utf8'),
        set_cookie: value.set_cookie,
      };
    },
  });
  await owner.readRecord(sourceRecord);
  return owner.readRendered(sourceRecord);
}
let live: Awaited<ReturnType<typeof start>> | undefined;
try {
  live = await start(sourceData);
  await read(live);
  const refused = snapshot('backup', sourceData, path.join(out, 'refused-live-writer'));
  assert.notEqual(refused.status, 0);
  assert(refused.stderr.toString().includes('stop the Canlang writer'));
  await live.stop();
  live = undefined;
  const cold = path.join(out, 'cold'),
    restored = path.join(out, 'restored');
  assert.equal(snapshot('backup', sourceData, cold).status, 0, 'stopped_application_snapshot_failed');
  assert.equal(snapshot('restore', cold, restored).status, 0, 'fresh_application_restore_failed');
  live = await start(restored);
  const restoredRead = await read(live);
  await live.stop();
  live = undefined;
  // The original remains usable with its original owner, record and directory.
  live = await start(sourceData);
  await read(live);
  await live.stop();
  live = undefined;
  await save('result.json', {
    format: 1,
    passed: true,
    check: 'actual-retained-application-cold-restore',
    source_proof: proof,
    source_data: sourceData,
    restored_data: restored,
    owner_team: prior.owner_team,
    record: sourceRecord,
    restored_read: restoredRead,
    live_writer_refused: true,
    real_curl_owner_transport: true,
    retirements,
    source_preserved: true,
    isolation: 'loopback-only native runtime; no fixture AI or schedules',
    scope: 'App storage only; not a complete Docker recovery qualification',
    storage_sha256: createHash('sha256')
      .update(await readFile(storage))
      .digest('hex'),
    module_sha256: createHash('sha256')
      .update(await readFile(moduleFile))
      .digest('hex'),
    guest_mutations: 0,
    provider_calls: 0,
  });
  console.log(JSON.stringify({passed: true, result: path.join(out, 'result.json')}));
} finally {
  await live?.stop();
}
