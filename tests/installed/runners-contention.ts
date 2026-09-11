// Fixed cases C2/C4, called only after the existing native-page/auth/guard handoff.
// No fixture creation, credentials from another target, provider calls or retries.
import assert from 'node:assert/strict';
import type {Frame, Page, Request} from 'playwright';
import type {} from '../../cockpit/src/cockpit/types.ts';
import {loginOperator, openOperatorPackage} from './operator.ts';
import {restrictedText} from './sodaspaces-matrix-native.ts';
import type {RunnerInput} from './runners-input.ts';
import type {RunnerEvidence} from './runners.ts';
import {runnerSSH, readRunnerState, preserveRunnerBaseline, verifyRunnerContention, type RunnerState} from './runners-native.ts';
import {decodeRunnerResponse} from '../../frontend/runners/soda-runner-response.ts';

export interface ContentionEvidence {
  held_at?: number; released_at?: number; web_dispatch_at?: number;
  web_return_at?: number; cockpit_dispatch_at?: number; cockpit_return_at?: number;
  cli_dispatch_at?: number; cli_return_at?: number; departed_at?: number;
  mutation_posts: number; result?: string;
}

export async function exerciseRunnerContention(operator: Page, input: RunnerInput, before: RunnerState,
  permit: (actor: string, route: string, body: string) => void, evidence: RunnerEvidence) {
  assert(input.phase === 'overlap' || input.phase === 'departure');
  const receipt: ContentionEvidence = {mutation_posts:0}; evidence.contention=receipt;
  const browser=operator.context().browser(); assert(browser);
  const cockpitContext=input.phase === 'overlap' ? await browser.newContext() : undefined;
  let frame: Frame | undefined;
  const route='/-/soda/api/settings/runners/'+input.runner_id+'/restart';
  const observe=(request: Request)=>{
    const url=new URL(request.url());
    if(url.origin === input.origin && url.pathname === route && request.method() === 'POST') {
      receipt.mutation_posts++; receipt.web_dispatch_at=performance.now();
    }
  };
  operator.on('request',observe);
  try {
    evidence.stage='prepare native and Cockpit controls before admission hold';
    if(cockpitContext) {
      const config=input.cockpit; assert(config);
      const page=await cockpitContext.newPage();
      page.setDefaultTimeout(30000);
      let password=(await restrictedText(config.password_file,65536)).replace(/\r?\n$/,'');
      assert(password && !/[\r\n]/.test(password));
      evidence.stage='Cockpit root login';
      await loginOperator(page,config.origin,password); password='';
      evidence.stage='Cockpit Runners package';
      frame=await openOperatorPackage(page,'Runners','soda-runners');
      evidence.stage='Cockpit native root authority';
      const native=await frame.evaluate(async()=>({
        host:(await window.cockpit.spawn(['hostname'],{err:'message'})).trim(),
        uid:(await window.cockpit.spawn(['id','-u'],{err:'message'})).trim(),
        domain:(await window.cockpit.spawn(['id','-Z'],{err:'message'})).trim(),
      }));
      assert(native.host === input.target && native.uid === '0' && !native.domain.includes(':cockpit_session_t:'));
      evidence.stage='Cockpit exact disposable Stop control';
      await frame.getByRole('row').filter({has:frame.getByText(input.runner_id,{exact:true})}).getByRole('button',{name:'Stop',exact:true}).waitFor();
    }
    assert(before.inventory.runners.find(r=>r.id === input.runner_id)?.service.active === 'active','Use an idle running disposable listener');
    evidence.stage='native exact Restart confirmation';
    const view=operator.locator('soda-runners');
    await view.getByRole('button',{name:'restart '+input.runner_id,exact:true}).click();
    await view.getByLabel('Exact runner ID',{exact:true}).fill(input.runner_id);
    // Recheck both native records after login and before the deliberately held lock.
    evidence.stage='native preservation before hold';
    const prepared=await readRunnerState(input); assert.deepEqual(prepared,before);
    const command=`test "$(hostname)" = ${input.target} && test -f /run/lock/soda/runners.lock && test ! -L /run/lock/soda/runners.lock && test "$(stat -c '%u:%a' /run/lock/soda/runners.lock)" = 0:600 && timeout 35s flock --exclusive /run/lock/soda/runners.lock sh -c 'printf "locked\\n"; sleep 20; printf "released\\n"'`;
    const holder=Bun.spawn([...runnerSSH(input),command],{stdin:'ignore',stdout:'pipe',stderr:'ignore'});
    const reader=holder.stdout.getReader();
    let text='';
    let timer: ReturnType<typeof setTimeout> | undefined;
    const next=async()=>{
      const chunk=await reader.read(); assert(!chunk.done && chunk.value);
      text+=new TextDecoder().decode(chunk.value); assert(text.length<=64);
    };
    const finished=(async()=>{
      await Promise.race([(async()=>{while(!text.includes('\n')) await next();})(),new Promise<never>((_,reject)=>{timer=setTimeout(()=>reject(Error('Admission holder unavailable')),10000);})]);
      clearTimeout(timer); assert.equal(text,'locked\n'); receipt.held_at=performance.now();
    })();
    // Even failed client work waits for this exact bounded holder; no PID lookup,
    // lock deletion, forced release, implicit replay or unbounded held admission.
    const release=async()=>{
      clearTimeout(timer);
      while(!text.includes('released\n')) await next();
      assert.equal(text,'locked\nreleased\n');
      assert.equal(await holder.exited,0); receipt.released_at=performance.now();
    };
    let web: Promise<void> | undefined;
    let cockpit: Promise<void> | undefined;
    let cli: Promise<void> | undefined;
    try {
      await finished;
      const held=receipt.held_at; assert(typeof held === 'number');
      evidence.stage='one native Restart under held admission';
      permit(input.operator_id,route.slice('/-/soda'.length),JSON.stringify({confirm_id:input.runner_id}));
      const dispatched=operator.waitForRequest(r=>new URL(r.url()).pathname === route && r.method() === 'POST');
      if(input.phase === 'overlap') {
        web=operator.waitForResponse(r=>new URL(r.url()).origin === input.origin && new URL(r.url()).pathname === route && r.request().method() === 'POST',{timeout:200000}).then(async response=>{
          receipt.web_return_at=performance.now(); assert.equal(response.status(),200); decodeRunnerResponse('create',await response.json());
        });
        void web.catch(()=>{});
      }
      await view.getByRole('button',{name:'Confirm restart',exact:true}).click();
      await dispatched;
      assert(receipt.web_dispatch_at && performance.now()-held < 15000,'Dispatch missed held interval');
      if(input.phase === 'overlap') {
        assert(frame);
        const cockpitFrame=frame;
        receipt.cockpit_dispatch_at=performance.now();
        await cockpitFrame.getByRole('row').filter({has:cockpitFrame.getByText(input.runner_id,{exact:true})}).getByRole('button',{name:'Stop',exact:true}).click();
        await cockpitFrame.getByRole('status').filter({hasText:'Stopping '+input.runner_id}).waitFor();
        cockpit=cockpitFrame.getByRole('status').filter({hasText:'Stopping '+input.runner_id}).waitFor({state:'hidden',timeout:200000}).then(()=>{receipt.cockpit_return_at=performance.now();});
        void cockpit.catch(()=>{});
        receipt.cli_dispatch_at=performance.now();
        const read=Bun.spawn([...runnerSSH(input),`test "$(hostname)" = ${input.target} && timeout 75s /usr/local/libexec/soda/soda-runners list`],{stdin:Buffer.from('{}\n'),stdout:'pipe',stderr:'ignore'});
        cli=(async()=>{const raw=await new Response(read.stdout).text(); assert(raw.length<=65536); assert.equal(await read.exited,0); decodeRunnerResponse('list',JSON.parse(raw)); receipt.cli_return_at=performance.now();})();
        void cli.catch(()=>{});
        // A fixed observation inside the held interval: none may finish through it.
        await new Promise(resolve=>setTimeout(resolve,1000));
        assert(performance.now()-held < 18000);
        assert(!receipt.web_return_at && !receipt.cockpit_return_at && !receipt.cli_return_at);
      } else {
        receipt.departed_at=performance.now();
        await operator.goto(input.origin+'/issues');
        assert(performance.now()-held < 18000,'Departure missed held interval');
      }
    } finally {
      await release();
      await Promise.allSettled([web,cockpit,cli].filter(p=>p !== undefined));
    }
    await Promise.all([web,cockpit,cli]);
    evidence.stage='native postconditions after bounded hold';
    const after=await readRunnerState(input,before); evidence.after=after;
    receipt.result=verifyRunnerContention(input,before,after);
    if(input.phase === 'overlap') {
      assert(frame);
      evidence.stage='Cockpit Stop success acknowledgement';
      // PatternFly prefixes its alert title with an accessible severity label.
      await frame.locator('.soda-diagnostic.pf-m-success').filter({hasText:input.runner_id+' was stopped.'}).waitFor();
      evidence.stage='Cockpit post-operation CLI inventory';
      // Cockpit returns its native thenable, not a browser Promise. Assimilate it
      // inside the page before Playwright serializes the result.
      const raw=await frame.evaluate(async()=>await window.cockpit.spawn(['/usr/local/libexec/soda/soda-runners','list'],{err:'message'}).input('{}\n'));
      const cliInventory=decodeRunnerResponse('list',JSON.parse(raw));
      // The native observer and web API deliberately expose only the configured
      // public origin, not saved private registration endpoints. Compare the same
      // projection; native state hashes independently retain the actual settings.
      const publicOrigin=cliInventory.forgejo_url; assert(publicOrigin === input.origin);
      for(const row of cliInventory.runners) row.registration_url=publicOrigin;
      assert.deepEqual(cliInventory,after.inventory);
      evidence.stage='Cockpit refreshed native row';
      await frame.getByRole('button',{name:'Refresh',exact:true}).click();
      const current=after.inventory.runners.find(r=>r.id === input.runner_id); assert(current);
      await frame.getByRole('row').filter({has:frame.getByText(input.runner_id,{exact:true})}).getByText(`${current.service.active}/${current.service.sub}; ${current.service.enabled}`,{exact:true}).waitFor();
    }
    evidence.stage='native return and reload without replay';
    await operator.goto(input.origin+'/?soda-view=runners');
    await operator.locator('#soda-native-content[data-actor="'+input.operator_id+'"]').waitFor();
    await operator.reload();
    await operator.locator('#soda-native-content[data-actor="'+input.operator_id+'"]').waitFor();
    await new Promise(resolve=>setTimeout(resolve,2000));
    assert.equal(receipt.mutation_posts,1,'Navigation replayed the mutation');
    const listed=await operator.evaluate(async actor=>{
      const response=await fetch('/-/soda/api/settings/runners',{cache:'no-store',headers:{'X-Soda-Expected-User-ID':actor}});
      if(!response.ok) throw Error('Post-case inventory unavailable'); return response.json();
    },input.operator_id);
    assert.deepEqual(decodeRunnerResponse('list',listed),after.inventory);
    evidence.outcome='confirmed';
  } finally {
    operator.off('request',observe);
    await cockpitContext?.close();
    if(!evidence.after) {
      evidence.after=await readRunnerState(input,before);
      preserveRunnerBaseline(input,before,evidence.after);
    }
  }
}
