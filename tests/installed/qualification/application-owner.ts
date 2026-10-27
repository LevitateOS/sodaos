// Canlang owns application Identity and records. This client uses its ordinary
// HTTP/MCP contract; a Soda/Forgejo account is never substituted for its owner.
import assert from 'node:assert/strict';
import {createHash, createHmac} from 'node:crypto';
import {request as httpRequest} from 'node:http';
import {request as httpsRequest} from 'node:https';

export interface ApplicationRecordWitness {
  readonly format: 1;
  readonly owner_team: string;
  readonly operation_id: string;
  readonly id: string;
  readonly version: string;
  readonly label: string;
  readonly record_sha256: string;
}
export interface ApplicationJournal {
  readonly save: (relative: string, value: unknown) => Promise<void>;
  readonly load?: (relative: string) => Promise<unknown | null>;
}
interface RecordView {
  id: string;
  version: number | string;
  data: {label: string};
}
interface CreateIntent {
  format: 1;
  owner_team: string;
  operation_id: string;
  label: string;
  before: {id: string; sha256: string}[];
  automatic_retry: false;
}
const digest = (value: unknown) => createHash('sha256').update(JSON.stringify(value)).digest('hex');
const teamID = /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/;
const operationID = /^[a-f0-9]{8}-[a-f0-9]{4}-7[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/;
const object = (value: unknown): Record<string, any> => {
  assert(value && typeof value === 'object' && !Array.isArray(value), 'application_response_unconfirmed');
  return value as Record<string, any>;
};
export interface ApplicationHttpRequest {
  readonly route: string;
  readonly method: 'GET' | 'POST';
  readonly body: string | null;
  readonly headers: Record<string, string>;
}
export interface ApplicationHttpResponse {
  readonly status: number;
  readonly text: string;
  readonly set_cookie: readonly string[];
}

export async function openApplicationOwner(input: {
  readonly origin: string;
  readonly owner_team: string;
  readonly credentials: {readonly email: string; readonly password: string};
  /** Explicit public appliance CA; HTTPS verification remains enabled. */
  readonly ca?: Uint8Array;
  /** Fixed SSH/namespace HTTP transport for an isolated restored container.
   * Requests still use the same origin, TLS and ordinary owner protocol. */
  readonly transport?: (request: ApplicationHttpRequest) => Promise<ApplicationHttpResponse>;
}) {
  const origin = new URL(input.origin);
  assert(
    origin.origin === input.origin &&
      !origin.username &&
      !origin.password &&
      (origin.protocol === 'https:' || (origin.protocol === 'http:' && origin.hostname === '127.0.0.1')),
    'application_origin_invalid'
  );
  assert(
    teamID.test(input.owner_team) && input.credentials.email && input.credentials.password,
    'application_owner_invalid'
  );
  const cookies = new Map<string, string>();
  // A single native request has no redirect or retry behavior. Response bodies
  // remain in memory, bounded, and are never copied into the public journal.
  const request = (route: string, method: 'GET' | 'POST', body?: unknown, extra: Record<string, string> = {}) =>
    new Promise<{status: number; text: string}>((resolve, reject) => {
      assert(route.startsWith('/') && !route.startsWith('//') && !route.includes('#'));
      const data = body === undefined ? undefined : JSON.stringify(body);
      const headers = {
        accept: 'application/json',
        origin: input.origin,
        cookie: [...cookies].map(([key, value]) => key + '=' + value).join('; '),
        ...(data === undefined
          ? {}
          : {'content-type': 'application/json', 'content-length': String(Buffer.byteLength(data))}),
        ...extra,
      };
      const retainCookies = (values: readonly string[]) => {
        for (const header of values) {
          const pair = header.split(';', 1)[0]!,
            at = pair.indexOf('=');
          if (at > 0) cookies.set(pair.slice(0, at), pair.slice(at + 1));
        }
      };
      if (input.transport) {
        void input
          .transport({route, method, body: data ?? null, headers})
          .then((response) => {
            assert(
              Number.isSafeInteger(response.status) &&
                response.status >= 100 &&
                response.status <= 599 &&
                Buffer.byteLength(response.text) <= 2 << 20 &&
                response.set_cookie.length <= 64,
              'application_namespace_response_invalid'
            );
            retainCookies(response.set_cookie);
            resolve({status: response.status, text: response.text});
          })
          .catch(() => reject(Error('application_transport_uncertain')));
        return;
      }
      const native = origin.protocol === 'https:' ? httpsRequest : httpRequest;
      const req = native(
        new URL(route, origin),
        {
          method,
          agent: false,
          ...(input.ca === undefined ? {} : {ca: Buffer.from(input.ca)}),
          headers,
        },
        (res) => {
          retainCookies(res.headers['set-cookie'] ?? []);
          const chunks: Buffer[] = [];
          let size = 0;
          res.on('data', (bytes: Buffer) => {
            size += bytes.length;
            if (size > 2 << 20) res.destroy(Error('application_response_too_large'));
            else chunks.push(bytes);
          });
          res.on('error', () => reject(Error('application_transport_uncertain')));
          res.on('end', () => resolve({status: res.statusCode!, text: Buffer.concat(chunks).toString('utf8')}));
        }
      );
      req.setTimeout(20000, () => req.destroy(Error('application_transport_timeout')));
      req.on('error', () => reject(Error('application_transport_uncertain')));
      req.end(data);
    });
  const json = async (route: string, method: 'GET' | 'POST', body?: unknown, headers?: Record<string, string>) => {
    const response = await request(route, method, body, headers);
    assert.equal(response.status, 200, 'application_owner_request_denied');
    try {
      return object(JSON.parse(response.text));
    } catch {
      throw Error('application_response_unconfirmed');
    }
  };
  const pre = await json('/auth/login', 'GET');
  assert(typeof pre.preSessionToken === 'string' && pre.preSessionToken, 'application_login_unavailable');
  await json('/auth/login', 'POST', {...input.credentials, _presession: pre.preSessionToken});
  const session = cookies.get('can_session');
  assert(session, 'application_session_missing');
  const csrf = createHmac('sha256', decodeURIComponent(session)).update('can-csrf-v1').digest('base64url');
  const headers = {'x-csrf-token': csrf};
  const teams = await json('/auth/teams', 'GET');
  assert(
    Array.isArray(teams.teams) && teams.teams.filter((row: any) => row.team_id === input.owner_team).length === 1,
    'application_owner_team_unavailable'
  );
  const selected = await json('/auth/select-team', 'POST', {team: input.owner_team, _csrf: csrf}, headers);
  assert(selected.team_id === input.owner_team, 'application_owner_team_changed');
  const grant = await json('/mcp/grants', 'POST', {client_id: 'soda-qualification'}, headers);
  assert(typeof grant.token === 'string' && grant.token, 'application_read_grant_unavailable');
  async function records(): Promise<RecordView[]> {
    const reply = await json(
      '/mcp',
      'POST',
      {
        jsonrpc: '2.0',
        id: 1,
        method: 'tools/call',
        params: {name: 'SodaRuntime.Entry.read', arguments: {}},
      },
      {authorization: 'Bearer ' + grant.token, accept: 'application/json, text/event-stream'}
    );
    assert(!reply.error && reply.result && !reply.result.isError, 'application_record_read_denied');
    const result = object(reply.result);
    let data = result.structuredContent;
    if (data === undefined) {
      assert(Array.isArray(result.content), 'application_record_reply_invalid');
      const text = result.content.find((entry: any) => entry.type === 'text');
      assert(text && typeof text.text === 'string', 'application_record_reply_invalid');
      try {
        data = JSON.parse(text.text);
      } catch {
        throw Error('application_record_reply_invalid');
      }
    }
    const rows = object(data).records;
    assert(Array.isArray(rows) && rows.length <= 100, 'application_record_reply_invalid');
    const ids = new Set<string>();
    for (const row of rows) {
      assert(
        row &&
          typeof row.id === 'string' &&
          row.id &&
          !ids.has(row.id) &&
          row.data &&
          typeof row.data.label === 'string' &&
          /^[1-9][0-9]*$/.test(String(row.version)),
        'application_record_reply_invalid'
      );
      ids.add(row.id);
    }
    return rows;
  }
  function witness(row: RecordView, operation_id: string): ApplicationRecordWitness {
    return {
      format: 1,
      owner_team: input.owner_team,
      operation_id,
      id: row.id,
      version: String(row.version),
      label: row.data.label,
      record_sha256: digest(row),
    };
  }
  async function readRecord(expected: ApplicationRecordWitness) {
    assert(
      expected.format === 1 &&
        expected.owner_team === input.owner_team &&
        operationID.test(expected.operation_id) &&
        expected.id === expected.operation_id,
      'application_record_owner_changed'
    );
    const matches = (await records()).filter((row) => row.id === expected.id);
    assert(matches.length === 1, 'application_record_missing');
    assert.deepEqual(witness(matches[0]!, expected.operation_id), expected, 'application_record_changed');
    return expected;
  }
  return {
    async ensureRecord(selected: {operation_id: string; label: string}, journal: ApplicationJournal) {
      assert(
        operationID.test(selected.operation_id) &&
          selected.label.trim() === selected.label &&
          selected.label.length > 0 &&
          selected.label.length <= 200,
        'application_record_input_invalid'
      );
      let intent: CreateIntent;
      const retained = await journal.load?.('application/create.request.json');
      if (retained !== undefined && retained !== null) {
        intent = object(retained) as unknown as CreateIntent;
        assert(
          intent.format === 1 &&
            intent.owner_team === input.owner_team &&
            intent.operation_id === selected.operation_id &&
            intent.label === selected.label &&
            intent.automatic_retry === false &&
            Array.isArray(intent.before),
          'application_create_intent_changed'
        );
        assert(
          intent.before.every((row) => typeof row.id === 'string' && /^[a-f0-9]{64}$/.test(row.sha256)),
          'application_create_intent_invalid'
        );
      } else {
        const before = await records();
        // A new collector may follow a failed, retired collector that already
        // committed this exact selected operation. Adopt its normal readback;
        // never repeat the create POST or replace it with a new operation ID.
        const existing = before.find((row) => row.id === selected.operation_id);
        if (existing) {
          assert(existing.data.label === selected.label, 'application_existing_operation_changed');
          const adopted = witness(existing, selected.operation_id);
          await journal.save('application/adopted.json', adopted);
          return adopted;
        }
        intent = {
          format: 1,
          owner_team: input.owner_team,
          ...selected,
          before: before.map((row) => ({id: row.id, sha256: digest(row)})),
          automatic_retry: false,
        };
        await journal.save('application/create.request.json', intent);
        let reply: {status: number; text: string} | null = null;
        try {
          reply = await request(
            '/api/operations/SodaRuntime.Entry.create',
            'POST',
            {
              operation_id: selected.operation_id,
              inputs: {label: selected.label},
            },
            headers
          );
        } catch {
          /* Read this original effect below; never send another POST. */
        }
        if (reply && reply.status >= 400 && reply.status < 500) throw Error('application_create_denied');
        let confirmed = false;
        if (reply?.status === 200) {
          try {
            const body = object(JSON.parse(reply.text));
            confirmed = ['committed', 'replayed'].includes(body.status) && body.operation_id === selected.operation_id;
          } catch {
            /* Malformed response remains uncertain until normal readback. */
          }
        }
        await journal.save('application/create.response.json', {
          format: 1,
          status: reply?.status ?? null,
          confirmed,
          automatic_retry: false,
        });
      }
      const after = await records();
      for (const previous of intent.before) {
        const row = after.find((record) => record.id === previous.id);
        assert(row && digest(row) === previous.sha256, 'application_existing_record_changed');
      }
      const added = after.filter((row) => !intent.before.some((prior) => prior.id === row.id));
      // This pinned generated CRUD fixture uses operation_id as record ID.
      // Another writer's matching label cannot settle our uncertain operation.
      assert(
        added.length === 1 && added[0]!.id === selected.operation_id && added[0]!.data.label === selected.label,
        'application_create_outcome_unconfirmed'
      );
      return witness(added[0]!, selected.operation_id);
    },
    readRecord,
    async readRendered(expected: ApplicationRecordWitness) {
      await readRecord(expected);
      const response = await request('/', 'GET', undefined, {accept: 'text/html'});
      assert(
        response.status === 200 &&
          /<h1\b[^>]*>SodaRuntime<\/h1>/.test(response.text) &&
          response.text.includes(
            expected.label.replace(
              /[&<>"']/g,
              (character) => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'})[character]!
            )
          ),
        'application_rendered_record_unconfirmed'
      );
      return {status: response.status, heading: 'SodaRuntime', record_id: expected.id};
    },
  };
}
