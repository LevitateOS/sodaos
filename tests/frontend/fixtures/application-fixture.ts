import {SodaApplications} from '../../../frontend/spaces/sodaspaces-applications';
import type {ApplicationRecord} from '../../../frontend/spaces/sodaspaces-application-response';

interface ApplicationFixture {
  element: SodaApplications;
  calls: {path: string; body?: Record<string, unknown>}[];
  fail: boolean;
  delay: boolean;
  release: (() => void) | undefined;
}
declare global {
  interface Window {
    applicationFixture: ApplicationFixture;
    createApplicationFixture: () => Promise<void>;
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
window.createApplicationFixture = async () => {
  const element = new SodaApplications(),
    record = initial();
  const fixture: ApplicationFixture = {element, calls: [], fail: false, delay: false, release: undefined};
  element.active = true;
  element.sessionKey = 'generation-1';
  element.request = async (path, body) => {
    fixture.calls.push({path, ...(body ? {body} : {})});
    if (fixture.delay)
      await new Promise<void>((resolve) => {
        fixture.release = resolve;
      });
    if (fixture.fail) throw Error('private diagnostic');
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
