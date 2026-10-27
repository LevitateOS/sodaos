// Protected synthetic accounts, real login, permissions and business operations.
import assert from 'node:assert/strict';
import {randomBytes} from 'node:crypto';
import {mkdir, writeFile} from 'node:fs/promises';
import {join} from 'node:path';
export function actorChecks(context, source, runtime) {
  const {work, APP, TEAM_FIELD} = context;
  const {run} = source;
  const {transport, occurrence} = runtime;
  function csrf(html) {
    const inputs = html.match(/<input\b[^>]*>/g) ?? [];
    const token = inputs.find((input) => /\bname="_csrf"/.test(input))?.match(/\bvalue="([^"]+)"/)?.[1];
    assert(token, 'The real authenticated page did not supply CSRF.');
    return token;
  }

  async function login(server, credentials, team, mustSucceed = true) {
    const send = transport(server);
    const descriptor = await send('/auth/login');
    assert.equal(descriptor.status, 200, 'Real login descriptor unavailable.');
    const response = await send('/auth/login', {
      email: credentials.email,
      password: credentials.password,
      _presession: descriptor.json.preSessionToken,
    });
    if (!mustSucceed) {
      assert.equal(response.status, 403, 'Company credentials must not authenticate in preview.');
      return;
    }
    assert.equal(response.status, 200, 'Real account login failed.');
    const page = await send('/auth/select-team', undefined, {accept: 'text/html'});
    let token = csrf(page.text);
    assert.equal(
      (await send('/auth/select-team', {[TEAM_FIELD]: team}, {'x-csrf-token': token})).status,
      200,
      'Real team selection failed.'
    );
    const application = await send('/', undefined, {accept: 'text/html'});
    assert.equal(application.status, 200, 'Application page unavailable.');
    token = csrf(application.text);
    return {send, token};
  }

  async function operation(actor, name, inputs) {
    return actor.send(
      '/api/operations/' + (name.startsWith('system.') ? name : APP + '.' + name),
      {operation_id: occurrence(), inputs},
      {'x-csrf-token': actor.token}
    );
  }

  async function read(actor) {
    if (!actor.grant) {
      const grant = await actor.send(
        '/mcp/grants',
        {client_id: 'northstar-protected-check'},
        {'x-csrf-token': actor.token}
      );
      assert.equal(grant.status, 200, 'Real self-issued MCP grant failed.');
      actor.grant = grant.json.token;
    }
    const result = await actor.send(
      '/mcp',
      {jsonrpc: '2.0', id: 1, method: 'tools/call', params: {name: APP + '.EquipmentRequest.read', arguments: {}}},
      {authorization: 'Bearer ' + actor.grant, accept: 'application/json, text/event-stream'}
    );
    assert.equal(result.status, 200, 'Authorized MCP read failed.');
    assert(
      !result.json.result?.isError && Array.isArray(result.json.result?.structuredContent?.records),
      'Business read did not return records.'
    );
    return result.json.result.structuredContent.records;
  }

  const ref = (row) => ({id: row.id, version: String(row.version)});
  const row = async (actor, id) => {
    const result = (await read(actor)).find((record) => record.id === id);
    assert(result, 'Expected saved record missing.');
    return result;
  };
  async function submit(actor, title, minor) {
    const result = await operation(actor, 'request_equipment', {
      title,
      reason: 'Synthetic acceptance request',
      amount: {minor, currency: 'EUR'},
    });
    assert.equal(result.status, 200, 'Equipment submission failed.');
    const saved = (await read(actor)).find((record) => record.data.title === title);
    assert(saved, 'Request was not saved.');
    assert.equal(saved.data.amount.minor, minor, 'Money must retain exact EUR minor units.');
    assert.equal(saved.data.amount.currency, 'EUR');
    return saved;
  }

  async function denied(actor, name, inputs, observer, id) {
    const before = await read(observer);
    const response = await operation(actor, name, inputs);
    assert([403, 422].includes(response.status), `${name}: expected business refusal, got HTTP ${response.status}.`);
    assert.deepEqual(await read(observer), before, 'Refused operation changed protected records.');
    if (id) assert((await read(observer)).some((record) => record.id === id));
  }

  async function identities(bundleFile, label) {
    const data = join(work, label + '-data');
    await mkdir(data, {mode: 0o700});
    const people = {};
    let owner;
    for (const name of ['maya', 'alex', 'jordan', 'robin', 'evan']) {
      const credentials = {
        email: name + '@example.invalid',
        password: randomBytes(32).toString('base64url'),
        timezone: 'UTC',
      };
      const file = join(work, label + '-' + name + '.json');
      await writeFile(file, JSON.stringify(credentials), {mode: 0o600});
      const commandName = name === 'maya' ? 'bootstrap' : 'member';
      const result = JSON.parse(
        await run('can-soda', [
          commandName,
          '--bundle',
          bundleFile,
          '--data',
          data,
          ...(owner ? ['--owner', owner.teamId] : []),
          '--credentials',
          file,
        ])
      );
      assert(
        result.ok === true && typeof result.teamId === 'string' && typeof result.userId === 'string',
        'Canonical member provisioning failed.'
      );
      if (!owner) owner = result;
      people[name] = {...credentials, userId: result.userId};
    }
    return {data, people, owner: owner.teamId};
  }

  async function actors(server, fixture, finance, existing) {
    const opened = existing ?? {};
    if (!existing)
      for (const [name, person] of Object.entries(fixture.people))
        opened[name] = await login(server, person, fixture.owner);
    for (const [name, role] of [
      ['maya', 'operations'],
      ['jordan', 'manager'],
      ...(finance ? [['robin', 'finance']] : []),
    ]) {
      const result = await operation(opened.maya, 'system.team.role', {
        member: {id: fixture.people[name].userId},
        role: APP + '.' + role,
      });
      assert.equal(result.status, 200, 'Actual owner role grant failed.');
    }
    return opened;
  }

  return {login, operation, read, ref, row, submit, denied, identities, actors};
}
