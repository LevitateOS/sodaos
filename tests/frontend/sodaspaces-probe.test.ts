// SPDX-License-Identifier: Apache-2.0
// Execute the probe's actual request guard with CDP/page doubles. Native redirect
// delivery remains owned by the separately authorized browser journey.
import test from 'node:test';
import assert from 'node:assert/strict';
import {runInNewContext as runJS} from 'node:vm';
import {mkdtemp, lstat, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {launchNativeBrowser} from '../installed/native-browser.ts';

const probe = await Bun.file(new URL('../installed/sodaspaces.ts', import.meta.url)).text();
const transpiler = new Bun.Transpiler({loader: 'ts', target: 'bun'});
const runInNewContext = (source: string, scope: object): unknown => runJS(transpiler.transformSync(source), scope);
interface Evidence {anonymous_repository?: string; own_connections?: Array<{user_id: string; command: string}>}
interface Paused {requestId: string; redirectedRequestId?: string; request: {url: string; method: string; postData?: string | undefined; headers?: Record<string, string>}}
interface Intent {path: string; body: string; actor: string; method?: string; page?: object}
const start = probe.indexOf('  async function guardedPage(pageContext: BrowserContext = context) {');
const end = probe.indexOf('  await context.addInitScript(', start);
assert(start > 0 && end > start);

test('native coordinated logout requires both Soda and Forgejo response contracts', async () => {
  const from = probe.indexOf('  assert.equal((await sodaEnded).status(), 204);');
  const to = probe.indexOf('  for (const p of [other, page])', from);
  assert(from > 0 && to > from);
  for (const [sodaStatus, forgejoStatus] of [[204,200],[200,200],[401,200],[403,200],[500,200],[204,204],[204,401],[204,500]]) {
    const attempt = runInNewContext('(async () => {\n' + probe.slice(from, to) + '\n})()',
      {assert, sodaEnded: Promise.resolve({status() { return sodaStatus; }}),
        forgejoEnded: Promise.resolve({status() { return forgejoStatus; }})});
    if (sodaStatus === 204 && forgejoStatus === 200) await attempt;
    else await assert.rejects(async () => await attempt);
  }
});

test('runner mutation waiter is bounded beyond native drain and retains exact response matching', async () => {
  const source = await Bun.file(new URL('../installed/runners.ts', import.meta.url)).text();
  const from = source.indexOf('operator.waitForResponse(', source.indexOf("evidence.stage='dispatch ' + input.phase;"));
  const to = source.indexOf(',\n        view.getByRole', from);
  assert(from > 0 && to > from);
  const origin = 'https://fixture.invalid', route = '/api/settings/runners/probe-one/stop';
  const operator = {waitForResponse(predicate: (r: {url(): string; request(): {method(): string}}) => boolean, options?: {timeout: number}) {
    assert.equal(options?.timeout, 200000);
    for (const [url, method, expected] of [[origin+'/-/soda'+route,'POST',true], [origin+'/-/soda'+route,'GET',false], ['https://other.invalid/-/soda'+route,'POST',false], [origin+'/-/soda'+route+'-other','POST',false]] as const) {
      assert.equal(predicate({url: () => url, request: () => ({method: () => method})}), expected);
    }
    return Promise.resolve();
  }};
  await runInNewContext(source.slice(from, to), {operator, URL, input: {origin}, route});
});

test('native layout guard measures the visual viewport without hiding scrollbars or permitting overflow', async () => {
  const from = probe.indexOf('    const box = await drawer.boundingBox();');
  const to = probe.indexOf('    if (width === 360)', from);
  assert(from > 0 && to > from);
  for (const [visualWidth, x, boxWidth, valid] of [[345, 0, 345, true], [360, 0, 360, true],
    [345, 0, 360, false], [345, 0, 300, false], [345, -15, 360, false], [0, 0, 0, false], [375, 0, 375, false]] as const) {
    const result: Record<string, unknown> = {};
    const scope = {assert, result, width: 360, colorScheme: 'light',
      drawer: {async boundingBox() { return {x, y: 44, width: boxWidth, height: 856}; }},
      page: {async evaluate() { return {width: visualWidth, height: 900, top: 0, control_height: '44px'}; },
        getByRole() { return {async getAttribute() { return 'true'; }}; }}};
    const attempt = runInNewContext('(async () => {\n' + probe.slice(from, to) + '\n})()', scope);
    if (valid) {await attempt; assert(result.native_layout);}
    else await assert.rejects(async () => await attempt);
  }
});

test('private repository declaration is read-only and cannot combine with access mode', () => {
  const from = probe.indexOf('  const [inputFile, home, permission, ...extra]');
  const to = probe.indexOf('  assert(inputFile && home);', from);
  assert(from > 0 && to > from);
  for (const [args, access, privateRepo] of [[[], false, false], [['--private-repository'], false, true], [['--allow-environment-access'], true, false], [['--allow-existing-terminal'], false, false], [['--allow-existing-management', '/private/request.json'], false, false]] as const) {
    const scope = {assert, accessMode: false, Bun: {argv: ['bun', 'probe', 'input', 'home', '--allow-auth-transitions', ...args]}};
    const mode = runInNewContext('(() => {\n' + probe.slice(from, to) + '\nreturn {accessMode, privateRepository}; })()', scope);
    assert(mode && typeof mode === 'object' && 'accessMode' in mode && 'privateRepository' in mode);
    assert.equal(mode.accessMode, access);
    assert.equal(mode.privateRepository, privateRepo);
  }
  for (const args of [['--private-repository','--allow-environment-access'], ['--allow-existing-management'], ['--allow-existing-terminal','/private/request.json'], ['--allow-existing-management','/private/request.json','--private-repository']]) {
    assert.throws(() => runInNewContext(probe.slice(from, to), {assert, Bun: {argv: ['bun','probe','input','home','--allow-auth-transitions',...args]}}));
  }
});

test('runner mode is a distinct explicit phase, never combined with environment actions', () => {
  const from=probe.indexOf('  const [inputFile, home, permission, ...extra]');
  const to=probe.indexOf('  assert(inputFile && home);',from);
  for(const phase of ['list','register','start','stop','restart','remove','dispatch','job']) {
    const scope={assert,runnerMode:false,accessMode:false,terminalMode:false,managementMode:false,matrixMode:false,
      Bun:{argv:['bun','probe','input','home','--allow-auth-transitions','--runner-phase','/private/runner','--allow-runner-'+phase]}};
    runInNewContext(probe.slice(from,to),scope);
    assert(scope.runnerMode && !scope.accessMode && !scope.terminalMode && !scope.managementMode && !scope.matrixMode);
  }
  for(const extra of [['--runner-phase'],['--runner-phase','/private/runner','--allow-runner-all'],
    ['--runner-phase','/private/runner','--allow-runner-stop','--allow-environment-access']]) {
    assert.throws(()=>runInNewContext(probe.slice(from,to),{assert,Bun:{argv:['bun','probe','input','home','--allow-auth-transitions',...extra]}}));
  }
});

test('runner branch authenticates distinct contexts and consumes one exact permit without environment journeys',async()=>{
  const from=probe.indexOf('  if (runnerRequest) {\n    const browser');
  const to=probe.indexOf("\n  } else {\n  stage = 'anonymous and native-cookie-only contexts';",from);
  assert(from > 0 && to > from);
  for(const scenario of ['confirmed','wrong-cookie','wrong-role','unconsumed','failure'] as const) {
    let invoked=0, closed=0;
    const visits:string[]=[];
    const deniedContext={setDefaultTimeout(){},async close(){closed++;}};
    const context={browser(){return {async newContext(options: {serviceWorkers: string}) {assert.equal(options.serviceWorkers,'block'); return deniedContext;}};}};
    function actorPage(index: number) {
      return {context(){return index === 0 ? context : deniedContext;},
        async goto(url:string){visits.push(url);},
        locator(){return {async waitFor(){},async isVisible(){return false;},async getAttribute(){return 'epoch';}};},
        async evaluate(){return {id:scenario === 'wrong-cookie' ? '1' : String(index+1),provider_id:String(index+1),operator:index === 0,admin:scenario === 'wrong-role' ? true : index === 1};}};
    }
    const page=actorPage(0), denied=actorPage(1);
    const scope={assert,context,page,origin:new URL('https://fixture.invalid'),presentationVersion:'epoch',
      runnerRequest:{phase:'stop',runner_id:'one',operator_id:'1'},input:{users:[{id:'1'},{id:'2'}]},
      extra:['--runner-phase','/private/runner','--allow-runner-stop'],result:{},
      interrupted:false,refusedRequest:false,runnerConfirmed:false,accessWrite:null as Intent | null,
      async guardedPage(selected:object){assert.equal(selected,deniedContext);return denied;},
      async nativeLogin(p:object,index:number){assert.equal(p,index === 0 ? page : denied);},
      async exerciseRunners(operator:object,nonoperator:object,_input:unknown,permission:string,
        permit:(actor:string,route:string,body:string)=>void,evidence:{outcome?:string}) {
        invoked++; assert.equal(operator,page); assert.equal(nonoperator,denied); assert.equal(permission,'--allow-runner-stop');
        const body='{"confirm_id":"one"}'; permit('1','/api/settings/runners/one/stop',body);
        assert(scope.accessWrite && scope.accessWrite.body === body && scope.accessWrite.page === page);
        if(scenario === 'failure') throw Error('synthetic operation failure');
        if(scenario !== 'unconsumed') scope.accessWrite=null;
        evidence.outcome='confirmed';
      }};
    const attempt=runInNewContext('(async()=>{'+probe.slice(from,to)+'\n}})()',scope);
    if(scenario === 'confirmed') {await attempt; assert(scope.runnerConfirmed);}
    else {await assert.rejects(async()=>await attempt); assert(!scope.runnerConfirmed);}
    assert.equal(scope.accessWrite,null); assert.equal(closed,1);
    assert.equal(invoked,scenario === 'wrong-cookie' || scenario === 'wrong-role' ? 0 : 1);
    assert(visits.every(url=>url === 'https://fixture.invalid/?soda-view=runners'));
  }
});

test('private anonymous probe requires native denial without Soda repository reads', async () => {
  const from = probe.indexOf("  stage = 'anonymous and native-cookie-only contexts';");
  const to = probe.indexOf('  await nativeLogin(page, 0);', from);
  assert(from > 0 && to > from);
  for (const [status, visible, reads, valid] of [[404, false, 0, true], [200, false, 0, false], [404, true, 0, false], [404, false, 1, false]]) {
    const scope = {assert, privateRepository: true, repoURL: 'https://fixture.invalid/owner/private', environmentReads: reads, result: {} as Evidence,
      page: {async goto() { return {status() { return status; }}; }, locator() { return {async isVisible() { return visible; }}; }}};
    const attempt = runInNewContext('(async () => {\n' + probe.slice(from, to) + '\n})()', scope);
    if (valid) { await attempt; assert.equal(scope.result.anonymous_repository, 'native 404; no Soda repository reads'); }
    else await assert.rejects(async () => await attempt);
  }
});

test('read-only probe records only a usable displayed own connection', async () => {
  const from = probe.indexOf("    if (await control('connection').isVisible()) {");
  const to = probe.indexOf('    const cookies = await context.cookies', from);
  assert(from > 0 && to > from);
  for (const disabled of [false, true]) {
    const scope = {assert, input: {users: [{id: '2'}], repository_id: '1'}, user: {id: '2'}, index: 0, result: {} as Evidence,
      control() { return {async isVisible() { return true; }, async inputValue() { return 'ssh alice@10.89.0.2'; },
        async innerText() { return 'Ed25519 host-key fingerprint: SHA256:' + 'A'.repeat(43); },
        async getAttribute() { return '#soda-command-1'; }, async isDisabled() { return disabled; }}; },
    };
    const run = runInNewContext('(async () => {\n'+probe.slice(from, to)+'\n})()', scope);
    if (disabled) await assert.rejects(async () => await run);
    else { await run; assert.equal(scope.result.own_connections?.[0]?.user_id, '2'); assert.equal(scope.result.own_connections?.[0]?.command, 'ssh alice@10.89.0.2'); }
  }
});

test('native probe guards every paused redirect before transmission', async () => {
  let paused: ((event: Paused) => Promise<void>) | undefined;
  const calls: Array<{method: string; params: {patterns?: Array<{requestStage: string}>; errorReason?: string; requestId?: string}}> = [];
  const lifetime = new Map<string, (frame?: object) => void>();
  const frame={};
  const page = {on(event: string, callback: (frame?: object)=>void) {lifetime.set(event,callback);}, mainFrame() {return frame;},
    async setViewportSize(size: {width: number; height: number}) { assert.equal(size.width, 1280); assert.equal(size.height, 900); }};
  const cdp = {
    on(name: string, handler: (event: Paused) => Promise<void>) { assert.equal(name, 'Fetch.requestPaused'); paused = handler; },
    async send(method: string, params: (typeof calls)[number]['params']) { calls.push({method, params}); },
  };
  const scope = {URL, origin: new URL('https://fixture.invalid'), input: {oauth_client_id: 'synthetic-client', users:[{id:'1'},{id:'2'}]},
    writes: new Set(['/user/login']), accessWrite: null as Intent | null, result: {} as Evidence, refusedRequest: false, interrupted: false,
    authorizations: 0, environmentReads: 0,
    context: {async newPage() { return page; }, async newCDPSession(p: typeof page) { assert.equal(p, page); return cdp; }}};
  assert.equal(await runInNewContext(probe.slice(start, end) + '\nguardedPage()', scope), page);
  assert(paused);
  assert.equal(calls[0]?.method, 'Fetch.enable');
  assert.equal(calls[0]?.params.patterns?.[0]?.requestStage, 'Request');
  await paused({requestId: 'allowed', redirectedRequestId: 'previous',
    request: {url: 'https://fixture.invalid/login/oauth/authorize?client_id=synthetic-client', method: 'GET'}});
  assert.equal(scope.authorizations, 1);
  assert.equal(calls.at(-1)?.method, 'Fetch.continueRequest');
  await paused({requestId: 'native-logout-redirect', request: {url: 'https://fixture.invalid/-/fetch-redirect', method: 'POST', postData: 'redirect=%2F'}});
  assert.equal(calls.at(-1)?.method, 'Fetch.continueRequest');
  for (const [url, method, postData] of [['https://outside.invalid/', 'GET'],
    ['https://fixture.invalid/-/fetch-redirect', 'POST'],
    ['https://fixture.invalid/-/fetch-redirect', 'POST', 'redirect=https%3A%2F%2Foutside.invalid'],
    ['https://fixture.invalid/-/fetch-redirect', 'POST', 'redirect=%2F&extra=value'],
    ['https://fixture.invalid/-/soda/api/environments', 'POST'],
    ['https://fixture.invalid/login/oauth/authorize?client_id=wrong', 'GET']] as const) {
    scope.refusedRequest = false;
    await paused({requestId: 'denied', redirectedRequestId: 'previous', request: {url, method, postData}});
    assert(scope.refusedRequest);
    assert.equal(calls.at(-1)?.method, 'Fetch.failRequest');
    assert.equal(calls.at(-1)?.params.errorReason, 'BlockedByClient');
    assert.equal(scope.authorizations, 1);
    assert.equal(scope.environmentReads, 0);
  }
  const intent = {path: '/-/soda/api/environments', body: '{"repository_id":"42"}', actor: '1'};
  const allowed = {url: 'https://fixture.invalid' + intent.path, method: 'POST', postData: intent.body, headers: {'X-Soda-Expected-User-ID': '1'}};
  for (const request of [{...allowed, postData: '{"repository_id":"43"}'}, {...allowed, headers: {'X-Soda-Expected-User-ID': '2'}}, {...allowed, url: allowed.url + '?extra=1'}]) {
    scope.refusedRequest = false;
    scope.accessWrite = intent;
    await paused({requestId: 'wrong-access', request});
    assert.equal(calls.at(-1)?.method, 'Fetch.failRequest');
    assert.equal(scope.accessWrite, null);
  }
  scope.refusedRequest = false;
  scope.accessWrite = intent;
  await paused({requestId: 'one-access', request: allowed});
  assert.equal(calls.at(-1)?.method, 'Fetch.continueRequest');
  assert.equal(scope.accessWrite, null);
  await paused({requestId: 'replay-access', request: allowed});
  assert.equal(calls.at(-1)?.method, 'Fetch.failRequest');
  scope.refusedRequest=false;
  const removeIntent = {path:'/-/soda/api/me/development-keys/9', body:'{}', actor:'1', method:'DELETE'};
  scope.accessWrite = removeIntent;
  const removal = {url:'https://fixture.invalid'+scope.accessWrite.path, method:'DELETE', postData:'{}', headers:allowed.headers};
  await paused({requestId:'wrong-method',request:{...removal,method:'POST'}});
  assert.equal(calls.at(-1)?.method,'Fetch.failRequest');
  assert.equal(scope.accessWrite,null);
  scope.refusedRequest=false; scope.accessWrite=removeIntent;
  await paused({requestId:'one-removal',request:removal});
  assert.equal(calls.at(-1)?.method,'Fetch.continueRequest');
  assert.equal(scope.accessWrite,null);
  await paused({requestId:'replay-removal',request:removal});
  assert.equal(calls.at(-1)?.method,'Fetch.failRequest');

  // Runner mutation admission is page-bound as well as actor/path/body-bound.
  const runnerIntent={path:'/-/soda/api/settings/runners/one/stop',body:'{"confirm_id":"one"}',actor:'1',page};
  const runnerWrite={url:originURL(runnerIntent.path),method:'POST',postData:runnerIntent.body,headers:allowed.headers};
  function originURL(route: string) {return 'https://fixture.invalid'+route;}
  for(const state of ['fresh','wrong-page','interrupted','refused','navigated','closed'] as const) {
    scope.interrupted=state === 'interrupted'; scope.refusedRequest=state === 'refused';
    scope.accessWrite={...runnerIntent,...(state === 'wrong-page' ? {page:{}} : {})};
    if(state === 'navigated') lifetime.get('framenavigated')?.(frame);
    if(state === 'closed') lifetime.get('close')?.();
    await paused({requestId:state,request:runnerWrite});
    assert.equal(calls.at(-1)?.method,state === 'fresh' ? 'Fetch.continueRequest' : 'Fetch.failRequest');
    assert.equal(scope.accessWrite,null);
  }
  scope.interrupted=false;
  const cancel={url:originURL('/-/soda/api/login/cancel'),method:'POST',postData:'{}',headers:{'X-Soda-Logout':'1','X-Soda-Expected-User-ID':'1','X-CSRF-Token':'c'.repeat(43)}};
  for(const request of [cancel,{...cancel,method:'GET',postData:undefined}]) {
    scope.refusedRequest=false;
    await paused({requestId:'bounded-cancel',request});
    assert.equal(calls.at(-1)?.method,'Fetch.continueRequest');
  }
  for(const request of [{...cancel,postData:'{"extra":1}'},{...cancel,url:cancel.url+'?extra=1'},
    {...cancel,headers:{...cancel.headers,'X-Soda-Logout':'0'}},
    {...cancel,headers:{...cancel.headers,'X-Soda-Expected-User-ID':'3'}},
    {...cancel,headers:{...cancel.headers,'X-CSRF-Token':'invalid'}}]) {
    scope.refusedRequest=false;
    await paused({requestId:'bad-cancel',request});
    assert.equal(calls.at(-1)?.method,'Fetch.failRequest');
  }
});

test('native browser refuses long private socket paths before starting a process', async () => {
  await assert.rejects(launchNativeBrowser({executablePath() { assert.fail('must not launch'); }, connectOverCDP() { assert.fail('must not connect'); }}, '/'+ 'x'.repeat(110), '/unused'), /socket path too long/);
});

test('native browser refuses an occupied socket without removing its owner', async () => {
  const run = await mkdtemp(path.join(tmpdir(), 'soda-cdp-'));
  const socket = path.join(run, 'cdp.sock');
  const owner = Bun.serve({unix: socket, fetch() {return new Response('test owner');}});
  try {
    await assert.rejects(launchNativeBrowser({executablePath() { assert.fail('must not launch'); }, connectOverCDP() { assert.fail('must not connect'); }}, run, run), /attachment failed/);
    assert((await lstat(socket)).isSocket());
    assert(owner.url.protocol === 'unix:');
  } finally {
    owner.stop(true);
    await rm(run, {recursive: true}); // Exact temporary test-owned directory only.
  }
});

test('Bun private browser pipe exchanges CDP frames and closes its owned process', {skip: Bun.env.SODA_BROWSER_PIPE_CHECK !== '1'}, async () => {
  const {chromium} = await import('playwright');
  // Short test-owned path also fits Linux's sockaddr_un limit on macOS.
  const run = await mkdtemp('/tmp/soda-cdp-smoke-');
  const browser = await launchNativeBrowser(chromium, run, run);
  try {
    const page = await browser.context.newPage();
    await page.goto('data:text/html,<title>Bun private pipe</title><main>Isolated transport fixture</main>');
    assert.equal(await page.title(), 'Bun private pipe');
  } finally {await browser.close();}
  console.log(`Private browser smoke profile retained at ${run}`);
});
