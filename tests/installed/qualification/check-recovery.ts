// Actual local operator recovery boundaries. The deliberately absent driver,
// authority and candidate cannot dispatch a worker or reach an appliance.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {constants} from 'node:fs';
import {access, chmod, copyFile, lstat, mkdir, readFile, realpath, writeFile} from 'node:fs/promises';
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
assert(source.isFile() && (source.mode & 0o111) !== 0);
const sha = (bytes: Uint8Array | string) => createHash('sha256').update(bytes).digest('hex');
async function trusted(directory: string) {
  for (let current = directory; ; current = path.dirname(current)) {
    const stat = await lstat(current);
    assert(stat.isDirectory() && stat.uid === 0 && (stat.mode & 0o022) === 0);
    if (current === '/') break;
  }
}
async function directory(file: string) {
  await mkdir(file, {mode: 0o700});
  await trusted(file);
}
async function write(file: string, value: unknown, mode = 0o600) {
  await writeFile(file, JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode});
  await chmod(file, mode);
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
  return {stdout, stderr, code};
}
await trusted('/home');
await directory(out);
const selected = path.join(out, 'soda-release');
await copyFile(operator, selected, constants.COPYFILE_EXCL);
await chmod(selected, 0o700);
const operatorSHA = sha(await readFile(selected));
assert.equal(operatorSHA, sha(await readFile(operator)));
assert.equal((await lstat(selected)).nlink, 1);
const unavailableDriver = {path: path.join(out, 'driver-deliberately-unavailable'), sha256: '0'.repeat(64)};
await absent(unavailableDriver.path);
async function runInput(name: string) {
  const run = path.join(out, name);
  await directory(run);
  const scenario = path.join(run, 'scenario.json');
  await write(scenario, {driver: {executable: unavailableDriver}}, 0o444);
  const scenarioReference = {path: scenario, sha256: sha(await readFile(scenario))};
  await write(
    path.join(run, 'request.json'),
    {format: 1, driver: unavailableDriver, scenario: scenarioReference},
    0o444
  );
  return {run, scenarioReference};
}
const outcomes: {name: string; state: string; reason?: string}[] = [];
const admitted = await runInput('retained-ready-admission');
const admission = path.join(admitted.run, 'run-admission.json');
await write(admission, {format: 1, state: 'ready', held: false, approval_state: 'approved', blockers: []});
const admissionSHA = sha(await readFile(admission));
const response = await execute([selected, 'journey', 'run', '--run', admitted.run]);
assert.equal(response.code, 0, response.stderr);
const uncertain = JSON.parse(response.stdout).recovery;
assert.equal(uncertain.state, 'uncertain');
assert.equal(uncertain.reason, 'admission_retained_without_dispatch_intent');
assert.equal(uncertain.automatic_actions, false);
assert.equal(sha(await readFile(admission)), admissionSHA);
await absent(path.join(admitted.run, 'collector-intent.json'));
outcomes.push({name: 'retained-ready-admission', state: uncertain.state, reason: uncertain.reason});
const admittedStatus = await execute([selected, 'journey', 'status', '--run', admitted.run]);
assert.equal(admittedStatus.code, 0, admittedStatus.stderr);
assert.deepEqual(JSON.parse(admittedStatus.stdout).recovery, uncertain);
const defective = await runInput('driver-unavailable-status');
const driverStatus = await execute([selected, 'journey', 'status', '--run', defective.run]);
assert.equal(driverStatus.code, 0, driverStatus.stderr);
const defect = JSON.parse(driverStatus.stdout).recovery;
assert.equal(defect.state, 'uncertain');
assert.equal(defect.reason, 'retained_driver_unavailable_or_changed');
assert.equal(typeof defect.next_action, 'string');
outcomes.push({name: 'driver-unavailable-status', state: defect.state, reason: defect.reason});

const retained = await runInput('retained-intent-no-request');
async function reference(name: string) {
  const file = path.join(retained.run, name);
  await write(file, {local_recovery_fixture: name});
  return {path: file, sha256: sha(await readFile(file))};
}
const approval = await reference('approval.json'),
  build = await reference('build.json'),
  resolved = await reference('resolved-input.json'),
  collectionOutput = path.join(retained.run, 'absent-collection');
const intent = path.join(retained.run, 'collector-intent.json');
await write(intent, {
  format: 1,
  scenario: retained.scenarioReference,
  approval,
  build_receipt: build,
  resolved_input: resolved,
  collector_sha256: '1'.repeat(64),
  collection: {
    authority_file: path.join(retained.run, 'absent-authority.json'),
    candidate_directory: path.join(retained.run, 'absent-candidate'),
    output_directory: collectionOutput,
    build_receipt_file: build.path,
  },
});
const intentSHA = sha(await readFile(intent));
for (const action of ['run', 'status']) {
  const result = await execute([selected, 'journey', action, '--run', retained.run]);
  assert.equal(result.code, 0, result.stderr);
  const recovery = JSON.parse(result.stdout).recovery;
  assert.equal(recovery.state, 'uncertain');
  assert.equal(recovery.reason, 'dispatch_intent_retained_without_collector_request');
  assert.equal(recovery.automatic_actions, false);
  assert.equal(sha(await readFile(intent)), intentSHA);
  await absent(collectionOutput);
  await absent(path.join(retained.run, 'collector-binding.json'));
  outcomes.push({name: `retained-intent-${action}-driver-unavailable`, state: recovery.state, reason: recovery.reason});
}

const lock = path.join(admitted.run, 'journey-controller.lock');
const holder = Bun.spawn(['/usr/bin/flock', '--no-fork', '-n', lock, '/usr/bin/sleep', '60'], {
  stdin: 'ignore',
  stdout: 'ignore',
  stderr: 'ignore',
});
try {
  const deadline = Date.now() + 3000;
  let acquired = false;
  while (Date.now() < deadline) {
    assert.equal(holder.exitCode, null, 'lock holder exited before acquisition');
    const probe = await execute(['/usr/bin/flock', '-n', lock, '/usr/bin/true']);
    assert(probe.code === 0 || probe.code === 1);
    if (probe.code === 1) {
      acquired = true;
      break;
    }
    await Bun.sleep(20);
  }
  assert(acquired && holder.exitCode === null);
  const blocked = await execute([selected, 'journey', 'run', '--run', admitted.run]);
  assert.notEqual(blocked.code, 0);
  assert(blocked.stderr.includes('journey controller already active'));
  assert.equal(blocked.stdout, '');
  outcomes.push({name: 'live-run-lock', state: 'concurrent_controller_refused'});
} finally {
  if (holder.exitCode === null) holder.kill('SIGTERM');
  await holder.exited;
}
const result = {
  format: 1,
  operator_sha256: operatorSHA,
  integration_sha256: sha(await readFile(import.meta.path)),
  outcomes,
  collector_requests: 0,
  guest_effects: 0,
  model_calls: 0,
};
await write(path.join(out, 'result.json'), result);
console.log(JSON.stringify(result));
