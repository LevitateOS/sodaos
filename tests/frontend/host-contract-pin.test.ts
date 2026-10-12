import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import type {MountEntry} from '../../frontend/spaces/fountain-browser-contract.js';

// Lane C re-pins source + hash on every canonical contract change (H-N prep).
const source = 'sdk/browser/extension-contract.d.ts at 5e812ba9';
const pin = '96bc12f2f3dc7b5234feec337b05efc76ceef18c7fce8bff7d0e469ef941dd9d';
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
