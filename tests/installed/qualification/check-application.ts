// Provider-free actual can-soda owner/record integration. Reuse the qualified
// runtime and portable RC7 bundle; never rebuild an image or touch guest data.
import assert from 'node:assert/strict';
import {randomBytes, createHash} from 'node:crypto';
import {spawn, spawnSync} from 'node:child_process';
import {mkdir, open, readFile, realpath, lstat} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import type {ApplicationRecordWitness} from './application-owner';
import {startLocalApplication} from './local-application-runtime';

process.umask(0o077);
const args = process.argv.slice(2);
const measuring = args.length === 14;
assert.deepEqual(
  args.filter((_, index) => index % 2 === 0),
  [
    '--runtime',
    '--bundle',
    '--module',
    '--out',
    ...(measuring ? ['--native-config', '--ssh-config', '--ssh-host'] : []),
  ]
);
assert(args.length === 8 || measuring);
const runtime = args[1]!,
  bundle = args[3]!,
  moduleFile = args[5]!,
  out = args[7]!;
assert(out.startsWith('/home/') && path.resolve(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out));
await mkdir(out, {mode: 0o700});
await mkdir(path.join(out, 'private'), {mode: 0o700});
const data = path.join(out, 'data'),
  temporary = path.join(out, 'private', 'tmp');
await mkdir(data, {mode: 0o700});
await mkdir(temporary, {mode: 0o700});
async function save(relative: string, value: unknown) {
  assert(
    /^[A-Za-z0-9._/-]+$/.test(relative) &&
      !relative.startsWith('/') &&
      relative.split('/').every((part) => part && part !== '.' && part !== '..')
  );
  const destination = path.join(out, relative),
    parent = path.dirname(destination);
  await mkdir(parent, {recursive: true, mode: 0o700});
  const file = await open(destination, 'wx', 0o600);
  try {
    await file.writeFile(JSON.stringify(value, null, 2) + '\n');
    await file.sync();
  } finally {
    await file.close();
  }
  const directory = await open(parent, 'r');
  try {
    await directory.sync();
  } finally {
    await directory.close();
  }
}
// Store credentials durably before bootstrap. A later failed check must not
// strand an otherwise reusable owner whose only password lived in memory.
const credentials = {
  email: 'qualification-' + randomBytes(8).toString('hex') + '@example.invalid',
  password: randomBytes(32).toString('base64url'),
};
await save('private/owner.json', credentials);
const credentialFile = path.join(out, 'private', 'owner.json');
const hash = (bytes: Uint8Array) => createHash('sha256').update(bytes).digest('hex');
for (const file of [runtime, bundle, moduleFile]) {
  assert(file.startsWith('/home/') && (await realpath(file)) === file);
  const metadata = await lstat(file);
  assert(metadata.isFile() && !metadata.isSymbolicLink());
}
const runtimeBytes = await readFile(runtime),
  bundleBytes = await readFile(bundle),
  moduleBytes = await readFile(moduleFile);
const {openApplicationOwner, startJourneyMeasurements, measurementSnapshotScript, innerDocker, matrixShellCommand} =
  (await import(pathToFileURL(moduleFile).href)) as Pick<
    typeof import('./phases/index'),
    | 'openApplicationOwner'
    | 'startJourneyMeasurements'
    | 'measurementSnapshotScript'
    | 'innerDocker'
    | 'matrixShellCommand'
  >;
const input = JSON.parse(bundleBytes.toString('utf8'));
assert.equal(hash(bundleBytes), '268d567e4d55575273d156f3f14a7f7ba14a7883384162a4cd4490896ed765e5');
assert.equal(input.descriptor.identity.appName, 'SodaRuntime');
assert.equal(
  input.descriptor.identity.artifactDigest,
  '502ef67c3c45a7158544bb18dce86c164d1b0ac6a474390ed530a4fb02604039'
);
assert.equal(input.bundle.mixedSha256, 'f012e5b7af968aef3a194131d6e0c973f9dbbcdea88a606405dc2f842f8d6fbf');
assert.equal(input.storageFingerprint, 'eb636eea007e384cd278d3b3be3a00c4ff76f1bde9fdeeb9d107a68b9248a73a');
const tools = JSON.parse(await readFile(path.join(path.dirname(runtime), '..', 'tools.json'), 'utf8'));
assert.equal(tools.sourceRevision, '228f83d47b8462dc6949143ebf619a7d81b8a8e0');
assert.equal(tools.dirty, false);
const env = {...process.env, TMPDIR: temporary};
let stage = 'bootstrap',
  live: Awaited<ReturnType<typeof start>> | undefined;
const retirements: {exit_code: number | null; signal: NodeJS.Signals | null; writer_lock_available: boolean}[] = [];
async function start(team: string) {
  return startLocalApplication({
    runtime,
    bundle,
    data,
    env,
    owner_team: team,
    retired: (result) => {
      retirements.push(result);
    },
  });
}
function operationID() {
  const bytes = randomBytes(16),
    time = Date.now();
  bytes.writeUIntBE(time, 0, 6);
  bytes[6] = (bytes[6]! & 0x0f) | 0x70;
  bytes[8] = (bytes[8]! & 0x3f) | 0x80;
  const hex = bytes.toString('hex');
  return [hex.slice(0, 8), hex.slice(8, 12), hex.slice(12, 16), hex.slice(16, 20), hex.slice(20)].join('-');
}

async function measure(owner: Awaited<ReturnType<typeof openApplicationOwner>>, witness: ApplicationRecordWitness) {
  // A real persistent PTY, exercising the same shell marker command as the
  // browser wire probe. No terminal emulator or provider replies are fabricated.
  const terminal = spawn(
    '/usr/bin/script',
    ['--quiet', '--flush', '--return', '--command', 'PS1=; export PS1; exec bash --noprofile --norc -i', '/dev/null'],
    {
      env: {...env, HOME: path.join(out, 'private'), HISTFILE: '/dev/null', TERM: 'xterm-256color'},
      stdio: ['pipe', 'pipe', 'pipe'],
    }
  );
  const closed = new Promise<void>((resolve) => terminal.once('close', () => resolve()));
  terminal.stderr.on('data', () => {});
  let output = '',
    expected: unknown,
    disposed = false;
  terminal.stdout.on('data', (bytes: Buffer) => {
    output = (output + bytes.toString()).slice(-32768);
  });
  const sampleTerminal = async () => {
    assert(!disposed && terminal.exitCode === null, 'measurement_terminal_not_live');
    output = '';
    const marker = 'SODA_FACT_' + randomBytes(16).toString('hex');
    terminal.stdin.write(matrixShellCommand(marker, 'Measurement PTY', expected === undefined) + '\n');
    const deadline = Date.now() + 15000;
    do {
      const line = output
        .replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '')
        .replaceAll('\r', '')
        .split('\n')
        .find((line) => line.startsWith(marker + ':'));
      if (line) {
        const facts = JSON.parse(line.slice(marker.length + 1));
        assert(
          facts.tty === true && facts.marker === 'Measurement PTY' && facts.pid > 0 && /^[0-9]+$/.test(facts.start)
        );
        if (expected === undefined) expected = facts;
        else assert.deepEqual(facts, expected);
        return;
      }
      await new Promise((resolve) => setTimeout(resolve, 25));
    } while (Date.now() < deadline);
    throw Error('measurement_real_pty_unconfirmed');
  };
  const config = JSON.parse(await readFile(args[9]!, 'utf8'));
  const program = innerDocker(measurementSnapshotScript(config.factory_phase));
  assert(/^[A-Za-z0-9][A-Za-z0-9._-]{0,128}$/.test(args[13]!));
  const nativeRead = async () => {
    const child = spawn(
      '/usr/bin/ssh',
      [
        '-T',
        '-F',
        args[11]!,
        '-oBatchMode=yes',
        '-oStrictHostKeyChecking=yes',
        '-oIdentityAgent=none',
        '-oControlMaster=no',
        '-oControlPath=none',
        '-oConnectTimeout=10',
        '-oForwardAgent=no',
        '-oForwardX11=no',
        '-oPermitLocalCommand=no',
        '-oClearAllForwardings=yes',
        args[13]!,
        '/usr/bin/node -',
      ],
      {stdio: ['pipe', 'pipe', 'pipe']}
    );
    let text = '';
    child.stdout.on('data', (bytes: Buffer) => {
      text += bytes.toString();
      if (text.length > 4 << 20) child.kill();
    });
    child.stderr.on('data', () => {});
    const timer = setTimeout(() => child.kill(), 210000);
    const done = new Promise<void>((resolve, reject) => {
      child.once('error', () => reject(Error('measurement_native_transport_unavailable')));
      child.once('close', (code) => (code === 0 ? resolve() : reject(Error('measurement_native_read_unconfirmed'))));
    });
    child.stdin.end(program);
    try {
      await done;
      return JSON.parse(text);
    } finally {
      clearTimeout(timer);
    }
  };
  let measurement: Awaited<ReturnType<typeof startJourneyMeasurements>> | undefined;
  try {
    measurement = await startJourneyMeasurements({
      application: {owner, witness},
      terminal: {
        sample: sampleTerminal,
        dispose() {
          if (!disposed) {
            disposed = true;
            terminal.stdin.end('exit\n');
          }
        },
      },
      save,
    });
    // Both are real, read-only observations of the retained appliance while the
    // local App and PTY remain alive. No Factory trial/policy/action is sent.
    await measurement.observeSnapshot(await nativeRead());
    await measurement.observeSnapshot(await nativeRead());
    const result = await measurement.finish();
    assert(result.observed && result.native_samples === 2 && result.response_samples === 3);
    await save('measurement-scope.json', {
      format: 1,
      scope: 'Local qualified App and PTY with live remote native observation; no Factory execution',
      terminal: expected,
      factory_lifecycle: {ci: null, review: null, delivery: null},
      guest_mutations: 0,
      provider_calls: 0,
    });
  } finally {
    if (measurement) await measurement.finish();
    else terminal.stdin.end('exit\n');
    const timer = setTimeout(() => terminal.kill('SIGTERM'), 5000);
    try {
      await closed;
    } finally {
      clearTimeout(timer);
    }
    assert.equal(terminal.exitCode, 0, 'measurement_terminal_retirement_unconfirmed');
  }
}
try {
  const bootstrap = spawnSync(
    runtime,
    ['bootstrap', '--bundle', bundle, '--data', data, '--credentials', credentialFile],
    {env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], timeout: 30000, maxBuffer: 65536}
  );
  assert.equal(bootstrap.status, 0, 'application_bootstrap_failed');
  const identity = JSON.parse(bootstrap.stdout);
  assert(
    identity.ok === true &&
      identity.command === 'bootstrap' &&
      typeof identity.teamId === 'string' &&
      typeof identity.userId === 'string',
    'application_bootstrap_unconfirmed'
  );
  await save('owner.public.json', {
    format: 1,
    team_id: identity.teamId,
    user_id: identity.userId,
    credentials_file: credentialFile,
  });
  stage = 'native-start';
  live = await start(identity.teamId);
  stage = 'owner-login';
  let owner = await openApplicationOwner({origin: live.origin, owner_team: identity.teamId, credentials});
  const journal = {
    save,
    load: async (relative: string) => {
      try {
        return JSON.parse(await readFile(path.join(out, relative), 'utf8'));
      } catch (error) {
        if ((error as NodeJS.ErrnoException).code === 'ENOENT') return null;
        throw error;
      }
    },
  };
  stage = 'create-record';
  const selected = {operation_id: operationID(), label: 'Public appliance persisted record'};
  const witness: ApplicationRecordWitness = await owner.ensureRecord(selected, journal);
  await save('application/record.json', witness);
  stage = 'record-and-rendered-read';
  await owner.readRecord(witness);
  const rendered = await owner.readRendered(witness);
  if (measuring) {
    stage = 'coordinated-measurements';
    await measure(owner, witness);
  }
  stage = 'anonymous-denial';
  const anonymous = await fetch(live.origin + '/api/operations/SodaRuntime.Entry.create', {
    method: 'POST',
    redirect: 'manual',
    signal: AbortSignal.timeout(20000),
    headers: {'content-type': 'application/json', origin: live.origin},
    body: JSON.stringify({operation_id: selected.operation_id, inputs: {label: 'Forbidden change'}}),
  });
  assert.equal(anonymous.status, 403, 'application_anonymous_write_not_denied');
  stage = 'native-stop';
  await live.stop();
  live = undefined;
  stage = 'native-restart';
  live = await start(identity.teamId);
  stage = 'same-owner-login';
  owner = await openApplicationOwner({origin: live.origin, owner_team: identity.teamId, credentials});
  stage = 'retained-record';
  await owner.readRecord(witness);
  await owner.readRendered(witness);
  // Re-enter the same journal after native restart: reconcile the original
  // record and intent without issuing another create operation.
  assert.deepEqual(await owner.ensureRecord(selected, journal), witness);
  stage = 'final-retirement';
  await live.stop();
  live = undefined;
  await save('result.json', {
    format: 1,
    passed: true,
    check: 'actual-native-application-owner-record',
    scope: 'Local qualified runtime and RC7 fixture; no installed guest or release qualification',
    runtime: {path: runtime, sha256: hash(runtimeBytes), tools_source: tools.sourceRevision},
    module: {path: moduleFile, sha256: hash(moduleBytes)},
    bundle: {
      path: bundle,
      sha256: hash(bundleBytes),
      artifact: input.descriptor.identity.artifactDigest,
      bundle: input.bundle.mixedSha256,
      storage: input.storageFingerprint,
    },
    credentials_file: credentialFile,
    owner_team: identity.teamId,
    record: witness,
    rendered,
    anonymous_status: anonymous.status,
    retirements,
    same_owner_and_record_after_restart: true,
    create_reentry_read_only: true,
    credentials_or_tokens_exported: false,
    guest_mutations: 0,
    provider_calls: 0,
  });
  console.log(JSON.stringify({passed: true, result: path.join(out, 'result.json')}));
} catch {
  await save('failure.public.json', {
    format: 1,
    passed: false,
    stage,
    credentials_file: credentialFile,
    diagnostics_exported: false,
    guest_mutations: 0,
    provider_calls: 0,
  });
  throw Error('application_check_failed_at_' + stage);
} finally {
  await live?.stop();
}
