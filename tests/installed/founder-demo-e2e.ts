// Founder-demo end-to-end journey: Describe → Decide → Build → Use → Improve.
// Installed script (not a *.test.ts unit test). Runs phase by phase against the
// demo VM, with screenshots per phase and a clear failure naming the exact state.
// Resumes from a saved request id without repeating spends.
// Usage:
//   bun tests/installed/founder-demo-e2e.ts [--resume <intentId>] [--out <dir>] [--phase <describe|explore|decide|plan|check|accept|build|app|improve|all>]
// Run artifacts stay outside the repo under ~/Projects/.soda-demo-checks/e2e/run-*/.
import assert from 'node:assert/strict';
import {mkdir, readFile, writeFile} from 'node:fs/promises';
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
const OBJECTIVE = `Replace our equipment-request spreadsheet with a small app inside Soda Forge. Employees submit a request for a laptop, monitor or other equipment, with a reason and a needed-by date. Their manager approves or rejects it with a comment. Operations sees approved requests and marks them ordered, then delivered. Employees see the status of their own requests only.`;
const MAX_ISSUES = 1;
const PLANNING_CALLS = 12;
const PLAN_TIMEOUT_MS = 10 * 60 * 1000;

type Phase = 'describe' | 'explore' | 'decide' | 'answers' | 'plan' | 'check' | 'accept' | 'build' | 'app' | 'improve';
const PHASES: Phase[] = ['describe', 'explore', 'decide', 'answers', 'plan', 'check', 'accept', 'build', 'app', 'improve'];

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
  const guard = {armed: false};
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
  return {page, blocked, allowed, close: () => b.close().catch(() => {})};
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

/** Every Factory run on the journey must be Muse muse-spark-1.3 with
 * contributor mode on (per the installed Factory policy). Fail naming the
 * exact provenance text found when the markers are absent. */
export function assertMuseProvenance(details: string, where: string): void {
  assert(
    /muse/i.test(details) && /muse-spark-1\.3/.test(details) && /contributor/i.test(details),
    `${where}: expected Muse muse-spark-1.3 contributor provenance, found: ${details.slice(0, 500)}`
  );
}

async function phaseBuildFollow(page: Page, out: string, intent: string): Promise<void> {
  const started = Date.now();
  await page.goto(`${ORIGIN}/workspace?repository=${REPOSITORY}`);
  await nativeSpacesMount(page);
  const spaces = await spacesContent(page);
  await spaces.getByRole('button', {name: 'Feed', exact: true}).first().click();
  await page.waitForTimeout(2000);
  const text = await visibleText(page, 8000);
  assert(
    text.includes(intent.slice(0, 8)) || /Run ended|Started|Run problem|needs your decision/i.test(text),
    `Build: no Factory run post visible yet for the journey in Feed (page: ${text.slice(0, 300)})`
  );
  await shot(page, `${out}/build-feed.png`);
  // Provenance comes from the run post's Technical details on the VM; assert
  // it here once a run post exists. Until then the text scan above fails first.
  const details = await page.evaluate(() => {
    const parts: string[] = [];
    const scan = (root: Document | ShadowRoot) => {
      for (const d of Array.from(root.querySelectorAll('details'))) {
        const t = (d.textContent || '').replace(/\s+/g, ' ').slice(0, 2000);
        if (/muse|harness|model|contributor|assignment|run/i.test(t)) parts.push(t);
      }
      for (const el of Array.from(root.querySelectorAll('*'))) if ((el as Element).shadowRoot) scan((el as Element).shadowRoot as ShadowRoot);
    };
    scan(document);
    return parts.join('\n---\n');
  });
  await writeFile(`${out}/build-provenance.txt`, `started=${new Date(started).toISOString()}\n${details.slice(0, 4000)}\n`);
  assertMuseProvenance(details, 'Build');
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
    let revision: string | null = null;
    if (only === 'all' || only === 'plan') {
      revision = await phaseBind(s.page, dir);
      if (!revision) await phaseStartPlan(s.page, dir);
      report.revision = revision;
      await phasePlan(s.page, dir, intent, revision);
    }
    if (only === 'all' || only === 'check') await phaseCheck(s.page, dir, intent);
    if (only === 'all' || only === 'accept') await phaseAccept(s.page, dir, intent);
    if (only === 'all' || only === 'build') await phaseBuildFollow(s.page, dir, intent);
    if (only === 'all' || only === 'app' || only === 'improve') {
      await shot(s.page, `${dir}/${only}-observed.png`);
      const text = await visibleText(s.page, 2000);
      assert.fail(`${only}: App/Improve follow-up is not automated yet on the installed VM (page: ${text.slice(0, 300)})`);
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
    await writeFile(`${dir}/report.json`, JSON.stringify(report, null, 2));
    await s.close();
  }
  console.log(JSON.stringify(report, null, 2));
}

if (import.meta.main) await main();
