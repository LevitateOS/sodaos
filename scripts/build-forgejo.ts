import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { basename, posix, resolve } from 'node:path';
import { parseArgs } from 'node:util';
import payload from '../internal/nativebuild/forgejo-payload.json';

const root = resolve(import.meta.dir, '..');
const modules = Object.entries(payload).filter(([, source]) => source.startsWith('@build/forgejo-js/'));
const destinations = new Map(modules.map(([target, source]) => [basename(source), target]));
assert.equal(destinations.size, modules.length, 'Each browser module must have exactly one public destination');
const litSource = resolve(root, 'assets/branding/forgejo/lit.ts');
const litDestination = destinations.get('lit.js');

// Also used by the browser smoke fixture: imports must work from either public
// asset directory, including when Forgejo has an AppSubUrl prefix.
export async function buildForgejoModule(source: string, destination: string) {
  assert(litDestination, 'The shared Lit runtime must be in the production payload');
  const runtime = resolve(source) === litSource;
  const relative = posix.relative(posix.dirname(destination), litDestination);
  const runtimeURL = relative.startsWith('.') ? relative : `./${relative}`;
  const result = await Bun.build({
    entrypoints: [source],
    target: 'browser',
    format: 'esm',
    minify: true,
    plugins: runtime ? [] : [{
      name: 'forgejo-public-imports',
      setup(build) {
        build.onResolve({filter: /.*/}, args => {
          if (args.kind === 'entry-point-build') return;
          if (args.path === 'lit') return {path: runtimeURL, external: true};
          if (/^(?:lit\/|lit-element(?:\/|$)|lit-html(?:\/|$)|@lit(?:-labs)?\/)/.test(args.path)) {
            throw Error(`Unsupported Lit submodule ${args.path}: add an explicit shared-runtime export before using it`);
          }
          // Preserve all existing relative native/xterm/Sodaspaces boundaries.
          return {path: args.path, external: true};
        });
      },
    }],
  });
  assert(result.success, `Browser build failed for ${source}: ${result.logs.join('\n')}`);
  const [output] = result.outputs;
  assert(output && result.outputs.length === 1, `Expected one browser asset for ${source}`);
  return output;
}

export async function buildForgejoAssets(out: string) {
  const sources = new Map<string, string>();
  for (const directory of ['assets/branding/forgejo', 'appliance/forgejo/public/assets']) {
    for (const file of new Bun.Glob('*.ts').scanSync(resolve(root, directory))) {
      if (file.endsWith('.d.ts')) continue;
      const output = file.replace(/\.ts$/, '.js');
      assert(!sources.has(output), `Duplicate browser module: ${output}`);
      sources.set(output, resolve(root, directory, file));
    }
  }
  assert.deepEqual(new Set(sources.keys()), new Set(destinations.keys()), 'Browser source and staged module inventory must match');
  await mkdir(out, { recursive: true });
  for (const [file, source] of sources) {
    const destination = destinations.get(file);
    assert(destination);
    const output = await buildForgejoModule(source, destination);
    await Bun.write(resolve(out, file), output);
  }
}
if (import.meta.main) {
  const {values} = parseArgs({args: Bun.argv.slice(2), options: {out: {type: 'string', default: resolve(root, '.artifacts/forgejo-js')}}});
  assert(values.out);
  await buildForgejoAssets(resolve(values.out));
}
