// Core-owned, opt-in U08 browser mutations on exact run-owned fixtures.
// This does not claim client SSH, shared tools, nested workload or persistence proof.
import assert from 'node:assert/strict';
import { readFile, writeFile, lstat } from 'node:fs/promises';
import path from 'node:path';
import { createHash } from 'node:crypto';

export async function runFirstWorkflow({ browser, operatorPage, soda, forgejo, fixtureDirectory }) {
 assert.equal(process.env.SODA_NATIVE_VALIDATE, 'soda-test');
 assert(path.isAbsolute(fixtureDirectory));
 const root = await lstat(fixtureDirectory); assert(root.isDirectory() && root.uid === process.getuid() && !(root.mode & 0o077));
 const fixture = JSON.parse(await readFile(path.join(fixtureDirectory, 'fixture.json'), 'utf8'));
 assert.equal(fixture.people.length, 2);
 const secrets = []; const contexts = []; const observed = [];
 let stage = 'fixture input validation';
 async function saveBindings() { await writeFile(path.join(fixtureDirectory, 'observed-bindings.json'), JSON.stringify(observed, null, 2), { mode: 0o600 }); }
 try {
  for (const person of fixture.people) {
   assert(/^u08-(alice|bob)-[0-9a-f]{4}$/.test(person.login));
   assert(/^[a-z0-9-]+$/.test(person.repository));
   assert(path.dirname(person.directory) === fixtureDirectory);
   const dir = await lstat(person.directory); assert(dir.isDirectory() && dir.uid === process.getuid() && !(dir.mode & 0o077));
   for (const field of ['initial-password', 'password']) {
    const file = path.join(person.directory, field); const st = await lstat(file); assert(st.isFile() && st.uid === process.getuid() && !(st.mode & 0o077));
    person[field] = (await readFile(file, 'utf8')).trim(); assert(person[field]); secrets.push(person[field]);
   }
   person.publicKey = (await readFile(path.join(person.directory, 'development.pub'), 'utf8')).trim();
   assert(person.publicKey.startsWith('ssh-ed25519 '));
  }
  if (process.env.SODA_U08_RESUME_FIXTURES === '1') {
   const retained = JSON.parse(await readFile(path.join(fixtureDirectory, 'observed-bindings.json'), 'utf8'));
   assert(retained.every(p => fixture.people.some(person => person.login === p.login)));
   observed.push(...retained);
   console.log('Explicit fixture resume: retained identities and completed states will be checked, not recreated.');
  } else await writeFile(path.join(fixtureDirectory, 'observed-bindings.json'), '[]\n', { mode: 0o600, flag: 'wx' });
  stage = 'native People creation through Soda';
  await operatorPage.goto(soda + '/app/administration/people');
  await operatorPage.getByRole('heading', { name: 'Forgejo People', exact: true }).waitFor();
  for (const person of fixture.people) {
   stage = 'native People creation: ' + person.login;
   const retained = observed.find(p => p.login === person.login);
   if (retained) {
    const exists = await operatorPage.evaluate(async expected => {
     const result = await (await fetch('/api/forgejo/admin/users')).json();
     return result.items.some(user => user.id === expected.providerID && user.login === expected.login);
    }, retained);
    assert(exists, 'Retained fixture identity changed'); continue;
   }
   await operatorPage.locator('#person-login').fill(person.login);
   await operatorPage.locator('#person-email').fill(person.login + '@example.test');
   await operatorPage.locator('#person-password').fill(person['initial-password']);
   const created = operatorPage.waitForResponse(r => new URL(r.url()).pathname === '/api/forgejo/admin/users' && r.request().method() === 'POST');
   await operatorPage.getByRole('button', { name: 'Create person in Forgejo', exact: true }).click();
   const response = await created; assert.equal(response.status(), 201, 'Native person creation not confirmed; never retry automatically');
   const native = await response.json(); assert.equal(native.login, person.login);
   observed.push({ login: person.login, providerID: native.id, repository: person.repository }); await saveBindings();
   await operatorPage.getByText(`Forgejo created ${person.login}. They must change the initial password in native Forgejo and sign into Soda themselves.`, { exact: false }).waitFor();
  }
  for (const person of fixture.people) {
   stage = 'native first-login password change: ' + person.login;
   const context = await browser.newContext({ locale: 'en-US' }); contexts.push(context);
   const page = await context.newPage(); person.page = page;
   await page.goto(forgejo + '/user/login');
   const binding = observed.find(p => p.login === person.login);
   await page.locator('#user_name').fill(person.login); await page.locator('#password').fill(binding.passwordChanged ? person.password : person['initial-password']);
   await page.getByRole('button', { name: 'Sign in', exact: true }).click();
   if (!binding.passwordChanged) {
    await page.locator('#retype').waitFor();
    await page.locator('#password').fill(person.password); await page.locator('#retype').fill(person.password);
    await page.locator('form').filter({ has: page.locator('#retype') }).getByRole('button').click();
    await page.locator('#retype').waitFor({ state: 'detached' });
    binding.passwordChanged = true; await saveBindings();
   } else await page.locator('#user_name').waitFor({ state: 'detached' });
   stage = 'developer OAuth: ' + person.login;
   // Bob explicitly consents to admin scope so the negative case exercises
   // native non-admin authority, not merely an insufficient-scope guard.
   await page.goto(soda + '/login?return_to=%2Fapp%2F' + (person.login.includes('-bob-') ? '&administration=1' : ''));
   await page.waitForFunction(origin => (location.origin === origin && location.pathname === '/app/') || [...document.querySelectorAll('button')].some(b => b.textContent.trim() === 'Authorize Application'), soda);
   if (new URL(page.url()).origin === forgejo) await page.getByRole('button', { name: 'Authorize Application', exact: true }).click();
   await page.waitForURL(soda + '/app/');
   await page.getByRole('heading', { name: 'My work', exact: true }).waitFor();
   const identity = await page.evaluate(async () => { const r = await fetch('/api/session'); const v = await r.json(); return { login: v.user.login, id: v.user.id, operator: v.soda_operator }; });
   assert.equal(identity.login, person.login); assert.equal(identity.id, observed.find(p => p.login === person.login).providerID); assert.equal(identity.operator, false);
   stage = 'development public key registration: ' + person.login;
   await page.getByRole('link', { name: 'Profile and development keys', exact: true }).click();
   const fingerprint = 'SHA256:' + createHash('sha256').update(Buffer.from(person.publicKey.split(/\s+/)[1], 'base64')).digest('base64').replace(/=+$/, '');
   const registered = await page.evaluate(async fingerprint => (await (await fetch('/api/me/development-keys')).json()).items.some(key => key.fingerprint === fingerprint), fingerprint);
   if (!registered) {
    await page.locator('#public-key').fill(person.publicKey);
    await page.getByRole('button', { name: 'Register public key', exact: true }).click();
    await page.getByText('Public key registered for future joins. Existing environment accounts were not changed.', { exact: false }).waitFor();
   } else assert.equal(process.env.SODA_U08_RESUME_FIXTURES, '1', 'Refuse to adopt a preexisting key without explicit fixture resume');
   await page.getByText(fingerprint, { exact: true }).waitFor();
   stage = 'native repository creation: ' + person.login;
   if (!binding.repositoryID) {
    await page.goto(soda + '/app/repositories/new');
    await page.locator('#repository-name').fill(person.repository);
    const created = page.waitForResponse(r => new URL(r.url()).pathname === '/api/forgejo/repositories' && r.request().method() === 'POST');
    await page.getByRole('button', { name: 'Create repository in Forgejo', exact: true }).click();
    const response = await created; assert.equal(response.status(), 201, 'Repository creation not confirmed');
    binding.repositoryID = (await response.json()).id; await saveBindings();
   } else {
    await page.goto(soda + '/app/repositories/' + person.login + '/' + person.repository);
    const current = await page.evaluate(async p => (await (await fetch('/api/forgejo/repos/' + p.login + '/' + p.repository)).json()).id, { login: person.login, repository: person.repository });
    assert.equal(current, binding.repositoryID);
   }
   await page.getByRole('heading', { name: person.login + '/' + person.repository, exact: true }).waitFor();
   stage = 'persistent environment provisioning: ' + person.login;
   if (!binding.environmentID) {
    const provisioned = page.waitForResponse(r => new URL(r.url()).pathname === '/api/environments' && r.request().method() === 'POST', { timeout: 120000 });
    await page.getByRole('button', { name: 'Create persistent Rocky environment', exact: true }).click();
    const response = await provisioned; const result = await response.json();
    binding.environmentID = result.id ?? result.environment?.id; binding.provisioningStatus = response.status(); await saveBindings();
    assert.equal(response.status(), 201, 'Provisioning not confirmed; reservation retained and no retry performed');
   } else {
    assert.equal(binding.provisioningStatus, 201, 'Incomplete reservations require explicit diagnosis, not a provisioning retry');
    await page.goto(soda + '/app/environments/' + binding.environmentID);
   }
   await page.waitForURL(soda + '/app/environments/' + binding.environmentID);
   const before = await page.evaluate(async id => (await (await fetch('/api/environments/' + id)).json()).login, binding.environmentID);
   if (!binding.joined) {
    assert(!before, 'Creator was implicitly joined');
    stage = 'explicit owner join: ' + person.login;
    await page.getByRole('button', { name: 'Add me to this project', exact: true }).click();
    await page.getByText('Joined as', { exact: false }).waitFor();
    binding.joined = true; await saveBindings();
   } else assert.equal(before, person.login);
   console.log('Native person/password change/OAuth/key/repository/environment/explicit owner join verified for ' + person.login);
  }
  const alice = fixture.people.find(p => p.login.includes('-alice-')), bob = fixture.people.find(p => p.login.includes('-bob-'));
  const primary = observed.find(p => p.login === alice.login);
  stage = 'native non-admin denial';
  const denied = await bob.page.evaluate(async () => { const r = await fetch('/api/forgejo/admin/users'); const v = await r.json(); return { status: r.status, code: v.error?.code }; });
  assert.equal(denied.status, 403); assert.equal(denied.code, 'provider_forbidden');
  stage = 'private repository visibility before native collaboration';
  const privateStatus = await bob.page.evaluate(async owner => (await fetch('/api/forgejo/repos/' + owner.login + '/' + owner.repository)).status, { login: alice.login, repository: alice.repository });
  assert.equal(privateStatus, 404);
  stage = 'native collaborator provisioning, separate from Linux membership';
  await alice.page.goto(soda + '/app/repositories/' + alice.login + '/' + alice.repository + '/collaborators');
  await alice.page.locator('#collaborator-login').fill(bob.login);
  await alice.page.locator('#collaborator-permission').selectOption('write');
  const collaboration = alice.page.waitForResponse(r => new URL(r.url()).pathname.endsWith('/collaborators/' + bob.login) && r.request().method() === 'PUT');
  await alice.page.getByRole('button', { name: 'Set native collaborator permission', exact: true }).click();
  assert.equal((await collaboration).status(), 204);
  assert(!(await bob.page.evaluate(async id => (await (await fetch('/api/environments/' + id)).json()).login, primary.environmentID)), 'Git permission implicitly joined the environment');
  stage = 'non-owner environment creation denial';
  const ownerDenial = await bob.page.evaluate(async person => {
   const session = await (await fetch('/api/session')).json();
   const r = await fetch('/api/environments', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token }, body: JSON.stringify({ owner: person.login, repository: person.repository }) });
   return { status: r.status, code: (await r.json()).error?.code };
  }, { login: alice.login, repository: alice.repository });
  assert.equal(ownerDenial.status, 403); assert.equal(ownerDenial.code, 'owner_required');
  stage = 'second developer explicit join';
  await bob.page.goto(soda + '/app/environments/' + primary.environmentID);
  await bob.page.getByRole('button', { name: 'Add me to this project', exact: true }).click();
  await bob.page.getByText('Joined as', { exact: false }).waitFor();
  const own = await bob.page.evaluate(async id => { const r = await fetch('/api/environments/' + id); const v = await r.json(); return { login: v.login, admin: v.environment_administrator }; }, primary.environmentID);
  assert.equal(own.login, bob.login); assert.equal(own.admin, false);
  console.log('Two project reservations, explicit memberships, native non-admin and non-owner denials verified. Direct SSH/workloads/persistence remain separate checks.');
 } catch (failure) {
  // Do not print assertion values, native bodies or Playwright call logs.
  let summary = String(failure.message).split('\n')[0].replace(/(https?:\/\/[^\s?]+)\?[^\s]*/g, '$1?[redacted]');
  for (const value of secrets) summary = summary.replaceAll(value, '[redacted]');
  console.error('Developer fixture journey stopped during ' + stage + ': ' + summary + '; run-owned state retained in private observed-bindings.json.');
  throw new Error('Developer fixture journey incomplete; no automatic retry or cleanup');
 } finally { for (const context of contexts) await context.close(); for (const person of fixture.people) { person.password = ''; person['initial-password'] = ''; } secrets.fill(''); }
}
