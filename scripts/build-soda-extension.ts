import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {copyFile, mkdir, readdir, readFile, stat, writeFile} from 'node:fs/promises';
import {basename, resolve} from 'node:path';
import {parseArgs} from 'node:util';

const root = resolve(import.meta.dir, '..');
const entries = {
  'soda-spaces-entry.js': 'frontend/spaces/soda-spaces-entry.ts',
  'soda-workspace-panel-entry.js': 'frontend/spaces/soda-workspace-panel-entry.ts',
  'soda-tailnet-entry.js': 'frontend/tailnet/soda-tailnet-entry.ts',
} as const;
const styles = {
  'components.css': 'assets/branding/forgejo/components.css',
  'sodaspaces.css': 'frontend/spaces/sodaspaces.css',
  'sodaspaces-page.css': 'frontend/spaces/sodaspaces-workspace.css',
  'sodaspaces-drawer.css': 'frontend/spaces/sodaspaces-project.css',
  'sodaspaces-terminal.css': 'frontend/spaces/sodaspaces-terminal.css',
  'soda-settings.css': 'frontend/tailnet/soda-settings.css',
  'soda-tailnet.css': 'frontend/tailnet/soda-tailnet.css',
} as const;

function source(path: string) {
  return resolve(root, path);
}

async function checkStylesUsedByEntries() {
  const used = new Set<string>();
  for (const entry of Object.values(entries)) {
    const body = await readFile(source(entry), 'utf8');
    const declaration = body.match(/const styles = \[([\s\S]*?)\]\.map\(/);
    const styleNames = declaration?.[1];
    assert(styleNames, `Missing stylesheet declaration in ${entry}`);
    for (const match of styleNames.matchAll(/'([a-z0-9-]+\.css)'/g)) {
      if (match[1]) used.add(match[1]);
    }
  }
  assert.deepEqual([...used].sort(), Object.keys(styles).sort(), 'Native entry styles and package styles differ');
}

async function checkManifestEntries() {
  const manifest = JSON.parse(await readFile(source('system/containers/extension/extension.json'), 'utf8')) as {
    pages: Array<{entry: string}>;
    panels: Array<{entry: string}>;
  };
  const declared = [...manifest.pages, ...manifest.panels].map((item) => item.entry);
  assert.deepEqual(declared.sort(), Object.keys(entries).sort(), 'Soda manifest entries and browser source differ');
}

async function buildScripts(assets: string) {
  const result = await Bun.build({
    entrypoints: Object.values(entries).map(source),
    outdir: assets,
    target: 'browser',
    format: 'esm',
    splitting: true,
    minify: true,
    naming: {entry: '[name].js', chunk: 'chunk-[hash].js'},
    define: {'process.env.NODE_ENV': '"production"'},
    plugins: [
      {
        name: 'extension-terminal-imports',
        setup(build) {
          build.onResolve({filter: /^\.\/soda-terminal\/(?:xterm|addon-fit)\.mjs$/}, (args) => ({
            path: args.path,
            external: true,
          }));
        },
      },
    ],
  });
  assert(result.success, `Soda browser build failed: ${result.logs.join('\n')}`);
  const output = result.outputs.filter((file) => file.kind === 'entry-point').map((file) => basename(file.path));
  assert.deepEqual(output.sort(), Object.keys(entries).sort(), 'Manifest entries lack built browser modules');
}

async function buildStyles(assets: string) {
  for (const [name, path] of Object.entries(styles)) {
    const result = await Bun.build({
      entrypoints: [source(path)],
      outdir: assets,
      target: 'browser',
      minify: true,
      naming: {entry: name},
      plugins: [
        {
          name: 'extension-icons',
          setup(build) {
            build.onResolve({filter: /^soda\/forgejo\/icons\/[a-z0-9.-]+\.svg$/}, (args) => ({
              path: source('assets/branding/icons/octicons/' + basename(args.path)),
            }));
          },
        },
      ],
    });
    assert(result.success, `Soda stylesheet build failed for ${name}: ${result.logs.join('\n')}`);
    assert.deepEqual(
      result.outputs.map((file) => basename(file.path)),
      [name]
    );
  }
  const terminal = resolve(assets, 'sodaspaces-terminal.css');
  await writeFile(terminal, `@import url("./soda-terminal/xterm.css");\n${await readFile(terminal, 'utf8')}`);
}

async function copyLockedTerminal(assets: string, terminal: string) {
  const lock = JSON.parse(await readFile(source('tools/release-assets/terminal-assets.lock.json'), 'utf8')) as Array<{
    files: Array<{file: string; sha256: string}>;
  }>;
  const destination = resolve(assets, 'soda-terminal');
  await mkdir(destination);
  for (const item of lock) {
    for (const file of item.files) {
      const bytes = await readFile(resolve(terminal, file.file));
      assert.equal(
        createHash('sha256').update(bytes).digest('hex'),
        file.sha256,
        `Locked terminal asset differs: ${file.file}`
      );
      await writeFile(resolve(destination, file.file), bytes);
    }
  }
}

async function copyNotices(assets: string) {
  const notices = resolve(assets, 'licenses');
  await mkdir(resolve(notices, 'fonts'), {recursive: true});
  const fonts = await readFile(source('assets/branding/fonts/fonts.css'), 'utf8');
  const families = new Set(
    [...fonts.matchAll(/url\("\.\/([a-z0-9-]+)\/[a-z0-9.-]+\.woff2"\)/g)].map((match) => match[1])
  );
  assert(families.size > 0, 'Expected local font families');
  for (const family of families) {
    await copyFile(source(`assets/branding/fonts/${family}/LICENSE`), resolve(notices, 'fonts', `${family}.LICENSE`));
  }
  await copyFile(source('assets/branding/fonts/sources.json'), resolve(notices, 'font-sources.json'));
  await copyFile(source('system/licenses/lit-LICENSE'), resolve(notices, 'lit.LICENSE'));
}

async function filesUnder(path: string, prefix = ''): Promise<string[]> {
  const files: string[] = [];
  for (const entry of await readdir(path, {withFileTypes: true})) {
    const name = prefix ? `${prefix}/${entry.name}` : entry.name;
    if (entry.isDirectory()) files.push(...(await filesUnder(resolve(path, entry.name), name)));
    else {
      assert(entry.isFile(), `Unsafe generated asset ${name}`);
      files.push(name);
    }
  }
  return files;
}

export async function buildSodaExtensionAssets(out: string, terminal: string) {
  assert(
    (await stat(out).then(
      () => true,
      () => false
    )) === false,
    'Fresh Soda extension asset output required'
  );
  await checkStylesUsedByEntries();
  await checkManifestEntries();
  const assets = resolve(out, 'assets');
  await mkdir(assets, {recursive: true});
  await buildScripts(assets);
  await buildStyles(assets);
  await copyLockedTerminal(assets, terminal);
  await copyNotices(assets);
  const files = (await filesUnder(assets)).sort();
  assert(files.length > Object.keys(entries).length + Object.keys(styles).length);
  await writeFile(resolve(out, 'files.json'), JSON.stringify(files, null, 2) + '\n');
  return files;
}

if (import.meta.main) {
  const {values} = parseArgs({
    args: Bun.argv.slice(2),
    options: {out: {type: 'string'}, 'terminal-assets': {type: 'string'}},
  });
  assert(values.out && values['terminal-assets'], 'Explicit output and locked terminal assets required');
  await buildSodaExtensionAssets(resolve(values.out), resolve(values['terminal-assets']));
}
