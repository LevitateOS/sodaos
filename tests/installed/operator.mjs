#!/usr/bin/env node
// P11 only: existing root Cockpit/Tailnet/Runners, no dashboard/user fixtures.
// Opening Tailnet can invoke its existing Forgejo advertisement refresh effect.
import assert from 'node:assert/strict';
import { lstat, mkdir, readFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';

const args = process.argv.slice(2);
assert.equal(args.length, 5, 'usage: operator.mjs HTTPS_ORIGIN PASSWORD_FILE PRIVATE_BROWSER_HOME HOSTNAME --allow-advertisement-refresh');
const [origin, passwordFile, browserHome, hostname, permission] = args;
assert.equal(permission, '--allow-advertisement-refresh', 'Tailnet page effect must be explicitly selected');
const url = new URL(origin);
assert.equal(url.protocol, 'https:');
assert(!url.username && !url.password && !url.search && !url.hash && url.pathname === '/');
assert(path.isAbsolute(passwordFile) && path.isAbsolute(browserHome));
const secretStat = await lstat(passwordFile), homeStat = await lstat(browserHome);
assert(secretStat.isFile() && secretStat.size <= 65536 && (secretStat.mode & 0o077) === 0);
assert(homeStat.isDirectory() && (homeStat.mode & 0o077) === 0);
const password = (await readFile(passwordFile, 'utf8')).replace(/\r?\n$/, '');
assert(password && !/[\r\n]/.test(password));
const profile = path.join(browserHome, 'cockpit-profile');
await mkdir(profile, { mode: 0o700 }); // exclusive; no inherited login/cookie state
const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
const { chromium } = require('playwright');
let context;
let stage = 'trusted browser startup';
let interrupted = false;
const interrupt = async () => {
  interrupted = true;
  process.exitCode = 1;
  try { await context?.close(); } catch { /* main/finally reports failure */ }
};
process.once('SIGTERM', interrupt);
process.once('SIGINT', interrupt);
try {
  context = await chromium.launchPersistentContext(profile, {
    headless: true,
    env: { ...process.env, HOME: browserHome, XDG_CONFIG_HOME: path.join(browserHome, '.config'), XDG_DATA_HOME: path.join(browserHome, '.local/share') },
  });
  context.setDefaultTimeout(30_000);
  const page = await context.newPage();
  stage = 'root Cockpit password login';
  await page.goto(url.origin + '/');
  await page.locator('#login-user-input').fill('root');
  await page.locator('#login-password-input').fill(password);
  await page.locator('#login-button').click();
  await page.getByRole('link', { name: 'Tailscale', exact: true }).waitFor();
  assert.equal(await page.getByRole('link', { name: 'Accounts', exact: true }).count(), 0, 'Accounts navigation is not hidden (PAM gate is checked separately)');

  stage = 'native target and core origin snapshot before page effects';
  const originProbe = 'import json; x=json.load(open("/etc/soda/dashboard.json")); print(json.dumps({k:x[k] for k in ("forgejo_url","forgejo_internal_url")},sort_keys=True))';
  await page.waitForFunction(() => [window, ...[...document.querySelectorAll('iframe')].map(frame => frame.contentWindow)].some(candidate => candidate?.cockpit));
  const before = await page.evaluate(async script => {
    const candidate = [window, ...[...document.querySelectorAll('iframe')].map(frame => frame.contentWindow)].find(value => value?.cockpit);
    const c = candidate.cockpit;
    return { host: (await c.spawn(['hostname'], { err: 'message' })).trim(), origins: await c.spawn(['python3', '-c', script], { err: 'message' }) };
  }, originProbe);
  assert.equal(before.host, hostname);

  async function openPackage(label, name) {
    await page.getByRole('link', { name: label, exact: true }).click();
    await page.waitForFunction(suffix => [...document.querySelectorAll('iframe')].some(frame => frame.src && new URL(frame.src).pathname.endsWith(suffix)), `/${name}/index.html`);
    const frame = page.frames().find(candidate => candidate.url() && new URL(candidate.url()).pathname.endsWith(`/${name}/index.html`));
    assert(frame, 'native Cockpit package frame not found');
    await frame.getByRole('heading', { name: label, exact: true }).waitFor();
    return frame;
  }
  stage = 'Tailnet native root session and SELinux/socket access';
  const tailnet = await openPackage('Tailscale', 'soda-tailscale');
  const native = await tailnet.evaluate(async () => {
    const c = window.cockpit;
    const uid = (await c.spawn(['id', '-u'], { err: 'message' })).trim();
    const host = (await c.spawn(['hostname'], { err: 'message' })).trim();
    const domain = (await c.spawn(['id', '-Z'], { err: 'message' })).trim();
    const enforcing = (await c.spawn(['getenforce'], { err: 'message' })).trim();
    const status = JSON.parse(await c.spawn(['/usr/bin/tailscale', 'status', '--json'], { err: 'message' }));
    const http = c.http('/var/run/tailscale/tailscaled.sock', { superuser: 'require', headers: { Host: 'local-tailscaled.sock' } });
    try {
      const prefs = JSON.parse(await http.get('/localapi/v0/prefs'));
      return { uid, host, domain, enforcing, backend: status.BackendState, wantRunning: prefs.WantRunning };
    } finally { http.close(); }
  });
  assert.equal(native.uid, '0');
  assert.equal(native.host, hostname);
  assert.equal(native.enforcing, 'Enforcing');
  assert(native.domain.includes(':') && !native.domain.includes(':cockpit_session_t:'), 'root stayed in the restricted preauthentication SELinux domain');
  assert.equal(typeof native.backend, 'string');
  assert.equal(typeof native.wantRunning, 'boolean');
  console.log('Native root Cockpit session read Tailnet status/preferences with its native SELinux transition. No enrollment/exit-node/route action was requested.');

  stage = 'Runners native read path';
  const runners = await openPackage('Runners', 'soda-runners');
  const summary = await runners.evaluate(async () => {
    const raw = await window.cockpit.spawn(['/usr/local/libexec/soda/soda-runners', 'list'], { err: 'message' }).input('{}\n');
    const data = JSON.parse(raw);
    return { count: data.runner_count, listeners: data.active_listeners, capacity: data.total_capacity };
  });
  assert(Object.values(summary).every(Number.isInteger));
  console.log('Native Cockpit runner list/capacity read completed. Registration/lifecycle/trusted provider jobs remain separate.');
  assert.equal(await runners.evaluate(script => window.cockpit.spawn(['python3', '-c', script], { err: 'message' }), originProbe), before.origins, 'Core browser origins changed during retained page effects');
  stage = 'Cockpit sign-out';
  await runners.evaluate(() => window.cockpit.logout(true));
  await page.locator('#login-user-input').waitFor({ state: 'visible' });
  assert(!interrupted);
} catch (error) {
  // Never retain browser call logs, bodies, cookies, queries or password text.
  console.error(`Operator browser check failed during ${stage} (${error?.name || 'Error'}).`);
  process.exitCode = 1;
} finally {
  try { await context?.close(); } catch { console.error('Operator browser process cleanup failed.'); process.exitCode = 1; }
  process.removeListener('SIGTERM', interrupt);
  process.removeListener('SIGINT', interrupt);
}
