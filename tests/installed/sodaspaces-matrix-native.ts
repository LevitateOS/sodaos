// Fixed-operation SSH observations and terminal probes. Never provisioning,
// installation, cleanup, host sockets, environment dumps or arbitrary host commands.
import assert from 'node:assert/strict';
import {lstat} from 'node:fs/promises';
import type {Page, WebSocket} from 'playwright';
import {object} from './sodaspaces-input';
import {terminalID} from '../../frontend/spaces/sodaspaces-api';
import type {MatrixInput, MatrixProject} from './sodaspaces-matrix-input';
import type {MatrixFacts, MatrixSession} from './sodaspaces-workspace-journey';
export async function restrictedText(file: string, limit: number) {
  assert(file.startsWith('/')); const stat = await lstat(file);
  assert(stat.isFile() && !(stat.mode & 0o077) && stat.size <= limit);
  return Bun.file(file).text();
}
const quote = (text: string) => "'" + text.replaceAll("'", "'\\''") + "'";
export function sshArgs(request: MatrixInput, project: MatrixProject, actor: number) {
  const alias = project.ssh[actor]; assert(alias);
  return ['-F', request.ssh_config, '-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=yes', '-o', 'PermitLocalCommand=no', '-o', 'ClearAllForwardings=yes', '-o', 'ForwardAgent=no', '-o', 'ForwardX11=no', '-o', 'RemoteCommand=none', '-o', 'ConnectTimeout=10', '-o', 'ControlMaster=no', '-o', 'ControlPath=none', '-o', 'IdentityAgent=none', '-o', 'IdentitiesOnly=yes', alias];
}
export function matrixSSH(request: MatrixInput, project: MatrixProject, actor: number, command: string) {
  const child = Bun.spawnSync(['ssh', '-T', ...sshArgs(request, project, actor), command], {stdin: 'ignore', stdout: 'pipe', stderr: 'pipe', timeout: 15000, maxBuffer: 16384});
  assert.equal(child.exitCode, 0, 'Matrix SSH observation failed'); return child.stdout.toString().trim();
}
export async function inspectMatrixProcess(request: MatrixInput, project: MatrixProject, actor: number, session: MatrixSession, facts: MatrixFacts, ended: boolean,
  read: (command: string) => string = command => matrixSSH(request, project, actor, command)) {
  assert(Number.isSafeInteger(facts.pid) && facts.pid > 0 && /^\d+$/.test(facts.start));
  assert(terminalID(session.id) && session.environment === project.environment && session.actor === request.actors[actor]);
  const code = `import os,pwd,json,subprocess,stat
from pathlib import Path
identifier=${JSON.stringify(session.id)}
unit='soda-terminal-'+identifier+'.service'
def read(path):
 try: return Path(path).read_text()
 except FileNotFoundError: return None
def inspect(path):
 try: return os.stat(path,follow_symlinks=False)
 except FileNotFoundError: return None
p=read('/proc/${facts.pid}/stat')
record=inspect('/run/soda-terminals/'+identifier)
sock=inspect('/run/soda-terminals/'+identifier+'/screen/socket')
events=read('/sys/fs/cgroup/system.slice/'+unit+'/cgroup.events')
properties='LoadState,ActiveState,Description,FragmentPath'
result=subprocess.run(['systemctl','show',unit,'--property='+properties],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,timeout=3)
assert len(result.stdout)<4096
fields=dict(line.split('=',1) for line in result.stdout.decode('ascii').splitlines())
assert set(fields)==set(properties.split(',')) and (result.returncode==0 or fields['LoadState']=='not-found')
assert fields['LoadState']=='not-found' or (fields['Description']=='Soda terminal '+identifier and fields['FragmentPath']=='/run/systemd/transient/'+unit)
print(json.dumps(dict(login=pwd.getpwuid(os.getuid()).pw_name,start=p.rsplit(') ',1)[1].split()[19] if p else None,record=record is not None,socket=sock is not None,owned_socket=bool(sock and stat.S_ISSOCK(sock.st_mode) and sock.st_uid==os.getuid()),populated=dict(row.split() for row in events.splitlines()).get('populated') if events is not None else '0',state=fields['ActiveState'])))`;
  const deadline = Date.now() + (ended ? 15000 : 1);
  do {
    const found = object(JSON.parse(read('python3 -I -c ' + quote(code))));
    assert.equal(found.login, facts.login, 'SSH and browser must observe the same original account');
    if (ended ? found.start !== facts.start && found.record === false && found.socket === false && found.populated === '0' && (found.state === 'inactive' || found.state === 'failed')
      : found.start === facts.start && found.record === true && found.owned_socket === true && found.populated === '1' && found.state === 'active') return;
    if (!ended) break;
    await new Promise(resolve => setTimeout(resolve, 250));
  } while (Date.now() < deadline);
  throw Error(ended ? 'Exact shell cleanup not independently confirmed' : 'Original shell continuity lost');
}
export function observeMatrixShell(page: Page) {
  const buffers = new Map<string, string>(), sizes = new Map<string, {bytes: number; chunks: number}>();
  const decoders = new Map<string, TextDecoder>(), owners = new Map<string, WebSocket>();
  const handlers = new Map<WebSocket, (event: {payload: string | Buffer}) => void>();
  const observe = (socket: WebSocket) => {
    if (!new URL(socket.url()).pathname.endsWith('/terminal')) return;
    let id = '';
    const receive = ({payload}: {payload: string | Buffer}) => {
      try {
        const frame = object(JSON.parse(String(payload)));
        if (frame.type === 'session' && terminalID(frame.id)) {id = frame.id; owners.set(id, socket); decoders.set(id, new TextDecoder());}
        if (frame.type === 'output' && id && owners.get(id) === socket && typeof frame.data === 'string') {
          const bytes = Buffer.from(frame.data, 'base64'), size = sizes.get(id) || {bytes: 0, chunks: 0};
          buffers.set(id, ((buffers.get(id) || '') + (decoders.get(id)?.decode(bytes, {stream: true}) || '')).slice(-16384));
          sizes.set(id, {bytes: size.bytes + bytes.length, chunks: size.chunks + 1});
        }
      } catch { /* Framing assertions belong to the scenario's exact wire observer. */ }
    };
    handlers.set(socket, receive); socket.on('framereceived', receive);
  };
  page.on('websocket', observe);
  return {
    text: (id: string) => buffers.get(id) || '',
    metrics: (id: string) => sizes.get(id) || {bytes: 0, chunks: 0},
    clear: (id: string) => {buffers.delete(id); sizes.delete(id);},
    async shell(session: MatrixSession, initialize: boolean): Promise<MatrixFacts> {
      const marker = 'SODA_FACT_' + crypto.randomUUID().replaceAll('-', ''); buffers.delete(session.id);
      const code = `import os,pwd,json; p=os.getppid(); print(${JSON.stringify(marker + ':')}+json.dumps(dict(pid=p,start=open('/proc/%d/stat'%p).read().split()[21],login=pwd.getpwuid(os.getuid()).pw_name,marker=os.environ.get('SODA_MATRIX_MARKER',''),tty=os.isatty(0),term=os.environ.get('TERM',''))))`;
      const command = (initialize ? 'export SODA_MATRIX_MARKER=' + quote(session.name) + '; ' : '') + 'python3 -I -c ' + quote(code);
      const screen = page.locator('.soda-workspace-terminal:visible .xterm-helper-textarea'); await screen.focus();
      await page.keyboard.insertText(command); await page.keyboard.press('Enter');
      const deadline = Date.now() + 15000;
      do {
        const wire = (buffers.get(session.id) || '').replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '').replaceAll('\r', '');
        const line = wire.split('\n').find(line => line.startsWith(marker + ':'));
        if (line) {
          const facts = object(JSON.parse(line.slice(marker.length + 1))); buffers.delete(session.id);
          const {pid, start, login, marker: retained, tty, term} = facts;
          assert(typeof pid === 'number' && Number.isSafeInteger(pid) && pid > 0 && typeof start === 'string' && /^\d+$/.test(start));
          assert(typeof login === 'string' && typeof retained === 'string' && typeof tty === 'boolean' && typeof term === 'string' && /^[a-zA-Z0-9-]{1,64}$/.test(term));
          return {pid, start, login, marker: retained, tty, term};
        }
        await new Promise(resolve => setTimeout(resolve, 100));
      } while (Date.now() < deadline);
      throw Error('Native output facts not confirmed');
    },
    dispose() {page.off('websocket', observe); for (const [socket, receive] of handlers) socket.off('framereceived', receive); buffers.clear(); sizes.clear(); decoders.clear(); owners.clear();},
  };
}
