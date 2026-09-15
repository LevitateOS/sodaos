import test from 'node:test';
import assert from 'node:assert/strict';
import {workspaceEntryLocation, workspaceFrameLocator} from '../../frontend/spaces/sodaspaces-frame.ts';

test('workspace frame locator admits same-origin Forgejo paths only', () => {
  assert.equal(workspaceFrameLocator('/'), '/');
  assert.equal(workspaceFrameLocator('/alice/repo'), '/alice/repo');
  assert.equal(workspaceFrameLocator('/alice/repo?tab=issues'), '/alice/repo?tab=issues');
  assert.equal(workspaceFrameLocator(''), undefined);
  assert.equal(workspaceFrameLocator('https://evil.test/'), undefined);
  assert.equal(workspaceFrameLocator('//evil.test/x'), undefined);
  assert.equal(workspaceFrameLocator('/foo/../bar'), undefined);
  assert.equal(workspaceFrameLocator('/-/soda/workspace'), undefined);
  assert.equal(workspaceFrameLocator('/-/soda/api/session'), undefined);
  assert.equal(workspaceFrameLocator('/login?code=secret'), undefined);
  assert.equal(workspaceFrameLocator('/callback?state=x'), undefined);
});

test('workspace entry wraps ordinary Forgejo paths and keeps native Soda hosts', () => {
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/', ''), '/-/soda/workspace');
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/alice/repo', ''),
    '/-/soda/workspace?to=%2Falice%2Frepo'
  );
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/native/alice/repo', '/native'),
    '/native/-/soda/workspace?to=%2Falice%2Frepo'
  );
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/?soda-view=spaces', ''), undefined);
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/admin?soda-view=runners', ''), undefined);
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/user/login', ''), undefined);
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/?soda-view=spaces&soda-connect=failed', ''),
    undefined
  );
});
