// Loopback-only development fixture: real browser components, synthetic APIs and IO.
import {watch, type FSWatcher} from 'node:fs';
import {resolve, basename} from 'node:path';
import {parseArgs} from 'node:util';
import payload from '../internal/nativebuild/forgejo-payload.json';
import {buildForgejoAssets, buildForgejoModule} from './build-forgejo.ts';
import {scenarioModel, scenarios, type Scenario, type Model} from './fixtures/spaces-scenarios.ts';
import {object} from '../frontend/spaces/sodaspaces-api.ts';
import type {ServerWebSocket} from 'bun';
const root = resolve(import.meta.dir, '..');
type Fault = 'healthy' | 'slow' | 'offline' | 'expired' | 'join-failed' | 'create-failed';
type Review = ReturnType<typeof scenarioModel> & {scenario: Scenario; generation: string; fault: Fault; history: Map<string, string>; peers: Set<ServerWebSocket<SocketData>>};
type SocketData = {review: Review; url: string; peer?: InstanceType<Model['Socket']>; terminal?: string; line: string};
const faults: Fault[] = ['healthy', 'slow', 'offline', 'expired', 'join-failed', 'create-failed'];
const json = (data: unknown, status = 200) => Response.json(data, {status, headers: {'Cache-Control': 'no-store'}});
export async function startSpacesPreview(port = 24455, liveReload = false) {
  const out = resolve(root, '.artifacts/spaces-dev/modules');
  await buildForgejoAssets(out);
  let client = await buildForgejoModule(resolve(root, 'scripts/fixtures/spaces-review-client.ts'), 'public/assets/spaces-review-client.js');
  let revision = crypto.randomUUID();
  const watchers: FSWatcher[] = [];
  const clients = new Map<string, Review>();
  function fresh(origin: string, scenario: Scenario): Review {
    return {...scenarioModel(origin, scenario), scenario, generation: crypto.randomUUID(), fault: scenario === 'expired' ? 'expired' : scenario === 'join-failed' ? 'join-failed' : 'healthy', history: new Map(), peers: new Set()};
  }
  function sendOutput(ws: ServerWebSocket<SocketData>, text: string) {
    const bytes = Buffer.from(text);
    for (let offset = 0; offset < bytes.length; offset += 16384) {
      ws.send(JSON.stringify({type: 'output', data: bytes.subarray(offset, offset + 16384).toString('base64')}));
    }
  }
  function output(ws: ServerWebSocket<SocketData>, text: string) {
    if (ws.data.terminal) {
      const history = ws.data.review.history;
      history.set(ws.data.terminal, ((history.get(ws.data.terminal) || '') + text).slice(-32768));
    }
    sendOutput(ws, text);
  }
  function command(line: string): string {
    const input = line.trim();
    if (!input) return '';
    if (input === 'help') return 'Simulated shell — no commands execute on this computer.\r\nTry: pwd, ls, git status, bun test, echo hello, clear\r\n';
    if (input === 'pwd') return '/home/alice/project\r\n';
    if (input === 'ls') return 'README.md  package.json  src/  tests/\r\n';
    if (input === 'git status') return "On branch main\r\nChanges not staged for commit:\r\n  modified: src/app.ts\r\n";
    if (input === 'bun test' || input === 'npm test') return '\x1b[32m✓\x1b[0m project navigation\r\n\x1b[32m✓\x1b[0m terminal layout\r\n\r\n2 simulated tests passed\r\n';
    if (input === 'clear') return '\x1b[2J\x1b[H';
    if (input.startsWith('echo ')) return input.slice(5) + '\r\n';
    return 'Fixture shell: command not simulated. Type help.\r\n';
  }
  const server = Bun.serve<SocketData>({hostname: '127.0.0.1', port,
    async fetch(req, server) {
      const url = new URL(req.url), origin = server.url.origin;
      // No proxying, credentials, filesystem APIs or remote listeners.
      if (url.origin !== origin || req.headers.get('host') !== server.url.host) return json({error: 'Unexpected fixture host'}, 403);
      const cookie = req.headers.get('cookie')?.match(/(?:^|;\s*)spaces_fixture=([a-f0-9-]{36})(?:;|$)/)?.[1];
      let review = cookie ? clients.get(cookie) : undefined;
      if (!review) {
        if (url.pathname !== '/' || req.method !== 'GET') return json({error: 'Open the fixture homepage first'}, 401);
        if (clients.size >= 32) return json({error: 'Restart the fixture to clear old development sessions'}, 503);
        const id = crypto.randomUUID(); review = fresh(origin, 'welcome'); clients.set(id, review);
        return new Response(Bun.file(resolve(root, 'scripts/fixtures/spaces-review.html')), {headers: {'Content-Type': 'text/html', 'Cache-Control': 'no-store', 'Set-Cookie': `spaces_fixture=${id}; HttpOnly; SameSite=Strict; Path=/`}});
      }
      if (!['GET', 'HEAD'].includes(req.method) && req.headers.get('origin') !== origin) return json({error: 'Fixture controls require same-origin requests'}, 403);
      if (url.pathname === '/-/soda/login' && req.method === 'GET') {
        review.model.setStatus(200); review.fault = 'healthy';
        return Response.redirect(origin + '/?soda-view=spaces', 303);
      }
      if (url.pathname.startsWith('/_fixture/')) {
        if (url.pathname === '/_fixture/review.css') return new Response(Bun.file(resolve(root, 'scripts/fixtures/spaces-review.css')), {headers: {'Content-Type': 'text/css', 'Cache-Control': 'no-store'}});
        if (url.pathname === '/_fixture/state' && req.method === 'GET') return json({scenario: review.scenario, fault: review.fault, generation: review.generation, revision, layout: review.layout});
        if (req.method !== 'POST' || Number(req.headers.get('content-length')) > 4096) return json({error: 'Invalid fixture control'}, 400);
        const body = object(await req.json());
        if (url.pathname === '/_fixture/reset' && scenarios.includes(body.scenario as Scenario)) {
          for (const peer of review.peers) peer.close();
          review = fresh(origin, body.scenario as Scenario); clients.set(cookie!, review);
          return json({scenario: review.scenario, fault: review.fault, generation: review.generation, revision, layout: review.layout});
        }
        if (url.pathname === '/_fixture/fault' && faults.includes(body.fault as Fault)) {
          review.fault = body.fault as Fault;
          review.model.setStatus(review.fault === 'expired' ? 401 : review.fault === 'offline' ? 503 : 200);
          review.model.setJoinFailure(review.fault === 'join-failed');
          review.model.setCreateOutcome(review.fault === 'create-failed' ? 'rejected' : 'confirmed');
          return json({ok: true});
        }
        return json({error: 'Unknown fixture control'}, 400);
      }
      if (url.pathname.startsWith('/-/soda/api/')) {
        if (req.headers.get('upgrade') === 'websocket') {
          if (req.headers.get('origin') !== origin || review.fault === 'expired' || review.fault === 'offline') return json({error: 'Fixture socket unavailable'}, 403);
          if (!/^\/-\/soda\/api\/environments\/p[0-9a-f]{24}\/terminal$/.test(url.pathname)) return json({error: 'Unknown fixture socket'}, 404);
          return server.upgrade(req, {data: {review, url: url.href, line: ''}}) ? undefined : json({error: 'Socket upgrade failed'}, 400);
        }
        if (req.headers.get('X-Soda-Expected-User-ID') !== '1') return json({error: {code: 'identity_mismatch'}}, 403);
        if (review.fault === 'slow') await Bun.sleep(1200);
        if (req.method !== 'GET' && req.headers.get('X-CSRF-Token') !== 'synthetic-only') return json({error: {code: 'invalid_csrf'}}, 403);
        const body = req.method === 'GET' ? null : object(await req.json());
        return review.model.request(url.href, req.method, body);
      }
      if (url.pathname === '/assets/spaces-review-client.js') return new Response(client, {headers: {'Content-Type': 'text/javascript', 'Cache-Control': 'no-store'}});
      const source = Object.entries(payload).find(([key]) => key === 'public' + url.pathname)?.[1];
      if (source) {
        const path = source.startsWith('@build/forgejo-js/') ? resolve(out, basename(source)) : source.startsWith('@build/terminal-assets/') ? resolve(root, '.artifacts/browser-terminal/vendor', basename(source)) : resolve(root, source);
        return new Response(Bun.file(path), {headers: {'Cache-Control': 'no-store'}});
      }
      if (url.pathname === '/' && req.method === 'GET') return new Response(Bun.file(resolve(root, 'scripts/fixtures/spaces-review.html')), {headers: {'Content-Type': 'text/html', 'Cache-Control': 'no-store'}});
      return json({error: 'This route is outside the local Spaces fixture'}, 404);
    },
    websocket: {
      maxPayloadLength: 32769,
      open(ws) {
        const peer = ws.data.peer = new ws.data.review.model.Socket(ws.data.url);
        ws.data.review.peers.add(ws);
        peer.onmessage = event => {
          ws.send(event.data);
          const frame = object(JSON.parse(event.data));
          if (frame.type === 'ready') {
            const saved = ws.data.terminal ? ws.data.review.history.get(ws.data.terminal) : undefined;
            if (saved) sendOutput(ws, saved);
            else output(ws, '\x1b[90mSoda development fixture · simulated terminal\x1b[0m\r\nType help for sample commands.\r\n\r\nalice@project:~$ ');
          }
        };
        peer.onclose = () => {if (ws.readyState === 1) ws.close();};
      },
      message(ws, message) {
        try {
          const text = String(message), frame = object(JSON.parse(text));
          if (typeof frame.id === 'string') ws.data.terminal = frame.id;
          if (frame.type === 'input' && typeof frame.data === 'string') {
            for (const ch of Buffer.from(frame.data, 'base64').toString('utf8')) {
              if (ch === '\r' || ch === '\n') {output(ws, '\r\n' + command(ws.data.line) + 'alice@project:~$ '); ws.data.line = '';}
              else if (ch === '\x7f') {if (ws.data.line) {ws.data.line = ws.data.line.slice(0, -1); output(ws, '\b \b');}}
              else if (ch === '\x03') {ws.data.line = ''; output(ws, '^C\r\nalice@project:~$ ');}
              else if (ch >= ' ' && ws.data.line.length < 4096) {ws.data.line += ch; output(ws, ch);}
            }
          } else ws.data.peer?.send(text);
        } catch {ws.close(1008, 'Invalid fixture frame');}
      },
      close(ws) {ws.data.review.peers.delete(ws); if (!ws.data.peer?.closed) ws.data.peer?.close();}
    },
    error() {return json({error: 'Invalid fixture request'}, 400);}
  });
  if (liveReload) {
    let pending: ReturnType<typeof setTimeout> | undefined;
    let builds = Promise.resolve();
    for (const directory of ['frontend/spaces', 'frontend/tailnet', 'assets/branding/forgejo', 'assets/branding/theme', 'scripts/fixtures']) {
      watchers.push(watch(resolve(root, directory), {recursive: true}, (_event, name) => {
        if (!name || !/\.(?:ts|css|html|svg)$/.test(name)) return;
        clearTimeout(pending);
        pending = setTimeout(() => {
          builds = builds.then(async () => {
            await buildForgejoAssets(out);
            client = await buildForgejoModule(resolve(root, 'scripts/fixtures/spaces-review-client.ts'), 'public/assets/spaces-review-client.js');
            revision = crypto.randomUUID();
            console.log('Spaces rebuilt; browser reload requested. Mock project state retained.');
          }).catch(error => console.error('Spaces rebuild failed:', error instanceof Error ? error.message : 'build error'));
        }, 150);
      }));
    }
  }
  return {url: server.url, stop(close = true) {for (const watcher of watchers) watcher.close(); server.stop(close);}};
}
if (import.meta.main) {
  const {values} = parseArgs({args: Bun.argv.slice(2), options: {port: {type: 'string', default: '24455'}}});
  const port = Number(values.port);
  if (!Number.isInteger(port) || port < 1024 || port > 65535) throw Error('Choose a local port from 1024–65535');
  const server = await startSpacesPreview(port, true);
  console.log(`Spaces development fixture: ${server.url}\nLocal mock APIs and simulated terminals. No VM or shell execution.\nFrontend edits rebuild and reload automatically; mock project state stays in memory.`);
}
