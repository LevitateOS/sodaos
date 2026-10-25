// Built command reconciliation against a real retained policy record. The send
// callback deliberately loses its reply WITHOUT dispatching a guest mutation.
import assert from 'node:assert/strict';
import {mkdir, open, readFile, realpath, stat} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {pinnedSSH, scenarioInput, sha256} from './validate';
import type {FactoryCommandIntent, FactoryCommandRuntime} from './phases/factory-command';

process.umask(0o077);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--module', '--scenario', '--out']
);
assert.equal(args.length, 6, 'explicit built module, scenario and fresh /home output required');
const [moduleFile, scenarioFile, out] = [args[1]!, args[3]!, args[5]!];
for (const file of [moduleFile, scenarioFile]) {
  assert(path.isAbsolute(file) && (await realpath(file)) === file && (await stat(file)).isFile());
}
assert(out.startsWith('/home/') && path.resolve(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out));
const moduleSHA = sha256(await readFile(moduleFile)),
  scenarioBytes = await readFile(scenarioFile);
const scenario = scenarioInput(JSON.parse(scenarioBytes.toString('utf8')));
assert.equal(scenario.target.distribution, 'docker');
assert.equal(scenario.factory.length, 1);
const subject = scenario.factory[0]!.subject,
  repository = Number(subject.repository_id);
assert(Number.isSafeInteger(repository) && repository > 0);
const argv = await pinnedSSH(scenario);
const built = (await import(pathToFileURL(moduleFile).href)) as Pick<
  typeof import('./phases/index'),
  'innerDocker' | 'issueFactoryCommand'
>;
await mkdir(out, {mode: 0o700});

function observe(commandID: string | null = null) {
  assert(commandID === null || /^[a-f0-9]{32}$/.test(commandID));
  const program =
    `const REPOSITORY=${repository},CONNECTION=${JSON.stringify(subject.connection_id)},COMMAND=${JSON.stringify(commandID)};\n` +
    String.raw`
const cp=require('node:child_process');
const selected="type='policy' AND target='repository/"+REPOSITORY+"/policy'"+(COMMAND===null?" AND finished<>''":" AND id='"+COMMAND+"'");
const q="BEGIN READ ONLY; SELECT json_build_object('history',json_build_object('runs',(SELECT count(*) FROM factory_runs),'charged_minutes',(SELECT coalesce(sum(minutes),0) FROM factory_usage)),'connection',(SELECT json_build_object('id',id,'state',state,'generation',generation) FROM identity_connections WHERE id='"+CONNECTION+"'),'command',(SELECT row_to_json(c) FROM (SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands WHERE "+selected+" ORDER BY created DESC,id DESC LIMIT 1) c)); COMMIT;";
console.log(cp.execFileSync('/usr/bin/podman',['--remote=false','exec','soda-postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-U','postgres','-d','soda','-c',q],{encoding:'utf8',timeout:20000,maxBuffer:65536,stdio:['ignore','pipe','pipe']}).trim());`;
  const child = Bun.spawnSync(argv, {
    stdin: Buffer.from(built.innerDocker(program)),
    stdout: 'pipe',
    stderr: 'pipe',
    timeout: 90000,
    maxBuffer: 65536,
  });
  assert.equal(child.exitCode, 0, 'retained command metadata unavailable');
  return JSON.parse(child.stdout.toString());
}
async function save(relative: string, value: unknown) {
  assert(
    /^[A-Za-z0-9._/-]{1,256}$/.test(relative) &&
      !relative.startsWith('/') &&
      relative.split('/').every((p) => p !== '' && p !== '.' && p !== '..')
  );
  const file = path.join(out, relative),
    directory = path.dirname(file);
  await mkdir(directory, {recursive: true, mode: 0o700});
  const handle = await open(file, 'wx', 0o600);
  try {
    await handle.writeFile(JSON.stringify(value, null, 2) + '\n');
    await handle.sync();
  } finally {
    await handle.close();
  }
  const dir = await open(directory, 'r');
  try {
    await dir.sync();
  } finally {
    await dir.close();
  }
}
const before = observe(),
  record = before.command;
assert.equal(before.connection?.state, 'ready');
assert(Number.isSafeInteger(before.connection.generation) && before.connection.generation > 0);
assert(record && /^[a-f0-9]{32}$/.test(record.id) && record.type === 'policy' && record.finished !== '');
assert.equal(record.target, 'repository/' + repository + '/policy');
const payload = JSON.parse(record.payload);
assert.equal(payload.repository, subject.repository_id);
assert.equal(record.principal, 'native:' + payload.granted_by);
assert(Number.isSafeInteger(payload.revision) && payload.revision >= 0);
const {repository: _repository, revision, granted_by: owner, ...policy} = payload;
for (const key of ['publish_actor', 'create_actor', 'review_actor', 'resolve_actor', 'merge_actor']) {
  if (policy[key]) policy[key] = {actor_id: policy[key].actor_id, token_id: policy[key].token_id};
}
const intent: FactoryCommandIntent = {
  label: 'retained-policy',
  repository,
  owner,
  method: 'PUT',
  path: 'repositories/' + repository + '/factory/policy',
  body: {...policy, command_id: record.id, expected_revision: revision},
};
await save('canonical-command.json', record);
let sends = 0,
  lookups = 0;
const runtime = (selected: FactoryCommandIntent, mode: 'retained' | 'pending' | 'missing'): FactoryCommandRuntime => ({
  prepare: () => save('commands/' + selected.label + '.request.json', selected),
  send: async () => {
    assert.deepEqual(
      JSON.parse(await readFile(path.join(out, 'commands/' + selected.label + '.request.json'), 'utf8')),
      selected
    );
    sends++;
    throw Error('deliberate_lost_reply_no_guest_dispatch');
  },
  lookup: async (requestPath) => {
    assert.equal(requestPath, 'factory/commands/' + record.id);
    lookups++;
    const actual = observe(record.id);
    assert.deepEqual(actual, before, 'canonical command/history changed during lookup');
    // Explicit local response variants; the real retained row is preserved.
    return mode === 'missing'
      ? {status: 404, body: {}}
      : {status: 200, body: mode === 'pending' ? {...actual.command, finished: ''} : actual.command};
  },
  save,
});
const recovered = await built.issueFactoryCommand(intent, runtime(intent, 'retained'));
assert.deepEqual(recovered, JSON.parse(record.outcome));
const receipt = JSON.parse(await readFile(path.join(out, 'commands/retained-policy.receipt.json'), 'utf8'));
assert.equal(receipt.recovered, true);
assert.equal(receipt.binding.id, record.id);
assert.equal(sends, 1);
assert.equal(lookups, 1);
// Re-entry cannot replace the already durable intent or reach send again.
await assert.rejects(built.issueFactoryCommand(intent, runtime(intent, 'retained')));
assert.equal(sends, 1);
assert.equal(lookups, 1);
for (const variant of ['changed-request', 'pending', 'missing'] as const) {
  const selected = {...intent, label: variant, body: {...intent.body}};
  if (variant === 'changed-request') selected.body.paused = !selected.body.paused;
  await assert.rejects(
    built.issueFactoryCommand(selected, runtime(selected, variant === 'changed-request' ? 'retained' : variant)),
    /factory_command_requires_reconciliation/
  );
  const uncertain = JSON.parse(await readFile(path.join(out, 'commands/' + variant + '.uncertain.json'), 'utf8'));
  assert.equal(uncertain.automatic_retry, false);
}
assert.equal(sends, 4);
assert.equal(lookups, 4);
assert.deepEqual(observe(), before, 'history, charge, connection or command changed during read-only integration');
assert.equal(sha256(await readFile(moduleFile)), moduleSHA);
assert.equal(sha256(await readFile(scenarioFile)), sha256(scenarioBytes));
const result = {
  format: 1,
  check: 'retained-command-native-reconciliation',
  passed: true,
  module_sha256: moduleSHA,
  scenario_sha256: sha256(scenarioBytes),
  repository,
  command_id: record.id,
  command_digest: record.digest,
  canonical_record_sha256: sha256(Buffer.from(JSON.stringify(record))),
  durable_request_before_send: true,
  recovered_canonical_outcome: true,
  existing_intent_refused_before_send: true,
  changed_request_pending_and_404_stop_without_retry: true,
  deliberate_lost_reply_callbacks: sends,
  exact_native_record_lookups: lookups,
  history: before.history,
  current_generation: before.connection.generation,
  history_usage_and_ready_connection_unchanged: true,
  scope: 'observational retained-record integration; no live mutation dispatched or normal HTTP API acceptance claimed',
  guest_mutations: 0,
  provider_calls: 0,
  qualification_claimed: false,
};
await save('result.json', result);
console.log(JSON.stringify(result));
