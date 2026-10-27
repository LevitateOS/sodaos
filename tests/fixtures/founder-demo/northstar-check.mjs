// Development-only protected Northstar integration. Uses installed public commands
// and real authenticated HTTP/MCP; it never supplies or rewrites application source.
import assert from 'node:assert/strict';
import {mkdir, mkdtemp, readFile, realpath} from 'node:fs/promises';
import {createRequire} from 'node:module';
import {dirname, join, resolve} from 'node:path';
import {pathToFileURL} from 'node:url';
import {sourceChecks} from './northstar-check-source.mjs';
import {runtimeChecks} from './northstar-check-runtime.mjs';
import {actorChecks} from './northstar-check-actors.mjs';
import {policyChecks} from './northstar-check-policy.mjs';

process.umask(0o077);
const root = await realpath(process.cwd());
const config = JSON.parse(await readFile(process.argv[2], 'utf8'));
const sourceOnly = process.argv[3] === '--source-only';
assert(process.argv.length === (sourceOnly ? 4 : 3), 'Unknown protected check arguments.');
assert(sourceOnly || process.env.SODA_VERIFICATION_RUN_ID, 'Whole-app checks require the native verification phase.');
assert(root.startsWith('/home/'), 'Use the assigned checkout beneath /home.');
assert(['manager', 'finance'].includes(config.policy), 'Unknown acceptance policy.');
assert(
  typeof config.source === 'string' &&
    /^[A-Za-z0-9_./-]+\.can$/.test(config.source) &&
    !config.source.split('/').some((part) => part === '..' || part === '.' || part === ''),
  'Invalid source entry.'
);
assert(
  config.policy === 'finance'
    ? /^[0-9a-f]{40}$/.test(config.baseline_source_commit)
    : config.baseline_source_commit === null,
  'Finance verification requires the actual approved v1 source commit.'
);
const resources = new Set();
for (const name of ['company', 'preview']) {
  const environment = config[name];
  const origin = new URL(environment.origin);
  assert(
    origin.protocol === 'https:' && origin.origin === environment.origin && !origin.username && !origin.password,
    'Use a canonical configured HTTPS origin.'
  );
  for (const field of ['identity_resource', 'state_resource']) {
    assert(
      typeof environment[field] === 'string' &&
        /^[a-z0-9-]+$/.test(environment[field]) &&
        !resources.has(environment[field]),
      'Company and preview resources must be independent.'
    );
    resources.add(environment[field]);
  }
}
assert.notEqual(config.company.origin, config.preview.origin, 'Preview must have its own origin.');
const home = await realpath(process.env.HOME);
assert(home.startsWith('/home/'), 'Use assigned private storage beneath /home.');
const work = await mkdtemp(join(home, 'northstar-check-'));
await mkdir(join(work, 'tmp'), {mode: 0o700});
const tools = resolve(dirname(process.execPath), '..');
const require = createRequire(join(tools, 'lib/node_modules/@canlang/cloudflare/package.json'));
const {CONTRACTS_VERSION, TEAM_FIELD} = await import(pathToFileURL(require.resolve('@canlang/contracts')).href);
const command = (name) => join(tools, 'bin', name);
const childEnv = {...process.env, TMPDIR: join(work, 'tmp')};
const APP = 'NorthstarEquipment';
let runtime;

const context = {root, config, work, tools, CONTRACTS_VERSION, TEAM_FIELD, command, childEnv, APP};
const source = sourceChecks(context);
const services = runtimeChecks(context);
const accounts = actorChecks(context, source, services);
const {compile, bundle, generatedSource, publish} = source;
const {start} = services;
const {identities, actors, row, read, login, submit, denied, ref, operation} = accounts;
const {baselineChecks, financeChecks} = policyChecks(context, accounts);

if (sourceOnly) {
  await compile(root, 'source-check');
  console.log(
    'Northstar source check: actual candidate compilation and nonempty operation examples passed; whole-app outcomes not checked.'
  );
} else
  try {
    console.log('Northstar: compiling actual candidate and running nonempty examples.');
    const candidate = await compile(root, 'candidate');
    const company = await bundle(candidate, 'company');
    const preview = await bundle(candidate, 'preview');
    const baseline =
      config.policy === 'finance'
        ? await bundle(await compile(await generatedSource(config.baseline_source_commit), 'baseline-build'), 'company')
        : company;
    const fixture = await identities(baseline, 'company');
    runtime = await start(baseline, fixture.data, fixture.owner);
    assert.equal(runtime.origin, config.company.origin, 'Runtime replaced the configured HTTPS origin.');
    const companyServer = {...runtime};
    let opened = await actors(companyServer, fixture, false);
    const retained = await baselineChecks(opened);
    await runtime.stop();
    runtime = undefined;
    runtime = await start(company, fixture.data, fixture.owner);
    Object.assign(companyServer, runtime);
    opened = await actors(companyServer, fixture, config.policy === 'finance', opened);
    assert.deepEqual(
      await row(opened.alex, retained.monitor.id),
      retained.monitor,
      'Restart/update lost delivered record.'
    );
    assert.deepEqual(
      await row(opened.alex, retained.pending.id),
      retained.pending,
      'Restart/update changed pending record.'
    );
    if (config.policy === 'finance') await financeChecks(opened, retained);
    const companyRecords = await read(opened.maya);
    await runtime.stop();
    runtime = undefined;
    const previewFixture = await identities(preview, 'preview');
    runtime = await start(preview, previewFixture.data, previewFixture.owner);
    assert.equal(runtime.origin, config.preview.origin);
    await login(runtime, fixture.people.alex, fixture.owner, false);
    const previewActors = await actors(runtime, previewFixture, config.policy === 'finance');
    assert.equal((await read(previewActors.alex)).length, 0, 'Preview starts with company records.');
    if (config.policy === 'manager') await baselineChecks(previewActors);
    else {
      let sample = await submit(previewActors.alex, 'Preview-only EUR1001', '100100');
      assert.equal(sample.data.needs_finance, true);
      await denied(previewActors.maya, 'deliver', {request: ref(sample)}, previewActors.alex, sample.id);
      assert.equal(
        (await operation(previewActors.jordan, 'approve_finance_route', {request: ref(sample)})).status,
        200
      );
      sample = await row(previewActors.robin, sample.id);
      assert.equal((await operation(previewActors.robin, 'approve_finance', {request: ref(sample)})).status, 200);
      sample = await row(previewActors.maya, sample.id);
      assert.equal((await operation(previewActors.maya, 'deliver', {request: ref(sample)})).status, 200);
      assert.equal((await row(previewActors.alex, sample.id)).data.stage, 'delivered');
    }
    await runtime.stop();
    runtime = undefined;
    runtime = await start(company, fixture.data, fixture.owner);
    Object.assign(companyServer, runtime);
    assert.deepEqual(await read(opened.maya), companyRecords, 'Preview changed company records.');
    await runtime.stop();
    runtime = undefined;
    await publish(company, join(root, '.soda/application/bundle.json'));
    await publish(preview, join(root, '.soda/application/preview/bundle.json'));
    console.log(
      `Northstar ${config.policy}: real accounts, permissions, exact money, saved history, restart/update retention and separate preview passed. Browser/TLS/deployment remain separate checks.`
    );
  } catch (error) {
    try {
      await runtime?.stop();
    } catch (cleanup) {
      throw new AggregateError([error, cleanup], 'Northstar check and runtime retirement failed.');
    }
    runtime = undefined;
    throw error;
  } finally {
    await runtime?.stop();
  }
