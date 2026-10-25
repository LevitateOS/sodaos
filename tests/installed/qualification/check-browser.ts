// Provider-free integration of the built qualification module with real browser requests.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdir, readFile, realpath, stat, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {chromium} from 'playwright';
import {launchNativeBrowser} from '../native-browser';

process.umask(0o077);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, index) => index % 2 === 0),
  ['--module', '--retained-request', '--out'],
  'usage: bun check-browser.ts --module ABSOLUTE_MODULE --retained-request PUBLIC_REQUEST --out FRESH_HOME_DIRECTORY'
);
assert.equal(args.length, 6, 'three explicit inputs required');
const [moduleFile, retainedRequest, output] = [args[1]!, args[3]!, args[5]!];
for (const file of [moduleFile, retainedRequest]) {
  assert(path.isAbsolute(file), 'input paths must be absolute');
  assert.equal(await realpath(file), file, 'input paths must be canonical');
  assert((await stat(file)).isFile(), 'input must be a regular file');
}
assert(path.isAbsolute(output) && output.startsWith('/home/'), 'output must be under /home');
assert.equal(path.normalize(output), output, 'output must be canonical');
assert.equal(await realpath(path.dirname(output)), path.dirname(output), 'output parent must be canonical');
const browserDirectory = path.join(output, 'browser');
assert(Buffer.byteLength(path.join(browserDirectory, 'cdp.sock')) <= 103, 'output path exceeds browser socket limit');
const hash = (bytes: Uint8Array | string) => createHash('sha256').update(bytes).digest('hex');
const moduleSHA = hash(await readFile(moduleFile));
const requestBytes = await readFile(retainedRequest);
const requestSHA = hash(requestBytes);
const actual = JSON.parse(requestBytes.toString('utf8'));
assert.equal(actual.method, 'POST', 'saved request must be a current-context POST');
assert.equal(typeof actual.path, 'string', 'saved request path required');
const run = /^\/-\/native\/pages\/soda\/spaces\/api\/factory\/runs\/([a-f0-9]{32})\/actions$/.exec(actual.path)?.[1];
assert(run, 'saved request must identify one canonical Factory run action');
assert.equal(actual.body?.action, 'use_current_context', 'saved request must select current context');
const built = (await import(pathToFileURL(moduleFile).href)) as {
  FactoryPermits: typeof import('./phases/factory-phase').FactoryPermits;
};
assert.equal(typeof built.FactoryPermits, 'function', 'built module must export FactoryPermits');
await mkdir(output, {mode: 0o700});
await mkdir(browserDirectory, {mode: 0o700});
const browserHome = path.join(output, 'home');
await mkdir(browserHome, {mode: 0o700});
const origin = 'https://qualification.invalid';
const label = 'use-current-context-' + run;
const wrongID = run === '0'.repeat(32) ? '1'.repeat(32) : '0'.repeat(32);
const journal: Array<{name: string; sha256: string}> = [];
const permits = new built.FactoryPermits(origin, 1, async (name, value) => {
  journal.push({name, sha256: hash(JSON.stringify(value))});
});
await assert.rejects(permits.arm('use-current-context-' + wrongID, actual.method, actual.path, actual.body));
assert.equal(journal.length, 0);
await permits.arm(label, actual.method, actual.path, actual.body);
const browser = await launchNativeBrowser(chromium, browserDirectory, browserHome);
const observed: Array<{method: string; admitted: boolean; bodyRefused: boolean}> = [];
let fulfilled = 0;
try {
  const context = browser.context;
  await context.setOffline(true);
  await context.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === 'GET' && url.origin === origin && url.pathname === '/permit-fixture') {
      fulfilled++;
      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: '<!doctype html><title>Offline permit integration</title>',
      });
      return;
    }
    let admitted = false;
    let bodyRefused = false;
    try {
      admitted = await permits.admitRunAction(request);
    } catch {
      bodyRefused = true;
    }
    observed.push({method: request.method(), admitted, bodyRefused});
    fulfilled++;
    // Every request is fulfilled locally, including unexpected requests. Never continue or fall back.
    await route.fulfill({status: admitted ? 200 : 403, contentType: 'application/json', body: '{}'});
  });
  const page = await context.newPage();
  await page.goto(origin + '/permit-fixture');
  const send = (requestPath: string, body: unknown, method: 'POST' | 'PUT' = 'POST') =>
    page.evaluate(
      async ({requestPath, body, method}) => {
        const options = {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)};
        const response =
          method === 'POST'
            ? await fetch(requestPath, {...options, method: 'POST'})
            : await fetch(requestPath, {...options, method: 'PUT'});
        return response.status;
      },
      {requestPath, body, method}
    );
  assert.equal(await send(actual.path.replace('/pages/soda/', '/pages/another-owner/'), actual.body), 403);
  assert.equal(await send(actual.path.replace(run, wrongID), actual.body), 403);
  assert.equal(await send(actual.path, {...actual.body, action: 'retry'}), 403);
  const wrongCommand = actual.body.command_id === '0'.repeat(32) ? '1'.repeat(32) : '0'.repeat(32);
  assert.equal(await send(actual.path, {...actual.body, command_id: wrongCommand}), 403);
  assert.equal(await send(actual.path, actual.body, 'PUT'), 403);
  assert.equal(await send(actual.path + '?changed=1', actual.body), 403);
  assert.equal(await send(actual.path, actual.body), 200);
  assert.equal(await send(actual.path, actual.body), 403);
  await assert.rejects(permits.arm(label, actual.method, actual.path, actual.body));
  assert.equal(observed.length, 8);
  assert.equal(observed.filter((request) => request.admitted).length, 1);
  assert.equal(observed.filter((request) => request.bodyRefused).length, 2);
  assert.deepEqual(permits.used, [label]);
  assert.deepEqual([...permits.usedRunActionPaths], [actual.path]);
  assert.equal(journal.length, 1);
  assert.equal(fulfilled, 9);
} finally {
  await browser.close();
}
assert.equal(hash(await readFile(moduleFile)), moduleSHA, 'built module changed during integration');
assert.equal(hash(await readFile(retainedRequest)), requestSHA, 'saved request changed during integration');
const result = {
  format: 1,
  check: 'offline-browser-current-context-permit',
  passed: true,
  module_sha256: moduleSHA,
  retained_request_sha256: requestSHA,
  real_playwright_requests: observed.length,
  altered_owner_path_run_body_command_method_query_refused: true,
  mismatched_run_label_refused: true,
  exact_request_accepted_once: true,
  replay_and_rearm_refused: true,
  journal,
  browser_offline: true,
  all_requests_fulfilled_locally: true,
  network_continuations: 0,
  guest_requests: 0,
  guest_mutations: 0,
  model_calls: 0,
  backend_context_eligibility_or_resume_asserted: false,
};
await writeFile(path.join(output, 'result.json'), JSON.stringify(result, null, 2) + '\n', {flag: 'wx', mode: 0o600});
console.log(JSON.stringify(result));
