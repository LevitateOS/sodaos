import {test} from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, chmod, readFile} from 'node:fs/promises';
import {createServer} from 'node:tls';
import path from 'node:path';
import {decodeProbeHTTP, readProbeHTTPS} from '../installed/sodaspaces-http';

test('probe HTTP decoding bounds headers/body and refuses ambiguous validators', () => {
  const response = decodeProbeHTTP(Buffer.from('HTTP/1.1 304 Not Modified\r\nETag: "fixture"\r\nSet-Cookie: fixture=unretained\r\n\r\n'));
  assert.equal(response.status, 304); assert.deepEqual(response.headers, {etag: '"fixture"'});
  for (const value of ['not HTTP\r\n\r\n', 'HTTP/1.1 200 OK\r\n folded\r\n\r\n', 'HTTP/1.1 200 OK\r\nETag: one\r\netag: two\r\n\r\n', 'HTTP/1.1 200 OK\r\n\r\n' + 'x'.repeat(65537), 'HTTP/1.1 200 OK\r\nX: ' + 'x'.repeat(16384) + '\r\n\r\n']) assert.throws(() => decodeProbeHTTP(Buffer.from(value)));
});

test('raw HTTPS keeps constrained CA verification, exact paths and bounded responses', {timeout: 30000}, async () => {
  const root = await mkdtemp(path.resolve('.artifacts/probe-https-')); await chmod(root, 0o700);
  const openssl = (...args: string[]) => assert.equal(Bun.spawnSync(['openssl', ...args], {cwd: root, stdout: 'ignore', stderr: 'ignore'}).exitCode, 0);
  openssl('req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-subj', '/CN=Probe root', '-days', '1', '-keyout', 'root.key', '-out', 'root.pem', '-addext', 'nameConstraints=permitted;DNS:localhost,permitted;IP:127.0.0.1/255.255.255.255');
  openssl('req', '-newkey', 'rsa:2048', '-nodes', '-subj', '/CN=localhost', '-keyout', 'leaf.key', '-out', 'leaf.csr', '-addext', 'subjectAltName=DNS:localhost');
  openssl('x509', '-req', '-in', 'leaf.csr', '-CA', 'root.pem', '-CAkey', 'root.key', '-CAcreateserial', '-days', '1', '-copy_extensions', 'copy', '-out', 'leaf.pem');
  const requests: string[] = [];
  const server = createServer({key: await readFile(path.join(root, 'leaf.key')), cert: await readFile(path.join(root, 'leaf.pem'))}, socket => {
    socket.on('error', () => {}); // Test-owned peer may be closed by a bounded refusal.
    socket.once('data', data => {
      const line = data.toString().split('\r\n')[0] || ''; requests.push(line);
      const body = line.includes('/oversized') ? 'x'.repeat(65537) : 'hello';
      socket.end(`HTTP/1.1 200 OK\r\nContent-Length: ${body.length}\r\nCache-Control: max-age=0, must-revalidate\r\nETag: "fixture"\r\nConnection: close\r\n\r\n${body}`);
    });
  });
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
  const address = server.address(); assert(address && typeof address !== 'string');
  const origin = new URL(`https://localhost:${address.port}`), ca = path.join(root, 'root.pem');
  try {
    const response = await readProbeHTTPS(origin, ca, '/api/../%61pi//session');
    assert.equal(response.status, 200); assert.equal(response.body.toString(), 'hello');
    assert.equal(requests[0], 'GET /api/../%61pi//session HTTP/1.1');
    await assert.rejects(readProbeHTTPS(new URL(`https://127.0.0.1:${address.port}`), ca, '/wrong-hostname'));
    await assert.rejects(readProbeHTTPS(origin, path.join(root, 'absent.pem'), '/untrusted'));
    await assert.rejects(readProbeHTTPS(origin, ca, '/oversized'));
    await assert.rejects(readProbeHTTPS(origin, ca, '/', {Authorization: 'not-permitted'}));
  } finally {await new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve()));}
});
