import {diagnoseClosure, assertResourceInventory, type ResourceScope} from '../closure';
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
    const terminal = name.match(/^end-terminal-([a-f0-9]{32})$/),
      project = name.match(/^stop-project-(p[a-f0-9]{24})$/),
      application = name.match(/^stop-application-(a[a-f0-9]{24})$/);
    if (terminal) {
      check(
        new RegExp('/environments/p[a-f0-9]{24}/terminal-sessions/' + terminal[1] + '$').test(path),
        'fixed_terminal_path'
      );
      assert.deepEqual(body, {action: 'end'});
    } else if (project) {
      check(path.endsWith('/environments/' + project[1] + '/lifecycle'), 'fixed_project_path');
      assert.deepEqual(body, {action: 'stop', confirm_stop: true});
    } else if (application) {
      const v = body as any;
      check(
        path.endsWith('/applications/' + application[1] + '/actions') &&
          Object.keys(v).sort().join(',') === 'action,confirm_stop,expected_container,revision' &&
          v.action === 'stop' &&
          v.confirm_stop === true &&
          Number.isSafeInteger(v.revision) &&
          v.revision > 0 &&
          /^[a-f0-9]{64}$/.test(v.expected_container),
        'fixed_application_stop'
      );
    } else check(false, 'fixed_lifecycle_action');
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

// Normal owner transport only. Diagnose the entire set before the first action;
// a permitted subset never hides an unowned writer or unknown native resource.
export async function retireOwnedResources(
  page: any,
  apiBase: string,
  w: Witness,
  b: DockerBinding,
  io: DockerIO,
  permits: PersistencePermits,
  scope: ResourceScope,
  actor: string
) {
  check(
    !scope.scenario.authority.held && scope.scenario.authority.scope_receipt_file !== null,
    'journey_execution_held'
  );
  validateWitness(w);
  check(/^\/[^?#]*$/.test(apiBase) && !apiBase.endsWith('/'), 'canonical_transport_base');
  const inspect = () => readDocker(io, b, 'idle', w, scope);
  const before = await inspect();
  const diagnose = (current: any) => {
    assertResourceInventory(scope, current.inner.resource_inventory);
    return diagnoseClosure(scope.scenario, scope.observation, current.inner);
  };
  const initial = diagnose(before);
  await io.save('persistence/retirement-plan.json', initial);
  check(initial.scope_ready, 'resource_retirement_scope_blocked');
  check(
    initial.retirement_actions.every((a) =>
      a.kind === 'terminal'
        ? a.actor_id === actor
        : a.kind !== 'project' || scope.observation.inventory.projects.find((p) => p.id === a.id)?.owner_id === actor
    ),
    'retirement_actor_outside_owner_session'
  );
  const sameIncarnations = (current: any) => {
    for (const kind of ['project_runtime', 'application_runtime']) {
      assert.deepEqual(
        current.inner[kind].map((r: any) => [r.id, r.container_id]).sort(),
        before.inner[kind].map((r: any) => [r.id, r.container_id]).sort(),
        'retained_native_incarnations'
      );
    }
    assert.deepEqual(current.inner.facts.runs, before.inner.facts.runs, 'retained_run_history');
    assert.deepEqual(current.inner.facts.policies, before.inner.facts.policies, 'retained_factory_policies');
  };
  const normalRead = (route: string) =>
    page.evaluate(async (route: string) => {
      const response = await (document.querySelector('soda-spaces') as any).binding.transport.request(route, {
        method: 'GET',
        signal: AbortSignal.timeout(20000),
      });
      if (response.status !== 200) throw Error('retirement_read_unconfirmed');
      const view = await response.json();
      return {application: view.application, observation: view.observation, restore: view.restore ?? null};
    }, route);
  const applicationStop = async (id: string, current: any) => {
    const expected = scope.observation.inventory.applications.find((a) => a.id === id)!,
      native = current.inner.application_runtime.find((a: any) => a.id === id),
      view = await normalRead('applications/' + id),
      app = view.application,
      observed = view.observation;
    check(
      app?.id === expected.id &&
        app.data === expected.data_id &&
        app.owner_team === expected.owner_team &&
        app.origin === expected.origin &&
        Number.isSafeInteger(app.revision) &&
        app.revision > 0 &&
        observed?.id === app.id &&
        observed.revision === app.revision &&
        observed.container === native.container_id &&
        view.restore === null,
      'current_application_stop_binding'
    );
    return {action: 'stop', revision: app.revision, expected_container: observed.container, confirm_stop: true};
  };
  // Resolve every App's existing normal-owner generation before ending any
  // terminal. Missing observations are held, never repaired by an Inspect POST.
  for (const action of initial.retirement_actions)
    if (action.kind === 'application') await applicationStop(action.id, before);
  const request = async (name: string, route: string, body: unknown) => {
    permits.arm(name, apiBase + '/' + route, body);
    let reply: {status: number; transport_uncertain?: boolean};
    try {
      reply = await page.evaluate(
        async ({route, body}: any) => {
          const response = await (document.querySelector('soda-spaces') as any).binding.transport.request(route, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(body),
            signal: AbortSignal.timeout(240000),
          });
          return {status: response.status};
        },
        {route, body}
      );
    } catch {
      reply = {status: 0, transport_uncertain: true};
    }
    await io.save('persistence/' + name + '.receipt.json', reply);
    check(reply.status === 200 || reply.status === 0 || reply.status >= 500, 'normal_lifecycle_command_denied');
    // The caller immediately reads this exact resource/native incarnation.
    // An uncertain HTTP reply is never another normal stop request.
    return reply;
  };
  for (const action of initial.retirement_actions) {
    const current = await inspect();
    sameIncarnations(current);
    const closure = diagnose(current);
    check(closure.scope_ready, 'resource_retirement_scope_changed');
    if (!closure.retirement_actions.some((a) => a.kind === action.kind && a.id === action.id)) continue;
    const name = action.kind === 'terminal' ? 'end-terminal-' + action.id : 'stop-' + action.kind + '-' + action.id;
    let route: string, body: unknown;
    if (action.kind === 'terminal') {
      route = 'environments/' + action.project_id + '/terminal-sessions/' + action.id;
      body = {action: 'end'};
    } else if (action.kind === 'project') {
      route = 'environments/' + action.id + '/lifecycle';
      body = {action: 'stop', confirm_stop: true};
    } else {
      route = 'applications/' + action.id + '/actions';
      body = await applicationStop(action.id, current);
    }
    const reply = await request(name, route, body);
    const after = await inspect();
    sameIncarnations(after);
    const confirmed = diagnose(after);
    check(
      confirmed.scope_ready &&
        confirmed.resources.some((r) => r.kind === action.kind && r.id === action.id && r.state === 'stopped'),
      'native_resource_retirement_unconfirmed'
    );
    await io.save('persistence/' + name + '.native.json', {
      http_status: reply.status,
      recovered_after_uncertain_reply: reply.status !== 200,
      closure: confirmed,
      project_runtime: after.inner.project_runtime,
      application_runtime: after.inner.application_runtime,
    });
  }
  const stopped = await readDocker(io, b, 'snapshot', w, scope);
  sameIncarnations({...stopped, inner: {...stopped.inner, facts: stopped.inner.protected.facts}});
  await io.save('persistence/resources-stopped-native.json', stopped);
  return stopped;
}

export function recreateScript(b: DockerBinding, w: Witness, before: any, scope: ResourceScope) {
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
    JSON.stringify(nativeReadScript(b, 'snapshot', w, scope)) +
    ';\n' +
    RECREATE
  );
}

export async function exercisePersistence(
  b: DockerBinding,
  w: Witness,
  installationNative: any,
  io: DockerIO,
  scope: ResourceScope
) {
  check(
    !scope.scenario.authority.held &&
      scope.scenario.authority.scope_receipt_file !== null &&
      scope.scenario.permitted_effects.same_image_recreation,
    'persistence_execution_held'
  );
  validateWitness(w);
  check(
    installationNative.target === b.target &&
      installationNative.registry_manifest === b.manifest &&
      installationNative.registry_config === b.config,
    'installation_phase_binding'
  );
  const before = await readDocker(io, b, 'snapshot', w, scope);
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
  let action: any;
  try {
    action = JSON.parse(await io.sshRead(recreateScript(b, w, before, scope), 1800000));
  } catch {
    const recovered = JSON.parse(await io.sshRead(persistenceRecoveryScript(b.host)));
    const attempted = recovered.attempt,
      result = recovered.result;
    check(
      attempted?.target === b.target &&
        attempted.source === b.source &&
        attempted.outer_before === before.outer_id &&
        attempted.protected_before_sha256 === before.inner.protected_sha256 &&
        result?.target === b.target &&
        result.source === b.source &&
        result.outer_before === before.outer_id &&
        result.registry_manifest === b.manifest &&
        result.registry_config === b.config &&
        result.volumes_sha256 === sha(JSON.stringify(before.volumes)) &&
        result.exit_code === 0,
      'persistence_exact_operation_unconfirmed'
    );
    action = {result};
    await io.save('persistence/recreation-recovered.json', {result, automatic_retry: false});
  }
  await io.save('persistence/action-result.json', action.result);
  const after = await readDocker(io, b, 'snapshot', w, scope);
  check(after.outer_id === action.result.outer_after && after.outer_id !== before.outer_id, 'actual_new_outer');
  assert.deepEqual(after.volumes, before.volumes, 'all_three_original_volumes');
  assert.deepEqual(after.inner.protected, before.inner.protected, 'all_retained_state');
  assert.deepEqual(
    after.inner.application_runtime,
    before.inner.application_runtime,
    'retained_application_incarnations'
  );
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
