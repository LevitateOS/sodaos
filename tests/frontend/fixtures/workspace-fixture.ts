// Real emitted workspace, project controls and xterm; synthetic HTTP/socket peer.
import {mountSodaspaces} from '../../../frontend/spaces/sodaspaces-workspace.js';
import {object} from '../../../frontend/spaces/sodaspaces-api.js';
import type {Space, TerminalMetadata} from '../../../frontend/spaces/sodaspaces-api.js';
export const projectA = 'p' + '1'.repeat(24), projectB = 'p' + '2'.repeat(24);
export function installWorkspaceModel(actor = '1', repository?: {id: string; owner: string; name: string}, firstUse = false) {
  const root = document.querySelector<HTMLElement>('main, .page-content'); if (!root) throw Error('Missing workspace mount');
  const now = Math.floor(Date.now() / 1000);
  const metadata = (id: string, environment: string, repository: string, name: string): TerminalMetadata => ({id, environment_id: environment, repository_id: repository, user_id: actor, login: 'alice', name,
    created_at: now, ready: true, attached: false, state: 'ready'});
  const spaces: Space[] = [{id: projectA, repository: '7', name: 'Alpha'}, {id: projectB, repository: '8', name: 'Beta'}].map(p => ({environment: {id: p.id, repository_id: p.repository, owner_id: '1', name: p.name, repository: 'alice/' + p.name, provisioned: true}, login: 'alice', environment_administrator: true, authority_unavailable: false, native_unavailable: false, observed: {id: p.id, running: true}, terminals: []}));
  const alpha = spaces[0], beta = spaces[1]; if (!alpha || !beta) throw Error('Missing fixture projects');
  alpha.terminals.push(metadata('a'.repeat(32), projectA, '7', 'Build'), metadata('b'.repeat(32), projectA, '7', 'Edit'));
  beta.terminals.push(metadata('c'.repeat(32), projectB, '8', 'Other project'));
  if (repository) {
    alpha.environment.repository_id = repository.id;
    alpha.environment.repository = repository.owner + '/' + repository.name;
    for (const terminal of alpha.terminals) terminal.repository_id = repository.id;
    for (const space of spaces) for (const terminal of space.terminals) {
      const saved = sessionStorage.getItem('fixture-terminal:' + terminal.id);
      if (saved === 'ended' || saved === 'ending') {terminal.state = saved; terminal.ready = false;}
    }
  }
  if (firstUse) spaces.splice(0);
  const profile = {id: 'rocky-headless', distribution: 'rocky', version: '9', interface: 'headless', architecture: 'amd64', image: 'sha256:' + '1'.repeat(64), revision: '1'.repeat(40)};
  let createOutcome: 'confirmed' | 'uncertain' | 'incomplete' | 'rejected' = 'confirmed', joinFailure = false, profileAvailable = true;
  const calls: {path: string; method: string; body: Record<string, unknown> | null}[] = [], sockets: Socket[] = [];
  let user = actor, complete = true, status = 200, unknownEnd = false, serial = 0;
  const reservations = new Set<string>();
  let pause: Promise<void> | undefined;
  const nativeFetch = window.fetch.bind(window);
  Object.defineProperty(window, 'fetch', {configurable: true, value: async (input: RequestInfo | URL, init?: RequestInit) => {
    if (!String(input).includes('/-/soda/api/')) return nativeFetch(input, init);
    const path = String(input), method = init?.method || 'GET', body = typeof init?.body === 'string' ? object(JSON.parse(init.body)) : null;
    calls.push({path, method, body}); await pause;
    if (status !== 200 || user !== actor) return new Response(null, {status: user !== actor ? 403 : status});
    if (path.endsWith('/api/session')) return Response.json({user: {id: user, login: 'alice'}, csrf_token: 'synthetic-only', forgejo_url: location.origin});
    if (path.endsWith('/api/spaces')) return Response.json({items: spaces.map(space => ({...space, terminals: space.terminals.filter(t => t.state !== 'ended')})), complete});
    if (path.endsWith('/api/forgejo/me')) return Response.json({id: user});
    if (firstUse) {
      const url = new URL(path, location.origin);
      if (url.pathname.endsWith('/api/repositories')) {
        const items = [{id: '7', name: 'Alpha'}, {id: '8', name: 'Beta'}].filter(r => r.name.toLowerCase().includes((url.searchParams.get('q') || '').toLowerCase())).map(r => {
          const project = spaces.find(s => s.environment.repository_id === r.id);
          return {...r, owner: 'alice', can_create: !project, project: project ? {id: project.environment.id, provisioned: project.environment.provisioned} : null};
        });
        return Response.json({items, page: Number(url.searchParams.get('page')), more: false, limited: false});
      }
      if (path.endsWith('/profiles')) return profileAvailable ? Response.json({items: [profile]}) : Response.json({error: {code: 'profile_unavailable'}}, {status: 503});
      if (path.endsWith('/tailnet-options')) return new Response(null, {status: 404});
      if (url.pathname.endsWith('/api/environments') && method === 'GET') {
        const id = url.searchParams.get('repository_id'), space = spaces.find(s => s.environment.repository_id === id);
        return Response.json({items: space ? [space.environment] : [], repository: {id, owner: 'alice', name: id === '7' ? 'Alpha' : 'Beta'}, can_create: !space});
      }
      if (path.endsWith('/api/environments') && method === 'POST') {
        if (createOutcome === 'rejected') return Response.json({error: {code: 'profile_unavailable'}}, {status: 422});
        const base = body?.repository_id === '7' ? alpha : beta;
        if (spaces.some(s => s.environment.id === base.environment.id)) throw Error('Fixture refuses duplicate project creation');
        const space: Space = {...base, login: '', terminals: [], environment: {...base.environment, profile, provisioned: createOutcome !== 'incomplete'}};
        spaces.push(space);
        return createOutcome === 'confirmed' ? Response.json(space.environment, {status: 201}) : Response.json({error: {code: 'provisioning_incomplete'}, environment: space.environment}, {status: 502});
      }
      if (path.endsWith('/api/me/development-keys')) return new Response(null, {status: 503});
    }
    if (path.endsWith('/api/me/development-keys')) return Response.json({items: []});
    const space = spaces.find(p => path.includes(p.environment.id) || path.endsWith('repository_id=' + p.environment.repository_id));
    if (!space) return new Response(null, {status: 404});
    if (path.endsWith('/join') && method === 'POST') {
      if (joinFailure) return Response.json({error: {code: 'join_failed'}}, {status: 502});
      if (body?.ssh_keys !== 'none') throw Error('First-use must join without keys');
      space.login = 'alice'; return Response.json({login: space.login});
    }
    if (path.endsWith('/terminal-sessions') && method === 'POST') {
      const id = (++serial).toString(16).padStart(32, '0'); reservations.add(id); return Response.json({id}, {status: 201});
    }
    if (path.includes('/terminal-sessions/')) {
      const id = path.split('/').at(-1), terminal = space.terminals.find(t => t.id === id);
      if (terminal && body) {
        if (body.action === 'rename' && typeof body.name === 'string') terminal.name = body.name;
        if (body.action === 'end') {
          if (unknownEnd) return new Response(null, {status: 503});
          terminal.state = 'ended'; terminal.ready = terminal.attached = false;
          for (const socket of sockets) socket.end(terminal.id);
          if (repository) sessionStorage.setItem('fixture-terminal:' + terminal.id, terminal.state);
        }
      }
      return Response.json({terminal: terminal || null});
    }
    if (path.includes('/api/environments?')) return Response.json({items: [space.environment], repository: {id: space.environment.repository_id, owner: repository?.owner || 'alice', name: repository?.name || space.environment.name}, can_create: false});
    if (path.endsWith('/lifecycle')) {
      if (body && space.observed) space.observed.running = body.action === 'start';
      return Response.json({environment: space.observed, boot_enabled: space.observed?.running === true});
    }
    if (path.endsWith('/connection')) return Response.json({login: 'alice', connection: {environment: {...space.observed, ip: '10.89.0.2'}, fingerprint: 'SHA256:' + 'A'.repeat(43)}});
    return Response.json(space);
  }});
  class Socket {
    readyState = 0; bufferedAmount = 0; closed = 0; sent: Record<string, unknown>[] = [];
    private attachment: TerminalMetadata | undefined;
    onopen?: () => void; onclose?: () => void; onerror?: () => void; onmessage?: (event: {data: string}) => void;
    constructor(readonly url: string | URL) {sockets.push(this); window.setTimeout(() => {if (!this.closed) {this.readyState = 1; this.onopen?.();}}, 0);}
    send(text: string) {
      const frame = object(JSON.parse(text)); this.sent.push(frame);
      if (frame.type === 'input' && typeof frame.data === 'string') this.onmessage?.({data: JSON.stringify({type: 'output', data: frame.data})});
      if (frame.action !== 'attach' && frame.action !== 'create') return;
      const space = spaces.find(p => String(this.url).includes(p.environment.id)); if (!space) throw Error('Unknown fixture project');
      let terminal = space.terminals.find(t => t.id === frame.id);
      if (frame.action === 'create') {
        if (typeof frame.id !== 'string' || !reservations.delete(frame.id) || terminal) throw Error('Unreserved/reused fixture creation');
        terminal = metadata(frame.id, space.environment.id, space.environment.repository_id, typeof frame.name === 'string' ? frame.name : ''); space.terminals.push(terminal);
      }
      if (!terminal) throw Error('Fixture has no exact terminal');
      terminal.attached = true;
      this.attachment = terminal;
      this.onmessage?.({data: JSON.stringify({type: 'ready'})});
    }
    end(id: string) {if (this.attachment?.id === id) this.close();}
    close() {
      this.closed++; this.readyState = 3;
      // Closing detaches this writer; it never Ends native work.
      if (this.attachment) this.attachment.attached = false;
      this.attachment = undefined;
      this.onclose?.();
    }
  }
  Object.defineProperty(window, 'WebSocket', {configurable: true, value: Socket});
  return {root, spaces, calls, sockets, setCreateOutcome(value: typeof createOutcome) {createOutcome = value;}, setJoinFailure(value: boolean) {joinFailure = value;}, setProfileAvailable(value: boolean) {profileAvailable = value;}, setUser(value: string) {user = value;}, setStatus(value: number) {status = value;}, setComplete(value: boolean) {complete = value;}, setUnknownEnd() {unknownEnd = true;}, pause(value: Promise<void> | undefined) {pause = value;}};
}
function createWorkspaceFixture(mode: 'native' | 'page' = 'page', firstUse = false) {
  const model = installWorkspaceModel('1', undefined, firstUse);
  const context = mode === 'page' ? {kind: 'page' as const, expectedUserId: '1'} : {kind: 'native' as const, expectedUserId: '1', repositoryId: '7', pageRepositoryId: '7'};
  let api = mountSodaspaces(model.root, context);
  return {get api() {return api;}, ...model, async remount() {api.dispose(); api = mountSodaspaces(model.root, context); await api.ready; await api.refresh();}};
}
declare global {interface Window {createWorkspaceFixture: typeof createWorkspaceFixture; workspaceFixture: ReturnType<typeof createWorkspaceFixture>}}
window.createWorkspaceFixture = createWorkspaceFixture;
