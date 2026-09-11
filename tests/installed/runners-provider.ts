// Forgejo 15.0.7's official workflow dispatch/exact-run APIs. No runner creation,
// deletion, workflow publication, retries, latest-run selection or borrowed login.
import assert from 'node:assert/strict';
import {decodeProbeHTTP} from './sodaspaces-http.ts';
import {restrictedText} from './sodaspaces-matrix-native.ts';
import {object} from './sodaspaces-input.ts';
import type {RunnerInput} from './runners-input.ts';

export function dispatchedRunnerJob(value: unknown) {
  const run = object(value);
  assert(typeof run.id === 'number' && Number.isSafeInteger(run.id) && run.id > 0);
  assert(typeof run.run_number === 'number' && Number.isSafeInteger(run.run_number) && run.run_number > 0);
  assert(Array.isArray(run.jobs) && run.jobs.length === 1 && run.jobs[0] === 'native');
  return {run_id:run.id,run_number:run.run_number};
}
export function runnerJobObservation(input: RunnerInput, value: unknown) {
  const p = input.provider; assert(p?.run_id);
  const run = object(value), repository = object(run.repository);
  assert(run.id === p.run_id && Number.isSafeInteger(repository.id) && String(repository.id) === p.repository_id, 'Wrong provider run/repository');
  // Forgejo separates the workflow's `on` trigger from a causing webhook.
  // API dispatch can have an empty webhook event; trigger_event owns this check.
  assert(run.commit_sha === p.commit && run.workflow_id === p.workflow && run.trigger_event === 'workflow_dispatch', 'Wrong workflow, commit or trigger');
  assert(typeof run.event_payload === 'string');
  const payload = object(JSON.parse(run.event_payload)), inputs = object(payload.inputs);
  assert(inputs.observation === p.observation && inputs.hold_seconds === String(p.hold_seconds), 'Wrong job observation');
  assert(typeof run.status === 'string' && ['unknown','waiting','running','success','failure','cancelled','skipped','blocked'].includes(run.status));
  // Do not retain event payload, repository metadata, titles, arbitrary URLs or logs.
  return {run_id:p.run_id,repository_id:p.repository_id,commit:p.commit,observation:p.observation,status:run.status};
}
export async function exerciseRunnerProvider(input: RunnerInput,
  runCurl: (args: string[], stdin: Buffer) => {exitCode: number; stdout: Buffer} = (args,stdin) =>
    Bun.spawnSync(args,{stdin,stdout:'pipe',stderr:'ignore',timeout:31000,maxBuffer:81920})) {
  assert(process.env.SODA_NATIVE_VALIDATE === input.target && (input.phase === 'dispatch' || input.phase === 'job'));
  const p = input.provider; assert(p);
  await restrictedText(input.ca_file,65536);
  const request = async (route: string, body?: Record<string, unknown>) => {
    let token = (await restrictedText(p.token_file,8192)).trim();
    assert(/^[A-Za-z0-9_-]{10,8192}$/.test(token), 'Invalid restricted provider token');
    const args = ['curl','-q','--silent','--noproxy','*','--proto','=https','--http1.1','--max-time','30',
      '--max-filesize','65536','--cacert',input.ca_file,'--dump-header','-','--header','@-'];
    if (body) args.push('--request','POST','--data-binary',JSON.stringify(body));
    args.push('--url',input.origin + route);
    // Only this protected stdin carries the token. Payload argv is non-secret
    // fixed workflow/commit/observation data; redirects and curl config are disabled.
    let headers = 'Authorization: token ' + token + '\nContent-Type: application/json\n';
    token = '';
    const child = runCurl(args,Buffer.from(headers));
    headers = '';
    assert(child.exitCode === 0, 'Provider result unconfirmed; do not replay dispatch');
    const response = decodeProbeHTTP(child.stdout);
    assert(response.status === (body ? 201 : 200), 'Provider result unconfirmed; inspect the exact approved resource');
    const value: unknown = JSON.parse(response.body.toString());
    return value;
  };
  const version = object(await request('/api/v1/version'));
  assert(typeof version.version === 'string' && /^15\.0\.7(?:\+gitea-1\.22\.0)?$/.test(version.version), 'Selected Forgejo API version required');
  const repository = object(await request('/api/v1/repos/' + p.repository_path));
  assert(Number.isSafeInteger(repository.id) && String(repository.id) === p.repository_id, 'Provider repository ID mismatch; no dispatch sent');
  const route = '/api/v1/repos/' + p.repository_path + '/actions/' + (input.phase === 'dispatch'
    ? 'workflows/' + p.workflow + '/dispatches' : 'runs/' + p.run_id);
  const value = await request(route,input.phase === 'dispatch' ? {ref:p.commit,inputs:{observation:p.observation,hold_seconds:String(p.hold_seconds)},return_run_info:true} : undefined);
  return input.phase === 'dispatch' ? dispatchedRunnerJob(value) : runnerJobObservation(input,value);
}
