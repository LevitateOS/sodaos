// Opt-in personal native Git-key registration; never receives a private key.
import assert from 'node:assert/strict';
import { readFile, writeFile, lstat } from 'node:fs/promises';
import path from 'node:path';
export async function registerPersonalGit({ page, soda, username, directory }) {
 assert.equal(process.env.SODA_NATIVE_VALIDATE, 'soda-test');
 assert(path.isAbsolute(directory)); const stat = await lstat(directory);
 assert(stat.isDirectory() && stat.uid === process.getuid() && !(stat.mode & 0o077));
 assert(/^u08-(alice|bob)-8417$/.test(username));
 const key = (await readFile(path.join(directory, username + '.pub'), 'utf8')).trim();
 assert(/^ssh-ed25519 [A-Za-z0-9+/]+={0,2}( .*)?$/.test(key));
 await page.goto(soda + '/app/account');
 await page.locator('#git-key-title').fill('U08 personal project Git ' + username);
 await page.locator('#git-key').fill(key);
 const pending = page.waitForResponse(r => new URL(r.url()).pathname === '/api/forgejo/me/git-keys' && r.request().method() === 'POST');
 await page.getByRole('button', { name: 'Register Forgejo Git key', exact: true }).click();
 const response = await pending; assert.equal(response.status(), 201, 'Git key registration not confirmed; do not retry blindly');
 await page.getByText('Forgejo Git key registered. No project account was changed.', { exact: false }).waitFor();
 const repo = await page.evaluate(async () => {
  const response = await fetch('/api/forgejo/repos/u08-alice-8417/shared-alice');
  if (response.status !== 200) return null;
  const body = await response.json(); return { id: body.id, ssh_url: body.ssh_url, owner: body.owner.login, name: body.name };
 });
 assert(repo && repo.owner === 'u08-alice-8417' && repo.name === 'shared-alice');
 await writeFile(path.join(directory, username + '-repository.json'), JSON.stringify(repo, null, 2), { mode: 0o600, flag: 'wx' });
 console.log('Acting-user personal Forgejo Git public key registered; native repository clone URL retained.');
}
