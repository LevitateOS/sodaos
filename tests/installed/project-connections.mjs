// Read-only connection/host-key observations for the existing U08 fixtures.
// Operator SSH independently verifies public keys; no developer private key is
// read, transferred or used, and client reachability is not inferred from this.
import assert from 'node:assert/strict';
import { readFile, writeFile, lstat } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { isIP } from 'node:net';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
export async function observeProjectConnections({ page, username, fixtureDirectory }) {
 assert.equal(process.env.SODA_NATIVE_VALIDATE, 'soda-test');
 assert(path.isAbsolute(fixtureDirectory));
 assert(/^u08-(alice|bob)-[0-9a-f]{4}$/.test(username));
 const st = await lstat(fixtureDirectory); assert(st.isDirectory() && st.uid === process.getuid() && !(st.mode & 0o077));
 const bindings = JSON.parse(await readFile(path.join(fixtureDirectory, 'observed-bindings.json'), 'utf8'));
 const vm = fileURLToPath(new URL('../../scripts/test-vm.sh', import.meta.url));
 const keys = [], connections = [];
 for (const binding of bindings) {
  assert(/^p[0-9a-f]{24}$/.test(binding.environmentID));
  const result = await page.evaluate(async id => { const response = await fetch('/-/soda/api/environments/' + id + '/connection'); return { status: response.status, body: await response.json() }; }, binding.environmentID);
  if (binding.login !== username && !username.includes('-bob-')) { assert.equal(result.status, 403, 'Connection disclosed before project join'); continue; }
  assert.equal(result.status, 200); assert.equal(result.body.login, username); assert.equal(result.body.routing_verified, false);
  const connection = result.body.connection; assert.equal(connection.environment.id, binding.environmentID); assert.equal(connection.environment.running, true);
  const ip = connection.environment.ip; assert.equal(isIP(ip), 4); assert(ip.startsWith('10.89.0.'));
  // Fixed public-key operation, with a strict ID from the authenticated core API.
  const actual = execFileSync(vm, ['ssh', `podman exec soda-${binding.environmentID} /usr/bin/head -n 1 /etc/ssh/ssh_host_ed25519_key.pub`], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim().split(/\s+/).slice(0, 2).join(' ');
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
