// Runner phases called by the Soda-pages installed driver AFTER its native-shell
// connection handoff. This module never logs in, navigates, seeds cookies, retries
// a mutation, creates a browser fixture or cleans up provider/native resources.
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {fileURLToPath} from 'node:url';
import {restrictedText} from './sodaspaces-matrix-native.ts';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response.ts';
import {runnerInput, type RunnerInput} from './runners-input.ts';
import {readRunnerState, preserveRunnerBaseline, verifyRunnerOperation} from './runners-native.ts';
import {exerciseRunnerProvider} from './runners-provider.ts';

export async function loadRunnerInput(file: string, permission: string) {
  try {
    const input = runnerInput(JSON.parse(await restrictedText(file,16384)),permission,process.env.SODA_NATIVE_VALIDATE);
    const root = fileURLToPath(new URL('../../',import.meta.url));
    const git = (args: string[]) => Bun.spawnSync(['git',...args],{cwd:root,stdout:'pipe',stderr:'ignore'});
    const revision = git(['rev-parse','HEAD']), state = git(['status','--porcelain']);
    assert(revision.exitCode === 0 && revision.stdout.toString().trim() === input.revision && state.exitCode === 0 && state.stdout.length === 0, 'Exact clean journey revision required');
    await restrictedText(input.ca_file,65536);
    await restrictedText(input.ssh_config,16384);
    return input;
  } catch {throw Error('Runner input or candidate prerequisite refused; no runner effects requested');}
}

export interface RunnerEvidence {
  phase?: string; target?: string; revision?: string; stage?: string;
  before?: Awaited<ReturnType<typeof readRunnerState>>;
  after?: Awaited<ReturnType<typeof readRunnerState>>;
  provider?: Awaited<ReturnType<typeof exerciseRunnerProvider>>;
  outcome?: 'confirmed' | 'unconfirmed';
}
// permit is the existing installed driver's one-shot exact-request guard. It must
// discard the transient serialized body on transmission/abort and never log it.
export async function exerciseRunners(operator: Page, denied: Page, request: RunnerInput,
  permission: string, permit: (actor: string, route: string, body: string) => void, evidence: RunnerEvidence) {
  const input = runnerInput(request,permission,process.env.SODA_NATIVE_VALIDATE);
  evidence.phase=input.phase; evidence.target=input.target; evidence.revision=input.revision;
  evidence.outcome='unconfirmed';
  try {
    evidence.stage='native preflight';
    const before = await readRunnerState(input); evidence.before=before;
    // Require preservation inputs to resolve before any effect, not only afterward.
    preserveRunnerBaseline(input,before,before);
    if (input.phase === 'dispatch' || input.phase === 'job') {
      if (input.phase === 'dispatch') {
        assert(!before.proof, 'Observation already exists; do not redispatch or erase its proof');
        assert(before.inventory.runners.some(row => row.id === input.runner_id && row.service.active === 'active' && row.service.sub === 'running'), 'Declared fixture listener not running');
      }
      evidence.stage='provider ' + input.phase;
      try {
        evidence.provider=await exerciseRunnerProvider(input);
      } finally {
        // Dispatch can succeed upstream even if its response is lost. Preserve
        // post-attempt observations on both paths; never replay or roll back.
        evidence.after=await readRunnerState(input,before);
        preserveRunnerBaseline(input,before,evidence.after);
      }
      if (input.phase === 'job') {
        assert(evidence.after.proof, 'No exact native job identity observed on the declared runner');
        if ('status' in evidence.provider && evidence.provider.status === 'success') assert(evidence.after.proof.steps === 2, 'Successful provider run lacks two-step native proof');
      }
      // A dispatch receipt confirms only dispatch; a job read reports its actual
      // status, never promotes waiting/running or a local listener to job success.
      evidence.outcome='confirmed';
      return;
    }
    evidence.stage='handed-off native pages';
    for (const [page,actor] of [[operator,input.operator_id],[denied,input.denied_id]] as const) {
      const url=new URL(page.url());
      assert(url.origin === input.origin && url.pathname === '/' && url.search === '?soda-view=runners' && !url.hash, 'Native Runners page handoff required');
      assert(await page.locator('#soda-native-content[data-view="runners"]').getAttribute('data-actor') === actor, 'Original native actor mismatch');
    }
    const view=operator.locator('soda-runners');
    assert(await view.getAttribute('data-actor') === input.operator_id, 'Original Soda operator mismatch');
    // Read-only denial: list permission never opts into even an expected-denied mutation.
    const status=await denied.evaluate(async actor => {
      const response=await fetch('/-/soda/api/settings/runners',{credentials:'same-origin',cache:'no-store',redirect:'error',headers:{'X-Soda-Expected-User-ID':actor}});
      await response.body?.cancel(); return response.status;
    },input.denied_id);
    assert(status === 403,'Nonoperator runner read was not denied');
    const refresh=async()=>{
      const [response]=await Promise.all([
        operator.waitForResponse(r => new URL(r.url()).origin === input.origin && new URL(r.url()).pathname === '/-/soda/api/settings/runners' && r.request().method() === 'GET'),
        view.getByRole('button',{name:'Refresh status',exact:true}).click(),
      ]);
      assert(response.status() === 200, 'Local inventory unavailable');
      const data=decodeRunnerResponse('list',await response.json());
      // The production page discards late generations. This observer does not
      // substitute a response or reimplement that browser lifetime mechanism.
      return data;
    };
    const listed=await refresh();
    assert.deepEqual(listed,before.inventory,'Web and native CLI observations differ');
    if(input.phase === 'list') {
      evidence.after=await readRunnerState(input);
      assert.deepEqual(evidence.after,before,'Read-only page changed observed runner state');
      evidence.outcome='confirmed'; return;
    }
    const existing=before.inventory.runners.find(row => row.id === input.runner_id);
    assert(input.phase === 'register' ? !existing : Boolean(existing), 'Wrong fixture registration state');
    const route='/api/settings/runners' + (input.phase === 'register' ? '' : '/' + input.runner_id + '/' + input.phase);
    let body='';
    evidence.stage='prepare ' + input.phase;
    if(input.phase === 'register') {
      const registration=input.registration; assert(registration);
      let token=(await restrictedText(registration.token_file,8192)).replace(/\r?\n$/,'');
      assert(token && !/[\x00\r\n]/.test(token),'Invalid restricted registration token');
      await view.getByLabel('Local runner ID',{exact:true}).fill(input.runner_id);
      await view.getByLabel('Forgejo runner UUID',{exact:true}).fill(registration.uuid);
      await view.getByLabel('Labels',{exact:true}).fill(registration.labels);
      await view.getByLabel('Registration token',{exact:true}).fill(token);
      body=JSON.stringify({id:input.runner_id,provider:'forgejo',registration_url:'',registration_id:registration.uuid,labels:registration.labels,registration_token:token}); token='';
    } else {
      await view.getByRole('button',{name:input.phase + ' ' + input.runner_id,exact:true}).click();
      await view.getByLabel('Exact runner ID',{exact:true}).fill(input.runner_id);
      body=JSON.stringify({confirm_id:input.runner_id});
    }
    permit(input.operator_id,route,body); body='';
    evidence.stage='dispatch ' + input.phase;
    try {
      const [response]=await Promise.all([
        operator.waitForResponse(r => new URL(r.url()).origin === input.origin && new URL(r.url()).pathname === '/-/soda'+route && r.request().method() === 'POST'),
        view.getByRole('button',{name:input.phase === 'register' ? 'Register and start listener' : 'Confirm '+input.phase,exact:true}).click(),
      ]);
      assert(response.status() === 200,'Runner mutation unconfirmed; do not replay');
      decodeRunnerResponse('create',await response.json());
      assert(await view.locator('input[type=password]').inputValue() === '', 'Credential input was not cleared');
    } finally {
      // Inspection only, including failed/partial operations. No rollback or retry.
      await view.locator('input[type=password]').fill('').catch(()=>{});
      evidence.after=await readRunnerState(input,before);
      preserveRunnerBaseline(input,before,evidence.after);
    }
    evidence.stage='native result and preserved baseline';
    const after=evidence.after; assert(after);
    verifyRunnerOperation(input,before,after);
    assert.deepEqual(await refresh(),after.inventory,'Post-operation web/native observations differ');
    evidence.outcome='confirmed';
  } catch {
    // Also scrub an unsent token if preparation or the caller's guard failed.
    await operator.locator('soda-runners input[type=password]').evaluateAll(inputs => {
      for (const input of inputs) if (input instanceof HTMLInputElement) input.value='';
    }).catch(()=>{});
    throw Error('Runner journey unconfirmed; retain stage/evidence and inspect before retrying');
  }
}
