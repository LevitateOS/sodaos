// Runner-only installed inputs. No login, browser bootstrap or provider registration authority.
import assert from 'node:assert/strict';
import path from 'node:path';
import {object, validID} from './sodaspaces-input.ts';

export type RunnerPhase = 'list' | 'register' | 'start' | 'stop' | 'restart' | 'remove' | 'dispatch' | 'job' | 'contention' | 'departure';
export interface RunnerInput {
  phase: RunnerPhase;
  target: string; architecture: 'x86_64' | 'aarch64'; revision: string;
  origin: string; ca_file: string; ssh_config: string; ssh_host: string;
  operator_id: string; denied_id: string; runner_id: string; preserved_ids: string[];
  // Read-only retained-target observations may declare their actual native roles.
  // Mutation/provider scenarios retain the independent cross-role fixture gate.
  native_admins?: {operator: boolean; denied: boolean};
  registration?: {uuid: string; token_file: string; labels: string; scope: 'system'};
  provider?: {token_file: string; repository_id: string; repository_path: string; workflow: string; commit: string; observation: string; hold_seconds: number; run_id?: number};
}
const runnerID = (value: unknown): value is string => typeof value === 'string' && /^[a-z][a-z0-9-]{0,15}$/.test(value);
function file(value: unknown): asserts value is string {
  assert(typeof value === 'string' && path.isAbsolute(value) && path.normalize(value) === value && !/[\x00-\x1f\x7f]/.test(value), 'Expected a fixed absolute file path');
}
export function runnerInput(value: unknown, permission: string, target: string | undefined): RunnerInput {
  const v = object(value);
  const {phase, architecture, revision, origin, ca_file, ssh_config, ssh_host, operator_id, denied_id, runner_id, preserved_ids} = v;
  assert(phase === 'list' || phase === 'register' || phase === 'start' || phase === 'stop' || phase === 'restart' || phase === 'remove' || phase === 'dispatch' || phase === 'job' || phase === 'contention' || phase === 'departure', 'Unknown runner phase');
  assert.equal(permission, '--allow-runner-' + phase, 'Select exactly this phase, not a blanket native opt-in');
  assert(typeof v.target === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]{0,252}$/.test(v.target) && target === v.target, 'Exact SODA_NATIVE_VALIDATE target required');
  assert.deepEqual(Object.keys(v).sort(), ['phase','target','architecture','revision','origin','ca_file','ssh_config','ssh_host','operator_id','denied_id','runner_id','preserved_ids', ...(phase === 'register' ? ['registration'] : []), ...(['dispatch','job'].includes(phase) ? ['provider'] : []), ...(Object.hasOwn(v,'native_admins') ? ['native_admins'] : [])].sort());
  assert(architecture === 'x86_64' || architecture === 'aarch64');
  assert(typeof revision === 'string' && /^[0-9a-f]{40}$/.test(revision));
  assert(typeof origin === 'string');
  const url = new URL(origin);
  assert(url.protocol === 'https:' && url.origin === origin && !url.username && !url.password, 'Canonical HTTPS origin required');
  file(ca_file); file(ssh_config);
  assert(typeof ssh_host === 'string' && /^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/.test(ssh_host));
  assert(validID(operator_id) && validID(denied_id) && operator_id !== denied_id);
  assert(runnerID(runner_id));
  assert(Array.isArray(preserved_ids) && preserved_ids.length <= 64 && preserved_ids.every(runnerID));
  assert(new Set(preserved_ids).size === preserved_ids.length && !preserved_ids.includes(runner_id), 'Preservation baseline must not be the destructive fixture');
  const result: RunnerInput = {phase,target:v.target,architecture,revision,origin,ca_file,ssh_config,ssh_host,operator_id,denied_id,runner_id,preserved_ids};
  if (Object.hasOwn(v,'native_admins')) {
    assert(phase === 'list', 'Actual retained roles are scoped to read-only list proof');
    const roles = object(v.native_admins);
    assert.deepEqual(Object.keys(roles).sort(), ['denied','operator']);
    assert(typeof roles.operator === 'boolean' && typeof roles.denied === 'boolean');
    result.native_admins = {operator:roles.operator,denied:roles.denied};
  }
  if (phase === 'register') {
    const r = object(v.registration);
    assert.deepEqual(Object.keys(r).sort(), ['labels','scope','token_file','uuid']);
    assert(r.scope === 'system' && typeof r.uuid === 'string' && /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(r.uuid));
    file(r.token_file);
    assert(typeof r.labels === 'string' && r.labels.length <= 4096 && r.labels.split(',').every(label => /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}:host$/.test(label)));
    result.registration = {uuid:r.uuid,token_file:r.token_file,labels:r.labels,scope:r.scope};
  }
  if (phase === 'dispatch' || phase === 'job') {
    const p = object(v.provider);
    assert.deepEqual(Object.keys(p).sort(), ['token_file','repository_id','repository_path','workflow','commit','observation','hold_seconds', ...(phase === 'job' ? ['run_id'] : [])].sort());
    file(p.token_file); assert(validID(p.repository_id));
    assert(typeof p.repository_path === 'string' && /^[A-Za-z0-9][A-Za-z0-9_.-]*\/[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(p.repository_path));
    assert(typeof p.workflow === 'string' && /^[A-Za-z0-9][A-Za-z0-9_.-]*\.ya?ml$/.test(p.workflow));
    assert(typeof p.commit === 'string' && /^[0-9a-f]{40}$/.test(p.commit));
    assert(typeof p.observation === 'string' && /^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$/.test(p.observation));
    assert(typeof p.hold_seconds === 'number' && Number.isInteger(p.hold_seconds) && p.hold_seconds >= 0 && p.hold_seconds <= 600);
    if (phase === 'job') assert(typeof p.run_id === 'number' && Number.isSafeInteger(p.run_id) && p.run_id > 0);
    result.provider = {token_file:p.token_file,repository_id:p.repository_id,repository_path:p.repository_path,workflow:p.workflow,commit:p.commit,observation:p.observation,hold_seconds:p.hold_seconds,...(typeof p.run_id === 'number' ? {run_id:p.run_id} : {})};
  }
  return result;
}
