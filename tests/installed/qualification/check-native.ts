// Read-only integration of the built native observer with retained Factory history.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdir, readFile, realpath, stat, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {pinnedSSH, scenarioInput} from './validate';

process.umask(0o077);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--module', '--scenario', '--repository', '--issue', '--out']
);
assert.equal(args.length, 10, 'explicit built module, scenario, historical repository/issue and fresh output required');
const [moduleFile, scenarioFile, repository, issue, out] = [
  args[1]!,
  args[3]!,
  Number(args[5]),
  Number(args[7]),
  args[9]!,
];
for (const n of [repository, issue]) assert(Number.isSafeInteger(n) && n > 0);
for (const file of [moduleFile, scenarioFile]) {
  assert(path.isAbsolute(file) && (await realpath(file)) === file);
  assert((await stat(file)).isFile());
}
assert(out.startsWith('/home/') && path.resolve(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out));
const hash = (bytes: Uint8Array) => createHash('sha256').update(bytes).digest('hex');
const moduleSHA = hash(await readFile(moduleFile));
const scenarioBytes = await readFile(scenarioFile);
const scenario = scenarioInput(JSON.parse(scenarioBytes.toString('utf8')));
assert.equal(scenario.target.distribution, 'docker');
assert.equal(scenario.factory.length, 1);
const connection = scenario.factory[0]!.subject.connection_id;
assert(/^[a-f0-9]{32}$/.test(connection));
const argv = await pinnedSSH(scenario);
const built = (await import(pathToFileURL(moduleFile).href)) as Pick<
  typeof import('./phases/index'),
  'innerDocker' | 'nativeExecutionScript' | 'nativeExecutionSucceeded'
>;
await mkdir(out, {mode: 0o700});
function execute(program: string) {
  const child = Bun.spawnSync(argv, {
    stdin: Buffer.from(built.innerDocker(program)),
    stdout: 'pipe',
    stderr: 'pipe',
    timeout: 90000,
    maxBuffer: 1 << 20,
  });
  assert.equal(child.exitCode, 0, 'selected native metadata unavailable');
  return JSON.parse(child.stdout.toString());
}
// Fixed allowlisted metadata. No run DTO, provider output, reason or secret is
// selected. Native Inspect is likewise projected inside the guest by the bundle.
const query =
  `const REPOSITORY=${repository}, ISSUE=${issue}, CONNECTION=${JSON.stringify(connection)};\n` +
  String.raw`
const cp=require('node:child_process');
const q="BEGIN READ ONLY; SELECT json_build_object('history',json_build_object('runs',(SELECT count(*) FROM factory_runs),'charged_minutes',(SELECT coalesce(sum(minutes),0) FROM factory_usage)),'connection',(SELECT json_build_object('id',id,'state',state,'generation',generation) FROM identity_connections WHERE id='"+CONNECTION+"'),'rows',(SELECT coalesce(json_agg(json_build_object('repository',a.repository,'issue',a.issue,'assignment',a.id,'project',a.data->>'project_id','role',a.data->>'role','phase',a.data->>'phase','outcome',a.data->>'outcome','run',json_build_object('id',r.id,'role',r.data->>'role','generation',r.data->'identity_generation','lease',r.data->>'identity_lease_id','binding',json_build_object('invocation_id',r.data#>>'{identity_binding,invocation_id}'))) ORDER BY a.id),'[]') FROM factory_assignments a JOIN factory_runs r ON a.run=r.id WHERE a.repository="+REPOSITORY+" AND a.issue="+ISSUE+")); COMMIT;";
console.log(cp.execFileSync('/usr/bin/podman',['--remote=false','exec','soda-postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-U','postgres','-d','soda','-c',q],{encoding:'utf8',timeout:20000,maxBuffer:1<<20,stdio:['ignore','pipe','pipe']}).trim());`;
const before = execute(query);
assert.equal(before.connection?.state, 'ready');
assert(Number.isSafeInteger(before.connection.generation) && before.connection.generation > 0);
assert(Array.isArray(before.rows) && before.rows.length > 0);
const observations = [];
for (const assignment of before.rows) {
  const selected = {
    repository: {id: assignment.repository, project: assignment.project},
    issue: {number: assignment.issue},
  };
  const native = execute(built.nativeExecutionScript(selected, assignment.run.id));
  const accepted = built.nativeExecutionSucceeded(native, assignment.run, assignment, native.container);
  assert(!Object.hasOwn(native, 'output') && !Object.hasOwn(native, 'reason'));
  if (accepted) {
    assert.equal(assignment.outcome, 'succeeded');
    assert(
      before.connection.generation > assignment.run.generation,
      'retained successful generations must precede the current Ready generation'
    );
  }
  observations.push({
    assignment: assignment.assignment,
    phase: assignment.phase,
    recorded_outcome: assignment.outcome,
    recorded_generation: assignment.run.generation,
    native,
    accepted_as_native_success: accepted,
  });
}
assert(
  observations.some((row) => row.accepted_as_native_success),
  'retained native success required'
);
assert(
  observations.some((row) => !row.accepted_as_native_success),
  'retained unsuccessful native observation required'
);
assert.deepEqual(execute(query), before, 'history, charges or connection changed during read-only integration');
assert.equal(hash(await readFile(moduleFile)), moduleSHA);
assert.equal(hash(await readFile(scenarioFile)), hash(scenarioBytes));
await writeFile(path.join(out, 'native-observations.json'), JSON.stringify(observations, null, 2) + '\n', {
  flag: 'wx',
  mode: 0o600,
});
const result = {
  format: 1,
  check: 'retained-native-execution',
  passed: true,
  module_sha256: moduleSHA,
  scenario_sha256: hash(scenarioBytes),
  historical_repository: repository,
  historical_issue: issue,
  real_retired_runs: observations.length,
  native_successes: observations.filter((row) => row.accepted_as_native_success).length,
  native_unsuccessful: observations.filter((row) => !row.accepted_as_native_success).length,
  recorded_generations: observations.map((row) => row.recorded_generation),
  current_generation: before.connection.generation,
  history: before.history,
  connection_still_ready: true,
  history_and_usage_unchanged: true,
  raw_provider_output_exported: false,
  guest_mutations: 0,
  provider_calls: 0,
  qualification_claimed: false,
};
await writeFile(path.join(out, 'result.json'), JSON.stringify(result, null, 2) + '\n', {flag: 'wx', mode: 0o600});
console.log(JSON.stringify(result));
