// SPDX-License-Identifier: Apache-2.0
// Native Chromium focus/visibility, without Playwright's always-focused override.
// A private Unix WebSocket bridges the browser's pipe: no TCP debugger listener,
// protocol logs, screenshots, cookie seeding or browser-security bypasses.
import type { BrowserType, Browser } from 'playwright';
import {closeSync} from 'node:fs';
import {chmod} from 'node:fs/promises';
import path from 'node:path';

export async function launchNativeBrowser(chromium: Pick<BrowserType, "executablePath" | "connectOverCDP">, run: string, home: string) {
  const socket = path.join(run, 'cdp.sock');
  // Linux sockaddr_un limit; refuse instead of falling back to a TCP listener.
  if (Buffer.byteLength(socket) > 103) throw new Error('private browser socket path too long');
  let server: Bun.Server<undefined> | undefined;
  let child: Bun.Subprocess<'ignore', 'ignore', 'ignore'> | undefined, browser: Browser | undefined, peer: Bun.ServerWebSocket<undefined> | undefined;
  let input: Bun.FileSink | undefined, output: ReadableStreamDefaultReader<Uint8Array> | undefined;
  let inputFD: number | null | undefined, outputFD: number | null | undefined;
  let closing: Promise<void> | undefined;
  const close = () => closing ||= (async () => {
    let shutdownTimer: ReturnType<typeof setTimeout> | undefined;
    try {
      if (browser) await Promise.race([
        (async () => {
          const cdp = await browser?.newBrowserCDPSession();
          await cdp?.send('Browser.close');
        })(),
        new Promise<void>(resolve => { shutdownTimer = setTimeout(resolve, 2000); }),
      ]);
    } catch { /* Pipe closure is expected during browser shutdown. */ }
    finally { clearTimeout(shutdownTimer); }
    peer?.terminate();
    await server?.stop(true);
    try {await output?.cancel();} catch { /* Browser pipe may already have ended. */ }
    try {await input?.end();} catch { /* Browser pipe may already have ended. */ }
    // Bun.file borrows these descriptors; this invocation still owns their closure.
    if (typeof inputFD === 'number') {closeSync(inputFD); inputFD = undefined;}
    if (typeof outputFD === 'number') {closeSync(outputFD); outputFD = undefined;}
    if (child && child.exitCode === null) {
      const exited = child.exited;
      child.kill('SIGTERM'); // Only this invocation's browser, never shared processes.
      let timer: ReturnType<typeof setTimeout> | undefined;
      await Promise.race([exited, new Promise<void>(resolve => {
        timer = setTimeout(() => { child?.kill('SIGKILL'); resolve(); }, 10000);
      })]);
      clearTimeout(timer);
    }
  })();
  try {
    server = Bun.serve<undefined>({
      unix: socket,
      fetch(request, server) {
        if (peer || closing || !input || new URL(request.url).pathname !== '/cdp') return new Response(null, {status: 409});
        return server.upgrade(request, {data: undefined}) ? undefined : new Response(null, {status: 400});
      },
      websocket: {
        maxPayloadLength: 64 * 1024 * 1024, idleTimeout: 0,
        open(connection) {
          if (peer || closing) {connection.terminate(); return;}
          peer = connection;
        },
        async message(connection, message) {
          try {
          input?.write(typeof message === 'string' ? message + '\0' : Buffer.concat([message, Buffer.from([0])]));
          await input?.flush();
          } catch {connection.terminate();}
        },
      },
    });
    await chmod(socket, 0o600);
    child = Bun.spawn([chromium.executablePath(), '--headless=new', '--remote-debugging-pipe',
      '--no-first-run', '--no-default-browser-check', '--disable-background-networking',
      '--disable-component-update', '--disable-sync', '--password-store=basic',
      `--user-data-dir=${path.join(run, 'profile')}`, 'about:blank'], {
      env: {...process.env, HOME: home, XDG_CONFIG_HOME: path.join(home, '.config'), XDG_DATA_HOME: path.join(home, '.local/share')},
      stdio: ['ignore', 'ignore', 'ignore', 'socket-fd', 'socket-fd'],
      onExit: () => peer?.terminate(),
    });
    [, , , inputFD, outputFD] = child.stdio;
    if (typeof inputFD !== 'number' || typeof outputFD !== 'number') throw Error('Browser pipes unavailable');
    input = Bun.file(inputFD).writer();
    output = Bun.file(outputFD).stream().getReader();
    const reader = output;
    const decoder = new TextDecoder('utf8');
    let pending = '';
    void (async () => {
      try {
        for (;;) {
          const {value, done} = await reader.read();
          if (done) {peer?.terminate(); return;}
          pending += decoder.decode(value, {stream: true});
          if (pending.length > 64 * 1024 * 1024) {
            pending = ''; peer?.terminate(); child?.kill('SIGTERM'); return;
          }
          let end;
          while ((end = pending.indexOf('\0')) !== -1) {
            const message = pending.slice(0, end);
            pending = pending.slice(end + 1);
            if (peer?.readyState === 1) peer.send(message);
          }
        }
      } catch {peer?.terminate();}
    })();
    browser = await chromium.connectOverCDP(`ws+unix://${socket}:/cdp`, {noDefaults: true, timeout: 30000});
    const context = browser.contexts()[0];
    if (!context) throw new Error('native default context missing');
    return {context, close};
  } catch {
    await close();
    throw new Error('native browser attachment failed');
  }
}
