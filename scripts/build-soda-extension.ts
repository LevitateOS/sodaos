import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {copyFile, mkdir, readdir, readFile, stat, writeFile} from 'node:fs/promises';
import {basename, dirname, resolve} from 'node:path';
import {parseArgs} from 'node:util';
import {feedRolePortraits} from '../frontend/spaces/sodaspaces-factory-feed-portraits.js';

const root = resolve(import.meta.dir, '..');
const allEntries = {
  'soda-spaces-entry.js': 'frontend/spaces/soda-spaces-entry.ts',
  'soda-feed-entry.js': 'frontend/spaces/soda-feed-entry.ts',
  'soda-repository-entry.js': 'frontend/spaces/soda-repository-entry.ts',
  'soda-tailnet-entry.js': 'frontend/tailnet/soda-tailnet-entry.ts',
} as const;
export const styles = {
  'components.css': 'assets/branding/forgejo/components.css',
  'sodaspaces.css': 'frontend/spaces/sodaspaces.css',
  'sodaspaces-page.css': 'frontend/spaces/sodaspaces-workspace.css',
  'sodaspaces-drawer.css': 'frontend/spaces/sodaspaces-project.css',
  'sodaspaces-terminal.css': 'frontend/spaces/sodaspaces-terminal.css',
  'soda-settings.css': 'frontend/tailnet/soda-settings.css',
  'soda-tailnet.css': 'frontend/tailnet/soda-tailnet.css',
} as const;

export type ExtensionPackage = 'soda' | 'tailnet';
function packageAssets(packageID: ExtensionPackage) {
  assert(packageID === 'soda' || packageID === 'tailnet', 'Exact extension package required');
  const entry = packageID === 'soda' ? 'soda-spaces-entry.js' : 'soda-tailnet-entry.js';
  const selectedStyles = Object.fromEntries(
    Object.entries(styles).filter(
      ([name]) =>
        name === 'components.css' || (packageID === 'soda' ? name.startsWith('sodaspaces') : name.startsWith('soda-'))
    )
  );
  const entries =
    packageID === 'soda'
      ? {
          [entry]: allEntries[entry],
          'soda-feed-entry.js': allEntries['soda-feed-entry.js'],
          'soda-repository-entry.js': allEntries['soda-repository-entry.js'],
        }
      : {[entry]: allEntries[entry]};
  return {entries, selectedStyles};
}

function source(path: string) {
  return resolve(root, path);
}

async function checkStylesUsedByEntries(entries: Record<string, string>, selectedStyles: Record<string, string>) {
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
  assert.deepEqual(
    [...used].sort(),
    Object.keys(selectedStyles).sort(),
    'Native entry styles and package styles differ'
  );
}

async function checkManifestEntries(packageID: ExtensionPackage, entries: Record<string, string>) {
  const manifest = JSON.parse(
    await readFile(
      source(`system/containers/extension/${packageID === 'tailnet' ? 'tailnet-package/' : ''}extension.json`),
      'utf8'
    )
  ) as {
    pages: Array<{entry: string}>;
    panels: Array<{entry: string}>;
  };
  // One shared Workspace owner (NFX-D02/H-OWNER) declares the same entry
  // for its page and panel, so manifest entries are compared as a set.
  const declared = [...new Set([...manifest.pages, ...(manifest.panels || [])].map((item) => item.entry))];
  assert.deepEqual(declared.sort(), Object.keys(entries).sort(), 'Soda manifest entries and browser source differ');
}

async function buildScripts(assets: string, entries: Record<string, string>) {
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

function closingParen(css: string, open: number): number {
  let depth = 0;
  for (let i = open; i < css.length; i++) {
    if (css[i] === '(') depth++;
    else if (css[i] === ')') {
      depth--;
      if (depth === 0) return i;
    }
  }
  throw new Error('Unbalanced parentheses in emitted stylesheet');
}

// Pinned Bun 1.4.2 lowers light-dark() to guard-variable fragments but emits
// the selection guards only alongside in-graph color-scheme rules. Native
// scheme selection lives outside this compilation graph, so restore the native
// call and let the browser resolve it against the external selection.
function restoreNativeSchemeCalls(css: string): string {
  const lightMarker = 'var(--buncss-light,';
  const darkMarker = 'var(--buncss-dark,';
  let output = '';
  let rest = css;
  for (;;) {
    const light = rest.indexOf(lightMarker);
    if (light < 0) {
      assert(!rest.includes('--buncss-'), 'Unpaired lowered scheme fragment in emitted stylesheet');
      return output + rest;
    }
    const lightEnd = closingParen(rest, light + 3);
    let dark = lightEnd + 1;
    while (rest[dark] === ' ' || rest[dark] === '\n' || rest[dark] === '\t' || rest[dark] === '\r') dark++;
    if (!rest.startsWith(darkMarker, dark)) throw new Error('Unpaired lowered scheme fragment in emitted stylesheet');
    const darkEnd = closingParen(rest, dark + 3);
    output += `${rest.slice(0, light)}light-dark(${rest.slice(light + lightMarker.length, lightEnd)},${rest.slice(dark + darkMarker.length, darkEnd)})`;
    rest = rest.slice(darkEnd + 1);
  }
}

function matchBrace(css: string, open: number): number {
  let depth = 0;
  for (let i = open; i < css.length; i++) {
    if (css[i] === '{') depth++;
    else if (css[i] === '}') {
      depth--;
      if (depth === 0) return i;
    }
  }
  throw new Error('Unbalanced braces in emitted stylesheet');
}

// H owns document-level native branding: the extension copy must never
// redeclare :root. Strip top-level :root rules from emitted extension
// stylesheets so the single document value source governs via inheritance.
function stripDocumentTokens(css: string): string {
  const kept: string[] = [];
  let i = 0;
  while (i < css.length) {
    let j = i;
    let parens = 0;
    let closer = -1;
    let isRule = false;
    while (j < css.length) {
      const char = css[j];
      if (char === '(') parens++;
      else if (char === ')') parens--;
      else if (parens === 0 && char === '{') {
        closer = j;
        isRule = true;
        break;
      } else if (parens === 0 && char === ';') {
        closer = j;
        break;
      }
      j++;
    }
    if (closer < 0) {
      kept.push(css.slice(i));
      break;
    }
    if (!isRule) {
      kept.push(css.slice(i, closer + 1));
      i = closer + 1;
      continue;
    }
    const selector = css.slice(i, closer).trim();
    const end = matchBrace(css, closer);
    if (selector === ':root') {
      i = end + 1;
      continue;
    }
    if (selector.includes(':root')) throw new Error(`Grouped :root selector needs explicit scoping: ${selector}`);
    kept.push(css.slice(i, end + 1));
    i = end + 1;
  }
  return kept.join('');
}

export async function buildExtensionStylesheets(
  assets: string,
  selectedStyles: Record<string, string> = styles
): Promise<void> {
  for (const [name, path] of Object.entries(selectedStyles)) {
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
    const emitted = resolve(assets, name);
    await writeFile(emitted, stripDocumentTokens(restoreNativeSchemeCalls(await readFile(emitted, 'utf8'))));
  }
  if (!('sodaspaces-terminal.css' in selectedStyles)) return;
  const terminal = resolve(assets, 'sodaspaces-terminal.css');
  await writeFile(terminal, `@import url("./soda-terminal/xterm.css");\n${await readFile(terminal, 'utf8')}`);
}

async function copyRolePortraits(assets: string) {
  for (const file of Object.values(feedRolePortraits)) {
    const destination = resolve(assets, file);
    await mkdir(dirname(destination), {recursive: true});
    await copyFile(source(`assets/branding/${file}`), destination);
  }
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

export async function buildSodaExtensionAssets(out: string, terminal: string, packageID: ExtensionPackage) {
  assert(
    (await stat(out).then(
      () => true,
      () => false
    )) === false,
    'Fresh Soda extension asset output required'
  );
  const {entries, selectedStyles} = packageAssets(packageID);
  await checkStylesUsedByEntries(entries, selectedStyles);
  await checkManifestEntries(packageID, entries);
  const assets = resolve(out, 'assets');
  await mkdir(assets, {recursive: true});
  await buildScripts(assets, entries);
  await buildExtensionStylesheets(assets, selectedStyles);
  if (packageID === 'soda') {
    await copyLockedTerminal(assets, terminal);
    await copyRolePortraits(assets);
  }
  await copyNotices(assets);
  const files = (await filesUnder(assets)).sort();
  assert(files.length > Object.keys(entries).length + Object.keys(selectedStyles).length);
  await writeFile(resolve(out, 'files.json'), JSON.stringify(files, null, 2) + '\n');
  return files;
}

if (import.meta.main) {
  const {values} = parseArgs({
    args: Bun.argv.slice(2),
    options: {package: {type: 'string'}, out: {type: 'string'}, 'terminal-assets': {type: 'string'}},
  });
  assert(values.out && values['terminal-assets'], 'Explicit output and locked terminal assets required');
  assert(values.package === 'soda' || values.package === 'tailnet', 'Explicit --package=soda|tailnet required');
  await buildSodaExtensionAssets(resolve(values.out), resolve(values['terminal-assets']), values.package);
}
