#!/usr/bin/env node
// Local page captures with a dedicated, reusable manual-login profile.
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { createRequire } from 'node:module';
import path from 'node:path';
import { createInterface } from 'node:readline/promises';
import { fileURLToPath } from 'node:url';
import { parseArgs } from 'node:util';

const root = fileURLToPath(new URL('../', import.meta.url));
const help = `Usage:
  node scripts/screenshot.mjs --login URL
  node scripts/screenshot.mjs [options] URL [URL ...]

Options:
  --login          Open Chrome for manual login; press Enter here when finished
  --profile DIR    Dedicated browser profile (default: .local/screenshot-profile)
  --out DIR        New output directory (default: fresh .artifacts/screenshots/capture-*)
  --width N        Screenshot width (default: 1440)
  --height N       Screenshot height (default: 1000)
  --wait N         Extra settling time in milliseconds (default: 1500)
  --scroll-top     Scroll to the page top after settling, before capture
  --local-css      Use this checkout's Soda CSS on localhost:3300; server templates stay unchanged
  --verify         Reject redirects, error responses, missing landmarks, stale Soda assets and browser errors
  --landmark CSS   Expected visible page selector (requires --verify)
  --theme NAME     Browser-only light/dark preview; does not change saved preferences
  --help           Show this help

Requires Node.js, the existing Playwright dependency, and Google Chrome.
Set CHROME to override the browser executable.
Captures the viewport, in URL order, as 001.png, 002.png, etc.
Keep the dedicated profile closed between runs. Fixtures are managed manually.`;

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
  const registrySource = await readFile(path.join(root, 'appliance/forgejo/templates/custom/header.tmpl'), 'utf8');
  const expectedRevision = registrySource.match(/name="soda-presentation-revision" content="([^"]+)"/)?.[1];
  const expectedStyles = [...registrySource.matchAll(/\/soda\/forgejo\/([a-z-]+\.css)\?v=([0-9]+)/g)].map(([, name, version]) => ({ name, version }));
  for (const option of ['width', 'height', 'wait']) {
    if (!/^\d+$/.test(values[option]) || !Number.isSafeInteger(Number(values[option])) || Number(values[option]) < 1) {
      throw new Error(`--${option} must be a positive integer.`);
    }
  }
  if (values.login && !process.stdin.isTTY) throw new Error('Run --login in an interactive terminal.');

  // Profiles contain authentication cookies; captures can contain private content.
  process.umask(0o077);
  const profile = path.resolve(values.profile);
  await mkdir(profile, { recursive: true, mode: 0o700 });
  let chromium;
  try {
    ({ chromium } = createRequire(new URL('../cockpit/package.json', import.meta.url))('playwright'));
  } catch {
    throw new Error('Playwright is missing. Prepare the existing Cockpit development dependencies, or expose an installed Playwright through NODE_PATH.');
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
  let localStyles;
  if (values['local-css']) {
    const header = await readFile(path.join(root, 'appliance/forgejo/templates/custom/header.tmpl'), 'utf8');
    const names = [...header.matchAll(/\/soda\/forgejo\/([a-z-]+\.css)\?v=/g)].map(([, name]) => name);
    localStyles = new Map(await Promise.all(names.map(async name => [
      name, await readFile(path.join(root, 'assets/branding/forgejo', name), 'utf8'),
    ])));
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
        if (!localStyles.has(name)) return route.continue();
        await route.fulfill({ status: 200, contentType: 'text/css', body: localStyles.get(name) });
      });
      console.log('Capture uses local Soda CSS; native server templates are unchanged.');
    }
    const page = context.pages()[0] || await context.newPage();
    page.setDefaultTimeout(30000);
    if (values.login) {
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
    const browserErrors = [];
    page.on('pageerror', error => browserErrors.push(error.message));
    page.on('console', message => { if (message.type() === 'error') browserErrors.push(message.text()); });
    for (const [index, url] of urls.entries()) {
      const filename = path.join(output, `${String(index + 1).padStart(3, '0')}.png`);
      try {
        browserErrors.length = 0;
        const response = await page.goto(url, { waitUntil: 'load' });
        if (values.verify) {
          if (!response || !response.ok()) throw new Error(`HTTP ${response?.status()}`);
          if (page.url() !== url) throw new Error('Requested URL redirected');
          await page.locator(values.landmark || '[role="main"], main').first().waitFor({ state: 'visible' });
          if (await page.locator('.soda-status, .error-code').count()) throw new Error('Unexpected status page');
          if (!expectedRevision || await page.locator('meta[name="soda-presentation-revision"]').getAttribute('content') !== expectedRevision) throw new Error('Stale template revision');
          const loaded = await page.locator('link[rel="stylesheet"]').evaluateAll(links => links.map(link => link.href));
          for (const {name, version} of expectedStyles) {
            const asset = `http://localhost:3300/assets/soda/forgejo/${name}?v=${version}`;
            if (!loaded.includes(asset)) throw new Error(`Stale stylesheet registry: ${name}`);
            const remote = await page.request.get(asset);
            const local = await readFile(path.join(root, 'assets/branding/forgejo', name));
            expectedStyles.find(style => style.name === name).sha256 = createHash('sha256').update(local).digest('hex');
            if (!remote.ok() || !(await remote.body()).equals(local)) throw new Error(`Stale stylesheet bytes: ${name}`);
          }
        }
        if (values.theme) {
          await page.evaluate(async theme => {
            const link = document.createElement('link');
            link.rel = 'stylesheet'; link.href = `/assets/css/theme-forgejo-${theme}.css`;
            await new Promise((resolve, reject) => { link.onload = resolve; link.onerror = reject; document.head.append(link); });
            document.documentElement.dataset.theme = `forgejo-${theme}`;
            document.documentElement.dataset.sodaLoginTheme = theme;
            document.documentElement.style.colorScheme = theme;
          }, values.theme);
        }
        if (localStyles) {
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
        await page.screenshot({ path: filename });
        if (values.verify) await writeFile(filename.replace('.png', '.json'), JSON.stringify({
          requestedURL: url, actualURL: page.url(), landmark: values.landmark || '[role="main"], main',
          status: response.status(), theme: values.theme || 'native',
          viewport: page.viewportSize(), browserErrors,
          presentationRevision: expectedRevision,
          registrySHA256: createHash('sha256').update(registrySource).digest('hex'),
          styles: expectedStyles, verifiedAt: new Date().toISOString(),
        }, null, 2));
      } catch (error) {
        throw new Error(`Capture ${index + 1} failed (${error.message}). Earlier captures remain in ${output}`);
      }
      console.log(filename);
    }
  } finally { await context.close(); }
}

main().catch(error => { console.error(error.message); process.exitCode = 1; });
