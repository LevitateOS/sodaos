import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {applicationResponse, applicationsResponse} from '../../frontend/spaces/sodaspaces-application-response';
const fixture = () => ({
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
  administrator_id: '9223372036854775807',
  operator: false,
  observation: {
    id: 'a' + '1'.repeat(24),
    revision: 1,
    container: 'a'.repeat(64),
    image: 'sha256:' + '6'.repeat(64),
    origin: 'https://app.example.test',
    running: true,
    ready: true,
  },
});
test('application receipts preserve stable administrator precision and refuse unadmitted readiness', () => {
  const source = fixture();
  assert.equal(applicationResponse(source).administrator_id, '9223372036854775807');
  assert.equal(applicationResponse({...source, observation: null}).observation, null);
  assert.throws(() => applicationResponse({...source, observation: {...source.observation, revision: 2}}));
  assert.throws(() => applicationResponse({...source, observation: {...source.observation, running: false}}));
  assert.throws(() =>
    applicationResponse({...source, application: {...source.application, origin: 'javascript:alert(1)'}})
  );
  assert.throws(() => applicationsResponse({applications: [source, source], complete: true}));
});
