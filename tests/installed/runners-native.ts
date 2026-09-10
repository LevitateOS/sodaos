// Fixed read-only observer, using the existing pinned-SSH/private-file conventions.
import assert from 'node:assert/strict';
import {restrictedText} from './sodaspaces-matrix-native.ts';
import {object} from './sodaspaces-input.ts';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response.ts';
import type {RunnerInput} from './runners-input.ts';

export async function readRunnerState(input: RunnerInput) {
  assert(process.env.SODA_NATIVE_VALIDATE === input.target, 'Native target not selected');
  await restrictedText(input.ssh_config, 16384);
  const program = await Bun.file(new URL('./runner-state.py', import.meta.url)).text();
  const command = ['env', 'SODA_NATIVE_VALIDATE=' + input.target, ...(input.provider ? ['SODA_RUNNER_OBSERVATION=' + input.provider.observation] : []), 'timeout', '--signal=KILL', '75s', 'python3', '-I', '-', input.target, input.runner_id, ...input.preserved_ids].join(' ');
  // All remote words above are validated identifiers, never browser-selected shell text.
  // Stock timeout bounds only this run-owned observer/CLI-read process group;
  // killing the SSH client alone would not establish remote reader termination.
  const child = Bun.spawnSync(['ssh','-F',input.ssh_config,'-T','-o','BatchMode=yes','-o','StrictHostKeyChecking=yes',
    '-o','PermitLocalCommand=no','-o','ClearAllForwardings=yes','-o','ForwardAgent=no','-o','ForwardX11=no',
    '-o','RemoteCommand=none','-o','ConnectTimeout=10','-o','ControlMaster=no','-o','ControlPath=none',
    '-o','IdentityAgent=none','-o','IdentitiesOnly=yes',input.ssh_host,command],
    {stdin:Buffer.from(program),stdout:'pipe',stderr:'ignore',timeout:90000,maxBuffer:65536});
  assert(child.exitCode === 0, 'Native runner observation unavailable');
  const value = object(JSON.parse(child.stdout.toString()));
  assert(value.target === input.target && value.architecture === input.architecture, 'Native target/architecture mismatch');
  const inventory = decodeRunnerResponse('list',value.inventory);
  assert(inventory.forgejo_url === input.origin, 'Configured browser origin differs from declared target');
  const states = object(value.states), confinement = object(value.confinement);
  assert.deepEqual(Object.keys(states).sort(), [input.runner_id,...input.preserved_ids].sort());
  for (const id of Object.keys(states)) {
    const state = object(states[id]);
    assert(typeof state.present === 'boolean');
    if (state.present) {
      assert(typeof state.uid === 'number' && Number.isSafeInteger(state.uid) && state.uid > 0);
      assert(typeof state.gid === 'number' && Number.isSafeInteger(state.gid) && state.gid > 0);
      assert(state.home === '/var/lib/soda/runners/' + id + '/state' && state.shell === '/usr/sbin/nologin');
      const tree = object(state.tree); assert(typeof tree.sha256 === 'string' && /^[0-9a-f]{64}$/.test(tree.sha256));
      assert(confinement[id] === true, 'Effective runner confinement differs');
    }
    assert(state.present === inventory.runners.some(row => row.id === id), 'Account/state and CLI disagree');
  }
  assert(Array.isArray(value.packages) && value.packages.length === 2 && value.packages.every(p => typeof p === 'string' && p.length < 256));
  const proof = value.job_proof === null ? null : object(value.job_proof);
  if (proof) {
    assert(input.provider && proof.observation === input.provider.observation && proof.account === 'soda-runner-'+input.runner_id);
    assert(proof.uid === object(states[input.runner_id]).uid && typeof proof.alive === 'boolean' && (proof.steps === 1 || proof.steps === 2));
  }
  return {inventory,states,packages:value.packages,architecture:input.architecture,target:input.target,proof};
}
export type RunnerState = Awaited<ReturnType<typeof readRunnerState>>;

export function preserveRunnerBaseline(input: RunnerInput, before: RunnerState, after: RunnerState) {
  assert.deepEqual(after.packages,before.packages, 'Installed runner/systemd versions changed');
  assert.deepEqual(before.inventory.runners.filter(row => row.id !== input.runner_id).map(row => row.id).sort(), [...input.preserved_ids].sort(), 'Undeclared retained runner; expand the approved preservation scope before effects');
  for (const id of input.preserved_ids) {
    assert(object(before.states[id]).present === true, 'Declared preservation runner is missing');
    assert.deepEqual(after.states[id],before.states[id], 'Preserved runner identity or work/credential state changed');
    assert.deepEqual(after.inventory.runners.find(row => row.id === id),before.inventory.runners.find(row => row.id === id), 'Preserved listener policy changed');
  }
  const others = (state: RunnerState) => state.inventory.runners.filter(row => row.id !== input.runner_id);
  assert.deepEqual(others(after),others(before), 'Unrelated local inventory changed');
}
