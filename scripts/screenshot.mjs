#!/usr/bin/env node
// Local page captures with a dedicated, reusable manual-login profile.
import { mkdir, mkdtemp } from 'node:fs/promises';
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
    for (const [index, url] of urls.entries()) {
      const filename = path.join(output, `${String(index + 1).padStart(3, '0')}.png`);
      try {
        await page.goto(url, { waitUntil: 'load' });
        await page.evaluate(() => document.fonts.ready);
        await page.waitForTimeout(Number(values.wait));
        await page.screenshot({ path: filename });
      } catch {
        throw new Error(`Capture ${index + 1} failed. Earlier captures remain in ${output}`);
      }
      console.log(filename);
    }
  } finally { await context.close(); }
}

main().catch(error => { console.error(error.message); process.exitCode = 1; });
