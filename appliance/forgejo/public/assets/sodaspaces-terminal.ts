import { object } from "./sodaspaces-api.js";
import type { Terminal, ITerminalOptions, ITerminalInitOnlyOptions, ITerminalAddon } from "@xterm/xterm";
import type { FitAddon } from "@xterm/addon-fit";

export type TerminalView = Pick<Terminal, "cols" | "rows" | "options" | "parser" | "loadAddon" | "attachCustomKeyEventHandler" | "onData" | "open" | "focus" | "dispose" | "write">;
export interface Renderer { Terminal: new (options: ITerminalOptions & ITerminalInitOnlyOptions) => TerminalView; FitAddon: new () => Pick<FitAddon, "fit"> & ITerminalAddon }
export interface TerminalContext { expectedUserId: string; repositoryId: string; environmentId: string; login: string }
// Self-contained optional drawer content. No template selectors, navigation,
// global mounting or native cookie access. The host supplies one mount and
// immutable native-page consistency hints; only the backend grants authority.
const identifier = (value: unknown): value is string => typeof value === 'string' && /^[1-9][0-9]*$/.test(value) && BigInt(value) <= 9223372036854775807n;
const terminalID = (value: unknown): value is string => typeof value === 'string' && /^[0-9a-f]{32}$/.test(value);
const renderer = async (): Promise<Renderer> => {
  const [{Terminal}, {FitAddon}] = await Promise.all([
    import('./soda-terminal/xterm.mjs'), import('./soda-terminal/addon-fit.mjs'),
  ]);
  return {Terminal, FitAddon};
};

// Shared by the two Soda API components; never accept HTML or unbounded bodies.
export async function readSodaJSON(response: Response): Promise<unknown> {
  if (!/^application\/json(?:;|$)/i.test(response.headers.get('Content-Type') || '') || Number(response.headers.get('Content-Length')) > 65536) {
    await response.body?.cancel(); throw Error('Invalid Soda response');
  }
  if (!response.body) throw Error("Missing Soda response body");
  const reader = response.body.getReader(); const decoder = new TextDecoder('utf-8', {fatal: true});
  let size = 0, text = '';
  try {
    for (;;) {
      const {done, value} = await reader.read(); if (done) break;
      size += value.byteLength; if (size > 65536) throw Error('Oversized Soda response');
      text += decoder.decode(value, {stream: true});
    }
    return JSON.parse(text + decoder.decode());
  } catch (error) { await reader.cancel(); throw error; }
  finally { reader.releaseLock(); }
}

export function mountTerminal(root: HTMLElement, context: TerminalContext, loadRenderer: () => Promise<Renderer> = renderer) {
  const {expectedUserId, repositoryId, environmentId, login} = context;
  if (!root || !identifier(expectedUserId) || !identifier(repositoryId) || !/^p[0-9a-f]{24}$/.test(environmentId) || !/^[a-z][a-z0-9_-]{0,30}$/.test(login) || login === 'root') throw Error('Invalid terminal mounting context');
  const doc = root.ownerDocument, view = doc.defaultView;
  if (!view) throw Error('A document window is required');
  const win = view as Window & typeof globalThis;
  const lifetime = new win.AbortController();
  const storageKey = `soda-terminal:${expectedUserId}:${environmentId}`;
  let id: string | undefined, uncertainCreate = false;
  try { const saved = win.sessionStorage.getItem(storageKey); if (terminalID(saved)) id = saved; else if (saved === 'pending') uncertainCreate = true; } catch { /* persistence unavailable; never weaken authorization */ }
  const remember = (value: string | null) => { try { if (value) win.sessionStorage.setItem(storageKey, value); else win.sessionStorage.removeItem(storageKey); } catch { /* locator only */ } };
  let state = 'idle', disposed = false, generation = 0, retries = 0;
  let socket: WebSocket | undefined, terminal: TerminalView | undefined, fit: (Pick<FitAddon, 'fit'> & ITerminalAddon) | undefined;
  let observer: ResizeObserver | undefined, timer: number | undefined, request: AbortController | undefined;
  const box = doc.createElement('section'); box.className = 'soda-terminal';
  const heading = doc.createElement('h3'); heading.textContent = `Project terminal as ${login}`;
  const warning = doc.createElement('p'); warning.textContent = 'The same tmux session survives navigation and connection loss. Detached or hidden work is retained for 30 minutes, within Soda authentication. Reconnecting alone does not extend retention. Tmux copy-mode holds history; no commands are replayed.';
  const button = (text: string, primary = false) => { const b = doc.createElement('button'); b.type = 'button'; b.className = `ui ${primary ? 'primary' : 'basic'} button`; b.textContent = text; return b; };
  const open = button(id ? 'Reconnect terminal' : uncertainCreate ? 'Find pending terminal' : 'Open terminal', true);
  const end = button('End terminal'), resume = button('Continue working'), away = button('Keep for two hours');
  end.disabled = resume.disabled = away.disabled = !id;
  const status = doc.createElement('p'); status.setAttribute('role', 'status'); status.textContent = 'Not connected.';
  const screen = doc.createElement('div'); screen.className = 'soda-terminal-screen'; screen.hidden = true;
  screen.setAttribute('aria-label', `Terminal for ${login}; Ctrl+Shift+Enter focuses End terminal`);
  const toolbar = doc.createElement('div'); toolbar.className = 'soda-terminal-toolbar';
  toolbar.append(open, end, resume, away, status); box.append(heading, warning, toolbar, screen); root.append(box);
  const live = (n: number) => !disposed && state !== 'stale' && generation === n;
  const detach = (message: string, stale = false) => {
    ++generation; state = stale ? 'stale' : 'closed'; box.classList.remove('is-connected');
    win.clearTimeout(timer); request?.abort(); request = undefined;
    observer?.disconnect(); observer = undefined;
    const old = socket; socket = undefined; if (old && old.readyState < 2) old.close();
    terminal?.dispose(); terminal = undefined; fit = undefined; screen.replaceChildren(); screen.hidden = true;
    open.textContent = id ? 'Reconnect terminal' : uncertainCreate ? 'Find pending terminal' : 'Open terminal';
    open.disabled = stale; end.disabled = resume.disabled = away.disabled = stale || !id;
    status.textContent = message;
  };
  const invalidate = () => detach('Page context changed. No action was replayed; reconnect through a fresh authorized page.', true);
  async function json(path: string, session: {csrf_token: string} | null, body: Record<string, unknown> | null, signal: AbortSignal) {
    const headers: Record<string, string> = {'X-Soda-Expected-User-ID': expectedUserId};
    if (body) { if (!session) throw Error('session'); headers['Content-Type'] = 'application/json'; headers['X-CSRF-Token'] = session.csrf_token; }
    const response = await win.fetch('/-/soda' + path, {method: body ? 'POST' : 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers, ...(body ? {body: JSON.stringify(body)} : {}), signal});
    if (!response.ok) throw Error('Soda request refused');
    return object(await readSodaJSON(response));
  }
  async function session(signal: AbortSignal) {
    const value = await json('/api/session', null, null, signal);
    if (object(value.user).id !== expectedUserId || typeof value.csrf_token !== 'string' || !value.csrf_token || value.forgejo_url !== win.location.origin) throw Error('session');
    return {csrf_token: value.csrf_token};
  }
  async function retention(action: 'end' | 'return' | 'retain', seconds?: 1800 | 7200) {
    if (disposed || state === 'stale' || !id) return;
    const target = id, n = generation, control = new win.AbortController();
    const timeout = win.setTimeout(() => control.abort(), 20000);
    try {
      const current = await session(control.signal);
      if (!live(n) || id !== target) return;
      const result = await json(`/api/environments/${environmentId}/terminal-session`, current, {action, id: target, ...(seconds ? {seconds} : {})}, control.signal);
      if (!live(n)) return;
      if (action === 'end') {
        if (result.ending !== true) throw Error('outcome');
        id = undefined; uncertainCreate = false; remember(null);
        detach('End requested. Native cleanup continues; files and independent services are not undone.');
      } else {
        const retained = object(result.terminal);
        if (retained.id !== target || typeof retained.retain_until !== 'number' || !Number.isSafeInteger(retained.retain_until) || retained.retain_until < 0) throw Error('outcome');
        status.textContent = retained.retain_until ? `Retained until ${new Date(retained.retain_until * 1000).toLocaleTimeString()}, or authentication expiry.` : `Active as ${login}; authentication and native safety leases still apply.`;
      }
    } catch { if (live(n)) status.textContent = 'Terminal lifetime action was not confirmed. No retry or replacement was made.'; }
    finally { win.clearTimeout(timeout); }
  }
  const send = (frame: {type: 'resize'; cols: number; rows: number} | {type: 'input'; data: string}) => {
    if (state !== 'ready' || !socket || socket.readyState !== 1 || socket.bufferedAmount > 65536) { detach('Connection lost or overloaded. Reconnect the existing terminal; input was not replayed.'); return false; }
    try { socket.send(JSON.stringify(frame)); return true; }
    catch { detach('Connection lost. No input was replayed.'); return false; }
  };
  const resize = () => {
    if (!terminal || !fit || state !== 'ready' || !screen.isConnected || screen.clientWidth === 0) return;
    fit.fit();
    if (terminal.cols >= 2 && terminal.cols <= 500 && terminal.rows >= 2 && terminal.rows <= 300) {
      send({type: 'resize', cols: terminal.cols, rows: terminal.rows});
    }
  };
  async function connect(automatic = false) {
    if (disposed || state === 'stale' || state === 'opening' || state === 'ready') return;
    if (automatic && !id) return;
    if (!automatic && (root.closest('[hidden]') || doc.visibilityState === 'hidden' || !doc.hasFocus())) return;
    win.clearTimeout(timer); state = 'opening'; open.disabled = true;
    const n = ++generation; request = new win.AbortController();
    timer = win.setTimeout(() => detach('Terminal connection timed out. Use the existing locator or find the pending terminal; creation was not retried.'), 45000);
    status.textContent = 'Checking your Soda session and original account…';
    try {
      const current = await session(request.signal); if (!live(n)) return;
      const own = await json(`/api/environments/${environmentId}`, null, null, request.signal); if (!live(n)) return;
      const env = object(own.environment);
      if (env.id !== environmentId || env.repository_id !== repositoryId || env.provisioned !== true || own.login !== login) throw Error('membership');
      const metadata = await json(`/api/environments/${environmentId}/terminal-session`, null, null, request.signal); if (!live(n)) return;
      const existing = metadata.terminal === null ? null : object(metadata.terminal);
      if (existing !== null && (!terminalID(existing?.id) || existing.login !== login || existing.repository_id !== repositoryId)) throw Error('terminal metadata');
      if (id && existing?.id !== id) {
        id = undefined; remember(null); detach('That terminal ended or expired. Nothing was created. Open terminal explicitly to start a new shell.'); return;
      }
      if (!id && existing && terminalID(existing.id)) { id = existing.id; uncertainCreate = false; remember(id); }
      if (existing?.state === 'ending' || existing?.state === 'unconfirmed') { detach('Native cleanup is pending or unconfirmed. This slot is reserved; no attachment or replacement was made. Ask the operator to inspect an unconfirmed outcome.'); return; }
      if (!id && uncertainCreate) { detach('Creation outcome remains unconfirmed and no terminal is listed. Ask the operator to inspect; no creation was retried.'); return; }
      if (automatic && !id) { detach('Terminal absent. Nothing was created.'); return; }
      const action = id ? 'attach' : 'create';
      const {Terminal, FitAddon} = await loadRenderer(); if (!live(n)) return;
      terminal = new Terminal({allowProposedApi: true, disableStdin: true, scrollback: 1000, windowOptions: {}, convertEol: false, cols: 80, rows: 24});
      fit = new FitAddon(); terminal.loadAddon(fit);
      for (const code of [0, 1, 2, 8, 52]) terminal.parser.registerOscHandler(code, () => true);
      terminal.attachCustomKeyEventHandler(event => {
        if (event.ctrlKey && event.shiftKey && event.key === 'Enter') { event.preventDefault(); event.stopPropagation(); end.focus(); return false; }
        return true;
      });
      terminal.onData(data => {
        if (data.length > 65536) { detach('Input too large. Nothing was replayed.'); return; }
        const bytes = new TextEncoder().encode(data);
        for (let i = 0; i < bytes.length; i += 16384) if (!send({type: 'input', data: win.btoa(String.fromCharCode(...bytes.subarray(i, i + 16384)))})) break;
      });
      screen.hidden = false; terminal.open(screen); if (screen.clientWidth) fit.fit();
      const cols = Math.max(2, Math.min(500, terminal.cols)), rows = Math.max(2, Math.min(300, terminal.rows));
      const url = new URL(`/-/soda/api/environments/${environmentId}/terminal`, win.location.origin); url.protocol = 'wss:';
      const peer = socket = new win.WebSocket(url);
      peer.onopen = () => {
        if (!live(n)) { peer.close(); return; }
        if (action === 'create') { uncertainCreate = true; remember('pending'); }
        peer.send(JSON.stringify({action, ...(id ? {id} : {}), expected_user_id: expectedUserId, repository_id: repositoryId, csrf_token: current.csrf_token, cols, rows}));
        status.textContent = action === 'create' ? 'Starting the managed terminal…' : 'Attaching the existing terminal…';
      };
      let queuedOutput = 0;
      peer.onmessage = event => {
        if (!live(n) || !terminal) return;
        try {
          if (typeof event.data !== 'string' || event.data.length > 32768) throw Error('frame');
          const frame = object(JSON.parse(event.data)), keys = Object.keys(frame).sort().join(',');
          if (frame.type === 'session' && keys === 'id,type' && state === 'opening' && terminalID(frame.id) && (!id || frame.id === id)) {
            id = frame.id; uncertainCreate = false; remember(id); end.disabled = resume.disabled = away.disabled = false;
          } else if (frame.type === 'ready' && keys === 'type' && state === 'opening' && id) {
            win.clearTimeout(timer); state = 'ready'; retries = 0; terminal.options.disableStdin = false; box.classList.add('is-connected');
            status.textContent = action === 'create' ? `Connected as ${login}.` : `Reconnected as ${login}. Choose Continue working to renew a detached deadline.`;
            if (doc.hasFocus() && doc.visibilityState !== 'hidden' && !root.closest('[hidden]') && (doc.activeElement === doc.body || box.contains(doc.activeElement))) terminal.focus();
            observer = new win.ResizeObserver(resize); observer.observe(screen); resize();
          } else if (frame.type === 'output' && state === 'ready' && keys === 'data,type' && typeof frame.data === 'string') {
            const decoded = win.atob(frame.data);
            if (!decoded.length || decoded.length > 4096 || queuedOutput + decoded.length > 262144) throw Error('output');
            queuedOutput += decoded.length; terminal.write(Uint8Array.from(decoded, c => c.charCodeAt(0)), () => { queuedOutput -= decoded.length; });
          } else if (frame.type === 'closed' && keys === 'reason,type') {
            detach('Attachment ended or unavailable. Reconnect only the existing terminal; no replacement was launched.');
          } else throw Error('frame');
        } catch { detach('Invalid or overloaded stream. No input or creation was replayed.'); }
      };
      peer.onerror = peer.onclose = () => {
        if (!live(n)) return;
        detach('Connection lost. The terminal has bounded retention; no input was replayed.');
        // Bounded attach-only retries, never renewed abandonment or creation.
        if (id && retries < 3) { const wait = 1000 * 2 ** retries++; timer = win.setTimeout(() => connect(true), wait); }
      };
    } catch { if (live(n)) detach('Could not authorize or attach. Sign in again or inspect the existing terminal; nothing was replayed.'); }
  }
  open.addEventListener('click', () => { retries = 0; void connect(); }, {signal: lifetime.signal});
  end.addEventListener('click', () => { if (!end.disabled && !root.closest('[hidden]')) void retention('end'); }, {signal: lifetime.signal});
  resume.addEventListener('click', () => { if (!resume.disabled && !root.closest('[hidden]')) void retention('return'); }, {signal: lifetime.signal});
  away.addEventListener('click', () => { if (!away.disabled && !root.closest('[hidden]')) void retention('retain', 7200); }, {signal: lifetime.signal});
  screen.addEventListener('keydown', event => { if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); } }, {signal: lifetime.signal});
  win.addEventListener('pagehide', invalidate, {signal: lifetime.signal});
  win.addEventListener('pageshow', event => { if (event.persisted) invalidate(); }, {signal: lifetime.signal});
  return {
    get started() { return state !== 'idle'; },
    restore() { if (id) void connect(true); },
    retain() { return retention('retain', 1800); },
    returnToWork() { return retention('return'); },
    invalidate,
    disconnect() { detach('Detached. The terminal has bounded retention.'); },
    dispose() { if (disposed) return; detach('Detached.', true); disposed = true; lifetime.abort(); box.remove(); },
  };
}
