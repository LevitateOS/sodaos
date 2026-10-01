import {expect, test} from 'bun:test';
import {mkdir, mkdtemp, readFile, rm, stat} from 'node:fs/promises';
import {dirname, resolve} from 'node:path';
import {buildSodaExtensionAssets} from './build-soda-extension.js';

const root = resolve(import.meta.dir, '..');

test('native Soda package contains its entire local browser asset graph', async () => {
  const artifactRoot = resolve(root, '.artifacts');
  await mkdir(artifactRoot, {recursive: true});
  const work = await mkdtemp(resolve(artifactRoot, 'soda-extension-test-'));
  try {
    const terminal = resolve(work, 'terminal');
    const fetch = Bun.spawnSync(['python3', 'scripts/fetch-terminal.py', '--out', terminal], {cwd: root});
    expect(fetch.exitCode).toBe(0);
    const out = resolve(work, 'output');
    const files = await buildSodaExtensionAssets(out, terminal);
    const listed = JSON.parse(await readFile(resolve(out, 'files.json'), 'utf8')) as string[];
    expect(listed).toEqual(files);
    for (const name of [
      'soda-spaces-entry.js',
      'soda-workspace-panel-entry.js',
      'soda-tailnet-entry.js',
      'components.css',
      'soda-settings.css',
      'soda-tailnet.css',
      'sodaspaces.css',
      'sodaspaces-page.css',
      'sodaspaces-drawer.css',
      'sodaspaces-terminal.css',
      'soda-terminal/xterm.mjs',
      'soda-terminal/addon-fit.mjs',
      'soda-terminal/xterm.css',
      'soda-terminal/xterm.LICENSE',
      'soda-terminal/fit.LICENSE',
      'licenses/lit.LICENSE',
    ])
      expect(files).toContain(name);
    for (const name of files.filter((file) => file.endsWith('.js'))) {
      const body = await readFile(resolve(out, 'assets', name), 'utf8');
      for (const match of body.matchAll(/(?:from\s*|import\(\s*)["'](\.[^"']+)["']/g)) {
        const specifier = match[1];
        if (!specifier) throw Error('Missing generated import path');
        const target = resolve(out, 'assets', dirname(name), specifier);
        expect((await stat(target)).isFile()).toBe(true);
      }
    }
    const components = await readFile(resolve(out, 'assets/components.css'), 'utf8');
    const workspace = await readFile(resolve(out, 'assets/sodaspaces-page.css'), 'utf8');
    const terminalCSS = await readFile(resolve(out, 'assets/sodaspaces-terminal.css'), 'utf8');
    expect(components).toContain('data:font/woff2;base64,');
    expect(workspace).toContain('data:image/svg+xml;base64,');
    expect(workspace).not.toContain('soda/forgejo/icons/');
    expect(terminalCSS).toContain('@import url("./soda-terminal/xterm.css");');
  } finally {
    await rm(work, {recursive: true, force: true});
  }
});
