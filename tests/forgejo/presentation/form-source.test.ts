import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFile} from 'node:fs/promises';
import {test} from 'node:test';
import {contracts} from './settings-contracts.ts';
import snapshot from './form-native-contracts.json';

test('redesigned forms preserve native submission controls and capability gates', async () => {
  for (const [path, before] of Object.entries(snapshot.entries)) {
    const source = await readFile(new URL(`../../../appliance/forgejo/templates/${path}`, import.meta.url), 'utf8');
    const actual = contracts(source);
    assert.equal(createHash('sha256').update(JSON.stringify(actual.controls)).digest('hex'), before.controls, `${path}: changed native submission controls`);
    for (const gate of before.gates) assert(actual.gates.includes(gate), `${path}: lost native gate ${gate}`);
  }
});
