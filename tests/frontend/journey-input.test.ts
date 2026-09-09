import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {journeyInput, managementInput} from '../installed/sodaspaces-input.ts';
const user = (id: string, login: string) => ({id, login, password_file: '/synthetic/private-password'});
const input = () => ({ca_file: '/synthetic/ca', origin: 'https://fixture.invalid', target: 'fixture',
  oauth_client_id: 'synthetic-client', repository_id: '42', repository_path: '/alice/repo', revision: '1'.repeat(40),
  users: [user('1', 'alice'), user('2', 'bob')]});

test('private journey input preserves two distinct actors and rejects ambiguous identities', () => {
  assert.deepEqual(journeyInput(input(), false), input());
  for (const users of [[user('1', 'alice'), user('1', 'bob')], [user('1', 'alice'), user('2', 'alice')],
    [user('9223372036854775808', 'alice'), user('2', 'bob')], [user('1', 'alice')]]) {
    assert.throws(() => journeyInput({...input(), users}, false));
  }
  assert.throws(() => journeyInput({...input(), repository_id: 42}, false));
  assert.throws(() => journeyInput({...input(), target: '../fixture'}, false));
});

test('public-key inputs require explicit access mode and valid project logins', () => {
  const withKeys = {...input(), users: input().users.map(user => ({...user, public_key_file: '/synthetic/public-key'}))};
  assert.deepEqual(journeyInput(withKeys, true), withKeys);
  assert.throws(() => journeyInput(withKeys, false));
  assert.throws(() => journeyInput(input(), true));
  assert.throws(() => journeyInput({...withKeys, users: withKeys.users.map(user => ({...user, login: 'root'}))}, true));
});

test('management input rejects extra authority fields and malformed container targets', () => {
  const request = {cid: 'a'.repeat(64), project: 'p' + 'b'.repeat(24), target: 'fixture', ssh_config: '/synthetic/ssh',
    key_a: '/synthetic/a', key_a_public: '/synthetic/a.pub', key_b: '/synthetic/b', key_b_public: '/synthetic/b.pub',
    original_alice: '/synthetic/alice', original_bob: '/synthetic/bob'};
  assert.deepEqual(managementInput(request), request);
  assert.throws(() => managementInput({...request, extra_host: 'another-fixture'}));
  assert.throws(() => managementInput({...request, cid: 'container-name'}));
  assert.throws(() => managementInput({...request, project: '../other'}));
});
