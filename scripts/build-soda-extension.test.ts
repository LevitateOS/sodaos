import {expect, test} from 'bun:test';
import {createHash} from 'node:crypto';
import {mkdir, mkdtemp, readFile, rm, stat} from 'node:fs/promises';
import {dirname, resolve} from 'node:path';
import {buildExtensionStylesheets, buildSodaExtensionAssets, styles} from './build-soda-extension.js';

const root = resolve(import.meta.dir, '..');

// NFX-E04 exact output identities (H-ASSET): sha256 of every emitted file at
// Soda 72284a49 after the wave-2 D/F/H entry and style graphs. Refresh this
// map (and only this map) when a graph input changes; any other mismatch is
// a builder regression.
const expectedIdentities: Record<string, string> = {
  'chunk-qht6w2m6.js': '220f12c5e7409060aaf0e45d01e7e4d3a58df6b7848f58a9f333952081a11a9d',
  'components.css': 'e7b2ecd9341decdb1ec7b010191baad14fdd148af632b2f86cd6c973ed778fb0',
  'licenses/font-sources.json': '93979bcffd6015d411f7ab171da4ed9f1a28ef841830bea430e9ca7635b92979',
  'licenses/fonts/barlow-condensed.LICENSE': '046ee27d8bddc6b6ee08eee53c96dbf3ec1f6df561644e018507dd8ffcb11d31',
  'licenses/fonts/barlow.LICENSE': '4877e0c1b66f0b16cc3ebc4801c4def24c06252f618b9229428b20c7387aa0c0',
  'licenses/fonts/fraunces.LICENSE': 'cd3384cafac6f2bddc3955273958a2e029f97027c8037ac539ef5744a77b579e',
  'licenses/fonts/ibm-plex-mono.LICENSE': '23b0a9d0c6d3f140a0b77e483c5cfa6bba574325ef5cb189ed9f2fec4884533f',
  'licenses/lit.LICENSE': '45d31799d0db956cc3eb5469346abbd9b7025babc5ff29fab10d7095da992ef1',
  'soda-settings.css': '9bd5b645d7c390d7054fb02276e712d0af99ce051a7f039680b543d938ad9250',
  'soda-spaces-entry.js': '2a7ecf25d1b5dfa6f6d02e7930e1aabd0dfd34932c25f6fe7136538d58c0b7e0',
  'soda-tailnet-entry.js': 'f55bd4af43db19efa6024d538980e2457f57e10502b503f718c65d5902afa138',
  'soda-tailnet.css': 'badea9c71552068047e5cee6bdabfdfa8ee1346536fb021c66c6f074e9c67b98',
  'soda-terminal/addon-fit.mjs': '2d87e1bddc73be9111de8beee5370c3bb7aac9c94e18e6f245f02ca741ef1769',
  'soda-terminal/fit.LICENSE': 'e256f01188af527e4d06d21d06fbf785ae9c50d4b328bf03cbe0ba7f0aa4228f',
  'soda-terminal/xterm.LICENSE': 'b569f629d00f2626a8100df2a1798210535621e42164dfd426a6fe5aac7b0ccd',
  'soda-terminal/xterm.css': '854a7c0fb70e8b1a083c16797ab827299fb18744f5ad34f227b48337e33293c6',
  'soda-terminal/xterm.mjs': 'b336ec65a086c056d4804b3d4c2347da5663d3f23c3f25be866467bd8857ad59',
  'sodaspaces-drawer.css': '0ad011730a229f7cfa44cc674f681e4340c6fc19ef2fd10be05392ad5a16a825',
  'sodaspaces-page.css': '22905de3233107bcfd86515a636bd4dbcac3617a3f4136126d35a79a87029e8a',
  'sodaspaces-terminal.css': '4bc8623981bde5b97e514d0ceca2ad907218f028ac5b3bfd7cd262c0eeaf0a56',
  'sodaspaces.css': 'ed093cc63d692c03d85bfda39aecfa725f5b38ae8c261ba861f4a292ff1bc82b',
};

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
      'licenses/font-sources.json',
      'licenses/fonts/barlow.LICENSE',
      'licenses/fonts/barlow-condensed.LICENSE',
      'licenses/fonts/fraunces.LICENSE',
      'licenses/fonts/ibm-plex-mono.LICENSE',
    ])
      expect(files).toContain(name);
    expect(files).toEqual(Object.keys(expectedIdentities).sort());
    for (const [name, sha256] of Object.entries(expectedIdentities)) {
      const bytes = await readFile(resolve(out, 'assets', name));
      expect(createHash('sha256').update(bytes).digest('hex')).toBe(sha256);
    }
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
  const palette = await readFile(resolve(root, 'assets/branding/theme/palette.css'), 'utf8');
  const components = await readFile(resolve(root, 'assets/branding/forgejo/components.css'), 'utf8');
  return `${palette}\n${components}`;
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
