// Founder-demo end-to-end journey: Describe → Decide → Build → Use → Improve.
// Installed script (not a *.test.ts unit test). Runs phase by phase against the
// demo VM, with screenshots per phase and a clear failure naming the exact state.
// Resumes from a saved request id without repeating spends.
// Usage:
//   bun tests/installed/founder-demo-e2e.ts [--resume <intentId>] [--out <dir>] [--phase <describe|explore|decide|plan|check|accept|build|app|improve|all>]
// Run artifacts stay outside the repo under ~/Projects/.soda-demo-checks/e2e/run-*/.
import assert from 'node:assert/strict';
import {appendFile, mkdir, readFile, writeFile} from 'node:fs/promises';
import path from 'node:path';
import type {Page} from 'playwright';
import {launchNativeBrowser} from './native-browser.ts';
import {loginNativeForgejo} from './sodaspaces-controls.ts';
import {nativeSpacesMount, spacesContent} from './sodaspaces-journey-evidence.ts';

const ORIGIN = 'https://forge.northstar.soda.test';
const VM_IP = '192.168.2.166';
const STAGE = '/home/vince/Projects/.soda-demo-checks/demo-native-20261026T073430335Z';
const RUN_BASE = '/home/vince/Projects/.soda-demo-checks/e2e';
const REPOSITORY = '1';
const REPOSITORY_SLUG = 'maya/northstar-equipment';
const INTENT_DD00 = 'dd00cb3160885e5f707ec7d95';
const STOPPED_RUN = '00976679f6258efbc2c82b7ce1719b87';
// Owner decision (via Claude): apps run at their own address, not inside
// Soda Forge — so new requests say "a small web app". dd00cb31's recorded
// request text is history and stays untouched (resume never refills it).
const OBJECTIVE = `Replace our equipment-request spreadsheet with a small web app. Employees submit a request for a laptop, monitor or other equipment, with a reason and a needed-by date. Their manager approves or rejects it with a comment. Operations sees approved requests and marks them ordered, then delivered. Employees see the status of their own requests only.`;
const MAX_ISSUES = 1;
const PLANNING_CALLS = 12;
const PLAN_TIMEOUT_MS = 20 * 60 * 1000;

type Phase = 'describe' | 'explore' | 'decide' | 'answers' | 'raise' | 'plan' | 'retry' | 'poll' | 'check' | 'accept' | 'build' | 'retry-build' | 'allowance' | 'app' | 'improve';
const PHASES: Phase[] = ['describe', 'explore', 'decide', 'answers', 'raise', 'plan', 'retry', 'poll', 'check', 'accept', 'build', 'retry-build', 'allowance', 'app', 'improve'];

function args() {
  const out: {resume?: string; out?: string; phase?: string} = {};
  const argv = Bun.argv.slice(2);
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === '--resume') out.resume = argv[++i];
    else if (argv[i] === '--out') out.out = argv[++i];
    else if (argv[i] === '--phase') out.phase = argv[++i];
    else throw new Error(`unknown arg ${argv[i]}`);
  }
  if (out.resume) assert.match(out.resume, /^d[0-9a-f]{24}$/, 'resume must be a request id');
  if (out.phase && out.phase !== 'all' && !(PHASES as string[]).includes(out.phase)) throw new Error(`unknown phase ${out.phase}`);
  return out;
}

async function readMayaPassword(): Promise<string> {
  const raw = await readFile(STAGE + '/private/forge-accounts.json', 'utf8');
  const data = JSON.parse(raw) as {actors: {username: string; password: string}[]};
  const actor = data.actors.find((a) => a.username === 'maya');
  assert(actor?.password, 'maya account missing');
  return actor.password;
}

async function openE2E(profile: string, allowWrite: (method: string, path: string) => boolean) {
  const runHome = '/home/vince/Projects/.soda-demo-browser-20261026';
  process.env.TMPDIR = runHome + '/tmp';
  await mkdir(runHome + '/tmp', {recursive: true, mode: 0o700});
  const dir = `${runHome}/browser-${profile}`;
  await mkdir(dir, {recursive: true, mode: 0o700});
  const {chromium} = await import('playwright');
  const exe: string = chromium.executablePath();
  const wrapper = `${runHome}/chromium-${profile}`;
  await writeFile(wrapper, `#!/bin/sh\nexec ${exe} --host-resolver-rules="MAP forge.northstar.soda.test ${VM_IP}" "$@"\n`, {mode: 0o700});
  const b = await launchNativeBrowser(
    {executablePath: () => wrapper, connectOverCDP: chromium.connectOverCDP.bind(chromium)} as never,
    dir,
    runHome + '/home'
  );
  const blocked: string[] = [];
  const allowed: string[] = [];
  const failed: string[] = [];
  const guard = {armed: false};
  b.context.on('response', (response) => {
    const url = new URL(response.url());
    if (url.origin !== ORIGIN || response.ok()) return;
    if (!url.pathname.startsWith('/-/native/pages/soda/spaces/api/')) return;
    const key = `${response.status()} ${url.pathname.slice('/-/native/pages/soda/spaces/api'.length)}${url.search.slice(0, 120)}`;
    void response
      .text()
      .then((body) => {
        failed.push(`${key} :: ${body.slice(0, 300)}`);
      })
      .catch(() => {
        failed.push(`${key} :: <unreadable body>`);
      });
  });
  await b.context.route('**/*', async (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const method = req.method();
    if (!guard.armed && method === 'POST' && url.origin === ORIGIN && url.pathname === '/user/login') return route.continue();
    if (['GET', 'HEAD', 'OPTIONS'].includes(method)) return route.continue();
    if (guard.armed && url.origin === ORIGIN && allowWrite(method, url.pathname)) {
      allowed.push(`${method} ${url.pathname}`);
      return route.continue();
    }
    blocked.push(`${method} ${url.pathname}${url.search}`);
    return route.abort();
  });
  const page = await b.context.newPage();
  page.setDefaultTimeout(30000);
  await page.setViewportSize({width: 1920, height: 1080});
  await loginNativeForgejo(page, ORIGIN, 'maya', await readMayaPassword());
  guard.armed = true;
  return {page, blocked, allowed, failed, close: () => b.close().catch(() => {})};
}

async function shot(page: Page, file: string) {
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(800);
  await page.screenshot({path: file});
}

async function visibleText(page: Page, chars = 6000): Promise<string> {
  return page.evaluate((n: number) => {
    const parts: string[] = [];
    const walk = (root: Node) => {
      for (const node of Array.from(root.childNodes)) {
        if (node.nodeType === Node.TEXT_NODE) {
          const t = node.textContent?.trim();
          if (t && (node.parentElement?.checkVisibility({visibilityProperty: true}) ?? false)) parts.push(t);
        } else if (node instanceof Element) {
          if (['SCRIPT', 'STYLE', 'TEMPLATE', 'NOSCRIPT'].includes(node.tagName)) continue;
          if (node.shadowRoot) walk(node.shadowRoot);
          walk(node);
        }
      }
    };
    walk(document.body);
    return parts.join(' ').replace(/\s+/g, ' ').slice(0, n);
  }, chars);
}

async function phaseDescribe(page: Page, out: string, resume?: string): Promise<string> {
  if (resume) {
    await page.goto(`${ORIGIN}/-/native/pages/soda/spaces/api/`.replace(/\/-\/native\/pages\/soda\/spaces\/api\/$/, '') + `/workspace/requests/${resume}`);
    // The mount above is indirect; navigate via known workspace route instead.
    await page.goto(`${ORIGIN}/workspace/requests/${resume}`);
    await nativeSpacesMount(page);
    await shot(page, `${out}/describe-resumed.png`);
    return resume;
  }
  await page.goto(`${ORIGIN}/workspace?repository=${REPOSITORY}`);
  await nativeSpacesMount(page);
  const spaces = await spacesContent(page);
  await spaces.getByRole('button', {name: 'Work', exact: true}).first().waitFor({timeout: 30000});
  await spaces.getByRole('button', {name: 'Work', exact: true}).first().click();
  await page.waitForTimeout(1200);
  const neo = spaces.getByRole('button', {name: /new work/i});
  await neo.first().waitFor({timeout: 30000});
  const text = await visibleText(page, 2000);
  assert(
    await neo.first().isVisible(),
    `Describe: expected "+ New work" in Work view, found page text: ${text.slice(0, 400)}`
  );
  await neo.first().click();
  const box = spaces.getByLabel('Describe what you want to build', {exact: true});
  await box.waitFor({timeout: 30000});
  const empty = (await box.inputValue()) === '';
  assert(empty, `Describe: request box must start empty, found: ${(await box.inputValue()).slice(0, 120)}`);
  await box.fill(OBJECTIVE);
  assert(Buffer.byteLength(OBJECTIVE) < 4000, 'Describe: request must stay under 4000 bytes (UI-24)');
  await spaces.locator('details.bounds > summary').first().click();
  await spaces.getByLabel('Maximum work items', {exact: true}).fill(String(MAX_ISSUES));
  await spaces.getByLabel('Planning calls', {exact: true}).fill(String(PLANNING_CALLS));
  await shot(page, `${out}/describe-typed.png`);
  const saving = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 60000}
  );
  await spaces.getByRole('button', {name: 'Save draft', exact: true}).click();
  const saved = await saving;
  assert.equal(saved.status(), 201, `Describe: Save draft must return 201, got ${saved.status()}`);
  const body = (await saved.json()) as {intent?: string; id?: string};
  assert.match(String(body.intent ?? ''), /^d[0-9a-f]{24}$/, 'Describe: saved draft must report its intent');
  await page.waitForURL((u) => new URL(u).pathname.endsWith(`/workspace/requests/${body.intent}`), {timeout: 60000});
  await shot(page, `${out}/describe-saved.png`);
  return String(body.intent);
}

async function phaseExplore(page: Page, out: string, intent: string): Promise<void> {
  const spaces = await spacesContent(page);
  const exploring = page.waitForResponse(
    (r) => r.url().endsWith(`/api/factory/intents/${intent}/explore`) && r.request().method() === 'POST',
    {timeout: 300000}
  );
  await spaces.getByRole('button', {name: 'Explore ideas', exact: true}).click();
  const done = await exploring;
  const text = await visibleText(page, 3000);
  assert.equal(done.status(), 201, `Explore: POST explore must return 201, got ${done.status()} (page: ${text.slice(0, 300)})`);
  await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
  await shot(page, `${out}/explore-proposal.png`);
}

const DECIDE_CORRECTION = `First user action: an employee submits an equipment request with item (laptop/monitor/other), reason and needed-by date. Success: the request is saved; the employee sees it with status Submitted and the manager sees it pending approval. Operations sees only approved requests and marks them ordered then delivered. Employees see only their own requests.`;
// Appended to the Desired outcome itself to answer the material question in
// the request text (the product's own suggested path: "Edit the desired
// outcome ... to explain the missing behavior, then update the proposal").
const OBJECTIVE_SUFFIX = `\n\nFirst action and success: an employee submits an equipment request with item, reason and needed-by date; the request is saved and visible as Submitted to the employee and pending to the manager.`;

async function phaseDecide(page: Page, out: string): Promise<void> {
  const spaces = await spacesContent(page);
  await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
  // List pending questions without answering blindly: the run fails naming them.
  const ideas = spaces.locator('section.intent-review > article');
  const pending: string[] = [];
  for (let i = 0; i < (await ideas.count()); i++) {
    const article = ideas.nth(i);
    const choice = article.locator('select[aria-label^="Choice: "]');
    if ((await choice.count()) && (await choice.inputValue()) === 'pending') {
      pending.push(((await article.locator('p').first().textContent()) ?? '').slice(0, 160));
    }
  }
  const challenges = spaces.locator('section[aria-label="Questions needing your answer"] article');
  for (let i = 0; i < (await challenges.count()); i++) {
    const article = challenges.nth(i);
    if (!((await article.textContent()) ?? '').includes('Answered:')) {
      pending.push('Q: ' + (((await article.locator('p').first().textContent()) ?? '').slice(0, 160)));
    }
  }
  await shot(page, `${out}/decide-questions.png`);
  assert.equal(
    pending.length,
    0,
    `Decide: product asks ${pending.length} question(s) needing recorded owner answers: ${pending.join(' | ').slice(0, 800)}`
  );
  // Answer the material assumption through the product's own correction box,
  // then Update proposal. Recorded owner correction (equipment spirit) is above.
  // The box lives inside the closed "Recorded choices" disclosure: open it first.
  // After the FUX09 producer fix is installed, the re-explore must render a
  // "Questions needing your answer" section (one article per pending material
  // assumption); the test asserts it instead of silently continuing generic.
  const correction = spaces.getByLabel(/^Correct the automatic choice:/);
  if ((await correction.count()) > 0) {
    if (!(await correction.first().isVisible())) {
      await spaces.locator('summary', {hasText: 'Recorded choices'}).first().click();
      await correction.first().waitFor({state: 'visible', timeout: 15000});
    }
    const current = await correction.first().inputValue();
    if (!current.trim()) {
      await correction.first().fill(DECIDE_CORRECTION);
      await shot(page, `${out}/decide-correction-filled.png`);
      const saving = page.waitForResponse(
        (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
        {timeout: 120000}
      );
      const explored = page.waitForResponse(
        (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/explore') && r.request().method() === 'POST',
        {timeout: 300000}
      );
      await spaces.getByRole('button', {name: 'Update proposal'}).click();
      const saved = await saving;
      assert.equal(saved.status(), 201, `Decide: Update proposal save must return 201, got ${saved.status()}`);
      const re = await explored;
      assert.equal(re.status(), 201, `Decide: Update proposal explore must return 201, got ${re.status()}`);
      await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
      await shot(page, `${out}/decide-updated.png`);
      // Read-only check that the fixed producer asked its questions: every
      // pending material assumption must render an answerable article.
      const questions = spaces.locator('section[aria-label="Questions needing your answer"] article');
      await shot(page, `${out}/decide-questions-after-update.png`);
      assert(
        (await questions.count()) > 0,
        'Decide: after Update proposal the fixed producer must render "Questions needing your answer" for the pending material assumptions'
      );
    }
  }
  // Post-fix path: the correction stands but no questions render and the
  // request is still unbound (the pre-fix deadlock state). Answer the
  // material question in the Desired outcome itself, raise the allowance to
  // the approved 20, and re-explore under the fixed producer.
  const questionsNow = spaces.locator('section[aria-label="Questions needing your answer"] article');
  const chooseNow = spaces.getByRole('button', {name: 'Choose repository'});
  if ((await questionsNow.count()) === 0 && (await chooseNow.count()) > 0 && !(await chooseNow.first().isEnabled())) {
    await spaces.locator('details.bounds > summary').first().click();
    const calls = spaces.getByLabel('Planning calls', {exact: true});
    await calls.fill('20');
    await spaces.getByRole('button', {name: 'Update proposal'}).click();
    await page.waitForTimeout(2000);
    await shot(page, `${out}/decide-allowance-raised.png`);
    const objective = spaces.getByLabel('Desired outcome', {exact: true});
    const before = await objective.inputValue();
    assert(!before.includes('First action and success'), 'Decide: objective suffix already applied, refusing to double-apply');
    const grown = before + OBJECTIVE_SUFFIX;
    assert(Buffer.byteLength(grown) < 4000, 'Decide: grown objective must stay under 4000 bytes (UI-24)');
    await objective.fill(grown);
    const saving = page.waitForResponse(
      (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
      {timeout: 120000}
    );
    const explored = page.waitForResponse(
      (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/explore') && r.request().method() === 'POST',
      {timeout: 300000}
    );
    await spaces.getByRole('button', {name: 'Update proposal'}).click();
    assert.equal((await saving).status(), 201, 'Decide: objective-update save must return 201');
    assert.equal((await explored).status(), 201, 'Decide: objective-update explore must return 201');
    await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
    await shot(page, `${out}/decide-objective-updated.png`);
    const questionsAfter = spaces.locator('section[aria-label="Questions needing your answer"] article');
    assert(
      (await questionsAfter.count()) > 0,
      'Decide: after the objective update the fixed producer must render "Questions needing your answer"'
    );
  }
}

async function phaseAnswerQuestions(page: Page, out: string): Promise<void> {
  const spaces = await spacesContent(page);
  await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
  const challenges = spaces.locator('section[aria-label="Questions needing your answer"] article');
  const count = await challenges.count();
  assert(count > 0, 'Decide-answers: no questions to answer; nothing to do');
  for (let i = 0; i < count; i++) {
    const article = challenges.nth(i);
    const text = ((await article.textContent()) ?? '').replace(/\s+/g, ' ');
    await article.scrollIntoViewIfNeeded();
    if (/first version within the existing repository/i.test(text)) {
      // Recorded answer: keep the first version in the existing repository.
      await article.getByRole('button', {name: /^Use recommended resolution/}).click();
    } else if (/first user action/i.test(text)) {
      // Recorded answer: the equipment-request first action + success text.
      await article.getByLabel(/^Explain a correction to /).fill(DECIDE_CORRECTION);
    } else {
      assert.fail(`Decide-answers: unrecorded product question, refusing to invent an answer: ${text.slice(0, 300)}`);
    }
  }
  await shot(page, `${out}/decide-answers-given.png`);
  const saving = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 120000}
  );
  const explored = page.waitForResponse(
    (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/explore') && r.request().method() === 'POST',
    {timeout: 300000}
  );
  await spaces.getByRole('button', {name: 'Update proposal'}).click();
  assert.equal((await saving).status(), 201, 'Decide-answers: answer save must return 201');
  assert.equal((await explored).status(), 201, 'Decide-answers: answer explore must return 201');
  await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
  await shot(page, `${out}/decide-answers-updated.png`);
}

async function phaseBind(page: Page, out: string): Promise<string | null> {
  const spaces = await spacesContent(page);
  const choose = spaces.getByRole('button', {name: 'Choose repository'});
  if ((await choose.count()) === 0 || !(await choose.first().isEnabled())) return null;
  await choose.click();
  const radio = spaces.getByRole('radio', {name: new RegExp(REPOSITORY_SLUG.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))});
  await radio.waitFor({timeout: 30000});
  await radio.check();
  await shot(page, `${out}/bind-picker.png`);
  const bound = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 120000}
  );
  const planning = page.waitForResponse(
    (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/plan') && r.request().method() === 'POST',
    {timeout: 180000}
  );
  await spaces.getByRole('button', {name: /^(Continue|Open project)$/}).click();
  const boundReply = await bound;
  assert.equal(boundReply.status(), 201, `Bind: repository bind must return 201, got ${boundReply.status()}`);
  const revision = String(((await boundReply.json()) as {id?: string}).id ?? '');
  assert.match(revision, /^d[0-9a-f]{24}$/, 'Bind: bind must report its revision');
  await writeFile(`${out}/bound-revision.json`, JSON.stringify({revision}, null, 2));
  const planReply = await planning;
  const planBody = (await planReply.json().catch(() => ({}))) as {
    error?: {code?: string; message?: string};
    revision_id?: string;
  };
  await writeFile(`${out}/plan-admission.json`, JSON.stringify({status: planReply.status(), revision, body: planBody}, null, 2));
  assert(
    [201, 202].includes(planReply.status()),
    `Plan: admission must be 201/202, got ${planReply.status()} (${planBody.error?.code ?? 'no code'}: ${(planBody.error?.message ?? 'no message').slice(0, 300)})`
  );
  await shot(page, `${out}/plan-admitted.png`);
  return revision;
}

async function phaseRaise(page: Page, out: string, amount: number): Promise<void> {
  const spaces = await spacesContent(page);
  await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
  await spaces.locator('details.bounds > summary').first().click();
  const calls = spaces.getByLabel('Planning calls', {exact: true});
  const before = await calls.inputValue();
  if (before === String(amount)) {
    await shot(page, `${out}/decide-allowance-already.png`);
    return;
  }
  await calls.fill(String(amount));
  const saving = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 120000}
  );
  await spaces.getByRole('button', {name: 'Update proposal'}).click();
  assert.equal((await saving).status(), 201, `Raise: allowance save must return 201`);
  await shot(page, `${out}/decide-allowance-raised.png`);
}

async function phaseStartPlanBypass(page: Page, out: string, _intent: string): Promise<string> {
  // TEMPORARY BYPASS, see e2e/PLAN-RETRY-GO.md (revised): attention has no
  // Continue to planning button (page bug, fixed separately), and a second
  // plan POST on the settled revision is refused as a closed attempt. The
  // page retries by saving a successor draft and planning the new revision,
  // so this calls the page's own `soda-intent` methods — no hand-built
  // bodies (if they are unreachable, this fails instead of improvising).
  // Replace with a real click once the page fix lands. Single use only.
  const spaces = await spacesContent(page);
  await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
  const planning = page.waitForResponse(
    (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/plan') && r.request().method() === 'POST',
    {timeout: 180000}
  );
  const sent = await page.evaluate(async () => {
    const roots: (Document | ShadowRoot)[] = [document];
    for (const el of Array.from(document.querySelectorAll('*'))) if ((el as Element).shadowRoot) roots.push((el as Element).shadowRoot as ShadowRoot);
    let el: any = null;
    for (const root of roots) {
      el = (root as Document).querySelector?.('soda-intent');
      if (el) break;
    }
    if (!el) return {ok: false as const, error: 'no soda-intent element'};
    if (typeof el.save !== 'function' || typeof el.startPlanning !== 'function') {
      return {ok: false as const, error: 'page methods unreachable (save/startPlanning); stopping per note'};
    }
    try {
      const signal = new AbortController().signal;
      const repo = el.revision?.repository;
      if (repo !== '1') return {ok: false as const, error: `unexpected repository ${String(repo)}`};
      const bound = await el.save('draft', repo, signal);
      await el.startPlanning(bound, signal);
      return {ok: true as const, revision: String(bound?.id ?? '')};
    } catch (err) {
      return {ok: false as const, error: String(err).slice(0, 300)};
    }
  });
  assert(sent.ok, `Plan retry: page method failed: ${(sent as {error?: string}).error ?? 'unknown'}`);
  const planReply = await planning;
  const planBody = (await planReply.json().catch(() => ({}))) as {
    error?: {code?: string; message?: string};
    revision_id?: string;
  };
  const revision = (sent as {revision: string}).revision;
  assert.match(revision, /^d[0-9a-f]{24}$/, 'Plan retry: successor draft must report its revision');
  await writeFile(`${out}/plan-admission.json`, JSON.stringify({status: planReply.status(), revision, via: 'page-save-startPlanning', body: planBody}, null, 2));
  assert(
    [201, 202].includes(planReply.status()),
    `Plan: admission must be 201/202, got ${planReply.status()} (${planBody.error?.code ?? 'no code'}: ${(planBody.error?.message ?? 'no message').slice(0, 300)})`
  );
  await shot(page, `${out}/plan-admitted.png`);
  return revision;
}

async function phaseStartPlan(page: Page, out: string): Promise<void> {
  const spaces = await spacesContent(page);
  const cont = spaces.getByRole('button', {name: 'Continue to planning', exact: true});
  await cont.waitFor({timeout: 60000});
  assert(await cont.isEnabled(), 'Plan: Continue to planning must be enabled on the bound draft');
  const planning = page.waitForResponse(
    (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/plan') && r.request().method() === 'POST',
    {timeout: 180000}
  );
  await cont.click();
  const planReply = await planning;
  const planBody = (await planReply.json().catch(() => ({}))) as {
    error?: {code?: string; message?: string};
    revision_id?: string;
  };
  await writeFile(`${out}/plan-admission.json`, JSON.stringify({status: planReply.status(), body: planBody}, null, 2));
  assert(
    [201, 202].includes(planReply.status()),
    `Plan: admission must be 201/202, got ${planReply.status()} (${planBody.error?.code ?? 'no code'}: ${(planBody.error?.message ?? 'no message').slice(0, 300)})`
  );
  await shot(page, `${out}/plan-admitted.png`);
}

async function phasePlan(page: Page, out: string, intent: string, revision: string | null): Promise<void> {
  const spaces = await spacesContent(page);
  const started = Date.now();
  let want = revision;
  const deadline = started + PLAN_TIMEOUT_MS;
  for (;;) {
    assert(Date.now() < deadline, 'Plan: finite planning observation expired; retain the admitted run');
    const refreshed = page.waitForResponse(
      (r) => r.url().endsWith(`/api/factory/intents/${intent}/plan`) && r.request().method() === 'GET',
      {timeout: 60000}
    );
    await spaces.getByRole('button', {name: 'Refresh status', exact: true}).click();
    const response = await refreshed;
    assert.equal(response.status(), 200, `Plan: Refresh status must return 200, got ${response.status()}`);
    const plan = (await response.json()) as {state?: string; revision?: string};
    if (!want) want = plan.revision ?? null;
    assert(want, 'Plan: refresh must report a revision');
    assert.equal(plan.revision, want, 'Plan: refresh must report the bound revision');
    if (plan.state !== 'pending') {
      await shot(page, `${out}/plan-${plan.state}.png`);
      await writeFile(`${out}/plan.json`, JSON.stringify({intent, revision, state: plan.state, startedUnixMS: started, endedUnixMS: Date.now(), plan}, null, 2));
      assert.equal(plan.state, 'completed', `Plan: planning ended in state "${plan.state}", not completed`);
      return;
    }
    await page.waitForTimeout(5000);
  }
}

async function phaseAppUse(page: Page, out: string): Promise<void> {
  // Owner decision: the app runs at its OWN address in its OWN browser tab.
  // Forge shows only status/version plus an Open app link. This phase never
  // asserts an embedded frame and builds no in-Forge embedding: it opens the
  // app's own origin in a new page and uses it there.
  await page.goto(`${ORIGIN}/workspace?repository=${REPOSITORY}`);
  await nativeSpacesMount(page);
  const spaces = await spacesContent(page);
  await spaces.getByRole('button', {name: 'App', exact: true}).first().click();
  await spaces.getByText('Applications').first().waitFor({timeout: 30000});
  const found = await page.evaluate(() => {
    const links: string[] = [];
    let iframes = 0;
    const scan = (root: Document | ShadowRoot) => {
      for (const a of Array.from(root.querySelectorAll('a[href]'))) {
        if (/open app/i.test(a.textContent || '')) links.push((a as HTMLAnchorElement).href);
      }
      iframes += root.querySelectorAll('iframe').length;
      for (const el of Array.from(root.querySelectorAll('*'))) if ((el as Element).shadowRoot) scan((el as Element).shadowRoot as ShadowRoot);
    };
    scan(document);
    return {links, iframes, url: location.href};
  });
  await writeFile(`${out}/app-discovery.json`, JSON.stringify(found, null, 2));
  assert(found.links.length > 0, 'App: no "Open app" link on the Forge side; cannot reach the app at its own address');
  const href = found.links[0];
  const appOrigin = new URL(href).origin;
  assert(
    appOrigin !== new URL(page.url()).origin,
    `App: the Open app link must leave Forge for the app's own address, got same-origin ${href.slice(0, 200)}`
  );
  const appPage = await page.context().newPage();
  try {
    await appPage.setViewportSize({width: 1920, height: 1080});
    const response = await appPage.goto(href, {waitUntil: 'networkidle', timeout: 60000});
    assert(response && response.ok(), `App: the app's own address must load, got HTTP ${response?.status()}`);
    await appPage.waitForTimeout(1500);
    await appPage.screenshot({path: `${out}/app-own-tab.png`});
    const text = (await appPage.evaluate(() => document.body.innerText || '')).replace(/\s+/g, ' ').slice(0, 2000);
    await writeFile(`${out}/app-text.txt`, text);
    assert(text.length > 0, 'App: the app tab rendered no text');
  } finally {
    await appPage.close().catch(() => {});
  }
  await shot(page, `${out}/app-forge-side.png`);
}

/**
 * Build delivery signal. The product never renders per-run harness/model
 * names in Feed: the Builder run rows show role/issue/outcome, and the
 * run watch view shows status plus last output. Muse provenance for the
 * journey is the installed Factory policy (rev 8: Muse 1.4.14,
 * muse-spark-1.3, contributor on), verified read-only against the run
 * records. The product's own delivery claim is the F10
 * "Application outcomes verified" post carrying an "Open project app"
 * button, so the follow-up waits for exactly that card and fails on any
 * newly posted failure card instead of asserting markers the UI has.
 */
interface FeedCard {
  id: string;
  status: string;
  role: string;
  text: string;
  appKeys: string[];
}

/** Owner-approved allowance raise (ALLOWANCE-RETRY-GO.md): repo 1 coder-time
 * sponsorship 120 -> 480 minutes through Settings → SodaOS → Factory's own
 * "Save sponsorship" control. Fails naming the exact state when the page
 * offers no such control; touches nothing else on the form. */
async function phaseAllowance(page: Page, out: string): Promise<void> {
  await page.goto(`${ORIGIN}/workspace?repository=${REPOSITORY}`);
  await nativeSpacesMount(page);
  const spaces = await spacesContent(page);
  const settings = spaces.getByRole('button', {name: 'Project settings'});
  await settings.first().waitFor({timeout: 30000});
  await settings.first().click();
  await page.waitForTimeout(3000);
  const factoryTab = spaces.getByRole('tab', {name: 'Factory'});
  assert((await factoryTab.count()) >= 1, 'Allowance: no Factory tab in project settings');
  await factoryTab.first().click();
  await page.waitForTimeout(3000);
  await shot(page, `${out}/allowance-settings.png`);
  const text = await visibleText(page, 8000);
  await writeFile(`${out}/allowance-settings-text.txt`, text);
  assert(/Allowance minutes/i.test(text), `Allowance: no sponsorship allowance control in Factory settings (page: ${text.slice(0, 400)})`);
  // The sponsorship submit requires the provider connection selected. Never
  // guess: proceed only when the inventory offers exactly one connection.
  const readOptions = () =>
    page.evaluate(() => {
      const select = document.querySelector('select[name="selected_connection"]');
      if (!select) return null;
      return Array.from(select.querySelectorAll('option')).map((o) => ({
        value: (o as HTMLOptionElement).value,
        label: (o.textContent || '').replace(/\s+/g, ' ').trim(),
      }));
    });
  let options = await readOptions();
  if (options !== null && options.filter((o) => o.value !== '').length === 0) {
    // Inventory can load behind the form: use the page's own refresh once,
    // then give the list a bounded wait before calling it a refusal.
    const refresh = spaces.getByRole('button', {name: 'Refresh status'});
    if ((await refresh.count()) > 0) await refresh.first().click();
    try {
      await page.waitForFunction(
        () => document.querySelectorAll('select[name="selected_connection"] option').length > 1,
        {timeout: 20000}
      );
    } catch {
      // Fall through to the refusal below with the recorded state.
    }
    options = await readOptions();
  }
  await writeFile(`${out}/allowance-connections.json`, JSON.stringify(options, null, 2));
  assert(options !== null, 'Allowance: provider-connection select not found');
  const candidates = (options as {value: string; label: string}[]).filter((o) => o.value !== '');
  assert(candidates.length >= 1, `Allowance: no provider connection in the account inventory (page: ${text.slice(0, 300)})`);
  assert(candidates.length === 1, `Allowance: ${candidates.length} provider connections; owner must choose: ${JSON.stringify(candidates)}`);
  const only = candidates[0];
  assert(only !== undefined);
  await spaces.locator('select[name="selected_connection"]').first().selectOption(only.value);
  // Selecting the connection reloads grants and the existing sponsorship;
  // wait for the grant list before touching the form.
  try {
    await page.waitForFunction(
      () => document.querySelectorAll('select[name="grant_id"] option').length > 1,
      {timeout: 30000}
    );
  } catch {
    const afterGrant = await visibleText(page, 2000);
    assert.fail(`Allowance: no grant loaded for the selected connection (page: ${afterGrant.slice(0, 300)})`);
  }
  const minutes = spaces.getByLabel(/Allowance minutes/i);
  assert((await minutes.count()) >= 1, 'Allowance: allowance minutes input not found');
  const before = await minutes.first().inputValue();
  await minutes.first().fill('480');
  const acted = page.waitForResponse(
    (r) => /\/api\/repositories\/1\/factory\/sponsorships\/.+$/.test(new URL(r.url()).pathname) && r.request().method() === 'PUT',
    {timeout: 120000}
  );
  const save = spaces.getByRole('button', {name: 'Save sponsorship'}).first();
  assert(await save.isEnabled(), 'Allowance: Save sponsorship is disabled for this account');
  await save.click();
  let reply;
  try {
    reply = await acted;
  } catch (error) {
    const refused = await visibleText(page, 3000);
    await writeFile(`${out}/allowance-refused.txt`, refused);
    await shot(page, `${out}/allowance-refused.png`);
    throw new Error(`Allowance: Save sponsorship sent no request (form refusal: ${refused.slice(0, 400)})`, {cause: error});
  }
  const body = await reply.json().catch(() => ({}));
  await writeFile(`${out}/allowance-reply.json`, JSON.stringify({before, after: '480', status: reply.status(), body}, null, 2));
  await shot(page, `${out}/allowance-saved.png`);
  assert(reply.ok(), `Allowance: sponsorship save must succeed, got HTTP ${reply.status()}: ${JSON.stringify(body).slice(0, 300)}`);
}

const BUILD_TIMEOUT_MS = 35 * 60 * 1000;

/** Retry the cancelled build ONCE through the product's own Retry control
 * (RetryRun path for stop-before-start). Fails naming the exact state when
 * the page offers no usable Retry for the stopped run. */
async function phaseRetryBuild(page: Page, out: string, run: string): Promise<void> {
  // The run interventions (incl. "Retry run") render in the per-issue work
  // detail on the request route, not in the project Work view.
  await page.goto(`${ORIGIN}/workspace/requests/${INTENT_DD00}`);
  await nativeSpacesMount(page);
  const spaces = await spacesContent(page);
  await page.waitForTimeout(4000);
  await shot(page, `${out}/retry-work.png`);
  const text = await visibleText(page, 8000);
  await writeFile(`${out}/retry-work-text.txt`, text);
  assert(
    text.includes(run.slice(0, 8)) || /Earlier runs|Retry run/i.test(text),
    `Retry: expected the stopped run ${run.slice(0, 8)} or an Earlier-runs/Retry control in Work (page: ${text.slice(0, 300)})`
  );
  const buttons = await page.evaluate(() => {
    const out: {text: string; disabled: boolean; section: string}[] = [];
    for (const b of Array.from(document.querySelectorAll('button'))) {
      const t = (b.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 120);
      if (!t) continue;
      const section = b.closest('details, section, li, article');
      const heading = section?.querySelector('summary, h1, h2, h3')?.textContent?.replace(/\s+/g, ' ').trim().slice(0, 120) ?? '';
      out.push({text: t, disabled: (b as HTMLButtonElement).disabled, section: heading});
    }
    return out;
  });
  await writeFile(`${out}/retry-buttons.json`, JSON.stringify(buttons, null, 2));
  const retry = spaces.getByRole('button', {name: 'Retry run'});
  const count = await retry.count();
  assert(count >= 1, `Retry: no "Retry run" button offered in Work (buttons: ${JSON.stringify(buttons).slice(0, 600)})`);
  assert(count === 1, `Retry: ${count} "Retry run" buttons; cannot tell which one retries run ${run.slice(0, 8)}`);
  const button = retry.first();
  assert(await button.isEnabled(), `Retry: the "Retry run" button is disabled for run ${run.slice(0, 8)}`);
  const acted = page.waitForResponse(
    (r) => /\/api\/factory\/runs\/[0-9a-f]{32}\/actions$/.test(new URL(r.url()).pathname) && r.request().method() === 'POST',
    {timeout: 120000}
  );
  await button.click();
  const reply = await acted;
  const body = await reply.json().catch(() => ({}));
  await writeFile(`${out}/retry-reply.json`, JSON.stringify({status: reply.status(), body}, null, 2));
  await shot(page, `${out}/retry-acted.png`);
  assert(reply.ok(), `Retry: run action must succeed, got HTTP ${reply.status()}: ${JSON.stringify(body).slice(0, 300)}`);
}

async function phaseBuildFollow(page: Page, out: string, intent: string, launchGraceMs = 0): Promise<void> {
  const started = Date.now();
  const deadline = started + BUILD_TIMEOUT_MS;
  const launchBy = launchGraceMs > 0 ? started + launchGraceMs : 0;
  const baseline = new Set<string>();
  const lastStatus = new Map<string, string>();
  let baselined = false;
  let launched = launchGraceMs === 0;
  await writeFile(`${out}/build-milestones.log`, `started=${new Date(started).toISOString()} intent=${intent} launchGraceMs=${launchGraceMs}\n`);
  for (;;) {
    assert(Date.now() < deadline, 'Build: 35-minute follow window expired; retain the last observed state');
    await page.goto(`${ORIGIN}/workspace?repository=${REPOSITORY}`);
    await nativeSpacesMount(page);
    const spaces = await spacesContent(page);
    await spaces.getByRole('button', {name: 'Feed', exact: true}).first().click();
    // Feed cards render asynchronously and the activity read can fail
    // transiently (e.g. right after an install); retry the read through
    // the page's own control before failing on an empty Feed.
    let loaded = false;
    for (let attempt = 0; attempt < 4 && !loaded; attempt++) {
      try {
        await page.waitForFunction(() => document.querySelectorAll('article.soda-feed-post').length > 0, {timeout: 30000});
        loaded = true;
      } catch {
        const retry = spaces.getByRole('button', {name: 'Retry loading activity'});
        if ((await retry.count()) === 0) break;
        await retry.first().click();
      }
    }
    if (!loaded) {
      await shot(page, `${out}/build-feed-empty.png`);
      const empty = await visibleText(page, 2000);
      await writeFile(`${out}/build-feed-empty.txt`, empty);
      assert.fail(`Build: no Feed post cards rendered after retries (page: ${empty.slice(0, 400)})`);
    }
    await shot(page, `${out}/build-feed.png`);
    // Scope to Feed post cards only: page chrome (navigation, run rows,
    // session lists) carries stale words like "ended" or "failed" from
    // older attempts and must never decide this build.
    const cards = await page.evaluate((): FeedCard[] => {
      const found: FeedCard[] = [];
      for (const el of Array.from(document.querySelectorAll('article.soda-feed-post'))) {
        const text = (el.textContent || '').replace(/\s+/g, ' ').slice(0, 2000);
        const appKeys = Array.from(el.querySelectorAll('button[data-feed-open-app]')).map(
          (b) => b.getAttribute('data-feed-open-app') || ''
        );
        found.push({
          id: el.id || '',
          status: el.getAttribute('data-status') || '',
          role: el.getAttribute('data-role') || '',
          text,
          appKeys,
        });
      }
      return found;
    });
    await writeFile(`${out}/build-feed-cards.json`, JSON.stringify(cards, null, 2));
    assert(cards.length > 0, 'Build: no Feed post cards rendered yet; the journey Feed is empty');
    if (!baselined) {
      for (const card of cards) {
        baseline.add(card.id);
        lastStatus.set(card.id, card.status);
      }
      baselined = true;
      await appendFile(`${out}/build-milestones.log`, `${new Date().toISOString()} baseline: ${cards.length} cards\n`);
    }
    const fresh = cards.filter((card) => card.id && !baseline.has(card.id));
    for (const card of fresh) {
      baseline.add(card.id);
      lastStatus.set(card.id, card.status);
      await appendFile(
        `${out}/build-milestones.log`,
        `${new Date().toISOString()} new-card status=${card.status} role=${card.role}: ${card.text.slice(0, 300)}\n`
      );
      if (card.status === 'failure') {
        assert.fail(`Build: a new Feed card reports failure (status=failure role=${card.role}): ${card.text.slice(0, 500)}`);
      }
    }
    // A baselined card can flip to failure in place (same id, e.g. the
    // retry assignment record going waiting -> needs attention). A flip
    // is terminal evidence; pre-existing baseline failures stay ignored.
    for (const card of cards) {
      if (!card.id || !baseline.has(card.id)) continue;
      const previous = lastStatus.get(card.id);
      lastStatus.set(card.id, card.status);
      if (previous !== undefined && previous !== 'failure' && card.status === 'failure') {
        await appendFile(
          `${out}/build-milestones.log`,
          `${new Date().toISOString()} flipped-card ${previous} -> failure role=${card.role}: ${card.text.slice(0, 300)}\n`
        );
        assert.fail(`Build: a Feed card flipped to failure (${previous} -> failure role=${card.role}): ${card.text.slice(0, 500)}`);
      }
    }
    if (!launched) {
      if (fresh.some((card) => /Started|assigned|Running/i.test(card.text))) {
        launched = true;
        await appendFile(`${out}/build-milestones.log`, `${new Date().toISOString()} launched: a new run-start card posted\n`);
      } else if (launchBy > 0 && Date.now() > launchBy) {
        assert.fail(
          'Build: retried run did not launch within ~6 minutes (no new run-start card); same hold as the stopped run — stopping per go-ahead'
        );
      }
    }
    const appCards = cards.filter((card) => card.appKeys.length > 0);
    if (appCards.length > 1) {
      assert.fail(
        `Build: ${appCards.length} "Open project app" cards in Feed; cannot tell which delivery is the journey's`
      );
    }
    if (appCards.length === 1) {
      const delivery = appCards[0];
      assert(
        /Application outcomes verified/i.test(delivery.text),
        `Build: the app card does not carry the verified-outcomes claim: ${delivery.text.slice(0, 300)}`
      );
      await writeFile(`${out}/build-delivery.json`, JSON.stringify(delivery, null, 2));
      await appendFile(
        `${out}/build-milestones.log`,
        `${new Date().toISOString()} app-ready key=${delivery.appKeys[0]}: ${delivery.text.slice(0, 300)}\n`
      );
      return;
    }
    await page.waitForTimeout(30000);
  }
}

async function phaseCheck(page: Page, out: string, intent: string): Promise<void> {
  const spaces = await spacesContent(page);
  const checking = page.waitForResponse(
    (r) => r.url().endsWith(`/api/factory/intents/${intent}/plan/review`) && r.request().method() === 'POST',
    {timeout: 180000}
  );
  await spaces.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
  const checked = await checking;
  assert.equal(checked.status(), 200, `Check: walkthrough review must return 200, got ${checked.status()}`);
  const review = (await checked.json()) as {consistency?: {mismatches?: unknown[]}};
  const mismatches = (review.consistency as {mismatches?: unknown[]})?.mismatches ?? [];
  await shot(page, `${out}/check-walkthrough.png`);
  assert.equal(mismatches.length, 0, `Check: JEV found ${mismatches.length} mismatch(s); correct before Accept`);
}

async function phaseAccept(page: Page, out: string, _intent: string): Promise<void> {
  const spaces = await spacesContent(page);
  const accepted = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 120000}
  );
  const generated = page.waitForResponse(
    (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/generate') && r.request().method() === 'POST',
    {timeout: 180000}
  );
  await spaces.getByRole('button', {name: 'Accept and build', exact: true}).click();
  const acceptReply = await accepted;
  assert.equal(acceptReply.status(), 201, `Accept: accept must return 201, got ${acceptReply.status()}`);
  const genReply = await generated;
  assert.equal(genReply.status(), 200, `Accept: generate must return 200, got ${genReply.status()}`);
  const generation = (await genReply.json()) as {state?: string};
  await shot(page, `${out}/accept-generated.png`);
  assert.equal(generation.state, 'completed', `Accept: generation ended in "${generation.state}", not completed`);
}

async function main() {
  const {resume, out, phase} = args();
  const stamp = new Date().toISOString().replace(/[:.]/g, '').slice(0, 15);
  const dir = out ?? `${RUN_BASE}/run-${stamp}`;
  await mkdir(dir, {recursive: true, mode: 0o700});
  const only = phase ?? 'all';
  const allowWrite = (method: string, p: string) => {
    if (method === 'POST' && /\/api\/factory\/intents$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intent-allowances$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/explore$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/plan$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/plan\/review$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/generate$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/runs\/[0-9a-f]{32}\/actions$/.test(p)) return true;
    if (method === 'PUT' && /\/api\/repositories\/1\/factory\/sponsorships\/.+$/.test(p)) return true;
    return false;
  };
  const s = await openE2E(`founder-${stamp.slice(9)}`.slice(0, 24).replace(/[^a-z0-9-]/gi, '').toLowerCase() || 'founder-e2e', allowWrite);
  const report: Record<string, unknown> = {dir, resume: resume ?? null, started: new Date().toISOString()};
  try {
    let intent = resume ?? '';
    if (only === 'all' || only === 'describe') {
      intent = await phaseDescribe(s.page, dir, resume);
      report.intent = intent;
      await writeFile(`${dir}/intent.json`, JSON.stringify({intent}, null, 2));
    } else if (resume) intent = resume;
    assert(intent, 'need a request id: run describe first or pass --resume <intent>');
    if (only !== 'all' && only !== 'describe') {
      await s.page.goto(`${ORIGIN}/workspace/requests/${intent}`);
      await nativeSpacesMount(s.page);
      await shot(s.page, `${dir}/resumed-${only}.png`);
    }
    if (only === 'all' || only === 'explore') await phaseExplore(s.page, dir, intent);
    if (only === 'all' || only === 'decide') await phaseDecide(s.page, dir);
    if (only === 'all' || only === 'answers') await phaseAnswerQuestions(s.page, dir);
    if (only === 'all' || only === 'raise') await phaseRaise(s.page, dir, 24);
    let revision: string | null = null;
    if (only === 'all' || only === 'plan') {
      revision = await phaseBind(s.page, dir);
      if (!revision) await phaseStartPlan(s.page, dir);
      report.revision = revision;
      await phasePlan(s.page, dir, intent, revision);
    }
    // Poll-only: follow an already-admitted run via Refresh status (GET).
    // Never POSTs a plan; use after a lost launch observation (503 with a
    // live run) so a retry cannot strand a duplicate assignment.
    if (only === 'poll') await phasePlan(s.page, dir, intent, null);
    // Approved single retry without the missing button (see PLAN-RETRY-GO.md):
    // one bypass POST through the page transport, then GET poll only.
    // Standalone phase only — never part of `all` (that path uses the
    // real Continue to planning button via `plan`).
    if (only === 'retry') {
      revision = await phaseStartPlanBypass(s.page, dir, intent);
      report.revision = revision;
      await phasePlan(s.page, dir, intent, revision);
    }
    if (only === 'all' || only === 'check') await phaseCheck(s.page, dir, intent);
    if (only === 'all' || only === 'accept') await phaseAccept(s.page, dir, intent);
    if (only === 'all' || only === 'build') await phaseBuildFollow(s.page, dir, intent);
    // Approved single build retry (see BUILD-RETRY-GO.md): one Retry run
    // click, then a follow with a ~6-minute launch deadline. Standalone
    // phase only — never part of `all`.
    if (only === 'retry-build') {
      assert(intent === INTENT_DD00, `retry-build: approved only for the journey request ${INTENT_DD00}, got ${intent}`);
      await phaseRetryBuild(s.page, dir, STOPPED_RUN);
      await phaseBuildFollow(s.page, dir, intent, 6 * 60 * 1000);
    }
    // Standalone phase only — never part of `all`. Owner-approved via
    // ALLOWANCE-RETRY-GO.md: raise repo 1 coder-time sponsorship 120 -> 480.
    if (only === 'allowance') {
      await phaseAllowance(s.page, dir);
    }
    if (only === 'all' || only === 'app') await phaseAppUse(s.page, dir);
    if (only === 'all' || only === 'improve') {
      await shot(s.page, `${dir}/${only}-observed.png`);
      const text = await visibleText(s.page, 2000);
      assert.fail(`${only}: Improve follow-up is not automated yet on the installed VM (page: ${text.slice(0, 300)})`);
    }
    report.ok = true;
  } catch (error) {
    report.ok = false;
    report.error = String(error).slice(0, 1200);
    try {
      await shot(s.page, `${dir}/failure.png`);
    } catch {}
    throw error;
  } finally {
    report.allowed = s.allowed;
    report.blocked = s.blocked;
    report.failed = s.failed;
    await writeFile(`${dir}/report.json`, JSON.stringify(report, null, 2));
    await s.close();
  }
  console.log(JSON.stringify(report, null, 2));
}

if (import.meta.main) await main();
