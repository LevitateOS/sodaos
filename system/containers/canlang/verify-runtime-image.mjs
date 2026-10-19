#!/usr/bin/env node
/** Local qualification of the two production recipes, using Rust-admitted frozen assets. */
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { randomBytes, createHash } from 'node:crypto';
import { copyFile, mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
const option = name => { const i = process.argv.indexOf(name); assert.ok(i >= 0 && process.argv[i + 1], `${name} is required`); return resolve(process.argv[i + 1]); };
const assets = option('--assets'), bundleFile = option('--bundle'), releaseFile = option('--release');
const state = option('--state'), cache = option('--podman-state');
const imageOption = name => { const i = process.argv.indexOf(name); return i < 0 ? undefined : process.argv[i + 1]; };
const existingRuntime = imageOption('--runtime-image'), existingApplication = imageOption('--application-image');
assert.equal(existingRuntime === undefined, existingApplication === undefined, 'both existing image references are required together');
assert.ok(state.startsWith('/home/') && cache.startsWith('/home/'));
const repo = resolve(fileURLToPath(new URL('../../../', import.meta.url)));
const canlang = join(dirname(repo), 'canlang');
const input = JSON.parse(await readFile(bundleFile, 'utf8'));
assert.match(input.storageFingerprint, /^[0-9a-f]{64}$/);
const release = JSON.parse(await readFile(releaseFile, 'utf8'));
assert.equal(input.descriptor.identity.appName, 'SodaRuntime', 'this finite witness uses the compiled SodaRuntime fixture');
assert.match(release.sha256, /^[0-9a-f]{64}$/);
const tools = JSON.parse(await readFile(join(assets, 'project-tools/canlang/tools.json'), 'utf8'));
assert.equal(tools.sourceRevision, release.sourceRevision);
assert.deepEqual(tools.versions, release.versions);
const origin = input.environment.vars.CAN_AUTH_ORIGIN;
assert.equal(new URL(origin).protocol, 'https:');
assert.equal(new URL(origin).hostname, '127.0.0.1', 'local image witness uses a task-owned namespace loopback HTTPS gateway');
assert.ok(Number(new URL(origin).port) > 0);
await mkdir(state, { recursive: true, mode: 0o700 });
const proof = join(state, 'proof'), data = join(state, 'data');
await mkdir(proof, { mode: 0o700 }); await mkdir(data, { mode: 0o700 });
await copyFile(bundleFile, join(state, 'bundle.json'));
await writeFile(join(state, '.containerignore'), '**\n!bundle.json\n');
const flags = ['--root', join(cache, 'podman/root'), '--runroot', join(cache, 'podman/run'), '--tmpdir', join(cache, 'podman/tmp')];
const env = { ...process.env, TMPDIR: join(cache, 'tmp') };
const commands = [];
const podman = (args, options = {}) => {
  commands.push(['podman', ...flags, ...args]);
  return execFileSync('podman', [...flags, ...args], { env, encoding: 'utf8', stdio: 'pipe', maxBuffer: 16 * 1024 * 1024, ...options });
};
const inspect = (kind, name) => JSON.parse(podman([kind, 'inspect', name]))[0];
const base = 'docker.io/rockylinux/rockylinux@sha256:827d37bc128288ccf160ee318bb3cb92d591164cb217e92f8bc61e3982ae1834';
const runtimeImage = existingRuntime ?? `localhost/canlang-runtime-proof:${process.pid}`, appImage = existingApplication ?? `localhost/canlang-application-proof:${process.pid}`;
if (!existingRuntime) await writeFile(join(state, 'runtime-build.log'), podman(['build', '--pull=never', '--network=none', '-f', join(repo, 'system/containers/canlang/Containerfile'),
  '--build-arg', `BASE_IMAGE=${base}`, '--build-arg', 'ARTIFACT_DIR=.', '--tag', runtimeImage, assets]));
const runtime = inspect('image', runtimeImage);
const runtimeDigest = `sha256:${runtime.Id.replace(/^sha256:/, '')}`;
if (!existingApplication) await writeFile(join(state, 'application-build.log'), podman(['build', '--pull=never', '--network=none', '-f', join(repo, 'system/containers/canlang/Application.Containerfile'),
  '--build-arg', `RUNTIME_IMAGE=${runtimeImage}`, '--build-arg', 'APP_BUNDLE=bundle.json',
  '--build-arg', `APP_NAME=${input.descriptor.identity.appName}`, '--build-arg', `ARTIFACT_DIGEST=${input.descriptor.identity.artifactDigest}`,
  '--build-arg', `BUNDLE_DIGEST=${input.bundle.mixedSha256}`, '--build-arg', `TOOLS_DIGEST=${release.sha256}`,
  '--build-arg', `STORAGE_DIGEST=${input.storageFingerprint}`,
  '--build-arg', `RUNTIME_DIGEST=${runtimeDigest}`, '--build-arg', `APP_ORIGIN=${input.environment.vars.CAN_AUTH_ORIGIN}`,
  '--tag', appImage, state]));
const application = inspect('image', appImage);
assert.equal(application.Config.User, '10000:10000');
for (const [key, value] of Object.entries({ 'io.canlang.app': input.descriptor.identity.appName,
  'io.canlang.artifact': input.descriptor.identity.artifactDigest, 'io.canlang.bundle': input.bundle.mixedSha256,
  'io.canlang.storage': input.storageFingerprint,
  'io.canlang.tools': release.sha256, 'io.soda.canlang.runtime': runtimeDigest,
  'io.canlang.origin': input.environment.vars.CAN_AUTH_ORIGIN })) assert.equal(application.Config.Labels[key], value);
const confinement = ['--network', 'none', '--userns', 'keep-id:uid=10000,gid=10000', '--read-only', '--read-only-tmpfs=false', '--cap-drop', 'ALL',
  '--security-opt', 'no-new-privileges', '--security-opt', 'label=disable', '--memory', '512m', '--cpus', '2', '--pids-limit', '256'];
const mounts = ['--volume', `${data}:/home/app-state:rw`, '--volume', `${proof}:/proof:ro`];
const audit = podman(['run', '--rm', ...confinement, '--entrypoint', '/bin/sh', appImage, '-c',
  'test "$(id -u)" = 10000 && test ! -e /opt/canlang/compiler && test ! -e /opt/canlang/libexec/bun && test ! -e /opt/canlang/bin/can && test ! -e /opt/canlang/bin/can-dev && test ! -e /opt/canlang/bin/can-platform && for tool in ssh scp sftp sshd systemctl sudo podman docker gcc cargo rustc bun; do if command -v "$tool"; then exit 1; fi; done && /opt/canlang/bin/node --version']);
assert.equal(audit.trim(), 'v24.21.0');
const password = `ImageProof!${randomBytes(24).toString('base64url')}`;
const credentials = { email: 'owner@image.can.test', password, timezone: 'UTC' };
await writeFile(join(proof, 'credentials.json'), JSON.stringify(credentials), { mode: 0o600 });
const memberPassword = `ImageMember!${randomBytes(24).toString('base64url')}`;
const memberCredentials = { email: 'member@image.can.test', password: memberPassword };
await writeFile(join(proof, 'credentials-member.json'), JSON.stringify(memberCredentials), { mode: 0o600 });
await writeFile(join(proof, 'actors.json'), JSON.stringify([{ label: 'Ava', email: credentials.email, password },
  { label: 'Ben', email: memberCredentials.email, password: memberPassword }]), { mode: 0o600 });
const bootstrap = JSON.parse(podman(['run', '--rm', ...confinement, ...mounts, '--entrypoint', '/opt/canlang/bin/can-soda', appImage,
  'bootstrap', '--bundle', '/opt/application/bundle.json', '--data', '/home/app-state', '--credentials', '/proof/credentials.json']).trim());
assert.equal(bootstrap.ok, true); assert.match(bootstrap.teamId, /^[0-9a-f-]{36}$/i);
const member = JSON.parse(podman(['run', '--rm', ...confinement, ...mounts, '--entrypoint', '/opt/canlang/bin/can-soda', appImage,
  'member', '--bundle', '/opt/application/bundle.json', '--data', '/home/app-state', '--owner', bootstrap.teamId,
  '--credentials', '/proof/credentials-member.json']).trim());
assert.equal(member.ok, true); assert.equal(member.teamId, bootstrap.teamId);
// Preserve only genuine HTTP cookies and MCP grants across disposable test clients.
// This task-local helper copy keeps the image and Canlang producers unchanged.
let clientSource = await readFile(join(canlang, 'scripts/verify-soda-runtime.mjs'), 'utf8');
const replaceClient = (before, after) => { assert.ok(clientSource.includes(before)); clientSource = clientSource.replace(before, after); };
replaceClient('  const cookies = new Map();', `  const sessionPath = '/home/app-state/image-client-' + label + '.json';
  let retainedSession;
  try { retainedSession = JSON.parse(await readFile(sessionPath, 'utf8')); } catch (error) { if (error.code !== 'ENOENT') throw error; }
  const cookies = new Map(retainedSession?.cookies ?? []);`);
replaceClient('  const openUrl = issueOpenUrl ? await issueOpenUrl() : bootstrapUrl;', '  if (!retainedSession) {\n  const openUrl = issueOpenUrl ? await issueOpenUrl() : bootstrapUrl;');
replaceClient('  const csrf = await deriveCsrfToken', '  }\n  const csrf = await deriveCsrfToken');
replaceClient('  const grantResponse = await request', '  let token = retainedSession?.token;\n  if (!token) {\n  const grantResponse = await request');
replaceClient('  const { token } = await grantResponse.json();', `  ({ token } = await grantResponse.json());
  }
  await writeFile(sessionPath, JSON.stringify({cookies:[...cookies],token}), {mode:0o600});`);
await writeFile(join(proof, 'client.mjs'), clientSource);
execFileSync('openssl', ['req', '-x509', '-newkey', 'rsa:2048', '-sha256', '-days', '1', '-nodes',
  '-subj', '/CN=127.0.0.1', '-addext', 'subjectAltName=IP:127.0.0.1',
  '-keyout', join(proof, 'test-key.pem'), '-out', join(proof, 'test-cert.pem')], { stdio: 'ignore' });
await writeFile(join(proof, 'phase.mjs'), `
import assert from 'node:assert/strict';
import {readFile,writeFile} from 'node:fs/promises';
import {openSodaHttpClient} from './client.mjs';
import {createServer} from 'node:https';
import {request} from 'node:http';
import {createRequire} from 'node:module';
import {pathToFileURL} from 'node:url';
const origin=${JSON.stringify(origin)};
const require=createRequire('/opt/canlang/lib/package.json');
const {Agent,setGlobalDispatcher}=await import(pathToFileURL(require.resolve('undici')).href);
const cert=await readFile('/proof/test-cert.pem');
const dispatcher=new Agent({connect:{ca:cert}});setGlobalDispatcher(dispatcher);
const gateway=createServer({cert,key:await readFile('/proof/test-key.pem')},(req,res)=>{
 const upstream=request({hostname:'127.0.0.1',port:8080,path:req.url,method:req.method,headers:req.headers},response=>{res.writeHead(response.statusCode,response.headers);response.pipe(res);});
 upstream.on('error',()=>{res.writeHead(502);res.end();});req.pipe(upstream);
});
await new Promise((resolve,reject)=>{gateway.once('error',reject);gateway.listen(Number(new URL(origin).port),'127.0.0.1',resolve);});
try {
const actors=JSON.parse(await readFile('/proof/actors.json','utf8'));
const client=await openSodaHttpClient({toolRoot:'/opt/canlang',origin,actors},'Ava');
const member=await openSodaHttpClient({toolRoot:'/opt/canlang',origin,actors},'Ben');
for(const path of ['/_can_dev/preview/bootstrap','/_can_dev/status','/__soda_ready','/__soda_wake'])assert.equal((await client.request(path)).status,404,path);
const page=await client.request('/');assert.equal(page.status,200);assert.ok((await page.text()).includes('Soda runtime integration'));
if(process.argv[2]==='write'){
 const created=await client.mutate('Entry.create',{label:'Image retained state'});
 assert.equal(created.records.length,1);const entry=created.records[0];assert.ok(entry);
 await client.upload(entry);
 assert.equal((await member.request('/files/read/SodaRuntime.Entry/'+entry.id+'/attachment')).status,404);
 const current=(await client.entries()).find(record=>record.id===entry.id);assert.ok(current);
 const due=Date.now()+30000;
 await client.mutate('arm',{entry:{id:current.id,version:String(current.version)},due:new Date(due).toISOString()});
 await writeFile('/home/app-state/image-proof.json',JSON.stringify({entry:entry.id,due}),{mode:0o600});
 console.log(JSON.stringify({written:true,due}));
}else{
 const saved=JSON.parse(await readFile('/home/app-state/image-proof.json','utf8'));
 let entry;const until=Date.now()+30000;
 while(Date.now()<until){entry=(await client.entries()).find(record=>record.id===saved.entry);if(entry?.data.ticks==='1')break;await new Promise(resolve=>setTimeout(resolve,250));}
 assert.equal(entry.data.label,'Image retained state');assert.equal(entry.data.ticks,'1');
 const file=await client.request('/files/read/SodaRuntime.Entry/'+saved.entry+'/attachment');assert.equal(file.status,200);
 assert.deepEqual(Buffer.from(await file.arrayBuffer()),Buffer.from('%PDF-1.4\\ntrailer\\n'));
 assert.equal((await member.request('/files/read/SodaRuntime.Entry/'+saved.entry+'/attachment')).status,404);
 console.log(JSON.stringify({retained:true,scheduledOnce:true,file:true,devRoutesAbsent:true}));
}
} finally {await dispatcher.close();await new Promise((resolve,reject)=>{gateway.close(error=>error?reject(error):resolve());gateway.closeIdleConnections();});}
`);
const owned = [];
const start = async index => {
  const name = `canlang-image-proof-${process.pid}-${index}`; owned.push(name);
  podman(['run', '--detach', '--name', name, ...confinement, ...mounts, appImage, '--owner', bootstrap.teamId]);
  const deadline = Date.now() + 60000;
  while (Date.now() < deadline) {
    const details = inspect('container', name);
    if (!details.State.Running) throw new Error(`production image exited: ${podman(['logs', name])}`);
    try {
      const status = podman(['exec', name, '/opt/canlang/bin/node', '--input-type=module', '-e',
        'console.log((await fetch("http://127.0.0.1:8080/auth/login")).status)']).trim();
      if (status === '200') return name;
    } catch {}
    await new Promise(resolve => setTimeout(resolve, 250));
  }
  throw new Error('production image readiness timed out');
};
const stop = async (name, abrupt = false) => {
  podman(abrupt ? ['kill', '--signal', 'KILL', name] : ['stop', '--time', '20', name]);
  if (abrupt) podman(['wait', name]); const details = inspect('container', name);
  assert.equal(details.State.ExitCode, abrupt ? 137 : 0, `shutdown failed: ${podman(['logs', name])}`);
  await writeFile(join(state, `${name}.log`), podman(['logs', name]));
  podman(['rm', name]); owned.splice(owned.indexOf(name), 1);
};
try {
  let name = await start(0);
  const written = JSON.parse(podman(['exec', name, '/opt/canlang/bin/node', '/proof/phase.mjs', 'write']).trim());
  assert.equal(written.written, true); await stop(name);
  await new Promise(resolve => setTimeout(resolve, Math.max(0, written.due + 250 - Date.now())));
  name = await start(1);
  const retained = JSON.parse(podman(['exec', name, '/opt/canlang/bin/node', '/proof/phase.mjs', 'read']).trim());
  assert.equal(retained.retained, true); await stop(name, true);
  name = await start(2);
  assert.equal(JSON.parse(podman(['exec', name, '/opt/canlang/bin/node', '/proof/phase.mjs', 'read']).trim()).retained, true);
  await stop(name);
  assert.throws(() => podman(['run', '--rm', ...confinement, appImage, '--owner', bootstrap.teamId]), error =>
    error.status !== 0 && /can-soda:|data|mount|ENOENT/i.test(String(error.stderr)));
  const results = { scope: 'Local native production recipes with a task-owned HTTPS test gateway; no appliance or production ingress qualification',
    runtimeImage: runtimeDigest, applicationImage: `sha256:${application.Id.replace(/^sha256:/, '')}`, baseImage: base,
    toolsArchive: release.sha256, artifact: input.descriptor.identity.artifactDigest, bundle: input.bundle.mixedSha256,
    bundleFileSha256: createHash('sha256').update(await readFile(bundleFile)).digest('hex'), labels: application.Config.Labels,
    unprivileged: true, offlineBootstrapAndHttp: true, noCompilerBunOrPrivilegedDaemons: true, noDevelopmentRoutes: true, gracefulShutdown: true,
    retainedReplacement: true, filePersistence: true, fileFieldDenial: true, ordinaryMemberViaCli: true, retainedSessionsAndGrants: true, abruptReplacement: true, missedScheduleRecovery: true, requiredMountRejected: true };
  await writeFile(join(state, 'commands.json'), JSON.stringify(commands, null, 2));
  await writeFile(join(state, 'results.json'), JSON.stringify(results, null, 2));
  console.log(JSON.stringify(results, null, 2));
} finally {
  await writeFile(join(state, 'commands.json'), JSON.stringify(commands, null, 2));
  for (const name of owned) { try { podman(['stop', '--time', '20', name]); podman(['rm', name]); } catch {} }
}
