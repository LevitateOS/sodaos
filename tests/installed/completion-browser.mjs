// Core U08 fresh fixture using existing real users, never seeded sessions/accounts.
import assert from 'node:assert/strict';
import { readFile, writeFile, lstat } from 'node:fs/promises';
import path from 'node:path';
export async function completionFixture({ page, soda, username, directory }) {
 assert.equal(process.env.SODA_NATIVE_VALIDATE, 'soda-test');
 assert(path.isAbsolute(directory)); const st = await lstat(directory);
 assert(st.isDirectory() && st.uid === process.getuid() && !(st.mode & 0o077));
 const file = path.join(directory, 'target.json'); const target = JSON.parse(await readFile(file, 'utf8'));
 assert(/^u08-completion-[a-f0-9]{7,40}$/.test(target.repository));
 assert(['u08-alice-8417', 'u08-bob-8417'].includes(username));
 const save = () => writeFile(file, JSON.stringify(target, null, 2), { mode: 0o600 });
 if (username === 'u08-alice-8417') {
  assert(!target.environment_id && !target.repository_id, 'Fresh creation only; inspect partial state before any retry');
  await page.goto(soda + '/app/repositories/new');
  await page.locator('#repository-name').fill(target.repository);
  const pending = page.waitForResponse(r => new URL(r.url()).pathname === '/api/forgejo/repositories' && r.request().method() === 'POST');
  await page.getByRole('button', { name: 'Create repository in Forgejo', exact: true }).click();
  const reply = await pending; assert.equal(reply.status(), 201);
  const repository = await reply.json(); target.repository_id = repository.id; target.ssh_url = repository.ssh_url; await save();
  await page.getByRole('heading', { name: username + '/' + target.repository, exact: true }).waitFor();
  const creation = page.waitForResponse(r => new URL(r.url()).pathname === '/api/environments' && r.request().method() === 'POST', { timeout: 180000 });
  await page.getByRole('button', { name: 'Create persistent Rocky environment', exact: true }).click();
  const response = await creation; const body = await response.json();
  target.environment_id = body.id ?? body.environment?.id; target.provision_status = response.status(); await save();
  assert.equal(response.status(), 201, 'Incomplete reservation retained; no automatic retry');
  await page.waitForURL(soda + '/app/environments/' + target.environment_id);
 }
 assert(/^p[0-9a-f]{24}$/.test(target.environment_id));
 await page.goto(soda + '/app/environments/' + target.environment_id);
 const before = await page.evaluate(async id => {
  const detail = await (await fetch('/api/environments/' + id)).json();
  const connection = await fetch('/api/environments/' + id + '/connection');
  return { login: detail.login, connectionStatus: connection.status };
 }, target.environment_id);
 assert(!before.login, 'Explicit join required; refuse to replay membership'); assert.equal(before.connectionStatus, 403);
 await page.getByRole('button', { name: 'Add me to this project', exact: true }).click();
 await page.getByText('Joined as', { exact: false }).waitFor();
 const connection = await page.evaluate(async id => {
  const r = await fetch('/api/environments/' + id + '/connection');
  return r.status === 200 ? await r.json() : null;
 }, target.environment_id);
 assert(connection && connection.login === username && connection.routing_verified === false);
 assert.equal(connection.connection.environment.id, target.environment_id);
 target.ip = connection.connection.environment.ip;
 target.host_key = connection.connection.host_key;
 target.fingerprint = connection.connection.fingerprint;
 target.joined = [...(target.joined ?? []), username]; await save();
 if (username === 'u08-alice-8417') {
  await page.goto(soda + '/app/repositories/' + username + '/' + target.repository + '/collaborators');
  await page.locator('#collaborator-login').fill('u08-bob-8417');
  await page.locator('#collaborator-permission').selectOption('write');
  const changed = page.waitForResponse(r => new URL(r.url()).pathname.endsWith('/collaborators/u08-bob-8417') && r.request().method() === 'PUT');
  await page.getByRole('button', { name: 'Set native collaborator permission', exact: true }).click();
  assert.equal((await changed).status(), 204);
 } else {
  const denial = await page.evaluate(async repository => {
   const session = await (await fetch('/api/session')).json();
   const r = await fetch('/api/environments', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token }, body: JSON.stringify({ owner: 'u08-alice-8417', repository }) });
   return { status: r.status, code: (await r.json()).error?.code };
  }, target.repository);
  assert.equal(denial.status, 403); assert.equal(denial.code, 'owner_required');
 }
 console.log('Fresh completion fixture: native creation/join/connection and acting-user authority verified for ' + username);
}
