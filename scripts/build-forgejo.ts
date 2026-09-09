import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { basename, resolve } from 'node:path';
import { parseArgs } from 'node:util';

const root = resolve(import.meta.dir, '..');
export async function buildForgejoAssets(out: string) {
  const payload: unknown = await Bun.file(resolve(root, 'internal/nativebuild/forgejo-payload.json')).json();
  assert(payload && typeof payload === 'object' && !Array.isArray(payload));
  const required = new Set(Object.values(payload).filter((value): value is string => typeof value === 'string' && value.startsWith('@build/forgejo-js/')).map(value => basename(value)));
  const sources = new Map<string, string>();
  for (const directory of ['assets/branding/forgejo', 'appliance/forgejo/public/assets']) {
    for (const file of new Bun.Glob('*.ts').scanSync(resolve(root, directory))) {
      if (file.endsWith('.d.ts')) continue;
      const output = file.replace(/\.ts$/, '.js');
      assert(!sources.has(output), `Duplicate browser module: ${output}`);
      sources.set(output, resolve(root, directory, file));
    }
  }
  assert.deepEqual(new Set(sources.keys()), required, 'Browser source and staged module inventory must match');
  await mkdir(out, { recursive: true });
  for (const [file, source] of sources) {
    // Each original module/script keeps its public URL and import boundaries.
    const result = await Bun.build({
      entrypoints: [source],
      target: 'browser',
      format: 'esm',
      external: ['*'],
      minify: true,
    });
    assert(result.success, `Browser build failed for ${file}: ${result.logs.join('\n')}`);
    const [output] = result.outputs;
    assert(output && result.outputs.length === 1, `Expected one browser asset for ${file}`);
    await Bun.write(resolve(out, file), output);
  }
}
if (import.meta.main) {
  const {values} = parseArgs({args: Bun.argv.slice(2), options: {out: {type: 'string', default: resolve(root, '.artifacts/forgejo-js')}}});
  assert(values.out);
  await buildForgejoAssets(resolve(values.out));
}
