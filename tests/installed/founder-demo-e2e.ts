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
// Repo-first places (UI installs A–E; the global /workspace is gone).
const PLACES = {
  feed: `/${REPOSITORY_SLUG}/feed`,
  requests: `/${REPOSITORY_SLUG}/requests`,
  request: (id: string) => `/${REPOSITORY_SLUG}/requests/${id}`,
  app: `/${REPOSITORY_SLUG}/app`,
  settings: `/${REPOSITORY_SLUG}/settings/sodaos`,
  settingsFactory: `/${REPOSITORY_SLUG}/settings/sodaos/factory`,
};
/** Mount a repo-first place: the owner element carries the page id
 * (repository-feed, repository-settings). Unknown pages just need load. */
async function mountRepoPage(page: Page, pageId: string | null): Promise<Page> {
  if (pageId === null) {
    await page.waitForLoadState('domcontentloaded');
    return page;
  }
  const owner = page.locator(`[data-extension-page-id="${pageId}"]`);
  await owner.first().waitFor({timeout: 30000});
  assert((await owner.count()) >= 1, `Repo place: expected owner element ${pageId}`);
  return page;
}
const INTENT_DD00 = 'dd00cb3160885e5f707ec7d95';
const STOPPED_RUN = '00976679f6258efbc2c82b7ce1719b87';
// Owner decision (via Claude): apps run at their own address, not inside
// Soda Forge — so new requests say "a small web app". dd00cb31's recorded
// request text is history and stays untouched (resume never refills it).
const OBJECTIVE = `Replace our equipment-request spreadsheet with a small web app. Employees submit a request for a laptop, monitor or other equipment, with a reason and a needed-by date. Their manager approves or rejects it with a comment. Operations sees approved requests and marks them ordered, then delivered. Employees see the status of their own requests only.`;
const MAX_ISSUES = 1;
const PLAN_TIMEOUT_MS = 20 * 60 * 1000;

type Phase = 'describe' | 'explore' | 'decide' | 'answers' | 'plan' | 'retry' | 'poll' | 'check' | 'correct' | 'accept' | 'build' | 'retry-build' | 'allowance' | 'places' | 'policy-minutes' | 'new-request' | 'bounds' | 'ready-explore' | 'app' | 'improve';
const PHASES: Phase[] = ['describe', 'explore', 'decide', 'answers', 'plan', 'retry', 'poll', 'check', 'correct', 'accept', 'build', 'retry-build', 'allowance', 'places', 'policy-minutes', 'new-request', 'bounds', 'ready-explore', 'app', 'improve'];

function args() {
  const out: {resume?: string | undefined; out?: string | undefined; phase?: string | undefined} = {};
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
  // Rehearsal run-status timeline (step 4): bodies of run-status GETs.
  const runStatuses: {t: string; url: string; body: string}[] = [];
  const guard = {armed: false};
  b.context.on('response', (response) => {
    const url = new URL(response.url());
    if (url.origin !== ORIGIN) return;
    // Step-4 timeline: keep bodies of run-status reads (bounded).
    if (
      response.ok() &&
      response.request().method() === 'GET' &&
      /\/api\/factory\/runs\/[0-9a-f]{32}\/?$/.test(url.pathname)
    ) {
      void response
        .text()
        .then((body) => {
          if (runStatuses.length < 500) runStatuses.push({t: new Date().toISOString(), url: url.pathname, body: body.slice(0, 4000)});
        })
        .catch(() => {});
      return;
    }
    if (response.ok()) return;
    const marker = '/-/native/pages/soda/';
    const at = url.pathname.indexOf(marker);
    if (at < 0) return;
    const rest = url.pathname.slice(at + marker.length);
    const apiAt = rest.indexOf('/api/');
    if (apiAt < 0) return;
    const key = `${response.status()} /api/${rest.slice(apiAt + 5).split('?')[0]}${url.search.slice(0, 120)}`;
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
  return {page, blocked, allowed, failed, runStatuses, close: () => b.close().catch(() => {})};
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
    // Repo-first request route (the global /workspace is gone).
    await page.goto(`${ORIGIN}${PLACES.request(resume)}`);
    await mountRepoPage(page, null);
    await shot(page, `${out}/describe-resumed.png`);
    return resume;
  }
  await page.goto(`${ORIGIN}${PLACES.feed}`);
  await mountRepoPage(page, 'repository-feed');
  const spaces = await mountRepoPage(page, null);
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
  const spaces = await mountRepoPage(page, null);
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
  const spaces = await mountRepoPage(page, null);
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
  // material question in the Desired outcome itself and re-explore under
  // the fixed producer.
  const questionsNow = spaces.locator('section[aria-label="Questions needing your answer"] article');
  const chooseNow = spaces.getByRole('button', {name: 'Choose repository'});
  if ((await questionsNow.count()) === 0 && (await chooseNow.count()) > 0 && !(await chooseNow.first().isEnabled())) {
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
  const spaces = await mountRepoPage(page, null);
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
  const spaces = await mountRepoPage(page, null);
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

async function phaseStartPlanBypass(page: Page, out: string, _intent: string): Promise<string> {
  // TEMPORARY BYPASS, see e2e/PLAN-RETRY-GO.md (revised): attention has no
  // Continue to planning button (page bug, fixed separately), and a second
  // plan POST on the settled revision is refused as a closed attempt. The
  // page retries by saving a successor draft and planning the new revision,
  // so this calls the page's own `soda-intent` methods — no hand-built
  // bodies (if they are unreachable, this fails instead of improvising).
  // Replace with a real click once the page fix lands. Single use only.
  const spaces = await mountRepoPage(page, null);
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
  const spaces = await mountRepoPage(page, null);
  const cont = spaces.getByRole('button', {name: 'Continue to planning', exact: true});
  await cont.waitFor({timeout: 60000});
  assert(await cont.isEnabled(), 'Plan: Continue to planning must be enabled on the bound draft');
  const planning = page.waitForResponse(
    (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/plan') && r.request().method() === 'POST',
    {timeout: 180000}
  );
  await cont.click();
  const planReply = await planning;
  // The click may navigate the page before the body is readable; the
  // status line survives, the body is best-effort.
  let planStatus = 0;
  try {
    planStatus = planReply.status();
  } catch {
    planStatus = 0;
  }
  let planBody: {error?: {code?: string; message?: string}; revision_id?: string};
  try {
    planBody = (await planReply.json()) as typeof planBody;
  } catch {
    planBody = {};
  }
  await writeFile(`${out}/plan-admission.json`, JSON.stringify({status: planStatus, body: planBody}, null, 2));
  if (![201, 202].includes(planStatus)) {
    // fux26: the click may show a 502 (or an unreadable reply after
    // navigation) while the run still starts. Watch the page's own plan
    // status poll for up to 3 min before calling it a failure.
    const watchUntil = Date.now() + 3 * 60 * 1000;
    for (;;) {
      assert(Date.now() < watchUntil, `Plan: admission got ${planStatus} and no plan appeared within 3 min`);
      const refreshed = page.waitForResponse(
        (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/plan') && r.request().method() === 'GET',
        {timeout: 60000}
      );
      await spaces.getByRole('button', {name: 'Refresh status', exact: true}).first().click();
      const response = await refreshed;
      if (response.status() === 200) {
        let plan: {state?: string; revision?: string} = {};
        try {
          plan = (await response.json()) as typeof plan;
        } catch {
          plan = {};
        }
        // state 'none' carries the revision id but means no plan (yet) —
        // only a live or settled plan counts as admitted.
        if (plan.revision && plan.state !== 'none') {
          await writeFile(`${out}/plan-admitted-late.json`, JSON.stringify({admissionStatus: planReply.status(), plan}, null, 2));
          await shot(page, `${out}/plan-admitted-late.png`);
          return;
        }
      }
      await page.waitForTimeout(15000);
    }
  }
  await shot(page, `${out}/plan-admitted.png`);
}

async function phasePlan(page: Page, out: string, intent: string, revision: string | null): Promise<void> {
  const spaces = await mountRepoPage(page, null);
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
    // state 'none' means the run has not registered yet — keep polling
    // inside the deadline; only a settled non-pending state is terminal.
    if (plan.state === 'none') {
      await page.waitForTimeout(15000);
      continue;
    }
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
  await page.goto(`${ORIGIN}${PLACES.app}`);
  await mountRepoPage(page, null);
  await page.getByRole('heading', {name: /^App$/}).first().waitFor({timeout: 30000});
  const found = await page.evaluate(() => {
    const links: string[] = [];
    let iframes = 0;
    const scan = (root: Document | ShadowRoot) => {
      for (const a of Array.from(root.querySelectorAll('a[href]'))) {
        // Lane A delivery anchor carries data-feed-open-app ("Open app ↗").
        if ((a as HTMLAnchorElement).hasAttribute('data-feed-open-app') || /open app/i.test(a.textContent || '')) {
          links.push((a as HTMLAnchorElement).href);
        }
      }
      iframes += root.querySelectorAll('iframe').length;
      for (const el of Array.from(root.querySelectorAll('*'))) if ((el as Element).shadowRoot) scan((el as Element).shadowRoot as ShadowRoot);
    };
    scan(document);
    return {links, iframes, url: location.href};
  });
  await writeFile(`${out}/app-discovery.json`, JSON.stringify(found, null, 2));
  assert(found.links.length > 0, 'App: no "Open app" link on the Forge side; cannot reach the app at its own address');
  const href = found.links[0] ?? '';
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
  appLinks: string[];
}

/** Owner-approved allowance raise (ALLOWANCE-RETRY-GO.md): repo 1 coder-time
 * sponsorship 120 -> 480 minutes through Settings → SodaOS → Factory's own
 * "Save sponsorship" control. Fails naming the exact state when the page
 * offers no such control; touches nothing else on the form. */
async function phaseAllowance(page: Page, out: string): Promise<void> {
  await page.goto(`${ORIGIN}${PLACES.settingsFactory}`);
  const spaces = await mountRepoPage(page, 'repository-settings');
  await spaces.getByRole('heading', {name: /^SodaOS$/}).first().waitFor({timeout: 30000});
  // The factory section may need selecting when the page opens elsewhere.
  const sponsorshipFormPresent = async () =>
    (await spaces.locator('form[data-action="sponsorship"]').count()) > 0;
  if (!(await sponsorshipFormPresent())) {
    const factoryTab = spaces.getByRole('tab', {name: 'Factory'});
    assert((await factoryTab.count()) >= 1, 'Allowance: no Factory tab in project settings');
    await factoryTab.first().click();
  }
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
  // GO-2: the grant select starts on its placeholder even when grants exist.
  // Pick the approved grant inside the SPONSORSHIP form only (grant_id also
  // appears in the unrelated Project grants form).
  const sponsorshipForm = spaces.locator('form[data-action="sponsorship"]');
  const grantSelect = sponsorshipForm.locator('select[name="grant_id"]');
  assert((await grantSelect.count()) >= 1, 'Allowance: sponsorship grant select not found');
  const grantValues: string[] = await grantSelect.first().evaluate((select: HTMLSelectElement) =>
    Array.from(select.options).map((o) => o.value).filter((v) => v !== '')
  );
  await writeFile(`${out}/allowance-grants.json`, JSON.stringify(grantValues, null, 2));
  const matching = grantValues.filter((v) => v.startsWith('cf013ecf'));
  assert(matching.length >= 1, `Allowance: approved grant cf013ecf… not offered (grants: ${JSON.stringify(grantValues)})`);
  assert(matching.length === 1, `Allowance: several grants match; owner must choose: ${JSON.stringify(matching)}`);
  const grant = matching[0];
  assert(grant !== undefined);
  await grantSelect.first().selectOption(grant);
  await page.waitForTimeout(2000);
  const minutes = spaces.getByLabel(/Allowance minutes/i);
  assert((await minutes.count()) >= 1, 'Allowance: allowance minutes input not found');
  const before = await minutes.first().inputValue();
  await minutes.first().fill('480');
  // GO-2 end state: Builder + Inspector checked, concurrent 1, active on.
  // Touch each only when it differs, and record what changed.
  const changes: Record<string, {before: string; after: string}> = {};
  const setChecked = async (name: string, want: boolean) => {
    const box = sponsorshipForm.locator(`input[name="${name}"]`).first();
    const was = await box.isChecked();
    if (was !== want) {
      await box.setChecked(want);
      changes[name] = {before: String(was), after: String(want)};
    }
  };
  await setChecked('coder', true);
  await setChecked('reviewer', true);
  await setChecked('sponsorship_active', true);
  const concurrent = sponsorshipForm.locator('input[name="max_concurrent"]').first();
  const concurrentBefore = await concurrent.inputValue();
  if (concurrentBefore !== '1') {
    await concurrent.fill('1');
    changes['max_concurrent'] = {before: concurrentBefore, after: '1'};
  }
  await writeFile(`${out}/allowance-form.json`, JSON.stringify({before, grant, changes}, null, 2));
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

/** Rehearsal step 1 (REHEARSAL-B-GO.md): set the policy attempt limit
 * "Active minutes per attempt" 30 -> 60 through the repository policy
 * form's own save. Fails naming the page state when the control is
 * absent; touches nothing else (other fields round-trip untouched). */
async function phasePolicyMinutes(page: Page, out: string): Promise<void> {
  await page.goto(`${ORIGIN}${PLACES.settingsFactory}`);
  const spaces = await mountRepoPage(page, 'repository-settings');
  await spaces.getByRole('heading', {name: /^SodaOS$/}).first().waitFor({timeout: 30000});
  const policyForm = spaces.locator('form[data-action="policy"]');
  try {
    await policyForm.first().waitFor({timeout: 30000});
  } catch {
    const text = await visibleText(page, 2000);
    assert.fail(`Policy: repository policy form not found (page: ${text.slice(0, 400)})`);
  }
  const minutes = policyForm.locator('input[name="active_minutes"]').first();
  assert((await minutes.count()) >= 1, 'Policy: Active minutes per attempt control not offered; stopping per go-ahead');
  const before = await minutes.inputValue();
  await minutes.fill('60');
  const acted = page.waitForResponse(
    (r) => /\/api\/repositories\/1\/factory\/policy$/.test(new URL(r.url()).pathname) && r.request().method() === 'PUT',
    {timeout: 120000}
  );
  const save = policyForm.getByRole('button', {name: 'Save repository policy'}).first();
  assert(await save.isEnabled(), 'Policy: Save repository policy is disabled');
  await save.click();
  const reply = await acted;
  const body = await reply.json().catch(() => ({}));
  await writeFile(`${out}/policy-reply.json`, JSON.stringify({before, after: '60', status: reply.status(), body}, null, 2));
  assert(reply.ok(), `Policy: save must succeed, got HTTP ${reply.status()}: ${JSON.stringify(body).slice(0, 300)}`);
}

const REHEARSAL_OBJECTIVE = 'Replace our equipment-request spreadsheet with a small app. Employees request a laptop, monitor or other equipment with a reason and a needed-by date. Their manager approves or rejects it with a comment. Operations marks approved requests ordered, then delivered. Employees see only their own requests.';

/** Rehearsal step 2: the presenter path — Requests → New request, typed
 * through the page (never d401…, never dd00…). Records Max work items as
 * found/set. */
async function phaseNewRequest(page: Page, out: string): Promise<string> {
  await page.goto(`${ORIGIN}/${REPOSITORY_SLUG}/requests/new`);
  await mountRepoPage(page, null);
  const box = page.getByLabel('Describe what you want to build', {exact: true});
  await box.first().waitFor({timeout: 30000});
  assert((await box.first().inputValue()) === '', 'New request: description must start empty');
  await box.first().fill(REHEARSAL_OBJECTIVE);
  // Shadow-DOM control inventory for the record.
  const controls = await page.evaluate(() => {
    const out: {label: string; name: string; value: string}[] = [];
    const roots: (Document | ShadowRoot)[] = [document];
    for (let i = 0; i < roots.length; i++) {
      for (const el of Array.from((roots[i] as Document | ShadowRoot).querySelectorAll('*'))) {
        if ((el as Element).shadowRoot) roots.push((el as Element).shadowRoot as ShadowRoot);
      }
    }
    for (const root of roots) {
      for (const el of Array.from(root.querySelectorAll('input[type="number"], input:not([type]), textarea'))) {
        const input = el as HTMLInputElement;
        if (input.tagName === 'INPUT' && input.type !== 'number' && input.type !== '' && input.type !== 'text') continue;
        const id = input.id;
        const label = id ? root.querySelector(`label[for="${id}"]`)?.textContent?.trim().slice(0, 60) ?? '' : '';
        const wrap = input.closest('label')?.textContent?.trim().slice(0, 60) ?? '';
        const aria = input.getAttribute('aria-label')?.slice(0, 60) ?? '';
        out.push({label: label || wrap || aria, name: input.name, value: input.value.slice(0, 200)});
      }
    }
    return out;
  });
  await writeFile(`${out}/new-request-controls.json`, JSON.stringify(controls, null, 2));
  // Bounds live in a collapsed section: pierce shadow roots, find the
  // input by its label, and open the enclosing disclosure first.
  await page.evaluate(() => {
    const roots: (Document | ShadowRoot)[] = [document];
    for (let i = 0; i < roots.length; i++) {
      for (const el of Array.from((roots[i] as Document | ShadowRoot).querySelectorAll('*'))) {
        if ((el as Element).shadowRoot) roots.push((el as Element).shadowRoot as ShadowRoot);
      }
    }
    for (const root of roots) {
      for (const el of Array.from(root.querySelectorAll('input'))) {
        const input = el as HTMLInputElement;
        const id = input.id;
        const label = id ? root.querySelector(`label[for="${id}"]`)?.textContent ?? '' : '';
        const wrap = input.closest('label')?.textContent ?? '';
        const aria = input.getAttribute('aria-label') ?? '';
        if (!/Maximum work items/i.test(label + ' ' + wrap + ' ' + aria)) continue;
        const details = input.closest('details');
        const summary = details?.querySelector(':scope > summary');
        if (details && !(details as HTMLDetailsElement).open) (summary as HTMLElement)?.click();
      }
    }
  });
  const setNumber = async (match: RegExp, value: string) => {
    const field = page.getByLabel(match);
    assert((await field.count()) === 1, `New request: expected exactly one ${match}, found ${await field.count()}`);
    await field.first().fill(value);
    const readBack = await field.first().inputValue();
    assert(readBack === value, `New request: ${match} did not take (want ${value}, shows ${readBack})`);
    return `${match}: set ${value} (verified)`;
  };
  const fields: Record<string, string> = {};
  fields['max'] = await setNumber(/Maximum work items/i, '1');
  await shot(page, `${out}/new-request-typed.png`);
  const saving = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 60000}
  );
  await page.getByRole('button', {name: 'Save draft', exact: true}).first().click();
  const saved = await saving;
  assert.equal(saved.status(), 201, `New request: Save draft must return 201, got ${saved.status()}`);
  const body = (await saved.json()) as {intent?: string; id?: string};
  const newIntent = String(body.intent ?? '');
  assert.match(newIntent, /^d[0-9a-f]{24}$/, 'New request: saved draft must report its intent');
  assert(newIntent !== INTENT_DD00 && !newIntent.startsWith('d4014235'), 'New request: must not reuse a protected request');
  await writeFile(`${out}/new-intent.json`, JSON.stringify({intent: newIntent, fields}, null, 2));
  return newIntent;
}

/** Rehearsal bounds fix: the synthetic fills at creation did not take, so
 * the draft saved defaults (Max 20, Planning 1). Set both through the
 * page with read-back verification, then Save draft (POST intents, already
 * allow-listed) so the next Explore can authorize its allowance. */
async function phaseBounds(page: Page, out: string): Promise<void> {
  // Bounds live in a collapsed section: pierce shadow roots, find the
  // input by its label, and open the enclosing disclosure first.
  await page.evaluate(() => {
    const roots: (Document | ShadowRoot)[] = [document];
    for (let i = 0; i < roots.length; i++) {
      for (const el of Array.from((roots[i] as Document | ShadowRoot).querySelectorAll('*'))) {
        if ((el as Element).shadowRoot) roots.push((el as Element).shadowRoot as ShadowRoot);
      }
    }
    for (const root of roots) {
      for (const el of Array.from(root.querySelectorAll('input'))) {
        const input = el as HTMLInputElement;
        const id = input.id;
        const label = id ? root.querySelector(`label[for="${id}"]`)?.textContent ?? '' : '';
        const wrap = input.closest('label')?.textContent ?? '';
        const aria = input.getAttribute('aria-label') ?? '';
        if (!/Maximum work items/i.test(label + ' ' + wrap + ' ' + aria)) continue;
        const details = input.closest('details');
        const summary = details?.querySelector(':scope > summary');
        if (details && !(details as HTMLDetailsElement).open) (summary as HTMLElement)?.click();
      }
    }
  });
  const setNumber = async (match: RegExp, value: string) => {
    const field = page.getByLabel(match);
    assert((await field.count()) === 1, `Bounds: expected exactly one ${match}, found ${await field.count()}`);
    await field.first().fill(value);
    const readBack = await field.first().inputValue();
    assert(readBack === value, `Bounds: ${match} did not take (want ${value}, shows ${readBack})`);
    return `${match}: ${value} (verified)`;
  };
  const applied: Record<string, string> = {};
  applied['max'] = await setNumber(/Maximum work items/i, '1');
  await shot(page, `${out}/bounds-set.png`);
  const saving = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 60000}
  );
  await page.getByRole('button', {name: 'Save draft', exact: true}).first().click();
  const saved = await saving;
  assert.equal(saved.status(), 201, `Bounds: Save draft must return 201, got ${saved.status()}`);
  await writeFile(`${out}/bounds.json`, JSON.stringify(applied, null, 2));
}

/** Rehearsal: fix bounds AND explore in one browser session, the way a
 * presenter would. Separate processes lose component state (each launch
 * rehydrates from the server head), so the explore run kept re-saving
 * stale limits. Fills verified, saves, then Explore ideas through the
 * page: 201 means the allowance posted and exploration ran. */
async function phaseReadyExplore(page: Page, out: string, intent: string): Promise<void> {
  const expandBounds = () =>
    page.evaluate(() => {
      const roots: (Document | ShadowRoot)[] = [document];
      for (let i = 0; i < roots.length; i++) {
        for (const el of Array.from((roots[i] as Document | ShadowRoot).querySelectorAll('*'))) {
          if ((el as Element).shadowRoot) roots.push((el as Element).shadowRoot as ShadowRoot);
        }
      }
      for (const root of roots) {
        for (const el of Array.from(root.querySelectorAll('input'))) {
          const input = el as HTMLInputElement;
          const id = input.id;
          const label = id ? root.querySelector(`label[for="${id}"]`)?.textContent ?? '' : '';
          const wrap = input.closest('label')?.textContent ?? '';
          const aria = input.getAttribute('aria-label') ?? '';
          if (!/Maximum work items/i.test(label + ' ' + wrap + ' ' + aria)) continue;
          const details = input.closest('details');
          const summary = details?.querySelector(':scope > summary');
          if (details && !(details as HTMLDetailsElement).open) (summary as HTMLElement)?.click();
        }
      }
    });
  await expandBounds();
  for (const [match, value] of [[/Maximum work items/i, '1']] as [RegExp, string][]) {
    const field = page.getByLabel(match);
    assert((await field.count()) === 1, `ReadyExplore: expected exactly one ${match}`);
    await field.first().fill(value);
    assert((await field.first().inputValue()) === value, `ReadyExplore: ${match} did not take`);
  }
  await shot(page, `${out}/ready-bounds.png`);
  const saving = page.waitForResponse(
    (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
    {timeout: 60000}
  );
  await page.getByRole('button', {name: 'Save draft', exact: true}).first().click();
  const saved = await saving;
  assert.equal(saved.status(), 201, `ReadyExplore: Save draft must return 201, got ${saved.status()}`);
  // Explore in the same session so component state (limits 1/12) is fresh:
  // the page authorizes the allowance itself, then explores.
  const explored = page.waitForResponse(
    (r) => r.url().endsWith(`/api/factory/intents/${intent}/explore`) && r.request().method() === 'POST',
    {timeout: 300000}
  );
  await page.getByRole('button', {name: 'Explore ideas'}).first().click();
  const done = await explored;
  const body = await done.json().catch(() => ({}));
  await writeFile(`${out}/explore.json`, JSON.stringify({status: done.status(), revision: (body as {id?: string}).id}, null, 2));
  await shot(page, `${out}/explore-done.png`);
  assert.equal(done.status(), 201, `ReadyExplore: explore must return 201, got ${done.status()}: ${JSON.stringify(body).slice(0, 300)}`);
}

/** Read-only migration check (UI installs A–E): visits each repo-first
 * place, asserts its heading, and records the Feed v2 post structure the
 * follow-up scopes to. No clicks that write or spend. */
async function phasePlaces(page: Page, out: string, intent: string): Promise<void> {
  const structure: Record<string, unknown> = {};
  const visit = async (name: string, url: string, pageId: string | null, heading: RegExp | null) => {
    await page.goto(`${ORIGIN}${url}`);
    await mountRepoPage(page, pageId);
    if (heading !== null) {
      await page.getByRole('heading', {name: heading}).first().waitFor({timeout: 30000});
    } else {
      await page.waitForTimeout(4000);
    }
    await shot(page, `${out}/places-${name}.png`);
    structure[name] = {url, text: (await visibleText(page, 2000)).slice(0, 600)};
    await writeFile(`${out}/places.json`, JSON.stringify(structure, null, 2));
  };
  await visit('feed', PLACES.feed, 'repository-feed', /^Feed$/);
  // Header tabs move between the repo places.
  const tabs = await page.evaluate(() =>
    Array.from(document.querySelectorAll('a[data-soda-place]')).map((a) => (a as HTMLAnchorElement).getAttribute('data-soda-place'))
  );
  structure['tabs'] = tabs;
  assert(tabs.includes('feed'), `Places: header tabs must include feed (tabs: ${JSON.stringify(tabs)})`);
  // Feed v2 post structure the follow-up scopes to.
  const posts = await page.evaluate(() => {
    const scoped = Array.from(document.querySelectorAll('article.soda-feed-post'));
    const all = Array.from(document.querySelectorAll('article')).map((a) => a.className);
    const first = document.querySelector('article.soda-feed-post');
    const openApp = Array.from(document.querySelectorAll('a[data-feed-open-app]')).map((a) => ({
      href: (a as HTMLAnchorElement).href,
      text: (a.textContent || '').trim().slice(0, 80),
    }));
    const buttons = Array.from(document.querySelectorAll('button[data-feed-open-app]')).map((b) => (b.textContent || '').trim().slice(0, 80));
    return {
      scoped: scoped.length,
      articles: all.slice(0, 8),
      first: first ? (first.textContent || '').replace(/\s+/g, ' ').slice(0, 400) : null,
      openApp,
      buttons,
    };
  });
  structure['feed-posts'] = posts;
  await writeFile(`${out}/places-feed.json`, JSON.stringify(posts, null, 2));
  assert(posts.scoped > 0, `Places: no article.soda-feed-post on the repo Feed (articles: ${JSON.stringify(posts.articles)})`);
  await visit('requests', PLACES.requests, null, /^Requests$/);
  // New-request controls for the rehearsal (read-only inventory only).
  await page.goto(`${ORIGIN}/${REPOSITORY_SLUG}/requests/new`);
  await mountRepoPage(page, null);
  await page.waitForTimeout(4000);
  await shot(page, `${out}/places-new.png`);
  structure['new-request'] = {
    url: `/${REPOSITORY_SLUG}/requests/new`,
    text: (await visibleText(page, 2000)).slice(0, 600),
    controls: await page.evaluate(() => {
      const labels = Array.from(document.querySelectorAll('label')).map((l) =>
        (l.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 60)
      );
      const buttons = Array.from(document.querySelectorAll('button')).map((b) => (b.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 60));
      const inputs = Array.from(document.querySelectorAll('input, textarea, select')).map((el) => ({
        tag: el.tagName.toLowerCase(),
        type: (el as HTMLInputElement).type || '',
        name: (el as HTMLInputElement).name || (el as HTMLSelectElement).name || '',
        label: (el.getAttribute('aria-label') || '').slice(0, 60),
      }));
      return {labels: [...new Set(labels)].slice(0, 20), buttons: [...new Set(buttons)].slice(0, 20), inputs: inputs.slice(0, 20)};
    }),
  };
  await writeFile(`${out}/places.json`, JSON.stringify(structure, null, 2));
  await visit('request', PLACES.request(intent), null, null);
  {
    const text = String((structure['request'] as {text?: string} | undefined)?.text ?? '');
    assert(
      /Issue #1|equipment-request|northstar-equipment/i.test(text),
      `Places: request page does not look like the journey request (page: ${text.slice(0, 300)})`
    );
    // Button inventory for the next go-ahead (presence only, never clicked):
    // Retry run, Check walkthrough, Accept and build.
    // Allowance authorization context for the rehearsal (read-only).
    structure['request-allowance'] = await page.evaluate(() => {
      const text = document.body.textContent || '';
      const out: string[] = [];
      const push = (re: RegExp) => {
        const i = text.search(re);
        if (i >= 0) out.push(text.slice(Math.max(0, i - 80), i + 220).replace(/\s+/g, ' '));
      };
      push(/planning allowance/i);
      push(/planning calls/i);
      push(/authorize/i);
      return out;
    });
    // Shadow-piercing control inventory (planning limits live in shadow DOM).
    structure['request-controls'] = await page.evaluate(() => {
      const roots: (Document | ShadowRoot)[] = [document];
      for (let i = 0; i < roots.length; i++) {
        for (const el of Array.from((roots[i] as Document | ShadowRoot).querySelectorAll('*'))) {
          if ((el as Element).shadowRoot) roots.push((el as Element).shadowRoot as ShadowRoot);
        }
      }
      const buttons: string[] = [];
      const inputs: {label: string; name: string; value: string}[] = [];
      for (const root of roots) {
        for (const b of Array.from(root.querySelectorAll('button'))) {
          const t = (b.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 80);
          if (t && !buttons.includes(t)) buttons.push(t);
        }
        for (const el of Array.from(root.querySelectorAll('input, textarea, select'))) {
          const input = el as HTMLInputElement;
          const id = input.id;
          const label = id ? root.querySelector(`label[for="${id}"]`)?.textContent?.trim().slice(0, 60) ?? '' : '';
          const wrap = input.closest('label')?.textContent?.trim().slice(0, 60) ?? '';
          const aria = input.getAttribute('aria-label')?.slice(0, 60) ?? '';
          const name = input.name || '';
          inputs.push({label: label || wrap || aria, name, value: (input.value || '').slice(0, 120)});
        }
      }
      return {buttons, inputs};
    });
    structure['request-buttons'] = await page.evaluate(() =>
      Array.from(document.querySelectorAll('button'))
        .map((b) => {
          const section = b.closest('details, section, li, article');
          const hosts: string[] = [];
          let node = b.parentElement;
          while (node && hosts.length < 4) {
            if (node.tagName.includes('-')) hosts.push(node.tagName.toLowerCase());
            node = node.parentElement;
          }
          return {
            text: (b.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 80),
            disabled: (b as HTMLButtonElement).disabled,
            section: (section?.querySelector('summary, h1, h2, h3')?.textContent || '').replace(/\s+/g, ' ').trim().slice(0, 80),
            hosts,
          };
        })
        .filter((b) => b.text !== '')
    );
  }
  await visit('app', PLACES.app, null, /^App$/);
  await visit('settings', PLACES.settingsFactory, 'repository-settings', /^SodaOS$/);
  const sponsorship = await page.evaluate(() => !!document.querySelector('form[data-action="sponsorship"]'));
  structure['settings-sponsorship'] = sponsorship;
  // Minute-limit inputs for rehearsal step 1 (read-only inventory).
  structure['settings-minutes'] = await page.evaluate(() => {
    const out: {label: string; name: string; value: string}[] = [];
    const roots: (Document | ShadowRoot)[] = [document];
    for (let i = 0; i < roots.length; i++) {
      for (const el of Array.from((roots[i] as Document | ShadowRoot).querySelectorAll('*'))) {
        if ((el as Element).shadowRoot) roots.push((el as Element).shadowRoot as ShadowRoot);
      }
    }
    for (const root of roots) {
      for (const el of Array.from(root.querySelectorAll('input[type="number"], input:not([type])'))) {
        const input = el as HTMLInputElement;
        if (input.type !== 'number' && input.type !== '') continue;
        const id = input.id;
        const label = id ? root.querySelector(`label[for="${id}"]`)?.textContent?.trim().slice(0, 60) ?? '' : '';
        const wrap = input.closest('label')?.textContent?.trim().slice(0, 60) ?? '';
        out.push({label: label || wrap, name: input.name, value: input.value});
      }
    }
    return out;
  });
  await writeFile(`${out}/places.json`, JSON.stringify(structure, null, 2));
}

const BUILD_TIMEOUT_MS = 35 * 60 * 1000;

/** Retry the cancelled build ONCE through the product's own Retry control
 * (RetryRun path for stop-before-start). Fails naming the exact state when
 * the page offers no usable Retry for the stopped run. */
async function phaseRetryBuild(page: Page, out: string, run: string): Promise<void> {
  // The run interventions (incl. "Retry run") render in the per-issue work
  // detail on the request route, not in the project Work view.
  await page.goto(`${ORIGIN}${PLACES.request(INTENT_DD00)}`);
  const spaces = await mountRepoPage(page, null);
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
    await page.goto(`${ORIGIN}${PLACES.feed}`);
    await mountRepoPage(page, 'repository-feed');
    // Feed cards render asynchronously and the activity read can fail
    // transiently (e.g. right after an install); retry the read through
    // the page's own control before failing on an empty Feed.
    let loaded = false;
    for (let attempt = 0; attempt < 4 && !loaded; attempt++) {
      try {
        await page.waitForFunction(() => document.querySelectorAll('article.soda-feed-post').length > 0, {timeout: 30000});
        loaded = true;
      } catch {
        const retry = page.getByRole('button', {name: 'Retry loading activity'});
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
        // Feed v2 delivers "Open app ↗" as an anchor carrying
        // data-feed-open-app on the result card.
        const appLinks = Array.from(el.querySelectorAll('a[data-feed-open-app]')).map(
          (a) => (a as HTMLAnchorElement).href
        );
        found.push({
          id: el.id || '',
          status: el.getAttribute('data-status') || '',
          role: el.getAttribute('data-role') || '',
          text,
          appKeys,
          appLinks,
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
    const appCards = cards.filter((card) => card.appKeys.length > 0 || card.appLinks.length > 0);
    if (appCards.length > 1) {
      assert.fail(
        `Build: ${appCards.length} "Open project app" cards in Feed; cannot tell which delivery is the journey's`
      );
    }
    if (appCards.length === 1) {
      const delivery = appCards[0]!;
      assert(
        /Application outcomes verified/i.test(delivery.text),
        `Build: the app card does not carry the verified-outcomes claim: ${delivery.text.slice(0, 300)}`
      );
      await writeFile(`${out}/build-delivery.json`, JSON.stringify(delivery, null, 2));
      await appendFile(
        `${out}/build-milestones.log`,
        `${new Date().toISOString()} app-ready key=${delivery.appKeys[0] ?? ''} link=${delivery.appLinks[0] ?? ''}: ${delivery.text.slice(0, 300)}\n`
      );
      return;
    }
    await page.waitForTimeout(30000);
  }
}

async function phaseCheck(page: Page, out: string, intent: string): Promise<void> {
  const spaces = await mountRepoPage(page, null);
  const walkthrough = spaces.getByRole('button', {name: 'Check walkthrough', exact: true});
  if ((await walkthrough.count()) === 0) {
    // A completed review replaces the Check button with its outcome. Open
    // the stored findings (GET only, no new spend) and report them.
    const findings = spaces.getByRole('button', {name: /see findings/i});
    assert((await findings.count()) > 0, 'Check: neither Check walkthrough nor See findings is present');
    await findings.first().scrollIntoViewIfNeeded();
    await findings.first().click();
    await page.waitForTimeout(1500);
    const text = await visibleText(page, 6000);
    await writeFile(`${out}/check-findings.txt`, text);
    await shot(page, `${out}/check-findings.png`);
    assert.fail(`Check: stored review needs correction before Accept. Findings: ${text.slice(0, 1500)}`);
  }
  const checking = page.waitForResponse(
    (r) => r.url().endsWith(`/api/factory/intents/${intent}/plan/review`) && r.request().method() === 'POST',
    {timeout: 180000}
  );
  await walkthrough.first().click();
  const checked = await checking;
  assert.equal(checked.status(), 200, `Check: walkthrough review must return 200, got ${checked.status()}`);
  const review = (await checked.json()) as {consistency?: {mismatches?: unknown[]}};
  const mismatches = (review.consistency as {mismatches?: unknown[]})?.mismatches ?? [];
  await writeFile(`${out}/check-review.json`, JSON.stringify(review, null, 2));
  await shot(page, `${out}/check-walkthrough.png`);
  assert.equal(mismatches.length, 0, `Check: JEV found ${mismatches.length} mismatch(s); correct before Accept`);
}

async function phaseCorrect(page: Page, out: string): Promise<void> {
  const spaces = await mountRepoPage(page, null);
  const more = spaces.locator('summary', {hasText: 'More options'});
  if ((await more.count()) > 0) {
    await more.first().scrollIntoViewIfNeeded();
    await more.first().click();
    await page.waitForTimeout(1500);
    await shot(page, `${out}/correct-more-options.png`);
  }
  const text0 = await visibleText(page, 6000);
  await writeFile(`${out}/correct-ui.txt`, text0);
  await shot(page, `${out}/correct-ui.png`);
  // Recovery path the product offers: "Change something" under Needs
  // attention. "Correct walkthrough first" is disabled in this state.
  const change = spaces.getByRole('button', {name: 'Change something', exact: true});
  assert((await change.count()) > 0, 'Correct: neither correction path is present');
  await change.first().scrollIntoViewIfNeeded();
  await change.first().click();
  await page.waitForTimeout(2000);
  const text1 = await visibleText(page, 8000);
  await writeFile(`${out}/correct-change-ui.txt`, text1);
  await shot(page, `${out}/correct-change-ui.png`);
  // Correction for JEV's effect_added finding on the new plan: the
  // walkthrough claims ordered/delivered steps record acting identities, but
  // the request never asks for that. Remove the effect: status updates only.
  const correction = [
    'Correction for the walkthrough finding: ordering and delivery update the request status only.',
    'The request does not record which Operations staff member performed each step, and no acting identities are shown.',
    'Remove that claim from the walkthrough.',
  ].join(' ');
  const field = spaces.locator('.change-field textarea');
  assert((await field.count()) >= 1, 'Correct: What-should-change field not found');
  await field.first().fill(correction);
  await writeFile(`${out}/correct-submitted.txt`, correction);
  await shot(page, `${out}/correct-filled.png`);
  const exploring = page.waitForResponse(
    (r) => r.url().includes('/api/factory/intents/') && r.url().endsWith('/explore') && r.request().method() === 'POST',
    {timeout: 180000}
  );
  await spaces.getByRole('button', {name: 'Submit change', exact: true}).first().click();
  const explored = await exploring;
  let exploreBody: {intent?: string; error?: {code?: string; message?: string}} = {};
  try {
    exploreBody = (await explored.json()) as typeof exploreBody;
  } catch {
    exploreBody = {};
  }
  await writeFile(`${out}/correct-explore.json`, JSON.stringify({status: explored.status(), body: exploreBody}, null, 2));
  assert.equal(explored.status(), 201, `Correct: Submit change must re-explore with 201, got ${explored.status()} (${exploreBody.error?.code ?? 'no code'}: ${(exploreBody.error?.message ?? '').slice(0, 300)})`);
  await page.waitForTimeout(3000);
  await shot(page, `${out}/correct-explored.png`);
  const text = await visibleText(page, 6000);
  await writeFile(`${out}/correct-after.txt`, text);
  assert.fail(`Correct: correction submitted, explore 201. Next: plan again. Page: ${text.slice(0, 1200)}`);
}

async function phaseAccept(page: Page, out: string, _intent: string): Promise<void> {
  const spaces = await mountRepoPage(page, null);
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
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/explore$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/plan$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/plan\/review$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/intents\/[0-9a-f]+\/generate$/.test(p)) return true;
    if (method === 'POST' && /\/api\/factory\/runs\/[0-9a-f]{32}\/actions$/.test(p)) return true;
    if (method === 'PUT' && /\/api\/repositories\/1\/factory\/sponsorships\/.+$/.test(p)) return true;
    if (method === 'PUT' && /\/api\/repositories\/1\/factory\/policy$/.test(p)) return true;
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
    assert(intent || only === 'new-request', 'need a request id: run describe first or pass --resume <intent>');
    if (only !== 'all' && only !== 'describe') {
      await s.page.goto(`${ORIGIN}${PLACES.request(intent)}`);
      await mountRepoPage(s.page, null);
      await shot(s.page, `${dir}/resumed-${only}.png`);
    }
    if (only === 'all' || only === 'explore') await phaseExplore(s.page, dir, intent);
    if (only === 'all' || only === 'decide') await phaseDecide(s.page, dir);
    if (only === 'all' || only === 'answers') await phaseAnswerQuestions(s.page, dir);
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
    // Standalone only: open the walkthrough correction UI after a flagged
    // check and record what the product asks (no submit yet).
    if (only === 'correct') await phaseCorrect(s.page, dir);
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
    // Read-only migration check (UI installs A–E). Standalone only.
    if (only === 'places') {
      await phasePlaces(s.page, dir, intent);
    }
    // Rehearsal steps (REHEARSAL-B-GO.md). Standalone only.
    if (only === 'policy-minutes') {
      await phasePolicyMinutes(s.page, dir);
    }
    if (only === 'new-request') {
      intent = await phaseNewRequest(s.page, dir);
      report.intent = intent;
      await writeFile(`${dir}/intent.json`, JSON.stringify({intent}, null, 2));
    }
    // Standalone only: fix Max work items on a saved draft.
    if (only === 'bounds') {
      await phaseBounds(s.page, dir);
    }
    // Standalone rehearsal step: bounds fix + Explore ideas in one session.
    if (only === 'ready-explore') {
      await phaseReadyExplore(s.page, dir, intent);
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
    if (s.runStatuses.length > 0) {
      await writeFile(`${dir}/run-status.json`, JSON.stringify(s.runStatuses, null, 2));
    }
    await writeFile(`${dir}/report.json`, JSON.stringify(report, null, 2));
    await s.close();
  }
  console.log(JSON.stringify(report, null, 2));
}

if (import.meta.main) await main();
