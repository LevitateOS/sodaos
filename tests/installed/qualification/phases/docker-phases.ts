import type {ResourceScope} from '../closure';
// Maintained composition of the selected installed Docker journey. The protected
// collector owns transport, credentials, browser runtime and durable evidence.
// Importing this module never launches it. journey run admission is separate.
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import type {BrowserContext} from 'playwright';
import type {ResolvedQualificationInput} from '../scenario';
import {scenarioInput} from '../validate';
import {loginNativeForgejo} from '../../sodaspaces-controls';
import {mountedSpacesFacts} from '../../sodaspaces-journey-evidence';
import {check, sha, COMMIT, SHA, rawRun, networkGuard, packageAssetHashes, packageAssetName} from './support';
import {ProjectPermits, exerciseProjectPhase, validateProjectPhase} from './project-phase';
import {FactoryPermits, exerciseFactoryPhase} from './factory-phase';
import {validateFactoryPhase} from './factory-native';
import {contextStatusScript} from './context-status';
import {writePersistenceWitness} from './project-persistence-witness';
import {
  loadDockerBinding,
  downloadDockerAssets,
  type DockerHostPaths,
  type DockerIO,
  type FileIdentity,
} from './docker-binding';
import {observeInstallation} from './docker-native';
import {PersistencePermits, retireOwnedResources, exercisePersistence} from './docker-persistence';

/** Public facts missing from the initial identity discovery, plus explicit file
 * locations. No duplicated source/target/resource IDs, paid limits or flags. The
 * controller must pin these dependencies in its admitted resolved input. */
export interface DockerPhaseDetails {
  readonly host: DockerHostPaths;
  readonly public_download_directory: string;
  readonly package_directory: string;
  readonly downloads: Record<string, FileIdentity>;
  readonly repository: {readonly owner: string; readonly name: string; readonly target_branch: string};
  readonly project_profile: string;
  readonly workspace_login: string;
  readonly owner_actor: string;
  readonly issue: {readonly title_digest: string; readonly content_digest: string};
  readonly grant_id: string;
  readonly ci_login: string;
  readonly witness_relative: string;
  readonly context_status: {
    readonly source_revision: string;
    readonly helper: {readonly path: string; readonly SHA256: string; readonly Bytes: number};
    readonly scope: {readonly path: string; readonly SHA256: string};
  };
}
export interface DockerPhaseRuntime extends DockerIO {
  // Existing installed terminal inspectors are synchronous. Do not begin an
  // asynchronous SSH action and then mistake its Promise for native facts.
  sshRead(fixedProgram: string, timeoutMs?: number): string;
  /** Creates a private, trusted context; guards are attached before any page. */
  openBrowser(): Promise<BrowserContext>;
  ownerPassword(): Promise<string>;
}

/** A projection into the migrated phase functions' wire shapes, never another
 * manually edited target.json. It is not paid admission or a readiness proof. */
export function dockerPhaseConfiguration(input: ResolvedQualificationInput, details: DockerPhaseDetails) {
  check(input?.format === 1, 'resolved_input_format');
  const s = scenarioInput(input.scenario);
  check(!s.authority.held && s.authority.scope_receipt_file !== null, 'journey_execution_held');
  check(s.target.distribution === 'docker', 'docker_scenario_required');
  assert.deepEqual(s.cases, ['installation', 'browser_project', 'factory', 'persistence']);
  check(s.factory.length === 1, 'single_factory_composition');
  const selected = s.factory[0]!,
    subject = selected.subject;
  const actual = input.observation.factory.filter(
    (f) => f.repository_id === subject.repository_id && f.issue_number === subject.issue_number
  );
  check(
    actual.length === 1 &&
      actual[0]!.project_id === subject.project_id &&
      actual[0]!.native_issue_id === subject.native_issue_id,
    'observed_factory_selection'
  );
  const connection = actual[0]!.connection;
  check(connection?.id === subject.connection_id && connection.state === 'ready', 'observed_ready_connection');
  check(
    s.managed.project_ids.includes(subject.project_id) &&
      s.permitted_effects.project_lifecycle.includes(subject.project_id) &&
      s.permitted_effects.factory_repositories.includes(subject.repository_id) &&
      s.permitted_effects.same_image_recreation,
    'scenario_phase_effects_required'
  );
  assert.deepEqual([...s.permitted_effects.current_context_phases].sort(), ['implement', 'review']);
  const numberID = (id: string) => {
    const n = Number(id);
    check(Number.isSafeInteger(n) && n > 0 && String(n) === id, 'phase_numeric_identity');
    return n;
  };
  check(
    COMMIT.test(details.context_status.source_revision) &&
      details.context_status.source_revision === s.candidate.source_revision,
    'context_installed_source_binding'
  );
  for (const digest of [details.issue.title_digest, details.issue.content_digest])
    check(SHA.test(digest), 'accepted_issue_digest');
  check(/^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$/.test(details.workspace_login), 'workspace_login');
  const repository = {
    id: numberID(subject.repository_id),
    owner: details.repository.owner,
    name: details.repository.name,
    source_commit: subject.source_commit,
    project: subject.project_id,
  };
  const factory_phase = {
    execution_released: true,
    operation_id: subject.operation_id,
    repository,
    target_branch: details.repository.target_branch,
    issue: {
      number: subject.issue_number,
      native_id: numberID(subject.native_issue_id),
      acceptance: subject.acceptance,
      ...details.issue,
    },
    connection: {
      id: connection.id,
      owner_id: connection.owner_id,
      generation: connection.generation,
      grant: details.grant_id,
    },
    funding: {
      sponsor_minutes: selected.limits.sponsor_minutes,
      rolling_minutes: selected.limits.rolling_minutes,
      max_concurrent: selected.limits.max_concurrent,
    },
    limits: {
      intent_minutes: selected.limits.intent_minutes,
      attempt_minutes: selected.limits.attempt_minutes,
      maximum_seconds: selected.limits.maximum_seconds,
    },
    requirements: subject.requirements,
    approval: subject.approval,
    approved_check_sha256: subject.check_script_sha256,
    check_script_path: subject.check_script_path,
    required_checks: subject.required_checks,
    actors: {
      coder: numberID(subject.actors.coder),
      reviewer: numberID(subject.actors.reviewer),
      ci: numberID(subject.actors.ci),
    },
    ci_login: details.ci_login,
    ci_token_path: '/var/lib/soda-qualification/ci-token',
  };
  validateFactoryPhase(factory_phase);
  validateProjectPhase({
    execution_released: true,
    repository,
    profile_id: details.project_profile,
    actor: details.owner_actor,
  });
  // Validate helper/scope pins before opening any runtime transport. These
  // identities are placeholders for validation only; no status read occurs.
  contextStatusScript(factory_phase, subject.operation_id, subject.operation_id, details.context_status);
  return {
    format: 1,
    purpose: 'production',
    distribution: 'docker',
    execution_released: true,
    target: s.target.name,
    guest: {
      ssh_user: 'root',
      ssh_port: 22,
      ssh_host: new URL(s.target.origin).hostname,
      origin: s.target.origin,
      workspace_user: details.workspace_login,
    },
    expected: s.candidate,
    host: details.host,
    assets: details.downloads,
    factory_phase,
    project_phase: {
      execution_released: true,
      repository,
      profile_id: details.project_profile,
      actor: details.owner_actor,
    },
    persistence_phase: {
      execution_released: true,
      operation_id: subject.operation_id,
      witness_relative: details.witness_relative,
    },
    context_status: details.context_status,
  };
}

/** Fixed inner interpreter, addressed through the existing pinned outer SSH. */
export function innerDocker(program: string) {
  return (
    'const cp=require("node:child_process");try{if(process.getuid()!==0)throw Error();process.stdout.write(cp.execFileSync("/usr/bin/docker",["--host","unix:///var/run/docker.sock","exec","-i","sodaos-soda-1","/usr/bin/node","-"],{input:' +
    JSON.stringify(program) +
    ',encoding:"utf8",timeout:180000,maxBuffer:4<<20,stdio:["pipe","pipe","pipe"]}));}catch{console.error("inner_fixed_command_unconfirmed");process.exitCode=1;}'
  );
}

export async function collectDockerPhases(
  input: ResolvedQualificationInput,
  details: DockerPhaseDetails,
  runtime: DockerPhaseRuntime
) {
  const cfg = dockerPhaseConfiguration(input, details);
  const b = await loadDockerBinding(details.public_download_directory, cfg, details.downloads);
  const io = runtime,
    phases: Record<string, any> = {},
    times: Record<string, {started: string; finished: string}> = {};
  const start = (name: string) => {
    times[name] = {started: new Date().toISOString(), finished: ''};
  };
  const finish = (name: string) => {
    times[name]!.finished = new Date().toISOString();
    check(Date.parse(times[name]!.finished) > Date.parse(times[name]!.started), 'actual_nonzero_phase_interval');
  };
  start('installation');
  await downloadDockerAssets(b, io);
  const installation = await observeInstallation(io, b);
  finish('installation');
  phases.installation = installation.native;
  const pack = b.payload.ExtensionCatalog.packages.find((p: any) => p.id === 'soda');
  check(pack, 'candidate_workspace_package');
  const leaf = new URL(pack.url).pathname.split('/').at(-1);
  check(leaf && /^[A-Za-z0-9._-]+$/.test(leaf), 'candidate_package_leaf');
  const assets = packageAssetHashes(await readFile(details.package_directory + '/' + leaf), pack.sha256),
    executedAssets: Promise<any>[] = [];
  const projectPermits = new ProjectPermits((n, v) => io.save('browser_project/' + n, v));
  const factoryPermits = new FactoryPermits(cfg.guest.origin, cfg.factory_phase.repository.id, (n, v) =>
    io.save('factory/' + n, v)
  );
  const persistencePermits = new PersistencePermits(io.save);
  const audit = {requests: [] as any[], blocked_raw_runs: 0, blocked_other: 0, raw_responses: 0, login_posts: 0};
  const context = await runtime.openBrowser();
  let loginPhase = true;
  try {
    const guard = networkGuard(
      cfg.guest.origin,
      audit,
      () => loginPhase,
      async (request) =>
        (await projectPermits.admit(request)) ||
        (await factoryPermits.admit(request)) ||
        (await persistencePermits.admit(request))
    );
    await context.route('**/*', async (route) => {
      const request = route.request();
      if (await factoryPermits.admitRunAction(request)) {
        audit.requests.push({
          method: request.method(),
          path: new URL(request.url()).pathname,
          blocked: false,
          runActionOnly: true,
        });
        return route.continue();
      }
      return guard(route);
    });
    await context.routeWebSocket('**/*', (socket) => {
      if (rawRun(socket.url())) {
        audit.blocked_raw_runs++;
        socket.close();
        return;
      }
      if (projectPermits.allowsSocket(socket.url(), cfg.guest.origin)) {
        socket.connectToServer();
        return;
      }
      audit.blocked_other++;
      socket.close();
    });
    context.on('response', (r) => {
      if (rawRun(r.url())) {
        if (!factoryPermits.isRunActionResponse(r)) audit.raw_responses++;
        return;
      }
      const relative = packageAssetName(r.url(), cfg.guest.origin, assets);
      if (relative === undefined) return;
      executedAssets.push(
        (async () => {
          const bytes = await r.body();
          check(r.status() === 200 && sha(bytes) === assets[relative], 'actual_shipping_ui_bytes');
          return {name: relative, url: r.url(), SHA256: sha(bytes), Bytes: bytes.length};
        })()
      );
    });
    const page = await context.newPage();
    page.setDefaultTimeout(30000);
    start('browser_project');
    const password = await runtime.ownerPassword();
    check(password.length > 0 && password.length <= 16384, 'owner_password_bound');
    await loginNativeForgejo(page, cfg.guest.origin, cfg.guest.workspace_user, password);
    loginPhase = false;
    const inspectProject = (project: string, r: any) => {
      check(project === '' || /^p[a-f0-9]{24}$/.test(project), 'owned_project_id');
      const program =
        'const REQUEST=' +
        JSON.stringify({project, owner: r.owner, name: r.name}) +
        ';\n' +
        String.raw`
const cp=require('node:child_process');const p=a=>cp.execFileSync('/usr/bin/podman',['--remote=false',...a],{encoding:'utf8',timeout:20000,maxBuffer:16384,stdio:['ignore','pipe','pipe']}).trim();try{if(process.getuid()!==0)throw Error();const source=p(['exec','--user','1000','soda-forgejo','git','--no-optional-locks','--git-dir=/data/git/repositories/'+REQUEST.owner+'/'+REQUEST.name+'.git','rev-parse','refs/heads/main']);if(!REQUEST.project){console.log(JSON.stringify({source}));process.exit(0)}const n='soda-'+REQUEST.project;console.log(JSON.stringify({project:REQUEST.project,container:p(['inspect','--type','container','--format','{{.ID}}',n]),image:p(['inspect','--type','container','--format','{{.Image}}',n]).replace(/^sha256:/,''),source}));}catch{console.error('owned_project_read_unavailable');process.exitCode=1;}`;
      const value = io.sshRead(innerDocker(program), 210000);
      return JSON.parse(value);
    };
    const inspectShell = (project: string, login: string, statement: string) => {
      check(
        /^p[a-f0-9]{24}$/.test(project) &&
          /^[a-z][a-z0-9_-]{0,30}$/.test(login) &&
          login !== 'root' &&
          statement.length <= 16384,
        'owned_member_read'
      );
      const program =
        'const REQUEST=' +
        JSON.stringify({project, login, statement}) +
        ';\n' +
        String.raw`
const cp=require('node:child_process');try{if(process.getuid()!==0)throw Error();process.stdout.write(cp.execFileSync('/usr/bin/podman',['--remote=false','exec','--user',REQUEST.login,'soda-'+REQUEST.project,'/bin/sh','-eu','-c',REQUEST.statement],{encoding:'utf8',timeout:20000,maxBuffer:16384,stdio:['ignore','pipe','pipe']}));}catch{console.error('owned_terminal_read_unavailable');process.exitCode=1;}`;
      const value = io.sshRead(innerDocker(program), 210000);
      return value;
    };
    let projectResult: any;
    const saveProject = async (n: string, v: any) => {
      await io.save('browser_project/' + n, v);
      if (n === 'result.json') projectResult = v;
    };
    await exerciseProjectPhase(
      page,
      cfg,
      {payload: b.payload},
      projectPermits,
      saveProject,
      inspectProject,
      inspectShell,
      async (p) => {
        await io.saveBytes('browser_project/terminal.png', await p.screenshot({fullPage: false}));
      }
    );
    check(projectResult, 'browser_result_required');
    const witness = await writePersistenceWitness(page, cfg, projectResult, (n, v) =>
      io.save('browser_project/' + n, v)
    );
    const loaded = await Promise.all(executedAssets);
    check(
      loaded.some((x) => x.name === 'soda-spaces-entry.js'),
      'executed_shipping_workspace'
    );
    await io.save('browser_project/executed-assets.json', loaded);
    finish('browser_project');
    phases.browser_project = {project: projectResult.environment.id, witness, loaded};
    start('factory');
    const factory = await exerciseFactoryPhase(
      page,
      cfg,
      {payload: b.payload},
      {
        save: (n, v) => io.save('factory/' + n, v),
        saveBytes: (n, v) => io.saveBytes('factory/' + n, v),
        sshRead: (program, timeout) => io.sshRead(innerDocker(program), Math.max(timeout ?? 45000, 210000)),
      },
      factoryPermits
    );
    check(
      factory.observed && factory.paused && factory.native_idle && factory.scope_runs_retired,
      'factory_phase_incomplete'
    );
    finish('factory');
    check('native' in factory, 'factory_native_result_required');
    phases.factory = factory.native;
    // The newly created, canonically returned browser terminal belongs to this
    // journey. Preserve every preceding terminal and its existing scope; never
    // adopt an unrelated native session by copying it into managed resources.
    check(
      projectResult.session.actor === cfg.project_phase.actor &&
        projectResult.session.id === witness.session &&
        projectResult.session.environment === witness.project,
      'created_terminal_binding'
    );
    check(
      !input.observation.inventory.terminals?.some((t) => t.id === witness.session),
      'new_terminal_identity_required'
    );
    const retirementScope: ResourceScope = {
      scenario: {
        ...input.scenario,
        managed: {...input.scenario.managed, terminal_ids: [...input.scenario.managed.terminal_ids, witness.session]},
        permitted_effects: {
          ...input.scenario.permitted_effects,
          terminal_lifecycle: [...input.scenario.permitted_effects.terminal_lifecycle, witness.session],
        },
      },
      observation: {
        ...input.observation,
        inventory: {
          ...input.observation.inventory,
          terminals:
            input.observation.inventory.terminals === null
              ? null
              : [
                  ...input.observation.inventory.terminals,
                  {id: witness.session, project_id: witness.project, actor_id: projectResult.session.actor},
                ],
        },
      },
    };
    start('persistence');
    const mount = await mountedSpacesFacts(page);
    await retireOwnedResources(
      page,
      mount.apiBase,
      witness,
      b,
      io,
      persistencePermits,
      retirementScope,
      cfg.project_phase.actor
    );
    await context.close();
    check(audit.raw_responses === 0, 'raw_provider_response_forbidden');
    await io.save('browser_project/network.json', audit);
    const persistence = await exercisePersistence(b, witness, installation.native, io, retirementScope);
    finish('persistence');
    phases.persistence = persistence.native;
    // Existing canonical collector owns observation envelopes and the seal.
    return {
      phases,
      times,
      artifacts: Object.fromEntries(Object.entries(b.downloads).map(([name, x]) => [name, x.SHA256])),
    };
  } finally {
    await context.close();
  }
}
