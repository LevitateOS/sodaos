import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import path from 'node:path';
import test from 'node:test';
import {chromium} from 'playwright';
import payload from '../../assets/branding/forgejo/forgejo-payload.json';
import {buildForgejoModule} from '../../scripts/build-forgejo';

const root = path.resolve(import.meta.dirname, '../..');
const installedOrigin = process.env.SODA_INSTALLED_ORIGIN;
const runtimePath = path.join(root, '.artifacts/forgejo-js/lit.js');
const appSubURL = '/native';
const runtimeURL = `${appSubURL}/assets/soda/forgejo/lit.js`;
const fixturePath = path.join(root, 'tests/forgejo/fixtures/lit-smoke.ts');
let fixtureBuild: Promise<{branding: string; publicRoot: string}> | undefined;

function buildFixtureModules() {
  return (fixtureBuild ??= Promise.all([
    buildForgejoModule(fixturePath, 'public/assets/soda/forgejo/lit-smoke.js'),
    buildForgejoModule(fixturePath, 'public/assets/lit-smoke.js'),
  ]).then(async ([branding, publicRoot]) => ({
    branding: await branding.text(),
    publicRoot: await publicRoot.text(),
  })));
}

test('Lit production runtime is staged as a self-contained browser module', async () => {
  assert.equal(payload['public/assets/soda/forgejo/lit.js'], '@build/forgejo-js/lit.js');
  const source = await readFile(runtimePath, 'utf8');
  assert(source.length > 0, 'emitted Lit runtime must not be empty');
  assert.doesNotMatch(
    source,
    /\b(?:from\s*|import\s*\()\s*["'](?:lit|@lit\/)/,
    'runtime must bundle its package imports'
  );
  const fixtures = await buildFixtureModules();
  assert.match(fixtures.branding, /from["']\.\/lit\.js\?v=[a-zA-Z0-9.-]+["']/);
  assert.match(fixtures.publicRoot, /from["']\.\/soda\/forgejo\/lit\.js\?v=[a-zA-Z0-9.-]+["']/);
  assert.doesNotMatch(fixtures.branding + fixtures.publicRoot, /from["']lit["']/);
});

test(
  'Lit production runtime upgrades and updates independent elements in Chromium',
  {
    skip: process.env.SODA_LIT_BROWSER !== '1',
  },
  async (t) => {
    const fixtures = await buildFixtureModules();
    const requests: string[] = [];
    const responses: string[] = [];
    const server = Bun.serve({
      hostname: '127.0.0.1',
      port: 0,
      fetch(request) {
        const url = new URL(request.url);
        requests.push(url.pathname);
        if (url.pathname === runtimeURL) {
          return new Response(Bun.file(runtimePath), {headers: {'Content-Type': 'text/javascript'}});
        }
        if (url.pathname === `${appSubURL}/assets/soda/forgejo/lit-smoke.js`) {
          return new Response(fixtures.branding, {headers: {'Content-Type': 'text/javascript'}});
        }
        if (url.pathname === `${appSubURL}/assets/lit-smoke.js`) {
          return new Response(fixtures.publicRoot, {headers: {'Content-Type': 'text/javascript'}});
        }
        if (url.pathname === `${appSubURL}/`) {
          return new Response(
            `<!doctype html>
          <html><head><link rel="icon" href="data:,">
            <script type="module" src="${appSubURL}/assets/soda/forgejo/lit-smoke.js"></script>
            <script type="module" src="${appSubURL}/assets/lit-smoke.js"></script>
          </head>
          <body>
            <form id="native-form"><label>Native input <input id="native-input" value="preserved"></label><button id="native-control" type="button">Native control</button></form>
            <soda-lit-smoke label="First"></soda-lit-smoke>
            <soda-lit-smoke label="Second"></soda-lit-smoke>
            <div id="lit-standalone"></div>
          </body></html>`,
            {headers: {'Content-Type': 'text/html'}}
          );
        }
        return new Response('Not found', {status: 404});
      },
    });
    t.after(() => server.stop(true));

    const browser = await chromium.launch({headless: true, chromiumSandbox: true});
    t.after(() => browser.close());
    const page = await browser.newPage();
    const browserErrors: string[] = [];
    page.on('pageerror', (error) => browserErrors.push(error.message));
    page.on('console', (message) => {
      if (message.type() === 'error') browserErrors.push(message.text());
    });
    page.on('requestfailed', (request) =>
      browserErrors.push(`${request.method()} ${request.url()}: ${request.failure()?.errorText ?? 'failed'}`)
    );
    page.on('response', (response) => {
      if (response.status() >= 400) responses.push(`${response.status()} ${new URL(response.url()).pathname}`);
    });

    await page.goto(new URL(`${appSubURL}/`, server.url).href);
    await page.locator('body[data-lit-smoke-modules="2"]').waitFor();
    assert.equal(await page.locator('#lit-standalone').innerText(), 'standalone runtime render');
    assert.equal(await page.locator('soda-lit-smoke').count(), 2);

    const nativeForm = page.locator('#native-form');
    const nativeMarkup = await nativeForm.evaluate((form) => form.outerHTML);
    const nativeInput = page.locator('#native-input');
    await nativeInput.fill('unsaved native value');
    const first = page.locator('soda-lit-smoke').nth(0).locator('button');
    const second = page.locator('soda-lit-smoke').nth(1).locator('button');
    assert.equal(await first.innerText(), 'First: 0');
    assert.equal(await second.innerText(), 'Second: 0');
    const keyed = await page.locator('soda-lit-smoke').first().locator('[data-key=a]').elementHandle();
    assert(keyed);
    await first.click();
    assert.equal(await page.locator('soda-lit-smoke').first().locator('.keyed').innerText(), 'ba');
    assert(await keyed.evaluate((el) => el.isConnected));
    await first.click();
    await second.click();
    assert.equal(await first.innerText(), 'First: 2');
    assert.equal(await second.innerText(), 'Second: 1');

    assert.deepEqual(
      await nativeForm.evaluate((form) => ({
        childCount: form.children.length,
        connected: form.isConnected,
        markup: form.outerHTML,
        inputValue: form.querySelector<HTMLInputElement>('#native-input')?.value,
        controlText: form.querySelector<HTMLButtonElement>('#native-control')?.textContent,
      })),
      {
        childCount: 2,
        connected: true,
        markup: nativeMarkup,
        inputValue: 'unsaved native value',
        controlText: 'Native control',
      }
    );

    assert.deepEqual(
      await page.evaluate(() => {
        const versions = globalThis as typeof globalThis & {
          litElementVersions?: string[];
          litHtmlVersions?: string[];
          reactiveElementVersions?: string[];
        };
        return {
          registered: typeof customElements.get('soda-lit-smoke') === 'function',
          litElement: versions.litElementVersions?.length,
          litHtml: versions.litHtmlVersions?.length,
          reactiveElement: versions.reactiveElementVersions?.length,
        };
      }),
      {registered: true, litElement: 1, litHtml: 1, reactiveElement: 1}
    );
    assert.deepEqual(browserErrors, []);
    assert.deepEqual(responses, []);
    assert.deepEqual(
      new Set(requests),
      new Set([
        `${appSubURL}/`,
        `${appSubURL}/assets/soda/forgejo/lit-smoke.js`,
        `${appSubURL}/assets/lit-smoke.js`,
        runtimeURL,
      ])
    );
    assert.equal(
      requests.filter((request) => request === runtimeURL).length,
      1,
      'both modules must share one browser runtime request'
    );
  }
);

test(
  'Installed appliance serves its first-party assets in Chromium',
  {
    skip: !installedOrigin && process.env.SODA_INSTALLED_GATE !== '1',
  },
  async (t) => {
    assert(installedOrigin, 'SODA_INSTALLED_ORIGIN is required for the installed-asset gate');
    const origin = new URL(installedOrigin);
    assert(
      (origin.protocol === 'http:' || origin.protocol === 'https:') && !origin.username && !origin.password,
      'installed origin must be an http(s) URL without credentials'
    );
    const browser = await chromium.launch({headless: true, chromiumSandbox: true});
    t.after(() => browser.close());
    const page = await browser.newPage();
    const missing: string[] = [];
    let firstPartyRequests = 0;
    page.on('response', (response) => {
      const url = new URL(response.url());
      if (url.origin !== origin.origin || !url.pathname.startsWith('/assets/soda/')) return;
      firstPartyRequests++;
      if (response.status() >= 400) missing.push(`${response.status()} ${url.pathname}`);
    });
    const home = await page.goto(origin.origin + '/', {waitUntil: 'load', timeout: 30000});
    assert.equal(home?.status(), 200, 'installed home page must load');
    // Deterministic probes for payload-staged first-party assets: the public
    // home page logo and the shared Lit runtime bundle.
    for (const asset of ['/assets/soda/source/soda-symbol-brutalist.svg', '/assets/soda/forgejo/lit.js']) {
      const response = await page.request.get(origin.origin + asset, {timeout: 15000});
      assert.equal(response.status(), 200, `installed first-party asset must be served: ${asset}`);
      assert((await response.body()).length > 0, `installed first-party asset must not be empty: ${asset}`);
    }
    assert(firstPartyRequests > 0, 'home page must exercise first-party assets');
    assert.deepEqual(missing, []);
  }
);
