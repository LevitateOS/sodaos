import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

test('Workspace declares one shared owner with one entry and declared subroutes', async () => {
  const manifest = JSON.parse(
    await readFile(new URL('../../system/containers/extension/extension.json', import.meta.url), 'utf8')
  ) as {
    pages: Array<{id: string; entry: string; owner?: string; subroutes?: Array<{pattern: string; title?: string}>}>;
    panels: Array<{id: string; entry: string; owner?: string}>;
  };
  const page = manifest.pages.find((item) => item.id === 'spaces');
  const panel = manifest.panels.find((item) => item.id === 'workspace');
  assert(page && panel, 'Workspace page and panel are declared');
  assert.equal(page.owner, 'workspace');
  assert.equal(panel.owner, 'workspace');
  assert.equal(page.entry, panel.entry, 'shared owner uses one entry');
  assert.deepEqual(
    page.subroutes?.map((route) => route.pattern),
    ['new', 'requests/{request}']
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
