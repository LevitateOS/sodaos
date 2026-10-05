import assert from 'node:assert/strict';
import {test} from 'node:test';
import payload from '../assets/branding/forgejo/forgejo-payload.json';
import {assertForgejoSourceClosure, collectForgejoSources} from './build-forgejo';

const staged = new Map(
  Object.entries(payload)
    .filter(([, source]) => source.startsWith('@build/forgejo-js/'))
    .map(([target, source]) => [source.split('/').pop()!, target])
);

test('Forgejo source inventory contains only branding modules and closes against staged modules', () => {
  const sources = collectForgejoSources();
  assert.deepEqual(new Set(sources.keys()), new Set(staged.keys()));
  for (const source of sources.values()) assert.match(source, /assets\/branding\/forgejo\/[^/]+\.ts$/);

  const unclassified = new Map(sources);
  unclassified.set('new-browser-module.js', 'frontend/spaces/new-browser-module.ts');
  assert.throws(
    () => assertForgejoSourceClosure(unclassified, staged),
    /source and staged module inventory must match/
  );
});
