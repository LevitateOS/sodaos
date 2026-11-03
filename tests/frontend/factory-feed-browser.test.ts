import {buildFactoryUI} from '@factory/product/scripts/build-factory-ui.js';
import {mkdir, mkdtemp, rm} from 'node:fs/promises';
import {resolve} from 'node:path';
import {homedir} from 'node:os';
import test from 'node:test';
import assert from 'node:assert/strict';
import {setupProjectControlsDriver, fixture, refresh} from './fixtures/project-controls-driver';
import {chromium} from 'playwright';
import {factoryAssetSource} from './fixtures/factory-asset-source.js';
setupProjectControlsDriver();

const recordId = 'a'.repeat(32);
const postId = 'c'.repeat(32);
const humanReplyId = 'd'.repeat(32);

test('shared issue editor preserves original bases and newer drafts across storage faults and retired replies', async (t) => {
  const page = await fixture(t),
    origin = new URL(page.url()).origin;
  const setup = () =>
    page.evaluate(() => {
      const repository = '9007199254740993',
        host = 'a'.repeat(32),
        parent = {kind: 'issue', repository, number: '42', external_id: 'native-42'};
      let actor = '11',
        mode = 'ready',
        release: (() => void) | undefined,
        lostCancel = false;
      const current = {title: 'Stored issue title', body: 'Stored issue body.'};
      const observation = {state: 'available', revision: 'declared-editor-peer', observed_at: '2026-11-03T12:00:00Z'};
      type Input = {
        command_id: string;
        title?: string;
        body?: string;
        expected: {external_id: string; title: string; body_digest: string};
      };
      const commands = new Map<string, {input: Input; committed: boolean; cancelled: boolean}>();
      window.addEventListener('editor-peer', (event) => {
        const next = (event as CustomEvent<{mode?: string; actor?: string}>).detail;
        mode = next.mode ?? mode;
        actor = next.actor ?? actor;
      });
      window.addEventListener('editor-release', () => release?.());
      const result = (command: string, row: {committed: boolean; cancelled: boolean}) =>
        Response.json({
          command_id: command,
          observation: row.committed || row.cancelled ? 'found' : 'not_observed',
          ...(row.committed || row.cancelled
            ? {
                effect: row.committed ? 'committed' : 'not_committed',
                cancellation: row.cancelled ? 'cancelled' : 'none',
                completion: mode === 'unfinished' ? 'needs_intervention' : 'complete',
              }
            : {}),
          finished: (row.committed || row.cancelled) && mode !== 'unfinished',
        });
      const hash = async (body: string) =>
        [...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(body)))]
          .map((byte) => byte.toString(16).padStart(2, '0'))
          .join('');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/factory/identity'))
          return Response.json({host, actor, native_actor: actor === '11' ? '1' : '2'});
        if (
          call.url.endsWith('/native-repositories/resolve') ||
          call.url.endsWith('/factory/repositories/' + repository + '/identity')
        )
          return Response.json({
            host,
            actor,
            repository,
            native_repository: '7',
            name: 'alice/demo',
            default_branch: '',
          });
        if (!call.url.includes('/collaboration/')) {
          if (call.url.includes('/factory/issues/')) return Response.json({}, {status: 404});
          return null;
        }
        if (call.url.endsWith('/capabilities'))
          return Response.json({
            repository,
            actor: mode === 'actor-mismatch' ? '22' : actor,
            issue: parent,
            pull: null,
            head: null,
            base: null,
            observation,
            actions: {
              'issue.edit': {state: mode === 'denied' ? 'denied' : 'allowed'},
              'issue.comment': {state: 'denied'},
            },
          });
        if (call.url.includes('/comments?')) return Response.json({observation, items: [], next: '', complete: true});
        const command = /\/edits\/commands\/([a-f0-9]{32})/.exec(call.url)?.[1];
        if (command) {
          const row = commands.get(command);
          if (!row) return Response.json({error: {code: 'edit_command_not_found'}}, {status: 404});
          if (call.method === 'POST') {
            if (mode === 'cancel-lost' && !lostCancel) {
              lostCancel = true;
              throw Error('Declared cancellation reply unavailable');
            }
            if (!row.committed) row.cancelled = true;
          }
          return result(command, row);
        }
        if (call.method === 'POST' && call.url.endsWith('/edits')) {
          const input = JSON.parse(call.body!) as Input;
          let row = commands.get(input.command_id);
          if (row && JSON.stringify(row.input) !== JSON.stringify(input)) return Response.json({}, {status: 409});
          row ??= {input, committed: false, cancelled: false};
          commands.set(input.command_id, row);
          if (mode === 'held')
            await new Promise<void>((resolve) => {
              release = resolve;
            });
          if (mode === 'malformed')
            return Response.json({
              command_id: input.command_id,
              observation: 'found',
              effect: 'pending',
              cancellation: 'none',
              completion: 'complete',
              finished: true,
            });
          if (mode !== 'unknown' && mode !== 'cancel-lost' && !row.committed && !row.cancelled) {
            if (
              input.expected.external_id !== parent.external_id ||
              input.expected.title !== current.title ||
              input.expected.body_digest !== (await hash(current.body))
            )
              row.cancelled = true;
            else {
              row.committed = true;
              if (input.title !== undefined) current.title = input.title;
              if (input.body !== undefined) current.body = input.body;
            }
          }
          return result(input.command_id, row);
        }
        return Response.json({
          ref: parent,
          observation,
          ...current,
          state: 'open',
          author: null,
          assignees: [],
          labels: [],
          created_at: null,
          updated_at: null,
          closed_at: null,
          capabilities: null,
        });
      });
      window.drawerFixture.mountRepository('7', '/alice/demo/feed/work/issues/42');
    });
  await setup();
  const editor = page.locator('soda-factory-issue-editor'),
    title = editor.getByRole('textbox', {name: 'Edit issue title', exact: true}),
    body = editor.getByRole('textbox', {name: 'Edit issue Markdown', exact: true}),
    save = editor.getByRole('button', {name: 'Save issue', exact: true});
  const idle = () => editor.locator('section[aria-busy=false]').waitFor();
  const open = async () => {
    await editor.getByRole('button', {name: 'Edit issue', exact: true}).click();
    await idle();
  };
  const mode = (value: string, actor?: string) =>
    page.evaluate(
      ({value, actor}) => window.dispatchEvent(new CustomEvent('editor-peer', {detail: {mode: value, actor}})),
      {value, actor}
    );
  const remount = async () => {
    await page.evaluate(() => window.drawerFixture.mountRepository('7', '/alice/demo/feed/work/issues/42'));
    await idle();
  };
  const posts = () =>
    page.evaluate(() =>
      window.drawerFixture.calls
        .filter((c) => c.method === 'POST' && c.url.endsWith('/edits'))
        .map(
          (c) =>
            JSON.parse(c.body!) as {
              command_id: string;
              title?: string;
              body?: string;
              expected: {external_id: string; title: string; body_digest: string};
            }
        )
    );
  const record = () =>
    page.evaluate(() => {
      const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-edit:') && k.includes(':11:'))!;
      return JSON.parse(localStorage.getItem(key)!) as {
        writing: import('@factory/product/frontend/factory/issues/editor-record.js').EditorWriting;
      };
    });
  await open();
  assert(await save.isDisabled());
  await title.fill('T'.repeat(251) + '🌋');
  await body.fill('<'.repeat(65532) + '🌋');
  assert(await save.isEnabled());
  await title.fill('T'.repeat(252) + '🌋');
  assert(await save.isDisabled());
  await title.fill('Valid');
  await body.fill('x'.repeat(65537));
  assert(await save.isDisabled());
  assert.equal((await body.inputValue()).length, 65537);
  await body.fill('');
  assert(await save.isEnabled());
  await title.fill('\u0085');
  assert(await save.isDisabled());
  await page.evaluate(() => {
    const field = document.querySelector<HTMLInputElement>('soda-factory-issue-editor input')!;
    field.value = 'invalid\ud800';
    field.dispatchEvent(new Event('input', {bubbles: true}));
  });
  assert(await save.isDisabled());
  await page.evaluate(() => {
    const original = Storage.prototype.setItem;
    window.addEventListener(
      'editor-storage-restore',
      () => {
        Storage.prototype.setItem = original;
      },
      {once: true}
    );
    Storage.prototype.setItem = function (key, value) {
      if (key.startsWith('soda-issue-edit:')) throw new DOMException('Declared quota failure', 'QuotaExceededError');
      return original.call(this, key, value);
    };
  });
  await title.fill('Unsaved original edit title');
  await body.fill('Unsaved original edit body.');
  await save.click();
  await idle();
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.dispatchEvent(new Event('editor-storage-restore')));
  await remount();
  assert.equal(await title.inputValue(), 'Unsaved original edit title');
  await page.evaluate(() => {
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-edit:') && k.includes(':11:'))!;
    const value = JSON.parse(localStorage.getItem(key)!);
    value.writing.draft = {
      ...value.writing.draft,
      revision: 'e'.repeat(32),
      title: 'Newer saved edit title',
      body: 'Newer saved edit body.',
    };
    delete value.writing.pending;
    localStorage.setItem(key, JSON.stringify(value));
  });
  await remount();
  await editor.getByRole('button', {name: 'Reload saved writing', exact: true}).click();
  await idle();
  await editor.getByRole('button', {name: 'Retry saved edit', exact: true}).click();
  await page
    .locator('soda-factory-issue')
    .getByRole('heading', {name: 'Unsaved original edit title', exact: true})
    .waitFor();
  await idle();
  assert.equal((await posts()).length, 1);
  assert.equal((await record()).writing.draft?.title, 'Newer saved edit title');
  assert.equal((await record()).writing.draft?.body, 'Newer saved edit body.');
  await editor.getByRole('button', {name: 'Use current issue as base', exact: true}).click();
  await mode('unknown');
  await title.fill('Cancelled original title');
  await body.fill('Cancelled original body.');
  await save.click();
  await idle();
  const original = (await record()).writing.pending!;
  assert(original);
  await mode('cancel-lost');
  await editor.getByRole('button', {name: 'Cancel edit submission', exact: true}).click();
  await idle();
  assert((await record()).writing.pending?.cancel);
  await page.evaluate(() => {
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-edit:') && k.includes(':11:'))!;
    const value = JSON.parse(localStorage.getItem(key)!);
    value.writing.draft = {
      ...value.writing.draft,
      revision: 'f'.repeat(32),
      title: 'Newer cancelled edit title',
      body: 'Newer cancelled edit body.',
    };
    localStorage.setItem(key, JSON.stringify(value));
  });
  await remount();
  await editor.getByRole('button', {name: 'Retry cancellation', exact: true}).click();
  await idle();
  assert.equal(await body.inputValue(), 'Cancelled original body.');
  await editor.getByRole('button', {name: 'Discard original edit and continue with newer draft', exact: true}).click();
  assert.equal(await title.inputValue(), 'Newer cancelled edit title');
  assert.equal(await body.inputValue(), 'Newer cancelled edit body.');
  const cancels = await page.evaluate(() =>
    window.drawerFixture.calls.filter((c) => c.method === 'POST' && c.url.endsWith('/cancel'))
  );
  assert.equal(cancels.length, 2);
  assert(cancels.every((c) => c.body === '{}' && c.url.includes(original.command)));
  assert.equal((await posts()).length, 2);
  await mode('malformed');
  await save.click();
  await idle();
  assert((await record()).writing.pending);
  await mode('unknown');
  await editor.getByRole('button', {name: 'Check edit status', exact: true}).click();
  await idle();
  assert((await record()).writing.pending);
  await mode('ready', '22');
  await remount();
  await open();
  assert.equal(await title.inputValue(), 'Unsaved original edit title');
  assert.equal((await record()).writing.pending?.source.title, 'Newer cancelled edit title');
  await mode('ready', '11');
  await remount();
  await editor.getByRole('button', {name: 'Retry saved edit', exact: true}).click();
  await page
    .locator('soda-factory-issue')
    .getByRole('heading', {name: 'Newer cancelled edit title', exact: true})
    .waitFor();
  await open();
  for (const denied of ['denied', 'actor-mismatch']) {
    await mode(denied);
    await editor.getByRole('button', {name: 'Check issue edit access', exact: true}).click();
    await idle();
    await body.fill('Refused pending writing');
    assert(await save.isDisabled());
  }
  await mode('held');
  await editor.getByRole('button', {name: 'Check issue edit access', exact: true}).click();
  await idle();
  await title.fill('Held original edit');
  await body.fill('Held original edit body.');
  await save.click();
  await page.waitForFunction(() =>
    window.drawerFixture.calls.some((c) => c.body?.includes('Held original edit body.'))
  );
  await page.evaluate(() => {
    window.drawerFixture.repositoryHandle?.dispose?.();
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-edit:') && k.includes(':11:'))!;
    const value = JSON.parse(localStorage.getItem(key)!);
    value.writing.draft = {
      ...value.writing.draft,
      revision: 'b'.repeat(32),
      title: 'Newer after retired edit',
      body: 'Retain this newer edit description.',
    };
    localStorage.setItem(key, JSON.stringify(value));
    window.dispatchEvent(new Event('editor-release'));
  });
  await mode('ready');
  await remount();
  await idle();
  assert.equal((await record()).writing.draft?.title, 'Newer after retired edit');
  assert.equal((await record()).writing.draft?.body, 'Retain this newer edit description.');
  await editor.getByRole('button', {name: 'Use current issue as base', exact: true}).click();
  await mode('unknown');
  await save.click();
  await idle();
  const retained = (await record()).writing.pending!;
  assert(retained);
  await page.goto(origin);
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(async () => {
    window.drawerFixture = window.createDrawerFixture({});
    await window.drawerFixture.api.ready;
  });
  await setup();
  await idle();
  assert.equal((await posts()).length, 0);
  assert.equal((await record()).writing.pending?.command, retained.command);
  assert.equal(await title.inputValue(), retained.source.title);
  assert.equal(await body.inputValue(), retained.source.body);
  await page.evaluate(() => {
    window.drawerFixture.repositoryHandle?.dispose?.();
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-edit:') && k.includes(':11:'))!;
    localStorage.setItem(key, '{"future-edit":"preserve"}');
  });
  await remount();
  assert(await editor.getByRole('button', {name: 'Reload saved writing', exact: true}).isEnabled());
  assert(await save.isDisabled());
  assert.equal((await posts()).length, 0);
  assert(
    await page.evaluate(() =>
      Object.keys(localStorage).some((key) => localStorage.getItem(key) === '{"future-edit":"preserve"}')
    )
  );
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});

test(
  'native Fountain browser editor saves, reviews stale content and recovers original commands',
  {skip: !process.env.SODA_FOUNTAIN_BROWSER_API || process.env.SODA_FOUNTAIN_ISSUE_EDITOR_BROWSER !== '1'},
  async (t) => {
    const page = await fixture(t),
      api = process.env.SODA_FOUNTAIN_BROWSER_API!,
      native = process.env.SODA_FOUNTAIN_BROWSER_REPOSITORY!;
    const mount = () =>
      page.evaluate(
        ({api, native}) => {
          window.drawerFixture.forwardAPI(api);
          window.drawerFixture.mountRepository(native, '/user1/human-comment-primary/feed/work/issues/1', '', true);
        },
        {api, native}
      );
    await mount();
    const editor = page.locator('soda-factory-issue-editor'),
      detail = page.locator('soda-factory-issue'),
      title = editor.getByRole('textbox', {name: 'Edit issue title', exact: true}),
      body = editor.getByRole('textbox', {name: 'Edit issue Markdown', exact: true});
    const idle = () => editor.locator('section[aria-busy=false]').waitFor();
    const open = async () => {
      await editor.getByRole('button', {name: 'Edit issue', exact: true}).click();
      await idle();
      await body.waitFor();
    };
    const save = () => editor.getByRole('button', {name: 'Save issue', exact: true}).click();
    await open();
    await title.fill('Editor native title 完整');
    await body.fill('Native browser edit body.\n');
    await save();
    await detail.getByRole('heading', {name: 'Editor native title 完整', exact: true}).waitFor();
    await detail.getByText('Native browser edit body.', {exact: true}).waitFor();
    await open();
    await body.fill('HTTP browser edit committed reply dropped.');
    await save();
    await editor.getByText(/No final outcome was confirmed/).waitFor();
    const origin = new URL(page.url()).origin;
    await page.goto(origin);
    await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
    await page.evaluate(async () => {
      window.drawerFixture = window.createDrawerFixture({});
      await window.drawerFixture.api.ready;
    });
    await mount();
    await detail.getByText('HTTP browser edit committed reply dropped.', {exact: true}).waitFor();
    await editor.getByRole('button', {name: 'Edit issue', exact: true}).waitFor();
    const posts = () =>
      page.evaluate(() =>
        window.drawerFixture.calls
          .filter((call) => call.method === 'POST' && call.url.endsWith('/edits'))
          .map(
            (call) =>
              JSON.parse(call.body!) as {
                command_id: string;
                title?: string;
                body?: string;
                expected: {external_id: string; title: string; body_digest: string};
              }
          )
      );
    assert.equal((await posts()).length, 0);
    await open();
    await body.fill('Native rebased edit body.');
    const canonical = await page.evaluate(() =>
      window.drawerFixture.calls
        .map((c) => /\/collaboration\/repositories\/(\d+)\/issues\/1/.exec(c.url)?.[1])
        .find(Boolean)
    );
    assert(canonical);
    const remote = await fetch(new URL(`api/factory/collaboration/repositories/${canonical}/issues/1/edits`, api), {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        command_id: 'f'.repeat(32),
        title: 'Remote native title changed',
        body: 'Remote native description changed.',
      }),
    });
    assert.equal(remote.status, 200);
    await save();
    await editor.getByText(/draft was preserved.*Review the current issue/).waitFor();
    assert.equal((await posts()).length, 0);
    assert.equal(
      await editor.getByRole('textbox', {name: 'Current issue title', exact: true}).inputValue(),
      'Remote native title changed'
    );
    assert.equal(
      await editor.getByRole('textbox', {name: 'Current issue Markdown', exact: true}).inputValue(),
      'Remote native description changed.'
    );
    await editor.getByRole('button', {name: 'Use current issue as base', exact: true}).click();
    assert.equal(await title.inputValue(), 'Remote native title changed');
    assert.equal(await body.inputValue(), 'Native rebased edit body.');
    await save();
    await detail.getByText('Native rebased edit body.', {exact: true}).waitFor();
    assert.equal((await posts())[0]?.title, undefined);
    assert.equal((await posts())[0]?.expected.title, 'Remote native title changed');
    await open();
    await body.fill('Native race original draft.');
    assert.equal((await fetch(new URL('fixture/race-next', api), {method: 'POST'})).status, 204);
    await save();
    await editor
      .getByText('No changes were saved. Your draft was preserved. Check the current issue before retrying.', {
        exact: true,
      })
      .waitFor();
    assert.equal(await body.inputValue(), 'Native race original draft.');
    await editor.getByRole('button', {name: 'Check issue edit access', exact: true}).click();
    await idle();
    assert.equal(
      await editor.getByRole('textbox', {name: 'Current issue Markdown', exact: true}).inputValue(),
      'Native race changed body.'
    );
    await editor.getByRole('button', {name: 'Use current issue as base', exact: true}).click();
    const hold = async () =>
      assert.equal((await fetch(new URL('fixture/hold-next', api), {method: 'POST'})).status, 204);
    await body.fill('Native browser cancelled edit.');
    await hold();
    await save();
    await editor.getByText(/No final outcome was confirmed/).waitFor();
    await editor.getByRole('button', {name: 'Cancel edit submission', exact: true}).click();
    await editor.getByText('Edit submission cancelled. Your draft was preserved.', {exact: true}).waitFor();
    assert.equal(await body.inputValue(), 'Native browser cancelled edit.');
    await body.fill('Native browser retry edit.');
    await hold();
    await save();
    await editor.getByText(/No final outcome was confirmed/).waitFor();
    await editor.getByRole('button', {name: 'Check edit status', exact: true}).click();
    await editor.getByText(/edit is not confirmed/).waitFor();
    await editor.getByRole('button', {name: 'Retry saved edit', exact: true}).click();
    await detail.getByText('Native browser retry edit.', {exact: true}).waitFor();
    const retries = (await posts()).filter((p) => p.body === 'Native browser retry edit.');
    assert.equal(retries.length, 2);
    assert.deepEqual(retries[0], retries[1]);
    await open();
    await title.fill('Native title-only edit');
    await save();
    await detail.getByRole('heading', {name: 'Native title-only edit', exact: true}).waitFor();
    const titleOnly = (await posts()).find((p) => p.title === 'Native title-only edit');
    assert(titleOnly);
    assert.equal(titleOnly.body, undefined);
    await open();
    await body.fill('');
    await save();
    await editor.getByRole('button', {name: 'Edit issue', exact: true}).waitFor();
    await open();
    assert.equal(await body.inputValue(), '');
    const maximalTitle = 'T'.repeat(251) + '🌋',
      maximalBody = '<'.repeat(65532) + '🌋';
    await title.fill(maximalTitle);
    await body.fill(maximalBody);
    await save();
    await detail.getByRole('heading', {name: maximalTitle, exact: true}).waitFor();
    const sends = await posts();
    assert(
      sends.every(
        (p) =>
          Object.keys(p.expected).sort().join(',') === 'body_digest,external_id,title' &&
          /^[a-f0-9]{64}$/.test(p.expected.body_digest)
      )
    );
    const cancels = await page.evaluate(() =>
      window.drawerFixture.calls
        .filter((c) => c.method === 'POST' && c.url.includes('/edits/commands/') && c.url.endsWith('/cancel'))
        .map((c) => c.body)
    );
    assert.deepEqual(cancels, ['{}']);
    await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
  }
);

test('shared new issue preserves scoped writing, pending creation and destinations across storage failure and retirement', async (t) => {
  const page = await fixture(t),
    origin = new URL(page.url()).origin;
  const setup = () =>
    page.evaluate(() => {
      const repository = '9007199254740993',
        host = 'a'.repeat(32);
      let actor = '11',
        mode = 'ready',
        release: (() => void) | undefined,
        lostCancel = false;
      const commands = new Map<string, {title: string; body: string; committed: boolean; cancelled: boolean}>();
      const observation = {state: 'available', revision: 'declared-create-peer', observed_at: '2026-11-03T12:00:00Z'};
      window.addEventListener('create-peer', (event) => {
        const next = (event as CustomEvent<{mode?: string; actor?: string}>).detail;
        mode = next.mode ?? mode;
        actor = next.actor ?? actor;
      });
      window.addEventListener('create-release', () => release?.());
      const result = (command: string, row: {committed: boolean; cancelled: boolean}) =>
        Response.json({
          command_id: command,
          observation: row.committed || row.cancelled ? 'found' : 'not_observed',
          ...(row.committed || row.cancelled
            ? {
                effect: row.committed ? 'committed' : 'not_committed',
                cancellation: row.cancelled ? 'cancelled' : 'none',
                completion: 'complete',
              }
            : {}),
          finished: row.committed || row.cancelled,
          issue:
            row.committed && mode !== 'hidden'
              ? {
                  kind: 'issue',
                  repository: mode === 'malformed' ? '8' : repository,
                  external_id: 'native-created',
                  number: '9223372036854775806',
                }
              : null,
        });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/factory/identity'))
          return Response.json({host, actor, native_actor: actor === '11' ? '1' : '2'});
        if (
          call.url.endsWith('/native-repositories/resolve') ||
          call.url.endsWith('/factory/repositories/' + repository + '/identity')
        )
          return Response.json({
            host,
            actor,
            repository,
            native_repository: '7',
            name: 'alice/demo',
            default_branch: '',
          });
        if (!call.url.includes('/collaboration/')) return null;
        if (call.url.endsWith('/capabilities'))
          return Response.json({
            repository,
            actor: mode === 'actor-mismatch' ? '22' : actor,
            issue: null,
            pull: null,
            head: null,
            base: null,
            observation,
            actions: {'issue.create': {state: mode === 'denied' ? 'denied' : 'allowed'}},
          });
        const command = /\/issue-creation\/commands\/([a-f0-9]{32})/.exec(call.url)?.[1];
        if (command) {
          const row = commands.get(command);
          if (!row) return Response.json({error: {code: 'issue_create_command_not_found'}}, {status: 404});
          if (call.method === 'POST') {
            if (mode === 'cancel-lost' && !lostCancel) {
              lostCancel = true;
              throw Error('Declared lost cancel reply');
            }
            if (!row.committed) row.cancelled = true;
          }
          return result(command, row);
        }
        if (call.method === 'POST' && call.url.endsWith('/issues')) {
          const input = JSON.parse(call.body!) as {command_id: string; title: string; body: string};
          let row = commands.get(input.command_id);
          if (row && (row.title !== input.title || row.body !== input.body)) return Response.json({}, {status: 409});
          row ??= {title: input.title, body: input.body, committed: false, cancelled: false};
          commands.set(input.command_id, row);
          if (mode === 'held')
            await new Promise<void>((resolve) => {
              release = resolve;
            });
          if (mode !== 'unknown' && mode !== 'cancel-lost' && !row.cancelled) row.committed = true;
          return result(input.command_id, row);
        }
        return Response.json({}, {status: 404});
      });
      window.drawerFixture.mountRepository('7', '/alice/demo/feed/work/issues/new');
    });
  await setup();
  const form = page.locator('soda-factory-issue-create'),
    title = form.getByRole('textbox', {name: 'Issue title', exact: true}),
    body = form.getByRole('textbox', {name: 'Issue Markdown', exact: true}),
    create = form.getByRole('button', {name: 'Create issue', exact: true});
  const idle = () => form.locator('section[aria-busy=false]').waitFor();
  const mode = (value: string, actor?: string) =>
    page.evaluate(
      ({value, actor}) => window.dispatchEvent(new CustomEvent('create-peer', {detail: {mode: value, actor}})),
      {value, actor}
    );
  const posts = () =>
    page.evaluate(() =>
      window.drawerFixture.calls
        .filter((c) => c.method === 'POST' && /\/collaboration\/repositories\/\d+\/issues$/.test(c.url))
        .map((c) => JSON.parse(c.body!) as {command_id: string; title: string; body: string})
    );
  const remount = async () => {
    await page.evaluate(() => window.drawerFixture.mountRepository('7', '/alice/demo/feed/work/issues/new'));
    await idle();
  };
  const saved = () =>
    page.evaluate(() => {
      const key = Object.keys(localStorage).find(
        (key) => key.startsWith('soda-issue-create:') && key.includes(':11:')
      )!;
      return JSON.parse(localStorage.getItem(key)!) as {
        writing: {
          draft: {revision: string; title: string; body: string};
          pending?: {command: string; title: string; body: string; draft: string; cancel: boolean};
          created?: {command: string; issue: {number: string}};
        };
      };
    });
  await idle();
  await title.fill('V'.repeat(251) + '🌋');
  await body.fill('<'.repeat(65532) + '🌋');
  assert(await create.isEnabled());
  await title.fill('V'.repeat(252) + '🌋');
  assert(await create.isDisabled());
  assert.equal((await title.inputValue()).length, 254);
  await title.fill('Valid');
  await body.fill('x'.repeat(65537));
  assert(await create.isDisabled());
  assert.equal((await body.inputValue()).length, 65537);
  await body.fill('');
  assert(await create.isEnabled());
  await title.fill('\u0085padded');
  assert(await create.isDisabled());
  await title.fill('\ufeff');
  assert(await create.isEnabled());
  await page.evaluate(() => {
    const field = document.querySelector<HTMLInputElement>('soda-factory-issue-create input')!;
    field.value = 'bad\ud800';
    field.dispatchEvent(new Event('input', {bubbles: true}));
  });
  assert(await create.isDisabled());
  await page.evaluate(() => {
    const original = Storage.prototype.setItem;
    window.addEventListener(
      'restore-create-storage',
      () => {
        Storage.prototype.setItem = original;
      },
      {once: true}
    );
    Storage.prototype.setItem = function (key, value) {
      if (key.startsWith('soda-issue-create:')) throw new DOMException('Declared quota failure', 'QuotaExceededError');
      return original.call(this, key, value);
    };
  });
  await title.fill('Unsaved original title');
  await body.fill('Unsaved original body.');
  await create.click();
  await idle();
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => window.dispatchEvent(new Event('restore-create-storage')));
  await remount();
  assert.equal(await title.inputValue(), 'Unsaved original title');
  assert.equal(await body.inputValue(), 'Unsaved original body.');
  // A separate document's persisted draft changes while this realm owns an
  // unsaved original submission. Explicit recovery must retain both.
  await page.evaluate(() => {
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-create:'))!;
    const value = JSON.parse(localStorage.getItem(key)!);
    value.writing.draft = {revision: 'e'.repeat(32), title: 'Newer saved title', body: 'Newer saved body.'};
    delete value.writing.pending;
    localStorage.setItem(key, JSON.stringify(value));
  });
  await remount();
  await form.getByRole('button', {name: 'Reload saved writing', exact: true}).click();
  await idle();
  await form.getByRole('button', {name: 'Retry saved issue', exact: true}).click();
  await idle();
  assert.equal((await posts()).length, 1);
  assert.equal((await posts())[0]!.title, 'Unsaved original title');
  assert.equal((await posts())[0]!.body, 'Unsaved original body.');
  assert.equal((await saved()).writing.draft.title, 'Newer saved title');
  assert.equal((await saved()).writing.draft.body, 'Newer saved body.');
  assert.equal((await saved()).writing.created?.issue.number, '9223372036854775806');
  await form.getByRole('button', {name: 'Open created issue', exact: true}).click();
  assert((await page.evaluate(() => document.body.dataset.navigated))?.endsWith('/work/issues/9223372036854775806'));
  await form.getByRole('button', {name: 'Write another issue', exact: true}).click();
  await mode('unknown');
  await title.fill('Cancel original');
  await body.fill('Keep cancel original.');
  await create.click();
  await idle();
  const original = (await saved()).writing.pending!;
  assert(original && !original.cancel);
  await mode('cancel-lost');
  await form.getByRole('button', {name: 'Cancel submission', exact: true}).click();
  await idle();
  assert((await saved()).writing.pending?.cancel);
  await page.evaluate(() => {
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-create:') && k.includes(':11:'))!;
    const value = JSON.parse(localStorage.getItem(key)!);
    value.writing.draft = {revision: 'f'.repeat(32), title: 'Newer cancelled title', body: 'Newer cancelled body.'};
    localStorage.setItem(key, JSON.stringify(value));
  });
  await remount();
  assert(await form.getByRole('button', {name: 'Retry cancellation', exact: true}).isEnabled());
  await form.getByRole('button', {name: 'Retry cancellation', exact: true}).click();
  await idle();
  assert.equal(await title.inputValue(), 'Cancel original');
  assert.equal(await body.inputValue(), 'Keep cancel original.');
  await form.getByRole('button', {name: 'Discard original writing and continue with newer draft', exact: true}).click();
  assert.equal(await title.inputValue(), 'Newer cancelled title');
  assert.equal(await body.inputValue(), 'Newer cancelled body.');
  const cancels = await page.evaluate(() =>
    window.drawerFixture.calls.filter((c) => c.method === 'POST' && c.url.endsWith('/cancel'))
  );
  assert.equal(cancels.length, 2);
  assert(cancels.every((c) => c.body === '{}' && c.url.includes(original.command)));
  assert.equal((await posts()).length, 2);
  await mode('malformed');
  await create.click();
  await idle();
  assert((await saved()).writing.pending);
  assert.equal(await form.getByRole('button', {name: 'Open created issue', exact: true}).count(), 0);
  await mode('hidden');
  await form.getByRole('button', {name: 'Check issue status', exact: true}).click();
  await idle();
  await form.getByText(/destination is not currently readable/).waitFor();
  assert((await saved()).writing.pending);
  await mode('ready', '22');
  await remount();
  assert.equal(await title.inputValue(), '');
  assert.equal(await body.inputValue(), '');
  assert.equal((await saved()).writing.pending?.title, 'Newer cancelled title');
  await mode('ready', '11');
  await remount();
  await form.getByRole('button', {name: 'Open created issue', exact: true}).waitFor();
  assert(!(await saved()).writing.pending);
  await form.getByRole('button', {name: 'Write another issue', exact: true}).click();
  for (const denied of ['denied', 'actor-mismatch']) {
    await mode(denied);
    await form.getByRole('button', {name: 'Check issue access', exact: true}).click();
    await idle();
    await title.fill('Refused');
    assert(await create.isDisabled());
  }
  await mode('held');
  await form.getByRole('button', {name: 'Check issue access', exact: true}).click();
  await idle();
  await title.fill('Held original');
  await body.fill('Held original body.');
  await create.click();
  await page.waitForFunction(() => window.drawerFixture.calls.some((c) => c.body?.includes('Held original body.')));
  await page.evaluate(() => {
    window.drawerFixture.repositoryHandle?.dispose?.();
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-create:') && k.includes(':11:'))!;
    const value = JSON.parse(localStorage.getItem(key)!);
    value.writing.draft = {
      revision: 'b'.repeat(32),
      title: 'Newer after retirement',
      body: 'Keep this newer description.',
    };
    localStorage.setItem(key, JSON.stringify(value));
    window.dispatchEvent(new Event('create-release'));
  });
  await mode('ready');
  await remount();
  await form.getByRole('button', {name: 'Open created issue', exact: true}).waitFor();
  assert.equal((await saved()).writing.draft.title, 'Newer after retirement');
  assert.equal((await saved()).writing.draft.body, 'Keep this newer description.');
  await form.getByRole('button', {name: 'Write another issue', exact: true}).click();
  await mode('unknown');
  await create.click();
  await idle();
  const retained = (await saved()).writing.pending!;
  // A new JS realm has only durable browser writing; no automatic POST occurs.
  await page.goto(origin);
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(async () => {
    window.drawerFixture = window.createDrawerFixture({});
    await window.drawerFixture.api.ready;
  });
  await setup();
  await idle();
  assert.equal((await posts()).length, 0);
  assert.equal((await saved()).writing.pending?.command, retained.command);
  assert.equal(await title.inputValue(), retained.title);
  assert.equal(await body.inputValue(), retained.body);
  await page.evaluate(() => {
    window.drawerFixture.repositoryHandle?.dispose?.();
    const key = Object.keys(localStorage).find((k) => k.startsWith('soda-issue-create:') && k.includes(':11:'))!;
    localStorage.setItem(key, '{"future":"preserve"}');
  });
  await remount();
  assert(await form.getByRole('button', {name: 'Reload saved writing', exact: true}).isEnabled());
  assert(await create.isDisabled());
  assert.equal((await posts()).length, 0);
  assert(
    await page.evaluate(() =>
      Object.keys(localStorage).some((key) => localStorage.getItem(key) === '{"future":"preserve"}')
    )
  );
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});

test(
  'native Fountain browser creation preserves exact destinations, lost replies, cancellation and original retry',
  {skip: !process.env.SODA_FOUNTAIN_BROWSER_API || process.env.SODA_FOUNTAIN_ISSUE_CREATE_BROWSER !== '1'},
  async (t) => {
    const page = await fixture(t),
      api = process.env.SODA_FOUNTAIN_BROWSER_API!,
      native = process.env.SODA_FOUNTAIN_BROWSER_REPOSITORY!;
    await page.evaluate(
      ({api, native}) => {
        window.drawerFixture.forwardAPI(api);
        window.drawerFixture.mountRepository(native, '/user1/human-comment-primary/feed/work/issues/new', '', true);
      },
      {api, native}
    );
    const form = page.locator('soda-factory-issue-create'),
      title = form.getByRole('textbox', {name: 'Issue title', exact: true}),
      body = form.getByRole('textbox', {name: 'Issue Markdown', exact: true});
    const idle = () => form.locator('section[aria-busy=false]').waitFor();
    const newIssue = async () => {
      await page.locator('nav[aria-label="Issue navigation"]').getByRole('link', {name: 'Issues', exact: true}).click();
      await page
        .locator('nav[aria-label="Issue navigation"]')
        .getByRole('link', {name: 'New issue', exact: true})
        .click();
      await idle();
      if (await form.getByRole('button', {name: 'Write another issue', exact: true}).count())
        await form.getByRole('button', {name: 'Write another issue', exact: true}).click();
    };
    const submit = async (t: string, b: string) => {
      await idle();
      await title.fill(t);
      await body.fill(b);
      await form.getByRole('button', {name: 'Create issue', exact: true}).click();
    };
    const detail = page.locator('soda-factory-issue');
    await submit('Native browser title 完整', 'Native browser issue 完整.\n');
    await detail.getByRole('heading', {name: 'Native browser title 完整', exact: true}).waitFor();
    await detail.getByText('Native browser issue 完整.', {exact: true}).waitFor();
    assert.equal(new URL(page.url()).pathname, '/user1/human-comment-primary/feed/work/issues/4');
    await newIssue();
    await submit('Native browser lost reply', 'HTTP create browser reply dropped.');
    await form.getByText(/No final outcome was confirmed/).waitFor();
    const postCount = () =>
      page.evaluate(
        () =>
          window.drawerFixture.calls.filter(
            (c) => c.method === 'POST' && /\/collaboration\/repositories\/\d+\/issues$/.test(c.url)
          ).length
      );
    // The server and its real journal survive a new browser JS realm. Restore
    // performs only lookup and navigates to the original committed creation.
    const fixtureOrigin = new URL(page.url()).origin;
    await page.goto(fixtureOrigin);
    await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
    await page.evaluate(
      async ({api, native}) => {
        window.drawerFixture = window.createDrawerFixture({});
        await window.drawerFixture.api.ready;
        window.drawerFixture.forwardAPI(api);
        window.drawerFixture.mountRepository(native, '/user1/human-comment-primary/feed/work/issues/new', '', true);
      },
      {api, native}
    );
    await detail.getByRole('heading', {name: 'Native browser lost reply', exact: true}).waitFor();
    assert.equal(await postCount(), 0);
    assert.equal(new URL(page.url()).pathname, '/user1/human-comment-primary/feed/work/issues/5');
    await newIssue();
    const hold = async () =>
      assert.equal((await fetch(new URL('fixture/hold-next', api), {method: 'POST'})).status, 204);
    await hold();
    await submit('Native browser cancelled title', 'Native browser issue cancelled.');
    await form.getByText(/No final outcome was confirmed/).waitFor();
    await form.getByRole('button', {name: 'Cancel submission', exact: true}).click();
    await form.getByText('Submission cancelled. Your writing was preserved.', {exact: true}).waitFor();
    assert.equal(await title.inputValue(), 'Native browser cancelled title');
    assert.equal(await body.inputValue(), 'Native browser issue cancelled.');
    await hold();
    await submit('Native browser retry title', 'Native browser issue retry original.');
    await form.getByText(/No final outcome was confirmed/).waitFor();
    await form.getByRole('button', {name: 'Check issue status', exact: true}).click();
    await form.getByText(/submission is not confirmed/).waitFor();
    await form.getByRole('button', {name: 'Retry saved issue', exact: true}).click();
    await detail.getByRole('heading', {name: 'Native browser retry title', exact: true}).waitFor();
    assert.equal(new URL(page.url()).pathname, '/user1/human-comment-primary/feed/work/issues/6');
    const submissions = await page.evaluate(() =>
      window.drawerFixture.calls
        .filter((c) => c.method === 'POST' && /\/collaboration\/repositories\/\d+\/issues$/.test(c.url))
        .map((c) => JSON.parse(c.body!) as {command_id: string; title: string; body: string})
    );
    const retries = submissions.filter((c) => c.body === 'Native browser issue retry original.');
    assert.equal(retries.length, 2);
    assert.deepEqual(retries[0], retries[1]);
    const cancellations = await page.evaluate(() =>
      window.drawerFixture.calls
        .filter((c) => c.method === 'POST' && c.url.includes('/issue-creation/commands/') && c.url.endsWith('/cancel'))
        .map((c) => c.body)
    );
    assert.deepEqual(cancellations, ['{}']);
    await newIssue();
    const maximalTitle = 'V'.repeat(251) + '🌋',
      maximalBody = '<'.repeat(65532) + '🌋';
    await submit(maximalTitle, maximalBody);
    await detail.getByRole('heading', {name: maximalTitle, exact: true}).waitFor();
    assert.equal(new URL(page.url()).pathname, '/user1/human-comment-primary/feed/work/issues/7');
    await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
  }
);

test('request places isolate host drafts and resolve native picker rows before canonical request writes', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const model = window.drawerFixture.intentModel('11');
    const intent = 'd' + '1'.repeat(24),
      idea = 'd' + '2'.repeat(24);
    model('/api/factory/intents', 'POST', {
      id: intent,
      intent,
      owner: '11',
      repository: '0',
      objective: 'Track reviewed work',
      state: 'draft',
      max_issues: 1,
      ideas: [
        {
          id: idea,
          description: 'Track progress',
          consequence: 'The owner sees work',
          assumption: false,
          decision: 'accept',
          source: 'requested',
          provenance: 'human',
          actor: '11',
        },
      ],
      tasks: [
        {key: 'd' + '3'.repeat(24), objective: 'Show progress', outcomes: ['Progress is visible'], idea_ids: [idea]},
      ],
      effective_outcomes: [idea],
    });
    let host = 'a'.repeat(32);
    window.addEventListener('request-host-a', () => {
      host = 'a'.repeat(32);
    });
    window.addEventListener('request-host-b', () => {
      host = 'b'.repeat(32);
    });
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/identity')) return Response.json({host, actor: '11', native_actor: '1'});
      if (call.url.endsWith('/api/factory/native-repositories/resolve')) {
        const native = JSON.parse(call.body!).repository;
        return Response.json({
          host,
          actor: '11',
          repository: native === '7' ? '9007199254740993' : '43',
          native_repository: native,
          name: native === '7' ? 'alice/demo' : 'alice/other',
        });
      }
      if (call.url.endsWith('/api/factory/repositories/43/identity'))
        return Response.json({host, actor: '11', repository: '43', native_repository: '9', name: 'alice/other'});
      if (call.url.endsWith(`/api/factory/intents/${intent}/review`))
        return Response.json({intent, head_revision: intent, status: 'missing'});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [
            {id: '7', owner: 'alice', name: 'demo', can_create: true, project: null},
            {id: '9', owner: 'alice', name: 'other', can_create: true, project: null},
          ],
        });
      return model(call.url, call.method, call.body ? JSON.parse(call.body) : null) ?? null;
    });
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/' + intent);
  });
  const composer = page.locator('.soda-repository-page soda-intent');
  const objective = composer.getByRole('textbox', {name: 'Desired outcome', exact: true});
  await objective.waitFor();
  assert.equal(await composer.evaluate((el) => (el as HTMLElement & {actor: string}).actor), '11');
  await objective.fill('Private A draft');
  const key = `soda-intent-draft:v1:${encodeURIComponent('fountain:' + 'a'.repeat(32))}:11:d${'1'.repeat(24)}`;
  await page.waitForFunction((key) => localStorage.getItem(key)?.includes('Private A draft'), key);
  await page.evaluate(() => {
    window.dispatchEvent(new Event('request-host-b'));
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24));
  });
  await objective.waitFor();
  assert.equal(await objective.inputValue(), 'Track reviewed work');
  await objective.fill('Private B draft');
  await page.evaluate(() => {
    window.dispatchEvent(new Event('request-host-a'));
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24));
  });
  await objective.waitFor();
  assert.equal(await objective.inputValue(), 'Private A draft');
  // Finish draft-isolation checks, then choose a repository for the unchanged
  // saved review; unsaved text correctly refuses acceptance until reviewed.
  await page.evaluate((key) => {
    localStorage.removeItem(key);
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24));
  }, key);
  await objective.waitFor();
  assert.equal(await objective.inputValue(), 'Track reviewed work');
  // The same scoped recovery works across real component remounts when the
  // browser denies storage. Host B cannot consume host A's in-document draft.
  const storage = await page.evaluateHandle(() => window.localStorage);
  await page.evaluate(() => {
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      get() {
        throw new DOMException('Declared fixture storage denial', 'SecurityError');
      },
    });
  });
  await objective.fill('Private A draft with storage denied');
  await page.evaluate(() => {
    window.dispatchEvent(new Event('request-host-b'));
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24));
  });
  await objective.waitFor();
  assert.equal(await objective.inputValue(), 'Track reviewed work');
  await objective.fill('Private B draft with storage denied');
  await page.evaluate(() => {
    window.dispatchEvent(new Event('request-host-a'));
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24));
  });
  await objective.waitFor();
  assert.equal(await objective.inputValue(), 'Private A draft with storage denied');
  await page.evaluate(() => {
    window.dispatchEvent(new Event('request-host-b'));
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24));
  });
  await objective.waitFor();
  assert.equal(await objective.inputValue(), 'Private B draft with storage denied');
  await page.evaluate(() => {
    window.dispatchEvent(new Event('request-host-a'));
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24));
  });
  await objective.waitFor();
  await objective.fill('Track reviewed work');
  // Reload the unchanged saved review before continuing its acceptance flow.
  await page.evaluate(() => window.drawerFixture.mountRepository('7', '/alice/demo/feed/requests/d' + '1'.repeat(24)));
  await objective.waitFor();
  assert.equal(await objective.inputValue(), 'Track reviewed work');
  await composer.evaluate((el) =>
    el.dispatchEvent(new CustomEvent('intent-reviewed', {detail: {authorizedIdeas: 1, tasks: 1}, bubbles: true}))
  );
  const picker = page.locator('.soda-request-picker');
  await picker.getByRole('radio', {name: /alice\/other/}).check();
  await picker.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.waitForFunction(() => !(document.querySelector('soda-intent') as unknown as {busy: boolean})?.busy);
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter((call) => call.url.endsWith('/api/factory/intents') && call.method === 'POST')
          .length
    ),
    0,
    'unreadable command storage refuses dispatch'
  );
  await page.evaluate(
    (storage) => Object.defineProperty(window, 'localStorage', {configurable: true, value: storage}),
    storage
  );
  await picker.getByRole('button', {name: 'Continue', exact: true}).click();
  await page.waitForFunction(() =>
    window.drawerFixture.calls.some((call) => call.url.endsWith('/api/factory/intents') && call.method === 'POST')
  );
  const writes = await page.evaluate(() =>
    window.drawerFixture.calls.filter((call) => call.url.endsWith('/api/factory/intents') && call.method === 'POST')
  );
  assert(writes.length > 0);
  for (const write of writes) {
    const body = JSON.parse(write.body!);
    assert.equal(body.repository, '43');
    assert(body.ideas.every((idea: {actor: string}) => idea.actor === '11'));
  }
  assert.equal(
    await page.evaluate(
      () => window.drawerFixture.calls.filter((call) => call.url.includes('/api/environments?repository_id=43')).length
    ),
    0
  );
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});

test('request places resolve another canonical repository only through visible host metadata and refuse another owner', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const intent = 'd' + '4'.repeat(24);
    let visible = true,
      owner = '11';
    window.addEventListener('hide-request-repository', () => {
      visible = false;
    });
    window.addEventListener('foreign-request-owner', () => {
      owner = '1';
    });
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/native-repositories/resolve'))
        return Response.json({
          host: 'a'.repeat(32),
          actor: '11',
          repository: '41',
          native_repository: '7',
          name: 'alice/demo',
        });
      if (call.url.endsWith('/api/factory/intents/' + intent))
        return Response.json({
          id: intent,
          intent,
          owner,
          repository: '43',
          objective: 'Another project',
          state: 'draft',
          max_issues: 1,
          ideas: [],
          tasks: [],
        });
      if (call.url.endsWith('/api/factory/repositories/43/identity'))
        return visible
          ? Response.json({
              host: 'a'.repeat(32),
              actor: '11',
              repository: '43',
              native_repository: '9',
              name: 'alice/other',
            })
          : Response.json({}, {status: 404});
      return null;
    });
    window.drawerFixture.mountRepository('7', '/prefix/alice/demo/feed/requests/' + intent, '/prefix');
  });
  const root = page.locator('.soda-repository-page');
  await root.getByRole('link', {name: 'alice/other', exact: true}).waitFor();
  assert.equal(
    await root.getByRole('link', {name: 'alice/other', exact: true}).getAttribute('href'),
    '/prefix/alice/other/feed/requests/d' + '4'.repeat(24)
  );
  assert.equal(await root.locator('soda-intent').count(), 0);
  await page.evaluate(() => {
    window.dispatchEvent(new Event('hide-request-repository'));
    window.drawerFixture.mountRepository('7', '/prefix/alice/demo/feed/requests/d' + '4'.repeat(24), '/prefix');
  });
  await root.getByRole('region', {name: 'Request in another repository'}).waitFor();
  assert.equal(await root.getByRole('link', {name: 'alice/other', exact: true}).count(), 0);
  await page.evaluate(() => {
    window.dispatchEvent(new Event('foreign-request-owner'));
    window.drawerFixture.mountRepository('7', '/prefix/alice/demo/feed/requests/d' + '4'.repeat(24), '/prefix');
  });
  await root.getByText('Could not load this request. Try again.', {exact: true}).waitFor();
  assert.equal(await root.locator('soda-intent').count(), 0);
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter((call) => /\/api\/environments\?repository_id=(41|43)(?:&|$)/.test(call.url))
          .length
    ),
    0
  );
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});

test('repository places keep native inventory and original links while Knowledge, ordinary issues and App retries use Factory identities', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const repository = '9007199254740993';
    const run = 'b'.repeat(32);
    let nativeRunScope = true;
    window.addEventListener(
      'correct-place-run-scope',
      () => {
        nativeRunScope = false;
      },
      {once: true}
    );
    const observation = {state: 'available', revision: 'native-issue', observed_at: '2026-11-03T12:00:00Z'};
    const issue = {
      ref: {kind: 'issue', repository, external_id: 'native-42', number: '42'},
      observation,
      title: 'Issue #42',
      state: 'open',
      author: null,
      assignees: [],
      labels: [],
      body: 'Ordinary issue content',
      created_at: null,
      updated_at: null,
      closed_at: null,
    };
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/repositories/' + repository + '/identity'))
        return Response.json({
          host: 'a'.repeat(32),
          actor: '11',
          repository,
          native_repository: '7',
          name: 'alice/demo',
          default_branch: 'main',
        });
      if (call.url.includes('/api/factory/collaboration/')) {
        if (call.url.includes('/comments?')) return Response.json({observation, items: [], next: '', complete: true});
        if (call.url.includes('/issues?'))
          return Response.json({observation, items: [issue], next: '', complete: true});
        return Response.json(issue);
      }
      if (call.url.endsWith('/api/factory/native-repositories/resolve'))
        return Response.json({
          host: 'a'.repeat(32),
          actor: '11',
          repository,
          native_repository: '7',
          name: 'alice/demo',
        });
      const libraryItem = {
        id: 'a'.repeat(32),
        repository,
        revision: 1,
        title: 'Recorded knowledge',
        kind: 'knowledge',
        body: 'Reviewed observation',
        source_post: 'c'.repeat(32),
      };
      if (call.url.includes('/api/factory/library?'))
        return Response.json({items: [libraryItem], next_cursor: '', complete: true});
      if (call.url.includes('/api/factory/library/item?')) return Response.json({item: libraryItem, history: []});
      if (call.url.endsWith('/api/spaces'))
        return Response.json({
          actor: {id: '1', login: 'alice'},
          complete: true,
          items: [
            {native: '7', environment: 'p' + '1'.repeat(24), issue: '42', run},
            {native: repository, environment: 'p' + '2'.repeat(24), issue: '99', run: 'c'.repeat(32)},
          ].map((item) => ({
            environment: {
              id: item.environment,
              repository_id: item.native,
              owner_id: '1',
              name: 'demo',
              repository: 'alice/demo',
              provisioned: true,
            },
            login: 'alice',
            environment_administrator: true,
            execution_allowed: true,
            authority_unavailable: false,
            native_unavailable: false,
            observed: {id: item.environment, running: true},
            terminals: [],
            factory_runs: [
              {id: 'd'.repeat(32), issue: item.issue, role: 'soda-coder', reconciled: true},
              {id: item.run, role: 'soda-reviewer', reconciled: true},
            ],
          })),
        });
      if (call.url.endsWith('/api/applications'))
        return Response.json({applications: [], launchable: [], complete: true, can_setup: false});
      if (call.url.endsWith(`/api/factory/runs/${run}`))
        return Response.json({
          run: {id: run, reconciled: true, role: 'soda-reviewer', outcome: 'succeeded'},
          view: {repository: nativeRunScope ? '7' : repository, attempt: 'f'.repeat(32)},
          verification_retry: {available: true},
        });
      if (call.url.endsWith(`/api/factory/runs/${run}/actions`)) {
        const body = JSON.parse(call.body!);
        return Response.json({
          command_id: body.command_id,
          prior_run: run,
          queued: true,
          reason: 'Verification queued',
        });
      }
      if (call.url.includes('/factory/issues/')) return Response.json({error: {code: 'not_found'}}, {status: 404});
      return null;
    });
    window.drawerFixture.mountRepository('7', '/soda/alice/demo/feed/knowledge', '/soda');
  });
  const root = page.locator('.soda-repository-page');
  await root.locator('soda-factory-library').waitFor();
  await page.waitForFunction(() =>
    window.drawerFixture.calls.some((call) => call.url.includes('/api/factory/library?'))
  );
  assert.equal(
    await root
      .locator('soda-factory-library')
      .evaluate((el) => (el as HTMLElement & {context: {repository: string}}).context.repository),
    '9007199254740993'
  );
  await root.getByRole('button', {name: 'Recorded knowledge', exact: true}).click();
  await root.getByRole('heading', {name: 'Recorded knowledge', exact: true}).waitFor();
  await root.getByRole('button', {name: 'Open source post', exact: true}).click();
  assert.equal(
    await page.evaluate(() => document.body.dataset.navigated),
    '/soda/alice/demo/feed?post=' + 'c'.repeat(32)
  );
  await page.evaluate(() => {
    delete document.body.dataset.navigated;
  });
  await root.locator('soda-factory-library').evaluate((el) =>
    el.dispatchEvent(
      new CustomEvent('library-source-open', {
        detail: {repository: '7', id: 'a'.repeat(32)},
        bubbles: true,
      })
    )
  );
  assert.equal(await page.evaluate(() => document.body.dataset.navigated), undefined);
  await root.locator('soda-factory-library').evaluate((el) =>
    el.dispatchEvent(
      new CustomEvent('library-source-open', {
        detail: {repository: '9007199254740993', id: 'a'.repeat(32)},
        bubbles: true,
      })
    )
  );
  assert.equal(
    await page.evaluate(() => document.body.dataset.navigated),
    '/soda/alice/demo/feed?post=' + 'a'.repeat(32)
  );
  await page.evaluate(() =>
    window.drawerFixture.repositoryHandle?.update?.({
      mode: 'page',
      route: '/soda/alice/demo/feed/work/issues',
      params: {},
      visible: true,
    })
  );
  await root.getByRole('button', {name: /Issue #42/}).waitFor();
  assert.equal(await root.getByRole('button', {name: /Issue #99/}).count(), 0);
  await page.evaluate(() =>
    window.drawerFixture.repositoryHandle?.update?.({
      mode: 'page',
      route: '/soda/alice/demo/feed/work/issues/42',
      params: {},
      visible: true,
    })
  );
  await root.getByRole('link', {name: 'Open in Fountain', exact: true}).waitFor();
  assert.equal(
    await root.getByRole('link', {name: 'Open in Fountain', exact: true}).getAttribute('href'),
    '/soda/alice/demo/issues/42'
  );
  await page.waitForFunction(() => window.drawerFixture.calls.some((call) => call.url.includes('/factory/issues/42')));
  const issueRequests = await page.evaluate(() =>
    window.drawerFixture.calls.filter((call) => call.url.includes('/factory/issues/'))
  );
  assert(issueRequests.every((call) => call.url.includes('/repositories/9007199254740993/factory/')));
  await page.evaluate(() =>
    window.drawerFixture.repositoryHandle?.update?.({
      mode: 'page',
      route: '/soda/alice/demo/feed/app',
      params: {},
      visible: true,
    })
  );
  const verification = root.locator('soda-factory-verification-retry');
  await verification.getByText(/Could not read current verification or repair progress/).waitFor();
  assert.equal(
    await verification.getByRole('button', {name: 'Retry application verification', exact: true}).count(),
    0
  );
  await page.evaluate(() => window.dispatchEvent(new Event('correct-place-run-scope')));
  await verification.getByRole('button', {name: 'Refresh verification', exact: true}).click();
  await root
    .locator('soda-factory-verification-retry')
    .getByRole('button', {name: 'Retry application verification', exact: true})
    .waitFor();
  assert.equal(
    await root.locator('soda-applications').evaluate((el) => (el as HTMLElement & {repository: string}).repository),
    '7'
  );
  assert.equal(
    await page.evaluate(
      () => window.drawerFixture.calls.filter((call) => call.url.endsWith('/api/factory/runs/' + 'c'.repeat(32))).length
    ),
    0
  );
  await root
    .locator('soda-factory-verification-retry')
    .getByRole('button', {name: 'Retry application verification', exact: true})
    .click();
  await page.waitForFunction(() =>
    window.drawerFixture.calls.some((call) => call.url.endsWith('/api/factory/runs/' + 'b'.repeat(32) + '/actions'))
  );
  const calls = await page.evaluate(() => window.drawerFixture.calls);
  assert(
    calls
      .filter((call) => call.url.endsWith('/api/factory/native-repositories/resolve'))
      .every((call) => JSON.parse(call.body!).repository === '7')
  );
  assert(
    calls
      .filter((call) => call.url.includes('/api/factory/library?'))
      .every((call) => new URL(call.url, 'http://fixture').searchParams.get('repository') === '9007199254740993')
  );
  assert(!calls.some((call) => call.url.includes('/repositories/7/factory')));
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});

test('repository places cancel pending discovery on navigation or hiding and refuse a mismatched native repository', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    let serial = 0;
    const gates: Array<() => void> = [];
    window.addEventListener('release-place-discovery', () => gates.splice(0).forEach((release) => release()));
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/native-repositories/resolve')) {
        const ordinal = ++serial;
        if (ordinal <= 2) await new Promise<void>((resolve) => gates.push(resolve));
        return Response.json({
          host: 'a'.repeat(32),
          actor: '11',
          repository: '41',
          native_repository: ordinal >= 5 ? '9' : '7',
          name: 'alice/demo',
        });
      }
      if (call.url.includes('/api/factory/library?'))
        return Response.json({items: [], next_cursor: '', complete: true});
      return null;
    });
    window.drawerFixture.mountRepository('7', '/alice/demo/feed/work/issues/42');
  });
  await page.waitForFunction(
    () => window.drawerFixture.calls.filter((call) => call.url.endsWith('/native-repositories/resolve')).length === 2
  );
  assert.equal(await page.locator('soda-factory-work-journey').count(), 0);
  await page.evaluate(() => {
    window.drawerFixture.repositoryHandle?.update?.({
      mode: 'page',
      route: '/alice/demo/feed/knowledge',
      params: {},
      visible: false,
    });
  });
  assert.equal(await page.locator('soda-factory-library').count(), 0);
  await page.evaluate(() => {
    window.dispatchEvent(new Event('release-place-discovery'));
    window.drawerFixture.repositoryHandle?.resume?.({sessionGeneration: 'fixture-generation'});
  });
  await page.locator('.soda-repository-page soda-factory-library').waitFor();
  await page.waitForFunction(() =>
    window.drawerFixture.calls.some((call) => call.url.endsWith('/repositories/41/factory/access'))
  );
  assert.equal(await page.locator('soda-factory-work-journey').count(), 0);
  const before = await page.evaluate(() => window.drawerFixture.calls.length);
  await page.evaluate(() =>
    window.drawerFixture.repositoryHandle?.update?.({
      mode: 'page',
      route: '/alice/demo/feed/app',
      params: {},
      visible: true,
    })
  );
  await page
    .locator('.soda-repository-page')
    .getByText('Repository information is unavailable.', {exact: true})
    .waitFor();
  const after = await page.evaluate((start) => window.drawerFixture.calls.slice(start), before);
  assert(after.every((call) => call.url.endsWith('/native-repositories/resolve')));
  assert.equal(await page.locator('soda-applications').count(), 0);
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});

test('portable host renders the production feed without Fountain CSS and isolates account drafts', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      get() {
        throw new DOMException('Declared fixture storage denial', 'SecurityError');
      },
    });
    document.querySelectorAll('link[rel=stylesheet]').forEach((link) => link.remove());
    const css = document.createElement('link');
    css.rel = 'stylesheet';
    css.href = '/factory/styles/base.css';
    document.head.append(css);
    const root = document.createElement('main');
    root.className = 'soda-factory-root';
    root.dataset.factoryScheme = 'dark';
    document.body.replaceChildren(root);
    const feed = document.createElement('soda-factory-feed');
    root.append(feed);
    let revision = 1;
    let selected = ['9007199254740993', '9007199254740992', '9223372036854775807'];
    const context = {
      repository: '7',
      transport: {
        generation: 'portable-session',
        assetBase: '/assets/',
        async request(path: string, init?: RequestInit) {
          if (path === 'factory/feed/subscription' && init?.method === 'PUT') {
            const body = JSON.parse(String(init.body));
            document.body.dataset.exactSubscription = JSON.stringify(body);
            selected = body.repositories;
            revision++;
            return Response.json({subscriber: '1', revision});
          }
          if (path === 'factory/feed/subscription')
            return Response.json({subscriber: '1', repositories: selected, revision});
          if (path.startsWith('factory/feed?'))
            return Response.json({
              entries: [
                {
                  kind: 'record',
                  id: 'a'.repeat(32),
                  repository: '7',
                  seq: 1,
                  issue: 42,
                  author: 'Builder',
                  actor: 'builder',
                  status: 'implement:running',
                  payload: {
                    communication: {
                      kind: 'partial',
                      title: 'Portable work',
                      summary: 'The host supplies only product capabilities.',
                    },
                  },
                },
              ],
              truncated: false,
            });
          if (/^repositories\/(7|9007199254740992|9007199254740993|9223372036854775807)\/factory$/.test(path))
            return Response.json({});
          document.body.dataset.unexpectedFeedRequest = path;
          throw Error('Unexpected portable feed request');
        },
      },
      host: {
        draftScope: 'desktop:github:alice',
        async repositories() {
          return {
            refs: new Map([
              ['7', {path: '/desktop/repositories/7', label: 'team/project'}],
              ['9007199254740993', {path: '/desktop/main', label: 'team/main'}],
              ['9007199254740992', {path: '/desktop/adjacent', label: 'team/adjacent'}],
              ['9223372036854775807', {path: '/desktop/max', label: 'team/max'}],
            ]),
            listedAll: true,
          };
        },
        links() {
          return {
            repository: '/desktop/repositories/7',
            feed: () => '/desktop/feed',
            request: (id: string) => `/desktop/requests/${id}`,
            issue: (id: string | number) => `/desktop/issues/${id}`,
            pull: (id: string | number) => `/desktop/pulls/${id}`,
            factoryStatus: () => '/desktop/work',
          };
        },
        navigate(path: string) {
          document.body.dataset.feedNavigation = path;
        },
      },
    };
    feed.context = context;
    Object.assign(feed, {
      enterGlobal() {
        feed.context = {...context, repository: '', global: true};
      },
      switchActor(actor: string) {
        feed.context = {...context, host: {...context.host, draftScope: `desktop:github:${actor}`}};
      },
    });
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Portable work', exact: true}).waitFor();
  assert.equal(
    await feed.getByRole('link', {name: 'Open issue #42', exact: true}).getAttribute('href'),
    '/desktop/issues/42'
  );
  await feed.getByRole('link', {name: 'Open issue #42', exact: true}).click();
  assert.equal(await page.evaluate(() => document.body.dataset.feedNavigation), '/desktop/issues/42');
  const draft = feed.locator('textarea[data-feed-draft="project:7"]');
  await draft.fill('Alice keeps this draft.');
  await feed.evaluate((node) => (node as HTMLElement & {switchActor(actor: string): void}).switchActor('bob'));
  await feed.getByRole('heading', {name: 'Portable work', exact: true}).waitFor();
  assert.equal(await draft.inputValue(), '');
  await draft.fill('Bob keeps a different draft.');
  await feed.evaluate((node) => (node as HTMLElement & {switchActor(actor: string): void}).switchActor('alice'));
  await feed.getByRole('heading', {name: 'Portable work', exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'Alice keeps this draft.');
  await page.waitForFunction(
    () => getComputedStyle(document.querySelector('.soda-factory-root')!).backgroundColor === 'rgb(16, 16, 16)'
  );
  await feed.evaluate((node) => (node as HTMLElement & {enterGlobal(): void}).enterGlobal());
  await feed.getByRole('heading', {name: 'Portable work', exact: true}).waitFor();
  await feed.locator('summary').filter({hasText: 'Choose repositories'}).click();
  assert(await feed.getByRole('checkbox', {name: 'team/main', exact: true}).isChecked());
  assert(await feed.getByRole('checkbox', {name: 'team/adjacent', exact: true}).isChecked());
  await feed.getByRole('checkbox', {name: 'team/adjacent', exact: true}).uncheck();
  await feed.getByRole('button', {name: 'Save feed subscription', exact: true}).click();
  await page.waitForFunction(() => document.body.dataset.exactSubscription);
  assert.deepEqual(JSON.parse((await page.evaluate(() => document.body.dataset.exactSubscription))!), {
    repositories: ['9007199254740993', '9223372036854775807'],
    expected_revision: 1,
  });
  assert.equal(await page.evaluate(() => document.body.dataset.unexpectedFeedRequest), undefined);
  await standaloneLibrary();
  await standaloneFactoryApp();
});

test('Fountain resolves canonical feed scopes, native links and account drafts before model requests', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    let actor = '11',
      host = 'a'.repeat(32),
      mismatch = false,
      unsafe = false;
    const calls: {path: string; body?: string}[] = [];
    const transport = {
      generation: 'canonical-session',
      assetBase: '/assets/',
      async request(path: string, init?: RequestInit) {
        calls.push({path, ...(typeof init?.body === 'string' ? {body: init.body} : {})});
        if (path === 'factory/identity') return Response.json({host, actor, native_actor: actor === '11' ? '1' : '2'});
        if (path === 'factory/native-repositories/resolve') {
          const native = JSON.parse(String(init?.body)).repository;
          return Response.json({
            host,
            actor: mismatch ? '99' : actor,
            repository: unsafe ? '9223372036854775808' : native === '7' ? '9007199254740993' : '43',
            native_repository: native,
            name: native === '7' ? 'team/main' : 'team/tools',
          });
        }
        if (path === 'repositories?q=')
          return Response.json({
            items: [
              {id: '7', owner: 'team', name: 'main', can_create: false, project: null},
              {id: '9', owner: 'team', name: 'tools', can_create: false, project: null},
            ],
            next_cursor: '',
          });
        if (path === 'factory/feed/subscription')
          return Response.json({subscriber: actor, repositories: ['9007199254740993'], revision: 1});
        if (path.startsWith('factory/intents?')) return Response.json({intents: []});
        if (path.startsWith('factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: 'a'.repeat(32),
                repository: new URLSearchParams(path.split('?')[1]).get('repositories') || '9007199254740993',
                seq: 1,
                issue: 42,
                author: 'Builder',
                actor: 'builder',
                status: 'implement:running',
                payload: {communication: {kind: 'partial', title: 'Canonical work', summary: 'Mapped repository.'}},
              },
            ],
            truncated: false,
          });
        if (path === 'repositories/9007199254740993/factory' || path === 'repositories/43/factory')
          return Response.json({});
        if (path === 'applications') return Response.json({}, {status: 503});
        document.body.dataset.unexpectedCanonicalRequest = path;
        throw Error('Unexpected canonical request');
      },
    };
    const feed = document.createElement('soda-factory-feed');
    feed.dataset.canonicalFeed = 'true';
    document.body.append(feed);
    const options = {
      repository: '7',
      repositories: new Map([
        ['7', 'team/main'],
        ['9', 'team/tools'],
      ]),
    };
    const resolved = await window.drawerFixture.feedContext(transport, options);
    feed.context = resolved;
    const catalogue = await resolved.host.repositories(['9007199254740993', '43'], new AbortController().signal);
    document.body.dataset.canonicalCatalogue = JSON.stringify([...catalogue.refs.keys()]);
    Object.assign(feed, {
      calls,
      async switchIdentity(value: string, namespace = 'a'.repeat(32)) {
        actor = value;
        host = namespace;
        feed.context = await window.drawerFixture.feedContext({...transport, generation: 'reconnected'}, options);
      },
      async refusesInvalid(kind: string) {
        mismatch = kind === 'account';
        unsafe = kind === 'range';
        try {
          await window.drawerFixture.feedContext(transport, options);
          return false;
        } catch {
          return true;
        } finally {
          mismatch = unsafe = false;
        }
      },
      async switchWorkDuringResolution() {
        if (!customElements.get('soda-project-work')) throw Error('Work UI module is unavailable');
        let release: ((response: Response) => void) | undefined;
        let started: (() => void) | undefined;
        const waiting = new Promise<void>((resolve) => {
          started = resolve;
        });
        let held = false;
        const workTransport = {
          ...transport,
          async request(path: string, init?: RequestInit) {
            if (
              path === 'factory/native-repositories/resolve' &&
              JSON.parse(String(init?.body)).repository === '7' &&
              !held
            ) {
              held = true;
              started?.();
              return new Promise<Response>((resolve) => {
                release = resolve;
              });
            }
            return transport.request(path, init);
          },
        };
        const work = document.createElement('soda-project-work') as HTMLElement & {
          active: boolean;
          surface: string;
          context: unknown;
        };
        work.active = true;
        work.surface = 'work';
        document.body.append(work);
        const context = {
          transport: workTransport,
          mode: 'project',
          repository: {id: '7', name: 'team/main'},
          repositoryNames: new Map([['7', 'team/main']]),
        };
        work.context = context;
        await waiting;
        const next = {
          ...context,
          repository: {id: '9', name: 'team/tools'},
          repositoryNames: new Map([['9', 'team/tools']]),
        };
        work.context = next;
        work.context = {...next};
        Object.assign(feed, {
          releaseWorkResolution: async () =>
            release?.(
              await transport.request('factory/native-repositories/resolve', {
                method: 'POST',
                body: JSON.stringify({repository: '7'}),
              })
            ),
        });
      },
    });
  });
  const feed = page.locator('soda-factory-feed[data-canonical-feed="true"]');
  await feed.getByRole('heading', {name: 'Canonical work', exact: true}).waitFor();
  assert.equal(
    await feed.getByRole('link', {name: 'Open issue #42', exact: true}).getAttribute('href'),
    '/team/main/feed/work/issues/42'
  );
  assert.equal(await page.evaluate(() => document.body.dataset.canonicalCatalogue), '["9007199254740993","43"]');
  const draft = feed.locator('textarea[data-feed-draft="project:9007199254740993"]');
  await draft.fill('Alice canonical draft');
  await feed.evaluate((node) =>
    (node as HTMLElement & {switchIdentity(actor: string, namespace?: string): Promise<void>}).switchIdentity('12')
  );
  await feed.getByRole('heading', {name: 'Canonical work', exact: true}).waitFor();
  assert.equal(await draft.inputValue(), '');
  await draft.fill('Bob canonical draft');
  await feed.evaluate((node) =>
    (node as HTMLElement & {switchIdentity(actor: string, namespace?: string): Promise<void>}).switchIdentity('11')
  );
  await feed.getByRole('heading', {name: 'Canonical work', exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'Alice canonical draft');
  await feed.evaluate((node) =>
    (node as HTMLElement & {switchIdentity(actor: string, namespace?: string): Promise<void>}).switchIdentity(
      '11',
      'b'.repeat(32)
    )
  );
  await feed.getByRole('heading', {name: 'Canonical work', exact: true}).waitFor();
  assert.equal(await draft.inputValue(), '');
  assert(
    await feed.evaluate((node) =>
      (node as HTMLElement & {refusesInvalid(kind: string): Promise<boolean>}).refusesInvalid('account')
    )
  );
  assert(
    await feed.evaluate((node) =>
      (node as HTMLElement & {refusesInvalid(kind: string): Promise<boolean>}).refusesInvalid('range')
    )
  );
  await feed.evaluate((node) =>
    (node as HTMLElement & {switchWorkDuringResolution(): Promise<void>}).switchWorkDuringResolution()
  );
  await page.waitForFunction(() => {
    const feed = document.querySelector<HTMLElement & {calls: {path: string}[]}>('[data-canonical-feed]');
    return feed?.calls.some(
      (call) =>
        call.path.startsWith('factory/intents?') &&
        new URLSearchParams(call.path.split('?')[1]).get('repository') === '43'
    );
  });
  await feed.evaluate((node) =>
    (node as HTMLElement & {releaseWorkResolution(): Promise<void>}).releaseWorkResolution()
  );
  const calls = await feed.evaluate((node) => (node as HTMLElement & {calls: {path: string; body?: string}[]}).calls);
  assert(
    calls
      .filter((call) => call.path.startsWith('factory/intents?'))
      .every((call) => new URLSearchParams(call.path.split('?')[1]).get('repository') === '43')
  );
  assert(
    calls.some(
      (call) =>
        call.path.startsWith('factory/feed?') &&
        new URLSearchParams(call.path.split('?')[1]).get('repositories') === '9007199254740993'
    )
  );
  assert(calls.some((call) => call.path === 'repositories/9007199254740993/factory'));
  assert(!calls.some((call) => call.path === 'repositories/7/factory'));
  assert.equal(await page.evaluate(() => document.body.dataset.unexpectedCanonicalRequest), undefined);
});

test('global feed pages the Factory catalogue without Spaces and exposes incomplete discovery until Refresh recovers', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const host = 'a'.repeat(32),
      actor = '11',
      first = '9007199254740993',
      last = '9223372036854775807';
    let mode = 'complete';
    const calls: {path: string; method: string}[] = [];
    const transport = {
      generation: 'catalogue-session',
      assetBase: '/assets/',
      async request(path: string, init?: RequestInit) {
        calls.push({path, method: init?.method ?? 'GET'});
        if (path === 'factory/identity') return Response.json({host, actor, native_actor: '1'});
        if (path === 'factory/feed/subscription')
          return Response.json({subscriber: actor, repositories: [], revision: 0});
        if (path.startsWith('factory/repositories')) {
          const after = new URLSearchParams(path.split('?')[1]).get('after');
          const name = mode === 'renamed' ? 'team/renamed' : 'team/main';
          if (!after)
            return Response.json({
              host,
              actor,
              items: mode === 'complete' || mode === 'renamed' ? [] : [{repository: first, name}],
              next_after: mode === 'complete' || mode === 'renamed' ? '7' : first,
              complete: false,
            });
          if (mode === 'outage') return Response.json({error: {code: 'catalogue_unavailable'}}, {status: 503});
          if (mode === 'partial') return Response.json({host, actor, items: [], next_after: '', complete: false});
          if (mode === 'repeat') return Response.json({host, actor, items: [], next_after: after, complete: false});
          if (mode === 'actor' || mode === 'host')
            return Response.json({
              host: mode === 'host' ? 'b'.repeat(32) : host,
              actor: mode === 'actor' ? '12' : actor,
              items: [{repository: last, name: 'private/other-account'}],
              next_after: '',
              complete: true,
            });
          if (mode === 'malformed')
            return Response.json({
              host,
              actor,
              items: [
                {repository: last, name: 'private/untrusted-page'},
                {repository: '9223372036854775808', name: 'private/unsafe-identity'},
              ],
              next_after: '',
              complete: true,
            });
          return Response.json({
            host,
            actor,
            items: [
              {repository: first, name},
              {repository: last, name: 'team/tools'},
            ],
            next_after: '',
            complete: true,
          });
        }
        if (path.startsWith('factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: 'a'.repeat(32),
                repository: first,
                seq: 1,
                issue: 42,
                author: 'Builder',
                actor: 'builder',
                status: 'implement:running',
                payload: {
                  communication: {kind: 'partial', title: 'Catalogue work', summary: 'Visible registered work.'},
                },
              },
            ],
            truncated: false,
          });
        if (path === 'applications') return Response.json({}, {status: 503});
        if (/^repositories\/(9007199254740993|9223372036854775807)\/factory$/.test(path)) return Response.json({});
        throw Error('Unexpected catalogue request: ' + path);
      },
    };
    const feed = document.createElement('soda-factory-feed');
    feed.dataset.factoryCatalogue = 'true';
    Object.assign(feed, {
      catalogueMode(value: string) {
        mode = value;
      },
      catalogueCalls: calls,
    });
    document.body.append(feed);
    feed.context = await window.drawerFixture.feedContext(transport, {repository: '', global: true});
  });
  const feed = page.locator('soda-factory-feed[data-factory-catalogue]');
  const refresh = feed.getByRole('button', {name: 'Refresh feed', exact: true});
  const incomplete = feed.getByText('Some repositories could not be loaded. Refresh to try again.', {exact: true});
  await feed.getByRole('heading', {name: 'Catalogue work', exact: true}).waitFor();
  await feed.locator('summary').filter({hasText: 'Choose repositories'}).click();
  const first = feed.getByRole('checkbox', {name: 'team/main', exact: true});
  const last = feed.getByRole('checkbox', {name: 'team/tools', exact: true});
  assert.equal(await first.getAttribute('value'), '9007199254740993');
  assert.equal(await last.getAttribute('value'), '9223372036854775807');
  assert(await first.isChecked());
  assert(await last.isChecked());
  assert.equal(await incomplete.count(), 0);
  assert.equal(
    await feed.getByRole('link', {name: 'Open issue #42', exact: true}).getAttribute('href'),
    '/team/main/feed/work/issues/42'
  );
  for (const mode of ['outage', 'partial', 'repeat', 'actor', 'host', 'malformed']) {
    const before = await feed.evaluate(
      (node) => (node as HTMLElement & {catalogueCalls: unknown[]}).catalogueCalls.length
    );
    await feed.evaluate(
      (node, value) => (node as HTMLElement & {catalogueMode(value: string): void}).catalogueMode(value),
      mode
    );
    await refresh.click();
    await incomplete.waitFor();
    await page.waitForFunction(
      () =>
        !document.querySelector<HTMLButtonElement>('[data-factory-catalogue] button[aria-label="Refresh feed"]')
          ?.disabled
    );
    assert.equal(await first.getAttribute('value'), '9007199254740993');
    assert.equal(await last.count(), 0);
    assert.equal(await feed.getByText(/private\//).count(), 0);
    const requests = await feed.evaluate(
      (node, start) =>
        (node as HTMLElement & {catalogueCalls: {path: string; method: string}[]}).catalogueCalls.slice(start),
      before
    );
    assert.equal(requests.filter((call) => call.path.startsWith('factory/repositories')).length, 2, mode);
    assert(requests.every((call) => call.method === 'GET'));
  }
  await feed.evaluate((node) => (node as HTMLElement & {catalogueMode(value: string): void}).catalogueMode('renamed'));
  await refresh.click();
  await feed.getByRole('checkbox', {name: 'team/renamed', exact: true}).waitFor();
  assert.equal(await incomplete.count(), 0);
  assert.equal(
    await feed.getByRole('link', {name: 'Open issue #42', exact: true}).getAttribute('href'),
    '/team/renamed/feed/work/issues/42'
  );
  const calls = await feed.evaluate(
    (node) => (node as HTMLElement & {catalogueCalls: {path: string; method: string}[]}).catalogueCalls
  );
  assert(calls.some((call) => call.path === 'factory/repositories?after=7'));
  assert(calls.some((call) => call.path === 'factory/repositories?after=9007199254740993'));
  assert(
    calls.every(
      (call) => call.method === 'GET' && !call.path.startsWith('spaces') && !/^repositories(?:\?|$)/.test(call.path)
    )
  );
});

test('combined feed refuses unsafe numeric subscription identities before requesting another repository', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const feed = document.createElement('soda-factory-feed');
    document.body.append(feed);
    feed.context = {
      repository: '',
      global: true,
      transport: {
        generation: 'unsafe-wire',
        assetBase: '/assets/',
        async request(path: string) {
          if (path === 'factory/feed/subscription')
            return new Response('{"subscriber":"11","revision":1,"repositories":[9007199254740993]}', {
              headers: {'Content-Type': 'application/json'},
            });
          document.body.dataset.unsafeRepositoryRequest = path;
          throw Error('Unsafe subscription reached a model request');
        },
      },
      host: {
        draftScope: 'unsafe-wire',
        async repositories() {
          return {refs: new Map(), listedAll: true};
        },
        links() {
          return undefined;
        },
        navigate() {},
      },
    };
  });
  await page.locator('soda-factory-feed [data-feed-message]').filter({hasText: /.+/}).waitFor();
  assert.equal(await page.evaluate(() => document.body.dataset.unsafeRepositoryRequest), undefined);
});

test('feed surfaces new issues, handoffs, outcomes and blockers while routine replies stay in threads', async (t) => {
  const page = await fixture(t);
  page.setDefaultTimeout(8000);
  await page.evaluate(() => {
    const id = (n: number) => n.toString(16).repeat(32);
    const entries = [
      {
        id: id(7),
        actor: 'builder',
        issue: 42,
        parent_id: id(2),
        status: 'implement:finished:succeeded',
        summary: 'Delivered the working task list.',
      },
      {
        id: id(6),
        actor: 'builder',
        issue: 42,
        parent_id: id(2),
        payload: {
          communication: {kind: 'blocked', title: 'Storage unavailable', summary: 'Waiting for storage access.'},
        },
      },
      {
        id: id(5),
        actor: 'builder',
        issue: 43,
        parent_id: id(3),
        status: 'implement:assigned',
        summary: 'Starting the next issue.',
      },
      {
        id: id(4),
        actor: 'inspector',
        issue: 42,
        parent_id: id(3),
        status: 'review:assigned',
        summary: 'Starting independent review.',
      },
      {
        id: id(3),
        actor: 'builder',
        issue: 42,
        parent_id: id(2),
        status: 'implement:running',
        summary: 'Routine progress on the task list.',
      },
      {
        id: id(2),
        actor: 'builder',
        issue: 42,
        parent_id: id(1),
        status: 'implement:assigned',
        summary: 'Starting the task list.',
      },
      {id: id(1), actor: 'architect', issue: 42, status: 'investigate:finished:succeeded', summary: 'Plan ready.'},
      // Parent lies outside the page: same-role progress still stays in its thread.
      {
        id: id(8),
        actor: 'builder',
        issue: 42,
        parent_id: id(9),
        parent_actor: 'builder',
        parent_author: 'Builder',
        parent_issue: 42,
        status: 'implement:running',
        summary: 'Off-page routine progress.',
      },
      // Human actor metadata can be empty; the first role handoff still surfaces.
      {
        id: id(10),
        actor: 'issue_maker',
        parent_id: id(11),
        parent_actor: '',
        parent_author: 'alice',
        status: 'issue_maker:running',
        summary: 'First Issue Maker handoff.',
      },
      {id: id(11), actor: '1', author: 'alice', summary: 'Please build a task list.'},
    ].map((entry, index) => ({
      kind: 'record',
      repository: '7',
      seq: 20 - index,
      author: entry.actor === 'builder' ? 'Builder' : entry.actor === 'architect' ? 'Architect' : 'Inspector',
      ...entry,
    }));
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: ['7'], revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({
          entries:
            document.body.dataset.humanOffPage === 'true' ? entries.filter((entry) => entry.id !== id(11)) : entries,
          truncated: false,
        });
      if (call.url.includes('/replies?')) {
        const parent = /\/posts\/([a-f0-9]{32})\/replies\?/.exec(call.url)?.[1];
        return Response.json({
          items: entries
            .filter((entry) => entry.parent_id === parent)
            .map((entry) => ({
              post_id: entry.id,
              repository: '7',
              author_name: entry.author,
              actor: entry.actor,
              issue_scope: entry.issue,
              parent_id: entry.parent_id,
              at: 1729000000,
              body: entry.summary ?? '',
              status: entry.status,
              communication: entry.payload?.communication,
            })),
          next_before: 0,
        });
      }
      return null;
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Starting the task list.', {exact: true}).waitFor();
  const ids = await feed.locator('article.soda-feed-post').evaluateAll((nodes) => nodes.map((node) => node.id));
  assert.deepEqual(
    ids,
    [7, 6, 5, 4, 2, 1, 10, 11].map((n) => 'feed-record-' + n.toString(16).repeat(32))
  );
  await page.evaluate(async () => {
    document.body.dataset.humanOffPage = 'true';
    await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
  });
  await feed.getByText('First Issue Maker handoff.', {exact: true}).waitFor();
  assert.equal(await feed.getByText('Please build a task list.', {exact: true}).count(), 0);
  assert.equal(await feed.getByText('Routine progress on the task list.', {exact: true}).count(), 0);
  assert.equal(await feed.getByText('Off-page routine progress.', {exact: true}).count(), 0);
  await feed.locator(`[data-feed-drawer-toggle="record:${'2'.repeat(32)}"]`).click();
  await feed
    .locator('#post-' + '3'.repeat(32))
    .getByText('Routine progress on the task list.', {exact: true})
    .waitFor();
  await feed
    .locator('#post-' + '3'.repeat(32))
    .locator('[data-feed-drawer-toggle]')
    .click();
  await feed
    .locator('#post-' + '4'.repeat(32))
    .getByText('Starting independent review.', {exact: true})
    .waitFor();
  await feedAccess(t);
});

test('Issue Maker and its receipts stay separate feed posts connected by parent links', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    document.body.dataset.makerState = 'waiting';
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: ['7'], revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [
            {
              id: '7',
              owner: 'soda-tester',
              name: 'demo',
              can_create: false,
              project: {id: 'p' + '1'.repeat(24), provisioned: true},
            },
          ],
          next_cursor: '',
        });
      if (call.url.endsWith('/api/factory/feed/posts/' + 'b'.repeat(32)))
        return Response.json({
          post_id: 'b'.repeat(32),
          repository: '7',
          actor: 'issue_maker',
          author_name: 'Issue Maker',
          at: 1729000000,
          status: `issue_maker:${document.body.dataset.makerState}`,
          communication: {
            kind: 'system',
            title: 'Making issues: add login and notifications',
            summary: `Issue Maker is ${document.body.dataset.makerState}.`,
          },
        });
      if (call.url.includes('/replies?'))
        return Response.json({
          items: [12, 13, 14].map((issue, index) => ({
            post_id: String(index + 1).repeat(32),
            repository: '7',
            actor: 'issue_maker',
            author_name: 'Issue Maker',
            at: 1729000000,
            issue_scope: issue,
            status: 'issue_maker:created',
            communication: {kind: 'system', title: `Requested change ${issue}`, summary: `Created issue #${issue}.`},
          })),
        });
      if (!call.url.includes('/api/factory/feed?')) return null;
      if (document.body.dataset.makerOffPage === 'true') return Response.json({entries: [], truncated: false});
      const parent = 'b'.repeat(32);
      return Response.json({
        entries: [
          ...[14, 13, 12].map((issue, index) => ({
            kind: 'post',
            id: String(index + 1).repeat(32),
            repository: '7',
            seq: 5 - index,
            actor: 'issue_maker',
            author: 'Issue Maker',
            issue,
            parent_id: parent,
            parent_author: 'Issue Maker',
            status: 'issue_maker:created',
            payload: {
              communication: {
                kind: 'system',
                title: `Requested change ${issue}`,
                summary: `Created issue #${issue}.`,
                parent_id: parent,
              },
            },
          })),
          {
            kind: 'post',
            id: parent,
            repository: '7',
            seq: 2,
            actor: 'issue_maker',
            author: 'Issue Maker',
            status: `issue_maker:${document.body.dataset.makerState}`,
            payload: {
              communication: {
                kind: 'system',
                title: 'Making issues: add login and notifications',
                summary:
                  document.body.dataset.makerState === 'waiting'
                    ? 'Issues exist. AI refinement is waiting for preparation.'
                    : 'Issue Maker run has started. Refining the recorded issues.',
              },
            },
          },
          {
            kind: 'post',
            id: 'c'.repeat(32),
            repository: '7',
            seq: 1,
            author: 'alice',
            post: {
              post_id: 'c'.repeat(32),
              repository: '7',
              revision: 0,
              author: '1',
              author_name: 'alice',
              body: 'Add login and notifications',
              at: 1729000000,
              intake: 'recorded',
              judgment: 'applied',
              route: 'new_issue',
              draft: 'applied',
              effect: 'verified',
              issue: 12,
              issue_maker: {post: parent, state: document.body.dataset.makerState},
            },
          },
        ],
        truncated: false,
      });
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const maker = feed.locator('#feed-post-' + 'b'.repeat(32));
  await maker.getByText('AI refinement is waiting for preparation.', {exact: false}).waitFor();
  assert.equal(await maker.count(), 1);
  assert.equal(await maker.getAttribute('data-state'), 'waiting');
  assert.equal(await maker.locator('[aria-label="working"]').count(), 0);
  assert.equal(await feed.locator('article[data-role="issue_maker"]').count(), 4);
  for (const [index, issue] of [14, 13, 12].entries()) {
    const receipt = feed.locator('#feed-post-' + String(index + 1).repeat(32));
    assert.match(
      (await receipt.getByRole('link', {name: `Open issue #${issue}`, exact: true}).getAttribute('href'))!,
      new RegExp(`/issues/${issue}$`)
    );
    assert.equal(await receipt.locator('.soda-feed-parent').innerText(), 'Replying to Issue Maker');
    assert.match((await receipt.locator('.soda-feed-parent a').getAttribute('href'))!, /post=b{32}$/);
    assert.equal(await receipt.locator('article').count(), 0);
  }
  const handoff = feed.getByRole('link', {name: 'Follow Issue Maker'});
  assert.match((await handoff.getAttribute('href'))!, /\/feed\?post=b{32}$/);
  assert.equal(await feed.locator('.soda-feed-outcome').getByText('Product Lead', {exact: true}).count(), 0);
  await page.evaluate(async () => {
    document.body.dataset.makerState = 'running';
    await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
  });
  await page.waitForFunction(
    () => document.querySelector('#feed-post-' + 'b'.repeat(32))?.getAttribute('data-state') === 'in_progress'
  );
  assert.equal(await maker.count(), 1);
  assert.equal(await feed.locator('article[data-role="issue_maker"]').count(), 4);
  await page.evaluate(() => {
    document.body.dataset.makerState = 'waiting';
    (document.querySelector('soda-factory-feed') as unknown as {post: string}).post = 'b'.repeat(32);
  });
  await maker.getByText('Issue Maker is waiting.', {exact: true}).waitFor();
  assert.equal(await maker.getAttribute('data-state'), 'waiting');
  for (const issue of [12, 13, 14]) await maker.getByRole('link', {name: new RegExp(`issue #${issue}:`)}).waitFor();
  // A focused post keeps updating even after it falls outside the newest feed page.
  await page.evaluate(async () => {
    document.body.dataset.makerOffPage = 'true';
    document.body.dataset.makerState = 'running';
    await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
  });
  await maker.getByText('Issue Maker is running.', {exact: true}).waitFor();
  assert.equal(await maker.getAttribute('data-state'), 'in_progress');
  assert.equal(await maker.count(), 1);
  for (const issue of [12, 13, 14])
    assert.match(
      (await maker.getByRole('link', {name: new RegExp(`issue #${issue}:`)}).getAttribute('href'))!,
      new RegExp(`/issues/${issue}$`)
    );
});

test('new-work authoring shows waiting, queued, started and settled states without false working indicators', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    document.body.dataset.authoringState = 'waiting';
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: ['7'], revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [
            {
              id: '7',
              owner: 'soda-tester',
              name: 'demo',
              can_create: false,
              project: {id: 'p' + '1'.repeat(24), provisioned: true},
            },
          ],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?')) {
        const state = document.body.dataset.authoringState;
        return Response.json({
          entries: [
            {
              kind: 'post',
              id: 'c'.repeat(32),
              repository: '7',
              seq: 1,
              author: 'alice',
              summary: 'Please add a login flow',
              post: {
                post_id: 'c'.repeat(32),
                repository: '7',
                revision: 0,
                author: '1',
                author_name: 'alice',
                body: 'Please add a login flow',
                at: 1729000000,
                intake: 'recorded',
                judgment: 'applied',
                route: 'new_issue',
                draft: state === 'created' ? 'applied' : 'pending',
                effect: state === 'created' ? 'verified' : 'pending',
                issue: state === 'created' ? 12 : 0,
                ...(state === 'created'
                  ? {}
                  : {
                      authoring: {
                        state,
                        reason: 'preparation_not_ready',
                        detail: state === 'waiting' ? 'project has no accepted requirements' : '',
                        intent: 'd' + 'b'.repeat(24),
                        run: state === 'running' ? 'recorded-run' : '',
                        since_unix: 1729000000,
                        checked_unix: 1729000010,
                      },
                    }),
              },
            },
          ],
          truncated: false,
        });
      }
      return null;
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const outcome = feed.locator('.soda-feed-outcome');
  await outcome.getByText('project has no accepted requirements', {exact: true}).waitFor();
  assert.equal(await outcome.getAttribute('data-phase'), 'draft_waiting');
  await outcome.getByText('Factory', {exact: true}).waitFor();
  await outcome.getByText('Waiting', {exact: true}).waitFor();
  await outcome.getByText(/Last checked/).waitFor();
  assert.equal(await outcome.locator('[aria-label="working"]').count(), 0);
  assert.match(
    (await outcome.getByRole('link', {name: 'View request'}).getAttribute('href'))!,
    /\/requests\/d[b]{24}$/
  );
  assert.match(
    (await outcome.getByRole('link', {name: 'Review Project setup'}).getAttribute('href'))!,
    /\/settings\/factory$/
  );
  for (const [state, phase, working] of [
    ['queued', 'draft_queued', 0],
    ['running', 'drafting', 1],
    ['settling', 'draft_settling', 0],
    ['failed', 'draft_failed', 0],
    ['unknown', 'draft_unknown', 0],
    ['created', 'created', 0],
  ] as const) {
    await page.evaluate(async (state) => {
      document.body.dataset.authoringState = state;
      await (document.querySelector('soda-factory-feed') as unknown as {reload(): Promise<void>}).reload();
    }, state);
    await page.waitForFunction(
      (phase) => document.querySelector('.soda-feed-outcome')?.getAttribute('data-phase') === phase,
      phase
    );
    assert.equal(await outcome.locator('[aria-label="working"]').count(), working, state);
  }
  await outcome.getByRole('link', {name: 'Open issue #12'}).waitFor();
});

test('feed distinguishes a failed read from no matches and keeps drafts through recovery', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
    document.body.dataset.feedUnavailable = 'true';
    window.drawerFixture.setReply(async (call) => {
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
      if (!call.url.includes('/api/factory/feed?')) return null;
      if (document.body.dataset.feedUnavailable === 'true') return Response.json({}, {status: 503});
      const filtered = new URL(call.url, location.origin).searchParams.get('view') === 'decisions';
      return Response.json({
        entries: filtered
          ? []
          : ['judgment:applied', 'draft:valid', 'effect:created', undefined].map((status, index) => ({
              kind: 'record',
              id: String(index + 1).repeat(32),
              repository: '7',
              seq: index + 1,
              ...(status ? {status} : {}),
            })),
        truncated: false,
      });
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Activity could not be loaded.', exact: true}).waitFor();
  assert.equal(await feed.getByText('No activity yet.', {exact: true}).count(), 0);
  const draft = feed.getByLabel('New post', {exact: true});
  await draft.fill('Keep this unsent note.');
  await page.evaluate(() => {
    document.body.dataset.feedUnavailable = 'false';
  });
  await feed.getByRole('button', {name: 'Retry loading activity', exact: true}).click();
  await feed.locator('.soda-feed-status').getByText('Decided', {exact: true}).waitFor();
  assert.equal(await feed.locator('.soda-feed-status').getByText('Done', {exact: true}).count(), 2);
  assert.equal(await feed.locator('.soda-feed-status').getByText('For your information', {exact: true}).count(), 1);
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  await feed.getByRole('button', {name: 'Choices & approvals', exact: true}).click();
  await feed.getByRole('heading', {name: 'No matching posts.', exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Clear filters', exact: true}).click();
  await feed.locator('.soda-feed-status').getByText('Decided', {exact: true}).waitFor();
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  await page.evaluate(() => {
    document.body.dataset.feedUnavailable = 'true';
  });
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Showing the last loaded activity. Refresh to check for updates.', {exact: true}).waitFor();
  assert.equal(await feed.locator('.soda-feed-post').count(), 4);
  assert.equal(await draft.inputValue(), 'Keep this unsent note.');
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.method !== 'GET' && !call.url.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0
  );
  await artifactPreviews();
  await artifactHost(t);
});

test('every feed author loads its own role portrait, while human and unknown authors keep their initials', async (t) => {
  const page = await fixture(t);
  const actors: [string, string][] = [
    ['product_lead', 'product-lead'],
    ['architect', 'architect'],
    ['builder', 'builder'],
    ['inspector', 'inspector'],
    ['fixer', 'fixer'],
    ['strategist', 'strategist'],
    ['test_pilot', 'test-pilot'],
    ['jev', 'jev'],
    ['factory', 'factory'],
    ['System One', 'jev'],
    ['systemone', 'jev'],
    ['system', 'factory'],
    ['17', ''],
    ['unknown_role', ''],
  ];
  await page.evaluate((actors) => {
    window.drawerFixture.api.setPresentation('settings');
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: ['7'], kinds: null, revision: 1});
      if (call.url.includes('/api/repositories?q='))
        return Response.json({
          items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
          next_cursor: '',
        });
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({
          entries: actors.map(([actor], index) => ({
            kind: actor === '17' ? 'post' : 'record',
            id: (index + 1).toString(16).padStart(32, '0'),
            repository: '7',
            seq: 100 - index,
            actor,
            author: actor === '17' ? 'Architect' : 'Builder',
            payload: {
              communication: {kind: 'flag', title: 'Portrait identity', summary: 'The author role owns this portrait.'},
            },
          })),
          truncated: false,
        });
      return null;
    });
  }, actors);
  await refresh(page);
  await page.evaluate(() => window.drawerFixture.mountFeed());
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Portrait identity', exact: true}).first().waitFor();
  assert.equal(await feed.locator('.soda-feed-post').count(), actors.length);
  for (const [index, [actor, file]] of actors.entries()) {
    const kind = actor === '17' ? 'post' : 'record';
    const card = feed.locator(`#feed-${kind}-${(index + 1).toString(16).padStart(32, '0')}`);
    const image = card.locator('.soda-feed-avatar img');
    if (!file) {
      assert.equal(await image.count(), 0, actor);
      assert.equal(await card.locator('.soda-feed-avatar').textContent(), actor === '17' ? 'A' : 'B');
      continue;
    }
    assert.equal(await card.getAttribute('data-role'), file.replaceAll('-', '_'));
    assert.equal(await image.getAttribute('src'), `/assets/role-portraits/ps1-v1/${file}.png`);
    assert.equal(await image.getAttribute('alt'), '');
    await image.scrollIntoViewIfNeeded();
    const size = await image.evaluate(async (node) => {
      const img = node as HTMLImageElement;
      await img.decode();
      return [img.naturalWidth, img.naturalHeight];
    });
    assert.deepEqual(size, [1254, 1254], actor);
  }
});

test('Project feed reads newest-first with enriched posts and live comment outcomes from one read', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    window.drawerFixture.api.setPresentation('settings');
  });
  await page.evaluate(
    async ({recordId, postId, humanReplyId}) => {
      const state = {feed: 0, comments: [] as string[]};
      (window as unknown as {feedProbeState?: unknown}).feedProbeState = state;
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription') && call.method === 'GET')
          return Response.json({subscriber: '1', repositories: ['7'], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q=') && call.method === 'GET')
          return Response.json({
            items: [
              {
                id: '7',
                owner: 'soda-tester',
                name: 'demo',
                can_create: false,
                project: {id: `p${'1'.repeat(24)}`, provisioned: true},
              },
            ],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?') && call.method === 'GET') {
          state.feed++;
          if (call.url.includes('cursor=')) {
            return Response.json({
              entries: [{kind: 'record', id: 'e'.repeat(32), repository: '7', seq: 30}],
              truncated: false,
            });
          }
          return Response.json({
            entries: [
              {kind: 'record', id: recordId, repository: '7', seq: 42, issue: 3},
              {
                kind: 'post',
                id: postId,
                repository: '7',
                seq: 41,
                actor: 'Product Lead',
                author: 'Ada',
                project: 'soda-tester/demo',
                at: 1720000000,
                status: 'route:discussion',
                post: {
                  post_id: postId,
                  revision: 0,
                  repository: '7',
                  author: '1',
                  author_name: 'Ada',
                  at: 1720000000,
                  edited: false,
                  body: 'Should the demo cover substitute approvers?',
                  intake: 'recorded',
                  judgment: 'applied',
                  route: 'discussion',
                  draft: 'pending',
                  effect: 'pending',
                  issue: 0,
                },
              },
            ],
            next: 'v1:41',
            truncated: true,
          });
        }
        if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST') {
          const body = JSON.parse((call as unknown as {body?: string}).body ?? '{}') as Record<string, unknown>;
          state.comments.push(typeof body.body === 'string' ? body.body : '');
          return Response.json({
            post_id: humanReplyId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        }
        if (call.url.includes('/replies?') && !call.url.includes(`/posts/${recordId}/replies?`))
          return Response.json({items: [], next_before: 0});
        if (call.url.includes('/replies?'))
          return Response.json({
            items:
              state.comments.length === 0
                ? []
                : [
                    {
                      post_id: humanReplyId,
                      revision: 0,
                      repository: '7',
                      author: '1',
                      author_name: 'alice',
                      at: 1720000000,
                      edited: false,
                      deleted: false,
                      revoked: false,
                      body: 'Please add substitute approvers.',
                      intake: 'recorded',
                      judgment: 'applied',
                      route: 'new_issue',
                      draft: 'applied',
                      effect: 'verified',
                      issue: 12,
                      source: {
                        kind: 'record',
                        id: recordId,
                        issue: 3,
                        status: 'implement:finished:succeeded',
                        summary: 'Built the form service.',
                      },
                    },
                  ],
            next_before: 0,
          });
        return null;
      });
    },
    {recordId, postId, humanReplyId}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  // Separate posts with the outside-card author context and stored text.
  await feed.getByText('Recorded work').first().waitFor();
  await feed.getByText('Ada').first().waitFor();
  await feed.locator('.soda-feed-header').getByText('Product Lead').first().waitFor();
  await feed.locator('.soda-feed-post[data-card=post]').first().waitFor();
  await feed
    .getByText('This reads as a discussion, so I’ve left it here for people to answer.', {exact: true})
    .waitFor();
  await feed.getByText('Should the demo cover substitute approvers?').waitFor();
  // Cursor history: older entries append behind the confirmed page.
  await feed.getByRole('button', {name: 'Load older entries'}).click();
  await feed.getByRole('button', {name: 'Back to latest'}).waitFor();
  // Drafts survive refreshes with reading position intact.
  const draft = feed.locator(`textarea[data-feed-draft="record:${recordId}"]`);
  await feed.locator(`[data-feed-drawer-toggle="record:${recordId}"]`).click();
  await draft.fill('half-written reply');
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.').waitFor();
  assert.equal(await draft.inputValue(), 'half-written reply');
  // Submit persists through the real intake shape and exposes progress.
  await feed.locator(`[data-post-submit="record:${recordId}"]`).click();
  await feed.getByText('Reply saved.').waitFor();
  const posted = await page.evaluate(() => {
    const drawer = window.drawerFixture as unknown as {calls: {url: string; method: string; body?: string}[]};
    return drawer.calls
      .filter((call) => call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST')
      .map((call) => JSON.parse(call.body ?? '{}'));
  });
  assert.equal(posted.length, 1);
  assert.equal(posted[0]?.repository, '7');
  // Commenting from a work card carries its authorized bound source issue
  // and names the source entry for server-side resolution.
  assert.equal(posted[0]?.issue, 3);
  assert.equal(posted[0]?.parent_id, recordId);
  assert.equal('source_id' in posted[0], false);
  assert.equal(posted[0]?.body, 'half-written reply');
  assert.match(
    typeof posted[0]?.idempotency_key === 'string' ? (posted[0]?.idempotency_key as string) : '',
    /^[A-Za-z0-9_-]{1,64}$/
  );
  assert.equal('author' in (posted[0] ?? {}), false);
  // The refresh after saving re-reads progress without another submission.
  const issueLink = feed.getByRole('link', {name: 'Open issue #12'});
  await issueLink.waitFor();
  assert.equal(await issueLink.getAttribute('href'), '/soda-tester/demo/issues/12');
  await feed.getByText(/I created issue #12 for this\./).waitFor();
  assert.equal(
    await page.evaluate(() => window.drawerFixture.calls.filter((call) => call.url.includes('/feed/comments/')).length),
    0
  );
});

test('a project comment works before the first worker post', async (t) => {
  const page = await fixture(t);
  await page.evaluate(
    ({humanReplyId}) => {
      const repository = '9223372036854775807';
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/native-repositories/resolve'))
          return Response.json({
            host: 'a'.repeat(32),
            actor: '1',
            repository,
            native_repository: '7',
            name: 'alice/demo',
          });
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: [repository], kinds: [], revision: 1});
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        const posted = window.drawerFixture.calls.some(
          (c) => c.url.endsWith('/api/factory/feed/posts') && c.method === 'POST'
        );
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: posted ? [{kind: 'post', id: humanReplyId, repository, seq: 1, post: stored}] : [],
            truncated: false,
          });
        if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST')
          return Response.json({
            post_id: humanReplyId,
            revision: 0,
            intake: 'recorded',
            judgment: 'pending',
            draft: 'pending',
            effect: 'pending',
          });
        return null;
      });
      const stored = {
        post_id: humanReplyId,
        revision: 0,
        repository,
        author: '1',
        author_name: 'alice',
        at: 1720000000,
        edited: false,
        deleted: false,
        revoked: false,
        body: 'Thanks for the update.',
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      };
      window.drawerFixture.mountFeed();
    },
    {humanReplyId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('No activity yet.').waitFor();
  await feed.getByLabel('New post', {exact: true}).fill('Thanks for the update.');
  await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
  await feed.getByText('This reads as a discussion, so I’ve left it here for people to answer.').waitFor();
  assert.equal(await feed.getByLabel('New post', {exact: true}).inputValue(), '');
  assert(await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).isDisabled());
  const posts = await page.evaluate(() =>
    window.drawerFixture.calls
      .filter((c) => c.url.endsWith('/api/factory/feed/posts') && c.method === 'POST')
      .map((c) => JSON.parse(c.body ?? '{}'))
  );
  assert.equal(posts.length, 1);
  assert.equal(posts[0].repository, '9223372036854775807');
  assert.equal(posts[0].body, 'Thanks for the update.');
  assert.equal('source_id' in posts[0], false);
  assert.equal('parent_id' in posts[0], false);
});

test('Project switch during feed loading starts the new read and releases Refresh', async (t) => {
  const page = await fixture(t);
  await page.evaluate(async () => {
    const feed = document.createElement('soda-factory-feed') as HTMLElement & {
      context: unknown;
      updateComplete: Promise<unknown>;
    };
    feed.dataset.feedRace = 'true';
    document.body.append(feed);
    let count = 0;
    let release: (response: Response) => void = () => {};
    const emptyFeed = () => Response.json({entries: [], truncated: false});
    const transport = {
      request: async (path: string, init?: RequestInit) => {
        if (path === 'factory/identity') return Response.json({host: 'a'.repeat(32), actor: '1', native_actor: '1'});
        if (path === 'factory/native-repositories/resolve') {
          const native = JSON.parse(String(init?.body)).repository;
          return Response.json({
            host: 'a'.repeat(32),
            actor: '1',
            repository: native,
            native_repository: native,
            name: `alice/repository-${native}`,
          });
        }
        if (path.startsWith('factory/feed?')) {
          count++;
          if (count === 1)
            return new Promise<Response>((resolve) => {
              release = resolve;
            });
          return emptyFeed();
        }
        if (path === 'repositories?q=') return Response.json({items: [], next_cursor: ''});
        throw Error('Unexpected feed read');
      },
    };
    Object.assign(feed, {releaseFirst: () => release(emptyFeed()), readCount: () => count});
    feed.context = await window.drawerFixture.feedContext(
      {...transport, generation: 'feed-race', assetBase: '/assets/'},
      {repository: '7'}
    );
    await feed.updateComplete;
    while (count === 0) await new Promise((resolve) => setTimeout(resolve, 0));
    feed.context = await window.drawerFixture.feedContext(
      {...transport, generation: 'feed-race', assetBase: '/assets/'},
      {repository: '8'}
    );
  });
  const feed = page.locator('soda-factory-feed[data-feed-race="true"]');
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).waitFor();
  await page.waitForFunction(
    () =>
      !document.querySelector<HTMLButtonElement>(
        'soda-factory-feed[data-feed-race="true"] button[aria-label="Refresh feed"]'
      )?.disabled
  );
  await feed.evaluate((el) => (el as HTMLElement & {releaseFirst(): void}).releaseFirst());
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  assert.ok((await feed.evaluate((el) => (el as HTMLElement & {readCount(): number}).readCount())) >= 3);
  assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
  await standaloneFeedLifetime();
});

test('an uncertain comment retry preserves its command through storage denial and remount', async (t) => {
  for (const persistent of [true, false]) {
    const page = await fixture(t);
    await page.evaluate(
      ({humanReplyId, persistent}) => {
        if (!persistent)
          Object.defineProperty(window, 'localStorage', {
            configurable: true,
            get() {
              throw new DOMException('Declared fixture storage denial', 'SecurityError');
            },
          });
        let attempts = 0;
        window.drawerFixture.setReply(async (call) => {
          if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
          if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
          if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST') {
            attempts++;
            return attempts === 1
              ? Response.json({error: {code: 'reply_lost'}}, {status: 503})
              : Response.json({
                  post_id: humanReplyId,
                  revision: 0,
                  intake: 'recorded',
                  judgment: 'applied',
                  route: 'discussion',
                  draft: 'pending',
                  effect: 'pending',
                  issue: 0,
                });
          }
          if (call.url.endsWith('/api/factory/feed/posts/' + humanReplyId))
            return Response.json({
              post_id: humanReplyId,
              revision: 0,
              body: 'Original request',
              intake: 'recorded',
              judgment: 'applied',
              route: 'discussion',
              draft: 'pending',
              effect: 'pending',
              issue: 0,
            });
          return null;
        });
        window.drawerFixture.mountFeed();
      },
      {humanReplyId, persistent}
    );
    const feed = page.locator('soda-factory-feed');
    await feed.getByText('No activity yet.').waitFor();
    let draft = feed.getByLabel('New post', {exact: true});
    await draft.fill('Original request');
    await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
    await feed.locator('[data-feed-message]').filter({hasText: 'draft is kept'}).waitFor();
    await draft.fill('New writing after the uncertain reply');
    await feed.evaluate((node) => node.remove());
    await page.evaluate(() => window.drawerFixture.mountFeed());
    await feed.getByText('No activity yet.').waitFor();
    draft = feed.getByLabel('New post', {exact: true});
    assert.equal(await draft.inputValue(), 'New writing after the uncertain reply');
    await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
    await feed.getByText('Previous post saved. Your newer edits are kept, ready to send.', {exact: true}).waitFor();
    assert.equal(await draft.inputValue(), 'New writing after the uncertain reply');
    const posts = await page.evaluate(() =>
      window.drawerFixture.calls
        .filter((call) => call.method === 'POST' && call.url.endsWith('/api/factory/feed/posts'))
        .map((call) => JSON.parse(call.body ?? '{}'))
    );
    assert.equal(posts.length, 2);
    assert.equal(posts[0].body, 'Original request');
    assert.deepEqual(posts[1], posts[0], 'retry must not attach new writing to the original idempotency key');
    await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
    await feed.getByText('Post saved.', {exact: true}).waitFor();
    const next = await page.evaluate(() =>
      window.drawerFixture.calls
        .filter((call) => call.method === 'POST' && call.url.endsWith('/api/factory/feed/posts'))
        .map((call) => JSON.parse(call.body ?? '{}'))
    );
    assert.equal(next.length, 3);
    assert.equal(next[2].body, 'New writing after the uncertain reply');
    assert.notEqual(next[2].idempotency_key, posts[0].idempotency_key);
    if (persistent) {
      const scope = `fountain:${'a'.repeat(32)}:1:project:7`;
      const pendingKey = `soda-feed-pending:v1:${scope}`;
      const commandKey = `soda-feed-draftkey:v2:${scope}`;
      const legacyKey = 'soda-feed-pending:v1:1:project:7';
      const recovered = {
        repository: '7',
        body: 'Original persisted command',
        idempotency_key: 'retained-command',
        issue: 42,
      };
      await page.evaluate(
        ({scope, pendingKey, commandKey, legacyKey, recovered}) => {
          localStorage.setItem(pendingKey, JSON.stringify(recovered));
          localStorage.removeItem(commandKey);
          localStorage.setItem(`soda-feed-draft:v1:${scope}`, 'Newer writing survives a browser reload');
          localStorage.setItem(legacyKey, 'Unscoped historical bytes');
        },
        {scope, pendingKey, commandKey, legacyKey, recovered}
      );
      await page.reload();
      await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
      await page.evaluate(
        async ({humanReplyId}) => {
          window.drawerFixture = window.createDrawerFixture({});
          await window.drawerFixture.api.ready;
          window.drawerFixture.setReply(async (call) => {
            if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
            if (call.url.includes('/api/factory/feed?')) return Response.json({entries: [], truncated: false});
            if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST')
              return Response.json({
                post_id: humanReplyId,
                revision: 0,
                intake: 'recorded',
                judgment: 'applied',
                route: 'discussion',
                draft: 'pending',
                effect: 'pending',
              });
            return null;
          });
          window.drawerFixture.mountFeed();
        },
        {humanReplyId}
      );
      await feed.getByText('No activity yet.').waitFor();
      draft = feed.getByLabel('New post', {exact: true});
      assert.equal(await draft.inputValue(), 'Newer writing survives a browser reload');
      await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
      await feed.getByText('Previous post saved. Your newer edits are kept, ready to send.', {exact: true}).waitFor();
      const restoredPosts = await page.evaluate(() =>
        window.drawerFixture.calls
          .filter((call) => call.method === 'POST' && call.url.endsWith('/api/factory/feed/posts'))
          .map((call) => JSON.parse(call.body ?? '{}'))
      );
      assert.deepEqual(restoredPosts, [recovered]);
      assert.equal(await draft.inputValue(), 'Newer writing survives a browser reload');
      assert.deepEqual(
        await page.evaluate(
          ({pendingKey, commandKey, legacyKey}) => [
            localStorage.getItem(pendingKey),
            localStorage.getItem(commandKey),
            localStorage.getItem(legacyKey),
          ],
          {pendingKey, commandKey, legacyKey}
        ),
        [null, null, 'Unscoped historical bytes']
      );

      // Numeric/foreign/corrupt pending copies have no proved current binding.
      // Submitting newer writing must neither replay nor replace those bytes.
      for (const pending of [
        JSON.stringify({...recovered, repository: 7}),
        JSON.stringify({...recovered, repository: '9'}),
        JSON.stringify({...recovered, parent_id: 'e'.repeat(32)}),
        JSON.stringify({...recovered, issue: -1}),
        '{unreadable',
      ]) {
        await page.evaluate(
          ({pendingKey, commandKey, pending}) => {
            localStorage.setItem(pendingKey, pending);
            localStorage.setItem(commandKey, 'retained-command');
          },
          {pendingKey, commandKey, pending}
        );
        await feed.locator('.soda-feed-composer').getByRole('button', {name: 'Post', exact: true}).click();
        await feed
          .getByText('A saved post could not be safely resumed. Its pending copy and your draft are kept.', {
            exact: true,
          })
          .waitFor();
        assert.deepEqual(
          await page.evaluate(
            ({pendingKey, commandKey}) => [localStorage.getItem(pendingKey), localStorage.getItem(commandKey)],
            {pendingKey, commandKey}
          ),
          [pending, 'retained-command']
        );
        assert.equal(await draft.inputValue(), 'Newer writing survives a browser reload');
        assert.equal(
          await page.evaluate(
            () =>
              window.drawerFixture.calls.filter(
                (call) => call.method === 'POST' && call.url.endsWith('/api/factory/feed/posts')
              ).length
          ),
          1
        );
      }
    }
  }
});

test('clarification answer draft survives reopen with zero writes and clears only on submit', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const clarifyId = 'd'.repeat(32);
  await page.evaluate(
    ({record, clarifyId}) => {
      const detail = {
        post_id: clarifyId,
        revision: 3,
        repository: '7',
        author_name: 'alice',
        at: 1720000000,
        edited: false,
        body: 'Original request text',
        intake: 'recorded',
        judgment: 'applied',
        draft: 'pending',
        effect: 'pending',
        route: 'clarify',
        detail: 'What should change, and where should it apply?',
        can_retry: true,
        issue: 0,
      };
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: '7', seq: 1}], truncated: false});
        if (call.url.includes('/replies?')) return Response.json({items: [detail], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/posts/${clarifyId}`) && call.method === 'GET')
          return Response.json(detail);
        if (call.url.endsWith(`/api/factory/feed/posts/${clarifyId}/retry`) && call.method === 'POST') {
          const input = JSON.parse(call.body ?? '{}');
          const digest = [
            ...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(input.body))),
          ]
            .map((byte) => byte.toString(16).padStart(2, '0'))
            .join('');
          return Response.json({
            command_id: input.command_id,
            repository: input.repository,
            author: '1',
            post_id: clarifyId,
            expected_revision: input.expected_revision,
            body_digest: digest,
            revision: 4,
          });
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, clarifyId}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Answer clarification', exact: true}).click();
  const answer = feed.locator('[data-revision-body]');
  await answer.waitFor();
  await answer.fill('My saved answer');
  const draftKey = `soda-feed-revision:v2:fountain:${'a'.repeat(32)}:1:7:${clarifyId}`;
  assert.deepEqual(await page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) ?? 'null'), draftKey), {
    body: 'My saved answer',
    original: 'Original request text',
  });
  // A reload drops the panel; the stored answer survives it.
  await page.evaluate(() => {
    document.querySelector('soda-factory-feed')?.remove();
    window.drawerFixture.mountFeed();
  });
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Answer clarification', exact: true}).click();
  await feed.locator('.soda-feed-revision').waitFor();
  assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'My saved answer');
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.method !== 'GET' && !call.url.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0,
    'restoring a saved answer must not write'
  );
  await feed.getByRole('button', {name: 'Submit answer', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const posted = await page.evaluate(() =>
    window.drawerFixture.calls.filter(
      (call) => call.method === 'POST' && !call.url.endsWith('/api/factory/native-repositories/resolve')
    )
  );
  const revisionCommand = JSON.parse(posted[0]?.body ?? '{}');
  assert.match(revisionCommand.command_id, /^[a-f0-9]{32}$/);
  assert.deepEqual(revisionCommand, {
    command_id: revisionCommand.command_id,
    repository: '7',
    expected_revision: 3,
    body: 'Original request text\n\nClarification: My saved answer',
  });
  assert.equal(await page.evaluate((key) => window.localStorage.getItem(key), draftKey), null);
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
});

test('posts and replies edit their own identities and a reply can receive another reply', async (t) => {
  const page = await fixture(t);
  page.setDefaultTimeout(8000);
  const root = 'a'.repeat(32),
    child = 'b'.repeat(32),
    nested = 'c'.repeat(32);
  await page.evaluate(
    ({root, child, nested}) => {
      const posts: Record<string, Record<string, unknown>> = {};
      const make = (id: string, body: string, parent = '') => ({
        post_id: id,
        repository: '7',
        revision: 0,
        author: '1',
        author_name: 'alice',
        at: 1720000000,
        body,
        parent_id: parent,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        can_retry: true,
      });
      posts[root] = make(root, 'Root post');
      posts[child] = make(child, 'Child reply', root);
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: ['7'], kinds: [], revision: 1});
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [{kind: 'post', id: root, repository: '7', seq: 1, post: posts[root]}],
            truncated: false,
          });
        if (call.url.includes('/replies?')) {
          const parent = /\/posts\/([a-f0-9]{32})\/replies\?/.exec(call.url)?.[1];
          return Response.json({
            items: Object.values(posts)
              .filter((post) => post.parent_id === parent)
              .reverse(),
            next_before: 0,
          });
        }
        if (call.url.endsWith('/api/factory/feed/posts') && call.method === 'POST') {
          const input = JSON.parse(call.body ?? '{}');
          posts[nested] = make(nested, input.body, input.parent_id);
          return Response.json(posts[nested]);
        }
        const match = /\/factory\/feed\/posts\/([a-f0-9]{32})(\/retry)?$/.exec(call.url);
        if (match && posts[match[1]!]) {
          const post = posts[match[1]!]!;
          if (match[2] && call.method === 'POST') {
            const input = JSON.parse(call.body ?? '{}');
            post.body = input.body;
            post.revision = Number(post.revision) + 1;
            const digest = [
              ...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(input.body))),
            ]
              .map((byte) => byte.toString(16).padStart(2, '0'))
              .join('');
            return Response.json({
              command_id: input.command_id,
              repository: input.repository,
              author: '1',
              post_id: post.post_id,
              expected_revision: input.expected_revision,
              body_digest: digest,
              revision: post.revision,
            });
          }
          return Response.json(post);
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {root, child, nested}
  );
  const feed = page.locator('soda-factory-feed');
  await feed.getByText('Root post', {exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
  const editor = feed.getByRole('region', {name: 'Edit post', exact: true});
  await editor.getByLabel('Edit your post', {exact: true}).fill('Updated root post');
  await editor.getByRole('button', {name: 'Save post and retry', exact: true}).click();
  await feed.getByText('Updated root post', {exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="post:${root}"]`).click();
  const childPost = feed.locator('#post-' + child);
  await childPost.getByText('Child reply', {exact: true}).waitFor();
  await childPost.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  const replyEditor = feed.getByRole('region', {name: 'Edit reply', exact: true});
  await replyEditor.getByLabel('Edit your reply', {exact: true}).fill('Updated child reply');
  await replyEditor.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await childPost.getByText('Updated child reply', {exact: true}).waitFor();
  await feed.getByText('Updated root post', {exact: true}).waitFor();
  await childPost.locator('[data-reply-compose]').click();
  await childPost.getByLabel('Reply to alice', {exact: true}).fill('A reply to the child');
  await childPost.locator(`[data-post-submit="post:${child}"]`).click();
  const nestedPost = feed.locator('#post-' + nested);
  await nestedPost.getByText('A reply to the child', {exact: true}).waitFor();
  assert.equal(await nestedPost.locator('.soda-feed-reply-parent a').getAttribute('href'), '#post-' + child);
  const writes = await page.evaluate(() =>
    window.drawerFixture.calls.filter(
      (call) => call.method === 'POST' && !call.url.endsWith('/api/factory/native-repositories/resolve')
    )
  );
  assert(writes[0]?.url.endsWith('/' + root + '/retry'));
  assert(writes[1]?.url.endsWith('/' + child + '/retry'));
  const posted = JSON.parse(writes[2]?.body ?? '{}');
  assert.equal(posted.parent_id, child);
  assert.equal('source_id' in posted, false);
  assert.doesNotMatch(await feed.innerText(), /\bcomments?\b/i);
  const accessibleLabels = await feed
    .locator('[aria-label], [title], [placeholder]')
    .evaluateAll((nodes) =>
      nodes
        .flatMap((node) => ['aria-label', 'title', 'placeholder'].map((name) => node.getAttribute(name) ?? ''))
        .join('\n')
    );
  assert.doesNotMatch(accessibleLabels, /\bcomments?\b/i);
});

test('revision draft keeps text on failure, shows the newer request on conflict, discards explicitly', async (t) => {
  const page = await fixture(t);
  const record = 'a'.repeat(32);
  const reviseId = 'e'.repeat(32);
  await page.evaluate(
    ({record, reviseId}) => {
      const state = {revision: 5, failNext: false};
      (window as unknown as {revState: typeof state}).revState = state;
      const detail = () => ({
        post_id: reviseId,
        revision: state.revision,
        repository: '7',
        author_name: 'alice',
        at: 1720000000,
        edited: false,
        body: state.revision === 5 ? 'Original request v5' : 'Original request v' + state.revision + ' changed',
        intake: 'recorded',
        judgment: 'applied',
        draft: 'pending',
        effect: 'pending',
        route: 'discussion',
        can_retry: true,
        issue: 0,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({entries: [{kind: 'record', id: record, repository: '7', seq: 1}], truncated: false});
        if (call.url.includes('/replies?')) return Response.json({items: [detail()], next_before: 0});
        if (call.url.endsWith(`/api/factory/feed/posts/${reviseId}`) && call.method === 'GET')
          return Response.json(detail());
        if (call.url.endsWith(`/api/factory/feed/posts/${reviseId}/retry`) && call.method === 'POST') {
          if (state.failNext)
            return Response.json({error: {message: 'Temporary failure. Your edits are kept.'}}, {status: 503});
          if (JSON.parse(call.body ?? '{}').expected_revision !== state.revision)
            return Response.json({error: {message: 'stale revision'}}, {status: 409});
          state.revision++;
          const input = JSON.parse(call.body ?? '{}');
          const digest = [
            ...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(input.body))),
          ]
            .map((byte) => byte.toString(16).padStart(2, '0'))
            .join('');
          return Response.json({
            command_id: input.command_id,
            repository: input.repository,
            author: '1',
            post_id: reviseId,
            expected_revision: input.expected_revision,
            body_digest: digest,
            revision: state.revision,
          });
        }
        return null;
      });
      window.drawerFixture.mountFeed();
    },
    {record, reviseId}
  );
  const feed = page.locator('soda-factory-feed');
  const setState = (next: {revision?: number; failNext?: boolean}) =>
    page.evaluate((next) => Object.assign((window as unknown as {revState: object}).revState, next), next);
  const draftKey = `soda-feed-revision:v2:fountain:${'a'.repeat(32)}:1:7:${reviseId}`;
  const storedBody = () =>
    page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) ?? 'null')?.body ?? null, draftKey);
  await feed.getByRole('region', {name: 'Feed', exact: true}).waitFor();
  await feed.locator(`[data-feed-drawer-toggle="record:${record}"]`).click();
  await feed.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  const body = feed.locator('[data-revision-body]');
  await body.waitFor();
  assert.equal(await body.inputValue(), 'Original request v5');
  await body.fill('Revised request kept');
  assert.equal(await storedBody(), 'Revised request kept');
  await setState({failNext: true});
  await feed.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await feed.locator('[data-feed-message]').filter({hasText: 'kept'}).waitFor();
  assert.equal(await body.inputValue(), 'Revised request kept');
  assert.equal(await storedBody(), 'Revised request kept');
  await setState({failNext: false});
  await feed.getByRole('button', {name: 'Retry saved edit', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  await feed.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  await body.waitFor();
  await body.fill('Revised request kept');
  // The request changes elsewhere while the panel is open: the stale submit
  // is refused, the newer text shows beside the kept draft, and the next
  // submit goes against the newer revision.
  await setState({failNext: false, revision: 7});
  await feed.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await feed
    .locator('.soda-feed-revision blockquote')
    .getByText('Original request v7 changed', {exact: true})
    .waitFor();
  assert.equal(await body.inputValue(), 'Revised request kept');
  await feed.getByRole('button', {name: 'Save reply and retry', exact: true}).click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const retries = await page.evaluate(() =>
    window.drawerFixture.calls
      .filter((call) => call.url.endsWith('/retry'))
      .map((call) => JSON.parse(call.body ?? '{}'))
  );
  assert.deepEqual(
    retries.map((retry) => retry.expected_revision),
    [5, 5, 6, 7]
  );
  assert.equal(await storedBody(), null);
  // Cancel discards kept text; the next open serves the server text.
  await feed.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  await body.waitFor();
  await body.fill('Abandoned edit');
  await feed.getByRole('button', {name: 'Cancel', exact: true}).click();
  assert.equal(await feed.locator('.soda-feed-revision').count(), 0);
  assert.equal(await storedBody(), null);
  await feed.getByRole('button', {name: 'Edit reply and retry', exact: true}).click();
  await body.waitFor();
  assert.equal(await body.inputValue(), 'Original request v8 changed');
  await feedEditLifetimes();
});

test('a role reply link pages the original thread until the older target is found and focused', async (t) => {
  const page = await fixture(t);
  const work = 'a'.repeat(32);
  const invite = 'b'.repeat(32);
  const target = 'c'.repeat(32);
  await page.evaluate(
    ({work, invite, target}) => {
      window.drawerFixture.api.setPresentation('settings');
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: ['7'], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: '7',
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          if (before)
            return Response.json({
              items: [
                {
                  post_id: target,
                  repository: '7',
                  revision: 0,
                  actor: 'architect',
                  author_name: 'Architect',
                  assignment: work,
                  parent_id: invite,
                  body: 'The role’s actual reply from the older page.',
                  at: 1729000000,
                  edited: false,
                },
              ],
              next_before: 0,
            });
          return Response.json({
            items: [
              {
                post_id: invite,
                repository: '7',
                revision: 0,
                actor: 'builder',
                author_name: 'Builder',
                assignment: work,
                parent_id: '',
                body: 'Consulted the architect.',
                at: 1729000001,
                edited: false,
                contribution: {
                  source: work,
                  revision: 0,
                  state: 'responded',
                  profile: 'architect',
                  reply: target,
                  body: 'Consulted the architect.',
                  roles: [],
                  replies: [],
                },
              },
            ],
            next_before: 5,
          });
        }
        return null;
      });
    },
    {work, invite, target}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  const link = feed.getByRole('link', {name: 'Read the role'});
  await link.waitFor();
  await link.click();
  await feed.locator(`#post-${target}`).getByText('The role’s actual reply from the older page.').waitFor();
  assert.equal(await page.evaluate(() => location.hash), '');
  await page.waitForFunction((id) => document.activeElement?.id === id, `post-${target}`);
  assert.ok(
    (await page.evaluate(
      () =>
        window.drawerFixture.calls.filter((call) => call.url.includes('/replies?') && call.url.includes('before='))
          .length
    )) >= 1,
    'the older replies page is read'
  );
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.method !== 'GET' && !call.url.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0
  );
});

test('older thread pages prepend and refreshes preserve canonical reply order', async (t) => {
  const page = await fixture(t);
  const work = '2'.repeat(32);
  // Canonical seq order is oldest, middle, newest; oldest and middle share
  // one Unix second across the page boundary. Ids are chosen so a
  // humanReplyId tiebreak would misorder them (newest sorts first).
  const oldest = 'f'.repeat(32);
  const middle = 'a'.repeat(32);
  const newest = '5'.repeat(32);
  const at = 1729000000;
  await page.evaluate(
    ({work, oldest, middle, newest, at}) => {
      window.drawerFixture.api.setPresentation('settings');
      const discussion = (post_id: string, author_name: string, body: string, stamp: number) => ({
        post_id,
        repository: '7',
        revision: 0,
        author: '1',
        author_name,
        at: stamp,
        edited: false,
        deleted: false,
        revoked: false,
        body,
        intake: 'recorded',
        judgment: 'applied',
        route: 'discussion',
        draft: 'pending',
        effect: 'pending',
        issue: 0,
      });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/api/factory/feed/subscription'))
          return Response.json({subscriber: '1', repositories: ['7'], kinds: null, revision: 1});
        if (call.url.includes('/api/repositories?q='))
          return Response.json({
            items: [{id: '7', owner: 'soda-tester', name: 'demo', can_create: false}],
            next_cursor: '',
          });
        if (call.url.includes('/api/factory/feed?'))
          return Response.json({
            entries: [
              {
                kind: 'record',
                id: work,
                repository: '7',
                seq: 1,
                actor: 'builder',
                author: 'Builder',
                status: 'assigned',
                summary: 'Working on the demo.',
              },
            ],
            truncated: false,
          });
        if (call.url.includes('/replies?')) {
          const before = new URL(call.url, location.origin).searchParams.get('before');
          // Every page arrives seq DESC, as the server sends it.
          if (before)
            return Response.json({
              items: [discussion(oldest, 'Ada', 'Can this cover substitute approvers?', at)],
              next_before: 0,
            });
          return Response.json({
            items: [
              discussion(newest, 'Fixer', 'Verified on the follow-up.', at + 10),
              discussion(middle, 'Builder', 'Covered in the candidate.', at),
            ],
            next_before: 99,
          });
        }
        return null;
      });
    },
    {work, oldest, middle, newest, at}
  );
  await refresh(page);
  await page.evaluate(() => {
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  const orderOf = () =>
    feed
      .locator('.soda-feed-thread li.soda-feed-reply')
      .evaluateAll((nodes) => nodes.map((node) => (node as HTMLElement).id));
  await feed.locator(`[data-feed-drawer-toggle="record:${work}"]`).click();
  await feed.locator(`#post-${middle}`).getByText('Covered in the candidate.').waitFor();
  assert.deepEqual(await orderOf(), [`post-${middle}`, `post-${newest}`]);
  // Older-page prepend keeps ascending canonical order across the
  // same-second page boundary.
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#post-${oldest}`).getByText('Can this cover substitute approvers?').waitFor();
  assert.deepEqual(await orderOf(), [`post-${oldest}`, `post-${middle}`, `post-${newest}`]);
  // A quiet feed refresh re-merges the newest page; dedup keeps every
  // canonical position instead of reordering.
  await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await feed.getByText('Feed refreshed; no new posts.').waitFor();
  assert.deepEqual(await orderOf(), [`post-${oldest}`, `post-${middle}`, `post-${newest}`]);
  // Reload replace re-reads the newest page, then paging restores the
  // older reply in front again.
  await feed.getByRole('button', {name: 'Refresh replies'}).click();
  await feed.locator('.soda-feed-thread').getByText('Covered in the candidate.').waitFor();
  assert.deepEqual(await orderOf(), [`post-${middle}`, `post-${newest}`]);
  await feed.getByRole('button', {name: 'Show more'}).click();
  await feed.locator(`#post-${oldest}`).getByText('Can this cover substitute approvers?').waitFor();
  assert.deepEqual(await orderOf(), [`post-${oldest}`, `post-${middle}`, `post-${newest}`]);
  assert.equal(
    await page.evaluate(
      () =>
        window.drawerFixture.calls.filter(
          (call) => call.method !== 'GET' && !call.url.endsWith('/api/factory/native-repositories/resolve')
        ).length
    ),
    0
  );
  await feedExamples(t);
});

// The same production library, bundled without Spaces or Fountain CSS, over a
// declared read peer. Native/provider effects are qualified by later consumers.
async function standaloneLibrary() {
  const build = await Bun.build({
    entrypoints: [new URL('./fixtures/factory-library-fixture.ts', import.meta.url).pathname],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, build.logs.join('\n'));
  const script = build.outputs[0]!;
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url);
      const asset = factoryAssetSource(url.pathname);
      if (asset) return asset;
      if (url.pathname === '/library.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root" data-factory-scheme="dark"></main><script type="module" src="/library.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true});
  const page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    assetErrors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.status() >= 400) assetErrors.push(response.url());
  });
  try {
    await page.goto(server.url.href);
    const library = page.locator('soda-factory-library');
    const first = library.getByRole('button', {name: 'Choose a bounded worker queue', exact: true});
    const mount = async (state: Parameters<Window['factoryLibraryFixture']['mount']>[0] = 'recorded', held = false) => {
      await page.evaluate(([state, held]) => window.factoryLibraryFixture.mount(state, held), [state, held] as const);
    };
    await first.waitFor();
    assert.equal(await library.locator('.soda-library-results article').count(), 3);
    await page.waitForFunction(
      () => getComputedStyle(document.querySelector('main')!).backgroundColor === 'rgb(16, 16, 16)'
    );
    const search = library.getByRole('searchbox', {name: 'Search the library'});
    await search.fill('bounded');
    await library.getByRole('button', {name: 'Search', exact: true}).click();
    await first.waitFor();
    assert.equal(await library.locator('.soda-library-results article').count(), 1);
    const before = await page.evaluate(() => window.factoryLibraryFixture.calls().length);
    await page.evaluate(() => window.factoryLibraryFixture.rebind());
    await library.evaluate((el) => (el as HTMLElement & {updateComplete: Promise<unknown>}).updateComplete);
    assert.equal(await search.inputValue(), 'bounded');
    assert.equal(await page.evaluate(() => window.factoryLibraryFixture.calls().length), before);
    await first.click();
    await library.getByRole('heading', {name: 'Choose a bounded worker queue', exact: true}).waitFor();
    assert.match(await library.locator('.soda-library-detail').innerText(), /Use a bounded queue/);
    assert.match(await library.locator('.soda-library-detail').innerText(), /<script>untrusted\(\)<\/script>/);
    assert.equal(await library.locator('script').count(), 0);
    await library.locator('summary').filter({hasText: 'Recent revisions'}).click();
    await library.locator('summary').filter({hasText: 'Revision 2'}).click();
    await library.getByRole('heading', {name: 'Earlier reviewed choice', exact: true}).waitFor();
    await library.getByRole('button', {name: 'Open source post', exact: true}).last().click();
    assert.deepEqual((await page.evaluate(() => window.factoryLibraryFixture.sources())).at(-1), {
      repository: '9007199254740993',
      id: 'e'.repeat(32),
      item: 'a'.repeat(32),
      revision: 2,
    });
    await page.evaluate(() => window.factoryLibraryFixture.pendingSource('e'.repeat(32)));
    assert(await library.getByRole('button', {name: 'Opening source…'}).isDisabled());
    await library.getByRole('button', {name: 'Open item bbbbbbbb', exact: true}).click();
    await library.getByRole('heading', {name: 'Retained observations', exact: true}).waitFor();
    await library.getByRole('button', {name: '← Back to results', exact: true}).click();
    await first.waitFor();
    assert.equal(await search.inputValue(), 'bounded');

    await mount('paged');
    await first.waitFor();
    assert.equal(await library.locator('.soda-library-results article').count(), 1);
    await library.getByRole('button', {name: 'More library items'}).click();
    await library.getByRole('button', {name: 'Earlier context', exact: true}).waitFor();
    assert.equal(await library.locator('.soda-library-results article').count(), 3);
    assert.equal(await library.getByRole('button', {name: 'More library items'}).count(), 0);
    assert(
      (await page.evaluate(() => window.factoryLibraryFixture.calls())).some((call) =>
        call.path.includes('cursor=' + 'a'.repeat(32))
      )
    );

    await mount('removed');
    await library.getByRole('button', {name: 'Retained observations', exact: true}).waitFor();
    assert.equal(await first.count(), 0);
    await library.getByRole('checkbox', {name: 'Include removed items'}).check();
    await first.waitFor();
    await library.getByRole('combobox', {name: 'Item type'}).selectOption('decision');
    await first.waitFor();
    assert.equal(await library.locator('.soda-library-results article').count(), 1);
    await first.click();
    await library.getByRole('heading', {name: 'Choose a bounded worker queue', exact: true}).waitFor();
    assert.match(await library.locator('.soda-library-detail').innerText(), /Removed/);

    // A failed detail refresh retains labelled facts; denied reads clear them.
    await page.evaluate(() => {
      window.factoryLibraryFixture.fail(503);
      void window.factoryLibraryFixture.refresh();
    });
    await library.getByRole('alert').filter({hasText: 'Last confirmed library items are shown'}).waitFor();
    assert.equal(await library.getByRole('button', {name: 'Open source post', exact: true}).count(), 1);
    for (const code of [401, 403, 404]) {
      await mount();
      await first.waitFor();
      await first.click();
      await library.getByRole('heading', {name: 'Choose a bounded worker queue', exact: true}).waitFor();
      await page.evaluate((code) => {
        window.factoryLibraryFixture.fail(code);
        void window.factoryLibraryFixture.refresh();
      }, code);
      await library.getByRole('alert').filter({hasText: 'not available to your account'}).waitFor();
      assert.equal(await library.locator('.soda-library-detail').count(), 0);
      assert.equal(await library.getByRole('button', {name: 'Open source post', exact: true}).count(), 0);
      await page.evaluate(() => window.factoryLibraryFixture.fail(0));
      await library.getByRole('button', {name: 'Retry library read'}).click();
      await library.getByRole('heading', {name: 'Choose a bounded worker queue', exact: true}).waitFor();
      assert((await page.evaluate(() => window.factoryLibraryFixture.calls())).at(-1)!.path.includes('/item?'));
    }

    // Malformed scope, item, history, revision and continuation never publish.
    for (const mismatch of ['repository', 'item', 'history', 'revision'] as const) {
      await mount();
      await first.waitFor();
      await page.evaluate((mismatch) => window.factoryLibraryFixture.mismatch(mismatch), mismatch);
      await first.click();
      await library.getByRole('alert').waitFor();
      assert.equal(await library.locator('.soda-library-detail').count(), 0);
      assert.equal(await library.getByRole('button', {name: 'Open source post', exact: true}).count(), 0);
    }
    await mount('paged');
    await first.waitFor();
    await page.evaluate(() => window.factoryLibraryFixture.mismatch('cursor'));
    await library.getByRole('button', {name: 'More library items'}).click();
    await library.getByRole('alert').waitFor();
    assert.equal(await library.locator('.soda-library-results article').count(), 1);
    assert.match(await library.getByRole('alert').innerText(), /Last confirmed/);
    await mount();
    await first.waitFor();
    await page.evaluate(() => {
      window.factoryLibraryFixture.mismatch('repository');
      void window.factoryLibraryFixture.refresh();
    });
    await library.getByRole('alert').waitFor();
    assert.equal(await library.locator('.soda-library-results article').count(), 0);

    // Internal navigation cancels a delayed item; the older response stays retired.
    await mount();
    await first.waitFor();
    await page.evaluate(() => window.factoryLibraryFixture.hold());
    await first.click();
    await library.getByRole('status').filter({hasText: 'Loading library'}).waitFor();
    await page.evaluate(() => window.factoryLibraryFixture.select('b'.repeat(32)));
    await page.waitForFunction(() =>
      window.factoryLibraryFixture
        .calls()
        .at(-1)
        ?.path.includes('id=' + 'b'.repeat(32))
    );
    await page.evaluate(() => window.factoryLibraryFixture.release());
    await library.getByRole('heading', {name: 'Retained observations', exact: true}).waitFor();
    assert.equal(await library.getByRole('heading', {name: 'Choose a bounded worker queue', exact: true}).count(), 0);
    assert(await page.evaluate(() => window.factoryLibraryFixture.aborted()));
    // Same transport/repository replacement preserves selections; a new binding resets them.
    await mount('recorded', true);
    await library.getByRole('status').filter({hasText: 'Loading library'}).waitFor();
    await page.evaluate(() => window.factoryLibraryFixture.switchRepository('9223372036854775807'));
    const another = library.getByRole('button', {name: 'Another repository choice', exact: true});
    await another.waitFor();
    await page.evaluate(() => window.factoryLibraryFixture.release());
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    assert.equal(await first.count(), 0);
    assert.equal(
      await library.evaluate((el) => (el as HTMLElement & {context: {repository: string}}).context.repository),
      '9223372036854775807'
    );
    await another.click();
    await library.getByRole('heading', {name: 'Another repository choice', exact: true}).waitFor();
    await library.getByRole('button', {name: 'Open source post', exact: true}).click();
    assert.equal(
      (await page.evaluate(() => window.factoryLibraryFixture.sources())).at(-1)!.repository,
      '9223372036854775807'
    );
    const sources = await page.evaluate(() => window.factoryLibraryFixture.sources().length);
    // A click queued on old DOM before Lit paints the replacement cannot
    // navigate an old source using the newly admitted repository/account.
    await page.evaluate(() => {
      const oldSource = Array.from(document.querySelectorAll<HTMLButtonElement>('soda-factory-library button')).find(
        (button) => button.textContent?.trim() === 'Open source post'
      );
      window.factoryLibraryFixture.switchRepository('9007199254740993');
      oldSource!.click();
    });
    await first.waitFor();
    assert.equal(await page.evaluate(() => window.factoryLibraryFixture.sources().length), sources);
    await first.click();
    await library.getByRole('heading', {name: 'Choose a bounded worker queue', exact: true}).waitFor();
    await page.evaluate(() => {
      const oldSource = Array.from(document.querySelectorAll<HTMLButtonElement>('soda-factory-library button')).find(
        (button) => button.textContent?.trim() === 'Open source post'
      );
      window.factoryLibraryFixture.retireGeneration();
      oldSource!.click();
    });
    await page.waitForFunction(() => !document.querySelector('soda-factory-library')?.textContent);
    assert.equal(await page.evaluate(() => window.factoryLibraryFixture.sources().length), sources);
    for (const retire of ['context', 'generation', 'disconnect'] as const) {
      await mount('recorded', true);
      await library.getByRole('status').filter({hasText: 'Loading library'}).waitFor();
      await page.evaluate((retire) => {
        if (retire === 'context') window.factoryLibraryFixture.retire();
        if (retire === 'generation') window.factoryLibraryFixture.retireGeneration();
        if (retire === 'disconnect') window.factoryLibraryFixture.disconnect();
        window.factoryLibraryFixture.release();
      }, retire);
      await page.waitForFunction(() => !document.querySelector('soda-factory-library')?.textContent);
      assert(await page.evaluate(() => window.factoryLibraryFixture.aborted()));
    }
    await mount('empty');
    await library.getByText('No knowledge or decisions recorded yet.', {exact: false}).waitFor();
    await mount('offline');
    await library.getByRole('alert').waitFor();
    assert.equal(await library.locator('.soda-library-empty').count(), 0);
    const calls = await page.evaluate(() => window.factoryLibraryFixture.calls());
    assert(calls.every((call) => call.method === 'GET'));
    assert(
      calls.every((call) =>
        ['9007199254740993', '9223372036854775807'].includes(
          new URL(call.path, 'https://declared.invalid').searchParams.get('repository')!
        )
      )
    );
    assert((await page.evaluate(() => window.factoryLibraryFixture.navigations())) > 10);
    assert.deepEqual(errors, []);
    assert.deepEqual(assetErrors, []);
  } finally {
    await browser.close();
    server.stop(true);
  }
}

async function standaloneFeedLifetime() {
  const build = await Bun.build({
    entrypoints: [new URL('./fixtures/factory-feed-lifetime-fixture.ts', import.meta.url).pathname],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, build.logs.join('\n'));
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url),
        asset = factoryAssetSource(url.pathname);
      if (asset) return asset;
      if (/^\/role-portraits\/ps1-v1\/[a-z_-]+\.png$/.test(url.pathname))
        return new Response(Bun.file(new URL('../../assets/branding' + url.pathname, import.meta.url)), {
          headers: {'Content-Type': 'image/png'},
        });
      if (url.pathname === '/feed.js')
        return new Response(build.outputs[0]!, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root"></main><script type="module" src="/feed.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    assetErrors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.status() >= 400) assetErrors.push(response.url());
  });
  try {
    await page.goto(server.url.href);
    const feed = page.locator('soda-factory-feed');
    const ready = (owner: string) =>
      feed
        .getByText('Request from ' + owner, {exact: true})
        .first()
        .waitFor();
    const settle = () =>
      page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const mount = async (owner: string, global = false) => {
      await page.evaluate(([owner, combined]) => window.factoryFeedLifetime.mount(owner, combined), [
        owner,
        global,
      ] as const);
      await ready(owner);
    };
    const hold = (prefix: string, mode: 'response' | 'body' | 'reject' = 'response') =>
      page.evaluate(([prefix, mode]) => window.factoryFeedLifetime.hold(prefix, mode), [prefix, mode] as const);
    const held = () => page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
    const refresh = () => feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
    const empty = () => page.waitForFunction(() => !document.querySelector('soda-factory-feed')?.textContent);
    await ready('A');
    // Context replacement aborts a real browser component's old request. A
    // declared peer ignores cancellation so its late payload is also exercised.
    await hold('GET factory/feed?', 'body');
    await refresh();
    await held();
    await page.evaluate(() => window.factoryFeedLifetime.switch('B'));
    await ready('B');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.getByText('Request from A', {exact: true}).count(), 0);
    assert(await page.evaluate(() => window.factoryFeedLifetime.aborted('A', 'factory/feed?')));
    assert(await page.evaluate(() => window.factoryFeedLifetime.aborted('A', '@applications')));

    for (const [ordinal, prefix] of [
      'GET factory/feed?',
      'GET factory/feed/subscription',
      '@catalogue',
      '@applications',
      'GET repositories/7/factory',
    ].entries()) {
      const owner = 'read-' + ordinal;
      await mount(owner);
      await hold(prefix);
      await refresh();
      await held();
      await page.evaluate(() => {
        window.factoryFeedLifetime.retireGeneration();
        window.factoryFeedLifetime.release();
      });
      await empty();
      assert(
        await page.evaluate(
          ([owner, prefix]) => window.factoryFeedLifetime.aborted(owner, prefix.replace('GET ', '')),
          [owner, prefix] as const
        )
      );
    }
    for (const mode of ['response', 'body'] as const) {
      await mount('generation-' + mode);
      await hold('GET factory/feed?', mode);
      await refresh();
      await held();
      await page.evaluate(() => {
        window.factoryFeedLifetime.retireGeneration();
        window.factoryFeedLifetime.release();
      });
      await empty();
    }
    for (const retired of ['context', 'disconnect'] as const) {
      await mount('retire-' + retired);
      await hold('GET factory/feed?');
      await refresh();
      await held();
      await page.evaluate((retired) => {
        if (retired === 'context') window.factoryFeedLifetime.retire();
        else window.factoryFeedLifetime.disconnect();
        window.factoryFeedLifetime.release();
      }, retired);
      await empty();
      assert(
        await page.evaluate((owner) => window.factoryFeedLifetime.aborted(owner, 'factory/feed?'), 'retire-' + retired)
      );
    }
    // Thread and addressed source reads have the same lifetime as feed pages.
    await mount('thread-A');
    await hold('GET factory/feed/posts/' + 'a'.repeat(32) + '/replies?');
    await feed.locator('[data-feed-drawer-toggle]').first().click();
    await held();
    await page.evaluate(() => window.factoryFeedLifetime.switch('thread-B'));
    await ready('thread-B');
    await feed.locator('[data-feed-drawer-toggle]').first().click();
    await feed.getByText('Reply from thread-B', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.getByText('Reply from thread-A', {exact: true}).count(), 0);
    assert(
      await page.evaluate(() =>
        window.factoryFeedLifetime.aborted('thread-A', 'factory/feed/posts/' + 'a'.repeat(32) + '/replies?')
      )
    );
    await mount('source-A');
    await hold('GET factory/feed/posts/' + 'a'.repeat(32), 'body');
    await page.evaluate(() => window.factoryFeedLifetime.selectPost());
    await held();
    await page.evaluate(() => window.factoryFeedLifetime.switch('source-B'));
    await ready('source-B');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.getByText('Request from source-A', {exact: true}).count(), 0);

    // An issued post can have an unknown result. Retirement preserves its
    // immutable pending command and isolates another account's newer writing.
    await mount('post-A');
    let draft = feed.locator('.soda-feed-post-box textarea');
    await draft.fill('A submitted body');
    await hold('POST factory/feed/posts');
    await feed.locator('[data-post-submit="project:7"]').click();
    await held();
    const pending = await page.evaluate(() => window.factoryFeedLifetime.pending('post-A'));
    assert(pending);
    await page.evaluate(() => window.factoryFeedLifetime.switch('post-B'));
    await ready('post-B');
    draft = feed.locator('.soda-feed-post-box textarea');
    await draft.fill('B newer writing');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await draft.inputValue(), 'B newer writing');
    assert.equal(await page.evaluate(() => window.factoryFeedLifetime.pending('post-A')), pending);
    assert.doesNotMatch(await feed.innerText(), /Post saved/);
    await page.evaluate(() => window.factoryFeedLifetime.switch('post-A'));
    await ready('post-A');
    assert.equal(await feed.locator('.soda-feed-post-box textarea').inputValue(), 'A submitted body');
    let sent = await page.evaluate(() =>
      window.factoryFeedLifetime.calls().filter((call) => call.account === 'post-A' && call.method === 'POST')
    );
    assert.equal(sent.length, 1);
    await feed.locator('[data-post-submit="project:7"]').click();
    await feed.getByText('Post saved.', {exact: true}).waitFor();
    sent = await page.evaluate(() =>
      window.factoryFeedLifetime.calls().filter((call) => call.account === 'post-A' && call.method === 'POST')
    );
    assert.equal(sent.length, 2);
    assert.equal(sent[0]!.body, sent[1]!.body);
    assert.equal(await page.evaluate(() => window.factoryFeedLifetime.pending('post-A')), null);

    // A late revision body must not erase A's writing or mutate B's editor.
    await mount('revision-A');
    await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
    await feed.locator('[data-revision-body]').fill('A revised writing');
    await hold('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry', 'body');
    await feed.getByRole('button', {name: 'Save post and retry', exact: true}).click();
    await held();
    const revisionDraft = await page.evaluate(() => window.factoryFeedLifetime.revisionDraft('revision-A'));
    assert(revisionDraft);
    await page.evaluate(() => window.factoryFeedLifetime.switch('revision-B'));
    await ready('revision-B');
    await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
    await feed.locator('[data-revision-body]').fill('B revision writing');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'B revision writing');
    assert.equal(await page.evaluate(() => window.factoryFeedLifetime.revisionDraft('revision-A')), revisionDraft);
    assert.doesNotMatch(await feed.innerText(), /Request updated/);

    // The old revision-load catch used to write into the replacement view.
    await mount('edit-A');
    await hold('GET factory/feed/posts/' + 'a'.repeat(32), 'reject');
    await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
    await held();
    await page.evaluate(() => window.factoryFeedLifetime.switch('edit-B'));
    await ready('edit-B');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.locator('[data-feed-message]').innerText(), '');
    assert.equal(await feed.locator('[data-revision-body]').count(), 0);

    // A retired subscription reply cannot reload its former account.
    await mount('subscription-A', true);
    await feed.locator('summary').filter({hasText: 'Choose repositories'}).click();
    await hold('PUT factory/feed/subscription');
    await feed.getByRole('button', {name: 'Save feed subscription', exact: true}).click();
    await held();
    const prior = await page.evaluate(
      () => window.factoryFeedLifetime.calls().filter((call) => call.account === 'subscription-A').length
    );
    await page.evaluate(() => window.factoryFeedLifetime.switch('subscription-B'));
    await ready('subscription-B');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(
      await page.evaluate(
        () => window.factoryFeedLifetime.calls().filter((call) => call.account === 'subscription-A').length
      ),
      prior
    );
    assert.doesNotMatch(await feed.innerText(), /Followed projects updated/);

    // An old review read never turns into a follow-up write after retirement.
    await page.evaluate(() => {
      window.factoryFeedLifetime.mount('review-A');
      window.factoryFeedLifetime.productChange(true);
    });
    await ready('review-A');
    await hold('GET factory/feed/posts/' + 'a'.repeat(32));
    await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
    await held();
    await page.evaluate(() => {
      window.factoryFeedLifetime.retireGeneration();
      window.factoryFeedLifetime.release();
    });
    await empty();
    assert.equal(
      await page.evaluate(
        () =>
          window.factoryFeedLifetime.calls().filter((call) => call.account === 'review-A' && call.method === 'POST')
            .length
      ),
      0
    );

    // A changed generation is also refused before any new click dispatch.
    await mount('click-A');
    await feed.locator('.soda-feed-post-box textarea').fill('Kept before retirement');
    await page.evaluate(() => window.factoryFeedLifetime.retireGeneration());
    await feed.locator('[data-post-submit="project:7"]').click();
    await empty();
    assert.equal(
      await page.evaluate(
        () =>
          window.factoryFeedLifetime.calls().filter((call) => call.account === 'click-A' && call.method === 'POST')
            .length
      ),
      0
    );
    assert.deepEqual(errors, []);
    assert.deepEqual(assetErrors, []);
  } finally {
    await browser.close();
    server.stop(true);
  }
}

async function artifactPreviews() {
  const build = await Bun.build({
    entrypoints: [new URL('./fixtures/factory-artifact-fixture.ts', import.meta.url).pathname],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, build.logs.join('\n'));
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url),
        asset = factoryAssetSource(url.pathname);
      if (asset) return asset;
      if (/^\/role-portraits\/ps1-v1\/[a-z_-]+\.png$/.test(url.pathname))
        return new Response(Bun.file(new URL('../../assets/branding' + url.pathname, import.meta.url)), {
          headers: {'Content-Type': 'image/png'},
        });
      if (url.pathname === '/artifact.js')
        return new Response(build.outputs[0]!, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root"></main><script type="module" src="/artifact.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    assetErrors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.status() >= 400) assetErrors.push(response.url());
  });
  try {
    await page.goto(server.url.href);
    const artifact = page.locator('soda-feed-artifact'),
      preview = artifact.getByRole('button', {name: 'Preview', exact: true});
    const mount = async (state: Parameters<Window['factoryArtifact']['mount']>[0] = 'text') => {
      await page.evaluate((state) => window.factoryArtifact.mount(state), state);
      await preview.waitFor();
    };
    const settle = () =>
      page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const held = () => page.waitForFunction(() => window.factoryArtifact.held() > 0);
    await preview.click();
    await artifact.locator('pre').filter({hasText: 'Recorded evidence.'}).waitFor();
    assert.match(await artifact.locator('pre').innerText(), /<script>untrusted\(\)<\/script>/);
    assert.equal(await artifact.locator('script').count(), 0);
    let download = artifact.getByRole('link', {name: 'Download evidence.txt', exact: true});
    const firstURL = await download.getAttribute('href');
    assert(firstURL?.startsWith('blob:'));
    assert.equal(await download.getAttribute('download'), 'evidence.txt');
    const initialCalls = await page.evaluate(() => window.factoryArtifact.calls().length);
    await page.evaluate(() => window.factoryArtifact.rebind());
    await settle();
    assert.equal(await download.getAttribute('href'), firstURL);
    await artifact.getByRole('button', {name: 'Hide preview', exact: true}).click();
    assert.equal(await artifact.locator('pre').count(), 0);
    await preview.click();
    await artifact.locator('pre').waitFor();
    assert.equal(await page.evaluate(() => window.factoryArtifact.calls().length), initialCalls);
    await page.evaluate(() => {
      window.factoryArtifact.text('Replacement bytes');
      window.factoryArtifact.replace({
        id: 'b'.repeat(32),
        href: '/api/factory/artifacts/' + 'b'.repeat(32) + '?repository=9007199254740993',
        name: 'replacement.txt',
      });
    });
    await preview.waitFor();
    assert.equal(await artifact.locator('a').count(), 0);
    assert((await page.evaluate(() => window.factoryArtifact.urls())).revoked.includes(firstURL!));
    await preview.click();
    await artifact.locator('pre').filter({hasText: 'Replacement bytes'}).waitFor();

    // Metadata and content cannot be retargeted while an older request is pending.
    for (const mode of ['headers', 'body'] as const) {
      await mount();
      await page.evaluate((mode) => window.factoryArtifact.hold(mode), mode);
      await preview.click();
      await held();
      await page.evaluate(() => {
        window.factoryArtifact.text('New artifact bytes');
        window.factoryArtifact.replace({
          id: 'b'.repeat(32),
          href: '/api/factory/artifacts/' + 'b'.repeat(32) + '?repository=9007199254740993',
          mediaType: 'text/plain',
          name: 'new.txt',
        });
        window.factoryArtifact.release();
      });
      await preview.waitFor();
      await preview.click();
      await artifact.locator('pre').filter({hasText: 'New artifact bytes'}).waitFor();
      assert.equal(await artifact.getByRole('link', {name: 'Download new.txt', exact: true}).count(), 1);
      assert.doesNotMatch(await artifact.innerText(), /Recorded evidence/);
      assert(await page.evaluate(() => window.factoryArtifact.aborted()));
      if (mode === 'body') assert((await page.evaluate(() => window.factoryArtifact.cancelled())) > 0);
    }
    await mount();
    await preview.click();
    await artifact.locator('pre').waitFor();
    const renamedURL = await artifact.locator('a').getAttribute('href');
    await page.evaluate(() => window.factoryArtifact.replace({name: 'renamed.txt'}));
    await preview.waitFor();
    assert.equal(await artifact.locator('a').count(), 0);
    assert((await page.evaluate(() => window.factoryArtifact.urls())).revoked.includes(renamedURL!));

    // A new account/runtime with the same artifact metadata starts with no bytes.
    await mount();
    await page.evaluate(() => window.factoryArtifact.hold('headers'));
    await preview.click();
    await held();
    const oldPeer = await page.evaluate(() => window.factoryArtifact.peerIndex());
    await page.evaluate(() => {
      window.factoryArtifact.nextContext();
      window.factoryArtifact.text('New account bytes');
      window.factoryArtifact.release();
    });
    await preview.waitFor();
    await preview.click();
    await artifact.locator('pre').filter({hasText: 'New account bytes'}).waitFor();
    assert(await page.evaluate((index) => window.factoryArtifact.aborted(index), oldPeer));
    assert.doesNotMatch(await artifact.innerText(), /Recorded evidence/);
    for (const retire of ['signal', 'generation', 'disconnect'] as const) {
      await mount();
      await page.evaluate(() => window.factoryArtifact.hold('body'));
      await preview.click();
      await held();
      const before = await page.evaluate(() => window.factoryArtifact.urls().created.length);
      await page.evaluate((retire) => {
        if (retire === 'signal') window.factoryArtifact.retire();
        if (retire === 'generation') window.factoryArtifact.retireGeneration();
        if (retire === 'disconnect') window.factoryArtifact.disconnect();
        window.factoryArtifact.release();
      }, retire);
      await page.waitForFunction(() => !document.querySelector('soda-feed-artifact')?.textContent);
      assert.equal(await page.evaluate(() => window.factoryArtifact.urls().created.length), before);
      assert(await page.evaluate(() => window.factoryArtifact.aborted()));
    }
    // Retiring during asynchronous text decoding cannot allocate a late Blob URL.
    await mount();
    await page.evaluate(() => window.factoryArtifact.holdDecode());
    await preview.click();
    await page.waitForFunction(() => window.factoryArtifact.decoding() > 0);
    const beforeDecode = await page.evaluate(() => window.factoryArtifact.urls().created.length);
    await page.evaluate(() => {
      window.factoryArtifact.retire();
      window.factoryArtifact.releaseDecode();
    });
    await settle();
    assert.equal(await page.evaluate(() => window.factoryArtifact.urls().created.length), beforeDecode);
    assert.equal(await artifact.locator('a, pre').count(), 0);
    await mount();
    await preview.click();
    await artifact.locator('pre').waitFor();
    const retiredURL = await artifact.locator('a').getAttribute('href');
    await page.evaluate(() => window.factoryArtifact.retireGeneration());
    await artifact.getByRole('button', {name: 'Hide preview', exact: true}).click();
    await settle();
    assert.equal(await artifact.locator('a, pre').count(), 0);
    assert((await page.evaluate(() => window.factoryArtifact.urls())).revoked.includes(retiredURL!));

    // Foreign and noncanonical hints never dispatch a request.
    for (const href of [
      'https://foreign.invalid/evidence',
      '/api/factory/artifacts/' + 'a'.repeat(32) + '?repository=7',
      '/api/factory/artifacts/' + 'a'.repeat(32) + '?repository=9007199254740993&repository=7',
    ]) {
      await mount();
      const before = await page.evaluate(() => window.factoryArtifact.calls().length);
      await page.evaluate((href) => window.factoryArtifact.replace({href}), href);
      await preview.click();
      await artifact.getByRole('alert').filter({hasText: 'does not match'}).waitFor();
      assert.equal(await page.evaluate(() => window.factoryArtifact.calls().length), before);
    }
    // A queued click from the old DOM cannot fetch a replacement artifact.
    await mount();
    const beforeClick = await page.evaluate(() => window.factoryArtifact.calls().length);
    await page.evaluate(() => {
      const old = document.querySelector<HTMLButtonElement>('soda-feed-artifact button')!;
      window.factoryArtifact.replace({name: 'new name.txt'});
      old.click();
    });
    await settle();
    assert.equal(await page.evaluate(() => window.factoryArtifact.calls().length), beforeClick);
    for (const code of [401, 403, 404, 503]) {
      await mount();
      await page.evaluate((code) => window.factoryArtifact.fail(code), code);
      await preview.click();
      await artifact.getByRole('alert').waitFor();
      assert.equal(await artifact.locator('a').count(), 0);
      await page.evaluate(() => window.factoryArtifact.fail(0));
      await preview.click();
      await artifact.locator('pre').waitFor();
    }
    await mount('too-large');
    await preview.click();
    await artifact.getByRole('alert').filter({hasText: 'preview limit'}).waitFor();
    assert.equal(await artifact.locator('a').count(), 0);
    await mount('truncated');
    await preview.click();
    await artifact.locator('pre').waitFor();
    assert.equal((await artifact.locator('pre').innerText()).length, 64 * 1024);
    await artifact.getByText('Showing the first 64 KiB.', {exact: false}).waitFor();
    await mount('image');
    await preview.click();
    await page.waitForFunction(
      () => document.querySelector<HTMLImageElement>('soda-feed-artifact img')?.naturalWidth === 1
    );
    await mount('binary');
    await preview.click();
    await artifact.getByText('This file is available to download.', {exact: true}).waitFor();
    assert.equal(await artifact.locator('pre, img, audio, video').count(), 0);

    // Exercise the actual deadline; a peer that ignores abort cannot keep busy set.
    await mount();
    await page.evaluate(() => window.factoryArtifact.hold('headers'));
    await preview.click();
    await held();
    await artifact.getByRole('alert').filter({hasText: 'timed out'}).waitFor({timeout: 18000});
    assert(await page.evaluate(() => window.factoryArtifact.aborted()));
    await page.evaluate(() => window.factoryArtifact.release());
    await preview.click();
    await artifact.locator('pre').waitFor();

    // Real feed composition passes the scoped lifetime to both a card and reply.
    await page.evaluate(() => window.factoryArtifact.mountFeed());
    const feed = page.locator('soda-factory-feed');
    await feed.getByRole('heading', {name: 'Recorded evidence', exact: true}).waitFor();
    await feed.locator('soda-feed-artifact').getByRole('button', {name: 'Preview', exact: true}).click();
    await feed.locator('soda-feed-artifact pre').waitFor();
    await feed.locator('[data-feed-drawer-toggle]').click();
    await feed.getByRole('heading', {name: 'Reply evidence', exact: true}).waitFor();
    const reply = feed.locator('soda-feed-artifact').last();
    await reply.getByRole('button', {name: 'Preview', exact: true}).click();
    await reply.locator('pre').waitFor();
    const visibleURLs = await feed
      .locator('soda-feed-artifact a')
      .evaluateAll((links) => links.map((link) => link.getAttribute('href')!));
    assert.equal(visibleURLs.length, 2);
    await page.evaluate(() => window.factoryArtifact.retireFeed());
    await settle();
    assert.equal(await feed.locator('soda-feed-artifact a').count(), 0);
    const urls = await page.evaluate(() => window.factoryArtifact.urls());
    assert(visibleURLs.every((url) => urls.revoked.includes(url)));
    assert(urls.created.every((url) => urls.revoked.includes(url)));
    const calls = await page.evaluate(() => window.factoryArtifact.calls());
    assert(calls.every((call) => call.method === 'GET' && call.path.endsWith('?repository=9007199254740993')));
    assert.deepEqual(errors, []);
    assert.deepEqual(assetErrors, []);
  } finally {
    await browser.close();
    server.stop(true);
  }
}

async function artifactHost(t: Parameters<typeof fixture>[0]) {
  const page = await fixture(t);
  await page.evaluate(() => {
    const communication = {
      kind: 'artifact',
      title: 'Host evidence',
      summary: 'Recorded bytes',
      artifacts: [
        {
          id: 'a'.repeat(32),
          name: 'host.txt',
          media_type: 'text/plain',
          href: '/api/factory/artifacts/' + 'a'.repeat(32) + '?repository=7',
        },
      ],
    };
    window.drawerFixture.setReply(async (call) => {
      if (call.url.includes('/api/factory/feed?'))
        return Response.json({
          entries: [{kind: 'record', id: 'b'.repeat(32), repository: '7', seq: 1, payload: {communication}}],
          truncated: false,
        });
      if (call.url.includes('/replies?'))
        return Response.json({
          items: [
            {
              post_id: 'c'.repeat(32),
              at: 1720000000,
              author_name: 'Human',
              communication: {...communication, title: 'Host reply evidence'},
            },
          ],
          next_before: 0,
        });
      if (call.url.includes('/api/factory/artifacts/'))
        return new Response('Native host evidence', {headers: {'Content-Type': 'text/plain'}});
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: ['7'], revision: 1});
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
      return null;
    });
    window.drawerFixture.mountFeed();
  });
  const feed = page.locator('soda-factory-feed');
  await feed.getByRole('heading', {name: 'Host evidence', exact: true}).waitFor();
  await feed.locator('soda-feed-artifact').getByRole('button', {name: 'Preview', exact: true}).click();
  await feed.locator('soda-feed-artifact pre').filter({hasText: 'Native host evidence'}).waitFor();
  await feed.locator('[data-feed-drawer-toggle]').click();
  await feed.getByRole('heading', {name: 'Host reply evidence', exact: true}).waitFor();
  await feed.locator('soda-feed-artifact').last().getByRole('button', {name: 'Preview', exact: true}).click();
  await feed.locator('soda-feed-artifact').last().locator('pre').filter({hasText: 'Native host evidence'}).waitFor();
  const calls = await page.evaluate(() =>
    window.drawerFixture.calls.filter((call) => call.url.includes('/api/factory/artifacts/'))
  );
  assert.equal(calls.length, 2);
  assert(calls.every((call) => call.method === 'GET' && call.url.endsWith('?repository=7')));
}

async function feedAccess(t: Parameters<typeof fixture>[0]) {
  const build = await Bun.build({
    entrypoints: [new URL('./fixtures/factory-feed-lifetime-fixture.ts', import.meta.url).pathname],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, build.logs.join('\n'));
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url),
        asset = factoryAssetSource(url.pathname);
      if (asset) return asset;
      if (/^\/role-portraits\/ps1-v1\/[a-z_-]+\.png$/.test(url.pathname))
        return new Response(Bun.file(new URL('../../assets/branding' + url.pathname, import.meta.url)), {
          headers: {'Content-Type': 'image/png'},
        });
      if (url.pathname === '/feed.js')
        return new Response(build.outputs[0]!, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root"></main><script type="module" src="/feed.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    assets: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.status() >= 400) assets.push(response.url());
  });
  try {
    await page.goto(server.url.href);
    const feed = page.locator('soda-factory-feed'),
      refresh = feed.getByRole('button', {name: 'Refresh feed', exact: true}),
      draft = feed.locator('.soda-feed-post-box textarea'),
      submit = feed.locator('[data-post-submit="project:7"]');
    const ready = (owner: string) =>
      feed
        .getByText('Request from ' + owner, {exact: true})
        .first()
        .waitFor();
    const mount = async (owner: string, combined = false) => {
      await page.evaluate(([owner, combined]) => window.factoryFeedLifetime.mount(owner, combined), [
        owner,
        combined,
      ] as const);
      await ready(owner);
    };
    const denied = () => feed.locator('[data-feed-access]').waitFor();
    const fail = (prefix: string, code: number) =>
      page.evaluate(([prefix, code]) => window.factoryFeedLifetime.fail(prefix, code), [prefix, code] as const);
    const settle = () =>
      page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const calls = (owner: string) =>
      page.evaluate((owner) => window.factoryFeedLifetime.calls().filter((call) => call.account === owner), owner);
    for (const code of [401, 403]) {
      const owner = 'access-' + code;
      await mount(owner);
      await draft.fill('Private unsent writing ' + code);
      await feed.locator('[data-feed-drawer-toggle="post:' + 'a'.repeat(32) + '"]').click();
      await feed.getByText('Reply from ' + owner, {exact: true}).waitFor();
      await fail('GET factory/feed?', code);
      await refresh.click();
      await denied();
      assert.equal(await feed.locator('.soda-feed-post,.soda-feed-reply').count(), 0);
      assert.equal(await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).count(), 0);
      assert.equal(await draft.inputValue(), 'Private unsent writing ' + code);
      assert(await submit.isDisabled());
      await draft.fill('Newer writing while denied');
      assert(await submit.isDisabled());
      const before = (await calls(owner)).length;
      await submit.dispatchEvent('click');
      await settle();
      assert.equal((await calls(owner)).length, before);
      // Transient refresh failure cannot restore write capability or fake empty success.
      await fail('GET factory/feed?', 503);
      await refresh.click();
      await feed.getByRole('heading', {name: 'Activity could not be loaded.', exact: true}).waitFor();
      assert(await submit.isDisabled());
      assert.equal(await feed.getByText('No activity yet.', {exact: true}).count(), 0);
      await fail('GET factory/feed?', 0);
      await refresh.click();
      await ready(owner);
      assert.equal(await feed.locator('[data-feed-access]').count(), 0);
      assert.equal(await draft.inputValue(), 'Newer writing while denied');
      assert(!(await submit.isDisabled()));
      assert.equal((await calls(owner)).filter((call) => call.method !== 'GET').length, 0);
    }
    // A real interval observation proves denial stops automatic polling.
    await mount('quiet');
    await fail('GET factory/feed?', 403);
    await refresh.click();
    await denied();
    const quiet = (await calls('quiet')).length;
    await page.waitForTimeout(10500);
    assert.equal((await calls('quiet')).length, quiet);

    // Cached artifact bytes retire with the withdrawn feed's visibility epoch.
    await page.evaluate(() => {
      window.factoryFeedLifetime.mount('artifact-access');
      window.factoryFeedLifetime.artifacts(true);
    });
    await ready('artifact-access');
    await feed.locator('soda-feed-artifact').getByRole('button', {name: 'Preview', exact: true}).click();
    await feed.locator('soda-feed-artifact pre').waitFor();
    const url = await feed.locator('soda-feed-artifact a').getAttribute('href');
    assert(url);
    await fail('GET factory/feed?', 403);
    await refresh.click();
    await denied();
    assert.equal(await feed.locator('soda-feed-artifact a, soda-feed-artifact pre').count(), 0);
    assert((await page.evaluate(() => window.factoryFeedLifetime.urls())).revoked.includes(url));

    // Delayed reads and writes from the lost epoch cannot repopulate the feed
    // or settle its preserved command after an explicit re-admission.
    await mount('pending');
    await draft.fill('Immutable submitted body');
    await page.evaluate(() => window.factoryFeedLifetime.hold('POST factory/feed/posts'));
    await submit.click();
    await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
    const pending = await page.evaluate(() => window.factoryFeedLifetime.pending('pending'));
    assert(pending);
    await fail('GET factory/feed?', 401);
    await refresh.click();
    await denied();
    await draft.fill('New writing after access loss');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await page.evaluate(() => window.factoryFeedLifetime.pending('pending')), pending);
    assert.equal(await draft.inputValue(), 'New writing after access loss');
    assert(await submit.isDisabled());
    await fail('GET factory/feed?', 0);
    await page.evaluate(() => window.factoryFeedLifetime.hold('GET factory/feed?', 'body'));
    await refresh.click();
    assert(await submit.isDisabled());
    await submit.dispatchEvent('click');
    assert.equal((await calls('pending')).filter((call) => call.method === 'POST').length, 1);
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await ready('pending');
    assert.equal((await calls('pending')).filter((call) => call.method === 'POST').length, 1);
    await submit.click();
    await feed.getByText('Previous post saved. Your newer edits are kept, ready to send.', {exact: true}).waitFor();
    const posts = (await calls('pending')).filter((call) => call.method === 'POST');
    assert.equal(posts.length, 2);
    assert.equal(posts[0]!.body, posts[1]!.body);
    assert.equal(await draft.inputValue(), 'New writing after access loss');
    assert.equal(await page.evaluate(() => window.factoryFeedLifetime.pending('pending')), null);

    await mount('late-read');
    await page.evaluate(() =>
      window.factoryFeedLifetime.hold('GET factory/feed/posts/' + 'a'.repeat(32) + '/replies?')
    );
    await feed.locator('[data-feed-drawer-toggle="post:' + 'a'.repeat(32) + '"]').click();
    await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
    await fail('GET factory/feed?', 403);
    await refresh.click();
    await denied();
    await fail('GET factory/feed?', 0);
    await refresh.click();
    await ready('late-read');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.getByText('Reply from late-read', {exact: true}).count(), 0);
    assert.equal(await feed.locator('[data-feed-access]').count(), 0);

    // Each current conversation/read boundary removes cached private facts.
    for (const boundary of ['thread', 'source', 'edit', 'subscription'] as const) {
      const owner = 'boundary-' + boundary;
      await mount(owner, boundary === 'subscription');
      const prefix =
        boundary === 'subscription'
          ? 'GET factory/feed/subscription'
          : boundary === 'thread'
            ? 'GET factory/feed/posts/' + 'a'.repeat(32) + '/replies?'
            : 'GET factory/feed/posts/' + 'a'.repeat(32);
      await fail(prefix, 403);
      if (boundary === 'thread') await feed.locator('[data-feed-drawer-toggle="post:' + 'a'.repeat(32) + '"]').click();
      if (boundary === 'source') await page.evaluate(() => window.factoryFeedLifetime.selectPost());
      if (boundary === 'edit') await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
      if (boundary === 'subscription') await refresh.click();
      await denied();
      assert.equal(
        await feed.locator('.soda-feed-post,.soda-feed-reply,.soda-feed-subscription,.soda-feed-revision').count(),
        0
      );
    }
    // A session refusal to a command preserves its immutable pending identity.
    await mount('same-account-old-denial');
    await fail('GET factory/feed/posts/' + 'a'.repeat(32), 403);
    await page.evaluate(() => window.factoryFeedLifetime.hold('GET factory/feed/posts/' + 'a'.repeat(32)));
    await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
    await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
    await fail('GET factory/feed?', 401);
    await refresh.click();
    await denied();
    await fail('GET factory/feed?', 0);
    await refresh.click();
    await ready('same-account-old-denial');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.locator('[data-feed-access]').count(), 0);
    await mount('command-denied');
    await draft.fill('Retry exactly after signing in');
    await fail('POST factory/feed/posts', 401);
    await submit.click();
    await denied();
    const command = await page.evaluate(() => window.factoryFeedLifetime.pending('command-denied'));
    assert(command);
    await fail('POST factory/feed/posts', 0);
    await refresh.click();
    await ready('command-denied');
    assert.equal((await calls('command-denied')).filter((call) => call.method === 'POST').length, 1);
    await submit.click();
    await feed.getByText('Post saved.', {exact: true}).waitFor();
    const retry = (await calls('command-denied')).filter((call) => call.method === 'POST');
    assert.equal(retry[0]!.body, retry[1]!.body);
    // An optional owner-only status denial does not revoke a readable feed.
    await mount('owner-status');
    await fail('GET repositories/7/factory', 403);
    await refresh.click();
    await ready('owner-status');
    await settle();
    assert.equal(await feed.locator('[data-feed-access]').count(), 0);
    await draft.fill('Still readable');
    assert(!(await submit.isDisabled()));
    // A retired refusal cannot revoke its replacement account.
    await mount('old-denial');
    await fail('GET factory/feed?', 401);
    await page.evaluate(() => window.factoryFeedLifetime.hold('GET factory/feed?'));
    await refresh.click();
    await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
    await page.evaluate(() => window.factoryFeedLifetime.switch('new-account'));
    await ready('new-account');
    await page.evaluate(() => window.factoryFeedLifetime.release());
    await settle();
    assert.equal(await feed.locator('[data-feed-access]').count(), 0);
    await page.evaluate(() => window.factoryFeedLifetime.story('expired'));
    await feed.locator('[data-feed-access]').filter({hasText: 'session expired'}).waitFor();
    await page.evaluate(() => window.factoryFeedLifetime.story('withdrawn'));
    await feed.getByRole('heading', {name: 'Search is taking shape', exact: true}).waitFor();
    await refresh.click();
    await denied();
    assert.equal(await feed.getByRole('heading', {name: 'Search is taking shape', exact: true}).count(), 0);
    assert.deepEqual(errors, []);
    assert.deepEqual(assets, []);
  } finally {
    await browser.close();
    server.stop(true);
  }

  // Same behavior in Fountain's actual composition and CSS.
  const host = await fixture(t);
  await host.evaluate(() => {
    document.body.dataset.feedAccessCode = '0';
    window.drawerFixture.setReply(async (call) => {
      if (call.url.includes('/api/factory/feed?')) {
        const status = Number(document.body.dataset.feedAccessCode);
        if (status) return Response.json({error: {code: 'denied'}}, {status});
        return Response.json({
          entries: [
            {
              kind: 'post',
              id: 'a'.repeat(32),
              repository: '7',
              seq: 1,
              post: {
                post_id: 'a'.repeat(32),
                repository: '7',
                author_name: 'Human',
                actor: '1',
                at: 1720000000,
                body: 'Native private feed',
              },
            },
          ],
          truncated: false,
        });
      }
      if (call.url.endsWith('/api/factory/feed/subscription'))
        return Response.json({subscriber: '1', repositories: ['7'], revision: 1});
      if (call.url.includes('/api/repositories?q=')) return Response.json({items: [], next_cursor: ''});
      return null;
    });
    window.drawerFixture.mountFeed();
  });
  const native = host.locator('soda-factory-feed');
  await native.getByText('Native private feed', {exact: true}).waitFor();
  await native.getByLabel('New post', {exact: true}).fill('Native unsent writing');
  await host.evaluate(() => {
    document.body.dataset.feedAccessCode = '403';
  });
  await native.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await native.locator('[data-feed-access]').waitFor();
  assert.equal(await native.getByText('Native private feed', {exact: true}).count(), 0);
  assert.equal(await native.getByLabel('New post', {exact: true}).inputValue(), 'Native unsent writing');
  assert(await native.locator('[data-post-submit="project:7"]').isDisabled());
  await host.evaluate(() => {
    document.body.dataset.feedAccessCode = '0';
  });
  await native.getByRole('button', {name: 'Refresh feed', exact: true}).click();
  await native.getByText('Native private feed', {exact: true}).waitFor();
  assert(!(await native.locator('[data-post-submit="project:7"]').isDisabled()));
}

async function feedExamples(t: Parameters<typeof fixture>[0]) {
  const build = await Bun.build({
    entrypoints: [new URL('./fixtures/factory-feed-examples-fixture.ts', import.meta.url).pathname],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, build.logs.join('\n'));
  const script = build.outputs[0]!;
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url),
        asset = factoryAssetSource(url.pathname);
      if (asset) return asset;
      if (/^\/role-portraits\/ps1-v1\/[a-z_-]+\.png$/.test(url.pathname))
        return new Response(Bun.file(new URL('../../assets/branding' + url.pathname, import.meta.url)), {
          headers: {'Content-Type': 'image/png'},
        });
      if (url.pathname === '/examples.js') return new Response(script, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root"></main><script type="module" src="/examples.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    assets: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.status() >= 400) assets.push(response.url());
  });
  try {
    await page.goto(server.url.href);
    const feed = page.locator('soda-factory-feed'),
      draft = feed.locator('.soda-feed-post-box textarea'),
      submit = feed.locator('[data-post-submit="project:7"]');
    const mount = async (state: Parameters<Window['factoryFeedExamples']['mount']>[0]) => {
      await page.evaluate((state) => window.factoryFeedExamples.mount(state), state);
      await feed.getByRole('heading', {name: 'Search is taking shape', exact: true}).waitFor();
    };
    const writes = () =>
      page.evaluate(() => window.factoryFeedExamples.calls().filter((call) => call.method !== 'GET'));
    await mount('paging');
    assert.equal(await feed.locator('.soda-feed-post').count(), 3);
    await feed.getByRole('button', {name: 'Load older entries', exact: true}).click();
    await feed.getByRole('heading', {name: 'Earlier library work 3', exact: true}).waitFor();
    assert.equal(await feed.locator('.soda-feed-post').count(), 6);
    assert.equal(await feed.getByRole('button', {name: 'Load older entries', exact: true}).count(), 0);
    assert(
      (await page.evaluate(() => window.factoryFeedExamples.calls())).some(
        (call) => new URL(call.path, 'https://declared.invalid').searchParams.get('cursor') === 'declared-older-window'
      )
    );
    await feed.getByRole('button', {name: 'Refresh feed', exact: true}).click();
    await feed.getByText('Feed refreshed; no new posts.', {exact: true}).waitFor();
    assert.equal(await feed.locator('.soda-feed-post').count(), 6);
    await feed.getByRole('button', {name: 'Back to latest', exact: true}).click();
    await page.waitForFunction(() => document.querySelectorAll('soda-factory-feed .soda-feed-post').length === 3);
    assert.equal((await writes()).length, 0);

    await mount('replies');
    await feed.locator('[data-feed-drawer-toggle="record:' + 'a'.repeat(32) + '"]').click();
    await feed.getByText('Keyboard navigation is ready for review.', {exact: true}).waitFor();
    const order = () =>
      feed.locator('.soda-feed-thread .soda-feed-reply').evaluateAll((nodes) => nodes.map((node) => node.id));
    assert.deepEqual(await order(), ['post-' + 'e'.repeat(32), 'post-' + 'f'.repeat(32)]);
    await feed.getByRole('button', {name: 'Show more', exact: true}).click();
    await feed.getByText('Keep the full search history.', {exact: true}).waitFor();
    assert.deepEqual(await order(), ['post-' + 'd'.repeat(32), 'post-' + 'e'.repeat(32), 'post-' + 'f'.repeat(32)]);
    await feed.getByRole('button', {name: 'Refresh replies', exact: true}).click();
    await page.waitForFunction(() => document.querySelectorAll('.soda-feed-thread .soda-feed-reply').length === 2);
    const replyKey = 'record:' + 'a'.repeat(32),
      reply = feed.locator('[data-feed-draft="' + replyKey + '"]');
    await reply.fill('Reply through the declared peer');
    await feed.locator('[data-post-submit="' + replyKey + '"]').click();
    await feed.getByText('Reply through the declared peer', {exact: true}).waitFor();
    const replyWrites = await writes();
    assert.equal(replyWrites.length, 1);
    assert.equal(JSON.parse(replyWrites[0]!.body).parent_id, 'a'.repeat(32));

    await mount('busy');
    await draft.fill('Busy submitted body');
    await submit.click();
    assert(await submit.isDisabled());
    await draft.fill('Newer writing while busy');
    assert(await submit.isDisabled());
    // Filtering the view must not retire a command issued by this account.
    await feed.getByRole('button', {name: 'Needs me', exact: true}).click();
    await feed.getByRole('heading', {name: 'No matching posts.', exact: true}).waitFor();
    await page.evaluate(() => window.factoryFeedExamples.release());
    await page.waitForFunction(
      () => !document.querySelector<HTMLButtonElement>('[data-post-submit="project:7"]')?.disabled
    );
    assert.equal(await draft.inputValue(), 'Newer writing while busy');
    assert.equal((await writes()).length, 1);
    assert.deepEqual(await page.evaluate(() => window.factoryFeedExamples.ledger()), {
      created: 1,
      revised: 0,
      commands: 1,
    });
    await feed.getByRole('button', {name: 'All', exact: true}).click();
    await feed.getByText('Busy submitted body', {exact: true}).waitFor();

    for (const state of ['retry', 'unknown'] as const) {
      await mount(state);
      await draft.fill(state + ' submitted body');
      await submit.click();
      await feed
        .locator('[data-feed-message]')
        .filter({hasText: state === 'retry' ? 'Could not save' : 'reply was lost'})
        .waitFor();
      assert.equal((await writes()).length, 1);
      await draft.fill('New writing after ' + state);
      await submit.click();
      await feed.getByText('Previous post saved. Your newer edits are kept, ready to send.', {exact: true}).waitFor();
      const sent = await writes();
      assert.equal(sent.length, 2);
      assert.equal(sent[0]!.body, sent[1]!.body);
      assert.deepEqual(await page.evaluate(() => window.factoryFeedExamples.ledger()), {
        created: 1,
        revised: 0,
        commands: 1,
      });
      assert.equal(await draft.inputValue(), 'New writing after ' + state);
      assert.equal(await feed.getByText(state + ' submitted body', {exact: true}).count(), 1);
    }
    await mount('stale');
    await feed.locator('[data-post-revise="' + 'c'.repeat(32) + '"]').click();
    await feed.locator('[data-revision-body]').fill('Reviewed changed request');
    await feed.getByRole('button', {name: 'Save post and retry', exact: true}).click();
    await feed
      .locator('.soda-feed-revision blockquote')
      .getByText('Current request changed elsewhere.', {exact: true})
      .waitFor();
    assert.equal(await feed.locator('[data-revision-body]').inputValue(), 'Reviewed changed request');
    assert.equal((await page.evaluate(() => window.factoryFeedExamples.ledger())).revised, 0);
    await feed.getByRole('button', {name: 'Save post and retry', exact: true}).click();
    await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
    assert.equal((await page.evaluate(() => window.factoryFeedExamples.ledger())).revised, 1);
    const revisionWrites = await writes();
    assert(revisionWrites.every((call) => call.path.endsWith('/retry')));
    assert.equal(JSON.parse(revisionWrites[0]!.body).expected_revision, 1);
    assert.equal(JSON.parse(revisionWrites[1]!.body).expected_revision, 2);
    // New-post creation is not a revision-conflict operation in this example.
    await draft.fill('A distinct new post');
    await submit.click();
    await feed.getByText('Post saved.', {exact: true}).waitFor();
    assert.equal((await page.evaluate(() => window.factoryFeedExamples.ledger())).created, 1);
    await mount('revision-unknown');
    await feed.locator('[data-post-revise="' + 'c'.repeat(32) + '"]').click();
    const revisionBody = feed.locator('[data-revision-body]');
    await revisionBody.fill('Declared saved revision');
    await feed.getByRole('button', {name: 'Save post and retry', exact: true}).click();
    await feed.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
    await revisionBody.fill('Newer declared writing');
    await feed.getByRole('button', {name: 'Check saved edit', exact: true}).click();
    await feed.getByText('Your newer edits are kept, ready to send.', {exact: false}).waitFor();
    assert.equal(await revisionBody.inputValue(), 'Newer declared writing');
    assert.equal((await writes()).length, 1);
    assert.equal((await page.evaluate(() => window.factoryFeedExamples.ledger())).revised, 1);
    await mount('review-unknown');
    await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
    await feed.getByText('This change could not be opened.', {exact: false}).waitFor();
    await feed.getByRole('button', {name: 'Check saved change review', exact: true}).click();
    await feed.getByText('Change saved for review.', {exact: false}).waitFor();
    assert.equal((await writes()).length, 1);
    assert((await writes())[0]!.path.endsWith('/review'));
    assert.deepEqual(errors, []);
    assert.deepEqual(assets, []);

    // Feed uses Fountain's real host adapter/CSS with the same declared peer.
    const native = await fixture(t);
    native.setDefaultTimeout(5000);
    const nativeErrors: string[] = [];
    native.on('pageerror', (error) => nativeErrors.push(error.message));
    await native.addScriptTag({type: 'module', content: await script.text()});
    await native
      .waitForFunction(() => !!window.factoryFeedExamples)
      .catch(() => {
        throw Error('Native example did not initialize: ' + nativeErrors.join('; '));
      });
    await native.evaluate(() => {
      window.factoryFeedExamples.prepare('unknown');
      document.querySelectorAll('soda-factory-feed').forEach((element) => element.remove());
      window.drawerFixture.setReply(async (call) => {
        const url = new URL(call.url, location.origin);
        const api = url.pathname.indexOf('/api/factory/feed');
        if (api < 0) return null;
        return window.factoryFeedExamples.request(url.pathname.slice(api + '/api/'.length) + url.search, {
          method: call.method,
          ...(call.body ? {body: call.body} : {}),
        });
      });
      window.drawerFixture.mountFeed();
    });
    const host = native.locator('soda-factory-feed');
    await host.getByRole('heading', {name: 'Search is taking shape', exact: true}).waitFor();
    await host.getByLabel('New post', {exact: true}).fill('Native declared unknown submission');
    await host.locator('[data-post-submit="project:7"]').click();
    await host.locator('[data-feed-message]').filter({hasText: 'reply was lost'}).waitFor();
    await host.locator('[data-post-submit="project:7"]').click();
    await host.getByText('Post saved.', {exact: true}).waitFor();
    assert.equal(await host.getByText('Native declared unknown submission', {exact: true}).count(), 1);
    const nativeWrites = await native.evaluate(() =>
      window.factoryFeedExamples.calls().filter((call) => call.method === 'POST')
    );
    assert.equal(nativeWrites.length, 2);
    assert.equal(nativeWrites[0]!.body, nativeWrites[1]!.body);
    assert.deepEqual(await native.evaluate(() => window.factoryFeedExamples.ledger()), {
      created: 1,
      revised: 0,
      commands: 1,
    });
    await native.evaluate(() => {
      window.factoryFeedExamples.prepare('revision-unknown');
      document.querySelectorAll('soda-factory-feed').forEach((element) => element.remove());
      window.drawerFixture.mountFeed();
    });
    await host.getByRole('heading', {name: 'Search is taking shape', exact: true}).waitFor();
    await host.locator('[data-post-revise="' + 'c'.repeat(32) + '"]').click();
    await host.locator('[data-revision-body]').fill('Fountain declared saved revision');
    await host.getByRole('button', {name: 'Save post and retry', exact: true}).click();
    await host.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
    await host.getByRole('button', {name: 'Check saved edit', exact: true}).click();
    await host.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
    assert.equal(
      (await native.evaluate(() => window.factoryFeedExamples.calls().filter((call) => call.method === 'POST'))).length,
      1
    );
    await native.evaluate(() => {
      window.factoryFeedExamples.prepare('review-unknown');
      document.querySelectorAll('soda-factory-feed').forEach((element) => element.remove());
      window.drawerFixture.mountFeed();
    });
    await host.getByRole('button', {name: 'Review proposed change', exact: true}).click();
    await host.getByText('This change could not be opened.', {exact: false}).waitFor();
    await host.getByRole('button', {name: 'Check saved change review', exact: true}).click();
    await host.getByText('Change saved for review.', {exact: false}).waitFor();
    const nativeReviews = await native.evaluate(() =>
      window.factoryFeedExamples.calls().filter((call) => call.method === 'POST')
    );
    assert.equal(nativeReviews.length, 1);
    assert(nativeReviews[0]!.path.endsWith('/review'));
  } finally {
    await browser.close();
    server.stop(true);
  }
}

async function feedEditLifetimes() {
  const build = await Bun.build({
    entrypoints: [new URL('./fixtures/factory-feed-lifetime-fixture.ts', import.meta.url).pathname],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, build.logs.join('\n'));
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url),
        asset = factoryAssetSource(url.pathname);
      if (asset) return asset;
      if (/^\/role-portraits\/ps1-v1\/[a-z_-]+\.png$/.test(url.pathname))
        return new Response(Bun.file(new URL('../../assets/branding' + url.pathname, import.meta.url)), {
          headers: {'Content-Type': 'image/png'},
        });
      if (url.pathname === '/edit.js')
        return new Response(build.outputs[0]!, {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory/styles/base.css"><main class="soda-factory-root"></main><script type="module" src="/edit.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    assets: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.status() >= 400) assets.push(response.url());
  });
  try {
    await page.goto(server.url.href);
    const feed = page.locator('soda-factory-feed'),
      edit = feed.locator('[data-revision-body]'),
      save = feed.getByRole('button', {name: 'Save post and retry', exact: true});
    const mount = async (owner: string, review = false) => {
      await page.evaluate(
        ([owner, review]) => {
          window.factoryFeedLifetime.mount(owner);
          window.factoryFeedLifetime.productChange(review);
          delete document.body.dataset.feedNavigation;
        },
        [owner, review] as const
      );
      await feed
        .getByText('Request from ' + owner, {exact: true})
        .first()
        .waitFor();
    };
    const open = async () => {
      await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
      await edit.waitFor();
    };
    const hold = (path: string) =>
      page.evaluate(
        (path) => window.factoryFeedLifetime.hold('POST factory/feed/posts/' + 'a'.repeat(32) + path),
        path
      );
    const held = () => page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
    const release = () => page.evaluate(() => window.factoryFeedLifetime.release());
    const filter = async () => {
      await feed.getByRole('button', {name: 'Needs me', exact: true}).click();
      await page.waitForFunction(
        () => document.querySelector<HTMLButtonElement>('button[aria-label="Refresh feed"]')?.disabled === false
      );
    };
    const settle = () =>
      page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const calls = (owner: string) =>
      page.evaluate((owner) => window.factoryFeedLifetime.calls().filter((call) => call.account === owner), owner);
    await mount('edit-filter');
    await open();
    await edit.fill('Submitted reviewed edit');
    await hold('/retry');
    await save.click();
    await held();
    await edit.fill('Newer editor writing');
    await filter();
    await release();
    await feed
      .getByText('Request updated. Processing will continue here. Your newer edits are kept, ready to send.', {
        exact: true,
      })
      .waitFor();
    assert.equal(await edit.inputValue(), 'Newer editor writing');
    assert(!(await save.isDisabled()));
    await save.click();
    await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
    let retries = (await calls('edit-filter')).filter((call) => call.path.endsWith('/retry'));
    assert.deepEqual(
      retries.map((call) => JSON.parse(call.body).expected_revision),
      [1, 2]
    );

    await mount('edit-failure');
    await open();
    await edit.fill('Kept through refusal');
    await page.evaluate(() =>
      window.factoryFeedLifetime.fail('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry', 503)
    );
    await hold('/retry');
    await save.click();
    await held();
    await filter();
    await release();
    await feed.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
    assert.equal(await edit.inputValue(), 'Kept through refusal');
    assert(await save.isDisabled());
    assert.equal(await feed.getByRole('button', {name: 'Retry saved edit', exact: true}).count(), 1);
    assert(await page.evaluate(() => window.factoryFeedLifetime.revisionDraft('edit-failure')));

    for (const mismatch of [
      'source-post',
      'source-repository',
      'source-numeric-repository',
      'source-revision',
      'source-unsafe-revision',
    ]) {
      const owner = 'invalid-' + mismatch;
      await mount(owner);
      await page.evaluate((mismatch) => window.factoryFeedLifetime.mismatch(mismatch), mismatch);
      await feed.getByRole('button', {name: 'Edit post and retry', exact: true}).click();
      await feed.getByText('Could not load this request.', {exact: false}).waitFor();
      assert.equal(await edit.count(), 0);
      assert.equal((await calls(owner)).filter((call) => call.method === 'POST').length, 0);
    }
    for (const mismatch of [
      'receipt-post',
      'receipt-command',
      'receipt-repository',
      'receipt-numeric-repository',
      'receipt-expected',
      'receipt-digest',
      'receipt-revision',
      'receipt-numeric-revision',
    ]) {
      const owner = 'invalid-' + mismatch;
      await mount(owner);
      await open();
      await edit.fill('Retained unconfirmed writing');
      const stored = await page.evaluate((owner) => window.factoryFeedLifetime.revisionDraft(owner), owner);
      await page.evaluate((mismatch) => window.factoryFeedLifetime.mismatch(mismatch), mismatch);
      await save.click();
      await feed.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
      assert.equal(await edit.inputValue(), 'Retained unconfirmed writing');
      assert.equal(await page.evaluate((owner) => window.factoryFeedLifetime.revisionDraft(owner), owner), stored);
    }
    // The already-applied draft branch legitimately retains its revision only
    // when the submitted body is unchanged and its scoped receipt confirms it.
    await mount('unchanged-applied');
    await open();
    await page.evaluate(() => window.factoryFeedLifetime.unchanged(true));
    await save.click();
    await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
    assert.equal(await edit.count(), 0);
    await mount('changed-applied');
    await open();
    await edit.fill('Changed body is not an unchanged retry');
    await page.evaluate(() => window.factoryFeedLifetime.unchanged(true));
    await save.click();
    await feed.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
    assert.equal(await edit.inputValue(), 'Changed body is not an unchanged retry');

    // A stale reread cannot promote another scope or revive withdrawn editing.
    await mount('stale-readonly');
    await open();
    await edit.fill('Retained readonly draft');
    await page.evaluate(() => {
      window.factoryFeedLifetime.fail('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry', 409);
      window.factoryFeedLifetime.mismatch('source-readonly');
    });
    await hold('/retry');
    await save.click();
    await held();
    await filter();
    await release();
    await feed.locator('.soda-feed-revision blockquote').waitFor();
    assert.equal(await edit.inputValue(), 'Retained readonly draft');
    assert(await save.isDisabled());
    await mount('stale-foreign');
    await open();
    await edit.fill('Keep foreign read refusal draft');
    await page.evaluate(() => {
      window.factoryFeedLifetime.fail('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry', 409);
      window.factoryFeedLifetime.mismatch('source-repository');
    });
    await save.click();
    await feed.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
    assert.equal(await edit.inputValue(), 'Keep foreign read refusal draft');

    // Closing and reopening the same target while a response is held creates a
    // new editor. Its draft/revision must not be settled by the earlier editor.
    await mount('replacement');
    await open();
    await edit.fill('Earlier issued edit');
    await hold('/retry');
    await save.click();
    await held();
    await feed.getByRole('button', {name: 'Cancel', exact: true}).click();
    await open();
    await edit.fill('Replacement editor writing');
    const replacement = await page.evaluate(() => window.factoryFeedLifetime.revisionDraft('replacement'));
    await release();
    await feed.getByText('An earlier edit finished.', {exact: false}).waitFor();
    assert.equal(await edit.inputValue(), 'Replacement editor writing');
    assert.equal(await page.evaluate(() => window.factoryFeedLifetime.revisionDraft('replacement')), replacement);
    await save.click();
    await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
    retries = (await calls('replacement')).filter((call) => call.path.endsWith('/retry'));
    assert.deepEqual(
      retries.map((call) => JSON.parse(call.body).expected_revision),
      [1, 2]
    );

    await mount('review-filter', true);
    await hold('/review');
    await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
    await held();
    await filter();
    await release();
    await page.waitForFunction(() => document.body.dataset.feedNavigation?.startsWith('/desktop/request/'));
    assert.equal((await calls('review-filter')).filter((call) => call.method === 'POST').length, 1);
    assert.equal(await feed.getByRole('button', {name: 'Continue change review', exact: true}).isDisabled(), false);
    for (const mismatch of [
      'source-post',
      'source-repository',
      'source-revision',
      'review-invalid-intent',
      'review-foreign-intent',
    ]) {
      const owner = 'review-' + mismatch;
      await mount(owner, true);
      await page.evaluate((mismatch) => window.factoryFeedLifetime.mismatch(mismatch), mismatch);
      await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
      await feed.getByText('This change could not be opened.', {exact: false}).waitFor();
      assert.equal(await page.evaluate(() => document.body.dataset.feedNavigation), undefined);
      assert.equal(
        (await calls(owner)).filter((call) => call.method === 'POST').length,
        mismatch === 'review-foreign-intent' ? 1 : 0
      );
    }
    await mount('review-retired', true);
    await hold('/review');
    await feed.getByRole('button', {name: 'Review proposed change', exact: true}).click();
    await held();
    await page.evaluate(() => window.factoryFeedLifetime.switch('review-new-account'));
    await feed.getByText('Request from review-new-account', {exact: true}).waitFor();
    await release();
    await settle();
    assert.equal(await page.evaluate(() => document.body.dataset.feedNavigation), undefined);
    assert.doesNotMatch(await feed.innerText(), /Change saved for review/);
    await feedPendingRevisions(page);
    await feedPendingReviews(page);
    assert.deepEqual(errors, []);
    assert.deepEqual(assets, []);
  } finally {
    await browser.close();
    server.stop(true);
  }
}

async function feedPendingReviews(page: import('playwright').Page) {
  const feed = page.locator('soda-factory-feed');
  const review = feed.getByRole('button', {name: 'Review proposed change', exact: true});
  const recover = feed.getByRole('button', {name: 'Check saved change review', exact: true});
  const retry = feed.getByRole('button', {name: 'Retry saved change review', exact: true});
  const mount = async (owner: string) => {
    await page.evaluate((owner) => {
      window.factoryFeedLifetime.mount(owner);
      window.factoryFeedLifetime.productChange(true);
      delete document.body.dataset.feedNavigation;
    }, owner);
    await feed
      .getByText('Request from ' + owner, {exact: true})
      .first()
      .waitFor();
  };
  const pending = () => page.evaluate(() => window.factoryFeedLifetime.reviewPending());
  const posts = () =>
    page.evaluate(() =>
      window.factoryFeedLifetime.calls().filter((call) => call.method === 'POST' && call.path.endsWith('/review'))
    );
  const lost = async (owner: string) => {
    await mount(owner);
    await page.evaluate(() => window.factoryFeedLifetime.loseReview());
    await review.click();
    await feed.getByText('This change could not be opened.', {exact: false}).waitFor();
    assert(await pending());
  };
  const confirmed = async () => {
    await page.waitForFunction(() => !!document.body.dataset.feedNavigation?.startsWith('/desktop/request/'));
    assert.equal(await pending(), null);
  };
  await lost('review-pending-reload');
  const original = (await pending())!;
  await feed.getByLabel('New post', {exact: true}).fill('Unsent conversation writing survives review recovery');
  await page.reload();
  await page.waitForFunction(() => !!window.factoryFeedLifetime);
  await mount('review-pending-reload');
  assert.equal(await pending(), original);
  assert.equal((await posts()).length, 0);
  await recover.click();
  await confirmed();
  assert.equal((await posts()).length, 0);
  assert.equal(
    await feed.getByLabel('New post', {exact: true}).inputValue(),
    'Unsent conversation writing survives review recovery'
  );

  await mount('review-pending-missing');
  await page.evaluate(() =>
    window.factoryFeedLifetime.fail('POST factory/feed/posts/' + 'a'.repeat(32) + '/review', 503)
  );
  await review.click();
  await feed.getByText('This change could not be opened.', {exact: false}).waitFor();
  const missing = await pending();
  await recover.click();
  await feed.getByText('No recorded review draft was confirmed.', {exact: false}).waitFor();
  assert.equal(await pending(), missing);
  await page.evaluate(() =>
    window.factoryFeedLifetime.fail('POST factory/feed/posts/' + 'a'.repeat(32) + '/review', 0)
  );
  await retry.click();
  await confirmed();
  const attempts = (await posts()).filter((call) => call.account === 'review-pending-missing');
  assert.equal(attempts.length, 2);
  assert.equal(attempts[0]!.body, attempts[1]!.body);
  assert.equal(await page.evaluate(() => window.factoryFeedLifetime.reviewEffects()), 1);

  await mount('review-pending-storage');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault('write'));
  await review.click();
  await feed.getByText('Nothing was submitted.', {exact: false}).waitFor();
  const retained = await page.evaluate(() => window.factoryFeedLifetime.reviewWrites().at(-1)!);
  assert.equal((await posts()).filter((call) => call.account === 'review-pending-storage').length, 0);
  await mount('review-pending-storage');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault(''));
  await retry.click();
  await confirmed();
  assert.equal(
    JSON.parse(retained).pending.revision,
    JSON.parse((await posts()).find((call) => call.account === 'review-pending-storage')!.body).revision
  );

  for (const invalid of ['unknown', 'foreign']) {
    const owner = 'review-pending-' + invalid;
    await mount(owner);
    const bytes = invalid === 'unknown' ? '{"version":99,"keep":true}' : original;
    await page.evaluate((bytes) => localStorage.setItem(window.factoryFeedLifetime.reviewRecordKey(), bytes), bytes);
    await review.click();
    await feed.getByText('The saved change review cannot be read safely.', {exact: false}).waitFor();
    assert.equal(await pending(), bytes);
    assert.equal((await posts()).filter((call) => call.account === owner).length, 0);
  }
  await mount('review-pending-unreadable');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault('read'));
  await review.click();
  await feed.getByText('The saved change review cannot be read safely.', {exact: false}).waitFor();
  assert.equal((await posts()).filter((call) => call.account === 'review-pending-unreadable').length, 0);
  await page.evaluate(() => window.factoryFeedLifetime.storageFault(''));
  await recover.click();
  await feed.getByText('No saved change review was found.', {exact: false}).waitFor();
  for (const mismatch of [
    'source-post',
    'source-repository',
    'source-revision',
    'review-invalid-intent',
    'review-accepted',
    'review-missing-draft',
    'review-invalid-draft',
  ]) {
    const owner = 'review-pending-' + mismatch;
    await lost(owner);
    const bytes = await pending();
    await page.evaluate((mismatch) => window.factoryFeedLifetime.mismatch(mismatch), mismatch);
    await recover.click();
    await feed.getByText('This change could not be opened.', {exact: false}).waitFor();
    assert.equal(await pending(), bytes);
    assert.equal(await page.evaluate(() => document.body.dataset.feedNavigation), undefined);
    assert.equal((await posts()).filter((call) => call.account === owner).length, 1);
  }
  await lost('review-pending-access');
  const withheld = await pending();
  await page.evaluate(() => window.factoryFeedLifetime.reviewAccess(false));
  await recover.click();
  await feed.getByText('Current access to the request owner’s review is required.', {exact: false}).waitFor();
  assert.equal(await pending(), withheld);
  assert.equal(await page.evaluate(() => document.body.dataset.feedNavigation), undefined);
  await lost('review-pending-changed');
  const unchanged = await pending();
  await page.evaluate(() => window.factoryFeedLifetime.advance('A newer source revision'));
  await retry.click();
  await feed.getByText('This change could not be opened.', {exact: false}).waitFor();
  assert.equal(await pending(), unchanged);
  assert.equal((await posts()).filter((call) => call.account === 'review-pending-changed').length, 1);

  await lost('review-pending-retired');
  const retired = await pending();
  await page.evaluate(() => window.factoryFeedLifetime.hold('GET factory/feed/posts/' + 'a'.repeat(32)));
  await recover.click();
  await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
  await page.evaluate(() => window.factoryFeedLifetime.switch('review-pending-other-account'));
  await feed.getByText('Request from review-pending-other-account', {exact: true}).first().waitFor();
  await page.evaluate(() => window.factoryFeedLifetime.release());
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  assert.equal(await page.evaluate(() => window.factoryFeedLifetime.reviewPending('review-pending-retired')), retired);
  assert.equal(await page.evaluate(() => document.body.dataset.feedNavigation), undefined);

  await mount('review-pending-remove');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault('remove'));
  await review.click();
  await page.waitForFunction(() => !!document.body.dataset.feedNavigation);
  assert(await pending());
  await page.reload();
  await page.waitForFunction(() => !!window.factoryFeedLifetime);
  await mount('review-pending-remove');
  assert.equal((await posts()).length, 0);
  await recover.click();
  await confirmed();
  assert.equal((await posts()).length, 0);
  await mount('review-pending-replaced');
  await page.evaluate(() => window.factoryFeedLifetime.hold('POST factory/feed/posts/' + 'a'.repeat(32) + '/review'));
  await review.click();
  await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
  await page.evaluate(() => {
    localStorage.setItem(window.factoryFeedLifetime.reviewRecordKey(), 'replaced:unknown');
    window.factoryFeedLifetime.release();
  });
  await feed.getByText('This change could not be opened.', {exact: false}).waitFor();
  assert.equal(await pending(), 'replaced:unknown');
  assert.equal(await page.evaluate(() => document.body.dataset.feedNavigation), undefined);
}

async function standaloneFactoryApp() {
  const temporary = process.env.TMPDIR ?? resolve(homedir(), '.cache/soda-tests');
  await mkdir(temporary, {recursive: true});
  const productOut = await mkdtemp(resolve(temporary, 'factory-ui-'));
  const product = await buildFactoryUI(productOut);
  const build = await Bun.build({
    entrypoints: [new URL('./fixtures/factory-app-fixture.ts', import.meta.url).pathname],
    plugins: [
      {
        name: 'production-factory-module',
        setup(build) {
          build.onResolve({filter: /factory\/app\/factory-app\.js$/}, () => ({
            path: '/factory-app.js',
            external: true,
          }));
        },
      },
    ],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(build.success, build.logs.join('\n'));
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const path = new URL(request.url).pathname,
        asset = factoryAssetSource(path);
      if (product.files.includes(path.slice(1))) return new Response(Bun.file(resolve(productOut, path.slice(1))));
      if (asset) return asset;
      if (/^\/role-portraits\/ps1-v1\/[a-z_-]+\.png$/.test(path))
        return new Response(Bun.file(new URL('../../assets/branding' + path, import.meta.url)), {
          headers: {'Content-Type': 'image/png'},
        });
      if (path === '/app.js') return new Response(build.outputs[0]!, {headers: {'Content-Type': 'text/javascript'}});
      if (path !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory.css"><div id="root"></div><script type="module" src="/app.js"></script>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    assets: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('response', (response) => {
    if (response.status() >= 400) assets.push(response.url());
  });
  try {
    await page.goto(server.url.href);
    assert.deepEqual(
      await page.evaluate(async () => {
        const product = await import(new URL('/factory-app.js', location.href).href);
        const error = new product.SodaRequestError(403, 'repository_access_denied');
        return {exports: Object.keys(product).sort(), status: error.status, code: error.code};
      }),
      {
        exports: [
          'SodaFactoryApp',
          'SodaFactoryIssue',
          'SodaFactoryIssueComment',
          'SodaFactoryIssueCreate',
          'SodaFactoryIssueEditor',
          'SodaFactoryIssues',
          'SodaRequestError',
        ],
        status: 403,
        code: 'repository_access_denied',
      }
    );

    const app = page.locator('soda-factory-app'),
      nav = app.getByRole('navigation', {name: 'Factory', exact: true});
    const select = (route: Parameters<Window['factoryAppFixture']['select']>[0]) =>
      page.evaluate((route) => window.factoryAppFixture.select(route), route);
    await app.getByRole('heading', {name: 'Factory', exact: true}).waitFor();
    await app.getByRole('heading', {name: 'Search is taking shape', exact: true}).waitFor();
    await app.getByLabel('New post', {exact: true}).fill('Kept while switching application views');
    await app.evaluate((element) => element.setAttribute('scheme', 'dark'));
    await page.waitForFunction(
      () =>
        getComputedStyle(document.querySelector('soda-factory-app > .soda-factory-root')!).backgroundColor ===
        'rgb(16, 16, 16)'
    );
    await app.evaluate((element) => element.setAttribute('scheme', 'light'));
    await page.waitForFunction(
      () =>
        getComputedStyle(document.querySelector('soda-factory-app > .soda-factory-root')!).backgroundColor ===
        'rgb(255, 255, 255)'
    );
    assert.equal(
      await app.getByLabel('New post', {exact: true}).inputValue(),
      'Kept while switching application views'
    );
    await nav.getByRole('button', {name: 'Work', exact: true}).click();
    await app.locator('soda-factory-work-overview').waitFor();
    assert.equal(await app.locator('soda-factory-feed').count(), 0);
    assert(await page.evaluate(() => window.factoryAppFixture.childAborted()));
    await select({kind: 'feed', repository: '7'});
    await app.getByRole('heading', {name: 'Search is taking shape', exact: true}).waitFor();
    assert.equal(
      await app.getByLabel('New post', {exact: true}).inputValue(),
      'Kept while switching application views'
    );
    await nav.getByRole('button', {name: 'New request', exact: true}).click();
    const objective = app.getByLabel('Describe what you want to build', {exact: true});
    await objective.fill('Keep this request while switching views');
    await nav.getByRole('button', {name: 'Feed', exact: true}).click();
    await app.locator('soda-factory-feed').waitFor();
    await nav.getByRole('button', {name: 'New request', exact: true}).click();
    assert.equal(await objective.inputValue(), 'Keep this request while switching views');

    await select({kind: 'work', repository: '7', intent: 'd' + 'a'.repeat(24)});
    await app.getByText('Make release readiness visible.', {exact: true}).waitFor();
    await select({kind: 'work-detail', repository: '7', issue: '12'});
    await app.locator('soda-factory-work-detail').waitFor();
    await page.waitForFunction(() =>
      window.factoryAppFixture.calls().some((call) => call.path === 'repositories/7/factory/issues/12')
    );
    await select({kind: 'plan', repository: '7', issues: ['12']});
    await app.locator('soda-factory-work-plan').waitFor();
    await page.waitForFunction(() =>
      window.factoryAppFixture.calls().some((call) => call.path.endsWith('/plan?limit=5'))
    );
    await select({kind: 'watch', repository: '7', run: 'a'.repeat(32), role: 'soda-coder', issue: '12'});
    await app.getByRole('button', {name: 'Watch run', exact: true}).waitFor();
    assert.equal(await app.locator('.xterm').count(), 0, 'mounting the frame must not attach a stream');
    await nav.getByRole('button', {name: 'Knowledge', exact: true}).click();
    const library = app.locator('soda-factory-library');
    await library.getByRole('button', {name: 'Another repository choice', exact: true}).click();
    await library.getByRole('button', {name: 'Open source post', exact: true}).click();
    await app.getByText('Source for recorded Knowledge', {exact: true}).waitFor();
    assert.equal(await app.locator('soda-factory-library').count(), 0);
    assert(
      (await page.evaluate(() => window.factoryAppFixture.calls())).every((call) => call.method === 'GET'),
      'opening/navigating views cannot submit workflow commands'
    );

    await nav.getByRole('button', {name: 'New issue', exact: true}).click();
    const newIssue = app.locator('soda-factory-issue-create');
    await newIssue.locator('section[aria-busy=false]').waitFor();
    await newIssue.getByRole('textbox', {name: 'Issue title', exact: true}).fill('Retained standalone issue title');
    await newIssue.getByRole('textbox', {name: 'Issue Markdown', exact: true}).fill('Retained standalone description.');
    assert(await newIssue.getByRole('button', {name: 'Create issue', exact: true}).isEnabled());
    await nav.getByRole('button', {name: 'Issues', exact: true}).click();
    await app.locator('soda-factory-issues').waitFor();
    await nav.getByRole('button', {name: 'New issue', exact: true}).click();
    await newIssue.locator('section[aria-busy=false]').waitFor();
    assert.equal(
      await newIssue.getByRole('textbox', {name: 'Issue title', exact: true}).inputValue(),
      'Retained standalone issue title'
    );
    assert.equal(
      await newIssue.getByRole('textbox', {name: 'Issue Markdown', exact: true}).inputValue(),
      'Retained standalone description.'
    );
    await nav.getByRole('button', {name: 'Issues', exact: true}).click();
    const issues = app.locator('soda-factory-issues'),
      issue = app.locator('soda-factory-issue');
    const facts = app.locator('soda-factory-work-journey');
    await issues.getByRole('button', {name: 'Second open issue', exact: true}).waitFor();
    const beforeIssue = await page.evaluate(() => window.factoryAppFixture.calls().length);
    await issues.getByRole('button', {name: 'Second open issue', exact: true}).click();
    await issue.getByRole('article', {name: 'Issue description', exact: true}).waitFor();
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();
    assert.equal(await app.locator('soda-factory-work-detail').count(), 0);
    assert.equal(await nav.getByRole('button', {name: 'Issues', exact: true}).getAttribute('aria-current'), 'page');
    const issueCalls = (await page.evaluate(() => window.factoryAppFixture.calls())).slice(beforeIssue);
    assert.equal(issueCalls[0]!.path, 'factory/collaboration/repositories/7/issues/12');
    assert.equal(issueCalls[1]!.path, 'factory/collaboration/repositories/7/issues/12/comments?limit=20');
    assert.equal(issueCalls[2]!.path, 'repositories/7/factory/issues/12');
    const reference = issue.getByRole('link', {name: 'Reference', exact: true});
    await reference.click();
    await page.waitForFunction(() => window.factoryAppFixture.links().length > 0);

    // Denial of the primary issue clears and aborts independently delayed Factory facts.
    await page.evaluate(() => window.factoryAppFixture.holdFacts());
    await facts.getByRole('button', {name: 'Refresh journey', exact: true}).click();
    await page.waitForFunction(() => window.factoryAppFixture.factsHeld() > 0);
    await page.evaluate(() => window.factoryAppFixture.failIssue(403));
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await issue.getByText('This issue is not available to this account.', {exact: true}).waitFor();
    assert.equal(await facts.innerText(), '');
    await page.evaluate(() => window.factoryAppFixture.releaseFacts());
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    assert.equal(await facts.innerText(), '');
    await page.evaluate(() => window.factoryAppFixture.failIssue(200));
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();
    await issue.evaluate((element) => element.dispatchEvent(new CustomEvent('factory-issue-read', {detail: null})));
    assert.equal(await facts.innerText(), '');
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryAppFixture.factsMode('unavailable'));
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await facts.getByText('Could not load the work journey. Refresh to try again.', {exact: true}).waitFor();
    assert.equal(await issue.getByRole('article', {name: 'Issue description', exact: true}).count(), 1);
    await page.evaluate(() => window.factoryAppFixture.factsMode('empty'));
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await facts.getByText('No work journey has been recorded.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryAppFixture.factsMode('recorded'));
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();

    await page.evaluate(() => window.factoryAppFixture.holdLink());
    await reference.click();
    await nav.getByRole('button', {name: 'Issues', exact: true}).click();
    await issues.getByRole('button', {name: 'Second open issue', exact: true}).waitFor();
    await page.evaluate(() => window.factoryAppFixture.releaseLink());
    assert(await page.evaluate(() => window.factoryAppFixture.linkAborted()));
    await issues.getByRole('button', {name: 'Second open issue', exact: true}).click();
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryAppFixture.holdIssue());
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await page.waitForFunction(() => window.factoryAppFixture.issueHeld() > 0);
    assert.equal(await facts.innerText(), '');
    await select({kind: 'issue', repository: '7', issue: '13'});
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryAppFixture.releaseIssue());
    assert((await issue.innerText()).includes('#13'));
    assert(!(await issue.innerText()).includes('#12'));

    for (const route of [
      {kind: 'issues', repository: '7'},
      {kind: 'issue-new', repository: '7'},
      {kind: 'issue', repository: '7', issue: '12'},
    ] as const) {
      const mismatches = [
        'actor',
        'host',
        'scope',
        'repository',
        'transport',
        ...(route.kind === 'issue' ? ['issue', 'journey-repository', 'journey-issue', 'journey-transport'] : []),
      ];
      for (const mismatch of mismatches) {
        const before = await page.evaluate(() => window.factoryAppFixture.calls().length);
        await page.evaluate(
          ({route, mismatch}) => {
            window.factoryAppFixture.mismatch(mismatch);
            window.factoryAppFixture.select(route);
          },
          {route, mismatch}
        );
        await app.getByText('Could not open this view.', {exact: false}).waitFor();
        assert.equal(await issue.count(), 0);
        assert.equal(await issues.count(), 0);
        assert.equal(await page.evaluate(() => window.factoryAppFixture.calls().length), before);
      }
    }
    await page.evaluate(() => {
      window.factoryAppFixture.mismatch('');
      window.factoryAppFixture.select({kind: 'issue', repository: '7', issue: '12'});
    });
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();
    await page.evaluate(() => {
      window.factoryAppFixture.holdIssue();
    });
    await issue.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await page.waitForFunction(() => window.factoryAppFixture.issueHeld() > 0);
    await page.evaluate(() => window.factoryAppFixture.connect('issue', 'New issue account'));
    await app.getByText('Acting as New issue account', {exact: true}).waitFor();
    await facts.getByText('Phase implement: finished.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryAppFixture.releaseIssue());
    assert((await page.evaluate(() => window.factoryAppFixture.calls())).every((call) => call.method === 'GET'));

    for (const mismatch of ['actor', 'scope', 'transport', 'kind']) {
      await page.evaluate((mismatch) => {
        window.factoryAppFixture.mismatch(mismatch);
        window.factoryAppFixture.select({kind: 'feed', repository: '7'});
      }, mismatch);
      await app.getByText('Could not open this view.', {exact: false}).waitFor();
      assert.equal(await app.locator('soda-factory-feed').count(), 0);
    }
    for (const mismatch of ['repository', 'issue']) {
      await page.evaluate((mismatch) => {
        window.factoryAppFixture.mismatch(mismatch);
        window.factoryAppFixture.select({kind: 'work-detail', repository: '7', issue: '12'});
      }, mismatch);
      await app.getByText('Could not open this view.', {exact: false}).waitFor();
      assert.equal(await app.locator('soda-factory-work-detail').count(), 0);
    }
    await page.evaluate(() => {
      window.factoryAppFixture.mismatch('');
      window.factoryAppFixture.unavailable(true);
      window.factoryAppFixture.select({kind: 'library', repository: '7'});
    });
    await app.getByText('Could not open this view.', {exact: false}).waitFor();
    await page.evaluate(() => window.factoryAppFixture.unavailable(false));
    await app.getByRole('button', {name: 'Refresh view', exact: true}).click();
    await app
      .locator('soda-factory-library')
      .getByRole('button', {name: 'Another repository choice', exact: true})
      .waitFor();

    await page.evaluate(() => {
      window.factoryAppFixture.hold();
      window.factoryAppFixture.select({kind: 'request'});
    });
    await page.waitForFunction(() => window.factoryAppFixture.held() > 0);
    assert.equal(await app.locator('soda-intent').count(), 0);
    await page.evaluate(() => window.factoryAppFixture.connect('ready', 'Riley'));
    await app.getByText('Acting as Riley', {exact: true}).waitFor();
    await app.locator('soda-factory-feed').waitFor();
    assert(await page.evaluate(() => window.factoryAppFixture.aborted()));
    await page.evaluate(() => window.factoryAppFixture.release());
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    assert.equal(await app.locator('soda-intent').count(), 0);
    assert.equal(await app.getByText('Acting as Alex', {exact: true}).count(), 0);
    await nav.getByRole('button', {name: 'New request', exact: true}).click();
    assert.equal(await objective.inputValue(), '', 'another account must not load the first account draft');
    const beforeInvalid = await page.evaluate(() => window.factoryAppFixture.calls().length);
    await page.evaluate(() =>
      window.factoryAppFixture.select({kind: 'work-detail', repository: '7', issue: 12 as unknown as string})
    );
    await app.getByText('The Factory account or destination is unavailable.', {exact: true}).waitFor();
    assert.equal(await app.locator('soda-intent').count(), 0);
    assert.equal(await page.evaluate(() => window.factoryAppFixture.calls().length), beforeInvalid);
    await page.evaluate(() => window.factoryAppFixture.connect('ready', 'Riley'));
    await app.locator('soda-factory-feed').waitFor();
    await page.evaluate(() => window.factoryAppFixture.retireGeneration());
    await nav.getByRole('button', {name: 'Feed', exact: true}).click();
    await app.getByText('Choose an account to open Factory.', {exact: true}).waitFor();
    assert.equal(await app.locator('soda-intent').count(), 0);
    await page.evaluate(() => window.factoryAppFixture.mount('disconnected'));
    assert.equal(await app.locator('main').getAttribute('aria-busy'), 'false');
    await page.evaluate(() => {
      window.factoryAppFixture.connect('ready', 'Unconnected view');
      window.factoryAppFixture.retireBeforeConnect();
    });
    await app.getByText('Choose an account to open Factory.', {exact: true}).waitFor();
    assert(
      await page.evaluate(() => window.factoryAppFixture.preConnectAborted()),
      'retirement must abort reads even before the child connects'
    );
    // The emitted production frame owns typed creation navigation as well as
    // the Fountain route join. This is a declared product API peer.
    await page.evaluate(async () => {
      const product: typeof import('@factory/product/frontend/factory/app/browser.js') = await import(
        new URL('/factory-app.js', location.href).href
      );
      const view = document.querySelector('soda-factory-app') as InstanceType<typeof product.SodaFactoryApp>;
      const repository = '9007199254740993',
        actor = '9007199254740995',
        host = 'a'.repeat(32),
        scope = 'standalone:created-issue';
      const transport: import('@factory/product/frontend/factory/app/browser.js').FactoryTransport = {
        generation: 'declared-create-frame',
        assetBase: '/',
        async request(path, init) {
          if (path.endsWith('/capabilities'))
            return Response.json({
              repository,
              actor,
              issue: null,
              pull: null,
              head: null,
              base: null,
              observation: {state: 'available', revision: 'declared-creation', observed_at: '2026-11-03T12:00:00Z'},
              actions: {'issue.create': {state: 'allowed'}},
            });
          const input = JSON.parse(String(init?.body)) as {command_id: string};
          document.body.dataset.standaloneIssuePosts = String(
            Number(document.body.dataset.standaloneIssuePosts ?? '0') + 1
          );
          return Response.json({
            command_id: input.command_id,
            observation: 'found',
            effect: 'committed',
            cancellation: 'none',
            completion: 'complete',
            finished: true,
            issue: {kind: 'issue', repository, external_id: 'native-created', number: '9223372036854775806'},
          });
        },
      };
      const context: import('@factory/product/frontend/factory/app/browser.js').FactoryAppContext = {
        transport,
        actor,
        host,
        scope,
        accountLabel: 'Creation peer',
        route: {kind: 'issue-new', repository},
        async resolve(route) {
          if (route.kind !== 'issue-new') throw Error('Declared route unavailable');
          return {
            kind: 'issue-new',
            context: {transport, repository, actor, host, scope, commandScope: async () => scope},
          };
        },
        navigate(route) {
          document.body.dataset.standaloneIssueNavigation = JSON.stringify(route);
        },
      };
      window.addEventListener('remount-standalone-creation', () => {
        view.context = undefined;
        view.context = context;
      });
      view.context = context;
    });
    const creation = app.locator('soda-factory-issue-create');
    await creation.locator('section[aria-busy=false]').waitFor();
    await creation.getByRole('textbox', {name: 'Issue title', exact: true}).fill('Standalone creation');
    await creation
      .getByRole('textbox', {name: 'Issue Markdown', exact: true})
      .fill('Preserve the exact returned issue.');
    await creation.getByRole('button', {name: 'Create issue', exact: true}).click();
    await page.waitForFunction(() => !!document.body.dataset.standaloneIssueNavigation);
    assert.deepEqual(await page.evaluate(() => JSON.parse(document.body.dataset.standaloneIssueNavigation!)), {
      kind: 'issue',
      repository: '9007199254740993',
      issue: '9223372036854775806',
    });
    await page.evaluate(() => window.dispatchEvent(new Event('remount-standalone-creation')));
    await creation.locator('section[aria-busy=false]').waitFor();
    await creation.getByRole('button', {name: 'Open created issue', exact: true}).waitFor();
    assert.equal(await page.evaluate(() => document.body.dataset.standaloneIssuePosts), '1');
    await page.evaluate(async () => {
      const product: typeof import('@factory/product/frontend/factory/app/browser.js') = await import(
        new URL('/factory-app.js', location.href).href
      );
      const view = document.querySelector('soda-factory-app') as InstanceType<typeof product.SodaFactoryApp>;
      const repository = '9007199254740993',
        actor = '9007199254740995',
        host = 'a'.repeat(32),
        issue = '9223372036854775806',
        scope = 'standalone:issue-editor';
      const ref = {kind: 'issue', repository, number: issue, external_id: 'native-editor-42'},
        observation = {state: 'available', revision: 'declared-standalone-editor', observed_at: '2026-11-03T12:00:00Z'};
      const current = {title: 'Original standalone title', body: 'Original standalone body.'};
      let mismatch = '';
      const transport: import('@factory/product/frontend/factory/app/browser.js').FactoryTransport = {
        generation: 'standalone-editor',
        assetBase: '/',
        async request(path, init) {
          document.body.dataset.standaloneEditorCalls = String(
            Number(document.body.dataset.standaloneEditorCalls ?? '0') + 1
          );
          if (path.endsWith('/edits')) {
            const input = JSON.parse(String(init?.body)) as {command_id: string; title?: string; body?: string};
            document.body.dataset.standaloneEditPayload = String(init?.body);
            if (input.title !== undefined) current.title = input.title;
            if (input.body !== undefined) current.body = input.body;
            return Response.json({
              command_id: input.command_id,
              observation: 'found',
              effect: 'committed',
              cancellation: 'none',
              completion: 'complete',
              finished: true,
            });
          }
          if (path.endsWith('/capabilities'))
            return Response.json({
              repository,
              actor,
              issue: ref,
              pull: null,
              head: null,
              base: null,
              observation,
              actions: {'issue.edit': {state: 'allowed'}},
            });
          if (path.includes('/comments?')) return Response.json({observation, items: [], next: '', complete: true});
          return Response.json({
            ref,
            observation,
            ...current,
            state: 'open',
            author: null,
            assignees: [],
            labels: [],
            created_at: null,
            updated_at: null,
            closed_at: null,
            capabilities: null,
          });
        },
      };
      const context: import('@factory/product/frontend/factory/app/browser.js').FactoryAppContext = {
        transport,
        actor,
        host,
        scope,
        accountLabel: 'Editor peer',
        route: {kind: 'issue', repository, issue},
        async resolve() {
          const editor = {transport, actor, host, scope, repository, issue, commandScope: async () => scope};
          if (mismatch === 'actor') editor.actor = '2';
          if (mismatch === 'host') editor.host = 'b'.repeat(32);
          if (mismatch === 'scope') editor.scope = 'foreign';
          if (mismatch === 'repository') editor.repository = '8';
          if (mismatch === 'issue') editor.issue = '43';
          if (mismatch === 'transport') editor.transport = {...transport};
          return {
            kind: 'issue',
            context: {
              transport,
              actor,
              host,
              scope,
              repository,
              issue,
              resolveLink: () => undefined,
              openLink: () => {},
            },
            editor,
          };
        },
        navigate: () => {},
      };
      window.addEventListener('standalone-editor-peer', (event) => {
        mismatch = (event as CustomEvent<string>).detail;
        view.context = {...context};
      });
      view.context = context;
    });
    const standaloneEditor = app.locator('soda-factory-issue-editor');
    await standaloneEditor.getByRole('button', {name: 'Edit issue', exact: true}).click();
    await standaloneEditor.locator('section[aria-busy=false]').waitFor();
    await standaloneEditor
      .getByRole('textbox', {name: 'Edit issue title', exact: true})
      .fill('Standalone edited title');
    await standaloneEditor
      .getByRole('textbox', {name: 'Edit issue Markdown', exact: true})
      .fill('Standalone editor body.');
    await standaloneEditor.getByRole('button', {name: 'Save issue', exact: true}).click();
    await app
      .locator('soda-factory-issue')
      .getByRole('heading', {name: 'Standalone edited title', exact: true})
      .waitFor();
    await standaloneEditor.locator('section[aria-busy=false]').waitFor();
    const editPayload = await page.evaluate(
      () =>
        JSON.parse(document.body.dataset.standaloneEditPayload!) as {
          expected: {external_id: string; title: string; body_digest: string};
        }
    );
    assert.equal(editPayload.expected.external_id, 'native-editor-42');
    assert.equal(editPayload.expected.title, 'Original standalone title');
    assert.match(editPayload.expected.body_digest, /^[a-f0-9]{64}$/);
    for (const mismatch of ['actor', 'host', 'scope', 'repository', 'issue', 'transport']) {
      const before = await page.evaluate(() => document.body.dataset.standaloneEditorCalls);
      await page.evaluate(
        (mismatch) => window.dispatchEvent(new CustomEvent('standalone-editor-peer', {detail: mismatch})),
        mismatch
      );
      await app.getByText('Could not open this view.', {exact: false}).waitFor();
      assert.equal(await standaloneEditor.count(), 0);
      assert.equal(await page.evaluate(() => document.body.dataset.standaloneEditorCalls), before);
    }
    assert.deepEqual(errors, []);
    assert.deepEqual(assets, []);
  } finally {
    await browser.close();
    server.stop(true);
    await rm(productOut, {recursive: true, force: true});
  }
}

async function feedPendingRevisions(page: import('playwright').Page) {
  const feed = page.locator('soda-factory-feed'),
    writing = feed.locator('[data-revision-body]');
  const save = feed.getByRole('button', {name: 'Save post and retry', exact: true});
  const recover = feed.getByRole('button', {name: 'Check saved edit', exact: true});
  const retry = feed.getByRole('button', {name: 'Retry saved edit', exact: true});
  const mount = async (owner: string, body = 'Request from ' + owner) => {
    await page.evaluate((owner) => window.factoryFeedLifetime.mount(owner), owner);
    await feed.getByText(body, {exact: true}).first().waitFor();
  };
  const open = async () => {
    const kept = feed.getByRole('button', {name: 'Recover saved edit', exact: true});
    await ((await kept.count()) ? kept : feed.getByRole('button', {name: 'Edit post and retry', exact: true})).click();
    await writing.waitFor();
  };
  const pending = () => page.evaluate(() => window.factoryFeedLifetime.revisionPending());
  const calls = () => page.evaluate(() => window.factoryFeedLifetime.calls());
  const lost = async (owner: string) => {
    await mount(owner);
    await open();
    await writing.fill('Saved before lost reply');
    await page.evaluate(() => window.factoryFeedLifetime.loseRevision());
    await save.click();
    await feed.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
    assert(await pending());
    assert(await save.isDisabled());
  };
  const reload = async (owner: string, body = 'Saved before lost reply') => {
    await page.reload();
    await page.waitForFunction(() => !!window.factoryFeedLifetime);
    await mount(owner, body);
    await open();
    assert.equal(
      (await calls()).filter((call) => call.account === owner && call.method === 'POST').length,
      0,
      'reload cannot retry an edit'
    );
  };
  // Actual browser reload reconstructs the component/module. The declared peer
  // retains its receipt separately; the production pending bytes stay exact.
  await lost('pending-reload');
  const original = (await pending())!;
  await writing.fill('Newer writing after the lost reply');
  await page.evaluate(() => window.factoryFeedLifetime.readonly(true));
  await reload('pending-reload');
  assert.equal(await pending(), original);
  assert.equal(await writing.inputValue(), 'Newer writing after the lost reply');
  await recover.click();
  await feed.getByText('Your newer edits are kept, ready to send.', {exact: false}).waitFor();
  assert.equal(await pending(), null);
  assert.equal(await writing.inputValue(), 'Newer writing after the lost reply');

  // The current head is evidence for the editor only, never a substitute for
  // the old command's receipt or a reason to rebase newer writing backwards.
  await lost('pending-head');
  await writing.fill('Local newer draft');
  await page.evaluate(() => window.factoryFeedLifetime.advance('A newer committed head'));
  await reload('pending-head', 'A newer committed head');
  await recover.click();
  await feed.getByText('Your newer edits are kept, ready to send.', {exact: false}).waitFor();
  await save.click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const last = (await calls())
    .filter((call) => call.account === 'pending-head' && call.path.endsWith('/retry'))
    .at(-1)!;
  assert.equal(JSON.parse(last.body).expected_revision, 3);

  // A retained attempt is explicit and sends its original body, even when the
  // visible draft was edited after a failed durable write and component remount.
  await mount('pending-storage');
  await open();
  await writing.fill('Original captured edit');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault('write'));
  await save.click();
  await feed.getByText('Nothing was sent.', {exact: false}).waitFor();
  const attempted = await page.evaluate(() => window.factoryFeedLifetime.revisionWrites().at(-1)!);
  assert.equal(
    (await calls()).filter((call) => call.account === 'pending-storage' && call.method === 'POST').length,
    0
  );
  await mount('pending-storage');
  await open();
  await writing.fill('Newer storage-failure writing');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault(''));
  await retry.click();
  await feed.getByText('Your newer edits are kept, ready to send.', {exact: false}).waitFor();
  const storageCall = (await calls()).find((call) => call.account === 'pending-storage' && call.method === 'POST')!;
  assert.equal(JSON.parse(storageCall.body).body, 'Original captured edit');
  assert.equal(JSON.parse(storageCall.body).command_id, JSON.parse(attempted).pending.command);
  assert.equal(await writing.inputValue(), 'Newer storage-failure writing');

  for (const invalid of ['unknown', 'foreign']) {
    const owner = 'pending-' + invalid;
    await mount(owner);
    const bytes = invalid === 'unknown' ? '{"version":99,"do_not_replace":true}' : original;
    await page.evaluate((bytes) => localStorage.setItem(window.factoryFeedLifetime.revisionRecordKey(), bytes), bytes);
    await open();
    await writing.fill('Keep blocked writing');
    await recover.click();
    await feed.getByText('The saved edit cannot be read safely.', {exact: false}).waitFor();
    assert.equal(await pending(), bytes);
    assert(await save.isDisabled());
    assert.equal((await calls()).filter((call) => call.account === owner && call.method !== 'GET').length, 0);
  }
  await mount('pending-unreadable');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault('read'));
  await open();
  assert(await save.isDisabled());
  await writing.fill('Keep unreadable writing');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault(''));
  await recover.click();
  await feed.getByText('No saved edit was found.', {exact: false}).waitFor();
  assert(!(await save.isDisabled()));

  for (const mismatch of [
    'receipt-command',
    'receipt-post',
    'receipt-repository',
    'receipt-author',
    'receipt-expected',
    'receipt-digest',
    'receipt-revision',
  ]) {
    await lost('pending-' + mismatch);
    const before = await pending();
    await page.evaluate((mismatch) => window.factoryFeedLifetime.mismatch(mismatch), mismatch);
    await recover.click();
    await feed.getByText('The saved edit could not be confirmed.', {exact: false}).waitFor();
    assert.equal(await pending(), before);
    assert.equal(await writing.inputValue(), 'Saved before lost reply');
  }
  await lost('pending-retired');
  const retired = await pending();
  await page.evaluate(() => window.factoryFeedLifetime.hold('GET factory/feed/posts/' + 'a'.repeat(32) + '/retry/'));
  await recover.click();
  await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
  await page.evaluate(() => window.factoryFeedLifetime.switch('pending-other-account'));
  await feed.getByText('Request from pending-other-account', {exact: true}).waitFor();
  await page.evaluate(() => window.factoryFeedLifetime.release());
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  assert.equal(await page.evaluate(() => window.factoryFeedLifetime.revisionPending('pending-retired')), retired);
  assert.equal(await feed.getByRole('button', {name: 'Recover saved edit', exact: true}).count(), 0);

  await mount('pending-missing');
  await open();
  await writing.fill('An uncommitted saved edit');
  await page.evaluate(() =>
    window.factoryFeedLifetime.fail('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry', 503)
  );
  await save.click();
  await feed.getByText('The edit could not be confirmed.', {exact: false}).waitFor();
  const missing = await pending();
  await recover.click();
  await feed.getByText('The saved edit could not be confirmed.', {exact: false}).waitFor();
  assert.equal(await pending(), missing);
  await page.evaluate(() => window.factoryFeedLifetime.fail('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry', 0));
  await retry.click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  const attempts = (await calls()).filter((call) => call.account === 'pending-missing' && call.method === 'POST');
  assert.equal(attempts.length, 2);
  assert.equal(attempts[0]!.body, attempts[1]!.body);

  // Failed removal can resurrect the durable bytes after reload, but only a
  // read of their original receipt may settle them; no second effect is sent.
  await mount('pending-remove');
  await open();
  await writing.fill('Retained removal failure');
  await page.evaluate(() => window.factoryFeedLifetime.storageFault('remove'));
  await save.click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  assert(await pending());
  await reload('pending-remove', 'Retained removal failure');
  await recover.click();
  await feed.getByText('Request updated. Processing will continue here.', {exact: true}).waitFor();
  assert.equal(await pending(), null);

  await mount('pending-replaced');
  await open();
  await writing.fill('Never erase replacement bytes');
  await page.evaluate(() => window.factoryFeedLifetime.hold('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry'));
  await save.click();
  await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
  await page.evaluate(() =>
    localStorage.setItem(window.factoryFeedLifetime.revisionRecordKey(), 'replacement:unknown')
  );
  await page.evaluate(() => window.factoryFeedLifetime.release());
  await feed.getByText('The saved edit changed.', {exact: false}).waitFor();
  assert.equal(await pending(), 'replacement:unknown');
  assert.equal(await writing.inputValue(), 'Never erase replacement bytes');
  await mount('pending-draft-read');
  await open();
  await writing.fill('Issued before draft read failed');
  await page.evaluate(() => window.factoryFeedLifetime.hold('POST factory/feed/posts/' + 'a'.repeat(32) + '/retry'));
  await save.click();
  await page.waitForFunction(() => window.factoryFeedLifetime.held() > 0);
  await page.evaluate(() => {
    localStorage.setItem(
      'soda-feed-revision:v2:declared:feed:pending-draft-read:7:' + 'a'.repeat(32),
      JSON.stringify({body: 'Newer writing hidden by read failure'})
    );
    window.factoryFeedLifetime.storageFault('draft-read');
    window.factoryFeedLifetime.release();
  });
  await feed.getByText('Your newer edits are kept, ready to send.', {exact: false}).waitFor();
  await page.evaluate(() => window.factoryFeedLifetime.storageFault(''));
  assert.equal(
    JSON.parse((await page.evaluate(() => window.factoryFeedLifetime.revisionDraft()))!).body,
    'Newer writing hidden by read failure'
  );
}

type IssueListPeer = ReturnType<
  typeof import('@factory/product/frontend/storybook/issues-fixtures.js').issuesStoryPeer
>;
type IssueListStoryState = import('@factory/product/frontend/storybook/issues-fixtures.js').IssueListStoryState;
declare global {
  interface Window {
    factoryIssueList: {
      view: import('@factory/product/frontend/factory/issues/issues.js').SodaFactoryIssues;
      peer: IssueListPeer;
      peers: IssueListPeer[];
      mount(state?: IssueListStoryState, repository?: string): void;
    };
  }
}

test('shared issue list uses production assets for filtered paging, scoped navigation and retired reads', async () => {
  const temporary = process.env.TMPDIR ?? resolve(homedir(), '.cache/soda-tests');
  await mkdir(temporary, {recursive: true});
  const out = await mkdtemp(resolve(temporary, 'issue-list-ui-'));
  const product = await buildFactoryUI(out);
  assert(product.sources.includes('frontend/factory/issues/issues.ts'));
  assert(product.sources.every((path) => !path.includes('/storybook/') && !path.endsWith('.stories.ts')));
  const peer = await Bun.build({
    entrypoints: [new URL(import.meta.resolve('@factory/product/frontend/storybook/issues-fixtures.ts')).pathname],
    target: 'browser',
    format: 'esm',
  });
  assert(peer.success, peer.logs.join('\n'));
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const path = new URL(request.url).pathname;
      if (product.files.includes(path.slice(1))) return new Response(Bun.file(resolve(out, path.slice(1))));
      if (path === '/peer.js') return new Response(peer.outputs[0], {headers: {'Content-Type': 'text/javascript'}});
      if (path !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory.css"><main id="root" class="soda-factory-root" data-factory-scheme="light"></main>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  try {
    await page.goto(server.url.href);
    await page.evaluate(async () => {
      const product = await import(new URL('/factory-app.js', location.href).href);
      const declared: typeof import('@factory/product/frontend/storybook/issues-fixtures.js') = await import(
        new URL('/peer.js', location.href).href
      );
      const view = new product.SodaFactoryIssues(),
        peer = declared.issuesStoryPeer();
      window.factoryIssueList = {
        view,
        peer,
        peers: [peer],
        mount(state = 'ready', repository) {
          const fixture = window.factoryIssueList;
          fixture.peer = declared.issuesStoryPeer(state, repository);
          fixture.peers.push(fixture.peer);
          const root = document.querySelector('#root')!;
          if (fixture.view.parentElement !== root) root.replaceChildren(fixture.view);
          fixture.view.context = fixture.peer.context;
        },
      };
      window.factoryIssueList.mount();
    });
    const list = page.locator('soda-factory-issues'),
      rows = list.locator('ul > li');
    const title = list.getByRole('button', {name: 'Keep issue context across views', exact: true});
    await title.waitFor();
    assert.equal(await rows.count(), 2);
    assert((await rows.first().innerText()).includes('#9223372036854775806'));
    assert((await rows.first().innerText()).includes('Assigned to alex'));
    assert.equal(await rows.first().locator('time').count(), 2);
    await title.focus();
    await page.keyboard.press('Enter');
    assert.deepEqual(await page.evaluate(() => window.factoryIssueList.peer.navigated), [
      {kind: 'issue', repository: '9007199254740993', external_id: '101', number: '9223372036854775806'},
    ]);
    await list.getByLabel('Issue state', {exact: true}).selectOption('closed');
    await list.getByRole('button', {name: 'Closed issue', exact: true}).waitFor();
    assert.equal(await rows.count(), 1);
    await list.getByLabel('Issue state', {exact: true}).selectOption('all');
    await list.getByText('State unavailable', {exact: true}).waitFor();
    assert.equal(await rows.count(), 4);
    await page.evaluate(() => window.factoryIssueList.mount('paged'));
    await title.waitFor();
    assert.equal(await rows.count(), 1);
    await list.getByRole('button', {name: 'Next page', exact: true}).click();
    await list.getByRole('button', {name: 'Second open issue', exact: true}).waitFor();
    assert.equal(await rows.count(), 1);
    assert(await list.getByRole('button', {name: 'Next page', exact: true}).isDisabled());
    assert.equal(
      await page.evaluate(() =>
        new URL(window.factoryIssueList.peer.calls.at(-1)!.path, location.href).searchParams.get('cursor')
      ),
      'opaque:/next?scope=declared&v=1'
    );
    await list.getByRole('button', {name: 'Previous page', exact: true}).click();
    await title.waitFor();
    assert(await list.getByRole('button', {name: 'Previous page', exact: true}).isDisabled());
    await list.getByRole('button', {name: 'Next page', exact: true}).click();
    await list.getByRole('button', {name: 'Second open issue', exact: true}).waitFor();
    await list.getByLabel('Issue state', {exact: true}).selectOption('closed');
    await list.getByRole('button', {name: 'Closed issue', exact: true}).waitFor();
    assert(!(await page.evaluate(() => window.factoryIssueList.peer.calls.at(-1)!.path.includes('cursor='))));
    assert(await list.getByRole('button', {name: 'Previous page', exact: true}).isDisabled());

    for (const code of [401, 403, 404, 503]) {
      await page.evaluate(() => window.factoryIssueList.mount());
      await title.waitFor();
      await page.evaluate((code) => window.factoryIssueList.peer.setStatus(code), code);
      await list.getByRole('button', {name: 'Refresh issues', exact: true}).click();
      await list
        .getByRole('status')
        .filter({hasText: code === 503 ? 'Issues could not be loaded' : 'Issues are not available to this account'})
        .waitFor();
      assert.equal(await rows.count(), 0);
      await page.evaluate(() => window.factoryIssueList.peer.setStatus(200));
      await list.getByRole('button', {name: 'Refresh issues', exact: true}).click();
      await title.waitFor();
    }
    for (const mismatch of ['repository', 'actor-host', 'numeric-id', 'duplicate', 'filter', 'cursor', 'observation']) {
      await page.evaluate(() => window.factoryIssueList.mount());
      await title.waitFor();
      await page.evaluate((value) => window.factoryIssueList.peer.mismatch(value), mismatch);
      await list.getByRole('button', {name: 'Refresh issues', exact: true}).click();
      await list.getByText('Issues could not be loaded. Try again.', {exact: true}).waitFor();
      assert.equal(await rows.count(), 0, mismatch);
    }
    await page.evaluate(() => window.factoryIssueList.mount('missing'));
    await list.getByText('Assignees unavailable', {exact: true}).waitFor();
    await list.getByText('Labels unavailable', {exact: true}).waitFor();
    assert((await rows.first().innerText()).includes('Unknown author'));
    assert.equal(await rows.first().locator('time').count(), 0);
    await page.evaluate(() => window.factoryIssueList.mount('plain-text'));
    await list.getByRole('button', {name: '<img src=x onerror=alert(1)> literal issue title', exact: true}).waitFor();
    assert.equal(await list.locator('img, script, [style]').count(), 0);
    assert((await rows.first().innerText()).includes('<script>alert(1)</script>'));
    await page.evaluate(() => window.factoryIssueList.mount('empty'));
    await list.getByText('No issues in this view.', {exact: true}).waitFor();
    assert(await list.getByRole('button', {name: 'Next page', exact: true}).isDisabled());

    await page.evaluate(() => window.factoryIssueList.mount('loading'));
    await list.getByText('Loading issues…', {exact: true}).waitFor();
    await page.waitForFunction(() => window.factoryIssueList.peer.held() > 0);
    await page.evaluate(() => window.factoryIssueList.mount('ready', '43'));
    await list.getByRole('button', {name: 'Another repository issue', exact: true}).waitFor();
    assert(await page.evaluate(() => window.factoryIssueList.peers.some((peer) => peer.aborted())));
    await page.evaluate(() => window.factoryIssueList.peers.forEach((peer) => peer.release()));
    await list.getByRole('button', {name: 'Another repository issue', exact: true}).waitFor();
    assert.equal(await title.count(), 0);

    await page.evaluate(() => window.factoryIssueList.mount('loading'));
    await page.waitForFunction(() => window.factoryIssueList.peer.held() > 0);
    await list.getByLabel('Issue state', {exact: true}).selectOption('closed');
    await page.waitForFunction(() => window.factoryIssueList.peer.held() === 2);
    await page.evaluate(() => window.factoryIssueList.peer.release());
    await list.getByRole('button', {name: 'Closed issue', exact: true}).waitFor();
    assert.equal(await title.count(), 0);

    await page.evaluate(() => window.factoryIssueList.mount());
    await title.waitFor();
    await page.evaluate(() => window.factoryIssueList.peer.holdOpen());
    await title.click();
    await page.waitForFunction(() => window.factoryIssueList.peer.opened.length === 1);
    const pendingNavigationPeer = await page.evaluate(() => window.factoryIssueList.peers.length - 1);
    await page.evaluate(() => window.factoryIssueList.mount('ready', '43'));
    await page.evaluate(() => window.factoryIssueList.peers.forEach((peer) => peer.releaseOpen()));
    assert(await page.evaluate((index) => window.factoryIssueList.peers[index]!.openAborted(), pendingNavigationPeer));
    assert.equal(
      await page.evaluate((index) => window.factoryIssueList.peers[index]!.navigated.length, pendingNavigationPeer),
      0
    );
    await list.getByRole('button', {name: 'Another repository issue', exact: true}).waitFor();
    await page.evaluate(() => window.factoryIssueList.peer.retireGeneration());
    await list.getByRole('button', {name: 'Refresh issues', exact: true}).click();
    await page.waitForFunction(() => !document.querySelector('soda-factory-issues')?.textContent);
    await page.evaluate(() => window.factoryIssueList.mount());
    await title.waitFor();
    await page.evaluate(() => window.factoryIssueList.peer.failOpen(true));
    await title.click();
    await list.getByText('This issue could not be opened. Try again.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryIssueList.peer.failOpen(false));
    await title.click();
    await page.waitForFunction(() => window.factoryIssueList.peer.navigated.length === 1);

    await page.setViewportSize({width: 360, height: 800});
    assert(await list.evaluate((element) => element.scrollWidth <= document.documentElement.clientWidth));
    if (process.env.SODA_ISSUE_LIST_EVIDENCE_DIR) {
      await mkdir(process.env.SODA_ISSUE_LIST_EVIDENCE_DIR, {recursive: true});
      await page.screenshot({path: resolve(process.env.SODA_ISSUE_LIST_EVIDENCE_DIR, 'issues-light.png')});
      await page.evaluate(() => document.querySelector('#root')!.setAttribute('data-factory-scheme', 'dark'));
      await page.screenshot({path: resolve(process.env.SODA_ISSUE_LIST_EVIDENCE_DIR, 'issues-dark.png')});
    }
    assert(
      (await page.evaluate(() => window.factoryIssueList.peers.flatMap((peer) => peer.calls))).every(
        (call) => call.method === 'GET'
      )
    );
    await page.evaluate(() => window.factoryIssueList.view.remove());
    assert(await page.evaluate(() => window.factoryIssueList.view.context === undefined));
    assert.deepEqual(errors, []);
  } finally {
    await browser.close();
    server.stop(true);
    await rm(out, {recursive: true, force: true});
  }
});

type IssueDetailPeer = ReturnType<
  typeof import('@factory/product/frontend/storybook/issue-detail-fixtures.js').issueDetailStoryPeer
>;
type IssueDetailStoryState =
  import('@factory/product/frontend/storybook/issue-detail-fixtures.js').IssueDetailStoryState;
declare global {
  interface Window {
    factoryIssueDetail: {
      view: import('@factory/product/frontend/factory/issues/detail.js').SodaFactoryIssue;
      peer: IssueDetailPeer;
      peers: IssueDetailPeer[];
      mount(state?: IssueDetailStoryState, repository?: string, number?: string): void;
    };
  }
}

test('shared issue detail renders sanitized Markdown and scoped paged conversation from production assets', async () => {
  const temporary = process.env.TMPDIR ?? resolve(homedir(), '.cache/soda-tests');
  await mkdir(temporary, {recursive: true});
  const out = await mkdtemp(resolve(temporary, 'issue-detail-ui-'));
  const product = await buildFactoryUI(out);
  for (const file of ['licenses/marked.LICENSE', 'licenses/dompurify.LICENSE', 'licenses/dompurify-MPL.LICENSE'])
    assert(product.files.includes(file));
  assert(product.sources.includes('frontend/factory/issues/detail.ts'));
  assert(product.sources.every((path) => !path.includes('/storybook/') && !path.endsWith('.stories.ts')));
  const peer = await Bun.build({
    entrypoints: [
      new URL(import.meta.resolve('@factory/product/frontend/storybook/issue-detail-fixtures.ts')).pathname,
    ],
    target: 'browser',
    format: 'esm',
  });
  assert(peer.success, peer.logs.join('\n'));
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const path = new URL(request.url).pathname;
      if (product.files.includes(path.slice(1))) return new Response(Bun.file(resolve(out, path.slice(1))));
      if (path === '/peer.js') return new Response(peer.outputs[0], {headers: {'Content-Type': 'text/javascript'}});
      if (path !== '/') return new Response(null, {status: 404});
      return new Response(
        '<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/factory.css"><main id="root" class="soda-factory-root" data-factory-scheme="light"></main>',
        {headers: {'Content-Type': 'text/html'}}
      );
    },
  });
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}),
    page = await browser.newPage();
  page.setDefaultTimeout(5000);
  const errors: string[] = [],
    external: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  page.on('request', (request) => {
    if (/^https?:/.test(request.url()) && !request.url().startsWith(server.url.origin)) external.push(request.url());
  });
  try {
    await page.goto(server.url.href);
    await page.evaluate(async () => {
      const product = await import(new URL('/factory-app.js', location.href).href);
      const declared: typeof import('@factory/product/frontend/storybook/issue-detail-fixtures.js') = await import(
        new URL('/peer.js', location.href).href
      );
      const view = new product.SodaFactoryIssue(),
        peer = declared.issueDetailStoryPeer();
      window.factoryIssueDetail = {
        view,
        peer,
        peers: [peer],
        mount(state = 'ready', repository, number) {
          const fixture = window.factoryIssueDetail;
          fixture.peer = declared.issueDetailStoryPeer(state, repository, number);
          fixture.peers.push(fixture.peer);
          const root = document.querySelector('#root')!;
          if (fixture.view.parentElement !== root) root.replaceChildren(fixture.view);
          fixture.view.context = fixture.peer.context;
        },
      };
      window.factoryIssueDetail.mount();
    });
    const detail = page.locator('soda-factory-issue'),
      description = detail.getByRole('article', {name: 'Issue description', exact: true});
    const comment = detail.getByRole('article', {name: 'Comment comment-201', exact: true});
    await description.getByText('reviewed context', {exact: true}).waitFor();
    assert.equal(await description.locator('strong').innerText(), 'reviewed context');
    assert.equal(await description.locator('table tbody td').first().innerText(), 'Read');
    assert.equal(await description.locator('pre code').innerText(), 'fmt.Println("hello")\n');
    assert.equal(await description.locator('blockquote').innerText(), 'Keep the conversation together.');
    assert.equal(await description.locator('input:disabled').count(), 2);
    assert.equal(await description.locator('input:checked').count(), 1);
    assert((await detail.innerText()).includes('#9223372036854775806'));
    await comment.getByText('discussion', {exact: true}).waitFor();
    assert((await comment.innerText()).includes('Edited 2026-11-03'));
    const reference = description.getByRole('link', {name: 'Reference', exact: true});
    assert.equal(await reference.getAttribute('href'), 'https://provider.invalid/team/repo/docs/design.md');
    await reference.focus();
    await page.keyboard.press('Enter');
    await page.waitForFunction(() => window.factoryIssueDetail.peer.opened.length === 1);
    assert.equal(
      await page.evaluate(() => window.factoryIssueDetail.peer.opened[0]),
      'https://provider.invalid/team/repo/docs/design.md'
    );
    await reference.click({button: 'right'});
    assert.equal(await page.evaluate(() => window.factoryIssueDetail.peer.opened.length), 1);
    await page.keyboard.press('Escape');
    await reference.click({button: 'middle'});
    await page.waitForFunction(() => window.factoryIssueDetail.peer.opened.length === 2);
    assert.equal(await description.locator('strong').innerText(), 'reviewed context');
    assert.equal(await description.locator('img').count(), 0);
    assert.equal(await description.getByRole('link', {name: 'Design image', exact: true}).count(), 1);

    await page.evaluate(() => window.factoryIssueDetail.mount('paged'));
    await comment.waitFor();
    assert.equal(await detail.locator('.soda-issue-comments > li').count(), 1);
    await detail.getByRole('button', {name: 'Next comments', exact: true}).click();
    await detail.getByText('Second conversation page.', {exact: true}).waitFor();
    assert.equal(await detail.locator('.soda-issue-comments > li').count(), 1);
    assert.equal(
      await page.evaluate(() =>
        new URL(window.factoryIssueDetail.peer.calls.at(-1)!.path, location.href).searchParams.get('cursor')
      ),
      'opaque:/comments?next=1&scope=declared'
    );
    assert(await detail.getByRole('button', {name: 'Next comments', exact: true}).isDisabled());
    await detail.getByRole('button', {name: 'Previous comments', exact: true}).click();
    await comment.waitFor();
    assert(await detail.getByRole('button', {name: 'Previous comments', exact: true}).isDisabled());
    assert.equal(
      await page.evaluate(
        () => window.factoryIssueDetail.peer.calls.filter((call) => !call.path.includes('/comments')).length
      ),
      3
    );
    await page.evaluate(() => window.factoryIssueDetail.peer.mismatch('parent-changed'));
    await detail.getByRole('button', {name: 'Next comments', exact: true}).click();
    await detail.getByText('The issue and conversation could not be loaded. Try again.', {exact: true}).waitFor();
    assert.equal(await description.count(), 0);

    for (const mismatch of [
      'repository',
      'number',
      'author',
      'comment-parent',
      'comment-kind',
      'comment-author',
      'duplicate',
      'cursor',
    ]) {
      await page.evaluate(() => window.factoryIssueDetail.mount());
      await comment.waitFor();
      await page.evaluate((value) => window.factoryIssueDetail.peer.mismatch(value), mismatch);
      await detail.getByRole('button', {name: 'Refresh issue', exact: true}).click();
      await detail.getByText('The issue and conversation could not be loaded. Try again.', {exact: true}).waitFor();
      assert.equal(await description.count(), 0, mismatch);
      assert.equal(await detail.locator('.soda-issue-comments > li').count(), 0);
    }
    for (const code of [401, 403, 404, 503]) {
      await page.evaluate(() => window.factoryIssueDetail.mount());
      await comment.waitFor();
      await page.evaluate((code) => window.factoryIssueDetail.peer.fail(code, '/comments'), code);
      await detail.getByRole('button', {name: 'Refresh issue', exact: true}).click();
      await detail
        .getByText(
          code === 503
            ? 'The issue and conversation could not be loaded. Try again.'
            : 'This issue is not available to this account.',
          {exact: true}
        )
        .waitFor();
      assert.equal(await description.count(), 0);
      await page.evaluate(() => window.factoryIssueDetail.peer.fail(200));
      await detail.getByRole('button', {name: 'Refresh issue', exact: true}).click();
      await comment.waitFor();
    }
    await page.evaluate(() => window.factoryIssueDetail.mount('hostile'));
    await description.getByText('Safe emphasis', {exact: true}).waitFor();
    assert.equal(await detail.locator('script, svg, img, form, [style], [onclick], [onerror], [name]').count(), 0);
    assert(
      (await detail.locator('a').evaluateAll((links) => links.map((a) => (a as HTMLAnchorElement).href))).every(
        (href) => /^https?:|^mailto:/.test(href) && !href.includes('@')
      )
    );
    assert.equal(await page.evaluate(() => Object.hasOwn(window, 'issueExecuted')), false);
    await description.getByRole('link', {name: 'Good', exact: true}).click();
    await page.waitForFunction(() => window.factoryIssueDetail.peer.opened.length === 1);
    await page.evaluate(() => window.factoryIssueDetail.mount('empty'));
    await description.getByText('No description provided.', {exact: true}).waitFor();
    await detail.getByText('No comments on this page.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryIssueDetail.mount('closed'));
    await comment.waitFor();
    assert((await detail.innerText()).includes('Closed 2026-11-03'));
    await page.evaluate(() => window.factoryIssueDetail.mount('missing'));
    await detail.getByText('Assignees unavailable', {exact: true}).waitFor();
    await detail.getByText('Labels unavailable', {exact: true}).waitFor();
    assert.equal(await detail.locator('time').count(), 2); // second comment retains its own timestamps
    await page.evaluate(() => window.factoryIssueDetail.mount('large'));
    await comment.waitFor();
    assert((await description.innerText()).length > 100000);
    assert((await comment.innerText()).length > 50000);

    await page.evaluate(() => window.factoryIssueDetail.mount('loading'));
    await page.waitForFunction(() => window.factoryIssueDetail.peer.held() > 0);
    const pendingRead = await page.evaluate(() => window.factoryIssueDetail.peers.length - 1);
    await page.evaluate(() => window.factoryIssueDetail.mount('ready', '43', '12'));
    await detail.getByRole('heading', {name: 'Another repository discussion', exact: true}).waitFor();
    await page.evaluate(() => window.factoryIssueDetail.peers.forEach((peer) => peer.release()));
    assert(await page.evaluate((index) => window.factoryIssueDetail.peers[index]!.aborted(), pendingRead));
    assert.equal(
      await detail.getByRole('heading', {name: 'Keep context while reviewing an issue', exact: true}).count(),
      0
    );
    await page.evaluate(() => window.factoryIssueDetail.mount());
    await comment.waitFor();
    await page.evaluate(() => window.factoryIssueDetail.peer.hold('/comments'));
    await detail.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await page.waitForFunction(() => window.factoryIssueDetail.peer.held() > 0);
    await page.evaluate(() => window.factoryIssueDetail.mount('ready', '43', '12'));
    await page.evaluate(() => window.factoryIssueDetail.peers.forEach((peer) => peer.release()));
    await detail.getByRole('heading', {name: 'Another repository discussion', exact: true}).waitFor();
    await page.evaluate(() => window.factoryIssueDetail.mount());
    await comment.waitFor();
    await page.evaluate(() => window.factoryIssueDetail.peer.holdLink());
    await reference.click();
    const pendingLink = await page.evaluate(() => window.factoryIssueDetail.peers.length - 1);
    await page.waitForFunction(() => window.factoryIssueDetail.peer.links.length === 1);
    await page.evaluate(() => window.factoryIssueDetail.mount('ready', '43', '12'));
    await page.evaluate(() => window.factoryIssueDetail.peers.forEach((peer) => peer.releaseLink()));
    assert(await page.evaluate((index) => window.factoryIssueDetail.peers[index]!.linkAborted(), pendingLink));
    assert.equal(await page.evaluate((index) => window.factoryIssueDetail.peers[index]!.opened.length, pendingLink), 0);
    await comment.waitFor();
    await page.evaluate(() => window.factoryIssueDetail.peer.retireGeneration());
    await detail.getByRole('button', {name: 'Refresh issue', exact: true}).click();
    await page.waitForFunction(() => !document.querySelector('soda-factory-issue')?.textContent);
    await page.evaluate(() => window.factoryIssueDetail.mount());
    await comment.waitFor();
    await page.evaluate(() => window.factoryIssueDetail.peer.failLink(true));
    await reference.click();
    await detail.getByText('This link could not be opened. Try again.', {exact: true}).waitFor();
    await page.evaluate(() => window.factoryIssueDetail.peer.failLink(false));
    await reference.click();
    await page.waitForFunction(() => window.factoryIssueDetail.peer.opened.length === 1);
    await page.setViewportSize({width: 360, height: 800});
    assert(await detail.evaluate((element) => element.scrollWidth <= document.documentElement.clientWidth));
    if (process.env.SODA_ISSUE_DETAIL_EVIDENCE_DIR) {
      await mkdir(process.env.SODA_ISSUE_DETAIL_EVIDENCE_DIR, {recursive: true});
      await page.screenshot({
        path: resolve(process.env.SODA_ISSUE_DETAIL_EVIDENCE_DIR, 'detail-light.png'),
        fullPage: true,
      });
      await page.evaluate(() => document.querySelector('#root')!.setAttribute('data-factory-scheme', 'dark'));
      await page.screenshot({
        path: resolve(process.env.SODA_ISSUE_DETAIL_EVIDENCE_DIR, 'detail-dark.png'),
        fullPage: true,
      });
    }
    assert(
      (await page.evaluate(() => window.factoryIssueDetail.peers.flatMap((peer) => peer.calls))).every(
        (call) => call.method === 'GET'
      )
    );
    assert.deepEqual(external, []);
    await page.evaluate(() => window.factoryIssueDetail.view.remove());
    assert(await page.evaluate(() => window.factoryIssueDetail.view.context === undefined));
    assert.deepEqual(errors, []);
  } finally {
    await browser.close();
    server.stop(true);
    await rm(out, {recursive: true, force: true});
  }
});

test('Fountain navigation consumes canonical native metadata over browser HTTP without guessing a branch', async (t) => {
  const page = await fixture(t);
  const root = process.env.SODA_FOUNTAIN_NAVIGATION_FIXTURES;
  const base = {
    host: 'a'.repeat(32),
    repository: '9007199254740993',
    actor: '9007199254740995',
    native_repository: '7',
    name: 'alice/demo',
    default_branch: 'main',
  };
  const samples: Record<string, typeof base> = {};
  for (const [phase, branch] of [
    ['initial', 'main'],
    ['changed', 'feature/完整'],
    ['unknown', ''],
  ])
    samples[phase!] = root
      ? await Bun.file(resolve(root, phase! + '.json')).json()
      : {...base, default_branch: branch!};
  let value: unknown = samples.initial,
    status = 200;
  const calls: {path: string; method: string}[] = [];
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(request) {
      const url = new URL(request.url);
      calls.push({path: url.pathname, method: request.method});
      return Response.json(value, {status, headers: {'Access-Control-Allow-Origin': '*'}});
    },
  });
  const read = (retired = false, aborted = false) =>
    page.evaluate(
      async ({api, repository, retired, aborted}) => {
        let generation = 'native-navigation';
        const controller = new AbortController();
        if (aborted) controller.abort();
        const transport = {
          get generation() {
            return generation;
          },
          assetBase: '/',
          async request(path: string, init?: RequestInit) {
            const response = await window.drawerFixture.browserFetch(new URL(path, api), init);
            if (retired) generation += ':retired';
            return response;
          },
        };
        try {
          return {
            value: await window.drawerFixture.repositoryNavigation(transport, repository, controller.signal),
            refused: false,
          };
        } catch {
          return {refused: true};
        }
      },
      {api: server.url.href, repository: samples.initial!.repository, retired, aborted}
    );
  try {
    for (const phase of ['initial', 'changed', 'unknown']) {
      value = samples[phase];
      const result = await read();
      assert.equal(result.refused, false);
      assert.deepEqual(result.value, {
        host: samples[phase]!.host,
        actor: samples[phase]!.actor,
        repository: samples[phase]!.repository,
        native: samples[phase]!.native_repository,
        name: samples[phase]!.name,
        defaultBranch: samples[phase]!.default_branch,
      });
    }
    for (const change of [
      {default_branch: 7},
      {default_branch: 'line\nbreak'},
      {default_branch: '🌋'.repeat(300)},
      {repository: '7'},
      {native_repository: 7},
      {host: 'foreign'},
      {name: '../foreign'},
    ]) {
      value = {...samples.initial, ...change};
      assert.equal((await read()).refused, true);
    }
    value = samples.changed;
    assert.equal((await read(true)).refused, true, 'retired generation released navigation');
    const beforeAbort = calls.length;
    assert.equal((await read(false, true)).refused, true);
    assert.equal(calls.length, beforeAbort, 'aborted navigation issued a request');
    for (status of [401, 403, 404, 503]) assert.equal((await read()).refused, true);
    assert(
      calls.every(
        (call) =>
          call.method === 'GET' && call.path === '/factory/repositories/' + samples.initial!.repository + '/identity'
      )
    );
  } finally {
    server.stop(true);
  }
});

test('Fountain issue host renders shared views over HTTP and separates scoped native and external navigation', async (t) => {
  const page = await fixture(t);
  const samples = process.env.SODA_FOUNTAIN_NAVIGATION_FIXTURES;
  const original: {
    host: string;
    actor: string;
    repository: string;
    native_repository: string;
    name: string;
    default_branch: string;
  } = samples
    ? await Bun.file(resolve(samples, 'changed.json')).json()
    : {
        host: 'a'.repeat(32),
        actor: '9007199254740995',
        repository: '9007199254740993',
        native_repository: '7',
        name: 'alice/demo',
        default_branch: 'feature/完整',
      };
  let metadata = {...original};
  let metadataStatus = 200;
  let retired = false;
  let holdList = false;
  let releaseList: (() => void) | undefined;
  const observation = {state: 'available', revision: 'native-observation', observed_at: '2026-11-03T12:00:00Z'};
  const issue = {
    ref: {kind: 'issue', repository: original.repository, external_id: 'native-issue-73', number: '9007199254740997'},
    observation,
    title: 'Connected host read',
    state: 'open',
    author: null,
    assignees: [],
    labels: [],
    created_at: null,
    updated_at: null,
    closed_at: null,
    body: '[Repository file](README.md) [Explicit branch](src/branch/feature/%E5%AE%8C%E6%95%B4/docs/readme.md) [Fragment](#issuecomment-81) [Query](?page=2) [External](https://docs.example.test/guide) [Mail](mailto:team@example.test) [Root](/soda/explore/repos) [Bad](javascript:alert%281%29) ![Attachment](assets/map.png)',
  };
  const calls: {path: string; method: string}[] = [];
  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    async fetch(request) {
      const url = new URL(request.url);
      calls.push({path: url.pathname + url.search, method: request.method});
      const headers = {'Access-Control-Allow-Origin': '*'};
      if (url.pathname === `/factory/repositories/${original.repository}/identity`)
        return Response.json(metadata, {status: metadataStatus, headers});
      const path = `/factory/collaboration/repositories/${original.repository}/issues`;
      if (url.pathname === path) {
        if (holdList) await new Promise<void>((resolve) => (releaseList = resolve));
        return Response.json({observation, items: [issue], next: '', complete: true}, {headers});
      }
      if (url.pathname === `${path}/${issue.ref.number}`) return Response.json(issue, {headers});
      if (url.pathname === `${path}/${issue.ref.number}/comments`)
        return Response.json(
          {
            observation,
            items: [
              {
                issue: issue.ref,
                comment: {
                  external_id: '81',
                  observation,
                  body: '[Comment file](docs/review.md)',
                  author: null,
                  created_at: null,
                  updated_at: null,
                },
              },
            ],
            next: '',
            complete: true,
          },
          {headers}
        );
      return Response.json({}, {status: 404, headers});
    },
  });
  const admit = (change: Record<string, unknown> = {}, aborted = false) =>
    page.evaluate(
      async ({api, binding, change, aborted, retired}) => {
        const fixture = window.drawerFixture,
          state = fixture.issueHost;
        state.lifetime.abort();
        state.list?.remove();
        state.detail?.remove();
        delete state.contexts;
        state.lifetime = new AbortController();
        if (aborted) state.lifetime.abort();
        state.transport = {
          generation: 'host-current',
          assetBase: '/assets/',
          async request(path, init) {
            const response = await fixture.browserFetch(new URL(path, api), init);
            if (retired) state.transport!.generation += ':retired';
            return response;
          },
        };
        const record = async (kind: string, url: string, signal: AbortSignal) => {
          state.opening = signal;
          if (state.release)
            await new Promise<void>((resolve) => {
              state.release = resolve;
            });
          signal.throwIfAborted();
          state.opened.push({kind, url});
        };
        try {
          const options = {
            host: binding.host,
            actor: binding.actor,
            repository: binding.repository,
            nativeRepository: binding.native_repository,
            name: binding.name,
            prefix: '/soda',
            origin: location.origin,
            issuePath: (number: string) => '/soda/' + binding.name + '/issues/' + number,
            navigate: (url: string, signal: AbortSignal) => record('native', url, signal),
            openExternal: (url: string, signal: AbortSignal) => record('external', url, signal),
            ...change,
          };
          state.contexts = await fixture.issueContexts(state.transport, options, state.lifetime.signal);
          // Admitted contexts capture these values, rather than following a mutable options object.
          options.actor = '7';
          options.repository = '7';
          options.name = 'elsewhere/rebound';
          return {
            refused: false,
            scope: state.contexts.list.scope,
            repository: state.contexts.list.repository,
            actor: state.contexts.list.actor,
          };
        } catch {
          return {refused: true};
        }
      },
      {api: server.url.href, binding: original, change, aborted, retired}
    );
  try {
    const admitted = await admit();
    assert.equal(admitted.refused, false);
    assert.equal(admitted.repository, original.repository);
    assert.equal(admitted.actor, original.actor);
    assert.equal(admitted.scope, `fountain:${original.host}:${original.actor}`);
    await page.evaluate(() => {
      const fixture = window.drawerFixture,
        state = fixture.issueHost;
      state.list = fixture.issuesElement();
      state.list.context = state.contexts!.list;
      fixture.root.append(state.list);
    });
    const list = page.locator('soda-factory-issues');
    await list.getByRole('button', {name: /Connected host read/}).click();
    assert.deepEqual(await page.evaluate(() => window.drawerFixture.issueHost.opened), [
      {kind: 'native', url: '/soda/' + original.name + '/issues/' + issue.ref.number},
    ]);
    await page.evaluate((number) => {
      const fixture = window.drawerFixture,
        state = fixture.issueHost;
      state.list!.remove();
      state.detail = fixture.issueElement();
      state.detail.context = state.contexts!.detail(number);
      fixture.root.append(state.detail);
    }, issue.ref.number);
    const detail = page.locator('soda-factory-issue');
    await detail.getByRole('heading', {name: issue.title, exact: true}).waitFor();
    const origin = new URL(page.url()).origin;
    const repo = origin + '/soda/' + original.name;
    for (const [label, href] of [
      ['Repository file', repo + '/README.md'],
      ['Explicit branch', repo + '/src/branch/feature/%E5%AE%8C%E6%95%B4/docs/readme.md'],
      ['Fragment', repo + '/issues/' + issue.ref.number + '#issuecomment-81'],
      ['Query', repo + '/issues/' + issue.ref.number + '?page=2'],
      ['Root', origin + '/soda/explore/repos'],
      ['Attachment', repo + '/assets/map.png'],
      ['Comment file', repo + '/docs/review.md'],
    ] as const)
      assert.equal(await detail.getByRole('link', {name: label, exact: true}).getAttribute('href'), href);
    assert.equal(await detail.getByRole('link', {name: 'Bad', exact: true}).count(), 0);
    assert.equal(await detail.locator('img').count(), 0);
    await detail.getByRole('link', {name: 'Repository file', exact: true}).click();
    await detail.getByRole('link', {name: 'External', exact: true}).click();
    await detail.getByRole('link', {name: 'Mail', exact: true}).click();
    assert.deepEqual((await page.evaluate(() => window.drawerFixture.issueHost.opened)).slice(-3), [
      {kind: 'native', url: '/soda/' + original.name + '/README.md'},
      {kind: 'external', url: 'https://docs.example.test/guide'},
      {kind: 'external', url: 'mailto:team@example.test'},
    ]);
    // A real UI click opens an asynchronous host port. Retiring the source view
    // cancels it even if the port's delayed work ignores cancellation until release.
    await page.evaluate(() => {
      window.drawerFixture.issueHost.release = () => {};
    });
    await detail.getByRole('link', {name: 'External', exact: true}).click();
    const before = await page.evaluate(() => {
      const state = window.drawerFixture.issueHost;
      const count = state.opened.length;
      state.detail!.context = undefined;
      const cancelled = state.opening!.aborted;
      state.release!();
      delete state.release;
      return {count, cancelled};
    });
    assert.equal(before.cancelled, true);
    await page.waitForFunction(() => window.drawerFixture.issueHost.opening!.aborted);
    assert.equal(await page.evaluate(() => window.drawerFixture.issueHost.opened.length), before.count);
    for (const change of [
      {actor: '7'},
      {host: 'b'.repeat(32)},
      {nativeRepository: '9'},
      {name: 'foreign/repo'},
      {origin: 'https://user:password@example.test'},
      {prefix: '/bad/../prefix'},
    ])
      assert.equal((await admit(change)).refused, true);
    for (const change of [{actor: '7'}, {host: 'b'.repeat(32)}, {native_repository: '9'}, {name: 'foreign/repo'}]) {
      metadata = {...original, ...change};
      assert.equal((await admit()).refused, true);
    }
    metadata = {...original};
    retired = true;
    assert.equal((await admit()).refused, true);
    retired = false;
    const beforeAbort = calls.length;
    assert.equal((await admit({}, true)).refused, true);
    assert.equal(calls.length, beforeAbort);
    for (metadataStatus of [401, 403, 404, 503]) assert.equal((await admit()).refused, true);
    metadataStatus = 200;
    metadata.default_branch = '';
    assert.equal(
      (await admit()).refused,
      false,
      'unknown branch must not invent a fallback or remove valid native relative links'
    );
    const refusals = await page.evaluate(async (ref) => {
      const state = window.drawerFixture.issueHost,
        contexts = state.contexts!;
      const denied = async (action: () => unknown) => {
        try {
          await action();
          return false;
        } catch {
          return true;
        }
      };
      const differentRepo = await denied(() =>
        contexts.list.openIssue({...ref, repository: '7', kind: 'issue'}, new AbortController().signal)
      );
      const unsafe = await denied(() =>
        contexts.detail(ref.number).openLink('javascript:alert(1)', new AbortController().signal)
      );
      const relative = contexts.detail(ref.number).resolveLink('README.md');
      state.transport!.generation += ':changed';
      const stale = await denied(() => contexts.list.openIssue({...ref, kind: 'issue'}, new AbortController().signal));
      return {differentRepo, unsafe, stale, relative};
    }, issue.ref);
    assert.deepEqual(refusals, {differentRepo: true, unsafe: true, stale: true, relative: repo + '/README.md'});
    // A place can retire before the pending browser HTTP read finishes. Its
    // transport aborts the read and refuses future reads even before disposal.
    assert.equal((await admit()).refused, false);
    holdList = true;
    await page.evaluate(() => {
      const fixture = window.drawerFixture,
        state = fixture.issueHost;
      state.list = fixture.issuesElement();
      state.list.context = state.contexts!.list;
      fixture.root.append(state.list);
    });
    await list.getByText('Loading issues…', {exact: true}).waitFor();
    await page.waitForFunction(() => window.drawerFixture.issueHost.list?.querySelector('[aria-busy=true]') !== null);
    // Wait for the server's actual pending request, not just the loading label.
    while (!releaseList) await new Promise((resolve) => setTimeout(resolve, 5));
    await page.evaluate(() => window.drawerFixture.issueHost.lifetime.abort());
    await list.getByText('Issues could not be loaded. Try again.', {exact: true}).waitFor();
    releaseList();
    holdList = false;
    const afterCancel = calls.length;
    await list.getByRole('button', {name: 'Refresh issues', exact: true}).click();
    await list.getByText('Issues could not be loaded. Try again.', {exact: true}).waitFor();
    assert.equal(calls.length, afterCancel, 'retired place issued another HTTP read');
    assert.equal(await list.getByRole('button', {name: /Connected host read/}).count(), 0);
    assert(calls.every((call) => call.method === 'GET' && call.path.startsWith('/factory/')));
    assert(calls.some((call) => call.path.endsWith('/comments?limit=20')));
  } finally {
    releaseList?.();
    server.stop(true);
  }
});

test('Fountain work routes mount shared issues, preserve Factory controls and follow browser history and fragments', async (t) => {
  const page = await fixture(t);
  await page.evaluate(() => {
    const repository = '9007199254740993',
      host = 'a'.repeat(32);
    const observation = {state: 'available', revision: 'native-observation', observed_at: '2026-11-03T12:00:00Z'};
    const issue = {
      ref: {kind: 'issue', repository, external_id: 'native-42', number: '42'},
      observation,
      title: 'Shared issue route',
      state: 'open',
      author: null,
      assignees: [],
      labels: [],
      created_at: null,
      updated_at: null,
      closed_at: null,
      body:
        '# Route heading\n[Heading](#route-heading) [Comment](#issuecomment-81) [File](README.md)\n\n' +
        'Description content. '.repeat(100),
    };
    let denied = false;
    window.addEventListener('deny-issue-route', () => {
      denied = true;
    });
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/api/factory/identity')) return Response.json({host, actor: '11', native_actor: '1'});
      if (call.url.endsWith('/api/factory/native-repositories/resolve'))
        return Response.json({
          host,
          actor: '11',
          repository,
          native_repository: '7',
          name: 'alice/demo',
          default_branch: 'main',
        });
      if (call.url.endsWith('/api/factory/repositories/' + repository + '/identity'))
        return Response.json({
          host,
          actor: '11',
          repository,
          native_repository: '7',
          name: 'alice/demo',
          default_branch: '',
        });
      if (call.url.includes('/api/factory/collaboration/')) {
        if (denied) return Response.json({}, {status: 403});
        if (call.url.includes('/comments?'))
          return Response.json({
            observation,
            items: [
              {
                issue: issue.ref,
                comment: {
                  external_id: '81',
                  observation,
                  body: '# Route heading\n[Comment heading](#route-heading)',
                  author: null,
                  created_at: null,
                  updated_at: null,
                },
              },
            ],
            next: '',
            complete: true,
          });
        if (call.url.includes('/issues?'))
          return Response.json({observation, items: [issue], next: '', complete: true});
        return Response.json(issue);
      }
      if (call.url.includes('/factory/issues/')) return Response.json({}, {status: 404});
      return null;
    });
    window.drawerFixture.mountRepository('7', '/soda/alice/demo/feed/work/issues', '/soda', true);
  });
  const root = page.locator('.soda-repository-page');
  await root
    .locator('soda-factory-issues')
    .getByRole('button', {name: /Shared issue route/})
    .click();
  assert.equal(new URL(page.url()).pathname, '/soda/alice/demo/feed/work/issues/42');
  const detail = root.locator('soda-factory-issue');
  await detail.getByRole('heading', {name: 'Shared issue route', exact: true}).waitFor();
  await detail.getByRole('link', {name: 'Comment heading', exact: true}).waitFor();
  assert.equal(await root.locator('soda-factory-work-journey').count(), 1);
  for (const tag of ['requirements-acceptance', 'context-resolution', 'completion-retry']) {
    const widget = root.locator('soda-factory-' + tag);
    await widget.waitFor({state: 'attached'});
  }
  await page.waitForFunction(
    () =>
      window.drawerFixture.calls.filter((call) => call.url.endsWith('/repositories/9007199254740993/factory/issues/42'))
        .length >= 2
  );
  const native = root.getByRole('navigation', {name: 'Issue navigation'}).getByRole('link', {name: 'Open in Fountain'});
  assert.equal(await native.getAttribute('href'), '/soda/alice/demo/issues/42');
  const readsBefore = await page.evaluate(() => window.drawerFixture.calls.length);
  await root
    .getByRole('navigation', {name: 'Issue navigation'})
    .getByRole('link', {name: 'Factory', exact: true})
    .click();
  assert.equal(new URL(page.url()).search, '?tab=factory');
  assert.equal(await root.locator('details.soda-issue-factory-controls').getAttribute('open'), '');
  assert(
    await root
      .locator('details.soda-issue-factory-controls')
      .evaluate((element) => element.getBoundingClientRect().top < 800),
    'Factory destination did not scroll to controls'
  );
  if (process.env.SODA_FOUNTAIN_ROUTE_SCREENSHOT)
    await root.screenshot({path: process.env.SODA_FOUNTAIN_ROUTE_SCREENSHOT, animations: 'disabled'});
  assert.equal(
    await page.evaluate(() => window.drawerFixture.calls.length),
    readsBefore,
    'tab change remounted live issue/controls'
  );
  await root
    .getByRole('navigation', {name: 'Issue navigation'})
    .getByRole('link', {name: 'Conversation', exact: true})
    .click();
  await detail.getByRole('link', {name: 'Comment', exact: true}).click();
  assert.equal(new URL(page.url()).hash, '#issuecomment-81');
  assert.equal(await detail.locator('#issuecomment-81').count(), 1);
  const articleTop = await detail
    .locator('#issuecomment-81')
    .evaluate((element) => element.getBoundingClientRect().top);
  assert(articleTop >= -2 && articleTop < 800, 'comment fragment did not scroll into view');
  await detail.getByRole('link', {name: 'Heading', exact: true}).click();
  const descriptionHeading = detail.getByRole('heading', {name: 'Route heading', exact: true}).first();
  assert.equal(new URL(page.url()).hash.slice(1), await descriptionHeading.getAttribute('id'));
  await detail.getByRole('link', {name: 'Comment heading', exact: true}).click();
  const commentHeading = detail.getByRole('heading', {name: 'Route heading', exact: true}).last();
  assert.equal(new URL(page.url()).hash.slice(1), await commentHeading.getAttribute('id'));
  assert.notEqual(await commentHeading.getAttribute('id'), await descriptionHeading.getAttribute('id'));
  await root
    .getByRole('navigation', {name: 'Issue navigation'})
    .getByRole('link', {name: 'Issues', exact: true})
    .click();
  await root
    .locator('soda-factory-issues')
    .getByRole('button', {name: /Shared issue route/})
    .waitFor();
  await page.goBack();
  await root.locator('soda-factory-issue').getByRole('heading', {name: 'Shared issue route', exact: true}).waitFor();
  assert.equal(
    new URL(page.url()).hash.slice(1),
    await root.getByRole('heading', {name: 'Route heading', exact: true}).last().getAttribute('id')
  );
  await page.evaluate(() => window.dispatchEvent(new Event('deny-issue-route')));
  await root.getByRole('button', {name: 'Refresh issue', exact: true}).click();
  await root.getByText('This issue is not available to this account.', {exact: true}).waitFor();
  assert.equal(await root.locator('soda-factory-requirements-acceptance').count(), 0);
  assert.equal((await root.locator('soda-factory-work-journey').textContent())?.trim(), '');
  assert.equal(await root.getByText('Description content.', {exact: true}).count(), 0);
  await page.evaluate(() =>
    window.drawerFixture.repositoryHandle?.update?.({
      mode: 'page',
      route: '/soda/alice/demo/feed/work/issues/new',
      params: {},
      visible: true,
    })
  );
  await root.locator('soda-factory-issue-create').getByRole('heading', {name: 'New issue', exact: true}).waitFor();
  for (const [suffix, message] of [
    ['/work/pulls/42', 'The shared pull request view is not available yet.'],
    ['/issues/42', 'This SodaOS page could not load.'],
    ['/work/issues/0', 'This SodaOS page could not load.'],
    ['/work/issues/9223372036854775808', 'This SodaOS page could not load.'],
    ['/work/issues/42/extra', 'This SodaOS page could not load.'],
  ] as const) {
    await page.evaluate(
      (route) => window.drawerFixture.repositoryHandle?.update?.({mode: 'page', route, params: {}, visible: true}),
      '/soda/alice/demo/feed' + suffix
    );
    await root.getByText(message, {exact: true}).waitFor();
  }
  await page.evaluate(() => {
    const repository = '9007199254740993',
      host = 'a'.repeat(32),
      name = 'alice/演示';
    const observation = {state: 'available', revision: 'unicode-repository', observed_at: '2026-11-03T12:00:00Z'};
    window.drawerFixture.setReply(async (call) => {
      if (call.url.endsWith('/native-repositories/resolve') || call.url.endsWith('/identity'))
        return Response.json({host, actor: '11', repository, native_repository: '7', name, default_branch: ''});
      if (call.url.includes('/collaboration/')) {
        if (call.url.includes('/comments?')) return Response.json({observation, items: [], next: '', complete: true});
        return Response.json({
          ref: {kind: 'issue', repository, number: '42', external_id: 'native-42'},
          observation,
          title: 'Unicode repository',
          state: 'open',
          author: null,
          assignees: [],
          labels: [],
          body: '[File](README.md)',
          created_at: null,
          updated_at: null,
          closed_at: null,
        });
      }
      if (call.url.includes('/factory/issues/')) return Response.json({}, {status: 404});
      return null;
    });
    window.drawerFixture.mountRepository(
      '7',
      '/soda/alice/' + encodeURIComponent('演示') + '/feed/work/issues/42',
      '/soda',
      true
    );
  });
  await root.getByRole('heading', {name: 'Unicode repository', exact: true}).waitFor();
  assert.equal(
    await root
      .getByRole('navigation', {name: 'Issue navigation'})
      .getByRole('link', {name: 'Open in Fountain'})
      .getAttribute('href'),
    '/soda/alice/演示/issues/42'
  );
  assert.equal(
    await root.locator('soda-factory-issue').getByRole('link', {name: 'File', exact: true}).getAttribute('href'),
    new URL(page.url()).origin + '/soda/alice/' + encodeURIComponent('演示') + '/README.md'
  );
  const calls = await page.evaluate(() => window.drawerFixture.calls);
  assert(!calls.some((call) => call.url.includes('/collaboration/repositories/7/')));
  assert(!calls.some((call) => call.method !== 'GET' && !call.url.endsWith('/native-repositories/resolve')));
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});

test(
  'native Fountain browser comment submits, recovers lost replies, cancels and retries the original journaled writing',
  {skip: !process.env.SODA_FOUNTAIN_BROWSER_API},
  async (t) => {
    const page = await fixture(t);
    const api = process.env.SODA_FOUNTAIN_BROWSER_API!;
    const native = process.env.SODA_FOUNTAIN_BROWSER_REPOSITORY!;
    await page.evaluate(
      ({api, native}) => {
        window.drawerFixture.forwardAPI(api);
        window.drawerFixture.mountRepository(native, '/user1/human-comment-primary/feed/work/issues/1', '', true);
      },
      {api, native}
    );
    const composer = page.locator('soda-factory-issue-comment');
    const post = composer.getByRole('button', {name: 'Post comment', exact: true});
    await post.waitFor();
    await page.waitForFunction(
      () => !document.querySelector<HTMLTextAreaElement>('soda-factory-issue-comment textarea')?.disabled
    );
    const field = composer.getByRole('textbox', {name: 'Comment Markdown'});
    const body = 'Native browser first 完整 comment.\n';
    await field.fill(body);
    await post.click();
    await page.locator('soda-factory-issue').getByText(body.trim(), {exact: true}).waitFor();
    assert.equal(await field.inputValue(), '');
    await field.fill('HTTP committed reply dropped.');
    await post.click();
    await composer.getByText(/No final outcome was confirmed/).waitFor();
    const before = await page.evaluate(() =>
      window.drawerFixture.calls
        .filter((call) => call.method === 'POST' && call.url.endsWith('/comments'))
        .map((call) => JSON.parse(call.body!))
    );
    await composer.getByRole('button', {name: 'Check comment status', exact: true}).click();
    await page.locator('soda-factory-issue').getByText('HTTP committed reply dropped.', {exact: true}).waitFor();
    assert.equal(
      await page.evaluate(
        () =>
          window.drawerFixture.calls.filter((call) => call.method === 'POST' && call.url.endsWith('/comments')).length
      ),
      before.length
    );
    const hold = async () => {
      const response = await fetch(new URL('fixture/hold-next', api), {method: 'POST'});
      assert.equal(response.status, 204);
    };
    await hold();
    await field.fill('Native browser cancel pending.');
    await post.click();
    await composer.getByText(/No final outcome was confirmed/).waitFor();
    await composer.getByRole('button', {name: 'Cancel submission', exact: true}).click();
    await composer.locator('section[aria-busy=false]').waitFor();
    assert.equal(await composer.getByRole('status').textContent(), 'Submission cancelled. Your writing was preserved.');
    await composer.getByText('Submission cancelled. Your writing was preserved.', {exact: true}).waitFor();
    assert.equal(await field.inputValue(), 'Native browser cancel pending.');
    assert.equal(
      await page.locator('soda-factory-issue').getByText('Native browser cancel pending.', {exact: true}).count(),
      0
    );
    await hold();
    await field.fill('Native browser retry original.');
    await post.click();
    await composer.getByText(/No final outcome was confirmed/).waitFor();
    await composer.getByRole('button', {name: 'Check comment status', exact: true}).click();
    await composer.getByText(/submission is not confirmed/).waitFor();
    await composer.getByRole('button', {name: 'Retry saved comment', exact: true}).click();
    await page.locator('soda-factory-issue').getByText('Native browser retry original.', {exact: true}).waitFor();
    const submissions = await page.evaluate(() =>
      window.drawerFixture.calls
        .filter((call) => call.method === 'POST' && call.url.endsWith('/comments'))
        .map((call) => JSON.parse(call.body!) as {command_id: string; body: string})
    );
    const retries = submissions.filter((value) => value.body === 'Native browser retry original.');
    assert.equal(retries.length, 2);
    assert.deepEqual(retries[0], retries[1]);
    assert.equal(submissions.filter((value) => value.body === 'HTTP committed reply dropped.').length, 1);
    assert.equal(await field.inputValue(), '');
    await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
  }
);

test('shared issue comment preserves scoped writing and pending intent across storage failure, retirement and a new browser realm', async (t) => {
  const page = await fixture(t);
  const origin = new URL(page.url()).origin;
  const setup = () =>
    page.evaluate(() => {
      const repository = '9007199254740993',
        host = 'a'.repeat(32);
      let actor = '11',
        mode = 'ready',
        release: (() => void) | undefined,
        lostCancel = false;
      const observation = {state: 'available', revision: 'declared-comment-peer', observed_at: '2026-11-03T12:00:00Z'};
      const parent = {kind: 'issue', repository, number: '42', external_id: 'native-42'};
      const replies: {id: string; body: string}[] = [];
      const commands = new Map<string, {body: string; committed: boolean; cancelled?: boolean}>();
      window.addEventListener('comment-peer', (event) => {
        const next = (event as CustomEvent<{mode?: string; actor?: string}>).detail;
        actor = next.actor ?? actor;
        mode = next.mode ?? mode;
      });
      window.addEventListener('comment-release', () => release?.());
      const result = (command: string, effect = 'committed', cancel = 'none', completion = 'complete') =>
        Response.json({
          command_id: command,
          observation: 'found',
          effect,
          cancellation: cancel,
          completion,
          finished: completion === 'complete',
        });
      window.drawerFixture.setReply(async (call) => {
        if (call.url.endsWith('/factory/identity'))
          return Response.json({host, actor, native_actor: actor === '11' ? '1' : '2'});
        if (
          call.url.endsWith('/native-repositories/resolve') ||
          call.url.endsWith('/factory/repositories/' + repository + '/identity')
        )
          return Response.json({
            host,
            actor,
            repository,
            native_repository: '7',
            name: 'alice/demo',
            default_branch: '',
          });
        if (!call.url.includes('/collaboration/')) {
          if (call.url.includes('/factory/issues/')) return Response.json({}, {status: 404});
          return null;
        }
        if (call.url.endsWith('/capabilities'))
          return Response.json({
            repository,
            actor: mode === 'actor-mismatch' ? '22' : actor,
            issue: {...parent, external_id: mode === 'parent-mismatch' ? 'foreign-parent' : parent.external_id},
            pull: null,
            head: null,
            base: null,
            observation,
            actions: {'issue.comment': {state: mode === 'denied' ? 'denied' : 'allowed', reason: ''}},
          });
        const command = /\/comments\/commands\/([a-f0-9]{32})/.exec(call.url)?.[1];
        if (command) {
          if (call.method === 'POST') {
            if (mode === 'cancel-lost' && !lostCancel) {
              lostCancel = true;
              throw Error('Declared cancellation reply unavailable');
            }
            const row = commands.get(command);
            if (!row) return Response.json({error: {code: 'comment_command_not_found'}}, {status: 404});
            if (row.committed) return result(command, 'committed', 'too_late');
            row.cancelled = true;
            return result(command, 'not_committed', 'cancelled');
          }
          const current = commands.get(command);
          if (!current) return Response.json({error: {code: 'comment_command_not_found'}}, {status: 404});
          if (current.cancelled) return result(command, 'not_committed', 'cancelled');
          return current?.committed
            ? result(command)
            : Response.json({command_id: command, observation: 'not_observed', finished: false});
        }
        if (call.method === 'POST' && call.url.endsWith('/comments')) {
          const input = JSON.parse(call.body!) as {command_id: string; body: string};
          const row = {body: input.body, committed: false};
          commands.set(input.command_id, row);
          if (mode === 'held')
            await new Promise<void>((resolve) => {
              release = resolve;
            });
          if (mode === 'unknown' || mode === 'cancel-lost')
            return Response.json(
              {command_id: input.command_id, observation: 'unknown', finished: false},
              {status: 202}
            );
          if (mode === 'malformed')
            return Response.json({
              command_id: input.command_id,
              observation: 'found',
              effect: 'pending',
              cancellation: 'none',
              completion: 'complete',
              finished: true,
            });
          row.committed = true;
          replies.push({id: String(replies.length + 81), body: input.body});
          return result(input.command_id);
        }
        if (call.url.includes('/comments?'))
          return Response.json({
            observation,
            items: replies.map((row) => ({
              issue: parent,
              comment: {
                external_id: row.id,
                observation,
                body: row.body,
                author: null,
                created_at: null,
                updated_at: null,
              },
            })),
            next: '',
            complete: true,
          });
        return Response.json({
          ref: parent,
          observation,
          title: 'Comment state fixture',
          body: 'Current ordinary issue',
          state: 'open',
          author: null,
          assignees: [],
          labels: [],
          created_at: null,
          updated_at: null,
          closed_at: null,
        });
      });
      window.drawerFixture.mountRepository('7', '/alice/demo/feed/work/issues/42', '', true);
    });
  const remount = () =>
    page.evaluate(() => window.drawerFixture.mountRepository('7', '/alice/demo/feed/work/issues/42', '', true));
  const mode = (value: string, actor?: string) =>
    page.evaluate(
      ({value, actor}) =>
        window.dispatchEvent(new CustomEvent('comment-peer', {detail: {mode: value, ...(actor ? {actor} : {})}})),
      {value, actor}
    );
  const composer = page.locator('soda-factory-issue-comment');
  const field = composer.getByRole('textbox', {name: 'Comment Markdown'});
  const post = composer.getByRole('button', {name: 'Post comment', exact: true});
  const idle = () => composer.locator('section[aria-busy=false]').waitFor();
  const posts = () =>
    page.evaluate(() =>
      window.drawerFixture.calls.filter((call) => call.method === 'POST' && call.url.endsWith('/comments'))
    );
  await setup();
  await idle();
  await field.fill('🌋'.repeat(16384));
  assert.equal(await post.isEnabled(), true);
  await field.fill('🌋'.repeat(16385));
  assert.equal(await post.isEnabled(), false);
  assert.equal((await field.inputValue()).length, 32770, 'large writing was truncated');
  await field.fill('\u0085');
  assert.equal(await post.isEnabled(), false);
  await field.evaluate((node) => {
    (node as HTMLTextAreaElement).value = String.fromCharCode(0xd800);
    node.dispatchEvent(new InputEvent('input', {bubbles: true}));
  });
  assert.equal(await post.isEnabled(), false);
  await field.fill('Alice unsent writing.');
  await remount();
  await idle();
  assert.equal(await field.inputValue(), 'Alice unsent writing.');
  assert.equal((await posts()).length, 0);
  await page.evaluate(() => {
    const original = Storage.prototype.setItem;
    window.addEventListener(
      'restore-comment-storage',
      () => {
        Storage.prototype.setItem = original;
      },
      {once: true}
    );
    window.addEventListener(
      'write-newer-comment-draft',
      () => {
        const key = Object.keys(localStorage).find(
          (key) => key.startsWith('soda-issue-comment:') && JSON.parse(localStorage.getItem(key)!).actor === '11'
        )!;
        const row = JSON.parse(localStorage.getItem(key)!);
        row.writing.draft = {revision: 'b'.repeat(32), body: 'Newer writing from another window.'};
        original.call(localStorage, key, JSON.stringify(row));
      },
      {once: true}
    );
    Storage.prototype.setItem = function (key, value) {
      if (key.startsWith('soda-issue-comment:')) throw new DOMException('Declared quota failure', 'QuotaExceededError');
      return original.call(this, key, value);
    };
  });
  await field.fill('Saved only in this window.');
  await post.click();
  await idle();
  assert.equal((await posts()).length, 0, 'undurable pending intent was dispatched');
  await composer.getByText(/only saved in this window/).waitFor();
  await page.evaluate(() => window.dispatchEvent(new Event('write-newer-comment-draft')));
  await remount();
  await idle();
  assert.equal(await field.inputValue(), 'Saved only in this window.');
  await page.evaluate(() => window.dispatchEvent(new Event('restore-comment-storage')));
  await composer.getByRole('button', {name: 'Reload saved writing', exact: true}).click();
  await idle();
  await composer.getByRole('button', {name: 'Retry saved comment', exact: true}).click();
  await page.locator('soda-factory-issue').getByText('Saved only in this window.', {exact: true}).waitFor();
  await idle();
  assert.equal(await field.inputValue(), 'Newer writing from another window.');
  assert.equal((await posts()).length, 1);
  await mode('cancel-lost');
  await field.fill('Cancellation keeps original writing.');
  await post.click();
  await idle();
  await page.evaluate(() => {
    const key = Object.keys(localStorage).find(
      (key) => key.startsWith('soda-issue-comment:') && JSON.parse(localStorage.getItem(key)!).actor === '11'
    )!;
    const row = JSON.parse(localStorage.getItem(key)!);
    row.writing.draft = {revision: 'c'.repeat(32), body: 'Newer writing after cancellation.'};
    localStorage.setItem(key, JSON.stringify(row));
  });
  await composer.getByRole('button', {name: 'Cancel submission', exact: true}).click();
  await idle();
  const beforeRestore = (await posts()).length;
  await remount();
  await idle();
  assert.equal(await composer.getByRole('button', {name: 'Retry saved comment', exact: true}).count(), 0);
  assert.equal(await composer.getByRole('button', {name: 'Retry cancellation', exact: true}).count(), 1);
  assert.equal((await posts()).length, beforeRestore, 'restore resubmitted a cancellation intent');
  await composer.getByRole('button', {name: 'Retry cancellation', exact: true}).click();
  await idle();
  await composer
    .getByText('No comment was posted. Your original text and a newer draft are both saved.', {exact: true})
    .waitFor();
  assert.equal(await field.inputValue(), 'Cancellation keeps original writing.');
  assert.equal(await field.evaluate((node) => (node as HTMLTextAreaElement).readOnly), true);
  await composer.getByText('Newer saved draft', {exact: true}).click();
  assert.equal(
    await composer.getByRole('textbox', {name: 'Newer saved draft', exact: true}).inputValue(),
    'Newer writing after cancellation.'
  );
  await remount();
  await idle();
  await composer
    .getByRole('button', {name: 'Discard original text and continue with newer draft', exact: true})
    .click();
  assert.equal(await field.inputValue(), 'Newer writing after cancellation.');
  await field.fill('Cancellation keeps original writing.');
  await mode('malformed');
  await post.click();
  await idle();
  await composer.getByText(/No final outcome was confirmed/).waitFor();
  assert.equal(
    await composer.getByRole('button', {name: 'Post comment', exact: true}).count(),
    0,
    'contradictory result released pending writing'
  );
  await mode('ready');
  await composer.getByRole('button', {name: 'Cancel submission', exact: true}).click();
  await idle();
  for (const value of ['denied', 'actor-mismatch', 'parent-mismatch']) {
    await mode(value);
    await composer.getByRole('button', {name: 'Check comment access', exact: true}).click();
    await idle();
    assert.equal(await post.isEnabled(), false, value);
    assert.equal(await field.inputValue(), 'Cancellation keeps original writing.');
  }
  await mode('ready');
  await composer.getByRole('button', {name: 'Check comment access', exact: true}).click();
  await idle();
  await field.fill('Alice private draft.');
  await mode('ready', '22');
  await remount();
  await idle();
  assert.equal(await field.inputValue(), '');
  await field.fill('Bob private draft.');
  await mode('ready', '11');
  await remount();
  await idle();
  assert.equal(await field.inputValue(), 'Alice private draft.');
  const beforeRealm = (await posts()).length;
  await page.goto(origin);
  await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
  await page.evaluate(async () => {
    window.drawerFixture = window.createDrawerFixture();
    await window.drawerFixture.api.ready;
  });
  await setup();
  await idle();
  assert.equal(await field.inputValue(), 'Alice private draft.');
  assert.equal((await posts()).length, 0, 'new realm submitted retained writing');
  assert(beforeRealm > 0);
  await mode('held');
  await field.fill('Original pending writing.');
  await post.click();
  await composer.locator('section[aria-busy=true]').waitFor();
  await page.waitForFunction(() =>
    window.drawerFixture.calls.some((call) => call.method === 'POST' && call.url.endsWith('/comments'))
  );
  await page.evaluate(() => {
    const key = Object.keys(localStorage).find(
      (key) => key.startsWith('soda-issue-comment:') && JSON.parse(localStorage.getItem(key)!).actor === '11'
    )!;
    const row = JSON.parse(localStorage.getItem(key)!);
    row.writing.draft = {revision: 'b'.repeat(32), body: 'Newer saved writing.'};
    localStorage.setItem(key, JSON.stringify(row));
    window.drawerFixture.repositoryHandle?.update?.({
      mode: 'page',
      route: '/alice/demo/feed/knowledge',
      params: {},
      visible: true,
    });
    window.dispatchEvent(new Event('comment-release'));
  });
  await mode('ready');
  await remount();
  await idle();
  assert.equal(await field.inputValue(), 'Newer saved writing.', 'receipt cleared another document’s newer draft');
  assert.equal((await posts()).length, 1, 'retired reply or restored pending intent dispatched again');
  await page.evaluate(() => {
    const key = Object.keys(localStorage).find(
      (key) => key.startsWith('soda-issue-comment:') && JSON.parse(localStorage.getItem(key)!).actor === '11'
    )!;
    localStorage.setItem(key, '{"unknown":"preserve"}');
  });
  await remount();
  await idle();
  assert.equal(await post.isEnabled(), false);
  await composer.getByRole('button', {name: 'Reload saved writing', exact: true}).click();
  await idle();
  assert.equal(
    await page.evaluate(() =>
      Object.keys(localStorage)
        .filter((key) => key.startsWith('soda-issue-comment:'))
        .some((key) => localStorage.getItem(key) === '{"unknown":"preserve"}')
    ),
    true
  );
  assert.equal((await posts()).length, 1);
  await page.evaluate(() => window.drawerFixture.repositoryHandle?.dispose?.());
});
