// SPDX-License-Identifier: Apache-2.0
// Native Chromium focus/visibility, without Playwright's always-focused override.
// A private Unix WebSocket bridges the browser's pipe: no TCP debugger listener,
// protocol logs, screenshots, cookie seeding or browser-security bypasses.
import {spawn} from 'node:child_process';
import {createServer} from 'node:http';
import {chmod} from 'node:fs/promises';
import {createRequire} from 'node:module';
import {StringDecoder} from 'node:string_decoder';
import path from 'node:path';
const require = createRequire(new URL('../../cockpit/package.json', import.meta.url));
// Reuse the existing Cockpit jsdom lock's WebSocket dependency; no new manifest.
const {WebSocketServer} = createRequire(require.resolve('jsdom'))('ws');

export async function launchNativeBrowser(chromium, run, home) {
  const socket = path.join(run, 'cdp.sock');
  // Linux sockaddr_un limit; refuse instead of falling back to a TCP listener.
  if (Buffer.byteLength(socket) > 103) throw new Error('private browser socket path too long');
  const server = createServer();
  const ws = new WebSocketServer({server, maxPayload: 64 * 1024 * 1024});
  let child, browser, peer;
  let closing = false;
  const close = async () => {
    if (closing) return;
    closing = true;
    try {
      if (browser) {
        const cdp = await browser.newBrowserCDPSession();
        await cdp.send('Browser.close');
      }
    } catch { /* Pipe closure is expected during browser shutdown. */ }
    peer?.terminate();
    ws.close();
    server.close();
    if (child && child.exitCode === null && child.signalCode === null) {
      const exited = new Promise(resolve => child.once('exit', resolve));
      child.kill('SIGTERM'); // Only this invocation's browser, never shared processes.
      let timer;
      await Promise.race([exited, new Promise(resolve => {
        timer = setTimeout(() => { child.kill('SIGKILL'); resolve(); }, 10000);
      })]);
      clearTimeout(timer);
    }
  };
  try {
    await new Promise((resolve, reject) => {
      server.once('error', reject);
      server.listen(socket, resolve);
    });
    await chmod(socket, 0o600);
    child = spawn(chromium.executablePath(), ['--headless=new', '--remote-debugging-pipe',
      '--no-first-run', '--no-default-browser-check', '--disable-background-networking',
      '--disable-component-update', '--disable-sync', '--password-store=basic',
      `--user-data-dir=${path.join(run, 'profile')}`, 'about:blank'], {
      env: {...process.env, HOME: home, XDG_CONFIG_HOME: path.join(home, '.config'), XDG_DATA_HOME: path.join(home, '.local/share')},
      stdio: ['ignore', 'ignore', 'ignore', 'pipe', 'pipe'],
    });
    child.on('error', () => peer?.terminate());
    child.stdio[3].on('error', () => peer?.terminate());
    child.stdio[4].on('error', () => peer?.terminate());
    const decoder = new StringDecoder('utf8');
    let pending = '';
    child.stdio[4].on('data', chunk => {
      pending += decoder.write(chunk);
      if (pending.length > 64 * 1024 * 1024) { peer?.terminate(); return; }
      let end;
      while ((end = pending.indexOf('\0')) !== -1) {
        const message = pending.slice(0, end);
        pending = pending.slice(end + 1);
        if (peer?.readyState === 1) peer.send(message);
      }
    });
    ws.on('connection', connection => {
      if (peer || closing) { connection.terminate(); return; }
      peer = connection;
      peer.on('error', () => {});
      peer.on('message', message => child.stdio[3].write(Buffer.concat([Buffer.from(message), Buffer.from([0])])));
    });
    browser = await chromium.connectOverCDP(`ws+unix://${socket}:/cdp`, {noDefaults: true, timeout: 30000});
    const context = browser.contexts()[0];
    if (!context) throw new Error('native default context missing');
    return {context, close};
  } catch {
    await close();
    throw new Error('native browser attachment failed');
  }
}
