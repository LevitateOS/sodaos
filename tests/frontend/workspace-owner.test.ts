import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

test('the manifest declares no global Workspace page or panel', async () => {
  const manifest = JSON.parse(
    await readFile(new URL('../../system/containers/extension/extension.json', import.meta.url), 'utf8')
  ) as {pages: Array<{id: string; title: string}>; panels?: Array<{id: string}>};
  assert(!manifest.pages.some((page) => page.id === 'spaces' || page.title === 'Workspace'));
  assert.deepEqual(
    manifest.panels?.map((panel) => panel.id),
    ['providers']
  );
});

test('shared Workspace entry returns a real MountHandle without host-ancestor workarounds', async () => {
  const entry = await readFile(new URL('../../frontend/spaces/soda-spaces-entry.ts', import.meta.url), 'utf8');
  for (const callback of ['update', 'suspend', 'resume', 'retire', 'dispose']) {
    assert(entry.includes(callback), `entry handles ${callback}`);
  }
  assert(!entry.includes('frameElement'), 'no frame detection');
  assert(!entry.includes('data-workspace-direct'), 'no competing-panel detection');
  assert(!entry.includes('data-extension-workspace'), 'no host ancestor inspection');
  const transport = await readFile(new URL('../../frontend/spaces/soda-extension.ts', import.meta.url), 'utf8');
  assert(!transport.includes('extensionURL'), 'no duplicate generic URL construction');
  assert(!transport.includes('new WebSocket'), 'no local stream fallback');
  assert(transport.includes('hostRequest'), 'host request helper is consumed');
  assert(transport.includes('hostStream'), 'host stream helper is consumed');
  assert(transport.includes('hostNavigate'), 'host navigate helper is consumed');
});

test('canonical mirror carries the shared-owner and navigation contract', async () => {
  const mirror = await readFile(
    new URL('../../frontend/spaces/fountain-browser-contract.d.ts', import.meta.url),
    'utf8'
  );
  assert(mirror.includes('ownerId'), 'mirror declares ownerId');
  assert(mirror.includes('navigate'), 'mirror declares navigate');
});
