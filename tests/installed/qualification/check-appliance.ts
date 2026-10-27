// Real provider-free native transport/install/inventory integration. This reads
// the retained installed appliance; no guest lifecycle action or fake provider.
import assert from 'node:assert/strict';
import {mkdir, readFile, realpath, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {pinnedSSH, scenarioInput, sha256} from './validate';
import {checkScenario} from './readiness';
import {journeyStatus} from './status';
import {
  loadApplianceBinding,
  observeApplianceBoot,
  applianceProgram,
  readAppliance,
  type ApplianceInstallation,
} from './phases/appliance-native';
import type {DockerIO} from './phases/docker-binding';

process.umask(0o077);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--scenario', '--installation', '--out']
);
assert.equal(args.length, 6);
const [scenarioFile, installationFile, out] = [args[1]!, args[3]!, args[5]!];
for (const file of [scenarioFile, installationFile])
  assert(file.startsWith('/home/') && (await realpath(file)) === file);
assert(out.startsWith('/home/') && path.resolve(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out));
const scenarioBytes = await readFile(scenarioFile),
  installationBytes = await readFile(installationFile);
const s = scenarioInput(JSON.parse(scenarioBytes.toString()));
assert(s.target.distribution === 'appliance' && s.authority.held, 'only held native observation is exercised');
const installation = JSON.parse(installationBytes.toString()) as ApplianceInstallation;
const binding = await loadApplianceBinding(s, installation);
// Real public files remain available at admitted worker paths even though the
// controller-local manifest path is absent inside that worker namespace.
await loadApplianceBinding(
  {
    ...s,
    candidate: {
      ...s.candidate,
      release_manifest: {...s.candidate.release_manifest, path: path.join(out, 'controller-path-not-mounted.json')},
    },
  },
  installation
);
const ssh = await pinnedSSH(s);
await mkdir(out, {mode: 0o700});
const io: DockerIO = {
  async save(name, value) {
    assert(/^[a-z0-9-]+\.json$/.test(name));
    await writeFile(path.join(out, name), JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode: 0o600});
  },
  async saveBytes() {
    throw Error('read-only check saves metadata only');
  },
  sshRead(program, timeout = 210000) {
    const child = Bun.spawnSync(ssh, {
      stdin: Buffer.from(program),
      stdout: 'pipe',
      stderr: 'pipe',
      timeout,
      maxBuffer: 4 << 20,
    });
    assert.equal(child.exitCode, 0, 'selected native read unavailable');
    return child.stdout.toString();
  },
};
const readiness = await checkScenario(s);
assert(readiness.native && readiness.observation, 'real native observation required');
for (const name of ['candidate_source', 'target_trust', 'installed_identity', 'installed_images', 'custody'])
  assert(
    readiness.checks.some((c) => c.name === name && c.state === 'pass'),
    name
  );
const boot = await observeApplianceBoot(io, binding);
const state = await readAppliance(io, binding, 'idle', {scenario: s, observation: readiness.observation});
for (const kind of ['projects', 'applications'] as const)
  assert.deepEqual(state.inner.resource_inventory[kind], readiness.observation.inventory[kind]);
assert.deepEqual(
  state.inner.resource_inventory.terminals.map(({id, project_id, actor_id}: any) => ({id, project_id, actor_id})),
  readiness.observation.inventory.terminals
);
assert.equal(state.inner.facts.runs.length, (readiness.native as any).history.runs);
assert.deepEqual(state.inner.facts.connections, readiness.observation.inventory.connections);
const wrong = s.target.uuid.slice(0, -1) + (s.target.uuid.endsWith('0') ? '1' : '0');
const refused = Bun.spawnSync(ssh, {
  stdin: Buffer.from(applianceProgram(wrong, 'console.log("unexpected")')),
  stdout: 'pipe',
  stderr: 'pipe',
  timeout: 30000,
  maxBuffer: 4096,
});
assert.notEqual(refused.exitCode, 0, 'another appliance UUID must refuse before interpreting the program');
assert.equal(refused.stdout.length, 0);
const status = journeyStatus(s, readiness);
assert.equal(status.qualification_phase, null);
assert.equal(status.next_action.code, 'user_hold');
assert(Object.values(status.automatic_actions).every((v) => v === false));
for (const c of status.future_observations.cases) {
  assert.equal(c.completion, null);
  assert.equal(c.measurements, null);
}
await io.save('readiness.json', readiness);
await io.save('status.json', status);
await io.save('boot.json', boot);
await io.save('native-state.json', state);
assert.equal(sha256(await readFile(scenarioFile)), sha256(scenarioBytes));
assert.equal(sha256(await readFile(installationFile)), sha256(installationBytes));
const result = {
  format: 1,
  passed: true,
  check: 'retained-native-appliance-transport',
  scenario: {path: scenarioFile, sha256: sha256(scenarioBytes)},
  installation: {path: installationFile, sha256: sha256(installationBytes)},
  uuid: boot.uuid,
  deployment: boot.deployment,
  current_boot_id: boot.boot_id,
  original_first_boot_id: binding.installation.first_boot.boot_id,
  current_boot_must_equal_first_boot: false,
  staged_manifest_used_without_controller_path: true,
  wrong_uuid_refused: true,
  projects: readiness.observation.inventory.projects.length,
  applications: readiness.observation.inventory.applications.length,
  terminals: readiness.observation.inventory.terminals?.length ?? null,
  connections: readiness.observation.inventory.connections,
  closure: readiness.closure,
  history: (readiness.native as any).history,
  guest_mutations: 0,
  provider_calls: 0,
  reboot_requested: false,
  qualification: null,
};
await io.save('result.json', result);
console.log(JSON.stringify(result));
