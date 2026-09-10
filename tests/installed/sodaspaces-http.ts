// Raw-path, anonymous HTTPS observations for the installed probe. Curl uses the
// supplied CA (including its name constraints), not Bun's BoringSSL verifier.
import assert from 'node:assert/strict';

export function decodeProbeHTTP(data: Buffer) {
  assert(data.length <= 81920, 'HTTPS observation exceeds limit');
  const end = data.indexOf('\r\n\r\n');
  assert(end >= 0 && end <= 16384, 'Invalid HTTPS response headers');
  const lines = data.subarray(0, end).toString('latin1').split('\r\n');
  const status = /^HTTP\/1\.[01] ([1-5][0-9]{2})(?: .*)?$/.exec(lines.shift() || '');
  assert(status?.[1], 'Invalid HTTPS response status');
  const headers: Record<string, string> = {};
  for (const line of lines) {
    const match = /^([!#$%&'*+.^_`|~0-9A-Za-z-]+):[ \t]*([^\r\n\0]*)$/.exec(line);
    assert(match?.[1] && match[2] !== undefined, 'Invalid HTTPS response header');
    const key = match[1].toLowerCase();
    // Only public cache validators are needed; never retain Set-Cookie.
    if (!['cache-control', 'etag', 'last-modified'].includes(key)) continue;
    assert(headers[key] === undefined, 'Ambiguous cache validator');
    headers[key] = match[2];
  }
  const body = data.subarray(end + 4);
  assert(body.length <= 65536, 'HTTPS body exceeds limit');
  return {status: Number(status[1]), headers, body};
}

export async function readProbeHTTPS(origin: URL, caFile: string, rawPath: string, headers: Record<string, string> = {}) {
  assert(origin.protocol === 'https:' && !origin.username && !origin.password);
  assert(rawPath.startsWith('/') && !/[\r\n\0]/.test(rawPath));
  const args = ['curl', '-q', '--silent', '--show-error', '--noproxy', '*', '--proto', '=https', '--http1.1',
    '--path-as-is', '--max-time', '15', '--max-filesize', '65536', '--cacert', caFile, '--dump-header', '-'];
  for (const [key, value] of Object.entries(headers)) {
    assert(['If-None-Match', 'If-Modified-Since'].includes(key) && !/[\r\n\0]/.test(value));
    args.push('--header', key + ': ' + value);
  }
  args.push('--url', origin.origin + rawPath);
  const child = Bun.spawn(args, {stdin: 'ignore', stdout: 'pipe', stderr: 'ignore', timeout: 16000, killSignal: 'SIGKILL'});
  const reader = child.stdout.getReader(), chunks: Buffer[] = [];
  let size = 0;
  try {
    for (;;) {
      const next = await reader.read(); if (next.done) break;
      size += next.value.byteLength; assert(size <= 81920, 'HTTPS observation exceeds limit');
      chunks.push(Buffer.from(next.value));
    }
    assert.equal(await child.exited, 0, 'Verified HTTPS observation failed');
    return decodeProbeHTTP(Buffer.concat(chunks));
  } finally {
    reader.releaseLock();
    if (child.exitCode === null) child.kill('SIGKILL');
    await child.exited;
  }
}
