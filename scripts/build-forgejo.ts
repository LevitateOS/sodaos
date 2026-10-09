import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {readFileSync} from 'node:fs';
import {basename, posix, resolve} from 'node:path';
import {parseArgs} from 'node:util';
import payload from '../frontend/forgejo/payload.json';

const root = resolve(import.meta.dir, '..');
// One reviewed presentation epoch covers the whole module graph, not just the
// HTML entry. Relative imports otherwise keep their old six-hour cache identity.
export const presentationVersion = readFileSync(
  resolve(root, 'frontend/forgejo/templates/custom/header.tmpl'),
  'utf8'
).match(/name="soda-presentation-revision" content="([a-zA-Z0-9.-]+)"/)?.[1];
assert(presentationVersion, 'Missing presentation cache epoch');
const versioned = (url: string) => `${url}?v=${presentationVersion}`;
const modules = Object.entries(payload).filter(([, source]) => source.startsWith('@build/forgejo-js/'));
const destinations = new Map(modules.map(([target, source]) => [basename(source), target]));
assert.equal(destinations.size, modules.length, 'Each browser module must have exactly one public destination');
const litSource = resolve(root, 'assets/branding/forgejo/lit.ts');
const litDestination = destinations.get('lit.js');
// Authored shared owner name, retained public URL. Not a second runtime entry.
const outputName = (source: string) => basename(source).replace(/\.ts$/, '.js');

export function assertForgejoSourceClosure(sources: Map<string, string>, staged: Map<string, string>) {
  assert.deepEqual(
    new Set(sources.keys()),
    new Set(staged.keys()),
    'Browser source and staged module inventory must match'
  );
}

export function collectForgejoSources() {
  const sources = new Map<string, string>();
  const directory = 'assets/branding/forgejo';
  for (const file of new Bun.Glob('*.ts').scanSync(resolve(root, directory))) {
    if (file.endsWith('.d.ts')) continue;
    const output = outputName(file);
    assert(!sources.has(output), `Duplicate browser module: ${output}`);
    sources.set(output, resolve(root, directory, file));
  }
  return sources;
}

// Also used by the browser smoke fixture: imports must work from either public
// asset directory, including when Forgejo has an AppSubUrl prefix.
export async function buildForgejoModule(source: string, destination: string, format: 'esm' | 'iife' = 'esm') {
  assert(litDestination, 'The shared Lit runtime must be in the production payload');
  const runtime = resolve(source) === litSource;
  const relative = posix.relative(posix.dirname(destination), litDestination);
  const runtimeURL = relative.startsWith('.') ? relative : `./${relative}`;
  const result = await Bun.build({
    entrypoints: [source],
    target: 'browser',
    format,
    minify: true,
    // Lit ships development assertions unless the production condition is set.
    define: {'process.env.NODE_ENV': '"production"'},
    plugins: [
      {
        name: 'forgejo-public-imports',
        setup(build) {
          build.onResolve({filter: /.*/}, (args) => {
            if (args.kind === 'entry-point-build') return;
            if (
              /^(?:lit-analyzer|typescript|web-component-analyzer|ts-simple-type)(?:\/|$)/.test(args.path) ||
              args.path.includes('tools/lit-check')
            ) {
              throw Error(`Development-only analysis tool cannot enter browser payload: ${args.path}`);
            }
            if (runtime) return;
            if (args.path === 'lit' || args.path === 'lit/directives/repeat.js')
              return {path: versioned(runtimeURL), external: true};
            if (/^(?:lit\/|lit-element(?:\/|$)|lit-html(?:\/|$)|@lit(?:-labs)?\/)/.test(args.path)) {
              throw Error(
                `Unsupported Lit submodule ${args.path}: add an explicit shared-runtime export before using it`
              );
            }
            // Native branding imports remain public-relative.
            return {path: args.path.startsWith('.') ? versioned(args.path) : args.path, external: true};
          });
        },
      },
    ],
  });
  assert(result.success, `Browser build failed for ${source}: ${result.logs.join('\n')}`);
  const [output] = result.outputs;
  assert(output && result.outputs.length === 1, `Expected one browser asset for ${source}`);
  return output;
}

export async function buildForgejoAssets(out: string) {
  const sources = collectForgejoSources();
  assertForgejoSourceClosure(sources, destinations);
  await mkdir(out, {recursive: true});
  for (const [file, source] of sources) {
    const destination = destinations.get(file);
    assert(destination);
    // The five behavior entrypoints have classic template callers; Lit and
    // the importing browser fixtures retain their module graph.
    const output = await buildForgejoModule(source, destination, file === 'lit.js' ? 'esm' : 'iife');
    await Bun.write(resolve(out, file), output);
  }
}
if (import.meta.main) {
  const {values} = parseArgs({
    args: Bun.argv.slice(2),
    options: {out: {type: 'string', default: resolve(root, '.artifacts/forgejo-js')}},
  });
  assert(values.out);
  await buildForgejoAssets(resolve(values.out));
}
