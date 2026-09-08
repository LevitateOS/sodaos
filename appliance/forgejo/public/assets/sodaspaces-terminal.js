// Self-contained optional drawer content. No template selectors, navigation,
// global mounting or native cookie access. The host supplies one mount and
// immutable native-page consistency hints; only the backend grants authority.
const identifier = value => typeof value === 'string' && /^[1-9][0-9]*$/.test(value) && BigInt(value) <= 9223372036854775807n;
const renderer = async () => {
  const [{Terminal}, {FitAddon}] = await Promise.all([
    import('./soda-terminal/xterm.mjs'), import('./soda-terminal/addon-fit.mjs'),
  ]);
  return {Terminal, FitAddon};
};

// Third argument is the renderer loading seam for DOM unit tests; native page
// integration uses only (mount, context). There is exactly one shipped renderer.
export function mountTerminal(root, context, loadRenderer = renderer) {
  const {expectedUserId, repositoryId, environmentId, login} = context;
  if (!root || !identifier(expectedUserId) || !identifier(repositoryId) ||
      !/^p[0-9a-f]{24}$/.test(environmentId) || !/^[a-z][a-z0-9_-]{0,30}$/.test(login) || login === 'root') {
    throw new Error('Invalid terminal mounting context');
  }
  const doc = root.ownerDocument, win = doc.defaultView;
  const abort = new win.AbortController();
  let state = 'idle', socket, terminal, fit, observer, timer, request;
  let disposed = false, generation = 0;
  const box = doc.createElement('section'); box.className = 'soda-terminal';
  const heading = doc.createElement('p'); heading.textContent = `Project terminal as ${login}`;
  const warning = doc.createElement('p'); warning.textContent = 'Switching tabs/apps disconnects this terminal. Closing can interrupt work; completed writes and detached workloads are not undone.';
  const open = doc.createElement('button'); open.type = 'button'; open.textContent = 'Open terminal';
  const disconnect = doc.createElement('button'); disconnect.type = 'button'; disconnect.textContent = 'Disconnect'; disconnect.disabled = true;
  const status = doc.createElement('p'); status.setAttribute('role', 'status'); status.textContent = 'Not connected.';
  const screen = doc.createElement('div'); screen.className = 'soda-terminal-screen'; screen.hidden = true;
  screen.setAttribute('aria-label', `Terminal for ${login}; Ctrl+Shift+Enter focuses Disconnect`);
  box.append(heading, warning, open, disconnect, status, screen); root.append(box);
  const live = n => !disposed && state !== 'stale' && generation === n;
  const stop = (message, stale = false) => {
    ++generation;
    state = stale ? 'stale' : 'closed';
    win.clearTimeout(timer); request?.abort(); request = undefined;
    observer?.disconnect(); observer = undefined;
    const old = socket; socket = undefined;
    if (old && old.readyState < 2) old.close();
    terminal?.dispose(); terminal = undefined; fit = undefined;
    screen.replaceChildren(); screen.hidden = true;
    // An ended/uncertain connection is never replayed on this page.
    open.disabled = true; disconnect.disabled = true; status.textContent = message;
  };
  const invalidate = () => stop('Page context changed. Reload the full repository page before opening another terminal.', true);
  const send = frame => {
    if (state !== 'ready' || !socket || socket.readyState !== 1 || socket.bufferedAmount > 65536) {
      stop('Terminal connection ended or could not keep up. Reload before opening another terminal.'); return false;
    }
    try { socket.send(JSON.stringify(frame)); return true; }
    catch { stop('Terminal connection ended. Reload before opening another terminal.'); return false; }
  };
  const resize = () => {
    if (!terminal || state !== 'ready' || !screen.isConnected || screen.clientWidth === 0) return;
    fit.fit();
    if (terminal.cols >= 2 && terminal.cols <= 500 && terminal.rows >= 2 && terminal.rows <= 300) {
      send({type: 'resize', cols: terminal.cols, rows: terminal.rows});
    }
  };
  open.addEventListener('click', async () => {
    if (state !== 'idle' || disposed || doc.visibilityState === 'hidden' || !doc.hasFocus()) return;
    state = 'opening'; open.disabled = true; disconnect.disabled = false;
    status.textContent = 'Checking your Soda session…'; const n = ++generation;
    request = new win.AbortController();
    timer = win.setTimeout(() => stop('Terminal authorization timed out. Reload before trying again.'), 20000);
    try {
      const response = await win.fetch('/-/soda/api/session', {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': expectedUserId}, signal: request.signal});
      if (!live(n)) return;
      if (!response.ok) throw new Error('session');
      const session = await response.json();
      if (!live(n)) return;
      if (session.user?.id !== expectedUserId || typeof session.csrf_token !== 'string' || !session.csrf_token || session.forgejo_url !== win.location.origin) throw new Error('session');
      const details = await win.fetch(`/-/soda/api/environments/${environmentId}`, {credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: {'X-Soda-Expected-User-ID': expectedUserId}, signal: request.signal});
      if (!live(n)) return;
      if (!details.ok) throw new Error('membership');
      const own = await details.json();
      if (!live(n)) return;
      if (own.environment?.id !== environmentId || own.environment.repository_id !== repositoryId || !own.environment.provisioned || own.login !== login) throw new Error('membership');
      const {Terminal, FitAddon} = await loadRenderer();
      if (!live(n)) return;
      terminal = new Terminal({allowProposedApi: true, disableStdin: true, scrollback: 1000, windowOptions: {}, convertEol: false, cols: 80, rows: 24});
      fit = new FitAddon(); terminal.loadAddon(fit);
      // Block terminal-driven clipboard, hyperlinks and page-title effects.
      for (const code of [0, 1, 2, 8, 52]) terminal.parser.registerOscHandler(code, () => true);
      terminal.attachCustomKeyEventHandler(event => {
        if (event.ctrlKey && event.shiftKey && event.key === 'Enter') { event.preventDefault(); event.stopPropagation(); disconnect.focus(); return false; }
        return true;
      });
      terminal.onData(data => {
        if (data.length > 65536) { stop('Terminal input was too large. Reload before opening another terminal.'); return; }
        const bytes = new TextEncoder().encode(data);
        for (let i = 0; i < bytes.length; i += 16384) {
          const part = bytes.subarray(i, i + 16384);
          if (!send({type: 'input', data: win.btoa(String.fromCharCode(...part))})) break;
        }
      });
      screen.hidden = false; terminal.open(screen);
      if (screen.clientWidth > 0) fit.fit();
      const cols = Math.max(2, Math.min(500, terminal.cols)), rows = Math.max(2, Math.min(300, terminal.rows));
      const url = new URL(`/-/soda/api/environments/${environmentId}/terminal`, win.location.origin);
      url.protocol = 'wss:';
      const current = socket = new win.WebSocket(url);
      current.onopen = () => {
        if (!live(n)) { current.close(); return; }
        current.send(JSON.stringify({expected_user_id: expectedUserId, repository_id: repositoryId, csrf_token: session.csrf_token, cols, rows}));
        status.textContent = 'Authorizing existing project account…';
      };
      let queuedOutput = 0;
      current.onmessage = event => {
        if (!live(n)) return;
        try {
          if (typeof event.data !== 'string' || event.data.length > 32768) throw new Error('frame');
          const frame = JSON.parse(event.data), keys = Object.keys(frame).sort().join(',');
          if (frame.type === 'ready' && keys === 'type' && state === 'opening') {
            win.clearTimeout(timer); state = 'ready'; terminal.options.disableStdin = false;
            status.textContent = `Connected as ${login}.`; terminal.focus();
            observer = new win.ResizeObserver(resize); observer.observe(screen); resize();
          } else if (frame.type === 'output' && state === 'ready' && keys === 'data,type' && typeof frame.data === 'string') {
            const decoded = win.atob(frame.data);
            if (!decoded.length || decoded.length > 4096 || queuedOutput + decoded.length > 262144) throw new Error('output');
            queuedOutput += decoded.length;
            terminal.write(Uint8Array.from(decoded, c => c.charCodeAt(0)), () => { queuedOutput -= decoded.length; });
          } else if (frame.type === 'closed' && keys === 'reason,type') {
            stop('Terminal ended or could not be opened. Reload the repository page before opening another terminal.');
          } else throw new Error('frame');
        } catch { stop('Invalid or overloaded terminal stream. Reload the repository page.'); }
      };
      current.onerror = current.onclose = () => { if (live(n)) stop('Terminal connection ended. Reload the repository page before opening another terminal.'); };
    } catch {
      if (live(n)) stop('Could not authorize or load the terminal. Reload the repository page and sign in again.');
    }
  }, {signal: abort.signal});
  disconnect.addEventListener('click', () => {
    if (state === 'ready') send({type: 'close'});
    stop('Disconnected. Native cleanup is not confirmed by this screen; completed work is not undone. Reload before opening another terminal.');
  }, {signal: abort.signal});
  screen.addEventListener('keydown', event => {
    // After xterm receives Escape, prevent the containing native dialog closing.
    if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); }
  }, {signal: abort.signal});
  win.addEventListener('blur', invalidate, {signal: abort.signal});
  win.addEventListener('pagehide', invalidate, {signal: abort.signal});
  doc.addEventListener('visibilitychange', () => { if (doc.visibilityState === 'hidden') invalidate(); }, {signal: abort.signal});
  win.addEventListener('pageshow', event => { if (event.persisted) invalidate(); }, {signal: abort.signal});
  return {
    invalidate,
    disconnect: () => stop('Disconnected. Reload the repository page before opening another terminal.'),
    dispose() { if (disposed) return; stop('Disconnected.', true); disposed = true; abort.abort(); box.remove(); },
  };
}
