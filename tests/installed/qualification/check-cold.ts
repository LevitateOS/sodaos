// Actual local Docker storage integration. Import a tiny executable fixture,
// reuse retained App storage, and exercise real stopped copy/isolation guards.
// This is not a Soda image build or a complete installed-system qualification.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {mkdir, readFile, writeFile, realpath} from 'node:fs/promises';
import path from 'node:path';

process.umask(0o077);
assert.equal(process.getuid?.(), 0);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--operator', '--app-proof', '--out']
);
assert.equal(args.length, 6);
const operator = args[1]!,
  proof = args[3]!,
  out = args[5]!;
assert.equal(path.dirname(out), '/home');
assert.equal(path.resolve(out), out);
assert.equal(await realpath(operator), operator);
await mkdir(out, {mode: 0o700});
const hash = (bytes: Uint8Array) => createHash('sha256').update(bytes).digest('hex');
async function save(relative: string, value: unknown) {
  await writeFile(path.join(out, relative), JSON.stringify(value, null, 2) + '\n', {flag: 'wx', mode: 0o600});
}
function command(executable: string, argv: string[], input?: Uint8Array, allowed = [0]) {
  const result = spawnSync(executable, argv, {
    input,
    env: {PATH: '/usr/sbin:/usr/bin:/sbin:/bin', HOME: out, TMPDIR: out, LC_ALL: 'C'},
    timeout: 120000,
    maxBuffer: 8 << 20,
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  assert(
    allowed.includes(result.status!),
    'local_cold_command_unconfirmed_' + path.basename(executable) + ': ' + result.stderr.toString().slice(0, 256)
  );
  return result;
}
const docker = (argv: string[]) =>
  command('/usr/bin/docker', ['--host', 'unix:///var/run/docker.sock', ...argv])
    .stdout.toString()
    .trim();
const project = 'sodaj19' + Date.now(),
  rootfs = path.join(out, 'rootfs');
await mkdir(rootfs, {mode: 0o700});
const executables = ['/usr/bin/true', '/usr/bin/sleep'];
const files = new Set(executables);
for (const file of executables) {
  const libraries = command('/usr/bin/ldd', [file]).stdout.toString();
  for (const match of libraries.matchAll(/(?:=>\s+|^\s*)(\/[^\s]+)/gm)) files.add(match[1]!);
}
for (const file of files) {
  await mkdir(path.join(rootfs, path.dirname(file)), {recursive: true, mode: 0o755});
  command('/usr/bin/cp', ['--dereference', '--preserve=mode,timestamps', '--', file, path.join(rootfs, file)]);
}
const archive = command('/usr/bin/tar', ['--create', '--file=-', '--directory', rootfs, '.']).stdout;
const image = command(
  '/usr/bin/docker',
  ['--host', 'unix:///var/run/docker.sock', 'image', 'import', '-', project + ':fixture'],
  archive
)
  .stdout.toString()
  .trim();
assert(/^sha256:[a-f0-9]{64}$/.test(image));
const settings = path.join(out, 'settings');
await mkdir(settings, {mode: 0o700});
for (const [name, content] of Object.entries({
  'compose.yaml': 'name: ' + project + '\nservices:\n  soda:\n    image: ' + image + '\n',
  '.env': 'SODA_IMAGE=' + image + '\n',
  'identity.json': JSON.stringify({format: 1, image, project}),
  'docker.json': JSON.stringify({format: 1, scope: 'local executable storage fixture', image}),
}))
  await writeFile(path.join(settings, name), content, {flag: 'wx', mode: 0o600});
const volumes = [];
for (const [name, destination] of [
  ['config', '/etc'],
  ['home', '/home'],
  ['state', '/var'],
]) {
  const volume = docker([
    'volume',
    'create',
    '--label',
    'com.docker.compose.project=' + project,
    '--label',
    'com.docker.compose.volume=' + name,
    project + '_' + name,
  ]);
  const native = JSON.parse(docker(['volume', 'inspect', volume]))[0];
  assert(native.Mountpoint.startsWith('/home/'));
  volumes.push({name: volume, destination: destination!, path: native.Mountpoint as string});
}
const home = volumes.find((v) => v.destination === '/home')!.path,
  state = volumes.find((v) => v.destination === '/var')!.path;
// Retained native App evidence is copied only while its actual lock is free.
command('/usr/bin/flock', [
  '--nonblock',
  path.join(proof, 'data', '.writer-lock'),
  '/usr/bin/cp',
  '--archive',
  '--reflink=auto',
  '--',
  path.join(proof, 'data'),
  path.join(home, 'application'),
]);
const app = JSON.parse(await readFile(path.join(proof, 'result.json'), 'utf8'));
assert(app.passed && app.retirements.every((r: any) => r.writer_lock_available));
await writeFile(path.join(home, 'member-witness'), 'Local cold fixture retained member data\n', {
  flag: 'wx',
  mode: 0o600,
});
await writeFile(path.join(state, 'record-witness.json'), JSON.stringify(app.record), {flag: 'wx', mode: 0o600});
// Exercise real hardlink and symlink preservation too.
command('/usr/bin/ln', [path.join(home, 'member-witness'), path.join(home, 'member-witness-hardlink')]);
command('/usr/bin/ln', ['--symbolic', 'member-witness', path.join(home, 'member-witness-link')]);
const mountArgs = volumes.flatMap((v) => ['--mount', 'type=volume,source=' + v.name + ',destination=' + v.destination]);
const container = docker([
  'container',
  'create',
  '--label',
  'com.docker.compose.project=' + project,
  '--label',
  'com.docker.compose.service=soda',
  ...mountArgs,
  '--entrypoint',
  '/usr/bin/true',
  image,
]);
await save('fixture.json', {
  format: 1,
  image,
  container,
  project,
  volumes,
  app_record: app.record,
  installed_system_qualification: false,
});
docker(['container', 'start', '--attach', container]);
const cold = (selectedOut: string) => [
  operator,
  'journey',
  'cold',
  'capture',
  '--container',
  container,
  '--project',
  project,
  '--image',
  image,
  '--settings',
  settings,
  '--out',
  selectedOut,
];
const executeCold = (argv: string[], allowed = [0]) => command(argv[0]!, argv.slice(1), undefined, allowed);
// A second actual running container holding one volume must fence capture.
const holder = docker([
  'container',
  'run',
  '--detach',
  '--network',
  'none',
  '--mount',
  'type=volume,source=' + volumes[1]!.name + ',destination=/home',
  '--entrypoint',
  '/usr/bin/sleep',
  image,
  '60',
]);
try {
  const denied = executeCold(cold(path.join(out, 'refused-active-writer')), [1]);
  assert(denied.stderr.toString().includes('another container writes the cold set'));
  await save('active-writer-refusal.json', {format: 1, exit_code: denied.status, observed: true, holder});
} finally {
  docker(['container', 'stop', '--time', '1', holder]);
}
const capture = JSON.parse(executeCold(cold(path.join(out, 'cold'))).stdout.toString());
assert(capture.complete && capture.writers_started === false);
await save('capture.public.json', capture);
const restored = JSON.parse(
  executeCold([
    operator,
    'journey',
    'cold',
    'restore',
    '--source',
    capture.directory,
    '--out',
    path.join(out, 'restored'),
  ]).stdout.toString()
);
assert(
  restored.complete && restored.network === 'none' && restored.state === 'created' && restored.container !== container
);
const before = JSON.parse(docker(['container', 'inspect', restored.container]))[0];
assert.equal(before.State.Running, false);
assert.equal(before.HostConfig.NetworkMode, 'none');
assert.equal(before.HostConfig.UsernsMode, 'host');
assert(before.Mounts.every((m: any) => m.Type === 'bind' && m.Source.startsWith(path.join(out, 'restored') + '/')));
assert.deepEqual(
  JSON.parse(await readFile(path.join(out, 'restored', 'var', 'record-witness.json'), 'utf8')),
  app.record
);
assert.equal(
  await readFile(path.join(out, 'restored', 'home', 'member-witness'), 'utf8'),
  'Local cold fixture retained member data\n'
);
docker(['container', 'start', '--attach', restored.container]);
const after = JSON.parse(docker(['container', 'inspect', restored.container]))[0];
assert.equal(after.State.Status, 'exited');
assert.equal(after.State.ExitCode, 0);
assert.equal(JSON.parse(docker(['container', 'inspect', container]))[0].State.Status, 'exited');
const result = {
  format: 1,
  passed: true,
  check: 'actual-local-docker-cold-storage',
  operator_sha256: hash(await readFile(operator)),
  capture,
  restore: restored,
  active_writer_refused: true,
  fresh_storage: true,
  retained_numeric_ownership_namespace: before.HostConfig.UsernsMode,
  same_app_storage_and_record_witness: true,
  restored_fixture_exit: after.State.ExitCode,
  source_still_stopped: true,
  scope: 'Tiny local executable fixture and retained App storage; no Soda boot or complete installed recovery claim',
  provider_calls: 0,
  guest_mutations: 0,
  image_pulls: 0,
};
await save('result.json', result);
console.log(JSON.stringify({passed: true, result: path.join(out, 'result.json')}));
