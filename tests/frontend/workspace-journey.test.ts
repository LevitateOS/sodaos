import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import {mkdtemp, chmod} from 'node:fs/promises';
import {chromium} from 'playwright';
import type {ServerWebSocket} from 'bun';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import {object, type Space, type TerminalMetadata} from '../../frontend/spaces/sodaspaces-api';
import {journeyInput} from '../installed/sodaspaces-input';
import {matrixInput} from '../installed/sodaspaces-matrix-input';
import {exerciseWorkspaceMatrix, type MatrixEvidence} from '../installed/sodaspaces-workspace-journey';
import {inspectMatrixProcess} from '../installed/sodaspaces-matrix-native';
import {cliProtocolObservation} from '../installed/sodaspaces-cli';
const base = journeyInput({ca_file: '/synthetic/ca', origin: 'https://fixture.invalid', target: 'fixture', oauth_client_id: 'synthetic-client', repository_id: '7', repository_path: '/alice/Alpha', revision: '1'.repeat(40), terminal_actions: ['create', 'end'], users: [{id: '1', login: 'alice', password_file: '/synthetic/a'}, {id: '2', login: 'bob', password_file: '/synthetic/b'}]}, false, true);
const scope = () => ({target: base.target, revision: base.revision, actors: ['1', '2'], sessions_per_actor: 6, actions: ['create', 'attach', 'hide', 'return', 'end'], ssh_config: '/synthetic/ssh', projects: [{environment: 'p' + '1'.repeat(24), repository_id: '7', repository_path: '/alice/Alpha', ssh: ['soda-matrix-a-alice', 'soda-matrix-a-bob']}, {environment: 'p' + '2'.repeat(24), repository_id: '8', repository_path: '/alice/Beta', ssh: ['soda-matrix-b-alice', 'soda-matrix-b-bob']}], cli: [], cli_effects: [], provider_use: 'none'});
test('matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope', () => {
  assert.deepEqual(matrixInput(scope(), base), scope()); assert.throws(() => matrixInput(base, base));
  for (const change of [{target: 'another'}, {revision: '2'.repeat(40)}, {actors: ['2', '1']}, {sessions_per_actor: 7}, {actions: ['create', 'end']}, {actions: [...scope().actions, 'stop']}, {provider_use: 'browser-and-ssh-for-declared-clis'}, {unknown: true}, {projects: [scope().projects[0], scope().projects[0]]}]) assert.throws(() => matrixInput({...scope(), ...change}, base));
  const cli = ['codex', 'claude', 'pi'].map(tool => ({tool, version: 'declared-version', prompt_file: '/synthetic/prompt', expected_text: '非echo', ready_text: 'Fixture CLI ready', minimum_output_bytes: 4096}));
  const cli_effects = ['personal-cli-state', 'provider-calls', 'browser-and-ssh-pty', 'interactive-input', 'interrupt-and-disconnect'];
  assert.equal(matrixInput({...scope(), cli, cli_effects, provider_use: 'browser-and-ssh-for-declared-clis'}, base).cli.length, 3);
  assert.throws(() => matrixInput({...scope(), cli}, base));
  for (const change of [{ready_text: ''}, {version: ''}, {minimum_output_bytes: 1}]) assert.throws(() => matrixInput({...scope(), cli: cli.map(item => ({...item, ...change})), cli_effects, provider_use: 'browser-and-ssh-for-declared-clis'}, base));
  assert.throws(() => matrixInput({...scope(), cli: [cli[0], cli[0], cli[0]], cli_effects, provider_use: 'browser-and-ssh-for-declared-clis'}, base));
  assert.throws(() => matrixInput({...scope(), cli: cli.map(item => ({...item, api_key: 'forbidden'})), cli_effects, provider_use: 'browser-and-ssh-for-declared-clis'}, base));
});
test('exact native observer requires original account, PID/start, unit, cgroup and records; quoted Python compiles without execution', async () => {
  const request = matrixInput(scope(), base), project = request.projects[0];
  const session = {id: 'a'.repeat(32), request: 'b'.repeat(32), attachment: 'c'.repeat(32), name: 'probe', actor: '1', environment: project.environment};
  const facts = {pid: 123, start: '456', login: 'alice', marker: 'probe', tty: true, term: 'screen-256color'};
  const live = {login: 'alice', start: '456', record: true, socket: true, owned_socket: true, populated: '1', state: 'active'};
  await inspectMatrixProcess(request, project, 0, session, facts, false, command => {
    assert(command.includes(session.id));
    const result = Bun.spawnSync(['python3', '-c', 'import shlex,sys; a=shlex.split(sys.stdin.read()); assert a[:3]==["python3","-I","-c"]; compile(a[3],"native-observer","exec")'], {stdin: Buffer.from(command), stdout: 'pipe', stderr: 'pipe'});
    assert.equal(result.exitCode, 0, result.stderr.toString()); return JSON.stringify(live);
  });
  await assert.rejects(inspectMatrixProcess(request, project, 0, session, facts, false, () => JSON.stringify({...live, login: 'bob'})));
  for (const change of [{start: '457'}, {owned_socket: false}, {record: false}, {populated: '0'}, {state: 'inactive'}]) await assert.rejects(inspectMatrixProcess(request, project, 0, session, facts, false, () => JSON.stringify({...live, ...change})));
  let reads = 0;
  await inspectMatrixProcess(request, project, 0, session, facts, true, () => JSON.stringify(++reads === 1 ? live : {...live, start: null, record: false, socket: false, populated: '0', state: 'inactive'}));
  assert.equal(reads, 2);
});
test('CLI parser records observed protocol only, never generated agent semantics or a pass', () => {
  assert.deepEqual(cliProtocolObservation('plain log', 'answer'), {expected_output: false, unicode: false, cursor: false, alternate_screen: false, mouse_mode: false, bracketed_paste: false});
  const observation = cliProtocolObservation('\x1b[?1049h\x1b[?1006h\x1b[?2004h\x1b[2;3H答案', '答案');
  assert(Object.values(observation).every(Boolean)); assert(!('outcome' in observation));
});
for (const actorIndex of [0, 1]) test(`installed matrix actor ${actorIndex} uses actual controls, six exact sockets and single-use requests against a synthetic peer`, {timeout: 30000}, async () => {
  const request = matrixInput(scope(), base), actor = request.actors[actorIndex]; assert(actor);
  const login = actorIndex ? 'bob' : 'alice', now = Math.floor(Date.now() / 1000), root = path.resolve(import.meta.dirname, '../..');
  const spaces: Space[] = request.projects.map(project => ({environment: {id: project.environment, repository_id: project.repository_id, repository: project.repository_path.slice(1), owner_id: '1', name: project.repository_path.split('/').at(-1) || '', provisioned: true}, login, environment_administrator: actor === '1', authority_unavailable: false, native_unavailable: false, observed: {id: project.environment, running: true}, terminals: []}));
  type Wire = {environment: string; id: string};
  const writers = new Map<string, ServerWebSocket<Wire>>(), writes: string[] = [], frames: string[] = [], errors: string[] = [];
  let ids = 0, attachments = 0, allowed = '';
  const tls = await mkdtemp(path.join(root, '.artifacts/matrix-tls-')); await chmod(tls, 0o700);
  assert.equal(Bun.spawnSync(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-subj', '/CN=localhost', '-days', '1', '-keyout', tls + '/key.pem', '-out', tls + '/cert.pem'], {stdout: 'ignore', stderr: 'ignore'}).exitCode, 0);
  const server = Bun.serve<Wire>({hostname: '127.0.0.1', port: 0, tls: {key: Bun.file(tls + '/key.pem'), cert: Bun.file(tls + '/cert.pem')}, 
    async fetch(req, server) {
      const url = new URL(req.url), route = url.pathname;
      if (route.endsWith('/terminal')) {
        const environment = route.split('/')[5]; assert(environment && spaces.some(space => space.environment.id === environment));
        if (server.upgrade(req, {data: {environment, id: ''}})) return;
      }
      if (route.startsWith('/-/soda/api/')) {
        assert.equal(req.headers.get('X-Soda-Expected-User-ID'), actor);
        if (route.endsWith('/session')) return Response.json({user: {id: actor, login}, csrf_token: 'synthetic-only', forgejo_url: url.origin});
        if (route.endsWith('/spaces')) return Response.json({items: spaces, complete: true});
        if (route.endsWith('/forgejo/me')) return Response.json({id: actor});
        const space = spaces.find(space => route.includes(space.environment.id)); assert(space);
        if (route.includes('/terminal-sessions/')) {
          const terminal = space.terminals.find(terminal => terminal.id === route.split('/').at(-1)); assert(terminal);
          if (req.method === 'POST') {
            const body = object(await req.json()), key = route + ':' + JSON.stringify(body); assert.equal(key, allowed); allowed = ''; writes.push(String(body.action));
            if (body.action === 'end') {terminal.state = 'ended'; terminal.ready = false; terminal.attached = false; writers.get(terminal.id)?.close(); return Response.json({ending: true});}
            terminal.retain_until = body.action === 'hide' ? now + 1800 : 0; terminal.effective_until = terminal.retain_until || terminal.hard_until;
          }
          return Response.json({terminal});
        }
        if (route.endsWith('/lifecycle')) return Response.json({environment: space.observed, boot_enabled: true});
        if (route.endsWith('/connection')) return Response.json({login, connection: {environment: {...space.observed, ip: '10.89.0.2'}, fingerprint: 'SHA256:' + 'A'.repeat(43)}});
        return Response.json(space);
      }
      const source = Object.entries(payload).find(([dest]) => dest === 'public' + route)?.[1];
      if (source) {const file = source.startsWith('@build/forgejo-js/') ? path.join(root, '.artifacts/forgejo-js', path.basename(source)) : source.startsWith('@build/terminal-assets/') ? path.join(root, '.artifacts/browser-terminal/vendor', path.basename(source)) : path.join(root, source); return new Response(Bun.file(file), {headers: {'Content-Type': /\.m?js$/.test(source) ? 'text/javascript' : source.endsWith('.css') ? 'text/css' : 'application/octet-stream'}});}
      if (route !== '/-/soda/spaces') return new Response(null, {status: 404});
      return new Response(`<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda-terminal/xterm.css"><style>main{height:calc(100dvh - 40px)}body{margin:0}</style><main id="spaces-page" data-soda-actor="${actor}"></main><script type="module" src="/assets/sodaspaces-page.js"></script>`, {headers: {'Content-Type': 'text/html'}});
    }, websocket: {
      message(ws, raw) {
        const frame = object(JSON.parse(String(raw))); if (frame.action !== 'create' && frame.action !== 'attach') return;
        frames.push(String(frame.action)); const space = spaces.find(space => space.environment.id === ws.data.environment); assert(space);
        let terminal = space.terminals.find(terminal => terminal.id === frame.id);
        if (frame.action === 'create') {
          assert(typeof frame.request_id === 'string' && typeof frame.name === 'string');
          terminal = {id: (++ids).toString(16).padStart(32, '0'), request_id: frame.request_id, environment_id: space.environment.id, repository_id: space.environment.repository_id, user_id: actor, login, name: frame.name, created_at: now, hard_until: now + 43200, retain_until: 0, effective_until: now + 43200, ready: true, attached: false, state: 'ready'} satisfies TerminalMetadata;
          space.terminals.push(terminal);
        }
        assert(terminal && !terminal.attached); terminal.attached = true; ws.data.id = terminal.id; writers.set(terminal.id, ws);
        ws.send(JSON.stringify({type: 'session', id: terminal.id, request_id: terminal.request_id, attachment_id: (++attachments).toString(16).padStart(32, '0')})); ws.send(JSON.stringify({type: 'ready'}));
      }, close(ws) {const terminal = spaces.flatMap(space => space.terminals).find(terminal => terminal.id === ws.data.id); if (terminal && writers.get(terminal.id) === ws) {terminal.attached = false; writers.delete(terminal.id);}},
    }});
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}), context = await browser.newContext({ignoreHTTPSErrors: true}), page = await context.newPage();
  page.setDefaultTimeout(7000); page.on('pageerror', error => errors.push(error.message));
  const evidence: MatrixEvidence = {sessions: []};
  try {
    // Like the installed caller, begin outside the full-page workspace. The
    // scenario must navigate before attempting splits, not use drawer projection.
    await page.goto(new URL('/', server.url).href);
    await exerciseWorkspaceMatrix(page, request, actorIndex, (session, action) => {assert.equal(allowed, ''); allowed = `/-/soda/api/environments/${session.environment}/terminal-sessions/${session.id}:` + JSON.stringify({action, ...(action === 'end' ? {} : {attachment_id: session.attachment})});}, {
      async shell(_page, session) {return {pid: parseInt(session.id, 16), start: '100', login, marker: session.name, tty: true, term: 'screen-256color'};},
      async inspect(project, observedActor, _session, facts, ended) {assert.equal(observedActor, actorIndex); const terminal = spaces.find(space => space.environment.id === project.environment)?.terminals.find(terminal => parseInt(terminal.id, 16) === facts.pid); assert(terminal); assert.equal(terminal.state === 'ended', ended);},
    }, evidence, async () => {const other = await context.newPage(); other.setDefaultTimeout(7000); return other;});
    assert.equal(evidence.sessions.length, 6); assert.equal(evidence.same_document, true); assert.equal(evidence.exact_reload, true);
    assert.deepEqual(evidence.cli, {outcome: 'not-run', reason: 'No declared CLI/provider scope'});
    assert.equal(frames.filter(action => action === 'create').length, 6); assert.equal(frames.filter(action => action === 'attach').length, 6);
    assert.deepEqual(writes, ['hide', 'return', 'end', 'end', 'end', 'end', 'end', 'end']); assert.equal(allowed, ''); assert.deepEqual(errors, []);
  } catch (error) {console.error({fixtureText: await page.locator('body').innerText(), errors, frames}); throw error;}
  finally {await browser.close(); server.stop(true);}
});
