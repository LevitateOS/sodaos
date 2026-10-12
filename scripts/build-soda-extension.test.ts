import {expect, test} from 'bun:test';
import {mkdir, mkdtemp, readFile, rm, stat} from 'node:fs/promises';
import {dirname, resolve} from 'node:path';
import {buildExtensionStylesheets, buildSodaExtensionAssets} from './build-soda-extension.js';

const root = resolve(import.meta.dir, '..');

test('native Soda package contains its entire local browser asset graph', async () => {
  const artifactRoot = resolve(root, '.artifacts');
  await mkdir(artifactRoot, {recursive: true});
  const work = await mkdtemp(resolve(artifactRoot, 'soda-extension-test-'));
  try {
    const terminal = resolve(work, 'terminal');
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

// Canonical palette endpoints each scheme token must resolve to under the
// externally selected native scheme. Independent literals, not source-derived.
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
  if (raw === undefined) throw new Error(`Missing emitted token ${token}`);
  const color = substituteVars(raw, scheme, declarations, 0).trim().toLowerCase();
  const short = /^#([0-9a-f]{3})$/.exec(color);
  const full = short?.[1]
    ? `#${short[1]
        .split('')
        .map((c) => c + c)
        .join('')}`
    : color;
  if (!/^#[0-9a-f]{6}$/.test(full) && !/^#[0-9a-f]{8}$/.test(full)) {
    throw new Error(
      `Emitted ${token} resolves to invalid ${scheme} color: ${substituteVars(raw, scheme, declarations, 0).trim()}`
    );
  }
  return full;
}

function schemeTokenNames(css: string): string[] {
  const names = new Set<string>();
  for (const match of css.matchAll(/--(soda-(?:page|button)-[a-z-]+):/g)) {
    if (match[1]) names.add(`--${match[1]}`);
  }
  return [...names].sort();
}

test('emitted extension styles resolve one valid color per native scheme', async () => {
  const artifactRoot = resolve(root, '.artifacts');
  await mkdir(artifactRoot, {recursive: true});
  const work = await mkdtemp(resolve(artifactRoot, 'soda-extension-styles-test-'));
  try {
    const assets = resolve(work, 'assets');
    await buildExtensionStylesheets(assets);
    for (const name of ['components.css', 'sodaspaces.css', 'sodaspaces-page.css']) {
      const body = await readFile(resolve(assets, name), 'utf8');
      expect(body.includes('--buncss-')).toBe(false);
    }
    const emitted = await readFile(resolve(assets, 'components.css'), 'utf8');
    expect(emitted.includes('light-dark(')).toBe(true);
    const declarations = collectDeclarations(emitted);
    for (const [token, expected] of Object.entries(schemeTokens)) {
      for (const scheme of ['light', 'dark'] as const) {
        expect(resolveSchemeColor(declarations, token, scheme)).toBe(expected[scheme]);
      }
    }
    for (const token of schemeTokenNames(emitted)) {
      if (token in schemeTokens) continue;
      for (const scheme of ['light', 'dark'] as const) {
        resolveSchemeColor(declarations, token, scheme);
      }
    }
  } finally {
    await rm(work, {recursive: true, force: true});
  }
});
