// Build the maintained read-only driver, phase module and native probe together.
// No appliance access, source copies, credential reads or dependency installation.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdir, readFile, realpath, stat, writeFile} from 'node:fs/promises';
import path from 'node:path';

process.umask(0o077);
const root = path.resolve(import.meta.dir, '..');
const args = process.argv.slice(2);
assert(
  args.length === 2 && args[0] === '--out',
  'usage: bun scripts/build-qualification.ts --out FRESH_HOME_DIRECTORY'
);
const out = args[1]!;
assert(path.isAbsolute(out) && out.startsWith('/home/') && path.normalize(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out), 'output parent must exist and be canonical');
assert(!out.startsWith(root + '/'), 'build outputs belong outside the source checkout');

async function command(argv: string[], cwd = root) {
  const process = Bun.spawn(argv, {cwd, env: {...Bun.env, CGO_ENABLED: '0'}, stdout: 'pipe', stderr: 'inherit'});
  const output = await new Response(process.stdout).text();
  assert.equal(await process.exited, 0, 'build command failed: ' + argv[0]);
  return output.trim();
}
const git = (argv: string[], cwd = root) => command(['git', ...argv], cwd);
const sha256 = (bytes: Uint8Array) => createHash('sha256').update(bytes).digest('hex');
async function artifact(file: string) {
  const bytes = await readFile(file);
  return {path: file, sha256: sha256(bytes), bytes: bytes.length};
}
async function cleanSource() {
  assert.equal(
    await git(['status', '--porcelain', '--untracked-files=all']),
    '',
    'commit source changes before building'
  );
  return git(['rev-parse', 'HEAD']);
}
const source = await cleanSource();
assert(/^[a-f0-9]{40}$/.test(source));
// go.mod owns local replacement selection. Observe it, rather than changing a
// sibling checkout to match an unrelated installed payload/Fountain pin.
const sdk = await realpath(await command(['go', 'list', '-m', '-f', '{{.Dir}}', 'forgejo.org/extension-sdk']));
const sdkRoot = await git(['rev-parse', '--show-toplevel'], sdk);
const sdkRelative = path.relative(sdkRoot, sdk);
assert(sdkRelative && !sdkRelative.startsWith('..'));
async function sdkSource() {
  assert.equal(
    await git(['status', '--porcelain', '--untracked-files=all', '--', sdkRelative], sdkRoot),
    '',
    'SDK inputs must be committed'
  );
  return git(['rev-parse', 'HEAD'], sdkRoot);
}
const sdkRevision = await sdkSource();
const inputs = await Promise.all(
  ['package.json', 'bun.lock', 'go.mod', 'go.sum'].map((file) => artifact(path.join(root, file)))
);
await mkdir(out, {mode: 0o700});
const driver = path.join(out, 'soda-qualification-journey');
const phases = path.join(out, 'phases.js');
const probe = path.join(out, 'soda-installed-probes');
for (const config of [
  {entrypoints: [path.join(root, 'tests/installed/qualification/main.ts')], compile: {outfile: driver}},
  {entrypoints: [path.join(root, 'tests/installed/qualification/phases/index.ts')], outdir: out, naming: 'phases.js'},
]) {
  const built = await Bun.build({...config, target: 'bun', minify: false});
  assert(built.success, 'qualification bundle failed: ' + built.logs.join('\n'));
}
const linker = 'github.com/levitateos/sodaos/internal/acceptance.FactoryContextToolRevision=' + source;
await command([
  'go',
  'build',
  '-mod=readonly',
  '-trimpath',
  '-ldflags',
  '-X ' + linker,
  '-o',
  probe,
  './tools/soda-installed-probes',
]);
assert.equal(await cleanSource(), source, 'source changed during build');
assert.equal(await sdkSource(), sdkRevision, 'SDK changed during build');
for (const input of inputs) assert.equal((await artifact(input.path)).sha256, input.sha256, 'dependency input changed');
for (const executable of [driver, probe]) assert((await stat(executable)).mode & 0o111, 'built executable required');
const receipt = {
  format: 1,
  source_revision: source,
  source_clean: true,
  build_entrypoint: 'scripts/build-qualification.ts',
  local_dependencies: [
    {module: 'forgejo.org/extension-sdk', path: sdk, source_revision: sdkRevision, clean_scope: sdkRelative},
  ],
  dependency_inputs: inputs,
  dependency_installation_performed: false,
  tool_paths: {bun: process.execPath, go: Bun.which('go')},
  go_build: {cgo_enabled: false, mod: 'readonly', trimpath: true, linker},
  outputs: await Promise.all([driver, phases, probe].map(artifact)),
  appliance_access: false,
  provider_calls: 0,
};
await writeFile(path.join(out, 'build.json'), JSON.stringify(receipt, null, 2) + '\n', {flag: 'wx', mode: 0o600});
console.log(JSON.stringify(receipt));
