// Project the same production payload into the local preview's branding mount.
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {dirname, resolve} from 'node:path';
import {parseArgs} from 'node:util';
import payload from '../internal/nativebuild/forgejo-payload.json';
import {buildForgejoAssets} from './build-forgejo.ts';

const root = resolve(import.meta.dir, '..');
const {values} = parseArgs({args: Bun.argv.slice(2), options: {
  out: {type: 'string', default: resolve(root, '.artifacts/forgejo-preview/branding')},
}});
assert(values.out);
const out = resolve(values.out), compiled = resolve(root, '.artifacts/forgejo-js');
await buildForgejoAssets(compiled);
for (const [target, source] of Object.entries(payload)) {
  const prefix = 'public/assets/soda/forgejo/';
  if (!target.startsWith(prefix)) continue;
  const input = source.startsWith('@build/forgejo-js/')
    ? resolve(compiled, source.slice('@build/forgejo-js/'.length)) : resolve(root, source);
  assert(!source.startsWith('@build/') || source.startsWith('@build/forgejo-js/'));
  const output = resolve(out, target.slice(prefix.length));
  await mkdir(dirname(output), {recursive: true});
  await Bun.write(output, Bun.file(input));
}
console.log(`Generated preview branding: ${out}`);
