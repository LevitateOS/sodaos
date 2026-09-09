// Real emitted workspace, project controls and xterm; synthetic HTTP/socket peer.
import {mountSodaspaces} from '../../../appliance/forgejo/public/assets/sodaspaces-drawer.js';
import {object} from '../../../appliance/forgejo/public/assets/sodaspaces-api.js';
import type {Space, TerminalMetadata} from '../../../appliance/forgejo/public/assets/sodaspaces-api.js';
export const projectA = 'p' + '1'.repeat(24), projectB = 'p' + '2'.repeat(24);
function createWorkspaceFixture(mode: 'native' | 'page' = 'page') {
  const root = document.querySelector<HTMLElement>('main'); if (!root) throw Error('Missing workspace mount');
  const now = Math.floor(Date.now() / 1000);
  const metadata = (id: string, environment: string, repository: string, name: string): TerminalMetadata => ({id, request_id: id, environment_id: environment, repository_id: repository, user_id: '1', login: 'alice', name,
    created_at: now, hard_until: now + 43200, retain_until: 0, effective_until: now + 43200, ready: true, attached: false, state: 'ready'});
  const spaces: Space[] = [{id: projectA, repository: '7', name: 'Alpha'}, {id: projectB, repository: '8', name: 'Beta'}].map(p => ({environment: {id: p.id, repository_id: p.repository, owner_id: '1', name: p.name, repository: 'alice/' + p.name, provisioned: true}, login: 'alice', environment_administrator: true, authority_unavailable: false, native_unavailable: false, observed: {id: p.id, running: true}, terminals: []}));
  const alpha = spaces[0], beta = spaces[1]; if (!alpha || !beta) throw Error('Missing fixture projects');
  alpha.terminals.push(metadata('a'.repeat(32), projectA, '7', 'Build'), metadata('b'.repeat(32), projectA, '7', 'Edit'));
  beta.terminals.push(metadata('c'.repeat(32), projectB, '8', 'Other project'));
  const calls: {path: string; method: string; body: Record<string, unknown> | null}[] = [], sockets: Socket[] = [];
  let user = '1', complete = true, status = 200, unknownEnd = false;
  let pause: Promise<void> | undefined;
  Object.defineProperty(window, 'fetch', {configurable: true, value: async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = String(input), method = init?.method || 'GET', body = typeof init?.body === 'string' ? object(JSON.parse(init.body)) : null;
    calls.push({path, method, body}); await pause;
    if (status !== 200) return new Response(null, {status});
    if (path.endsWith('/api/session')) return Response.json({user: {id: user, login: 'alice'}, csrf_token: 'synthetic-only', forgejo_url: location.origin});
    if (path.endsWith('/api/spaces')) return Response.json({items: spaces, complete});
    if (path.endsWith('/api/forgejo/me')) return Response.json({id: user});
    if (path.endsWith('/api/me/development-keys')) return Response.json({items: []});
    const space = spaces.find(p => path.includes(p.environment.id) || path.endsWith('repository_id=' + p.environment.repository_id));
    if (!space) return new Response(null, {status: 404});
    if (path.includes('/terminal-sessions/') || path.includes('/terminal-attempts/')) {
      const id = path.split('/').at(-1), terminal = space.terminals.find(t => t.id === id || t.request_id === id);
      if (terminal && body) {
        if (body.action === 'rename' && typeof body.name === 'string') terminal.name = body.name;
        if (body.action === 'end') {terminal.state = unknownEnd ? 'ending' : 'ended'; terminal.ready = terminal.attached = false; return Response.json({ending: true});}
        if (body.action === 'retain' || body.action === 'hide') terminal.retain_until = now + (body.seconds === 7200 ? 7200 : 1800);
        if (body.action === 'return') terminal.retain_until = 0;
        terminal.effective_until = terminal.retain_until || terminal.hard_until;
      }
      return Response.json({terminal: terminal || null});
    }
    if (path.includes('/api/environments?')) return Response.json({items: [space.environment], repository: {id: space.environment.repository_id, owner: 'alice', name: space.environment.name}, can_create: false});
    if (path.endsWith('/lifecycle')) return Response.json({environment: space.observed, boot_enabled: true});
    if (path.endsWith('/connection')) return Response.json({login: 'alice', connection: {environment: {...space.observed, ip: '10.89.0.2'}, fingerprint: 'SHA256:' + 'A'.repeat(43)}});
    return Response.json(space);
  }});
  class Socket {
    readyState = 0; bufferedAmount = 0; closed = 0; sent: Record<string, unknown>[] = [];
    onopen?: () => void; onclose?: () => void; onerror?: () => void; onmessage?: (event: {data: string}) => void;
    constructor(readonly url: string | URL) {sockets.push(this); window.setTimeout(() => {if (!this.closed) {this.readyState = 1; this.onopen?.();}}, 0);}
    send(text: string) {
      const frame = object(JSON.parse(text)); this.sent.push(frame);
      if (frame.action !== 'attach' && frame.action !== 'create') return;
      const space = spaces.find(p => String(this.url).includes(p.environment.id)); if (!space) throw Error('Unknown fixture project');
      let terminal = space.terminals.find(t => t.id === frame.id);
      if (frame.action === 'create') {terminal = metadata(sockets.length.toString(16).padStart(32, '0'), space.environment.id, space.environment.repository_id, ''); terminal.request_id = String(frame.request_id); space.terminals.push(terminal);}
      if (!terminal) throw Error('Fixture has no exact terminal');
      terminal.attached = true;
      this.onmessage?.({data: JSON.stringify({type: 'session', id: terminal.id, request_id: terminal.request_id, attachment_id: sockets.length.toString(16).padStart(32, '0')})});
      this.onmessage?.({data: JSON.stringify({type: 'ready'})});
    }
    close() {this.closed++; this.readyState = 3; this.onclose?.();}
  }
  Object.defineProperty(window, 'WebSocket', {configurable: true, value: Socket});
  const api = mountSodaspaces(root, mode === 'page' ? {kind: 'page', expectedUserId: '1'} : {kind: 'native', expectedUserId: '1', repositoryId: '7'});
  return {api, root, spaces, calls, sockets, setUser(value: string) {user = value;}, setStatus(value: number) {status = value;}, setComplete(value: boolean) {complete = value;}, setUnknownEnd() {unknownEnd = true;}, pause(value: Promise<void> | undefined) {pause = value;}};
}
declare global {interface Window {createWorkspaceFixture: typeof createWorkspaceFixture; workspaceFixture: ReturnType<typeof createWorkspaceFixture>}}
window.createWorkspaceFixture = createWorkspaceFixture;
