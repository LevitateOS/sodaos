#!/usr/bin/env node
// Opt-in native browser login check. Creates login/consent/session state only;
// never creates people, repositories, projects, keys, workloads or providers.
import assert from 'node:assert/strict';
import { readFile, stat } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';

const args = process.argv.slice(2);
if (args.length !== 5) {
  console.error('usage: node tests/installed/dashboard.mjs SODA_HTTPS_ORIGIN FORGEJO_HTTPS_ORIGIN OPERATOR_USERNAME ABSOLUTE_PASSWORD_FILE ISOLATED_BROWSER_HOME');
  process.exit(2);
}
const [soda, forgejo, username, passwordFile, browserHome] = args;
const sodaURL = new URL(soda), forgejoURL = new URL(forgejo);
for (const url of [sodaURL, forgejoURL]) {
  assert.equal(url.protocol, 'https:');
  assert.equal(url.pathname, '/');
  assert(!url.username && !url.password && !url.search && !url.hash, 'Provide a plain HTTPS origin');
}
assert.notEqual(sodaURL.origin, forgejoURL.origin, 'Select distinct browser origins');
assert(path.isAbsolute(passwordFile) && path.isAbsolute(browserHome), 'Use explicit absolute private file/directory paths');
const passwordStat = await stat(passwordFile), homeStat = await stat(browserHome);
assert(passwordStat.isFile() && (passwordStat.mode & 0o077) === 0, 'Password file must be private');
assert(homeStat.isDirectory() && (homeStat.mode & 0o077) === 0, 'Use a private isolated browser home with the test CA already trusted');
const password = (await readFile(passwordFile, 'utf8')).trim();
assert(password, 'Password file is empty');
const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
const { chromium } = require('playwright');
let stage = 'browser startup';
let browser;
try {
  browser = await chromium.launch({
    headless: true,
    env: {
      ...process.env,
      HOME: browserHome,
      XDG_CONFIG_HOME: path.join(browserHome, '.config'),
      XDG_DATA_HOME: path.join(browserHome, '.local/share'),
    },
  });
  // Playwright uses a fresh temporary browser profile. No TLS bypasses,
  // saved cookies, password screenshots or traces are used.
  const context = await browser.newContext({ locale: 'en-US' });
  const page = await context.newPage();
  stage = 'Soda landing page';
  const response = await page.goto(sodaURL.origin + '/');
  assert.equal(response.status(), 200);
  await page.getByRole('link', { name: 'Sign in with Forgejo', exact: true }).click();
  stage = 'Forgejo authentication';
  await page.locator('#user_name').waitFor();
  assert.equal(new URL(page.url()).origin, forgejoURL.origin);
  await page.locator('#user_name').fill(username);
  await page.locator('#password').fill(password);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  stage = 'Forgejo consent and Soda OAuth callback';
  // Forgejo may remember the user's previous consent; both are native flows.
  await page.waitForFunction(origin =>
    (location.origin === origin && location.pathname === '/projects') ||
    [...document.querySelectorAll('button')].some(button => button.textContent.trim() === 'Authorize Application'),
  sodaURL.origin);
  if (new URL(page.url()).origin === forgejoURL.origin) {
    await page.getByRole('button', { name: 'Authorize Application', exact: true }).click();
  }
  await page.waitForURL(sodaURL.origin + '/projects');
  await page.getByRole('heading', { name: 'Projects', exact: true }).waitFor();
  await page.getByText('Signed in as ' + username, { exact: true }).waitFor();
  const session = (await context.cookies(sodaURL.origin)).find(cookie => cookie.name === 'soda_session');
  assert(session?.secure && session.httpOnly, 'Soda must establish a secure HTTP-only session');
  console.log('Native Forgejo OAuth/PKCE login reached the authenticated Soda Projects page.');

  stage = 'repository picker';
  assert.equal(await page.locator('input[name="repository"]').count(), 0, 'Repository entry must use the picker, not manual text');
  const picker = page.getByRole('combobox', { name: 'Forgejo repository', exact: true });
  if (await picker.count()) {
    const values = await picker.locator('option').evaluateAll(options => options.map(option => option.value).filter(Boolean));
    assert(values.length > 0, 'Repository picker has no selectable repositories');
    await picker.selectOption(values[0]);
    assert.equal(await picker.inputValue(), values[0]);
    console.log('Available Forgejo repositories are selectable without typing; no environment was created.');
  } else {
    await page.getByText('No repositories available to create an environment.', { exact: false }).waitFor();
    console.log('Empty repository picker shows native Forgejo creation guidance.');
  }
  assert.equal(await page.getByRole('link', { name: 'Create a repository in Forgejo', exact: true }).getAttribute('href'), forgejoURL.origin + '/repo/create');
  await page.getByRole('link', { name: 'Refresh repositories', exact: true }).click();
  await page.getByRole('heading', { name: 'Projects', exact: true }).waitFor();

  stage = 'Profile navigation';
  await page.getByRole('link', { name: 'Profile', exact: true }).click();
  await page.getByRole('heading', { name: 'Your profile', exact: true }).waitFor();
  await page.getByRole('heading', { name: 'Development SSH keys', exact: true }).waitFor();
  stage = 'operator People navigation';
  await page.getByRole('link', { name: 'People', exact: true }).click();
  await page.getByRole('heading', { name: 'People', exact: true }).waitFor();
  await page.getByRole('button', { name: 'Create person', exact: true }).waitFor();
  console.log('Profile and operator People pages are reachable through the dashboard.');

  stage = 'Soda sign-out';
  await page.getByRole('button', { name: 'Sign out', exact: true }).click();
  await page.getByRole('link', { name: 'Sign in with Forgejo', exact: true }).waitFor();
  assert(!(await context.cookies(sodaURL.origin)).some(cookie => cookie.name === 'soda_session'));
  console.log('CSRF-protected sign-out cleared the Soda session.');
} catch (error) {
  // Native redirect URLs contain OAuth codes/state. Never put query strings,
  // credentials, page contents or full browser call logs into evidence.
  const summary = error.message.split('\n')[0].replaceAll(password, '[redacted]')
    .replace(/(https?:\/\/[^\s?]+)\?[^\s]*/g, '$1?[redacted]');
  console.error(`Dashboard browser check failed during ${stage}: ${summary}`);
  process.exitCode = 1;
} finally {
  await browser?.close();
}
