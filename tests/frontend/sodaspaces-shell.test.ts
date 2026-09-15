import test from 'node:test';
import assert from 'node:assert/strict';
import {workspaceFrameLocator} from '../../frontend/spaces/sodaspaces-frame.ts';

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
