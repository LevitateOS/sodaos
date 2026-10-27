// Protected native runtime lifecycle and authenticated HTTP transport.
import assert from 'node:assert/strict';
import {randomBytes} from 'node:crypto';
import {spawn} from 'node:child_process';
import {createServer, request as httpRequest} from 'node:http';
export function runtimeChecks(context) {
  const {root, command, childEnv} = context;
  function occurrence() {
    const bytes = randomBytes(16);
    bytes.writeUIntBE(Date.now(), 0, 6);
    bytes[6] = (bytes[6] & 15) | 112;
    bytes[8] = (bytes[8] & 63) | 128;
    const h = bytes.toString('hex');
    return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
  }

  async function start(bundleFile, data, owner) {
    const reservation = createServer();
    await new Promise((done, reject) => {
      reservation.once('error', reject);
      reservation.listen(0, '127.0.0.1', done);
    });
    const port = reservation.address().port;
    await new Promise((done) => reservation.close(done));
    // A nonzero fixed listener preserves the checked HTTPS CAN_AUTH_ORIGIN.
    // The production gateway reconstructs Worker URLs at that origin; this local
    // HTTP client qualifies that host-ingress boundary, not TLS/browser behavior.
    const child = spawn(
      command('can-soda'),
      [
        'serve',
        '--bundle',
        bundleFile,
        '--data',
        data,
        '--owner',
        owner,
        '--host',
        '127.0.0.1',
        '--port',
        String(port),
      ],
      {cwd: root, env: childEnv, detached: true, stdio: ['ignore', 'pipe', 'pipe']}
    );
    const exited = new Promise((done, reject) => {
      child.once('error', reject);
      child.once('close', done);
    });
    child.stderr.resume();
    let buffer = '';
    const ready = new Promise((done, reject) => {
      child.stdout.on('data', (chunk) => {
        buffer += chunk.toString();
        if (buffer.length > 65536) {
          reject(Error('Runtime output exceeded its bound.'));
          return;
        }
        for (;;) {
          const end = buffer.indexOf('\n');
          if (end < 0) break;
          const line = buffer.slice(0, end);
          buffer = buffer.slice(end + 1);
          try {
            const message = JSON.parse(line);
            if (message.ok && message.command === 'serve') done(message.origin);
          } catch {}
        }
      });
      exited.then(() => reject(Error('Actual application runtime exited before readiness.')), reject);
    });
    let stopped = false;
    const stop = async () => {
      if (stopped) return;
      stopped = true;
      if (child.exitCode === null && child.signalCode === null) {
        // Signal only the canonical supervisor; it drains the gateway/workerd
        // through its private IPC. A group SIGTERM would kill that child early.
        child.kill('SIGTERM');
        let timer;
        const code = await Promise.race([
          exited,
          new Promise((done) => {
            timer = setTimeout(() => done('timeout'), 15000);
          }),
        ]);
        clearTimeout(timer);
        if (code === 'timeout') {
          process.kill(-child.pid, 'SIGKILL');
          await exited;
          throw Error('Application did not retire cleanly.');
        }
        assert(code === 0 || code === null, 'Application shutdown failed.');
      }
    };
    let timer;
    try {
      const origin = await Promise.race([
        ready,
        new Promise((_, reject) => {
          timer = setTimeout(() => reject(Error('Application readiness timed out.')), 45000);
        }),
      ]);
      return {port, origin, stop};
    } catch (error) {
      await stop();
      throw error;
    } finally {
      clearTimeout(timer);
    }
  }

  function transport(server, jar = new Map()) {
    return async (path, body, headers = {}) => {
      const bytes = body === undefined ? undefined : Buffer.from(JSON.stringify(body));
      return new Promise((done, reject) => {
        const request = httpRequest(
          {
            host: '127.0.0.1',
            port: server.port,
            path,
            method: body === undefined ? 'GET' : 'POST',
            headers: {
              host: new URL(server.origin).host,
              origin: server.origin,
              accept: 'application/json',
              ...(jar.size ? {cookie: [...jar].map(([name, value]) => name + '=' + value).join('; ')} : {}),
              ...(bytes ? {'content-type': 'application/json', 'content-length': bytes.length} : {}),
              ...headers,
            },
          },
          (response) => {
            try {
              for (const cookie of response.headers['set-cookie'] ?? []) {
                const [pair] = cookie.split(';');
                const index = pair.indexOf('=');
                const name = pair.slice(0, index),
                  value = pair.slice(index + 1);
                assert(['can_session', 'can_presession'].includes(name), 'Unexpected authentication cookie.');
                assert(
                  /;\s*Secure(?:;|$)/i.test(cookie) &&
                    /;\s*HttpOnly(?:;|$)/i.test(cookie) &&
                    /;\s*SameSite=Lax(?:;|$)/i.test(cookie) &&
                    !/;\s*Domain=/i.test(cookie),
                  'Cookie security contract changed.'
                );
                if (value) jar.set(name, value);
                else jar.delete(name);
              }
            } catch (error) {
              response.resume();
              reject(error);
              return;
            }
            const chunks = [];
            let size = 0;
            response.on('data', (chunk) => {
              size += chunk.length;
              if (size > 4 * 1024 * 1024) request.destroy(Error('HTTP output exceeded its bound.'));
              else chunks.push(chunk);
            });
            response.once('error', reject);
            response.once('end', () => {
              const text = Buffer.concat(chunks).toString('utf8');
              let value;
              try {
                value = JSON.parse(text);
              } catch {}
              done({status: response.statusCode, headers: response.headers, text, json: value});
            });
          }
        );
        request.setTimeout(30000, () => request.destroy(Error('Application HTTP request timed out.')));
        request.once('error', reject);
        request.end(bytes);
      });
    };
  }

  return {occurrence, start, transport};
}
