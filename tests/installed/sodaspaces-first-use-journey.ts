import assert from 'node:assert/strict';
import type {Page, Response} from 'playwright';
import {object} from './sodaspaces-input';
import {projectId} from '../../frontend/spaces/sodaspaces-api';
import {mountedSpacesFacts, nativeSpacesMount, spacesContent} from './sodaspaces-journey-evidence';
import type {FirstUseEvidence} from './sodaspaces-journey-evidence';

/** Actual app-first journey through the native UI. Preparation and finite
execution authority remain with their existing owner callbacks. The returned
result is generated work, never a claim that the app was built or released. */
export interface FirstUseDecision {
  match: string;
  decision: 'accept' | 'decline' | 'edit';
  correction?: string;
}
export interface InstalledIntentOwners {
  /** Prepare the exact selected repository through native Project settings,
   * including the approved protected check, ready role preparations, account
   * grant and caller-authorized finite planning execution. Never seed app code. */
  prepareProject: (request: {repository: string; intent: string; revision: string}) => Promise<{project: string}>;
  /** Wait for the recorded plan to retire and hold native dispatch/policy
   * before generation; the existing Factory collector owns the later run. */
  holdExecution: (request: {
    repository: string;
    intent: string;
    assignment: string;
    run: string;
  }) => Promise<{project: string; paused: boolean; dispatchOpen: boolean}>;
  /** Record owner choices against the actual generated questions. Installed
   * callers may pause here; unknown questions never receive a default answer. */
  reviewDecisions?: (pending: {kind: 'idea' | 'challenge'; text: string}[]) => Promise<FirstUseDecision[]>;
  capture?: (name: string) => Promise<void>;
}
export async function exerciseFirstUse(
  page: Page,
  input: {
    actor: string;
    login: string;
    repository: string;
    repositoryName: string;
    objective: string;
    decisions: FirstUseDecision[];
    maxIssues: number;
    planningCalls: number;
    planningTimeoutMS: number;
    /** Continue the exact saved head after an interrupted caller. An explored
     * head must restore its recorded review; it never repeats exploration. */
    savedDraft?: {intent: string; revision: string; explored?: boolean};
  },
  recordWrite: (path: string, body: Record<string, unknown>) => void,
  native: InstalledIntentOwners,
  evidence: FirstUseEvidence
) {
  await nativeSpacesMount(page);
  const {subURL} = await mountedSpacesFacts(page);
  const spaces = await spacesContent(page);
  assert(input.maxIssues === 1, 'The initial finite founder journey admits one cohesive work item');
  assert(Number.isSafeInteger(input.planningCalls) && input.planningCalls >= 1 && input.planningCalls <= 100);
  assert(
    Number.isSafeInteger(input.planningTimeoutMS) &&
      input.planningTimeoutMS >= 60000 &&
      input.planningTimeoutMS <= 1800000
  );
  evidence.factory = [];
  const recordFactory = (response: Response) => {
    const request = response.request(),
      pathname = new URL(request.url()).pathname;
    if (pathname.includes('/api/factory/') && (request.method() !== 'GET' || pathname.endsWith('/allowance')))
      evidence.factory?.push(`${request.method()} ...${pathname.slice(-52)} -> ${response.status()}`);
  };
  page.on('response', recordFactory);
  try {
    evidence.stage = 'intent entry';
    assert(input.objective.trim(), 'First use needs a declared finite objective');
    let intent = '',
      revision = '',
      entry: URL | undefined;
    if (input.savedDraft) {
      intent = input.savedDraft.intent;
      revision = input.savedDraft.revision;
      assert.match(intent, /^d[0-9a-f]{24}$/);
      assert.match(revision, /^d[0-9a-f]{24}$/);
      const restored = page.waitForResponse(
        (response) =>
          response.url().endsWith('/api/factory/intents/' + intent) && response.request().method() === 'GET',
        {timeout: 60000}
      );
      const reviewRead = input.savedDraft.explored
        ? page.waitForResponse(
            (response) =>
              response.url().endsWith('/api/factory/intents/' + intent + '/review') &&
              response.request().method() === 'GET',
            {timeout: 60000}
          )
        : undefined;
      await page.goto(new URL(subURL + '/workspace/requests/' + intent, page.url()).href);
      await nativeSpacesMount(page);
      const reply = await restored;
      assert.equal(reply.status(), 200);
      const head = object(await reply.json());
      assert.equal(head.intent, intent);
      assert.equal(head.id, revision);
      assert.equal(head.state, 'draft');
      assert.equal(head.repository, '0');
      assert.equal(head.objective, input.objective.trim());
      assert.equal(head.max_issues, input.maxIssues);
      assert.equal(head.planning_calls, input.planningCalls);
      if (reviewRead) {
        assert(Array.isArray(head.ideas) && head.ideas.length > 0);
        assert(Array.isArray(head.tasks) && head.tasks.length > 0);
        const reviewed = await reviewRead;
        assert.equal(reviewed.status(), 200, 'An existing exploration must restore through its native review owner');
        const review = object(await reviewed.json());
        assert.equal(review.intent, intent);
        assert.equal(review.head_revision, revision);
        assert.equal(object(review.review).result_revision, revision);
      } else {
        assert(Array.isArray(head.ideas) && head.ideas.length === 0);
        assert(Array.isArray(head.tasks) && head.tasks.length === 0);
        await spaces.getByRole('button', {name: 'Explore ideas', exact: true}).waitFor();
      }
    } else {
      await spaces.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
      await native.capture?.('intent');
      assert(await spaces.getByRole('button', {name: 'Explore ideas'}).isDisabled());
      await spaces.getByPlaceholder('Describe the outcome you want…').fill(input.objective);
      assert(!(await spaces.getByRole('button', {name: 'Explore ideas'}).isDisabled()));
      await spaces.locator('details.bounds summary').click();
      await spaces.getByLabel('Maximum work items', {exact: true}).fill(String(input.maxIssues));
      await spaces.getByLabel('Planning calls', {exact: true}).fill(String(input.planningCalls));
      // No separate save step: the first Explore ideas records the request.
      entry = new URL(page.url());
    }
    await native.capture?.('draft');
    evidence.stage = 'funded design exploration';
    // Until the first Explore ideas records it, the request has no id yet.
    const intentURL = (url: string, suffix: string) =>
      intent
        ? url.endsWith(`/api/factory/intents/${intent}${suffix}`)
        : new RegExp(`/api/factory/intents/d[0-9a-f]{24}${suffix}$`).test(url);
    const exploreRound = async (button: string) => {
      const fresh = intent === '';
      const roundSaving = page.waitForRequest(
        (request) => request.url().endsWith('/api/factory/intents') && request.method() === 'POST',
        {timeout: 300000}
      );
      const roundSaved = page.waitForResponse(
        (r) => r.url().endsWith('/api/factory/intents') && r.request().method() === 'POST',
        {timeout: 300000}
      );
      const funding = page.waitForResponse((r) => intentURL(r.url(), '/allowance') && r.request().method() === 'GET', {
        timeout: 300000,
      });
      const consulting = page.waitForRequest(
        (request) => intentURL(request.url(), '/explore') && request.method() === 'POST',
        {timeout: 300000}
      );
      const consulted = page.waitForResponse((r) => intentURL(r.url(), '/explore') && r.request().method() === 'POST', {
        timeout: 300000,
      });
      const creating = page
        .waitForRequest(
          (request) => request.url().endsWith('/api/factory/intent-allowances') && request.method() === 'POST',
          {timeout: 300000}
        )
        .catch(() => undefined);
      const created = page
        .waitForResponse((r) => r.url().endsWith('/api/factory/intent-allowances') && r.request().method() === 'POST', {
          timeout: 300000,
        })
        .catch(() => undefined);
      await spaces.getByRole('button', {name: button}).click();
      const roundSaveRequest = await roundSaving,
        roundSaveBody = object(roundSaveRequest.postDataJSON());
      recordWrite(new URL(roundSaveRequest.url()).pathname, roundSaveBody);
      assert.equal(roundSaveBody.state, 'draft');
      if (fresh) {
        assert.equal(roundSaveBody.repository, '0');
        assert.equal(roundSaveBody.objective, input.objective.trim());
        assert.equal(roundSaveBody.planning_calls, input.planningCalls);
        assert.equal(roundSaveBody.max_issues, input.maxIssues);
        assert(Array.isArray(roundSaveBody.ideas) && roundSaveBody.ideas.length === 0);
        assert(Array.isArray(roundSaveBody.tasks) && roundSaveBody.tasks.length === 0);
      } else {
        assert.equal(roundSaveBody.intent, intent);
        assert.equal(roundSaveBody.predecessor, revision);
      }
      const roundSaveResponse = await roundSaved;
      assert.equal(roundSaveResponse.status(), 201);
      const roundSavedBody = object(await roundSaveResponse.json());
      if (fresh) {
        intent = String(roundSavedBody.intent ?? '');
        assert(/^d[0-9a-f]{24}$/.test(intent), 'The recorded request must report its intent');
        evidence.intent = intent;
      }
      revision = String(roundSavedBody.id ?? '');
      assert(revision, 'Explore must save its draft revision');
      const fundingReply = await funding;
      if (fundingReply.status() === 200) {
        const allowance = object(await fundingReply.json());
        assert.equal(allowance.intent, intent);
        assert(typeof allowance.amount === 'number' && allowance.amount > 0 && Number.isFinite(allowance.amount));
        evidence.factory?.push(
          `allowance funded amount=${allowance.amount} unit=${String(allowance.unit)} reserved=${String(allowance.reserved)} consumed=${String(allowance.consumed)}`
        );
      } else {
        assert.equal(fundingReply.status(), 404);
        const createRequest = await creating;
        assert(createRequest, 'Missing allowance must be created before explore');
        const createBody = object(createRequest.postDataJSON());
        recordWrite(new URL(createRequest.url()).pathname, createBody);
        assert.match(String(createBody.id ?? ''), /^d[0-9a-f]{24}$/);
        assert.equal(createBody.intent, intent);
        assert(typeof createBody.amount === 'number' && createBody.amount > 0 && Number.isFinite(createBody.amount));
        assert.equal(createBody.ceiling, createBody.amount);
        assert.equal(createBody.unit, 'planning_calls');
        const createResponse = await created;
        assert(createResponse, 'Allowance creation must report its receipt');
        assert.equal(createResponse.status(), 201);
        const createdAllowance = object(await createResponse.json());
        assert.equal(createdAllowance.intent, intent);
        assert.equal(createdAllowance.amount, createBody.amount);
        evidence.factory?.push(`allowance created amount=${String(createBody.amount)} unit=${String(createBody.unit)}`);
      }
      const exploreRequest = await consulting,
        exploreBody = object(exploreRequest.postDataJSON());
      recordWrite(new URL(exploreRequest.url()).pathname, exploreBody);
      assert.equal(exploreBody.revision, revision);
      const explored = await consulted;
      evidence.explore_status = explored.status();
      assert.equal(
        explored.status(),
        201,
        'Design exploration needs a funded planning allowance and reachable design judgments'
      );
      const proposal = object(await explored.json());
      assert.equal(proposal.intent, intent);
      assert.equal(proposal.predecessor, revision);
      assert.equal(proposal.state, 'draft');
      revision = String(proposal.id ?? '');
    };
    if (!input.savedDraft?.explored) await exploreRound('Explore ideas');
    evidence.intent = intent;
    if (entry) {
      await page.waitForURL((u) => new URL(u).pathname.endsWith(`/requests/${intent}`), {timeout: 60000});
      const routed = new URL(page.url());
      assert.equal(routed.search, entry.search, 'Native query must be preserved');
      assert.equal(routed.hash, entry.hash, 'Native fragment must be preserved');
    }
    await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor({timeout: 60000});
    evidence.stage = 'consequential review decisions';
    const applied: string[] = [];
    const decisions = [...input.decisions];
    const decide = (text: string | null, apply: (choice: FirstUseDecision) => Promise<void>) => {
      const found = decisions.filter((choice) => choice.match && (text ?? '').includes(choice.match));
      assert.equal(found.length, 1, `Exactly one recorded decision must match: ${(text ?? '').slice(0, 80)}`);
      const choice = found[0];
      assert(choice && !applied.includes(choice.match), 'Each recorded decision applies once');
      applied.push(choice.match);
      return apply(choice);
    };
    const decisionPass = async () => {
      const ideas = spaces.locator('section.intent-review > article');
      if (native.reviewDecisions) {
        const pending: {kind: 'idea' | 'challenge'; text: string}[] = [];
        for (let index = 0; index < (await ideas.count()); index++) {
          const article = ideas.nth(index),
            choice = article.locator('select[aria-label^="Choice: "]');
          if ((await choice.count()) && (await choice.inputValue()) === 'pending')
            pending.push({kind: 'idea', text: (await article.locator('p').first().textContent()) ?? ''});
        }
        const questions = spaces.locator('section[aria-label="Questions needing your answer"] article');
        for (let index = 0; index < (await questions.count()); index++) {
          const article = questions.nth(index);
          if (!((await article.textContent()) ?? '').includes('Answered:'))
            pending.push({kind: 'challenge', text: (await article.locator('p').first().textContent()) ?? ''});
        }
        const unknown = pending.filter((item) => !decisions.some((choice) => item.text.includes(choice.match)));
        if (unknown.length) decisions.push(...(await native.reviewDecisions(unknown)));
      }
      for (let index = 0; index < (await ideas.count()); index++) {
        const article = ideas.nth(index),
          choice = article.locator('select[aria-label^="Choice: "]');
        if (!(await choice.count()) || (await choice.inputValue()) !== 'pending') continue;
        const description = await article.locator('p').first().textContent();
        await decide(description, async (selected) => {
          await choice.selectOption(selected.decision);
          if (selected.decision === 'edit') {
            const correction = selected.correction;
            assert(typeof correction === 'string' && correction.trim(), 'An edit decision needs its correction');
            await article.getByLabel('What should change?').fill(correction);
          }
        });
      }
      const challenges = spaces.locator('section[aria-label="Questions needing your answer"] article');
      for (let index = 0; index < (await challenges.count()); index++) {
        const article = challenges.nth(index);
        if (((await article.textContent()) ?? '').includes('Answered:')) continue;
        const assumption = await article.locator('p').first().textContent();
        await decide(assumption, async (selected) => {
          if (selected.decision === 'accept')
            await article.getByRole('button', {name: /^Use recommended resolution for /}).click();
          else {
            const correction = selected.correction;
            assert(
              selected.decision === 'edit' && typeof correction === 'string' && correction.trim(),
              'Answer each question explicitly'
            );
            await article.getByLabel(/^Explain a correction to /).fill(correction);
          }
        });
      }
    };
    for (let round = 0; ; round++) {
      await decisionPass();
      const update = spaces.getByRole('button', {name: 'Update proposal'});
      if (await update.isDisabled()) break;
      assert(round < 2, 'Review must settle after recorded decisions and refresh');
      await exploreRound('Update proposal');
      await page.waitForFunction(
        () =>
          [...document.querySelectorAll('#sodaspaces-data button')].find(
            (node) => node.textContent?.trim() === 'Update proposal'
          ) instanceof HTMLButtonElement &&
          (
            [...document.querySelectorAll('#sodaspaces-data button')].find(
              (node) => node.textContent?.trim() === 'Update proposal'
            ) as HTMLButtonElement
          ).disabled,
        undefined,
        {timeout: 60000}
      );
    }
    assert.equal(applied.length, decisions.length, 'Every recorded decision must apply to review');
    evidence.decisions = applied;
    await native.capture?.('review');
    const cont = spaces.getByRole('button', {name: 'Choose repository'});
    assert(await cont.isEnabled(), 'Decided review must unlock repository choice');
    await cont.click();
    evidence.stage = 'native repository discovery';
    await spaces
      .getByRole('radio', {name: new RegExp(input.repositoryName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))})
      .check();
    await native.capture?.('picker');
    const boundRequest = page.waitForRequest(
      (request) => request.url().endsWith('/api/factory/intents') && request.method() === 'POST',
      {timeout: 120000}
    );
    const boundReply = page.waitForResponse(
      (response) => response.url().endsWith('/api/factory/intents') && response.request().method() === 'POST',
      {timeout: 120000}
    );
    const firstPlanning = page.waitForResponse(
      (response) =>
        response.url().endsWith('/api/factory/intents/' + intent + '/plan') && response.request().method() === 'POST',
      {timeout: 180000}
    );
    // Existing prepared Projects use the native Open project action; new
    // repositories use Continue. Both retain the same intent binding owner.
    await spaces.getByRole('button', {name: /^(Continue|Open project)$/}).click();
    const binding = await boundRequest;
    const bindingBody = object(binding.postDataJSON());
    recordWrite(new URL(binding.url()).pathname, bindingBody);
    assert.equal(bindingBody.state, 'draft');
    assert.equal(bindingBody.intent, intent);
    assert.equal(bindingBody.repository, input.repository);
    assert.equal(bindingBody.predecessor, revision);
    const bindingResponse = await boundReply;
    assert.equal(bindingResponse.status(), 201);
    revision = String(object(await bindingResponse.json()).id);
    assert.match(revision, /^d[0-9a-f]{24}$/);
    evidence.planning_revision = revision;
    return await completeInstalledIntentPlan(
      page,
      {...input, intent, revision},
      await firstPlanning,
      recordWrite,
      native,
      evidence
    );
  } finally {
    page.off('response', recordFactory);
  }
}

/** Shared installed path after the native planning POST. Both initial creation
 * and comment-driven changes use the same actual planner, authored walkthrough,
 * explicit review, accepted revision and graph owner. */
export async function completeInstalledIntentPlan(
  page: Page,
  input: {
    repository: string;
    objective: string;
    maxIssues: number;
    planningTimeoutMS: number;
    intent: string;
    revision: string;
  },
  initialPlan: Response,
  recordWrite: (path: string, body: Record<string, unknown>) => void,
  native: InstalledIntentOwners,
  evidence: FirstUseEvidence
): Promise<Record<string, unknown>> {
  const {subURL} = await mountedSpacesFacts(page);
  const spaces = await spacesContent(page);
  const {intent, revision} = input;
  recordWrite(new URL(initialPlan.url()).pathname, object(initialPlan.request().postDataJSON()));
  const initialResult = object(await initialPlan.json());
  evidence.factory?.push(`planning admission ${initialPlan.status()}: ${JSON.stringify(initialResult)}`);
  const admitted =
    (initialPlan.status() === 201 || initialPlan.status() === 202) && typeof initialResult.assignment_id === 'string';
  const recordPlan = (result: Record<string, unknown>) => {
    assert.match(String(result.assignment_id), /^[a-f0-9]{32}$/);
    assert.match(String(result.run_id), /^[a-f0-9]{32}$/);
    assert.equal(result.revision_id ?? result.revision, revision);
    if (evidence.planning_assignment) assert.equal(result.assignment_id, evidence.planning_assignment);
    if (evidence.planning_run) assert.equal(result.run_id, evidence.planning_run);
    evidence.planning_assignment = String(result.assignment_id);
    evidence.planning_run = String(result.run_id);
  };
  if (admitted) recordPlan(initialResult);
  if (!admitted) {
    assert(
      (initialPlan.status() === 202 && initialResult.launched === false && Array.isArray(initialResult.waits)) ||
        (initialPlan.status() === 503 && object(initialResult.error).code === 'planning_unavailable'),
      'Planning admission is uncertain or refused; inspect the original request instead of retrying'
    );
    evidence.stage = 'native project preparation';
    await spaces.getByRole('button', {name: 'Prepare project', exact: true}).click();
    const prepared = await native.prepareProject({repository: input.repository, intent, revision});
    assert(projectId(prepared.project), 'Preparation must return the actual native project identity');
    evidence.project = prepared.project;
    await page.goto(new URL(subURL + '/workspace/requests/' + intent, page.url()).href);
    await nativeSpacesMount(page);
    await spaces.getByRole('heading', {name: 'Review your product direction'}).waitFor();
    const rereading = page.waitForResponse(
      (response) =>
        response.url().endsWith('/api/factory/intents/' + intent + '/plan') && response.request().method() === 'GET',
      {timeout: 60000}
    );
    await spaces.getByRole('button', {name: 'Refresh status', exact: true}).click();
    const read = await rereading;
    assert.equal(read.status(), 200);
    const existing = object(await read.json());
    assert.equal(existing.revision, revision);
    if (existing.state === 'none') {
      const planning = page.waitForResponse(
        (response) =>
          response.url().endsWith('/api/factory/intents/' + intent + '/plan') && response.request().method() === 'POST',
        {timeout: 180000}
      );
      await spaces.getByRole('button', {name: 'Continue to planning', exact: true}).click();
      const launched = await planning;
      recordWrite(new URL(launched.url()).pathname, object(launched.request().postDataJSON()));
      const result = object(await launched.json());
      evidence.factory?.push(`planning admission ${launched.status()}: ${JSON.stringify(result)}`);
      assert(
        [201, 202].includes(launched.status()) && typeof result.assignment_id === 'string',
        'Planning remains held; preserve its actual reason'
      );
      recordPlan(result);
    } else {
      assert(
        ['pending', 'completed'].includes(String(existing.state)),
        'Inspect the existing planning result before another attempt'
      );
      recordPlan(existing);
    }
  }
  evidence.stage = 'actual provider planning';
  const deadline = Date.now() + input.planningTimeoutMS;
  let plan: Record<string, unknown>;
  for (;;) {
    assert(Date.now() < deadline, 'Finite planning observation expired; retain the admitted run');
    const refreshed = page.waitForResponse(
      (response) =>
        response.url().endsWith('/api/factory/intents/' + intent + '/plan') && response.request().method() === 'GET',
      {timeout: 60000}
    );
    await spaces.getByRole('button', {name: 'Refresh status', exact: true}).click();
    const response = await refreshed;
    assert.equal(response.status(), 200);
    plan = object(await response.json());
    assert.equal(plan.revision, revision);
    recordPlan(plan);
    if (plan.state !== 'pending') break;
    await page.waitForTimeout(1500);
  }
  assert.equal(plan.state, 'completed', 'Actual planning must complete; attention/failure is not substituted');
  const walkthrough = object(plan.walkthroughs);
  assert.equal(walkthrough.producer, 'soda-planning-agent-v1');
  assert.equal(walkthrough.intent, intent);
  assert.equal(walkthrough.revision, revision);
  assert.equal(walkthrough.objective, input.objective.trim());
  assert(Array.isArray(walkthrough.alternatives) && walkthrough.alternatives.length === 2);
  await native.capture?.('authored-walkthrough');
  const held = await native.holdExecution({
    repository: input.repository,
    intent,
    assignment: evidence.planning_assignment!,
    run: evidence.planning_run!,
  });
  assert(projectId(held.project), 'The execution owner must identify the actual project');
  if (evidence.project) assert.equal(held.project, evidence.project);
  evidence.project = held.project;
  assert(held.paused === true && held.dispatchOpen === false, 'Native execution must be held before graph generation');
  await page.goto(new URL(subURL + '/workspace/requests/' + intent, page.url()).href);
  await nativeSpacesMount(page);
  evidence.stage = 'explicit walkthrough review';
  if (!plan.review) {
    const checking = page.waitForResponse(
      (response) =>
        response.url().endsWith('/api/factory/intents/' + intent + '/plan/review') &&
        response.request().method() === 'POST',
      {timeout: 180000}
    );
    await spaces.getByRole('button', {name: 'Check walkthrough', exact: true}).click();
    const checked = await checking;
    recordWrite(new URL(checked.url()).pathname, object(checked.request().postDataJSON()));
    assert.equal(checked.status(), 200);
    plan.review = object(await checked.json());
  }
  const review = object(plan.review),
    consistency = object(review.consistency);
  assert.equal(object(review.content).revision, revision);
  assert.equal(object(review.recommendation).revision, revision);
  assert.equal(consistency.revision, revision);
  assert(
    Array.isArray(consistency.mismatches) && consistency.mismatches.length === 0,
    'Correct actual walkthrough findings before adoption'
  );
  evidence.walkthrough_reviewed = true;
  await native.capture?.('checked-walkthrough');
  evidence.stage = 'explicit accepted scope and native generation';
  const accepting = page.waitForRequest(
    (request) => request.url().endsWith('/api/factory/intents') && request.method() === 'POST',
    {timeout: 120000}
  );
  const accepted = page.waitForResponse(
    (response) => response.url().endsWith('/api/factory/intents') && response.request().method() === 'POST',
    {timeout: 120000}
  );
  const generated = page.waitForResponse(
    (response) =>
      response.url().endsWith('/api/factory/intents/' + intent + '/generate') && response.request().method() === 'POST',
    {timeout: 180000}
  );
  await spaces.getByRole('button', {name: 'Accept and build', exact: true}).click();
  const acceptRequest = await accepting,
    acceptBody = object(acceptRequest.postDataJSON());
  recordWrite(new URL(acceptRequest.url()).pathname, acceptBody);
  assert.equal(acceptBody.state, 'accepted');
  assert.equal(acceptBody.repository, input.repository);
  assert.equal(acceptBody.intent, intent);
  assert.equal(acceptBody.predecessor, revision);
  assert.equal(acceptBody.max_issues, input.maxIssues);
  assert.equal((await accepted).status(), 201);
  evidence.accepted_revision = String(acceptBody.id);
  const generateResponse = await generated;
  recordWrite(new URL(generateResponse.url()).pathname, object(generateResponse.request().postDataJSON()));
  assert.equal(generateResponse.status(), 200);
  const generation = object(await generateResponse.json());
  assert.equal(generation.intent_id, intent);
  assert.equal(generation.revision_id, evidence.accepted_revision);
  assert.equal(generation.state, 'completed');
  assert(
    Array.isArray(generation.submitted_tasks) &&
      generation.submitted_tasks.length > 0 &&
      generation.submitted_tasks.length <= input.maxIssues,
    'Native graph must fit the actual accepted finite plan'
  );
  evidence.generation = String(generation.state);
  await native.capture?.('generated-work');
  evidence.stage = 'generated work ready for Factory';
  return generation;
}

/** Continue a real product-bound comment. The caller must first verify the
 * delivered v1 application, renewed protected-check approval and remaining
 * allowance. This function never authors a comment or invents an issue graph. */
export async function exerciseProductChange(
  page: Page,
  input: {
    repository: string;
    comment: string;
    commentRevision: number;
    intent: string;
    acceptedRevision: string;
    objective: string;
    maxIssues: number;
    planningTimeoutMS: number;
    priorGeneration: Record<string, unknown>;
  },
  recordWrite: (path: string, body: Record<string, unknown>) => void,
  native: InstalledIntentOwners,
  evidence: FirstUseEvidence
): Promise<Record<string, unknown>> {
  await nativeSpacesMount(page);
  const spaces = await spacesContent(page);
  assert.match(input.comment, /^[0-9a-f]{32}$/);
  assert.equal(input.maxIssues, 2, 'Finance admits one additional work item while retaining v1');
  assert.equal(input.priorGeneration.intent_id, input.intent);
  assert.equal(input.priorGeneration.revision_id, input.acceptedRevision);
  assert.equal(input.priorGeneration.state, 'completed');
  const priorTasks = input.priorGeneration.submitted_tasks;
  assert(Array.isArray(priorTasks) && priorTasks.length === 1);
  evidence.intent = input.intent;
  evidence.stage = 'explicit Finance comment review';
  const review = page.waitForResponse(
    (response) =>
      response.url().endsWith(`/api/factory/feed/comments/${input.comment}/review`) &&
      response.request().method() === 'POST',
    {timeout: 60000}
  );
  const reading = page.waitForResponse(
    (response) =>
      response.url().endsWith(`/api/factory/intents/${input.intent}`) && response.request().method() === 'GET',
    {timeout: 60000}
  );
  const reviewButton = spaces.locator(`[data-comment-product-change="${input.comment}"]`);
  assert.equal((await reviewButton.textContent())?.trim(), 'Review proposed change');
  await reviewButton.click();
  const reviewed = await review;
  recordWrite(new URL(reviewed.url()).pathname, object(reviewed.request().postDataJSON()));
  assert.equal(reviewed.status(), 200);
  assert.equal(object(reviewed.request().postDataJSON()).revision, input.commentRevision);
  const projection = object(object(await reviewed.json()).product_change);
  assert.equal(projection.intent, input.intent);
  assert.equal(projection.accepted_revision, input.acceptedRevision);
  assert.equal(projection.state, 'draft');
  assert.equal(projection.can_review, true);
  const read = await reading;
  assert.equal(read.status(), 200);
  const draft = object(await read.json());
  assert.equal(draft.id, projection.draft_revision);
  assert.equal(draft.intent, input.intent);
  assert.equal(draft.predecessor, input.acceptedRevision);
  assert.equal(draft.repository, input.repository);
  assert.equal(draft.objective, input.objective.trim());
  assert.equal(draft.state, 'draft');
  assert.deepEqual(draft.tasks, priorTasks, 'The canonical handoff must supply actual generated task identities');
  assert(Array.isArray(draft.ideas) && Array.isArray(draft.effective_outcomes));
  const commentIdeas = draft.ideas
    .map(object)
    .filter((idea) =>
      String(idea.evidence ?? '').includes(`Feed comment ${input.comment} revision ${input.commentRevision}`)
    );
  assert.equal(commentIdeas.length, 1, 'The reviewed comment must remain one attributed outcome');
  assert(draft.effective_outcomes.includes(commentIdeas[0]!.id));
  await spaces.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
  await native.capture?.('finance-draft');
  let revision = String(draft.id);
  assert.equal(draft.max_issues, 1, 'Do not silently reuse a separately modified successor');
  evidence.stage = 'explicit retained-work bounds update';
  await spaces.locator('details.bounds summary').click();
  await spaces.getByLabel('Maximum work items', {exact: true}).fill(String(input.maxIssues));
  assert(await spaces.getByRole('button', {name: 'Continue to planning', exact: true}).isDisabled());
  const saving = page.waitForResponse(
    (response) => response.url().endsWith('/api/factory/intents') && response.request().method() === 'POST',
    {timeout: 60000}
  );
  await spaces.getByRole('button', {name: 'Update proposal', exact: true}).click();
  const saved = await saving;
  const body = object(saved.request().postDataJSON());
  recordWrite(new URL(saved.url()).pathname, body);
  assert.equal(saved.status(), 201);
  assert.equal(body.intent, input.intent);
  assert.equal(body.predecessor, revision);
  assert.equal(body.repository, input.repository);
  assert.equal(body.max_issues, input.maxIssues);
  assert.equal(body.planning_calls, draft.planning_calls);
  assert.deepEqual(body.tasks, draft.tasks, 'Bounds changes must preserve recorded generated work');
  assert.deepEqual(body.ideas, draft.ideas, 'Bounds changes must preserve comment identity and provenance');
  assert.deepEqual(body.effective_outcomes, draft.effective_outcomes);
  revision = String(object(await saved.json()).id);
  evidence.planning_revision = revision;
  await native.capture?.('finance-bounds');
  const planning = page.waitForResponse(
    (response) =>
      response.url().endsWith(`/api/factory/intents/${input.intent}/plan`) && response.request().method() === 'POST',
    {timeout: 180000}
  );
  await spaces.getByRole('button', {name: 'Continue to planning', exact: true}).click();
  const generation = await completeInstalledIntentPlan(
    page,
    {...input, revision},
    await planning,
    recordWrite,
    native,
    evidence
  );
  const priorIDs = object(input.priorGeneration.task_issue_ids);
  const nextIDs = object(generation.task_issue_ids);
  const delivered = object(generation.delivered);
  for (const [key, issue] of Object.entries(priorIDs)) {
    assert.equal(nextIDs[key], issue, 'Previously delivered native issue identities must be retained');
    assert(delivered[key], 'The new generation must carry delivered v1 history');
  }
  const added = Object.keys(nextIDs).filter((key) => !(key in priorIDs));
  assert.equal(added.length, 1, 'Finance creates exactly the newly accepted work item');
  assert(!Object.values(priorIDs).includes(nextIDs[added[0]!]));
  assert.equal(generation.submitted_tasks instanceof Array && generation.submitted_tasks.length, 2);
  evidence.stage = 'Finance work generated with v1 retained; implementation held';
  return generation;
}
