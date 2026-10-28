import assert from 'node:assert/strict';
import path from 'node:path';
import {lstat, readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {object, validID} from '../sodaspaces-input';
import type {PublicArtifact, QualificationScenario} from './scenario';

export const sha256 = (bytes: Uint8Array) => createHash('sha256').update(bytes).digest('hex');
const hex = /^[a-f0-9]{64}$/;
const revision = /^[a-f0-9]{40}$/;
const project = /^p[a-f0-9]{24}$/;
const opaque = /^[a-f0-9]{32}$/;
const decision = /^[df][a-f0-9]{24}$/;
const nonempty = (value: unknown): value is string => typeof value === 'string' && value.length > 0;
const absolute = (value: unknown): value is string =>
  nonempty(value) && value.startsWith('/home/') && path.normalize(value) === value && !/[\0\r\n]/.test(value);
const strings = (value: unknown, valid: (v: string) => boolean) =>
  Array.isArray(value) && value.every((v) => typeof v === 'string' && valid(v)) && new Set(value).size === value.length;
const positive = (n: unknown): n is number => typeof n === 'number' && Number.isSafeInteger(n) && n > 0;
function keys(value: Record<string, unknown>, allowed: readonly string[]) {
  assert(
    Object.keys(value).every((k) => allowed.includes(k)),
    'Unknown scenario field'
  );
}
function artifact(value: unknown): asserts value is PublicArtifact {
  const v = object(value);
  keys(v, ['path', 'sha256']);
  assert(absolute(v.path) && typeof v.sha256 === 'string' && hex.test(v.sha256), 'Invalid public artifact');
}

/** Validate before reading credentials, launching a transport or creating permits. */
export function scenarioInput(value: unknown): QualificationScenario {
  const s = object(value);
  keys(s, [
    'format',
    'candidate',
    'driver',
    'target',
    'trust',
    'cases',
    'managed',
    'factory',
    'applications',
    'credentials',
    'permitted_effects',
    'authority',
    'collection',
  ]);
  assert.equal(s.format, 1);
  const c = object(s.candidate),
    d = object(s.driver),
    t = object(s.target);
  keys(c, ['source_revision', 'candidate_sha256', 'payload_sha256', 'release_manifest']);
  keys(d, ['source_revision', 'executable']);
  keys(t, ['distribution', 'name', 'uuid', 'origin', 'ssh_config_file']);
  assert(typeof c.source_revision === 'string' && revision.test(c.source_revision));
  assert(typeof c.candidate_sha256 === 'string' && hex.test(c.candidate_sha256));
  assert(typeof c.payload_sha256 === 'string' && hex.test(c.payload_sha256));
  assert(typeof d.source_revision === 'string' && revision.test(d.source_revision));
  artifact(c.release_manifest);
  artifact(d.executable);
  assert(t.distribution === 'docker' || t.distribution === 'appliance');
  assert(typeof t.name === 'string' && /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(t.name));
  assert(typeof t.uuid === 'string' && /^[a-f0-9]{8}(-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(t.uuid));
  assert(typeof t.origin === 'string' && new URL(t.origin).origin === t.origin && t.origin.startsWith('https://'));
  assert(absolute(t.ssh_config_file));
  if (s.trust !== undefined) {
    const trust = object(s.trust);
    keys(trust, ['known_hosts', 'appliance_ca']);
    artifact(trust.known_hosts);
    artifact(trust.appliance_ca);
  }
  assert(Array.isArray(s.cases));
  assert(
    strings(s.cases, (v) =>
      [
        'installation',
        'browser_project',
        'factory',
        'persistence',
        'app',
        'basic_capacity',
        'two_issue_capacity',
        'cold_recovery',
      ].includes(v)
    ) && s.cases.length > 0
  );
  const managed = object(s.managed);
  keys(managed, ['project_ids', 'application_ids', 'terminal_ids']);
  assert(strings(managed.project_ids, (v) => project.test(v)));
  assert(strings(managed.application_ids, (v) => /^a[a-f0-9]{24}$/.test(v)));
  assert(strings(managed.terminal_ids, (v) => opaque.test(v)));
  assert(Array.isArray(s.factory));
  const selections = new Set<string>(),
    operations = new Set<string>();
  for (const value of s.factory) {
    const f = object(value);
    keys(f, ['subject', 'limits', 'execution']);
    if (f.execution !== undefined) {
      const e = object(f.execution),
        p = object(e.profile);
      keys(e, ['harness', 'model', 'profile']);
      keys(p, ['effort', 'contributor']);
      assert(e.harness === 'muse');
      assert(typeof e.model === 'string' && /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/.test(e.model));
      assert(['low', 'medium', 'high', 'xhigh'].includes(String(p.effort)) && typeof p.contributor === 'boolean');
    }
    const b = object(f.subject),
      l = object(f.limits);
    keys(b, [
      'repository_id',
      'project_id',
      'issue_number',
      'native_issue_id',
      'source_commit',
      'acceptance',
      'requirements',
      'approval',
      'coder_preparation',
      'reviewer_preparation',
      'connection_id',
      'operation_id',
      'actors',
      'required_checks',
      'check_script_sha256',
      'check_script_path',
    ]);
    assert(validID(b.repository_id) && validID(b.native_issue_id) && positive(b.issue_number));
    assert(typeof b.project_id === 'string' && project.test(b.project_id));
    assert(typeof b.source_commit === 'string' && revision.test(b.source_commit));
    for (const name of ['acceptance', 'requirements', 'approval', 'coder_preparation', 'reviewer_preparation'])
      assert(typeof b[name] === 'string' && decision.test(b[name] as string));
    assert(typeof b.connection_id === 'string' && opaque.test(b.connection_id));
    assert(typeof b.operation_id === 'string' && opaque.test(b.operation_id));
    const selection = `${b.repository_id}/${b.issue_number}`;
    assert(!selections.has(selection) && !operations.has(b.operation_id), 'Duplicate Factory selection');
    selections.add(selection);
    operations.add(b.operation_id);
    assert(typeof b.check_script_sha256 === 'string' && hex.test(b.check_script_sha256));
    assert(
      b.check_script_path === undefined ||
        (typeof b.check_script_path === 'string' &&
          /^[A-Za-z0-9_][A-Za-z0-9_./-]{0,255}$/.test(b.check_script_path) &&
          !b.check_script_path.includes('..'))
    );
    assert(strings(b.required_checks, (v) => /^[A-Za-z0-9_][A-Za-z0-9_./-]{0,255}$/.test(v) && !v.includes('..')));
    const actors = object(b.actors);
    keys(actors, ['coder', 'reviewer', 'ci']);
    assert(['coder', 'reviewer', 'ci'].every((k) => validID(actors[k])) && new Set(Object.values(actors)).size === 3);
    keys(l, [
      'sponsor_minutes',
      'rolling_minutes',
      'max_concurrent',
      'intent_minutes',
      'attempt_minutes',
      'maximum_seconds',
      'automatic_retry',
      'automatic_refill',
      'extra_inquiries',
    ]);
    assert(
      [
        'sponsor_minutes',
        'rolling_minutes',
        'max_concurrent',
        'intent_minutes',
        'attempt_minutes',
        'maximum_seconds',
      ].every((k) => positive(l[k]))
    );
    assert(l.automatic_retry === false && l.automatic_refill === false && l.extra_inquiries === false);
  }
  const auth = object(s.authority);
  keys(auth, ['held', 'scope_receipt_file']);
  assert(typeof auth.held === 'boolean' && (auth.scope_receipt_file === null || absolute(auth.scope_receipt_file)));
  if (s.collection !== undefined) {
    const collection = object(s.collection);
    const fields = ['authority_file', 'candidate_directory', 'output_directory', 'build_receipt_file'];
    keys(collection, [...fields, 'prior_attempts']);
    assert(fields.every((name) => absolute(collection[name])));
    if (collection.prior_attempts !== undefined) {
      assert(Array.isArray(collection.prior_attempts));
      for (const value of collection.prior_attempts) {
        const prior = object(value);
        keys(prior, ['request', 'executed', 'release_evidence']);
        artifact(prior.request);
        for (const name of ['executed', 'release_evidence']) if (prior[name] !== null) artifact(prior[name]);
      }
    }
  }
  const creds = object(s.credentials);
  keys(creds, ['guest_ssh_key_file', 'workspace_owner_password_file', 'application_owner_files']);
  assert(
    absolute(creds.guest_ssh_key_file) &&
      absolute(creds.workspace_owner_password_file) &&
      strings(creds.application_owner_files, absolute)
  );
  if (s.applications !== undefined) {
    assert(Array.isArray(s.applications) && s.applications.length <= 16);
    const selected = new Set<string>();
    for (const value of s.applications) {
      const a = object(value),
        record = object(a.record);
      keys(a, ['application_id', 'fixture', 'owner_credentials_file', 'record']);
      keys(record, ['operation_id', 'label']);
      assert(typeof a.application_id === 'string' && /^a[a-f0-9]{24}$/.test(a.application_id));
      assert(!selected.has(a.application_id) && (managed.application_ids as string[]).includes(a.application_id));
      selected.add(a.application_id);
      artifact(a.fixture);
      assert(
        absolute(a.owner_credentials_file) &&
          (creds.application_owner_files as string[]).includes(a.owner_credentials_file)
      );
      assert(
        typeof record.operation_id === 'string' &&
          /^[a-f0-9]{8}-[a-f0-9]{4}-7[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/.test(record.operation_id)
      );
      assert(
        typeof record.label === 'string' &&
          record.label.length >= 1 &&
          record.label.length <= 200 &&
          record.label.trim() === record.label &&
          !/[\u0000-\u001f\u007f]/u.test(record.label)
      );
    }
    assert(s.applications.length === 0 || s.cases.includes('app'));
  }
  const effects = object(s.permitted_effects);
  keys(effects, [
    'fixture_setup',
    'project_lifecycle',
    'application_lifecycle',
    'terminal_lifecycle',
    'factory_repositories',
    'current_context_phases',
    'same_image_recreation',
    'native_reboot',
    'retire_consumed_scratch',
    'retire_temporary_clone',
    'cold_restore',
  ]);
  for (const k of ['fixture_setup', 'same_image_recreation', 'cold_restore']) assert(typeof effects[k] === 'boolean');
  assert(effects.native_reboot === undefined || typeof effects.native_reboot === 'boolean');
  for (const key of ['retire_consumed_scratch', 'retire_temporary_clone'])
    assert(effects[key] === undefined || typeof effects[key] === 'boolean');
  for (const k of ['project_lifecycle', 'application_lifecycle', 'terminal_lifecycle', 'factory_repositories'])
    assert(strings(effects[k], nonempty));
  assert(strings(effects.current_context_phases, (v) => v === 'implement' || v === 'review'));
  return value as QualificationScenario;
}

export async function publicBytes(reference: PublicArtifact): Promise<Buffer> {
  const st = await lstat(reference.path);
  assert(st.isFile() && st.nlink === 1 && st.size <= 16 << 20, 'Bound public metadata file required');
  const bytes = await readFile(reference.path);
  assert.equal(sha256(bytes), reference.sha256, 'Public metadata identity changed');
  return bytes;
}

/** Parse a single explicit pinned alias; never let SSH execute config directives. */
export async function pinnedSSH(s: QualificationScenario): Promise<string[]> {
  assert(s.trust, 'Pinned target trust required');
  const st = await lstat(s.target.ssh_config_file);
  assert(st.isFile() && !(st.mode & 0o077) && st.size <= 8192);
  const fields = new Map<string, string>();
  for (const line of (await readFile(s.target.ssh_config_file, 'utf8')).split('\n')) {
    const v = line.trim();
    if (!v || v.startsWith('#')) continue;
    const pair = /^(\w+)\s+([^\s]+)$/.exec(v);
    assert(pair, 'Restricted SSH configuration required');
    const key = pair[1]!.toLowerCase();
    assert(!fields.has(key), 'One pinned SSH alias required');
    fields.set(key, pair[2]!);
  }
  assert(
    [...fields.keys()].every((k) =>
      [
        'host',
        'hostname',
        'user',
        'port',
        'identityfile',
        'identitiesonly',
        'userknownhostsfile',
        'stricthostkeychecking',
        'forwardagent',
        'forwardx11',
      ].includes(k)
    )
  );
  assert.equal(fields.get('host'), s.target.name);
  assert.equal(fields.get('hostname'), new URL(s.target.origin).hostname);
  assert.equal(fields.get('user'), 'root');
  assert.equal(fields.get('identityfile'), s.credentials.guest_ssh_key_file);
  assert.equal(fields.get('userknownhostsfile'), s.trust.known_hosts.path);
  assert.equal(fields.get('identitiesonly'), 'yes');
  assert.equal(fields.get('stricthostkeychecking'), 'yes');
  assert.equal(fields.get('forwardagent'), 'no');
  assert(fields.get('port') === undefined || /^[1-9][0-9]{0,4}$/.test(fields.get('port')!));
  const keyStat = await lstat(s.credentials.guest_ssh_key_file);
  assert(keyStat.isFile() && keyStat.nlink === 1 && (keyStat.mode & 0o777) === 0o600);
  await publicBytes(s.trust.known_hosts);
  await publicBytes(s.trust.appliance_ca);
  return [
    '/usr/bin/ssh',
    '-T',
    '-F',
    '/dev/null',
    '-p',
    fields.get('port') ?? '22',
    '-oBatchMode=yes',
    '-oIdentitiesOnly=yes',
    '-oIdentityAgent=none',
    '-oStrictHostKeyChecking=yes',
    '-oUpdateHostKeys=no',
    '-oForwardAgent=no',
    '-oForwardX11=no',
    '-oPermitLocalCommand=no',
    '-oClearAllForwardings=yes',
    '-oRemoteCommand=none',
    '-oConnectTimeout=10',
    '-oControlMaster=no',
    '-oControlPath=none',
    '-oControlPersist=no',
    '-oGlobalKnownHostsFile=/dev/null',
    '-oUserKnownHostsFile=' + s.trust.known_hosts.path,
    '-i',
    s.credentials.guest_ssh_key_file,
    'root@' + fields.get('hostname'),
    '/usr/bin/node -',
  ];
}
