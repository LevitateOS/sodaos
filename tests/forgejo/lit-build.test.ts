import assert from 'node:assert/strict';
import {mkdtemp, readFile, rm, writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {dirname, join, resolve} from 'node:path';
import test from 'node:test';
import {buildForgejoModule, presentationVersion} from '../../scripts/build-forgejo.ts';
import payload from '../../internal/nativebuild/forgejo-payload.json';

const root = resolve(import.meta.dirname, '../..');

test('Lit submodules cannot silently become unresolved or duplicate runtimes', async t => {
  const directory = await mkdtemp(join(tmpdir(), 'soda-lit-build-'));
  t.after(() => rm(directory, {recursive: true, force: true}));
  const source = join(directory, 'unsupported.ts');
  await writeFile(source, "export {classMap} from 'lit/directives/class-map.js';\n");
  await assert.rejects(buildForgejoModule(source, 'public/assets/component.js'), (error: unknown) => {
    assert(error instanceof AggregateError);
    assert(error.errors.some((diagnostic: unknown) => diagnostic instanceof Error &&
      diagnostic.message.includes('Unsupported Lit submodule lit/directives/class-map.js')));
    return true;
  });
});

test('analysis tools cannot enter browser payloads through external imports', async t => {
  const directory = await mkdtemp(join(tmpdir(), 'soda-analysis-boundary-'));
  t.after(() => rm(directory, {recursive: true, force: true}));
  const source = join(directory, 'tool-import.ts');
  for (const name of ['lit-analyzer', 'typescript', 'web-component-analyzer', '../../tools/lit-check/check.ts']) {
    await writeFile(source, `import ${JSON.stringify(name)};\n`);
    await assert.rejects(buildForgejoModule(source, 'public/assets/component.js'), (error: unknown) => {
      assert(error instanceof AggregateError);
      assert(error.errors.some((diagnostic: unknown) => diagnostic instanceof Error && diagnostic.message.includes('Development-only analysis tool')));
      return true;
    });
  }
});

test('moved workspace source imports retain canonical public URLs from either asset root', async () => {
  for (const [destination, expected] of [
    ['public/assets/probe.js', './sodaspaces-drawer.js'],
    ['public/assets/soda/forgejo/probe.js', '../../sodaspaces-drawer.js'],
  ]) {
    assert(destination && expected);
    const built = await buildForgejoModule(join(root, 'frontend/spaces/sodaspaces-page.ts'), destination);
    assert((await built.text()).includes(expected));
    assert(!(await built.text()).includes('frontend/spaces'));
  }
});

test('entry templates and every emitted relative import share the presentation cache epoch', async () => {
  const templates = await Promise.all(['custom/header', 'custom/footer', 'user/dashboard/dashboard'].map(name =>
    readFile(join(root, `appliance/forgejo/templates/${name}.tmpl`), 'utf8')));
  for (const name of ['sodaspaces.js', 'soda-native-page.js', 'soda-settings-link.js', 'soda-settings.css', 'sodaspaces-page.css', 'sodaspaces-drawer.css', 'sodaspaces-terminal.css', 'sodaspaces.css']) {
    assert(templates.join('\n').includes(`${name}?v=${presentationVersion}`), name);
  }
  for (const [destination, source] of Object.entries(payload)) {
    if (!source.startsWith('@build/forgejo-js/')) continue;
    const filename = source.split('/').at(-1); assert(filename);
    const emitted = await readFile(join(root, '.artifacts/forgejo-js', filename), 'utf8');
    // Includes static exports/imports and dynamic imports in the minified graph.
    for (const match of emitted.matchAll(/["'](\.{1,2}\/[^"']+\.js(?:\?[^"']*)?)["']/g)) {
      const specifier = match[1]; assert(specifier);
      const url = new URL(specifier, 'https://fixture.invalid/' + destination);
      assert.equal(url.search, `?v=${presentationVersion}`, `${destination}: ${url.pathname}`);
      assert(Object.hasOwn(payload, url.pathname.slice(1)), `Unstaged import ${url.pathname}`);
    }
  }
});

test('staged Lit notice matches the resolved browser dependency licenses', async () => {
  const license = await readFile(join(root, 'appliance/licenses/lit-LICENSE'), 'utf8');
  const litEntry = Bun.resolveSync('lit', root);
  for (const name of ['lit', 'lit-element', 'lit-html', '@lit/reactive-element']) {
    let directory = dirname(name === 'lit' ? litEntry : Bun.resolveSync(name, dirname(litEntry)));
    // Conditional Node entrypoints can live one level below the package root.
    while (!(await Bun.file(join(directory, 'package.json')).exists())) {
      const parent = dirname(directory);
      assert.notEqual(parent, directory, `Missing package metadata for ${name}`);
      directory = parent;
    }
    assert.equal((await readFile(join(directory, 'LICENSE'), 'utf8')).trimEnd(), license.trimEnd(), name);
  }
});

test('Lit scaffolding does not eagerly load or replace native page controls', async () => {
  for (const name of ['header', 'footer']) {
    const source = await readFile(join(root, `appliance/forgejo/templates/custom/${name}.tmpl`), 'utf8');
    assert.doesNotMatch(source, /lit\.js|soda-lit-smoke/);
  }
});
