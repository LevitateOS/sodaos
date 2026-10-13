import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import type {MountEntry} from '../../frontend/spaces/fountain-browser-contract.js';

// Lane C re-pins source + hash on every canonical contract change (H-N + H-OWNER).
const source = 'sdk/browser/extension-contract.d.ts at a9ba2ce0ba';
const pin = 'b1269647e41dae92ee625aecbe51ae57a6deeb0321ca0c66f5fcf31152951788';
const marker = '// --- contract body begins ---';

test('host contract mirror matches the pinned Fountain bytes', async () => {
  const mirror = await readFile(
    new URL('../../frontend/spaces/fountain-browser-contract.d.ts', import.meta.url),
    'utf8'
  );
  const at = mirror.indexOf(marker);
  assert(at >= 0, 'mirror header marker missing');
  const header = mirror.slice(0, at);
  assert(header.includes(source), 'mirror does not name its pinned Fountain source');
  assert(
    header.includes('contractMajor: 1') && header.includes('manifestProtocol: 1'),
    'mirror contract versions changed'
  );
  const body = mirror.slice(at + marker.length).replace(/^\r?\n/, '');
  assert.equal(
    createHash('sha256').update(body).digest('hex'),
    pin,
    'mirror body drifted from the pinned Fountain contract'
  );
});

test('mirror mount types resolve for consumers', () => {
  const entry: MountEntry = () => ({});
  assert.equal(typeof entry, 'function');
});
