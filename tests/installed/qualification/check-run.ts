// Provider-free admission integration of the built driver and Rust operator.
// Local approval fixtures never authorize execution against the appliance.
import assert from 'node:assert/strict';
import {access, mkdir, readFile, realpath, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {approvalScope} from './run';
import {scenarioInput, sha256} from './validate';

process.umask(0o077);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--operator', '--build', '--scenario', '--readiness', '--out']
);
assert.equal(args.length, 10);
const [operator, buildFile, scenarioFile, readinessFile, out] = [args[1]!, args[3]!, args[5]!, args[7]!, args[9]!];
for (const file of [operator, buildFile, scenarioFile, readinessFile]) {
  assert(file.startsWith('/home/') && path.resolve(file) === file);
  assert.equal(await realpath(file), file);
}
assert(out.startsWith('/home/') && path.resolve(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out));
const buildBytes = await readFile(buildFile),
  scenarioBytes = await readFile(scenarioFile),
  savedBytes = await readFile(readinessFile);
const build = JSON.parse(buildBytes.toString());
assert(build.format === 1 && build.source_clean === true && /^[a-f0-9]{40}$/.test(build.source_revision));
const selected = build.outputs.filter((o: {path: string}) => path.basename(o.path) === 'soda-qualification-journey');
assert.equal(selected.length, 1);
const driver = selected[0] as {path: string; sha256: string};
assert.equal(sha256(await readFile(driver.path)), driver.sha256);
const original = scenarioInput(JSON.parse(scenarioBytes.toString()));
const saved = JSON.parse(savedBytes.toString());
assert(saved.format === 1 && saved.observation && saved.native?.history && original.factory.length > 0);
const base = scenarioInput({
  ...original,
  driver: {source_revision: build.source_revision, executable: {path: driver.path, sha256: driver.sha256}},
  authority: {...original.authority, held: true},
});
// The receipt was scoped before selecting the rebuilt tool. Its scope must stay
// unchanged even when the driver revision, digest and executable location move.
assert.deepEqual(approvalScope(original), approvalScope(base));
await mkdir(out, {mode: 0o700});
const write = (file: string, value: unknown) =>
  writeFile(file, JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode: 0o600});
const read = async (file: string) => JSON.parse(await readFile(file, 'utf8'));
async function absent(file: string) {
  await assert.rejects(access(file), (error: unknown) => (error as NodeJS.ErrnoException).code === 'ENOENT');
}
async function execute(argv: string[], success = true) {
  const child = Bun.spawn(argv, {stdin: 'ignore', stdout: 'pipe', stderr: 'pipe'});
  const [stdout] = await Promise.all([new Response(child.stdout).text(), new Response(child.stderr).text()]);
  const exitCode = await child.exited;
  if (!success) {
    assert.notEqual(exitCode, 0);
    return null;
  }
  assert.equal(exitCode, 0, 'selected admission command failed');
  return JSON.parse(stdout);
}
const actualScenario = path.join(out, 'scenario.json');
await write(actualScenario, base);
const actualRun = path.join(out, 'operator-check');
const checked = await execute([operator, 'journey', 'check', '--scenario', actualScenario, '--out', actualRun]);
const actual = await read(path.join(actualRun, 'readiness.json'));
assert.deepEqual(actual.native.history, saved.native.history, 'retained history or charges changed');
const generations = (r: typeof actual) =>
  r.observation.factory.map((f: {connection: {id: string; state: string; generation: number} | null}) => f.connection);
assert.deepEqual(generations(actual), generations(saved), 'retained Ready connection changed');
const ran = await execute([operator, 'journey', 'run', '--run', actualRun]);
assert.equal(ran.execution.state, 'held');
assert.equal(ran.execution.held, true);
const status = await execute([operator, 'journey', 'status', '--run', actualRun]);
assert.deepEqual(status.execution, ran.execution);
assert.deepEqual(status.accounting, checked.accounting);
await absent(path.join(actualRun, 'resolved-input.json'));
const admissionBytes = await readFile(path.join(actualRun, 'run-admission.json'));
const repeated = await execute([operator, 'journey', 'run', '--run', actualRun]);
assert.deepEqual(repeated.execution, ran.execution);
assert.deepEqual(await readFile(path.join(actualRun, 'run-admission.json')), admissionBytes);

const scope = approvalScope(original);
const receipts = {approved: path.join(out, 'approved.fixture.json'), revoked: path.join(out, 'revoked.fixture.json')};
for (const state of ['approved', 'revoked'] as const)
  await write(receipts[state], {
    format: 1,
    state,
    approval_reference: 'local integration fixture only; not owner authority',
    scope,
  });
const variants = [
  {
    name: 'matching-held-rebuilt-driver',
    held: true,
    receipt: 'approved',
    changed: '',
    state: 'held',
    approval: 'approved',
  },
  {
    name: 'changed-limit-held',
    held: true,
    receipt: 'approved',
    changed: 'limit',
    state: 'held',
    approval: 'unvalidated',
  },
  {
    name: 'changed-target-held',
    held: true,
    receipt: 'approved',
    changed: 'target',
    state: 'held',
    approval: 'unvalidated',
  },
  {
    name: 'changed-lifecycle-held',
    held: true,
    receipt: 'approved',
    changed: 'lifecycle',
    state: 'held',
    approval: 'unvalidated',
  },
  {name: 'revoked-held', held: true, receipt: 'revoked', changed: '', state: 'held', approval: 'revoked'},
  {name: 'revoked-unheld', held: false, receipt: 'revoked', changed: '', state: 'revoked', approval: 'revoked'},
  {
    name: 'changed-limit-unheld',
    held: false,
    receipt: 'approved',
    changed: 'limit',
    state: 'blocked',
    approval: 'unvalidated',
  },
] as const;
const results = [];
for (const variant of variants) {
  const directory = path.join(out, variant.name);
  await mkdir(directory, {mode: 0o700});
  const value = structuredClone(base) as any;
  value.authority = {held: variant.held, scope_receipt_file: receipts[variant.receipt]};
  if (variant.changed === 'limit') value.factory[0].limits.rolling_minutes++;
  if (variant.changed === 'target')
    value.target.uuid = value.target.uuid.slice(0, -1) + (value.target.uuid.endsWith('0') ? '1' : '0');
  if (variant.changed === 'lifecycle') {
    assert(value.permitted_effects.project_lifecycle.length > 0);
    value.permitted_effects.project_lifecycle = value.permitted_effects.project_lifecycle.slice(1);
  }
  scenarioInput(value);
  await write(path.join(directory, 'scenario.json'), value);
  await write(path.join(directory, 'readiness.json'), saved);
  const response = await execute([driver.path, 'run', '--run', directory]);
  assert.equal(response.execution.state, variant.state);
  assert.equal(response.execution.approval_state, variant.approval);
  assert.equal(response.execution.held, variant.held);
  const view = await execute([driver.path, 'status', '--run', directory]);
  assert.deepEqual(view.execution, response.execution);
  await absent(path.join(directory, 'resolved-input.json'));
  const retained = await readFile(path.join(directory, 'run-admission.json'));
  const repeated = await execute([driver.path, 'run', '--run', directory]);
  assert.deepEqual(repeated.execution, response.execution);
  assert.deepEqual(await readFile(path.join(directory, 'run-admission.json')), retained);
  results.push({
    name: variant.name,
    state: variant.state,
    approval: variant.approval,
    admission_sha256: sha256(retained),
  });
}
// An interrupted admission is a real filesystem boundary, not permission to
// overwrite its saved read or reconstruct an effect. Use the actual saved
// appliance observation as input; never fabricate a successful product result.
const interruptedRun = path.join(out, 'interrupted-admission');
await mkdir(interruptedRun, {mode: 0o700});
await write(path.join(interruptedRun, 'scenario.json'), base);
await write(path.join(interruptedRun, 'readiness.json'), saved);
await write(path.join(interruptedRun, 'run-readiness.json'), saved);
const interruptedBytes = await readFile(path.join(interruptedRun, 'run-readiness.json'));
await execute([driver.path, 'run', '--run', interruptedRun], false);
const interruptedStatus = await execute([driver.path, 'status', '--run', interruptedRun]);
assert.equal(interruptedStatus.recovery.reason, 'admission_interrupted');
assert.equal(interruptedStatus.recovery.automatic_actions, false);
assert.deepEqual(await readFile(path.join(interruptedRun, 'run-readiness.json')), interruptedBytes);
await absent(path.join(interruptedRun, 'run-admission.json'));
await absent(path.join(interruptedRun, 'resolved-input.json'));
assert.equal(sha256(await readFile(driver.path)), driver.sha256);
assert.equal(sha256(await readFile(scenarioFile)), sha256(scenarioBytes));
assert.equal(sha256(await readFile(readinessFile)), sha256(savedBytes));
const result = {
  format: 1,
  passed: true,
  check: 'guarded-run-admission',
  operator: {path: operator, sha256: sha256(await readFile(operator))},
  build: {path: buildFile, sha256: sha256(buildBytes)},
  driver_source_revision: build.source_revision,
  scenario: {path: scenarioFile, sha256: sha256(scenarioBytes)},
  saved_readiness: {path: readinessFile, sha256: sha256(savedBytes)},
  actual_operator_run: actualRun,
  retained_history: actual.native.history,
  ready_connections: generations(actual),
  actual_held_dispatch_refused: true,
  repeated_admissions_return_original_without_dispatch: true,
  interrupted_admission_preserved_without_replay: true,
  fixtures_are_owner_authority: false,
  cases: results,
  resolved_inputs_created: 0,
  provider_calls: 0,
  guest_mutations: 0,
  qualification_claimed: false,
};
await write(path.join(out, 'result.json'), result);
console.log(JSON.stringify(result));
