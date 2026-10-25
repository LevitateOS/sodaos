import assert from 'node:assert/strict';
import {check, sha, SHA} from './support';
import {nativeReadScript, readDocker} from './docker-native';
import {
  validateDockerHostPaths,
  validateWitness,
  type DockerBinding,
  type DockerIO,
  type Witness,
} from './docker-binding';

// Compose's normal rootful Engine actions run ONLY on the exact SSH host. A
// durable O_EXCL request precedes stop; uncertain replies are read back, never
// automatically resent. Paths come only from the pinned host installation.
const RECREATE = String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
const need=(x,r)=>{if(!x)throw Error(r)};const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const run=(p,a,input,timeout=30000)=>cp.execFileSync(p,a,{input,encoding:'utf8',timeout,maxBuffer:4<<20,stdio:[input===undefined?'ignore':'pipe','pipe','pipe']}).trim();
const d=a=>run('/usr/bin/docker',['--host','unix:///var/run/docker.sock',...a]);
const observe=()=>JSON.parse(run('/usr/bin/node',['-'],READER,210000));
const writeNew=(p,v)=>{const fd=fs.openSync(p,fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_EXCL|fs.constants.O_NOFOLLOW,384);try{fs.writeFileSync(fd,JSON.stringify(v,null,2)+'\n');fs.fsyncSync(fd);}finally{fs.closeSync(fd)}const d=fs.openSync(HOST.install_directory+'/.qualification-persistence',fs.constants.O_RDONLY|fs.constants.O_DIRECTORY);try{fs.fsyncSync(d)}finally{fs.closeSync(d)}};
try{
 need(process.getuid()===0,'outer_root');
 const before=observe();need(before.outer_id===EXPECTED.outer&&before.inner.protected_sha256===EXPECTED.protected&&hash(JSON.stringify(before.volumes))===EXPECTED.volumes,'fresh_exact_before_snapshot');
 const directory=HOST.install_directory+'/.qualification-persistence';if(!fs.existsSync(directory)){fs.mkdirSync(directory,{mode:448});const p=fs.openSync(HOST.install_directory,fs.constants.O_RDONLY|fs.constants.O_DIRECTORY);try{fs.fsyncSync(p)}finally{fs.closeSync(p)}}
 const st=fs.lstatSync(directory);need(st.isDirectory()&&!st.isSymbolicLink()&&st.uid===0&&(st.mode&511)===448,'protected_action_directory');
 need(!fs.existsSync(directory+'/attempt.json')&&!fs.existsSync(directory+'/result.json'),'one_recreation_attempt');
 const common=['--host','unix:///var/run/docker.sock','compose','--project-name','sodaos','--project-directory',HOST.install_directory,'--env-file',HOST.install_directory+'/.env','-f',HOST.install_directory+'/compose.yaml'];
 need(run('/usr/bin/docker',[...common,'config','--images'])===before.installer_execution.image,'fixed_compose_image');
 need(run('/usr/bin/docker',[...common,'config','--services'])==='soda','fixed_compose_service');
 const at=new Date().toISOString();writeNew(directory+'/attempt.json',{format:1,action:'same-image-force-recreate',target:before.target,source:before.inner.source,outer_before:before.outer_id,registry_manifest:before.registry_manifest,registry_config:before.registry_config,protected_before_sha256:before.inner.protected_sha256,volumes_sha256:EXPECTED.volumes,started_at:at,automatic_retry:false});
 run('/usr/bin/docker',[...common,'stop'],undefined,300000);
 const stopped=JSON.parse(d(['container','inspect','--format','{"id":{{json .Id}},"status":{{json .State.Status}},"exit":{{json .State.ExitCode}},"image":{{json .Image}}}','sodaos-soda-1']));need(stopped.id===before.outer_id&&stopped.status==='exited'&&[0,130].includes(stopped.exit)&&stopped.image===before.engine_image,'owned_outer_clean_stop');
 writeNew(directory+'/stopped.json',stopped);
 run('/usr/bin/docker',[...common,'up','-d','--force-recreate','--pull','never','--no-build','--wait','--wait-timeout','1200'],undefined,1260000);
 const after=observe();need(after.outer_id!==before.outer_id&&after.inner.project.container===before.inner.project.container&&after.inner.protected_sha256===before.inner.protected_sha256&&JSON.stringify(after.volumes)===JSON.stringify(before.volumes),'actual_retained_native_state');
 const result={format:1,action:'same-image-force-recreate',target:before.target,source:before.inner.source,started_at:at,finished_at:new Date().toISOString(),exit_code:0,outer_before:before.outer_id,outer_after:after.outer_id,project_before:before.inner.project.container,project_after:after.inner.project.container,protected_before_sha256:before.inner.protected_sha256,protected_after_sha256:after.inner.protected_sha256,volumes_sha256:EXPECTED.volumes,registry_manifest:after.registry_manifest,registry_config:after.registry_config};
 writeNew(directory+'/result.json',result);console.log(JSON.stringify({result,after}));
}catch(e){console.error(JSON.stringify({reason:e instanceof Error&&/^[a-z0-9_]+$/.test(e.message)?e.message:'docker_recreation_unavailable',automatic_retry:false}));process.exitCode=1;}
`;

// Existing networkGuard delegates only exact armed normal-owner requests here.
// There is no permissive lifecycle prefix or caller-provided mutation body.
export class PersistencePermits {
  private armed: {name: string; path: string; body: unknown} | undefined;
  private used = new Set<string>();
  constructor(private save: DockerIO['save']) {}
  arm(name: string, path: string, body: unknown) {
    check(name === 'end-terminal' || /^stop-project-p[a-f0-9]{24}$/.test(name), 'fixed_lifecycle_action');
    check(!this.armed && !this.used.has(name), 'one_owned_lifecycle_request');
    this.armed = {name, path, body};
  }
  async admit(request: any) {
    const a = this.armed;
    if (!a || request.method() !== 'POST' || new URL(request.url()).pathname !== a.path) return false;
    assert.deepEqual(request.postDataJSON(), a.body);
    this.armed = undefined;
    this.used.add(a.name);
    await this.save('persistence/' + a.name + '.request.json', {
      method: 'POST',
      path: a.path,
      body: a.body,
      automatic_retry: false,
    });
    return true;
  }
}

// Uses the normal bound Soda transport; raw-run blockers and this permit must
// already be active on the context before login. Every effect is once-only.
export async function stopOwnedProject(
  page: any,
  apiBase: string,
  w: Witness,
  b: DockerBinding,
  io: DockerIO,
  permits: PersistencePermits,
  configuredFactoryProject: string
) {
  validateWitness(w);
  check(/^\/[^?#]*$/.test(apiBase) && !apiBase.endsWith('/'), 'canonical_transport_base');
  check(/^p[a-f0-9]{24}$/.test(configuredFactoryProject), 'configured_factory_project');
  const before = await readDocker(io, b, 'idle', w);
  assert.deepEqual(
    before.inner.facts.projects.map((p: any) => p.id).sort(),
    [...new Set([w.project, configuredFactoryProject])].sort(),
    'exact_factory_and_browser_project_scope'
  );
  check(
    before.inner.project.running === true &&
      before.inner.project.unit === 'active' &&
      before.inner.project.terminals.length === 1 &&
      before.inner.project.terminals[0] === w.session,
    'only_exact_owned_terminal'
  );
  await io.save('persistence/idle-before-stop.json', before);
  const request = async (kind: string, route: string, body: unknown) => {
    permits.arm(kind, apiBase + '/' + route, body);
    const reply = await page.evaluate(
      async ({route, body}: any) => {
        const response = await (document.querySelector('soda-spaces') as any).binding.transport.request(route, {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify(body),
          signal: AbortSignal.timeout(240000),
        });
        // Lifecycle replies contain no Factory provider output. Retain only its
        // HTTP acknowledgment; actual completion is independently native-read.
        return {status: response.status};
      },
      {route, body}
    );
    await io.save('persistence/' + kind + '.receipt.json', reply);
    check(reply.status === 200, 'normal_lifecycle_acknowledgment');
  };
  await request('end-terminal', 'environments/' + w.project + '/terminal-sessions/' + w.session, {action: 'end'});
  const ended = await readDocker(io, b, 'idle', w);
  check(
    ended.inner.project.container === before.inner.project.container &&
      ended.inner.project.running &&
      ended.inner.project.terminals.length === 0,
    'owned_terminal_native_retirement'
  );
  await io.save('persistence/terminal-ended-native.json', ended);
  for (const project of new Set([w.project, configuredFactoryProject])) {
    const current = await readDocker(io, b, 'idle', w),
      native = current.inner.projects.find((p: any) => p.project === project);
    check(native && native.terminals.length === 0, 'no_foreign_or_owned_ttys_before_stop');
    if (native.running)
      await request('stop-project-' + project, 'environments/' + project + '/lifecycle', {
        action: 'stop',
        confirm_stop: true,
      });
  }
  const stopped = await readDocker(io, b, 'snapshot', w);
  check(stopped.inner.project.container === before.inner.project.container, 'same_stopped_project_incarnation');
  await io.save('persistence/project-stopped-native.json', stopped);
  return stopped;
}

export function recreateScript(b: DockerBinding, w: Witness, before: any) {
  validateDockerHostPaths(b.host);
  validateWitness(w);
  check(
    /^[a-f0-9]{64}$/.test(before.outer_id) &&
      SHA.test(before.inner.protected_sha256) &&
      before.inner.project.project === w.project,
    'native_before_recreation'
  );
  const expected = {
    outer: before.outer_id,
    protected: before.inner.protected_sha256,
    volumes: sha(JSON.stringify(before.volumes)),
  };
  return (
    'const HOST=' +
    JSON.stringify(b.host) +
    ';\nconst EXPECTED=' +
    JSON.stringify(expected) +
    ';\nconst READER=' +
    JSON.stringify(nativeReadScript(b, 'snapshot', w)) +
    ';\n' +
    RECREATE
  );
}

export async function exercisePersistence(b: DockerBinding, w: Witness, installationNative: any, io: DockerIO) {
  validateWitness(w);
  check(
    installationNative.target === b.target &&
      installationNative.registry_manifest === b.manifest &&
      installationNative.registry_config === b.config,
    'installation_phase_binding'
  );
  const before = await readDocker(io, b, 'snapshot', w);
  check(before.outer_id === installationNative.outer_id, 'original_installed_outer');
  await io.save('persistence/before.json', before);
  // Protected output save MUST be O_EXCL+fsync. A lost SSH response leaves the
  // attempted marker and remote journal; the caller may observe, never retry.
  await io.save('persistence/recreate.request.json', {
    action: 'same-image-force-recreate',
    target: b.target,
    source: b.source,
    outer_before: before.outer_id,
    project: w.project,
    project_before: before.inner.project.container,
    protected_before_sha256: before.inner.protected_sha256,
    volumes_sha256: sha(JSON.stringify(before.volumes)),
    automatic_retry: false,
  });
  const action = JSON.parse(await io.sshRead(recreateScript(b, w, before), 1800000));
  await io.save('persistence/action-result.json', action.result);
  const after = await readDocker(io, b, 'snapshot', w);
  check(after.outer_id === action.result.outer_after && after.outer_id !== before.outer_id, 'actual_new_outer');
  assert.deepEqual(after.volumes, before.volumes, 'all_three_original_volumes');
  assert.deepEqual(after.inner.protected, before.inner.protected, 'all_retained_state');
  check(
    after.inner.project.container === before.inner.project.container && after.inner.project.sha256 === w.sha256,
    'actual_project_file_retention'
  );
  await io.save('persistence/after.json', after);
  const native = {
    target: b.target,
    registry_manifest: b.manifest,
    registry_config: b.config,
    outer_before: before.outer_id,
    outer_after: after.outer_id,
    project_before: before.inner.project.container,
    project_after: after.inner.project.container,
    protected_before_sha256: before.inner.protected_sha256,
    protected_after_sha256: after.inner.protected_sha256,
  };
  await io.save('persistence/native.json', native);
  return {native, before, after};
}

// Read-only lost-reply recovery. This never invokes Compose or writes remote
// journals. A missing result remains incomplete even if an outer is running.
export function persistenceRecoveryScript(host: {install_directory: string; fixture_directory: string}) {
  validateDockerHostPaths(host);
  return (
    'const HOST=' +
    JSON.stringify(host) +
    ';\n' +
    String.raw`
const fs=require('node:fs');try{if(process.getuid()!==0)throw Error();const root=HOST.install_directory+'/.qualification-persistence',read=n=>{const p=root+'/'+n,s=fs.lstatSync(p);if(!s.isFile()||s.isSymbolicLink()||s.uid!==0||(s.mode&63)!==0)throw Error();return JSON.parse(fs.readFileSync(p,'utf8'))};console.log(JSON.stringify({attempt:read('attempt.json'),result:fs.existsSync(root+'/result.json')?read('result.json'):null,automatic_retry:false}));}catch{console.error('persistence_recovery_unavailable');process.exitCode=1;}
`
  );
}
