// Operator prerequisite client. Product/native owners perform every effect;
// the protected collector remains the only qualification journey runner.
import assert from 'node:assert/strict';
import {mkdir, open, readFile, realpath, lstat} from 'node:fs/promises';
import path from 'node:path';
import {chromium} from 'playwright';
import {launchNativeBrowser} from '../native-browser';
import {loginNativeForgejo} from '../sodaspaces-controls';
import {nativeSpacesMount, mountedSpacesFacts} from '../sodaspaces-journey-evidence';
import {scenarioInput, pinnedSSH, sha256} from './validate';
import {checkScenario} from './readiness';
import {prepareApplication} from './prepare-application';
import {prepareFunding} from './prepare-funding';
import {requestContainerBytes} from './phases/namespace-http';
import {networkGuard} from './phases/support';

async function main() {
  process.umask(0o077);
  const args = process.argv.slice(2);
  assert.deepEqual(
    args.filter((_, i) => i % 2 === 0),
    ['--scenario', '--authorization', '--action', '--out', ...(args.length === 10 ? ['--terminal-receipt'] : [])]
  );
  assert(args.length === 8 || args.length === 10);
  const [scenarioFile, authorizationFile, action, out] = [args[1]!, args[3]!, args[5]!, args[7]!];
  assert(['application', 'funding', 'terminal'].includes(action));
  assert.equal(action === 'terminal', args.length === 10);
  const s = scenarioInput(JSON.parse(await readFile(scenarioFile, 'utf8')));
  assert.equal(s.target.distribution, 'docker');
  const authority = JSON.parse(await readFile(authorizationFile, 'utf8'));
  assert(authority.approved === true && typeof authority.questionCallID === 'string');
  const reviewBytes = await readFile(authority.review.path);
  assert.equal(sha256(reviewBytes), authority.review.sha256);
  const review = JSON.parse(reviewBytes.toString());
  assert.equal(review.task, 'J22');
  assert.deepEqual(review.target, s.target);
  assert.deepEqual(review.candidate, s.candidate);
  assert.equal(s.factory.length, 1);
  assert.deepEqual(review.paid_limits, s.factory[0]!.limits);
  const consent = JSON.parse(await readFile(review.paid_consent, 'utf8'));
  const subject = s.factory[0]!.subject;
  assert(
    consent.approved === true &&
      consent.one_trial === true &&
      consent.uuid === s.target.uuid &&
      consent.repository === Number(subject.repository_id) &&
      consent.project === subject.project_id &&
      consent.operation_id === subject.operation_id &&
      consent.connection_id === subject.connection_id
  );
  assert.deepEqual(consent.limits, s.factory[0]!.limits);
  assert(out.startsWith('/home/') && path.resolve(out) === out);
  assert.equal(await realpath(path.dirname(out)), path.dirname(out));
  try {
    await mkdir(out, {mode: 0o700});
  } catch (e) {
    if ((e as NodeJS.ErrnoException).code !== 'EEXIST') throw e;
  }
  const meta = await lstat(out);
  assert(
    meta.isDirectory() && !meta.isSymbolicLink() && meta.uid === process.getuid!() && (meta.mode & 0o777) === 0o700
  );
  const location = (name: string, privateFile = false) => {
    assert(/^[a-zA-Z0-9._/-]+$/.test(name) && name.split('/').every((p) => p && p !== '.' && p !== '..'));
    return path.join(out, privateFile ? 'private' : '', name);
  };
  const save = async (name: string, value: unknown, privateFile = false) => {
    const file = location(name, privateFile),
      parent = path.dirname(file);
    await mkdir(parent, {recursive: true, mode: 0o700});
    const bytes = JSON.stringify(value, null, 2) + '\n';
    assert(Buffer.byteLength(bytes) <= 2 << 20);
    const handle = await open(file, 'wx', 0o600);
    try {
      await handle.writeFile(bytes);
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
    return file;
  };
  const readSaved = async (name: string) => {
    for (const privateFile of [false, true]) {
      try {
        return JSON.parse(await readFile(location(name, privateFile), 'utf8'));
      } catch (e) {
        if ((e as NodeJS.ErrnoException).code !== 'ENOENT') throw e;
      }
    }
    return null;
  };
  await save('invocation-' + Date.now() + '.json', {
    format: 1,
    action,
    target: s.target,
    authorization: authorizationFile,
    authorization_sha256: sha256(await readFile(authorizationFile)),
    scenario_sha256: sha256(await readFile(scenarioFile)),
    source_files: await Promise.all(
      ['prepare-fixture.ts', 'prepare-funding.ts', 'prepare-application.ts'].map(async (name) => {
        const bytes = await readFile(path.join(import.meta.dir, name));
        return {name, sha256: sha256(bytes), bytes: bytes.length};
      })
    ),
  });
  const ready = await checkScenario(s),
    native = ready.native as any;
  assert(
    ['candidate_source', 'target_trust', 'native_observation', 'installed_identity', 'docker_identity'].every((n) =>
      ready.checks.some((c) => c.name === n && c.state === 'pass')
    ),
    'preparation_target_not_idle'
  );
  if (!ready.checks.some((c) => c.name === 'custody' && c.state === 'pass')) {
    const setup = await readSaved('application-preparation/setup.json');
    assert(
      action === 'application' &&
        /^a[a-f0-9]{24}$/.test(setup?.id) &&
        native.custody.setups === 1 &&
        Object.entries(native.custody).every(([name, count]) => name === 'setups' || count === 0),
      'unrelated_native_custody_blocks_preparation'
    );
    // The App owner inspects the journaled exact setup/spec before continuing.
    // A caller timeout never permits creating another setup.
  }
  assert(
    native.factory_readiness.every((f: any) => f.runs.length === 0 && f.allowances.length === 0),
    'approved_trial_already_started'
  );
  const owner = ready.observation!.inventory.connections.find((c) => c.id === subject.connection_id)!.owner_id;
  assert.equal(owner, review.guest_changes.install_app.administrator);
  if (!(await readSaved('before.json')))
    await save('before.json', {
      observed_at: native.observed_at,
      custody: native.custody,
      history: native.history,
      connection: ready.observation!.factory[0]!.connection,
    });
  const ssh = await pinnedSSH(s);
  const sshRead = (program: string) => {
    const child = Bun.spawnSync(ssh, {
      stdin: new TextEncoder().encode(program),
      stdout: 'pipe',
      stderr: 'pipe',
      timeout: 45000,
      maxBuffer: 4 << 20,
    });
    assert.equal(child.exitCode, 0, 'preparation_namespace_http_unconfirmed');
    return new TextDecoder().decode(child.stdout);
  };
  for (const name of ['home', 'browser', 'tmp']) await mkdir(path.join(out, name), {recursive: true, mode: 0o700});
  const home = path.join(out, 'home'),
    db = path.join(home, '.pki/nssdb');
  await mkdir(db, {recursive: true, mode: 0o700});
  const cert = (args: string[]) => {
    const r = Bun.spawnSync(['/usr/bin/certutil', ...args], {stdout: 'pipe', stderr: 'pipe'});
    assert.equal(r.exitCode, 0, 'preparation_browser_trust_unconfirmed');
  };
  if (!(await readSaved('browser-trust.json'))) {
    cert(['-N', '--empty-password', '-d', 'sql:' + db]);
    cert(['-A', '-d', 'sql:' + db, '-n', 'Admitted target CA', '-t', 'C,,', '-i', s.trust!.appliance_ca.path]);
    await save('browser-trust.json', {appliance_ca: s.trust!.appliance_ca});
  }
  process.env.TMPDIR = path.join(out, 'tmp');
  const browser = await launchNativeBrowser(chromium, path.join(out, 'browser'), home);
  let login = true;
  let armed: {method: string; path: string; body: unknown} | undefined;
  try {
    const audit = {requests: [] as any[], blocked_raw_runs: 0, blocked_other: 0, raw_responses: 0, login_posts: 0};
    await browser.context.route(
      '**/*',
      networkGuard(
        s.target.origin,
        audit,
        () => login,
        async (request) => {
          const u = new URL(request.url());
          if (
            !armed ||
            u.origin !== s.target.origin ||
            u.pathname !== armed.path ||
            u.search ||
            u.hash ||
            request.method() !== armed.method
          )
            return false;
          assert.deepEqual(request.postDataJSON(), armed.body);
          armed = undefined;
          return true;
        }
      )
    );
    const page = await browser.context.newPage();
    const password = (await readFile(s.credentials.workspace_owner_password_file, 'utf8')).trim();
    await loginNativeForgejo(
      page,
      s.target.origin,
      review.guest_changes.install_app.administrator === owner ? 'soda-user' : '',
      password
    );
    login = false;
    await page.goto(s.target.origin + '/workspace?repository=' + subject.repository_id);
    await nativeSpacesMount(page);
    const mount = await mountedSpacesFacts(page);
    const request = async (input: string, method = 'GET', body?: unknown) => {
      const route = input.startsWith('/api/') ? input.slice(5) : input;
      assert(/^[A-Za-z0-9_/-]+$/.test(route) && !route.includes('..'));
      if (method !== 'GET') {
        assert(!armed, 'preparation_request_still_uncertain');
        armed = {method, path: mount.apiBase + '/' + route, body};
      }
      try {
        return await page.evaluate(
          async ({route, method, body}) => {
            const response = await (document.querySelector('soda-spaces') as any).binding.transport.request(route, {
              method,
              ...(body === undefined
                ? {}
                : {headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)}),
              signal: AbortSignal.timeout(180000),
            });
            return {status: response.status, value: await response.json()};
          },
          {route, method, body}
        );
      } finally {
        armed = undefined;
      }
    };
    let result: unknown;
    if (action === 'application') {
      const fixturePath = review.guest_changes.install_app.fixture;
      result = await prepareApplication(
        s,
        {path: fixturePath, sha256: sha256(await readFile(fixturePath))},
        {
          administrator_id: owner,
          request,
          save,
          readSaved,
          savePrivate: (name, value) => save(name, value, true),
          appRequest: async (input) => {
            const response = await requestContainerBytes(
              {sshRead},
              {
                container: native.docker.container_id,
                image: native.docker.reference,
              },
              input.origin,
              new URL(s.target.origin).hostname,
              {
                method: input.method,
                route: input.path,
                headers: input.headers,
                body: JSON.stringify(input.body),
              }
            );
            return {status: response.status, text: Buffer.from(response.body_base64, 'base64').toString('utf8')};
          },
        }
      );
    } else if (action === 'funding') result = await prepareFunding(s, owner, {request, save, readSaved});
    else {
      // Retire only the exact owner reservation returned by a failed unpaid
      // browser phase. No Project/App stop, deletion or Factory action follows.
      const bytes = await readFile(args[9]!);
      assert(bytes.length <= 4096);
      const terminal = JSON.parse(bytes.toString());
      assert(
        terminal.status === 201 &&
          /^[a-f0-9]{32}$/.test(terminal.id) &&
          terminal.project === subject.project_id &&
          terminal.actor === owner
      );
      const intent = await readSaved('terminal/intent.json');
      if (intent) assert.deepEqual(intent.terminal, terminal);
      const observed = native.resource_inventory.terminals.filter((t: any) => t.id === terminal.id);
      if (observed.length) {
        assert(observed.length === 1 && observed[0].project_id === terminal.project && observed[0].actor_id === owner);
        assert(!intent, 'terminal_end_uncertain_no_retry');
        await save('terminal/intent.json', {
          terminal,
          receipt: {path: args[9], sha256: sha256(bytes)},
          automatic_retry: false,
        });
        let status = 0;
        try {
          status = (
            await request('environments/' + terminal.project + '/terminal-sessions/' + terminal.id, 'POST', {
              action: 'end',
            })
          ).status;
        } catch {
          /* Inspect the exact native result below; never repeat this POST. */
        }
        await save('terminal/response.json', {status, automatic_retry: false});
      }
      const after = await checkScenario(s);
      assert(after.observation && after.native);
      assert(
        !(after.native as any).resource_inventory.terminals.some((t: any) => t.id === terminal.id),
        'terminal_retirement_unconfirmed'
      );
      result = {
        format: 1,
        terminal: terminal.id,
        retired: true,
        history: (after.native as any).history,
        model_calls: 0,
      };
    }
    if (!(await readSaved('result.json'))) await save('result.json', result);
    console.log(JSON.stringify({complete: true, action, result_file: path.join(out, 'result.json'), model_calls: 0}));
  } finally {
    await browser.close();
  }
}
await main().catch((e: unknown) => {
  const message = e instanceof Error && /^[a-z_]+$/.test(e.message) ? e.message : 'fixture_preparation_unconfirmed';
  console.error(message);
  process.exitCode = 1;
});
