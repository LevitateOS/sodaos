import {SodaApplications} from '../../../frontend/spaces/sodaspaces-applications';
import type {ApplicationRecord} from '../../../frontend/spaces/sodaspaces-application-response';

interface ApplicationFixture {
  element: SodaApplications;
  calls: {path: string; body?: Record<string, unknown>}[];
  fail: boolean;
  delay: boolean;
  release: (() => void) | undefined;
  failRestore: boolean;
  setupExpiresAt: number | undefined;
}
declare global {
  interface Window {
    applicationFixture: ApplicationFixture;
    createApplicationFixture: (operator?: boolean) => Promise<void>;
  }
}
const initial = (): ApplicationRecord => ({
  application: {
    id: 'a' + '1'.repeat(24),
    revision: 1,
    name: 'Example',
    artifact_sha256: '2'.repeat(64),
    bundle_sha256: '3'.repeat(64),
    tools_sha256: '4'.repeat(64),
    runtime_image: 'sha256:' + '5'.repeat(64),
    image: 'sha256:' + '6'.repeat(64),
    data: 'a' + '7'.repeat(24),
    owner_team: '12345678-1234-1234-1234-123456789abc',
    origin: 'https://app.example.test',
    port: 20001,
  },
  administrator_id: '41',
  operator: false,
  observation: {
    id: 'a' + '1'.repeat(24),
    revision: 1,
    container: 'a'.repeat(64),
    image: 'sha256:' + '6'.repeat(64),
    origin: 'https://app.example.test',
    running: false,
    ready: false,
  },
});
window.createApplicationFixture = async (operator = false) => {
  const element = new SodaApplications(),
    record = initial();
  record.operator = operator;
  const snapshots = [{id: 'a' + '8'.repeat(24), created_at: 1792713600, application: {...record.application}}];
  const fixture: ApplicationFixture = {element, calls: [], fail: false, delay: false, release: undefined, failRestore: false, setupExpiresAt: undefined};
  const setups: {id: string; application: Record<string, unknown>; administrator_id: number; expires_at: number; state: string; observation: {container: string; running: boolean; ready: boolean} | null}[] = [];
  element.active = true;
  element.sessionKey = 'generation-1';
  element.request = async (path, body) => {
    fixture.calls.push({path, ...(body ? {body} : {})});
    if (fixture.delay)
      await new Promise<void>((resolve) => {
        fixture.release = resolve;
      });
    if (fixture.fail) throw Error('private diagnostic');
    if (path === '/api/applications/setups') {
      if (!operator) throw Error('operator required');
      if (!body) return {setups: setups.filter(row => row.state === 'pending').map(row => ({...row, expires_at: fixture.setupExpiresAt ?? row.expires_at}))};
      const row = {id: 'a' + 'c'.repeat(24), application: body.application as Record<string, unknown>, administrator_id: Number(body.administrator_id), expires_at: Math.floor(Date.now() / 1000) + 600, state: 'pending', observation: null};
      setups.push(row); return row;
    }
    if (path.startsWith('/api/applications/setups/') && body) {
      const row = setups.find(row => row.id === path.split('/')[4]);
      if (!row) throw Error('missing setup');
      row.observation = {container: 'c'.repeat(64), running: body.action === 'begin', ready: body.action === 'begin'};
      if (body.action === 'complete') row.state = 'complete';
      if (body.action === 'abort') row.state = 'aborted';
      return {setup: {...row, expires_at: fixture.setupExpiresAt ?? row.expires_at}, result: {operation: row.id, observation: row.observation, ...(body.action === 'begin' ? {setup_url: String(row.application.origin) + '/setup#operation=' + row.id + '&token=' + 'd'.repeat(64)} : {})}};
    }

    if (path.endsWith('/snapshots')) {
      if (body && !snapshots.some(row => row.id === body.id))
        snapshots.unshift({id: body.id as string, created_at: 1792713601, application: {...record.application}});
      return {snapshots: body ? snapshots.filter(row => row.id === body.id) : snapshots, complete: true};
    }
    if (path.endsWith('/restores') && body) {
      record.restore ??= {
        id: body.id as string, snapshot: body.snapshot as string,
        previous: {...record.application},
        next: {...record.application, revision: record.application.revision + 1, data: 'a' + '9'.repeat(24)},
        expected_container: body.expected_container as string, state: 'pending',
      };
      if (fixture.failRestore) { fixture.failRestore = false; record.observation = null; throw Error('lost reply'); }
      if (body.action !== 'abort') record.application = record.restore.next;
      delete record.restore;
      record.observation = {id: record.application.id, revision: record.application.revision, container: 'b'.repeat(64), image: record.application.image, origin: record.application.origin, running: false, ready: false};
      return record;
    }
    if (!body) return {applications: [record], complete: true};
    if (record.observation) {
      record.observation.running = body.action === 'start';
      record.observation.ready = record.observation.running;
    }
    return record;
  };
  document.querySelector('main')?.append(element);
  window.applicationFixture = fixture;
  await element.updateComplete;
};
