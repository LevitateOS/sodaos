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
  const errText = result.stderr.toString();
  if (errText) process.stderr.write(errText);
  if (result.exitCode !== 0) throw new Error(`${argv.join(' ')}: ${errText || result.stdout.toString()}`);
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
export function measure(files: Source[]): Row[] {
  const rows = measureTypeScript(files.filter((f) => language(f.Path) === 'typescript'));
  const go = files.filter((f) => language(f.Path) === 'go');
  if (go.length) rows.push(...JSON.parse(command(['go', 'run', '-buildvcs=false', './tools/source-complexity/go'], JSON.stringify(go))));
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
  // Distinguish conditional duplicate declarations in the report.
  const seen = new Map<string, number>();
  for (const row of rows) {
    const id = key(row),
      count = (seen.get(id) ?? 0) + 1;
    seen.set(id, count);
    if (count > 1) row.name += `#${count}`;
  }
  return rows;
}

function sources(paths: string[], staged: boolean): Source[] {
  return paths
    .filter((p) => language(p))
    .map((Path) => ({
      Path,
      Text: staged ? git('show', `:${Path}`) : readFileSync(Path, 'utf8'),
    }))
    .filter((f) => !generated(f.Text));
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
      language: {type: 'string'},
    },
  });
  const {staged, all, language: lang} = values;
  let paths = positionals;
  if (lang && !['go', 'rust', 'typescript'].includes(lang)) throw new Error('Unknown language');
  if (!paths.length) {
    if (all)
      paths = git('ls-files', '-z', '--cached', ...(staged ? [] : ['--others', '--exclude-standard'])).split('\0');
    else
      paths = git(
        'diff',
        ...(staged ? ['--cached'] : []),
        '--name-only',
        '-z',
        '--diff-filter=ACMR',
        values.base ?? 'HEAD'
      ).split('\0');
    if (!staged && !all) paths.push(...git('ls-files', '--others', '--exclude-standard', '-z').split('\0'));
  }
  paths = [...new Set(paths)].filter((p) => language(p) && (!lang || language(p) === lang));
  if (!staged) {
    if (positionals.length) {
      const missing = paths.find((p) => !existsSync(p));
      if (missing !== undefined) throw new Error(`Selected input is missing: ${missing}`);
    }
    paths = paths.filter((p) => existsSync(p));
  }
  const rows = measure(sources(paths, staged ?? false));
  summary(rows);
}
if (import.meta.main) {
  try {
    main();
  } catch (error) {
    console.error(String(error));
    process.exitCode = 1;
  }
}
