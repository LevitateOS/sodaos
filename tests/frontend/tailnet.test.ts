import assert from 'node:assert/strict';
import test from 'node:test';
import {authenticationURL, hostResult, settingsView} from '../../frontend/tailnet/soda-tailnet-response';
import type {Settings} from '../../frontend/tailnet/soda-tailnet-response';

const snapshot = (): Settings => ({
  host_unavailable: false,
  host: {
    revision: 'a'.repeat(64),
    state: 'Running',
    have_node_key: true,
    expired: false,
    tailnet: 'soda.example.test',
    magic_dns_enabled: true,
    dns_name: 'appliance.soda.ts.net',
    addresses: ['100.64.0.1'],
    health_issues: 1,
    preferences: {want_running: true, exit_node_id: '', exit_node_ip: '', allow_lan: false, advertise_exit_node: false},
    peers: [
      {
        id: 'exit',
        dns_name: 'exit.soda.ts.net',
        addresses: ['100.64.0.2'],
        online: true,
        exit_node: true,
        expired: false,
      },
    ],
  },
  enrollment: {
    revision: '0',
    binding: '',
    tailnet: '',
    tags: [],
    configured: false,
    admission: false,
    default: false,
    preauthorized: false,
    credential_checked: false,
    enrollment_verified: false,
    runtime_supported: false,
  },
});

test('Tailnet projections reject malformed, unsafe and mixed-version responses', () => {
  const value = snapshot();
  assert.deepEqual(settingsView(value), value);
  assert.throws(() => settingsView({host: null, enrollment: value.enrollment}));
  assert.equal(
    settingsView({...value, enrollment: {...value.enrollment, runtime_supported: true}}).enrollment.runtime_supported,
    true
  );
  assert.throws(() => settingsView({...value, enrollment: {...value.enrollment, default: true}}));
  assert.throws(() => settingsView({...value, host: {...value.host, state: 'invented'}}));
  for (const url of [
    'http://login.tailscale.com/a/test',
    'https://evil.test/a/test',
    'https://user@login.tailscale.com/a/test',
    'https://login.tailscale.com/a/test?secret=x',
    'https://login.tailscale.com/a/test#secret',
  ])
    assert.throws(() => authenticationURL(url));
  assert.throws(() => hostResult({outcome: 'pending', host: value.host, readback_unavailable: false}, 'signin'));
  assert.throws(() =>
    hostResult(
      {
        outcome: 'confirmed',
        host: value.host,
        readback_unavailable: false,
        auth_url: 'https://login.tailscale.com/a/test',
      },
      'logout'
    )
  );
  const safe = settingsView({
    ...value,
    host: {...value.host, auth_url: 'must-not-project', Health: ['private diagnostic']},
  });
  assert(!JSON.stringify(safe).includes('must-not-project'));
  assert(!JSON.stringify(safe).includes('private diagnostic'));
});

test('Tailnet projections reject malformed, unsafe and mixed-version responses', () => {
  const value = snapshot();
  assert.deepEqual(settingsView(value), value);
  assert.throws(() => settingsView({host: null, enrollment: value.enrollment}));
  assert.equal(
    settingsView({...value, enrollment: {...value.enrollment, runtime_supported: true}}).enrollment.runtime_supported,
    true
  );
  assert.throws(() => settingsView({...value, enrollment: {...value.enrollment, default: true}}));
  assert.throws(() => settingsView({...value, host: {...value.host, state: 'invented'}}));
  for (const url of [
    'http://login.tailscale.com/a/test',
    'https://evil.test/a/test',
    'https://user@login.tailscale.com/a/test',
    'https://login.tailscale.com/a/test?secret=x',
    'https://login.tailscale.com/a/test#secret',
  ])
    assert.throws(() => authenticationURL(url));
  assert.throws(() => hostResult({outcome: 'pending', host: value.host, readback_unavailable: false}, 'signin'));
  assert.throws(() =>
    hostResult(
      {
        outcome: 'confirmed',
        host: value.host,
        readback_unavailable: false,
        auth_url: 'https://login.tailscale.com/a/test',
      },
      'logout'
    )
  );
  const safe = settingsView({
    ...value,
    host: {...value.host, auth_url: 'must-not-project', Health: ['private diagnostic']},
  });
  assert(!JSON.stringify(safe).includes('must-not-project'));
  assert(!JSON.stringify(safe).includes('private diagnostic'));
});
