import {factorySource, factoryRoot, factoryBrowserImports} from './factory-source.js';
import {buildFactoryUI} from '@factory/product/scripts/build-factory-ui.js';
import {expect, test} from 'bun:test';
import {mkdir, mkdtemp, readFile, realpath, rm, stat} from 'node:fs/promises';
import {dirname, relative, resolve} from 'node:path';
import {buildExtensionStylesheets, buildSodaExtensionAssets, styles} from './build-soda-extension.js';
import {feedRolePortraits} from '@factory/product/frontend/factory/feed/feed-portraits.js';

const root = resolve(import.meta.dir, '..');

test('native Soda package contains its entire local browser asset graph', async () => {
  const artifactRoot = resolve(root, '.artifacts');
  await mkdir(artifactRoot, {recursive: true});
  const work = await mkdtemp(resolve(artifactRoot, 'soda-extension-test-'));
  try {
    const terminal = process.env.SODA_TEST_TERMINAL_ASSETS || resolve(work, 'terminal');
    if (!process.env.SODA_TEST_TERMINAL_ASSETS) {
      const fetch = Bun.spawnSync(
        [
          'cargo',
          'run',
          '--release',
          '--locked',
          '-p',
          'soda-release-assets',
          '--bin',
          'soda-fetch-terminal',
          '--',
          '--out',
          terminal,
        ],
        {cwd: root}
      );
      if (fetch.exitCode !== 0) throw Error(fetch.stderr.toString());
    }
    const out = resolve(work, 'output');
    const files = await buildSodaExtensionAssets(out, terminal, 'soda');
    expect(files).not.toContain('soda-tailnet-entry.js');
    expect(files).not.toContain('soda-spaces-entry.js');
    const optionalOut = resolve(work, 'tailnet');
    const optionalFiles = await buildSodaExtensionAssets(optionalOut, terminal, 'tailnet');
    expect(optionalFiles).toContain('soda-tailnet-entry.js');
    expect(optionalFiles).not.toContain('soda-spaces-entry.js');
    expect(optionalFiles.some((name) => name.startsWith('soda-terminal/'))).toBe(false);
    expect(optionalFiles.some((name) => name.startsWith('role-portraits/'))).toBe(false);
    for (const portrait of Object.values(feedRolePortraits)) {
      expect(files).toContain(portrait);
      expect(await readFile(resolve(out, 'assets', portrait))).toEqual(
        await readFile(resolve(root, 'assets/branding', portrait))
      );
    }
    for (const name of optionalFiles.filter((file) => file.endsWith('.js'))) {
      const body = await readFile(resolve(optionalOut, 'assets', name), 'utf8');
      for (const match of body.matchAll(/(?:from\s*|import\(\s*)["'](\.[^"']+)["']/g)) {
        if (!match[1]) throw Error('Missing generated import');
        expect((await stat(resolve(optionalOut, 'assets', dirname(name), match[1]))).isFile()).toBe(true);
      }
    }
    for (const name of [
      'soda-repository-entry.js',
      'components.css',
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
      'licenses/font-sources.json',
      'licenses/fonts/barlow.LICENSE',
      'licenses/fonts/barlow-condensed.LICENSE',
      'licenses/fonts/fraunces.LICENSE',
      'licenses/fonts/ibm-plex-mono.LICENSE',
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
    // Build the actual shared application independently of Forge and Storybook.
    const productOut = resolve(work, 'factory'),
      product = await buildFactoryUI(productOut);
    // Trace a real Fountain entry through Bun, then compare the physical common
    // component inputs with the independently built standalone product.
    const graph = await Bun.build({
      entrypoints: [resolve(root, 'frontend/spaces/soda-place-issues.ts')],
      target: 'browser',
      format: 'esm',
      metafile: true,
      define: {'process.env.NODE_ENV': '"production"'},
      plugins: [factoryBrowserImports()],
    });
    expect(graph.success).toBe(true);
    const inputs = Object.keys(graph.metafile!.inputs),
      common = new Set<string>();
    for (const input of inputs.filter((path) => path.includes('/frontend/factory/'))) {
      const path = await realpath(resolve(root, input));
      expect(path.startsWith(factoryRoot + '/frontend/factory/')).toBe(true);
      const source = relative(factoryRoot, path);
      if (product.sources.includes(source)) common.add(source);
    }
    expect(common.has('frontend/factory/issues/editor.ts')).toBe(true);
    expect(common.size).toBeGreaterThan(10);
    expect(inputs.filter((path) => path.endsWith('/lit-element/lit-element.js'))).toHaveLength(1);
    expect(product.files).toContain('factory-app.js');
    expect(product.files).toContain('factory.css');
    expect(product.sources.every((path) => path.startsWith('frontend/factory/') && !path.endsWith('.stories.ts'))).toBe(
      true
    );
    for (const source of [
      'app/browser.ts',
      'app/factory-app.ts',
      'feed/feed.ts',
      'request/request.ts',
      'request/exploration-record.ts',
      'work/progress.ts',
      'work/detail.ts',
      'library/library.ts',
      'watch/watch.ts',
    ])
      expect(product.sources).toContain('frontend/factory/' + source);
    const module = await readFile(resolve(productOut, 'factory-app.js'), 'utf8');
    expect(module).toContain('SodaFactoryApp');
    expect(module).toContain('SodaRequestError');
    expect(module).not.toContain('storybook');
    const css = await readFile(resolve(productOut, 'factory.css'), 'utf8');
    expect(css).toContain(':root{');
    expect(css).toContain('light-dark(');
    expect(css).not.toContain('--buncss-');
    expect(css).toContain('data:font/woff2;base64,');
    expect(css).not.toContain('data-page=');
    expect(product.files).toContain('licenses/lit.LICENSE');
    for (const portrait of Object.values(feedRolePortraits)) {
      expect(product.files).toContain(portrait);
      expect(await readFile(resolve(productOut, portrait))).toEqual(
        await readFile(resolve(root, 'assets/branding', portrait))
      );
    }
    for (const match of css.matchAll(/url\(([^)]+)\)/g)) {
      const url = match[1]?.replace(/^['"]|['"]$/g, '');
      if (!url || url.startsWith('data:')) continue;
      expect(url).not.toMatch(/^(?:[a-z]+:|\/\/)/i);
      expect((await stat(resolve(productOut, url))).isFile()).toBe(true);
    }
  } finally {
    await rm(work, {recursive: true, force: true});
  }
});

// Canonical palette endpoints each scheme token must resolve to under the
// externally selected native scheme. Independent literals, not source-derived.
// The minifier shortens some palette hex to named colors; map those explicitly.
const namedColors: Record<string, string> = {
  gray: '#808080',
  silver: '#c0c0c0',
};
const schemeTokens: Record<string, {light: string; dark: string}> = {
  '--soda-page-canvas': {light: '#ffffff', dark: '#101010'},
  '--soda-page-surface': {light: '#ffffff', dark: '#101010'},
  '--soda-page-text': {light: '#101010', dark: '#ffffff'},
  '--soda-page-muted': {light: '#555555', dark: '#bdbdbd'},
  '--soda-page-edge': {light: '#858585', dark: '#808080'},
  '--soda-page-link': {light: '#df001b', dark: '#ff6575'},
  '--soda-page-action': {light: '#df001b', dark: '#df001b'},
  '--soda-page-action-hover': {light: '#bc0017', dark: '#bc0017'},
  '--soda-page-on-action': {light: '#ffffff', dark: '#ffffff'},
};

function readCustomValue(css: string, valueStart: number, name: string): string {
  let depth = 0;
  for (let i = valueStart; i < css.length; i++) {
    const char = css[i];
    if (char === '(') depth++;
    else if (char === ')') depth--;
    else if (depth === 0 && (char === ';' || char === '}')) return css.slice(valueStart, i).trim();
  }
  throw new Error(`Unterminated emitted token ${name}`);
}

function collectDeclarations(css: string): Map<string, string> {
  const declarations = new Map<string, string>();
  for (const match of css.matchAll(/--([a-z0-9-]+):/g)) {
    const name = match[1] ? `--${match[1]}` : undefined;
    if (name === undefined || match.index === undefined || declarations.has(name)) continue;
    declarations.set(name, readCustomValue(css, match.index + name.length + 1, name));
  }
  return declarations;
}

function splitTopLevel(value: string, separator: string): string[] {
  const parts: string[] = [];
  let depth = 0;
  let current = '';
  for (const char of value) {
    if (char === '(') depth++;
    else if (char === ')') depth--;
    if (char === separator && depth === 0) {
      parts.push(current);
      current = '';
    } else {
      current += char;
    }
  }
  parts.push(current);
  return parts;
}

function matchClosingParen(text: string, open: number, value: string): number {
  let depth = 0;
  for (let i = open; i < text.length; i++) {
    if (text[i] === '(') depth++;
    else if (text[i] === ')') {
      depth--;
      if (depth === 0) return i;
    }
  }
  throw new Error(`Unbalanced parentheses in emitted value: ${value}`);
}

function substituteVars(
  value: string,
  scheme: 'light' | 'dark',
  declarations: Map<string, string>,
  depth: number
): string {
  if (depth > 10) throw new Error(`Unresolvable emitted value: ${value}`);
  const trimmed = value.trim();
  const lowered = /^light-dark\((.*)\)$/s.exec(trimmed);
  if (lowered?.[1] !== undefined) {
    const branches = splitTopLevel(lowered[1], ',');
    const branch = scheme === 'light' ? branches[0] : branches[1];
    if (branches.length !== 2 || branch === undefined) throw new Error(`Invalid emitted light-dark(): ${value}`);
    return substituteVars(branch.trim(), scheme, declarations, depth + 1);
  }
  let output = '';
  let rest = trimmed;
  for (;;) {
    const index = rest.indexOf('var(');
    if (index < 0) return output + rest;
    output += rest.slice(0, index);
    const end = matchClosingParen(rest, index + 3, value);
    const args = splitTopLevel(rest.slice(index + 4, end), ',');
    const name = args[0]?.trim();
    if (name === undefined || !name.startsWith('--')) throw new Error(`Invalid emitted var(): ${value}`);
    const bound = declarations.get(name);
    if (bound !== undefined) output += substituteVars(bound, scheme, declarations, depth + 1);
    else if (args.length === 2 && args[1] !== undefined)
      output += substituteVars(args[1].trim(), scheme, declarations, depth + 1);
    else throw new Error(`Unbound emitted variable ${name} in: ${value}`);
    rest = rest.slice(end + 1);
  }
}

function resolveSchemeColor(declarations: Map<string, string>, token: string, scheme: 'light' | 'dark'): string {
  const raw = declarations.get(token);
  if (raw === undefined) throw new Error(`Missing token ${token}`);
  const expanded = substituteVars(raw, scheme, declarations, 0).trim();
  const color = expanded.toLowerCase();
  if (/^[a-z]+$/.test(color)) {
    const mapped = namedColors[color];
    if (mapped === undefined) throw new Error(`${token} resolves to unmapped named ${scheme} color: ${expanded}`);
    return mapped;
  }
  const short = /^#([0-9a-f]{3})$/.exec(color);
  const full = short?.[1]
    ? `#${short[1]
        .split('')
        .map((c) => c + c)
        .join('')}`
    : color;
  if (!/^#[0-9a-f]{6}$/.test(full) && !/^#[0-9a-f]{8}$/.test(full)) {
    throw new Error(`${token} resolves to invalid ${scheme} color: ${expanded}`);
  }
  return full;
}

function schemeTokenNames(css: string): string[] {
  const names = new Set<string>();
  for (const match of css.matchAll(/--(soda-page-[a-z-]+|soda-button-primary-[a-z-]+|soda-button-danger):/g)) {
    if (match[1]) names.add(`--${match[1]}`);
  }
  return [...names].sort();
}

async function documentCss(): Promise<string> {
  const palette = await readFile(factorySource('assets/branding/theme/palette.css'), 'utf8');
  const components = await readFile(resolve(root, 'assets/branding/forgejo/components.css'), 'utf8');
  const tokens = await readFile(factorySource('assets/branding/theme/tokens.css'), 'utf8');
  return `${palette}\n${tokens}\n${components}`;
}

function matchClosingBrace(text: string, open: number, what: string): number {
  let depth = 0;
  for (let i = open; i < text.length; i++) {
    if (text[i] === '{') depth++;
    else if (text[i] === '}') {
      depth--;
      if (depth === 0) return i;
    }
  }
  throw new Error(`Unbalanced braces in ${what}`);
}

function ruleBody(css: string, selector: string): string {
  const start = css.indexOf(`${selector}{`);
  if (start < 0) throw new Error(`Missing emitted rule ${selector}`);
  return css.slice(start + selector.length + 1, matchClosingBrace(css, start + selector.length, `rule ${selector}`));
}

function declarationValue(body: string, property: string, what: string): string {
  for (const part of splitTopLevel(body, ';')) {
    const [name, ...rest] = part.split(':');
    if (name?.trim() === property) return rest.join(':').trim();
  }
  throw new Error(`Missing ${property} in ${what}`);
}

function resolveReferencedColors(
  value: string,
  scheme: 'light' | 'dark',
  declarations: Map<string, string>,
  what: string
): void {
  let found = false;
  let rest = value;
  for (;;) {
    const index = rest.indexOf('var(');
    if (index < 0) break;
    found = true;
    const end = matchClosingParen(rest, index + 3, value);
    const ref = rest.slice(index, end + 1);
    const resolved = substituteVars(ref, scheme, declarations, 0).trim().toLowerCase();
    if (!/^#[0-9a-f]{6}$/.test(resolved) && !/^#[0-9a-f]{8}$/.test(resolved) && namedColors[resolved] === undefined) {
      throw new Error(`${what} references invalid ${scheme} color: ${ref} resolves to ${resolved}`);
    }
    rest = rest.slice(end + 1);
  }
  if (!found) throw new Error(`${what} references no document token: ${value}`);
}

test('document scheme source resolves one valid color per native scheme', async () => {
  const doc = await documentCss();
  expect(doc.includes('light-dark(')).toBe(true);
  const declarations = collectDeclarations(doc);
  for (const [token, expected] of Object.entries(schemeTokens)) {
    for (const scheme of ['light', 'dark'] as const) {
      expect(resolveSchemeColor(declarations, token, scheme)).toBe(expected[scheme]);
    }
  }
  for (const token of schemeTokenNames(doc)) {
    if (token in schemeTokens) continue;
    for (const scheme of ['light', 'dark'] as const) {
      resolveSchemeColor(declarations, token, scheme);
    }
  }
});

test('emitted extension stylesheets defer to document tokens', async () => {
  const artifactRoot = resolve(root, '.artifacts');
  await mkdir(artifactRoot, {recursive: true});
  const work = await mkdtemp(resolve(artifactRoot, 'soda-extension-styles-test-'));
  try {
    const assets = resolve(work, 'assets');
    await buildExtensionStylesheets(assets);
    for (const name of Object.keys(styles)) {
      const body = await readFile(resolve(assets, name), 'utf8');
      expect(body.includes('--buncss-')).toBe(false);
      expect(body.includes(':root')).toBe(false);
    }
    const declarations = collectDeclarations(await documentCss());
    const dialog = ruleBody(await readFile(resolve(assets, 'sodaspaces-page.css'), 'utf8'), '.soda-workspace-dialog');
    const confirm = ruleBody(
      await readFile(resolve(assets, 'sodaspaces-terminal.css'), 'utf8'),
      '.soda-terminal-confirm'
    );
    for (const scheme of ['light', 'dark'] as const) {
      resolveReferencedColors(
        declarationValue(dialog, 'background', 'workspace dialog'),
        scheme,
        declarations,
        'workspace dialog background'
      );
      resolveReferencedColors(
        declarationValue(dialog, 'border', 'workspace dialog'),
        scheme,
        declarations,
        'workspace dialog border'
      );
      resolveReferencedColors(
        declarationValue(dialog, 'color', 'workspace dialog'),
        scheme,
        declarations,
        'workspace dialog text'
      );
      resolveReferencedColors(
        declarationValue(confirm, 'background', 'terminal confirm'),
        scheme,
        declarations,
        'terminal confirm background'
      );
      resolveReferencedColors(
        declarationValue(confirm, 'border', 'terminal confirm'),
        scheme,
        declarations,
        'terminal confirm border'
      );
      resolveReferencedColors(
        declarationValue(confirm, 'color', 'terminal confirm'),
        scheme,
        declarations,
        'terminal confirm text'
      );
    }
  } finally {
    await rm(work, {recursive: true, force: true});
  }
});
