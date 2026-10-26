// Local real-product HTTP/PG integration. The Go journey fixture owns native
// test authentication and drops one socket only after the command commits.
// This runner supplies the maintained caller and its real durable evidence.
import assert from 'node:assert/strict';
import {request} from 'node:http';
import {mkdir, open, readFile, realpath} from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {sha256} from './validate';
import type {FactoryCommandIntent, FactoryCommandRuntime} from './phases/factory-command';

process.umask(0o077);
const args = process.argv.slice(2);
assert.deepEqual(
  args.filter((_, i) => i % 2 === 0),
  ['--module', '--out']
);
assert.equal(args.length, 4);
const moduleFile = args[1]!,
  out = args[3]!;
assert(moduleFile.startsWith('/home/') && (await realpath(moduleFile)) === moduleFile);
assert(out.startsWith('/home/') && path.resolve(out) === out);
assert.equal(await realpath(path.dirname(out)), path.dirname(out));
const bytes = await Bun.stdin.text();
assert(Buffer.byteLength(bytes) <= 65536);
const input = JSON.parse(bytes) as {
  format: number;
  origin: string;
  owner: string;
  repository: number;
  command_id: string;
  headers: {get: Record<string, string>; post: Record<string, string>};
};
const origin = new URL(input.origin);
assert(input.format === 1 && origin.protocol === 'http:' && origin.hostname === '127.0.0.1' && origin.port !== '');
assert(origin.origin === input.origin && origin.pathname === '/' && !origin.username && !origin.password);
assert(/^[1-9][0-9]*$/.test(input.owner) && /^[a-f0-9]{32}$/.test(input.command_id));
assert(Number.isSafeInteger(input.repository) && input.repository > 0);
for (const method of ['get', 'post'] as const) {
  assert(input.headers[method] && Object.values(input.headers[method]).every((value) => typeof value === 'string'));
  assert.equal(new Headers(input.headers[method]).get('Origin'), 'https://soda.test');
}
const moduleBytes = await readFile(moduleFile);
const built = (await import(pathToFileURL(moduleFile).href)) as Pick<
  typeof import('./phases/index'),
  'issueFactoryCommand'
>;
await mkdir(out, {mode: 0o700});
async function save(relative: string, value: unknown) {
  assert(
    /^[A-Za-z0-9._/-]{1,256}$/.test(relative) &&
      !relative.startsWith('/') &&
      relative.split('/').every((part) => part !== '' && part !== '.' && part !== '..')
  );
  const file = path.join(out, relative),
    parent = path.dirname(file);
  await mkdir(parent, {recursive: true, mode: 0o700});
  const handle = await open(file, 'wx', 0o600);
  try {
    await handle.writeFile(JSON.stringify(value, null, 2) + '\n');
    await handle.sync();
  } finally {
    await handle.close();
  }
  const directory = await open(parent, 'r');
  try {
    await directory.sync();
  } finally {
    await directory.close();
  }
}
const intent: FactoryCommandIntent = {
  label: 'lost-http-pause',
  repository: input.repository,
  owner: input.owner,
  method: 'POST',
  path: `repositories/${input.repository}/factory/actions`,
  body: {command_id: input.command_id, action: 'pause'},
};
// node:http sends exactly one request: no redirect handling, pooled connection
// recovery or library retry can hide another mutation after the socket drops.
function http(method: 'GET' | 'POST', route: string, body?: unknown): Promise<{status: number; body: unknown}> {
  assert(route === intent.path || route === 'factory/commands/' + input.command_id);
  return new Promise((resolve, reject) => {
    const data = body === undefined ? undefined : JSON.stringify(body);
    const req = request(
      new URL('/api/' + route, origin),
      {
        method,
        agent: false,
        headers: {
          ...input.headers[method === 'GET' ? 'get' : 'post'],
          ...(data === undefined ? {} : {'Content-Length': Buffer.byteLength(data)}),
        },
      },
      (res) => {
        const chunks: Buffer[] = [];
        let size = 0;
        res.on('data', (chunk: Buffer) => {
          size += chunk.length;
          if (size > 65536) res.destroy(Error('bounded_command_reply'));
          else chunks.push(chunk);
        });
        res.on('error', reject);
        res.on('end', () => {
          try {
            resolve({status: res.statusCode!, body: JSON.parse(Buffer.concat(chunks).toString())});
          } catch (error) {
            reject(error);
          }
        });
      }
    );
    req.setTimeout(15000, () => req.destroy(Error('command_http_timeout')));
    req.on('error', reject);
    req.end(data);
  });
}
let sends = 0,
  lookups = 0;
const runtime: FactoryCommandRuntime = {
  prepare: () => save('commands/' + intent.label + '.request.json', intent),
  send: async () => {
    assert.deepEqual(
      JSON.parse(await readFile(path.join(out, 'commands/' + intent.label + '.request.json'), 'utf8')),
      intent
    );
    sends++;
    return http('POST', intent.path, intent.body);
  },
  lookup: async (route) => {
    assert.equal(route, 'factory/commands/' + input.command_id);
    lookups++;
    return http('GET', route);
  },
  save,
};
const receipt = await built.issueFactoryCommand(intent, runtime);
assert.equal(receipt.command_id, input.command_id);
assert.equal(receipt.paused, true);
assert.deepEqual(receipt.runs, []);
const saved = JSON.parse(await readFile(path.join(out, 'commands/' + intent.label + '.receipt.json'), 'utf8'));
assert.equal(saved.recovered, true);
assert.equal(saved.binding.id, input.command_id);
assert.equal(sends, 1);
assert.equal(lookups, 1);
// Real filesystem re-entry, after the response loss and successful lookup.
// The same durable intent must refuse before sending or looking up again.
await assert.rejects(
  built.issueFactoryCommand(intent, runtime),
  (error) => (error as NodeJS.ErrnoException).code === 'EEXIST'
);
assert.equal(sends, 1);
assert.equal(lookups, 1);
assert.deepEqual(await readFile(moduleFile), moduleBytes);
await save('result.json', {
  format: 1,
  passed: true,
  check: 'normal-api-lost-http-response',
  module_sha256: sha256(moduleBytes),
  runner_sha256: sha256(await readFile(import.meta.path)),
  command_id: input.command_id,
  command_digest: saved.binding.digest,
  request_durable_before_send: true,
  canonical_outcome_recovered: true,
  existing_intent_refused_before_second_send: true,
  sends,
  lookups,
  authority: 'ephemeral local native-auth callback fixture; not appliance or provider authority',
  provider_calls: 0,
  guest_mutations: 0,
  qualification_claimed: false,
});
console.log(JSON.stringify({passed: true, command_id: input.command_id, sends, lookups}));
