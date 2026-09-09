// Read-only connection/host-key observations for the existing U08 fixtures.
// Operator SSH independently verifies public keys; no developer private key is
// read, transferred or used, and client reachability is not inferred from this.
import assert from 'node:assert/strict';
import { writeFile, lstat } from 'node:fs/promises';
import type {Page} from 'playwright';
import { isIP } from 'node:net';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
export async function observeProjectConnections({ page, username, fixtureDirectory }: {page: Page; username: string; fixtureDirectory: string}) {
 assert.equal(process.env.SODA_NATIVE_VALIDATE, 'soda-test');
 assert(path.isAbsolute(fixtureDirectory));
 assert(/^u08-(alice|bob)-[0-9a-f]{4}$/.test(username));
 const st = await lstat(fixtureDirectory); assert(st.isDirectory() && st.uid === process.getuid?.() && !(st.mode & 0o077));
 const decoded: unknown = await Bun.file(path.join(fixtureDirectory, 'observed-bindings.json')).json();
 assert(Array.isArray(decoded));
 const bindings: unknown[] = decoded;
 const vm = fileURLToPath(new URL('../../scripts/test-vm.sh', import.meta.url));
 const actor = await page.evaluate(async () => {
  const response = await fetch('/-/soda/api/session');
  const body: unknown = await response.json();
  if (!body || typeof body !== 'object' || !('user' in body) || !body.user || typeof body.user !== 'object' || !('id' in body.user) || !('login' in body.user)) throw Error('Invalid session response');
  return { status: response.status, id: body.user?.id, login: body.user?.login };
 });
 assert.equal(actor.status, 200); assert.equal(actor.login, username);
 assert(typeof actor.id === 'string'); assert(/^[1-9][0-9]{0,18}$/.test(actor.id));
 const keys = [], connections = [];
 for (const binding of bindings) {
  assert(binding && typeof binding === 'object' && 'environmentID' in binding && typeof binding.environmentID === 'string' && 'login' in binding && typeof binding.login === 'string');
  assert(/^p[0-9a-f]{24}$/.test(binding.environmentID));
  const result: {status: number; body: unknown} = await page.evaluate(async ({ id, actorID }) => {
  const response = await fetch('/-/soda/api/environments/' + id + '/connection', { headers: { 'X-Soda-Expected-User-ID': actorID } });
  return { status: response.status, body: await response.json() as unknown };
 }, { id: binding.environmentID, actorID: actor.id });
  if (binding.login !== username && !username.includes('-bob-')) { assert.equal(result.status, 403, 'Connection disclosed before project join'); continue; }
  assert.equal(result.status, 200);
  const body = result.body;
  assert(body && typeof body === 'object' && 'login' in body && 'routing_verified' in body && 'connection' in body);
  assert.equal(body.login, username); assert.equal(body.routing_verified, false);
  const connection = body.connection;
  assert(connection && typeof connection === 'object' && 'environment' in connection && 'host_key' in connection && typeof connection.host_key === 'string' && 'fingerprint' in connection && typeof connection.fingerprint === 'string');
  assert(connection.environment && typeof connection.environment === 'object' && 'id' in connection.environment && 'running' in connection.environment && 'ip' in connection.environment && typeof connection.environment.ip === 'string'); assert.equal(connection.environment.id, binding.environmentID); assert.equal(connection.environment.running, true);
  const ip = connection.environment.ip; assert.equal(isIP(ip), 4); assert(ip.startsWith('10.89.0.'));
  // Fixed public-key operation, with a strict ID from the authenticated core API.
  const observed = Bun.spawnSync([vm, 'ssh', `podman exec soda-${binding.environmentID} /usr/bin/head -n 1 /etc/ssh/ssh_host_ed25519_key.pub`], {stdin: 'ignore', stdout: 'pipe', stderr: 'pipe'});
  assert(observed.exitCode === 0, 'Public host-key observation failed');
  const actual = observed.stdout.toString().trim().split(/\s+/).slice(0, 2).join(' ');
  const advertised = connection.host_key.trim().split(/\s+/).slice(0, 2).join(' ');
  assert(actual.startsWith('ssh-ed25519 ')); assert.equal(advertised, actual);
  keys.push(ip + ' ' + actual);
  connections.push({ id: binding.environmentID, ip, login: username, fingerprint: connection.fingerprint, clientRoutingVerified: false });
 }
 assert(keys.length > 0);
 await writeFile(path.join(fixtureDirectory, username + '-known-hosts'), keys.join('\n') + '\n', { mode: 0o600, flag: 'wx' });
 await writeFile(path.join(fixtureDirectory, username + '-connections.json'), JSON.stringify(connections, null, 2), { mode: 0o600, flag: 'wx' });
 console.log('Authenticated connection authorization and independent public host-key verification completed; direct client routing remains unverified.');
}
