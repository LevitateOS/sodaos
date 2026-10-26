// Normal operator setup of the admitted App. Private owner/token bytes never
// enter the public journal; retained intents prevent blind mutation replay.
import {randomBytes} from 'node:crypto';
import {applicationFixture} from './application';
import {applicationResponse, type ApplicationRecord} from '../../../frontend/spaces/sodaspaces-application-response';
import type {PublicArtifact, QualificationScenario} from './scenario';

export interface PrepareApplicationRuntime {
  readonly administrator_id: string;
  readonly port?: number;
  request(route: string, method?: 'GET' | 'POST', body?: unknown): Promise<{status: number; value: unknown}>;
  appRequest(input: {
    origin: string;
    method: 'POST';
    path: string;
    headers: Record<string, string>;
    body: unknown;
  }): Promise<{status: number; text: string}>;
  /** Durable, exclusive public JSON publication. */
  save(name: string, value: unknown): Promise<unknown>;
  /** Durable, exclusive private JSON publication; returns its absolute path. */
  savePrivate(name: string, value: unknown): Promise<string>;
  /** Reads either journal namespace without printing private values. */
  readSaved(name: string): Promise<unknown | null>;
}
export interface PreparedApplication {
  readonly format: 1;
  readonly application_id: string;
  readonly data_id: string;
  readonly owner_team: string;
  readonly container_id: string;
  readonly owner_credentials_file: string;
  readonly fixture: PublicArtifact;
}
type Row = Record<string, any>;
const prefix = 'application-preparation/';
const appID = /^a[a-f0-9]{24}$/;
const containerID = /^[a-f0-9]{64}$/;
const teamID = /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/;
function need(value: unknown, code: string): asserts value {
  if (!value) throw Error(code);
}
function row(value: unknown): Row {
  need(value && typeof value === 'object' && !Array.isArray(value), 'application_setup_response_invalid');
  return value as Row;
}
function same(a: Row, b: Row) {
  return Object.keys(a).length === Object.keys(b).length && Object.keys(a).every((key) => a[key] === b[key]);
}

export async function prepareApplication(
  scenario: QualificationScenario,
  fixtureRef: PublicArtifact,
  runtime: PrepareApplicationRuntime
): Promise<PreparedApplication> {
  // Replace transport/parser exceptions with bounded errors: an implementation
  // of a private HTTP callback must never expose its request or response bytes.
  try {
    return await prepare(scenario, fixtureRef, runtime);
  } catch (error) {
    const code = error instanceof Error ? error.message : '';
    throw Error(/^application_[a-z_]+$/.test(code) ? code : 'application_preparation_unconfirmed');
  }
}

async function prepare(s: QualificationScenario, ref: PublicArtifact, rt: PrepareApplicationRuntime) {
  const fixture = await applicationFixture(s, ref);
  const port = rt.port ?? 24000;
  need(
    /^[1-9][0-9]*$/.test(rt.administrator_id) && Number.isInteger(port) && port >= 20000 && port < 30000,
    'application_administrator_or_port_invalid'
  );
  const saved = (name: string) => rt.readSaved(prefix + name);
  const save = (name: string, value: unknown) => rt.save(prefix + name, value);
  const api = async (route: string, method: 'GET' | 'POST' = 'GET', body?: unknown) => {
    const response = await rt.request(route, method, body);
    need(
      response.status === (route === '/api/applications/setups' && method === 'POST' ? 201 : 200),
      'application_operator_request_unconfirmed'
    );
    return row(response.value);
  };
  const once = async (name: string, route: string, body: unknown) => {
    need((await saved(name)) === null, 'application_mutation_uncertain_inspect_same_identity');
    await save(name, {format: 1, method: 'POST', route, body, automatic_retry: false});
    return api(route, 'POST', body);
  };

  let selection = await saved('selection.private.json');
  if (selection === null) {
    selection = {
      application_id: 'a' + randomBytes(12).toString('hex'),
      data_id: 'a' + randomBytes(12).toString('hex'),
      target_uuid: s.target.uuid,
      fixture_sha256: ref.sha256,
      administrator_id: rt.administrator_id,
      port,
    };
    await rt.savePrivate(prefix + 'selection.private.json', selection);
  }
  const selected = row(selection);
  need(
    appID.test(selected.application_id) &&
      appID.test(selected.data_id) &&
      selected.application_id !== selected.data_id &&
      selected.target_uuid === s.target.uuid &&
      selected.fixture_sha256 === ref.sha256 &&
      selected.administrator_id === rt.administrator_id &&
      selected.port === port,
    'application_selection_changed'
  );
  const ownerName = 'owner.private.json';
  let credentials = await saved(ownerName),
    ownerReference = await saved('owner-reference.json');
  if (credentials === null) {
    need(ownerReference === null && (await saved('create.intent.json')) === null, 'application_private_owner_missing');
    credentials = {
      email: selected.application_id + '@qualification.soda.test',
      password: randomBytes(32).toString('hex'),
    };
    const file = await rt.savePrivate(prefix + ownerName, credentials);
    need(
      typeof file === 'string' && file.startsWith('/home/') && !/[\r\n\0]/.test(file),
      'application_private_owner_path_invalid'
    );
    ownerReference = {owner_credentials_file: file};
    await save('owner-reference.json', ownerReference);
  }
  const owner = row(credentials),
    ownerFile = row(ownerReference).owner_credentials_file;
  need(
    Object.keys(owner).sort().join(',') === 'email,password' &&
      typeof owner.email === 'string' &&
      typeof owner.password === 'string' &&
      owner.email.length <= 320 &&
      owner.password.length >= 8 &&
      owner.password.length <= 512 &&
      typeof ownerFile === 'string' &&
      ownerFile.startsWith('/home/'),
    'application_private_owner_invalid'
  );
  const spec = {
    id: selected.application_id,
    name: fixture.name,
    artifact_sha256: fixture.artifact_sha256,
    bundle_sha256: fixture.bundle_sha256,
    tools_sha256: fixture.tools_sha256,
    runtime_image: fixture.runtime_image,
    image: fixture.image,
    data: selected.data_id,
    origin: fixture.origin,
    port,
  };
  const intent = {
    format: 1,
    application: spec,
    administrator_id: rt.administrator_id,
    fixture: ref,
    target_uuid: s.target.uuid,
  };
  const oldIntent = await saved('intent.json');
  if (oldIntent === null) await save('intent.json', intent);
  else {
    const old = row(oldIntent);
    need(
      same(row(old.application), spec) &&
        old.administrator_id === rt.administrator_id &&
        old.target_uuid === s.target.uuid &&
        row(old.fixture).sha256 === ref.sha256,
      'application_admission_changed'
    );
  }
  const verifyApp = (value: unknown, expectedTeam?: string): ApplicationRecord => {
    const view = applicationResponse(value),
      a = view.application;
    need(
      same({...a, revision: undefined, owner_team: undefined}, {...spec, revision: undefined, owner_team: undefined}) &&
        a.revision === 1 &&
        teamID.test(a.owner_team) &&
        (!expectedTeam || a.owner_team === expectedTeam) &&
        view.administrator_id === rt.administrator_id &&
        view.operator &&
        !view.restore,
      'application_registered_identity_changed'
    );
    return view;
  };
  const normalPath = '/api/applications/' + spec.id;
  const currentApp = async () => {
    const response = await rt.request(normalPath, 'GET');
    if (response.status === 404) return null;
    need(response.status === 200, 'application_registered_read_unconfirmed');
    return verifyApp(response.value);
  };
  let normal = await currentApp();
  let claimReceipt = await saved('claim.json');
  if (!normal) {
    let setupValue = await saved('setup.json');
    if (setupValue === null && (await saved('create.intent.json')) !== null) {
      const listing = await api('/api/applications/setups');
      need(Array.isArray(listing.setups), 'application_setup_inventory_unconfirmed');
      const matches = listing.setups.filter((v: unknown) => row(row(v).application).id === spec.id);
      need(matches.length === 1, 'application_setup_creation_uncertain');
      setupValue = matches[0];
      await save('setup.json', setupValue);
    }
    if (setupValue === null) {
      setupValue = await once('create.intent.json', '/api/applications/setups', {
        application: spec,
        administrator_id: rt.administrator_id,
      });
      await save('setup.json', setupValue);
    }
    const setup = row(setupValue);
    need(
      appID.test(setup.id) &&
        same(row(setup.application), spec) &&
        String(setup.administrator_id) === rt.administrator_id &&
        setup.state === 'pending',
      'application_setup_identity_changed'
    );
    const actions = '/api/applications/setups/' + setup.id + '/actions';
    if (claimReceipt === null) {
      let begun = await saved('begin.private.json');
      if (begun === null) {
        let expected = '',
          journal = 'begin.intent.json';
        if ((await saved(journal)) !== null) {
          // Begin can lose its browser reply when activating shared ingress.
          // Only an exact ready provisional generation can return its existing
          // capability; never repeat activation against unknown/stopped state.
          const listing = await api('/api/applications/setups');
          need(Array.isArray(listing.setups), 'application_setup_inventory_unconfirmed');
          const matches = listing.setups.filter((v: unknown) => row(v).id === setup.id);
          need(
            matches.length === 1 && same(row(row(matches[0]).application), spec),
            'application_setup_identity_changed'
          );
          const previous = row(matches[0]).observation;
          const inspected = await api(actions, 'POST', {
            action: 'inspect',
            expected_container: previous === null ? '' : row(previous).container,
            confirm: false,
          });
          const current = row(inspected.setup),
            observed = row(current.observation);
          need(
            current.id === setup.id &&
              same(row(current.application), spec) &&
              current.state === 'pending' &&
              containerID.test(observed.container) &&
              observed.running &&
              observed.ready,
            'application_setup_begin_uncertain'
          );
          expected = observed.container;
          journal = 'begin-capability.intent.json';
        }
        const response = await once(journal, actions, {action: 'begin', expected_container: expected, confirm: false});
        const result = row(response.result),
          observed = row(row(response.setup).observation);
        need(
          row(response.setup).id === setup.id &&
            result.operation === setup.id &&
            containerID.test(observed.container) &&
            observed.running &&
            observed.ready &&
            typeof result.setup_url === 'string',
          'application_setup_begin_unconfirmed'
        );
        begun = {container: observed.container, setup_url: result.setup_url};
        await rt.savePrivate(prefix + 'begin.private.json', begun);
      }
      const begin = row(begun),
        url = new URL(begin.setup_url);
      const fragment = new URLSearchParams(url.hash.slice(1));
      need(
        url.origin === fixture.origin &&
          url.pathname === '/setup' &&
          !url.search &&
          !url.username &&
          !url.password &&
          fragment.get('operation') === setup.id &&
          containerID.test(fragment.get('token') ?? '') &&
          [...fragment.keys()].length === 2 &&
          containerID.test(begin.container),
        'application_private_setup_link_invalid'
      );
      need((await saved('claim.intent.json')) === null, 'application_owner_claim_uncertain');
      await save('claim.intent.json', {
        format: 1,
        setup_id: setup.id,
        container: begin.container,
        origin: fixture.origin,
        path: '/setup/claim',
        automatic_retry: false,
      });
      const response = await rt.appRequest({
        origin: fixture.origin,
        method: 'POST',
        path: '/setup/claim',
        headers: {origin: fixture.origin, 'content-type': 'application/json'},
        body: {
          operation: setup.id,
          token: fragment.get('token'),
          email: owner.email,
          password: owner.password,
          timezone: 'Europe/Brussels',
        },
      });
      need(response.status === 200 && response.text.length <= 8192, 'application_owner_claim_unconfirmed');
      const receipt = row(JSON.parse(response.text));
      need(
        receipt.schema === 'canlang.soda.setup.v1' &&
          receipt.operation === setup.id &&
          receipt.app === fixture.name &&
          receipt.artifactDigest === fixture.artifact_sha256 &&
          receipt.bundleDigest === fixture.bundle_sha256 &&
          containerID.test(receipt.storageFingerprint) &&
          teamID.test(receipt.ownerTeam) &&
          Number.isSafeInteger(receipt.completedAt) &&
          receipt.completedAt > 0 &&
          receipt.completedAt <= setup.expires_at,
        'application_owner_receipt_unconfirmed'
      );
      claimReceipt = {
        schema: receipt.schema,
        operation: receipt.operation,
        app: receipt.app,
        artifactDigest: receipt.artifactDigest,
        bundleDigest: receipt.bundleDigest,
        storageFingerprint: receipt.storageFingerprint,
        ownerTeam: receipt.ownerTeam,
        completedAt: receipt.completedAt,
      };
      await save('claim.json', claimReceipt);
    }
    normal = await currentApp();
    if (!normal) {
      const begin = row(await saved('begin.private.json'));
      const response = await once('complete.intent.json', actions, {
        action: 'complete',
        expected_container: begin.container,
        confirm: true,
      });
      need(
        row(response.setup).id === setup.id &&
          row(response.setup).state === 'complete' &&
          row(row(response.result).receipt).ownerTeam === row(claimReceipt).ownerTeam,
        'application_setup_completion_unconfirmed'
      );
      await save('complete.json', {format: 1, setup_id: setup.id, owner_team: row(claimReceipt).ownerTeam});
      normal = await currentApp();
    }
  }
  need(normal && claimReceipt, 'application_registered_owner_unconfirmed');
  normal = verifyApp(normal, row(claimReceipt).ownerTeam);
  if ((await saved('start.intent.json')) !== null) {
    // Inspection is read-only at the native owner; never repeat uncertain Start.
    normal = verifyApp(
      await api(normalPath + '/actions', 'POST', {
        action: 'inspect',
        revision: 1,
        expected_container: normal.observation?.container ?? '',
        confirm_stop: false,
      }),
      row(claimReceipt).ownerTeam
    );
  } else if (!normal.observation?.running || !normal.observation.ready) {
    need(
      normal.observation && containerID.test(normal.observation.container),
      'application_normal_container_unconfirmed'
    );
    normal = verifyApp(
      await once('start.intent.json', normalPath + '/actions', {
        action: 'start',
        revision: 1,
        expected_container: normal.observation.container,
        confirm_stop: false,
      }),
      row(claimReceipt).ownerTeam
    );
  }
  need(
    normal.observation?.running && normal.observation.ready && containerID.test(normal.observation.container),
    'application_running_observation_unconfirmed'
  );
  const result: PreparedApplication = {
    format: 1,
    application_id: spec.id,
    data_id: spec.data,
    owner_team: normal.application.owner_team,
    container_id: normal.observation.container,
    owner_credentials_file: ownerFile,
    fixture: ref,
  };
  if ((await saved('result.json')) === null) await save('result.json', result);
  return result;
}
