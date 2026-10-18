import {existsSync, mkdirSync, readFileSync} from 'node:fs';
import {resolve} from 'node:path';
import {parseArgs} from 'node:util';
import {measureTypeScript, type Row, type Source} from './typescript';

export function command(argv: string[], input?: string): string {
  const result = Bun.spawnSync(argv, {
    stdin: input === undefined ? undefined : Buffer.from(input),
    stdout: 'pipe',
    stderr: 'pipe',
    env: process.env,
  });
  if (result.exitCode !== 0)
    throw new Error(`${argv.join(' ')}: ${result.stderr.toString() || result.stdout.toString()}`);
  return result.stdout.toString();
}
function git(...args: string[]) {
  return command(['git', ...args]);
}
export function language(path: string): string | undefined {
  if (/^(build|dist)\//.test(path) || /(^|\/)(test|tests)\.rs$/.test(path)) return;
  if (
    /(^|\/)(tests?|__tests__|fixtures?|testdata|vendor|third_party|node_modules|target|generated|provenance|\.artifacts|\.local|\.git)(\/|$)/.test(
      path
    )
  )
    return;
  if (
    /(_test\.go|(?:[._](?:test|spec)|_tests)\.(?:rs|tsx?)|\.d\.ts|(?:\.gen|\.generated|_generated)\.(?:go|rs|tsx?))$/.test(
      path
    )
  )
    return;
  if (path.endsWith('.go')) return 'go';
  if (path.endsWith('.rs')) return 'rust';
  if (/\.tsx?$/.test(path)) return 'typescript';
}
export function generated(text: string): boolean {
  return /^(?:\/\/|\/\*|\s*\*)\s*(?:Code generated .*DO NOT EDIT|@generated\b|.*automatically generated)/im.test(
    text.slice(0, 2048)
  );
}
export function key(row: Row): string {
  return `${row.path}::${row.name}`;
}
export interface Baseline {
  version: number;
  revision: string;
  functions: Record<string, number>;
}
export function validateBaseline(baseline: Baseline) {
  if (baseline.version !== 1 || !/^[0-9a-f]{40}$/.test(baseline.revision))
    throw new Error('Invalid complexity baseline');
  for (const [name, cap] of Object.entries(baseline.functions)) {
    if (!name.includes('::') || !Number.isInteger(cap) || cap <= 20) throw new Error(`Invalid baseline cap: ${name}`);
  }
}

export interface Exception {
  max: number;
  reason: string;
}
export function violation(
  row: Row,
  before: Row | undefined,
  baseline: Baseline,
  exceptions: Record<string, Exception>
): string | undefined {
  const exception = exceptions[key(row)];
  if (exception) {
    if (!Number.isInteger(exception.max) || exception.max <= 20 || exception.reason.trim().length < 20)
      throw new Error(`Invalid narrow exception for ${key(row)}`);
    if (row.score <= exception.max) return;
    return `exceeds justified cap ${exception.max}: ${exception.reason}`;
  }
  if (row.score <= 20) return;
  const inherited = baseline.functions[key(row)];
  const cap = inherited === undefined ? 20 : Math.max(20, Math.min(inherited, before?.score ?? 20));
  if (row.score > cap) return `exceeds ${cap}; simplify substantively or record a specific capped justification`;
}

export function measure(files: Source[]): Row[] {
  const rows = measureTypeScript(files.filter((f) => language(f.Path) === 'typescript'));
  const go = files.filter((f) => language(f.Path) === 'go');
  if (go.length) rows.push(...JSON.parse(command(['go', 'run', './tools/source-complexity/go'], JSON.stringify(go))));
  const rust = files.filter((f) => language(f.Path) === 'rust');
  if (rust.length) {
    command([
      'cargo',
      'build',
      '--quiet',
      '--locked',
      '--offline',
      '--manifest-path',
      'tools/source-complexity/rust/Cargo.toml',
      '--target-dir',
      '.artifacts/source-complexity/rust',
    ]);
    rows.push(
      ...JSON.parse(command(['.artifacts/source-complexity/rust/debug/soda-source-complexity'], JSON.stringify(rust)))
    );
  }
  // Conditional duplicate declarations get separate identities; ambiguous identity
  // changes cannot inherit an unrelated high baseline.
  const seen = new Map<string, number>();
  for (const row of rows) {
    const id = key(row),
      count = (seen.get(id) ?? 0) + 1;
    seen.set(id, count);
    if (count > 1) row.name += `#${count}`;
  }
  return rows;
}

function sources(paths: string[], revision?: string): Source[] {
  return paths
    .filter((p) => language(p))
    .map((Path) => ({
      Path,
      Text: revision === undefined ? readFileSync(Path, 'utf8') : git('show', `${revision}:${Path}`),
    }))
    .filter((f) => !generated(f.Text));
}
function atRevision(paths: string[], revision: string): Source[] {
  const present = new Set(git('ls-tree', '-r', '--name-only', '-z', revision).split('\0'));
  return sources(
    paths.filter((p) => present.has(p)),
    revision
  );
}
function summary(rows: Row[]) {
  for (const lang of ['go', 'rust', 'typescript']) {
    const selected = rows.filter((r) => language(r.path) === lang);
    const max = selected.reduce<Row | undefined>((a, b) => (!a || b.score > a.score ? b : a), undefined);
    console.log(
      `${lang}: ${selected.length} functions; max ${max?.score ?? 0}${max ? ` (${key(max)})` : ''}; ${selected.filter((r) => r.score > 10).length} above preferred 10`
    );
  }
}
function main() {
  process.chdir(resolve(import.meta.dir, '../..'));
  mkdirSync('.artifacts/source-complexity/tmp', {recursive: true});
  process.env.TMPDIR = resolve('.artifacts/source-complexity/tmp');
  process.env.GOTMPDIR = process.env.TMPDIR;
  process.env.GOTOOLCHAIN = 'local';
  process.env.GOWORK = 'off';
  process.env.GOFLAGS = '-mod=readonly';
  const {values, positionals} = parseArgs({
    args: process.argv.slice(2),
    allowPositionals: true,
    options: {
      staged: {type: 'boolean'},
      all: {type: 'boolean'},
      report: {type: 'boolean'},
      base: {type: 'string', default: 'HEAD'},
      snapshot: {type: 'string'},
      language: {type: 'string'},
    },
  });
  const {staged, all, report, snapshot, language: lang} = values;
  let base = values.base;
  let paths = positionals;
  if (lang && !['go', 'rust', 'typescript'].includes(lang)) throw new Error('Unknown language');
  base = git('rev-parse', '--verify', `${base}^{commit}`).trim();
  if (snapshot !== undefined) {
    const revision = git('rev-parse', '--verify', `${snapshot}^{commit}`).trim();
    const files = git('ls-tree', '-r', '--name-only', '-z', revision).split('\0').filter(Boolean);
    const rows = measure(sources(files, revision));
    const functions = Object.fromEntries(rows.filter((r) => r.score > 20).map((r) => [key(r), r.score]));
    console.log(JSON.stringify({version: 1, revision, functions}, null, 2));
    return;
  }
  if (!paths.length) {
    if (all)
      paths = git('ls-files', '-z', '--cached', ...(staged ? [] : ['--others', '--exclude-standard'])).split('\0');
    else
      paths = git('diff', ...(staged ? ['--cached'] : []), '--name-only', '-z', '--diff-filter=ACMR', base).split('\0');
    if (!staged && !all) paths.push(...git('ls-files', '--others', '--exclude-standard', '-z').split('\0'));
  }
  paths = [...new Set(paths)].filter((p) => language(p) && (!lang || language(p) === lang));
  if (!staged) paths = paths.filter((p) => existsSync(p));
  const rows = measure(sources(paths, staged ? '' : undefined));
  summary(rows);
  if (report) return;
  const policyText = (path: string) => (staged ? git('show', `:${path}`) : readFileSync(path, 'utf8'));
  const baseline = JSON.parse(policyText('tools/source-complexity/baseline.json')) as Baseline;
  validateBaseline(baseline);
  const exceptions = JSON.parse(policyText('tools/source-complexity/exceptions.json')) as Record<string, Exception>;
  const previous = new Map(measure(atRevision(paths, base)).map((r) => [key(r), r]));
  let failures = 0;
  for (const row of rows) {
    const reason = violation(row, previous.get(key(row)), baseline, exceptions);
    if (reason) {
      console.error(`${row.path}:${row.line}: ${row.name} complexity ${row.score}: ${reason}`);
      failures++;
    }
  }
  if (failures) throw new Error(`${failures} complexity violation(s)`);
  console.log('Complexity gate passed (preferred <=10, normal maximum 20; existing excess cannot increase).');
}
if (import.meta.main) {
  try {
    main();
  } catch (error) {
    console.error(String(error));
    process.exitCode = 1;
  }
}
