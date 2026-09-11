#!/usr/bin/env bun
// Local page captures with a dedicated, reusable manual-login profile.
import assert from 'node:assert/strict';
import { mkdir, mkdtemp, stat } from 'node:fs/promises';
import type {Page} from 'playwright';
import path from 'node:path';
import { createInterface } from 'node:readline/promises';
import { fileURLToPath } from 'node:url';
import { parseArgs } from 'node:util';

const root = fileURLToPath(new URL('../', import.meta.url));
const help = `Usage:
  bun scripts/screenshot.ts --login URL
  bun scripts/screenshot.ts [options] URL [URL ...]

Options:
  --login          Open Chrome for manual login; press Enter here when finished
  --profile DIR    Dedicated browser profile (default: .local/screenshot-profile)
  --out DIR        New output directory (default: fresh .artifacts/screenshots/capture-*)
  --width N        Screenshot width (default: 1440)
  --height N       Screenshot height (default: 1000)
  --wait N         Extra settling time in milliseconds (default: 1500)
  --scroll-top     Scroll to the page top after settling, before capture
  --full-page      Include offscreen content at the requested viewport width
  --local-css      Use this checkout's Soda CSS on localhost:3300; server templates stay unchanged
  --verify         Reject redirects, error responses, missing landmarks, stale Soda assets and browser errors
  --landmark CSS   Expected visible page selector (requires --verify)
  --theme NAME     Browser-only light/dark preview; does not change saved preferences
  --help           Show this help

Requires Bun 1.4.2, the existing Playwright dependency, and Google Chrome.
Set CHROME to override the browser executable.
Captures the viewport, in URL order, as 001.png, 002.png, etc.
Keep the dedicated profile closed between runs. Fixtures are managed manually.`;

// Shared by the CLI and already-authenticated native page fixture consumers.
// This changes only this document, never a saved account preference.
export async function setCaptureTheme(page: Page, theme: 'light' | 'dark') {
  await page.evaluate(async theme => {
    const link = document.createElement('link');
    link.rel = 'stylesheet'; link.href = `/assets/css/theme-forgejo-${theme}.css`;
    await new Promise((resolve, reject) => { link.onload = resolve; link.onerror = reject; document.head.append(link); });
    document.documentElement.dataset.theme = `forgejo-${theme}`;
    document.documentElement.dataset.sodaLoginTheme = theme;
    document.documentElement.style.colorScheme = theme;
    await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
    // Capture the selected theme, not an intermediate CSS transition. Keep
    // native animations enabled; infinite cursor/attention animations stay live.
    const finite = document.getAnimations().filter(animation => {
      if (!(animation instanceof CSSTransition)) return false;
      const end = animation.effect?.getComputedTiming().endTime;
      if (typeof end !== 'number' || !Number.isFinite(end)) return false;
      if (end > 5000) throw Error('Theme animation exceeds capture settling bound');
      return true;
    });
    await Promise.all(finite.map(animation => animation.finished.catch(() => undefined)));
  }, theme);
}

// Reuse the consumer's native login and synthetic operation model. Never launch a
// second browser, inject HTML, mask controls or manufacture installed evidence.
export async function capturePageFixture(page: Page, name: string, landmark: string, theme: 'light' | 'dark') {
  const directory = process.env.SODA_PAGE_CAPTURES;
  if (!directory) return;
  assert(path.isAbsolute(directory) && /^[a-z0-9-]+$/.test(name), 'Explicit fixture capture destination required');
  const info = await stat(directory);
  assert(info.isDirectory() && (info.mode & 0o077) === 0, 'Capture parent must be private');
  const url = new URL(page.url());
  assert(url.origin === process.env.SODA_PAGE_ORIGIN && url.protocol === 'https:' && url.hostname === '127.0.0.1', 'Only the selected local native fixture may be captured');
  assert(url.pathname === '/' && [...url.searchParams.keys()].every(key => ['soda-view', 'repository_id'].includes(key)), 'Do not capture login, consent or credential URLs');
  const out = path.join(directory, name);
  await mkdir(out, {mode: 0o700}); // Exclusive; retain earlier captures/failures.
  await page.locator(landmark).first().waitFor({state: 'visible'});
  assert.equal(await page.locator('#navbar').count(), 1, 'Native header required');
  assert.equal(await page.locator('.soda-status, .error-code').count(), 0, 'Unexpected native error page');
  for (const field of await page.locator('input[type=password]').all()) {
    assert((await field.inputValue()) === '', 'Credential-entry captures are not permitted');
  }
  await setCaptureTheme(page, theme);
  await page.evaluate(() => document.fonts.ready);
  await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
  await page.screenshot({path: path.join(out, 'viewport.png')});
  await Bun.write(path.join(out, 'capture.json'), JSON.stringify({
    scope: 'Local native-page fixture; synthetic operation and terminal data, not installed proof',
    view: url.searchParams.get('soda-view'), landmark, theme, viewport: page.viewportSize(),
    presentationRevision: await page.locator('meta[name="soda-presentation-revision"]').getAttribute('content'),
    scroll: await page.evaluate(() => ({x: scrollX, y: scrollY})),
    selectedTabStyles: await page.locator('[role=tab][aria-selected=true]').evaluateAll(tabs => tabs.map(tab => {
      const style = getComputedStyle(tab);
      return {foreground: style.color, background: style.backgroundColor, opacity: style.opacity};
    })),
    capturedAt: new Date().toISOString(),
  }, null, 2));
}

async function main() {
  const { values, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      login: { type: 'boolean' }, help: { type: 'boolean' },
      'local-css': { type: 'boolean' },
      verify: { type: 'boolean' },
      landmark: { type: 'string' },
      theme: { type: 'string' },
      'scroll-top': { type: 'boolean' },
      'full-page': { type: 'boolean' },
      profile: { type: 'string', default: path.join(root, '.local/screenshot-profile') },
      out: { type: 'string' },
      width: { type: 'string', default: '1440' },
      height: { type: 'string', default: '1000' },
      wait: { type: 'string', default: '1500' },
    },
  });
  if (values.help) return console.log(help);
  if (!positionals.length || (values.login && positionals.length !== 1)) {
    throw new Error('Supply one login URL, or one or more capture URLs. Use --help.');
  }
  const urls = positionals.map(value => {
    let url;
    try { url = new URL(value); } catch { throw new Error('Supply absolute HTTP(S) URLs.'); }
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password) {
      throw new Error('Supply HTTP(S) URLs without embedded credentials.');
    }
    return url.href;
  });
  if (values['local-css'] && (values.login || urls.some(url => new URL(url).origin !== 'http://localhost:3300'))) {
    throw new Error('--local-css is only for captures of the local Forgejo preview.');
  }
  if (values.theme && !['light', 'dark'].includes(values.theme)) throw new Error('--theme must be light or dark');
  if ((values.verify || values.theme) && urls.some(url => new URL(url).origin !== 'http://localhost:3300')) throw new Error('Verification and theme previews are local Forgejo only');
  if (values.landmark && !values.verify) throw new Error('--landmark requires --verify');
  if (values.verify && values['local-css']) throw new Error('Verified native captures must load server assets without --local-css');
  const registrySource = await Bun.file(path.join(root, 'appliance/forgejo/templates/custom/header.tmpl')).text();
  const expectedRevision = registrySource.match(/name="soda-presentation-revision" content="([^"]+)"/)?.[1];
  const expectedStyles = [...registrySource.matchAll(/\/soda\/forgejo\/([a-z-]+\.css)\?v=([0-9]+)/g)].map(([, name, version]): {name: string; version: string; sha256?: string} => { assert(name && version); return {name, version}; });
  for (const option of ['width', 'height', 'wait'] as const) {
    if (!/^\d+$/.test(values[option]) || !Number.isSafeInteger(Number(values[option])) || Number(values[option]) < 1) {
      throw new Error(`--${option} must be a positive integer.`);
    }
  }
  if (values.login && !process.stdin.isTTY) throw new Error('Run --login in an interactive terminal.');

  // Profiles contain authentication cookies; captures can contain private content.
  process.umask(0o077);
  const profile = path.resolve(values.profile);
  await mkdir(profile, { recursive: true, mode: 0o700 });
  let chromium: typeof import('playwright').chromium;
  try {
    ({ chromium } = await import('playwright'));
  } catch {
    throw new Error('Playwright is missing. Run bun install --frozen-lockfile from the repository root.');
  }
  let output;
  if (!values.login && values.out) {
    output = path.resolve(values.out);
    await mkdir(path.dirname(output), { recursive: true });
    await mkdir(output); // Never overwrite an earlier capture directory.
  } else if (!values.login) {
    const parent = path.join(root, '.artifacts/screenshots');
    await mkdir(parent, { recursive: true });
    output = await mkdtemp(path.join(parent, 'capture-'));
  }
  let context;
  let localStyles = new Map<string, string>();
  if (values['local-css']) {
    const header = await Bun.file(path.join(root, 'appliance/forgejo/templates/custom/header.tmpl')).text();
    const names = [...header.matchAll(/\/soda\/forgejo\/([a-z-]+\.css)\?v=/g)].map(([, name]) => { assert(name); return name; });
    localStyles = new Map(await Promise.all(names.map(async name => [
      name, await Bun.file(path.join(root, 'assets/branding/forgejo', name)).text(),
    ] as const)));
  }
  try {
    context = await chromium.launchPersistentContext(profile, {
      ...(process.env.CHROME ? { executablePath: process.env.CHROME } : { channel: 'chrome' }),
      headless: !values.login,
      chromiumSandbox: true,
      viewport: { width: Number(values.width), height: Number(values.height) },
      deviceScaleFactor: 1,
    });
  } catch {
    throw new Error('Cannot launch Chrome. Check CHROME and that the dedicated profile is closed.');
  }
  try {
    if (values['local-css']) {
      // Browser-only stylesheet substitution: keep the fixture, live checkout,
      // native HTML, scripts and handlers untouched while reviewing a worktree.
      await context.route('http://localhost:3300/assets/soda/forgejo/*.css*', async route => {
        const name = new URL(route.request().url()).pathname.split('/').pop();
        if (!name) return route.continue();
        const body = localStyles.get(name); if (body === undefined) return route.continue();
        await route.fulfill({ status: 200, contentType: 'text/css', body });
      });
      console.log('Capture uses local Soda CSS; native server templates are unchanged.');
    }
    const page = context.pages()[0] || await context.newPage();
    page.setDefaultTimeout(30000);
    if (values.login) {
      assert(urls[0]);
      await page.goto(urls[0], { waitUntil: 'domcontentloaded' });
      const terminal = createInterface({ input: process.stdin, output: process.stdout });
      try {
        await Promise.race([
          terminal.question('Log in in Chrome, then press Enter here. '),
          new Promise(resolve => context.once('close', resolve)),
        ]);
      } finally { terminal.close(); }
      console.log(`Profile kept at ${profile}`);
      return;
    }
    const browserErrors: string[] = [];
    page.on('pageerror', error => browserErrors.push((error instanceof Error ? error.message : String(error))));
    page.on('console', message => { if (message.type() === 'error') browserErrors.push(message.text()); });
    for (const [index, url] of urls.entries()) {
      assert(output);
      const filename = path.join(output, `${String(index + 1).padStart(3, '0')}.png`);
      try {
        // Fragment-only navigation returns no HTTP response. Start each capture
        // from a fresh document so --verify always checks an actual native GET.
        await page.goto('about:blank');
        browserErrors.length = 0;
        const response = await page.goto(url, { waitUntil: 'load' });
        if (values.verify) {
          if (!response || !response.ok()) throw new Error(`HTTP ${response?.status()}`);
          if (page.url() !== url) throw new Error('Requested URL redirected');
          await page.locator(values.landmark || '[role="main"], main').first().waitFor({ state: 'visible' });
          if (await page.locator('.soda-status, .error-code').count()) throw new Error('Unexpected status page');
          if (!expectedRevision || await page.locator('meta[name="soda-presentation-revision"]').getAttribute('content') !== expectedRevision) throw new Error('Stale template revision');
          const loaded = await page.locator('link[rel="stylesheet"]').evaluateAll((links: HTMLLinkElement[]) => links.map(link => link.href));
          for (const style of expectedStyles) {
            const {name, version} = style;
            const asset = `http://localhost:3300/assets/soda/forgejo/${name}?v=${version}`;
            if (!loaded.includes(asset)) throw new Error(`Stale stylesheet registry: ${name}`);
            const remote = await page.request.get(asset);
            const local = await Bun.file(path.join(root, 'assets/branding/forgejo', name)).bytes();
            style.sha256 = new Bun.CryptoHasher('sha256').update(local).digest('hex');
            if (!remote.ok() || !(await remote.body()).equals(local)) throw new Error(`Stale stylesheet bytes: ${name}`);
          }
        }
        if (values.theme === 'light' || values.theme === 'dark') await setCaptureTheme(page, values.theme);
        if (values['local-css']) {
          // Use the candidate registry/order too, including added or removed sheets.
          await page.evaluate(async names => {
            document.querySelectorAll('link[rel="stylesheet"][href*="/assets/soda/forgejo/"]').forEach(link => link.remove());
            await Promise.all(names.map(name => new Promise((resolve, reject) => {
              const link = document.createElement('link');
              link.rel = 'stylesheet';
              link.href = `/assets/soda/forgejo/${name}`;
              link.onload = resolve;
              link.onerror = reject;
              document.head.append(link);
            })));
          }, [...localStyles.keys()]);
        }
        await page.evaluate(() => document.fonts.ready);
        await page.waitForTimeout(Number(values.wait));
        if (values['scroll-top']) await page.evaluate(() => window.scrollTo({ top: 0, left: 0, behavior: 'instant' }));
        if (values.verify && browserErrors.length) throw new Error(`Browser errors: ${browserErrors.join('; ')}`);
        await page.screenshot({ path: filename, fullPage: Boolean(values['full-page']) });
        if (values.verify) await Bun.write(filename.replace('.png', '.json'), JSON.stringify({
          requestedURL: url, actualURL: page.url(), landmark: values.landmark || '[role="main"], main',
          status: response?.status(), theme: values.theme || 'native',
          viewport: page.viewportSize(), fullPage: Boolean(values['full-page']), browserErrors,
          presentationRevision: expectedRevision,
          registrySHA256: new Bun.CryptoHasher('sha256').update(registrySource).digest('hex'),
          styles: expectedStyles, verifiedAt: new Date().toISOString(),
        }, null, 2));
      } catch (error) {
        throw new Error(`Capture ${index + 1} failed (${(error instanceof Error ? error.message : String(error))}). Earlier captures remain in ${output}`);
      }
      console.log(filename);
    }
  } finally { await context.close(); }
}

if (import.meta.main) main().catch((error: unknown) => { console.error((error instanceof Error ? error.message : String(error))); process.exitCode = 1; });
