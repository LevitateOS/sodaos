// Two projections of the canonical payload: the retained branding-only mount,
// and the complete public tree needed by Spaces. Neither changes a live mount.
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {dirname, resolve} from 'node:path';
import {parseArgs} from 'node:util';
import payload from '../internal/nativebuild/forgejo-payload.json';
import {buildForgejoAssets} from './build-forgejo.ts';

const root = resolve(import.meta.dir, '..');
const {values} = parseArgs({args: Bun.argv.slice(2), options: {
  out: {type: 'string', default: resolve(root, '.artifacts/forgejo-preview/branding')},
  'public-out': {type: 'string'},
}});
assert(values.out);
const out = resolve(values.out), publicOut = resolve(values['public-out'] || resolve(dirname(out), 'public'));
const compiled = resolve(root, '.artifacts/forgejo-js');
await buildForgejoAssets(compiled);
const terminalOut = resolve(publicOut, 'assets/soda-terminal');
const terminal = Bun.spawn(['python3', resolve(root, 'scripts/fetch-terminal.py'), '--out', terminalOut], {stdout: 'inherit', stderr: 'inherit'});
assert.equal(await terminal.exited, 0, 'Canonical terminal asset preparation failed');
for (const [target, source] of Object.entries(payload)) {
  if (!target.startsWith('public/')) continue;
  const input = source.startsWith('@build/forgejo-js/')
    ? resolve(compiled, source.slice('@build/forgejo-js/'.length)) : source.startsWith('@build/terminal-assets/') ? resolve(terminalOut, source.slice('@build/terminal-assets/'.length)) : resolve(root, source);
  assert(!source.startsWith('@build/') || source.startsWith('@build/forgejo-js/') || source.startsWith('@build/terminal-assets/'));
  const destinations = [resolve(publicOut, target.slice('public/'.length))];
  const prefix = 'public/assets/soda/forgejo/';
  if (target.startsWith(prefix)) destinations.push(resolve(out, target.slice(prefix.length)));
  for (const output of destinations) {
    if (output === input) continue;
    await mkdir(dirname(output), {recursive: true});
    await Bun.write(output, Bun.file(input));
  }
}
console.log(`Generated preview branding: ${out}\nGenerated complete preview public tree: ${publicOut}\nNo live mount or service was changed.`);
