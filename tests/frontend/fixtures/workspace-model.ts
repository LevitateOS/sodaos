// Shared synthetic API/socket model for component tests and the local review server.
import {object} from '@factory/product/frontend/factory/api/models.js';
import {createIntentModel} from './intent-model.js';
import type {Space} from '../../../frontend/spaces/sodaspaces-inventory-response.js';
import type {TerminalMetadata} from '../../../frontend/spaces/sodaspaces-terminal-response.js';
import type {DeliveryRepairProgress} from '@factory/product/frontend/factory/api/run-response.js';
export const projectA = 'p' + '1'.repeat(24),
  projectB = 'p' + '2'.repeat(24);
export function createWorkspaceModel(
  origin: string,
  actor = '1',
  repository?: {id: string; owner: string; name: string},
  firstUse = false,
  storage?: Pick<Storage, 'getItem' | 'setItem'>
) {
  const now = Math.floor(Date.now() / 1000);
  const metadata = (id: string, environment: string, repository: string, name: string): TerminalMetadata => ({
    id,
    environment_id: environment,
    repository_id: repository,
    user_id: actor,
    login: 'alice',
    name,
    created_at: now,
    ready: true,
    attached: false,
    state: 'ready',
  });
  const spaces: Space[] = [
    {id: projectA, repository: '7', name: 'Alpha'},
    {id: projectB, repository: '8', name: 'Beta'},
  ].map((p) => ({
    environment: {
      id: p.id,
      repository_id: p.repository,
      owner_id: '1',
      name: p.name,
      repository: 'alice/' + p.name,
      provisioned: true,
    },
    login: 'alice',
    environment_administrator: true,
    execution_allowed: true,
    authority_unavailable: false,
    native_unavailable: false,
    observed: {id: p.id, running: true},
    terminals: [],
    factory_runs: [],
  }));
  const alpha = spaces[0],
    beta = spaces[1];
  if (!alpha || !beta) throw Error('Missing fixture projects');
  const factoryRunId = 'd'.repeat(32),
    factoryOutput = 'Fixture factory output\r\n',
    factoryBinding = {
      container: 'a'.repeat(64),
      unit: `soda-factory-${factoryRunId}.service`,
      invocation: 'f'.repeat(32),
    };
  let factoryRunEnabled = false;
  let verificationEnabled = false,
    verificationAvailable = false,
    verificationRepair = false,
    repairAvailable = false,
    repairReceiptAvailable = false,
    verificationCommand = '';
  let repairProgress: DeliveryRepairProgress | undefined;
  alpha.terminals.push(
    metadata('a'.repeat(32), projectA, '7', 'Build'),
    metadata('b'.repeat(32), projectA, '7', 'Edit')
  );
  beta.terminals.push(metadata('c'.repeat(32), projectB, '8', 'Other project'));
  if (repository) {
    alpha.environment.repository_id = repository.id;
    alpha.environment.repository = repository.owner + '/' + repository.name;
    for (const terminal of alpha.terminals) terminal.repository_id = repository.id;
    for (const space of spaces)
      for (const terminal of space.terminals) {
        const saved = storage?.getItem('fixture-terminal:' + terminal.id);
        if (saved === 'ended' || saved === 'ending') {
          terminal.state = saved;
          terminal.ready = false;
        }
      }
  }
  const factoryNativeRepositoryId = alpha.environment.repository_id;
  const nativeRepositories = new Map(
    [alpha, beta].map((space) => [space.environment.repository_id, space.environment.repository])
  );
  const factoryRepositories = new Map<string, string>();
  let factoryActor = actor;
  let factoryRunRepository: string | undefined;
  let factoryDiscoveryAllowed = true;
  let factoryNativeActor: string | undefined;
  let factoryAccountWait: Promise<void> | undefined;
  let releaseFactoryAccount: (() => void) | undefined;
  let factoryDiscoveryWait: Promise<void> | undefined;
  let releaseFactoryDiscovery: (() => void) | undefined;
  if (firstUse) spaces.splice(0);
  const profile = {
    id: 'rocky-headless',
    distribution: 'rocky',
    version: '9',
    interface: 'headless',
    architecture: 'amd64',
    image: 'sha256:' + '1'.repeat(64),
    revision: '1'.repeat(40),
  };
  let createOutcome: 'confirmed' | 'uncertain' | 'incomplete' | 'rejected' = 'confirmed',
    joinFailure: false | string = false,
    profileAvailable = true;
  const calls: {path: string; method: string; body: Record<string, unknown> | null}[] = [],
    sockets: Socket[] = [];
  const generation = 'fixture-generation';
  let user = actor,
    complete = true,
    status = 200,
    unknownEnd = false,
    serial = 0;
  const reservations = new Set<string>();
  let pause: Promise<void> | undefined;
  const intentRequest = createIntentModel(() => factoryActor);
  const request = async (path: string, method = 'GET', body: Record<string, unknown> | null = null) => {
    calls.push({path, method, body});
    await pause;
    if (status !== 200 || user !== actor) return new Response(null, {status: user !== actor ? 403 : status});
    if (path.endsWith('/api/factory/identity')) {
      await factoryAccountWait;
      return Response.json({host: 'a'.repeat(32), actor: factoryActor, native_actor: factoryNativeActor ?? user});
    }
    if (path.endsWith('/api/factory/native-repositories/resolve')) {
      await factoryDiscoveryWait;
      if (!factoryDiscoveryAllowed) return new Response(null, {status: 404});
      const native = String(body?.repository);
      const space = spaces.find((item) => item.environment.repository_id === native);
      const name = space?.environment.repository ?? nativeRepositories.get(native);
      if (!name) return new Response(null, {status: 404});
      return Response.json({
        host: 'a'.repeat(32),
        actor: factoryActor,
        repository: factoryRepositories.get(native) ?? native,
        native_repository: native,
        name,
      });
    }
    const repositoryIdentity = new URL(path, origin).pathname.match(
      /\/api\/factory\/repositories\/([1-9][0-9]*)\/identity$/
    );
    if (repositoryIdentity) {
      await factoryDiscoveryWait;
      if (!factoryDiscoveryAllowed) return new Response(null, {status: 404});
      const native = [...nativeRepositories.keys()].find(
        (native) => (factoryRepositories.get(native) ?? native) === repositoryIdentity[1]
      );
      if (!native) return new Response(null, {status: 404});
      return Response.json({
        host: 'a'.repeat(32),
        actor: factoryActor,
        repository: repositoryIdentity[1],
        native_repository: native,
        name:
          spaces.find((space) => space.environment.repository_id === native)?.environment.repository ??
          nativeRepositories.get(native),
      });
    }
    const workIssue = new URL(path, origin).pathname.match(
      /\/api\/repositories\/([1-9][0-9]*)\/factory\/issues\/([1-9][0-9]*)$/
    );
    if (method === 'GET' && factoryRunEnabled && workIssue) {
      const visible = spaces.find(
        (space) =>
          (factoryRepositories.get(space.environment.repository_id) ?? space.environment.repository_id) ===
            workIssue[1] && space.factory_runs.some((run) => run.issue === workIssue[2])
      );
      if (visible)
        return Response.json({
          repository: workIssue[1],
          issue: workIssue[2],
          status: {valid: true},
          capabilities: {watch: {available: true}},
        });
    }
    const intent = intentRequest(path, method, body);
    if (intent) return intent;
    if (path.endsWith('/api/spaces'))
      return Response.json({
        actor: {id: user, login: 'soda-tester'},
        items: spaces.map((space) => ({...space, terminals: space.terminals.filter((t) => t.state !== 'ended')})),
        complete,
      });
    if (
      verificationEnabled &&
      path.endsWith(`/api/factory/runs/${factoryRunId}/actions`) &&
      method === 'POST' &&
      body?.action === (verificationRepair ? 'repair_delivery' : 'retry')
    ) {
      if (verificationRepair) {
        repairAvailable = false;
        return Response.json({
          run_id: repairReceiptAvailable ? factoryRunId : 'c'.repeat(32),
          cause_id: 'a'.repeat(32),
          cycle_id: 'b'.repeat(32),
          state: 'queued',
        });
      }
      verificationAvailable = false;
      repairAvailable = false;
      if (!verificationCommand) {
        verificationCommand = String(body.command_id);
        throw new TypeError('Fixture lost the recorded Retry reply');
      }
      if (body.command_id !== verificationCommand) return Response.json({}, {status: 409});
      return Response.json({
        command_id: verificationCommand,
        prior_run: factoryRunId,
        queued: true,
        reason: 'New verification queued',
      });
    }
    const factoryStatusPath = new URL(path, origin).pathname.match(/\/api\/factory\/runs\/([a-f0-9]{32})$/);
    if (factoryStatusPath) {
      if (!factoryRunEnabled || factoryStatusPath[1] !== factoryRunId) return new Response(null, {status: 404});
      const repository =
        factoryRunRepository ??
        factoryRepositories.get(alpha.environment.repository_id) ??
        alpha.environment.repository_id;
      if (verificationEnabled)
        return Response.json({
          run: {
            id: factoryRunId,
            project_id: alpha.environment.id,
            role: 'soda-reviewer',
            harness: 'fixture',
            model: 'fixture',
            input_sha: 'e'.repeat(40),
            reconciled: true,
            outcome: 'succeeded',
          },
          view: {repository, attempt: 'f'.repeat(32)},
          verification_retry: {available: verificationAvailable},
          delivery_repair: {available: repairAvailable, ...(repairProgress ? {progress: repairProgress} : {})},
        });
      return Response.json({
        run: {
          started: new Date(now * 1000).toISOString(),
          deadline: new Date((now + 3600) * 1000).toISOString(),
          id: factoryRunId,
          project_id: alpha.environment.id,
          role: 'soda-coder',
          harness: 'fixture',
          model: 'fixture',
          input_sha: 'e'.repeat(40),
          reconciled: false,
        },
        view: {repository, issue: '42', attempt: 'attempt-1'},
        state: {phase: 'running', live: true, terminal: false, output_truncated: false, ...factoryBinding},
      });
    }
    if (firstUse) {
      const url = new URL(path, origin);
      if (url.pathname.endsWith('/api/repositories')) {
        const items = [
          {id: '7', name: 'Alpha'},
          {id: '8', name: 'Beta'},
        ]
          .filter((r) => r.name.toLowerCase().includes((url.searchParams.get('q') || '').toLowerCase()))
          .map((r) => {
            const project = spaces.find((s) => s.environment.repository_id === r.id);
            return {
              ...r,
              owner: 'alice',
              can_create: !project,
              project: project ? {id: project.environment.id, provisioned: project.environment.provisioned} : null,
            };
          });
        return Response.json({items});
      }
      if (path.endsWith('/profiles'))
        return profileAvailable
          ? Response.json({items: [profile]})
          : Response.json({error: {code: 'profile_unavailable'}}, {status: 503});
      if (path.endsWith('/tailnet-options')) return new Response(null, {status: 404});
      if (url.pathname.endsWith('/api/environments') && method === 'GET') {
        const id = url.searchParams.get('repository_id'),
          space = spaces.find((s) => s.environment.repository_id === id);
        return Response.json({
          items: space ? [space.environment] : [],
          repository: {id, owner: 'alice', name: id === '7' ? 'Alpha' : 'Beta'},
          can_create: !space,
        });
      }
      if (path.endsWith('/api/environments') && method === 'POST') {
        if (createOutcome === 'rejected') return Response.json({error: {code: 'profile_unavailable'}}, {status: 422});
        const base = body?.repository_id === '7' ? alpha : beta;
        if (spaces.some((s) => s.environment.id === base.environment.id))
          throw Error('Fixture refuses duplicate project creation');
        const space: Space = {
          ...base,
          login: '',
          terminals: [],
          factory_runs: [],
          environment: {...base.environment, profile, provisioned: createOutcome !== 'incomplete'},
        };
        spaces.push(space);
        return createOutcome === 'confirmed'
          ? Response.json(space.environment, {status: 201})
          : Response.json({error: {code: 'provisioning_incomplete'}, environment: space.environment}, {status: 502});
      }
      if (path.endsWith('/api/me/development-keys')) return new Response(null, {status: 503});
    }
    if (path.endsWith('/api/me/development-keys')) return Response.json({items: []});
    const missingRepository = new URL(path, origin);
    if (
      missingRepository.pathname.endsWith('/api/environments') &&
      missingRepository.searchParams.get('repository_id') === '9'
    )
      return Response.json({items: [], repository: {id: '9', owner: 'alice', name: 'Missing'}, can_create: true});
    if (missingRepository.pathname.endsWith('/api/repositories/9/profiles')) return Response.json({items: [profile]});
    const space = spaces.find(
      (p) => path.includes(p.environment.id) || path.endsWith('repository_id=' + p.environment.repository_id)
    );
    if (!space) return new Response(null, {status: 404});
    if (path.endsWith('/join') && method === 'POST') {
      if (joinFailure) return Response.json({error: {code: joinFailure}}, {status: 502});
      if (body?.ssh_keys !== 'none') throw Error('First-use must join without keys');
      space.login = 'alice';
      return Response.json({login: space.login});
    }
    if (path.endsWith('/terminal-sessions') && method === 'POST') {
      const id = (++serial).toString(16).padStart(32, '0');
      reservations.add(id);
      return Response.json({id}, {status: 201});
    }
    if (path.includes('/terminal-sessions/')) {
      const id = path.split('/').at(-1),
        terminal = space.terminals.find((t) => t.id === id);
      if (terminal && body) {
        if (body.action === 'rename' && typeof body.name === 'string') terminal.name = body.name;
        if (body.action === 'end') {
          if (unknownEnd) return new Response(null, {status: 503});
          terminal.state = 'ended';
          terminal.ready = terminal.attached = false;
          for (const socket of sockets) socket.end(terminal.id);
          if (repository) storage?.setItem('fixture-terminal:' + terminal.id, terminal.state);
        }
      }
      return Response.json({terminal: terminal || null});
    }
    if (path.includes('/api/environments?'))
      return Response.json({
        items: [space.environment],
        repository: {
          id: space.environment.repository_id,
          owner: repository?.owner || 'alice',
          name: repository?.name || space.environment.name,
        },
        can_create: false,
      });
    if (path.endsWith('/lifecycle')) {
      if (body && space.observed) space.observed.running = body.action === 'start';
      return Response.json({environment: space.observed, boot_enabled: space.observed?.running === true});
    }
    if (path.endsWith('/connection'))
      return Response.json({
        login: 'alice',
        connection: {environment: {...space.observed, ip: '10.89.0.2'}, fingerprint: 'SHA256:' + 'A'.repeat(43)},
      });
    return Response.json(space);
  };
  class Socket {
    readyState = 0;
    bufferedAmount = 0;
    closed = 0;
    sent: Record<string, unknown>[] = [];
    received: Record<string, unknown>[] = [];
    private attachment: TerminalMetadata | undefined;
    onopen?: () => void;
    onclose?: () => void;
    onerror?: () => void;
    onmessage?: (event: {data: string}) => void;
    constructor(readonly url: string | URL) {
      sockets.push(this);
      setTimeout(() => {
        if (!this.closed) {
          this.readyState = 1;
          this.onopen?.();
        }
      }, 0);
    }
    send(text: string) {
      const frame = object(JSON.parse(text));
      this.sent.push(frame);
      if (String(this.url).includes(`/factory/runs/${factoryRunId}/output`)) {
        if (
          factoryRunEnabled &&
          frame.run_id === factoryRunId &&
          frame.repository_id === factoryNativeRepositoryId &&
          frame.session_generation === generation &&
          frame.cursor === 0
        ) {
          // WebSocket replies arrive after send returns, never synchronously.
          queueMicrotask(() => {
            if (this.closed) return;
            this.receive({
              type: 'status',
              run_id: factoryRunId,
              phase: 'running',
              ...factoryBinding,
              reason: '',
              live: true,
              terminal: false,
              exit_code: null,
            });
            const bytes = new TextEncoder().encode(factoryOutput);
            this.receive({
              type: 'output',
              data: btoa(String.fromCharCode(...bytes)),
              cursor: frame.cursor,
              next: Number(frame.cursor) + bytes.length,
              gap: false,
              truncated: false,
            });
          });
        } else this.close();
        return;
      }
      if (frame.type === 'input' && typeof frame.data === 'string')
        this.onmessage?.({data: JSON.stringify({type: 'output', data: frame.data})});
      if (frame.action !== 'attach' && frame.action !== 'create') return;
      if (frame.session_generation !== generation) {
        // Mirror the native handshake: a stale generation refuses the peer without
        // attaching or reporting ready.
        this.close();
        return;
      }
      const space = spaces.find((p) => String(this.url).includes(p.environment.id));
      if (!space) throw Error('Unknown fixture project');
      let terminal = space.terminals.find((t) => t.id === frame.id);
      if (frame.action === 'create') {
        if (typeof frame.id !== 'string' || !reservations.delete(frame.id) || terminal)
          throw Error('Unreserved/reused fixture creation');
        terminal = metadata(
          frame.id,
          space.environment.id,
          space.environment.repository_id,
          typeof frame.name === 'string' ? frame.name : ''
        );
        space.terminals.push(terminal);
      }
      if (!terminal) throw Error('Fixture has no exact terminal');
      terminal.attached = true;
      this.attachment = terminal;
      this.onmessage?.({data: JSON.stringify({type: 'ready'})});
    }
    end(id: string) {
      if (this.attachment?.id === id) this.close();
    }
    receive(frame: Record<string, unknown>) {
      this.received.push(frame);
      this.onmessage?.({data: JSON.stringify(frame)});
    }
    close() {
      this.closed++;
      this.readyState = 3;
      // Closing detaches this writer; it never Ends native work.
      if (this.attachment) this.attachment.attached = false;
      this.attachment = undefined;
      this.onclose?.();
    }
  }
  return {
    generation,
    request,
    Socket,
    profile,
    spaces,
    calls,
    sockets,
    addVisibleRepository(native: string, name: string) {
      nativeRepositories.set(native, name);
    },
    setFactoryNativeActor(native: string) {
      factoryNativeActor = native;
    },
    holdFactoryAccount() {
      factoryAccountWait = new Promise<void>((resolve) => {
        releaseFactoryAccount = resolve;
      });
    },
    releaseFactoryAccount() {
      factoryAccountWait = undefined;
      releaseFactoryAccount?.();
      releaseFactoryAccount = undefined;
    },
    setFactoryIdentity(native: string, repository: string, canonicalActor = actor) {
      factoryRepositories.set(native, repository);
      factoryActor = canonicalActor;
    },
    setFactoryRunRepository(repository: string) {
      factoryRunRepository = repository;
    },
    denyFactoryDiscovery() {
      factoryDiscoveryAllowed = false;
    },
    holdFactoryDiscovery() {
      factoryDiscoveryWait = new Promise<void>((resolve) => {
        releaseFactoryDiscovery = resolve;
      });
    },
    releaseFactoryDiscovery() {
      factoryDiscoveryWait = undefined;
      releaseFactoryDiscovery?.();
      releaseFactoryDiscovery = undefined;
    },
    setCreateOutcome(value: typeof createOutcome) {
      createOutcome = value;
    },
    enableFactoryRun(issue = '42') {
      factoryRunEnabled = true;
      if (!alpha.factory_runs.some((run) => run.id === factoryRunId))
        alpha.factory_runs.push({
          id: factoryRunId,
          role: 'soda-coder',
          issue,
          attempt: 'attempt-1',
          reconciled: false,
        });
    },
    enableVerificationRetry(repair = false) {
      factoryRunEnabled = verificationEnabled = true;
      verificationAvailable = !repair;
      verificationRepair = repairAvailable = repair;
      verificationCommand = '';
      repairProgress = undefined;
      repairReceiptAvailable = false;
      if (!alpha.factory_runs.some((run) => run.id === factoryRunId))
        alpha.factory_runs.push({
          id: factoryRunId,
          role: 'soda-reviewer',
          attempt: 'f'.repeat(32),
          reconciled: true,
          outcome: 'succeeded',
        });
    },
    allowRepairReceipt() {
      repairReceiptAvailable = true;
    },
    setRepairProgress(state: DeliveryRepairProgress['state'], foreign = false) {
      repairAvailable = false;
      repairProgress = {
        cause_id: 'a'.repeat(32),
        cycle_id: 'b'.repeat(32),
        failed_run: foreign ? 'c'.repeat(32) : factoryRunId,
        source_commit: 'e'.repeat(40),
        state,
        run_id: '9'.repeat(32),
        failures: [{check_ref: 'priority-state', detail: 'Priority did not persist after the shared-state update.'}],
        failure_count: 1,
        ...(['attention', 'adopted'].includes(state)
          ? {
              judgment: {
                option: 'accept' as const,
                accept_probability: 0.87,
                confidence: state === 'attention' ? 0.24 : 0.88,
              },
              reason:
                state === 'attention'
                  ? 'delivery repair proposal needs owner attention'
                  : 'repair_graph_adopted_awaiting_fresh_readiness',
            }
          : {}),
      };
    },
    setJoinFailure(value: boolean | string) {
      joinFailure = value === true ? 'account_incomplete' : value;
    },
    setProfileAvailable(value: boolean) {
      profileAvailable = value;
    },
    setUser(value: string) {
      user = value;
    },
    setStatus(value: number) {
      status = value;
    },
    setComplete(value: boolean) {
      complete = value;
    },
    setUnknownEnd() {
      unknownEnd = true;
    },
    pause(value: Promise<void> | undefined) {
      pause = value;
    },
  };
}
