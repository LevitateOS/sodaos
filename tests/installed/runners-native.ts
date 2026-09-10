// Fixed read-only observer, using the existing pinned-SSH/private-file conventions.
import assert from 'node:assert/strict';
import {restrictedText} from './sodaspaces-matrix-native.ts';
import {object} from './sodaspaces-input.ts';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response.ts';
import type {RunnerInput} from './runners-input.ts';

export interface RunnerProcess {pid: number; uid: number; start: string}
function processList(value: unknown): RunnerProcess[] {
  assert(Array.isArray(value) && value.length <= 256);
  const rows=value.map(item=>{
    const row=object(item);
    assert.deepEqual(Object.keys(row).sort(),['pid','start','uid']);
    assert(typeof row.pid === 'number' && Number.isSafeInteger(row.pid) && row.pid > 0 && row.pid < 2**31);
    assert(typeof row.uid === 'number' && Number.isSafeInteger(row.uid) && row.uid > 0 && row.uid < 2**32);
    assert(typeof row.start === 'string' && /^[0-9]{1,20}$/.test(row.start));
    return {pid:row.pid,uid:row.uid,start:row.start};
  });
  assert(new Set(rows.map(row=>row.pid)).size === rows.length, 'Duplicate PID observation');
  return rows;
}

export async function readRunnerState(input: RunnerInput, prior?: {boot_id: string; processes: Record<string, RunnerProcess[]>}) {
  assert(process.env.SODA_NATIVE_VALIDATE === input.target, 'Native target not selected');
  await restrictedText(input.ssh_config, 16384);
  const program = await Bun.file(new URL('./runner-state.py', import.meta.url)).text();
  const previous=prior ? processList(prior.processes[input.runner_id]) : [];
  const priorArgs=prior ? ['SODA_RUNNER_PRIOR_BOOT='+prior.boot_id,'SODA_RUNNER_PRIOR_PROCESSES='+previous.map(row=>`${row.pid}:${row.start}:${row.uid}`).join(',')] : [];
  if(prior) assert(/^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(prior.boot_id));
  const command = ['env', ...priorArgs, 'SODA_NATIVE_VALIDATE=' + input.target, ...(input.provider ? ['SODA_RUNNER_OBSERVATION=' + input.provider.observation] : []), 'timeout', '--signal=KILL', '75s', 'python3', '-I', '-', input.target, input.runner_id, ...input.preserved_ids].join(' ');
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
  assert(typeof value.boot_id === 'string' && /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(value.boot_id));
  if(prior) assert.equal(value.boot_id,prior.boot_id,'Boot changed across operation');
  const rawProcesses=object(value.processes), processes: Record<string,RunnerProcess[]>={};
  assert.deepEqual(Object.keys(rawProcesses).sort(),Object.keys(states).sort());
  for(const id of Object.keys(states)) {
    const rows=processList(rawProcesses[id]);
    assert(rows.every(row=>row.uid === object(states[id]).uid),'Process outside declared runner identity');
    const runner=inventory.runners.find(row=>row.id === id);
    if(runner?.service.active === 'active') assert(rows.length > 0,'Running listener without process observation');
    processes[id]=rows;
  }
  const prior_survivors=processList(value.prior_survivors);
  assert(prior_survivors.every(row=>previous.some(old=>old.pid === row.pid && old.uid === row.uid && old.start === row.start)), 'Unrequested survivor observation');
  if(prior) for(const row of processes[input.runner_id] || []) {
    if(previous.some(old=>old.pid === row.pid && old.uid === row.uid && old.start === row.start))
      assert(prior_survivors.some(old=>old.pid === row.pid && old.uid === row.uid && old.start === row.start),'Inconsistent survivor observation');
  }
  const proof = value.job_proof === null ? null : object(value.job_proof);
  if (proof) {
    assert(input.provider && proof.observation === input.provider.observation && proof.account === 'soda-runner-'+input.runner_id);
    assert(proof.uid === object(states[input.runner_id]).uid && typeof proof.alive === 'boolean' && (proof.steps === 1 || proof.steps === 2));
  }
  return {inventory,states,packages:value.packages,architecture:input.architecture,target:input.target,proof,boot_id:value.boot_id,processes,prior_survivors};
}
export type RunnerState = Awaited<ReturnType<typeof readRunnerState>>;

// Native postconditions belong with the observer, independent of the browser
// driver. Local fixtures exercise these comparisons, not installed execution.
export function verifyRunnerOperation(input: RunnerInput, before: RunnerState, after: RunnerState) {
  preserveRunnerBaseline(input,before,after);
  assert(['register','start','stop','restart','remove'].includes(input.phase), 'Expected a lifecycle phase');
  const originalRow=before.inventory.runners.find(row => row.id === input.runner_id);
  assert(input.phase === 'register' ? !originalRow : Boolean(originalRow), 'Wrong fixture registration state');
  const current=after.inventory.runners.find(row => row.id === input.runner_id);
  if(['stop','restart','remove'].includes(input.phase)) {
    assert.equal(after.prior_survivors.length,0,'An observed prior native process survived the operation');
    const previous=before.processes[input.runner_id]; assert(previous);
    assert((after.processes[input.runner_id] || []).every(row=>!previous.some(old=>old.pid === row.pid && old.start === row.start && old.uid === row.uid)), 'Old process remains in the runner scope');
  }
  if(input.phase === 'stop' || input.phase === 'remove') assert.deepEqual(after.processes[input.runner_id],[],'Runner process scope is not empty');
  if(input.phase === 'remove') {
    assert(!current && object(after.states[input.runner_id]).present === false, 'Local account/state removal not confirmed');
    return;
  }
  assert(current && current.capacity === 1 && current.account === 'soda-runner-'+input.runner_id);
  const retained=object(after.states[input.runner_id]);
  assert(retained.present === true, 'Native account/state unavailable');
  if (input.phase !== 'register') {
    const original=object(before.states[input.runner_id]);
    for (const key of ['uid','gid','home','shell','registration']) {
      assert(original[key] !== undefined, 'Incomplete lifecycle baseline');
      assert.deepEqual(retained[key],original[key], 'Fixture account or registration changed during lifecycle');
    }
    assert(current.version === originalRow?.version && current.architecture === originalRow.architecture, 'Fixture client changed during lifecycle');
  }
  assert(current.service.enabled === (input.phase === 'stop' ? 'disabled' : 'enabled'), 'Wrong host-boot policy');
  assert(input.phase === 'stop' ? current.service.active === 'inactive' : current.service.active === 'active' && current.service.sub === 'running', 'Listener state unconfirmed');
}

export function preserveRunnerBaseline(input: RunnerInput, before: RunnerState, after: RunnerState) {
  assert(after.target === before.target && after.target === input.target && after.architecture === before.architecture && after.architecture === input.architecture, 'Preservation target changed');
  assert.equal(after.boot_id,before.boot_id,'Unexpected reboot during runner phase');
  assert.deepEqual(after.packages,before.packages, 'Installed runner/systemd versions changed');
  assert.deepEqual(before.inventory.runners.filter(row => row.id !== input.runner_id).map(row => row.id).sort(), [...input.preserved_ids].sort(), 'Undeclared retained runner; expand the approved preservation scope before effects');
  for (const id of input.preserved_ids) {
    assert(object(before.states[id]).present === true, 'Declared preservation runner is missing');
    assert.deepEqual(after.states[id],before.states[id], 'Preserved runner identity or work/credential state changed');
    assert(Array.isArray(before.processes[id]),'Preserved process baseline missing');
    assert.deepEqual(after.processes[id],before.processes[id],'Preserved runner process scope changed');
    assert.deepEqual(after.inventory.runners.find(row => row.id === id),before.inventory.runners.find(row => row.id === id), 'Preserved listener policy changed');
  }
  const others = (state: RunnerState) => state.inventory.runners.filter(row => row.id !== input.runner_id);
  assert.deepEqual(others(after),others(before), 'Unrelated local inventory changed');
}
