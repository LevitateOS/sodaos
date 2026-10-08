// Fixed-operation SSH observations and terminal probes. Never provisioning,
// installation, cleanup, host sockets, environment dumps or arbitrary host commands.
import assert from 'node:assert/strict';
import {lstat} from 'node:fs/promises';
import type {Page, WebSocket} from 'playwright';
import {object} from './sodaspaces-input';
import {terminalID} from '../../frontend/spaces/sodaspaces-terminal-response';
import type {MatrixInput, MatrixProject} from './sodaspaces-matrix-input';
import type {MatrixFacts, MatrixSession} from './sodaspaces-journey-evidence';
export async function restrictedText(file: string, limit: number) {
  assert(file.startsWith('/'));
  const stat = await lstat(file);
  assert(stat.isFile() && !(stat.mode & 0o077) && stat.size <= limit);
  return Bun.file(file).text();
}
const quote = (text: string) => "'" + text.replaceAll("'", "'\\''") + "'";
export function matrixShellCommand(marker: string, name: string, initialize: boolean) {
  const code = `(p=$PPID; s=$(cat /proc/$p/stat) || exit 1; set -f; set -- $s; set +f; [ $# -ge 22 ] || exit 1; [ -t 0 ] && tty=true || tty=false; printf '%s{"pid": %s, "start": "%s", "login": "%s", "marker": "%s", "tty": %s, "term": "%s"}\\n' ${JSON.stringify(marker + ':')} "$p" "\${22}" "$(id -un)" "$SODA_MATRIX_MARKER" "$tty" "$TERM")`;
  return (initialize ? 'export SODA_MATRIX_MARKER=' + quote(name) + '; ' : '') + 'sh -c ' + quote(code);
}
export function sshArgs(request: MatrixInput, project: MatrixProject, actor: number) {
  const alias = project.ssh[actor];
  assert(alias);
  return [
    '-F',
    request.ssh_config,
    '-o',
    'BatchMode=yes',
    '-o',
    'StrictHostKeyChecking=yes',
    '-o',
    'PermitLocalCommand=no',
    '-o',
    'ClearAllForwardings=yes',
    '-o',
    'ForwardAgent=no',
    '-o',
    'ForwardX11=no',
    '-o',
    'RemoteCommand=none',
    '-o',
    'ConnectTimeout=10',
    '-o',
    'ControlMaster=no',
    '-o',
    'ControlPath=none',
    '-o',
    'IdentityAgent=none',
    '-o',
    'IdentitiesOnly=yes',
    alias,
  ];
}
export function matrixSSH(request: MatrixInput, project: MatrixProject, actor: number, command: string) {
  const child = Bun.spawnSync(['ssh', '-T', ...sshArgs(request, project, actor), command], {
    stdin: 'ignore',
    stdout: 'pipe',
    stderr: 'pipe',
    timeout: 15000,
    maxBuffer: 16384,
  });
  assert.equal(child.exitCode, 0, 'Matrix SSH observation failed');
  return child.stdout.toString().trim();
}
export async function inspectMatrixProcess(
  request: MatrixInput,
  project: MatrixProject,
  actor: number,
  session: MatrixSession,
  facts: MatrixFacts,
  ended: boolean,
  read: (command: string) => string = (command) => matrixSSH(request, project, actor, command)
) {
  assert(session.environment === project.environment && session.actor === request.actors[actor]);
  return inspectNativeTerminal(session, facts, ended, read);
}
// Same fixed native observation via an explicitly selected operator transport.
export async function inspectNativeTerminal(
  session: MatrixSession,
  facts: MatrixFacts,
  ended: boolean,
  read: (command: string) => string
) {
  assert(Number.isSafeInteger(facts.pid) && facts.pid > 0 && /^\d+$/.test(facts.start));
  assert(terminalID(session.id));
  const code = `identifier=${JSON.stringify(session.id)}
unit="soda-terminal-$identifier.service"
if { [ ! -e /proc/${facts.pid}/stat ] && [ ! -L /proc/${facts.pid}/stat ]; } || { [ -L /proc/${facts.pid}/stat ] && [ ! -e /proc/${facts.pid}/stat ]; }; then
  start=null
elif [ -f /proc/${facts.pid}/stat ] && [ -r /proc/${facts.pid}/stat ]; then
  p=$(cat /proc/${facts.pid}/stat)
  rest=\`echo "$p" | sed 's/.*) //'\`
  [ "$rest" != "$p" ] || exit 1
  set -f; set -- $rest; set +f
  [ $# -ge 20 ] || exit 1
  start="\\"\${20}\\""
else
  exit 1
fi
if [ -e /run/soda-terminals/$identifier ] || [ -L /run/soda-terminals/$identifier ]; then record=true; else record=false; fi
sock_path="/run/soda-terminals/$identifier/screen/socket"
if [ -e "$sock_path" ] || [ -L "$sock_path" ]; then socket=true; else socket=false; fi
if [ ! -L "$sock_path" ] && [ -S "$sock_path" ] && [ "$(stat -c %u "$sock_path")" = "$(id -u)" ]; then owned=true; else owned=false; fi
ev_path="/sys/fs/cgroup/system.slice/$unit/cgroup.events"
if { [ ! -e "$ev_path" ] && [ ! -L "$ev_path" ]; } || { [ -L "$ev_path" ] && [ ! -e "$ev_path" ]; }; then
  populated="\\"0\\""
elif [ -f "$ev_path" ] && [ -r "$ev_path" ]; then
  populated=$(awk '$1=="populated"{v=$2} END{print v}' "$ev_path")
  if [ -n "$populated" ]; then populated="\\"$populated\\""; else populated=null; fi
else
  exit 1
fi
out=$(timeout 3 systemctl show "$unit" --property=LoadState,ActiveState,Description,FragmentPath 2>/dev/null); rc=$?
[ "$(printf '%s\\n' "$out" | wc -c)" -lt 4096 ] || exit 1
printf '%s' "$out" | tr -d '\\t\\n' | LC_ALL=C grep -q '[^ -~]' && exit 1
[ "$(printf '%s\\n' "$out" | wc -l)" = 4 ] || exit 1
load=$(printf '%s\\n' "$out" | sed -n 's/^LoadState=//p')
active=$(printf '%s\\n' "$out" | sed -n 's/^ActiveState=//p')
desc=$(printf '%s\\n' "$out" | sed -n 's/^Description=//p')
frag=$(printf '%s\\n' "$out" | sed -n 's/^FragmentPath=//p')
[ "$(printf '%s\\n' "$out" | grep -c '^LoadState=')" = 1 ] || exit 1
[ "$(printf '%s\\n' "$out" | grep -c '^ActiveState=')" = 1 ] || exit 1
[ "$(printf '%s\\n' "$out" | grep -c '^Description=')" = 1 ] || exit 1
[ "$(printf '%s\\n' "$out" | grep -c '^FragmentPath=')" = 1 ] || exit 1
{ [ "$rc" = 0 ] || [ "$load" = not-found ]; } || exit 1
{ [ "$load" = not-found ] || { [ "$desc" = "Soda terminal $identifier" ] && [ "$frag" = "/run/systemd/transient/$unit" ]; }; } || exit 1
printf '{"login": "%s", "start": %s, "record": %s, "socket": %s, "owned_socket": %s, "populated": %s, "state": "%s"}' "$(id -un)" "$start" "$record" "$socket" "$owned" "$populated" "$active"`;
  const deadline = Date.now() + (ended ? 15000 : 1);
  do {
    const found = object(JSON.parse(read('sh -c ' + quote(code))));
    assert.equal(found.login, facts.login, 'Native observer and browser must observe the same original account');
    if (
      ended
        ? found.start !== facts.start &&
          found.record === false &&
          found.socket === false &&
          found.populated === '0' &&
          (found.state === 'inactive' || found.state === 'failed')
        : found.start === facts.start &&
          found.record === true &&
          found.owned_socket === true &&
          found.populated === '1' &&
          found.state === 'active'
    )
      return;
    if (!ended) break;
    await new Promise((resolve) => setTimeout(resolve, 250));
  } while (Date.now() < deadline);
  throw Error(ended ? 'Exact shell cleanup not independently confirmed' : 'Original shell continuity lost');
}
export function observeMatrixShell(page: Page) {
  const buffers = new Map<string, string>(),
    sizes = new Map<string, {bytes: number; chunks: number}>();
  const decoders = new Map<string, TextDecoder>(),
    owners = new Map<string, WebSocket>();
  const handlers = new Map<
    WebSocket,
    {sent: (event: {payload: string | Buffer}) => void; receive: (event: {payload: string | Buffer}) => void}
  >();
  const observe = (socket: WebSocket) => {
    if (!new URL(socket.url()).pathname.endsWith('/terminal')) return;
    let id = '';
    const sent = ({payload}: {payload: string | Buffer}) => {
      try {
        const frame = object(JSON.parse(String(payload)));
        if ((frame.action === 'create' || frame.action === 'attach') && terminalID(frame.id)) id = frame.id;
      } catch {
        /* scenario validates framing */
      }
    };
    const receive = ({payload}: {payload: string | Buffer}) => {
      try {
        const frame = object(JSON.parse(String(payload)));
        if (frame.type === 'ready' && id) {
          owners.set(id, socket);
          decoders.set(id, new TextDecoder());
        }
        if (frame.type === 'output' && id && owners.get(id) === socket && typeof frame.data === 'string') {
          const bytes = Buffer.from(frame.data, 'base64'),
            size = sizes.get(id) || {bytes: 0, chunks: 0};
          buffers.set(
            id,
            ((buffers.get(id) || '') + (decoders.get(id)?.decode(bytes, {stream: true}) || '')).slice(-16384)
          );
          sizes.set(id, {bytes: size.bytes + bytes.length, chunks: size.chunks + 1});
        }
      } catch {
        /* Framing assertions belong to the scenario's exact wire observer. */
      }
    };
    handlers.set(socket, {sent, receive});
    socket.on('framesent', sent);
    socket.on('framereceived', receive);
  };
  page.on('websocket', observe);
  return {
    text: (id: string) => buffers.get(id) || '',
    metrics: (id: string) => sizes.get(id) || {bytes: 0, chunks: 0},
    clear: (id: string) => {
      buffers.delete(id);
      sizes.delete(id);
    },
    async shell(session: MatrixSession, initialize: boolean): Promise<MatrixFacts> {
      const marker = 'SODA_FACT_' + crypto.randomUUID().replaceAll('-', '');
      buffers.delete(session.id);
      const command = matrixShellCommand(marker, session.name, initialize);
      const screen = page.locator('.soda-workspace-terminal:visible .xterm-helper-textarea');
      await screen.focus();
      await page.keyboard.insertText(command);
      await page.keyboard.press('Enter');
      const deadline = Date.now() + 15000;
      do {
        const wire = (buffers.get(session.id) || '').replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '').replaceAll('\r', '');
        const line = wire.split('\n').find((line) => line.startsWith(marker + ':'));
        if (line) {
          const facts = object(JSON.parse(line.slice(marker.length + 1)));
          buffers.delete(session.id);
          const {pid, start, login, marker: retained, tty, term} = facts;
          assert(
            typeof pid === 'number' &&
              Number.isSafeInteger(pid) &&
              pid > 0 &&
              typeof start === 'string' &&
              /^\d+$/.test(start)
          );
          assert(
            typeof login === 'string' &&
              typeof retained === 'string' &&
              typeof tty === 'boolean' &&
              typeof term === 'string' &&
              /^[a-zA-Z0-9-]{1,64}$/.test(term)
          );
          return {pid, start, login, marker: retained, tty, term};
        }
        await new Promise((resolve) => setTimeout(resolve, 100));
      } while (Date.now() < deadline);
      throw Error('Native output facts not confirmed');
    },
    dispose() {
      page.off('websocket', observe);
      for (const [socket, callbacks] of handlers) {
        socket.off('framesent', callbacks.sent);
        socket.off('framereceived', callbacks.receive);
      }
      buffers.clear();
      sizes.clear();
      decoders.clear();
      owners.clear();
    },
  };
}
