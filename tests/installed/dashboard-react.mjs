#!/usr/bin/env node
// Opt-in installed React OAuth/navigation check. No fixture/provider mutations
// beyond this user's native consent and Soda login/logout/session state.
// Optional SODA_RECONSENT_APPLICATION='SodaOS dashboard' explicitly revokes
// only the uniquely named grant in this user's native Applications UI.
// SODA_ADMIN_CONSENT=1 explicitly requests the native administrator scope.
// SODA_U08_FIXTURES_DIR explicitly enables the core-owned developer fixture
// mutations described in developer-first-workflow.mjs; it requires fresh inputs.
import assert from 'node:assert/strict';
import { readFile, lstat } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
assert.equal(process.env.SODA_NATIVE_VALIDATE, 'soda-test', 'Select the recorded isolated target explicitly');
const args = process.argv.slice(2);
assert.equal(args.length, 5, 'Supply Soda/Forgejo HTTPS origins, username, private password file and isolated browser home');
const [soda, forgejo, username, passwordFile, browserHome] = args;
const sodaURL = new URL(soda), forgejoURL = new URL(forgejo);
for (const value of [sodaURL, forgejoURL]) {
 assert.equal(value.protocol, 'https:'); assert.equal(value.pathname, '/');
 assert(!value.username && !value.password && !value.search && !value.hash);
}
assert.notEqual(sodaURL.origin, forgejoURL.origin);
assert(path.isAbsolute(passwordFile) && path.isAbsolute(browserHome));
const passwordStat = await lstat(passwordFile), homeStat = await lstat(browserHome);
assert(passwordStat.isFile() && (passwordStat.mode & 0o077) === 0);
assert(homeStat.isDirectory() && (homeStat.mode & 0o077) === 0);
const password = (await readFile(passwordFile, 'utf8')).trim(); assert(password);
const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
const { chromium } = require('playwright');
let browser, stage = 'browser startup';
try {
 browser = await chromium.launch({ headless: true, env: { ...process.env, HOME: browserHome, XDG_CONFIG_HOME: path.join(browserHome, '.config'), XDG_DATA_HOME: path.join(browserHome, '.local/share') } });
 const context = await browser.newContext({ locale: 'en-US' });
 const page = await context.newPage(); let renderingErrors = 0;
 page.on('pageerror', () => { renderingErrors++; });
 stage = 'React preview landing';
 assert.equal((await page.goto(sodaURL.origin + '/app/')).status(), 200);
 if (process.env.SODA_ADMIN_CONSENT === '1') await page.goto(sodaURL.origin + '/login?return_to=%2Fapp%2F&administration=1');
 else await page.getByRole('link', { name: 'Sign in with Forgejo', exact: true }).click();
 stage = 'native Forgejo authentication';
 await page.locator('#user_name').waitFor(); assert.equal(new URL(page.url()).origin, forgejoURL.origin);
 await page.locator('#user_name').fill(username); await page.locator('#password').fill(password);
 await page.getByRole('button', { name: 'Sign in', exact: true }).click();
 stage = 'native consent/callback';
 await page.waitForFunction(origin => (location.origin === origin && location.pathname === '/app/') || [...document.querySelectorAll('button')].some(button => button.textContent.trim() === 'Authorize Application'), sodaURL.origin);
 if (new URL(page.url()).origin === forgejoURL.origin) await page.getByRole('button', { name: 'Authorize Application', exact: true }).click();
 await page.waitForURL(sodaURL.origin + '/app/');
 if (process.env.SODA_RECONSENT_APPLICATION) {
  assert.equal(process.env.SODA_RECONSENT_APPLICATION, 'SodaOS dashboard');
  stage = 'explicit native Soda grant reconsent';
  await page.goto(forgejoURL.origin + '/user/settings/applications');
  const revoke = page.locator('.flex-item').filter({ has: page.locator('.flex-item-title').filter({ hasText: /^SodaOS dashboard$/ }) }).locator('button[data-modal-id="revoke-gitea-oauth2-grant"]');
  assert.equal(await revoke.count(), 1, 'Refuse ambiguous or missing Soda grant');
  await revoke.click();
  await page.locator('#revoke-gitea-oauth2-grant button.ok').click();
  await revoke.waitFor({ state: 'detached' });
  await page.goto(sodaURL.origin + '/login?return_to=%2Fapp%2F' + (process.env.SODA_ADMIN_CONSENT === '1' ? '&administration=1' : ''));
  await page.waitForFunction(origin => (location.origin === origin && location.pathname === '/app/') || [...document.querySelectorAll('button')].some(button => button.textContent.trim() === 'Authorize Application'), sodaURL.origin);
  if (new URL(page.url()).origin === forgejoURL.origin) await page.getByRole('button', { name: 'Authorize Application', exact: true }).click();
  await page.waitForURL(sodaURL.origin + '/app/');
  console.log('Explicit native reconsent replaced only this user’s uniquely named Soda grant.');
 }
 await page.getByRole('heading', { name: 'My work', exact: true }).waitFor();
 const cookie = (await context.cookies(sodaURL.origin)).find(value => value.name === 'soda_session');
 assert(cookie?.secure && cookie.httpOnly);
 const session = await page.evaluate(async () => {
  const response = await fetch('/api/session'); const body = await response.json();
  return { status: response.status, login: body.user?.login, idType: typeof body.user?.id };
 });
 assert.equal(session.status, 200); assert.equal(session.login, username); assert.equal(session.idType, 'string');
 console.log('Installed React OAuth/PKCE callback and secure session verified.');
 stage = 'acting-user native reads';
 for (const endpoint of ['/api/forgejo/me', '/api/forgejo/repositories', '/api/forgejo/notifications', ...(process.env.SODA_ADMIN_CONSENT === '1' ? ['/api/forgejo/admin/users'] : [])]) {
  const result = await page.evaluate(async endpoint => {
   const response = await fetch(endpoint); const body = await response.json();
   return { status: response.status, consentRequired: body?.error?.code === 'consent_required' };
  }, endpoint);
  if (result.status !== 200) throw new Error(`Native read ${endpoint} returned HTTP ${result.status}${result.consentRequired ? ' (explicit native reconsent required)' : ''}`);
 }
 stage = 'React profile and environment navigation';
 await page.getByRole('link', { name: 'Profile and development keys', exact: true }).click();
 await page.getByRole('heading', { name: 'Development-access public keys', exact: true }).waitFor();
 await page.getByRole('link', { name: 'Environments', exact: true }).click();
 await page.getByRole('heading', { name: 'Persistent environments', exact: true }).waitFor();
 await page.reload(); await page.getByRole('heading', { name: 'Persistent environments', exact: true }).waitFor();
 assert.equal(renderingErrors, 0, 'Installed React rendering failed');
 console.log('Acting-user native reads, React navigation and direct-link reload verified.');
 if (process.env.SODA_U08_FIXTURES_DIR) {
  assert.equal(process.env.SODA_ADMIN_CONSENT, '1');
  stage = 'core developer fixtures';
  const { runFirstWorkflow } = await import('./developer-first-workflow.mjs');
  await runFirstWorkflow({ browser, operatorPage: page, soda: sodaURL.origin, forgejo: forgejoURL.origin, fixtureDirectory: process.env.SODA_U08_FIXTURES_DIR });
 }
 if (process.env.SODA_U08_CONNECTIONS_DIR) {
  stage = 'project connection/public-key observations';
  const { observeProjectConnections } = await import('./project-connections.mjs');
  await observeProjectConnections({ page, username, fixtureDirectory: process.env.SODA_U08_CONNECTIONS_DIR });
 }
 if (process.env.SODA_U08_GIT_DIR) {
  stage = 'personal native Git key registration';
  const { registerPersonalGit } = await import('./personal-git.mjs');
  await registerPersonalGit({ page, soda: sodaURL.origin, username, directory: process.env.SODA_U08_GIT_DIR });
 }
 stage = 'Soda logout';
 await page.getByRole('button', { name: 'Sign out of Soda', exact: true }).click();
 await page.getByRole('link', { name: 'Sign in with Forgejo', exact: true }).waitFor();
 assert(!(await context.cookies(sodaURL.origin)).some(value => value.name === 'soda_session'));
 assert.equal(await page.evaluate(async () => (await fetch('/api/forgejo/me')).status), 401);
 console.log('Installed CSRF-protected logout cleared session/provider access. No client SSH/workload/persistence proof claimed.');
} catch (error) {
 const summary = error.message.split('\n')[0].replaceAll(password, '[redacted]').replace(/(https?:\/\/[^\s?]+)\?[^\s]*/g, '$1?[redacted]');
 console.error(`Installed React check failed during ${stage}: ${summary}`); process.exitCode = 1;
} finally { await browser?.close(); }
