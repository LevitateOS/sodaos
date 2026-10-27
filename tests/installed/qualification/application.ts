// Reuse an installed App through its existing owners. Soda owns registration
// and lifecycle; Canlang owns the separate account, team and application record.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {applicationResponse, type ApplicationRecord} from '../../../frontend/spaces/sodaspaces-application-response';
import {object} from '../sodaspaces-input';
import type {ApplicationSelection, PublicArtifact, QualificationScenario} from './scenario';
import type {Readiness} from './readiness';
import {publicBytes, scenarioInput} from './validate';

export interface ApplicationFixture {
  readonly reference: PublicArtifact;
  readonly name: string;
  readonly origin: string;
  readonly image: string;
  readonly runtime_image: string;
  readonly artifact_sha256: string;
  /** Semantic bundle digest, distinct from the bundle file's byte digest. */
  readonly bundle_sha256: string;
  readonly bundle_file_sha256: string;
  readonly tools_sha256: string;
}
export type ApplicationTransport =
  | {
      readonly distribution: 'appliance';
      readonly host_uuid: string;
      readonly data_namespace: 'appliance';
      readonly data_directory: string;
    }
  | {
      readonly distribution: 'docker';
      readonly host_uuid: string;
      readonly outer_container_id: string;
      readonly outer_container_name: 'sodaos-soda-1';
      readonly outer_image_reference: string;
      readonly data_namespace: 'soda-container';
      readonly data_directory: string;
    };
export interface ApplicationHandoff {
  readonly format: 1;
  readonly state: 'adopted';
  readonly fixture: PublicArtifact;
  readonly application: ApplicationRecord['application'];
  readonly administrator_id: string;
  readonly container_id: string;
  readonly observed_at: string;
  readonly target_source_revision: string;
  readonly target_payload_sha256: string;
  readonly transport: ApplicationTransport;
  readonly owner: {
    readonly team_id: string;
    readonly credentials_namespace: 'controller';
    readonly credentials_file: string;
  };
  readonly record: ApplicationSelection['record'];
}
const hex = /^[a-f0-9]{64}$/;
const image = /^sha256:[a-f0-9]{64}$/;

/** Read the retained publisher metadata, not an executable per-attempt helper. */
export async function applicationFixture(
  s: QualificationScenario,
  reference: PublicArtifact,
  admittedBytes?: Uint8Array
): Promise<ApplicationFixture> {
  const bytes = admittedBytes === undefined ? await publicBytes(reference) : Buffer.from(admittedBytes);
  assert(
    bytes.length <= 16 << 20 && createHash('sha256').update(bytes).digest('hex') === reference.sha256,
    'application_fixture_bytes_changed'
  );
  const value = object(JSON.parse(bytes.toString()));
  assert(
    value.source === s.candidate.source_revision && value.payloadSha256 === s.candidate.payload_sha256,
    'application_fixture_candidate_changed'
  );
  assert(value.name === 'SodaRuntime', 'qualified_application_fixture_required');
  assert(
    typeof value.origin === 'string' &&
      new URL(value.origin).origin === value.origin &&
      value.origin.startsWith('https://')
  );
  assert(typeof value.image === 'string' && image.test(value.image));
  assert(typeof value.runtimeImage === 'string' && image.test(value.runtimeImage));
  for (const field of ['artifactDigest', 'bundleDigest', 'bundleSHA256', 'toolsSha256'])
    assert(typeof value[field] === 'string' && hex.test(value[field]));
  // This journey reuses the qualified record fixture; it does not discover or
  // invent arbitrary App operations from a page heading or a display label.
  const fixture = object(value.fixture);
  assert(fixture.create === value.name + '.Entry.create' && fixture.read === value.name + '.Entry.read');
  return {
    reference,
    name: value.name,
    origin: value.origin,
    image: value.image,
    runtime_image: value.runtimeImage,
    artifact_sha256: value.artifactDigest as string,
    bundle_sha256: value.bundleDigest as string,
    bundle_file_sha256: value.bundleSHA256 as string,
    tools_sha256: value.toolsSha256 as string,
  };
}

function transport(s: QualificationScenario, r: Readiness, data: string): ApplicationTransport {
  const pass = (name: string) => r.checks.some((c) => c.name === name && c.state === 'pass');
  assert(
    ['target_trust', 'native_observation', 'installed_identity', 'installed_images'].every(pass),
    'application_target_observation_unconfirmed'
  );
  const native = object(r.native);
  assert(
    native.uuid === s.target.uuid &&
      native.distribution === s.target.distribution &&
      native.source_revision === s.candidate.source_revision &&
      native.payload_sha256 === s.candidate.payload_sha256,
    'application_target_binding_changed'
  );
  const data_directory = '/home/soda-applications/data/' + data;
  if (s.target.distribution === 'appliance') {
    assert(native.docker === undefined || native.docker === null, 'native_application_has_docker_outer');
    return {distribution: 'appliance', host_uuid: s.target.uuid, data_namespace: 'appliance', data_directory};
  }
  assert(pass('docker_identity'), 'application_docker_observation_unconfirmed');
  const outer = object(native.docker);
  assert(
    typeof outer.container_id === 'string' &&
      hex.test(outer.container_id) &&
      outer.container_name === 'sodaos-soda-1' &&
      typeof outer.reference === 'string' &&
      /^ghcr\.io\/levitateos\/sodaos@sha256:[a-f0-9]{64}$/.test(outer.reference),
    'application_outer_binding_changed'
  );
  return {
    distribution: 'docker',
    host_uuid: s.target.uuid,
    outer_container_id: outer.container_id,
    outer_container_name: outer.container_name,
    outer_image_reference: outer.reference,
    data_namespace: 'soda-container',
    data_directory,
  };
}

/** Read-only adoption. It neither installs an image, opens first-owner setup,
 * reads a password nor starts/stops an App. Those effects retain their owners. */
export async function adoptApplication(
  scenario: QualificationScenario,
  readiness: Readiness,
  applicationID: string,
  runtime: {
    get(route: string): Promise<unknown>;
    save(relative: string, value: unknown): Promise<unknown>;
    /** Worker-namespace copy from the root-admitted public runtime inventory. */
    fixtureBytes?: Uint8Array;
  }
): Promise<ApplicationHandoff> {
  const s = scenarioInput(scenario);
  const selections = s.applications?.filter((a) => a.application_id === applicationID) ?? [];
  assert(s.cases.includes('app') && selections.length === 1, 'application_selection_missing');
  const selected = selections[0]!;
  const observed = readiness.observation?.inventory.applications.filter((a) => a.id === applicationID) ?? [];
  assert(observed.length === 1, 'selected_application_missing');
  const nativeTransport = transport(s, readiness, observed[0]!.data_id);
  const fixture = await applicationFixture(s, selected.fixture, runtime.fixtureBytes);
  const record = applicationResponse(await runtime.get('/api/applications/' + applicationID));
  assert(!record.restore, 'application_restore_pending');
  const actual = record.application;
  assert(
    actual.id === applicationID &&
      actual.data === observed[0]!.data_id &&
      actual.owner_team === observed[0]!.owner_team &&
      actual.origin === observed[0]!.origin,
    'application_inventory_changed'
  );
  for (const key of [
    'name',
    'origin',
    'image',
    'runtime_image',
    'artifact_sha256',
    'bundle_sha256',
    'tools_sha256',
  ] as const)
    assert(actual[key] === fixture[key], 'application_fixture_bytes_changed');
  const native = object(readiness.native);
  const rows = Array.isArray(native.application_runtime)
    ? native.application_runtime.map(object).filter((a) => a.id === actual.id)
    : [];
  assert(
    rows.length === 1 &&
      record.observation &&
      hex.test(record.observation.container) &&
      rows[0]!.container_id === record.observation.container &&
      rows[0]!.running === record.observation.running,
    'application_native_generation_unconfirmed'
  );
  const handoff: ApplicationHandoff = {
    format: 1,
    state: 'adopted',
    fixture: fixture.reference,
    application: actual,
    administrator_id: record.administrator_id,
    container_id: record.observation.container,
    observed_at: readiness.checked_at,
    target_source_revision: s.candidate.source_revision,
    target_payload_sha256: s.candidate.payload_sha256,
    transport: nativeTransport,
    owner: {
      team_id: actual.owner_team,
      credentials_namespace: 'controller',
      credentials_file: selected.owner_credentials_file,
    },
    record: selected.record,
  };
  await runtime.save('app/' + applicationID + '/handoff.json', handoff);
  return handoff;
}

/** The only remote App command in this boundary is a fixed read-only probe.
 * Docker host paths must never be substituted for paths inside its Soda outer. */
export function applicationNativeProbe(handoff: ApplicationHandoff): string {
  const {application: app, transport: t} = handoff;
  // Reuse the canonical App response parser for all Spec/observation fields.
  applicationResponse({
    application: app,
    administrator_id: handoff.administrator_id,
    operator: false,
    observation: null,
  });
  assert(handoff.owner.team_id === app.owner_team && hex.test(handoff.container_id));
  assert(
    handoff.format === 1 &&
      handoff.owner.credentials_namespace === 'controller' &&
      /^[a-f0-9]{40}$/.test(handoff.target_source_revision) &&
      hex.test(handoff.target_payload_sha256)
  );
  assert(t.data_directory === '/home/soda-applications/data/' + app.data);
  assert(/^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(t.host_uuid));
  assert(t.distribution === 'appliance' || t.distribution === 'docker', 'explicit_application_distribution_required');
  assert(
    t.distribution === 'appliance'
      ? t.data_namespace === 'appliance'
      : t.data_namespace === 'soda-container' &&
          hex.test(t.outer_container_id) &&
          t.outer_container_name === 'sodaos-soda-1' &&
          /^ghcr\.io\/levitateos\/sodaos@sha256:[a-f0-9]{64}$/.test(t.outer_image_reference)
  );
  const inner =
    'const APP=' +
    JSON.stringify(app) +
    ',EXPECTED=' +
    JSON.stringify(handoff.container_id) +
    ',INSTALLED=' +
    JSON.stringify({source: handoff.target_source_revision, payload: handoff.target_payload_sha256}) +
    ';\n' +
    String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
try{if(process.getuid()!==0)throw Error();
const release=fs.readFileSync('/usr/share/soda/release.json');if(JSON.parse(release).Revision!==INSTALLED.source||crypto.createHash('sha256').update(release).digest('hex')!==INSTALLED.payload)throw Error();
const format='{"id":{{json .ID}},"image":{{json .Image}},"running":{{json .State.Running}},"labels":{{json .Config.Labels}}}';
const value=JSON.parse(cp.execFileSync('/usr/bin/podman',['--remote=false','container','inspect','--format',format,'soda-canlang-'+APP.id],{encoding:'utf8',timeout:20000,maxBuffer:16384,stdio:['ignore','pipe','pipe']}));
if(value.id!==EXPECTED||'sha256:'+value.image.replace(/^sha256:/,'')!==APP.image)throw Error();
for(const[key,expected]of Object.entries({'io.soda.canlang.id':APP.id,'io.soda.canlang.data':APP.data,'io.soda.canlang.owner':APP.owner_team,'io.soda.canlang.origin':APP.origin,'io.soda.canlang.port':String(APP.port),'io.soda.canlang.revision':String(APP.revision)}))if(value.labels[key]!==expected)throw Error();
const data=fs.lstatSync('/home/soda-applications/data/'+APP.data);if(!data.isDirectory()||data.isSymbolicLink()||data.uid!==10000||(data.mode&511)!==448)throw Error();
console.log(JSON.stringify({application_id:APP.id,container_id:value.id,image:APP.image,running:value.running,data_id:APP.data,owner_team:APP.owner_team,origin:APP.origin}));
}catch{console.error('application_native_observation_unconfirmed');process.exitCode=1;}`;
  return (
    'const TARGET=' +
    JSON.stringify(t) +
    ',INNER=' +
    JSON.stringify(inner) +
    ';\n' +
    String.raw`
const fs=require('node:fs'),cp=require('node:child_process');
try{if(process.getuid()!==0||fs.readFileSync('/sys/class/dmi/id/product_uuid','utf8').trim().toLowerCase()!==TARGET.host_uuid)throw Error();
if(TARGET.distribution==='appliance'){
 if(fs.existsSync('/run/systemd/container')&&fs.readFileSync('/run/systemd/container','utf8').trim()==='docker')throw Error();
 process.stdout.write(cp.execFileSync('/usr/bin/node',['-'],{input:INNER,encoding:'utf8',timeout:30000,maxBuffer:16384,stdio:['pipe','pipe','pipe']}));
}else{
 const d=a=>cp.execFileSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock',...a],{encoding:'utf8',timeout:20000,maxBuffer:16384,stdio:['ignore','pipe','pipe']});
 const outer=JSON.parse(d(['inspect','--type','container','--format','{"id":{{json .Id}},"name":{{json .Name}},"reference":{{json .Config.Image}}}',TARGET.outer_container_id]));
 if(outer.id!==TARGET.outer_container_id||outer.name!=='/'+TARGET.outer_container_name||outer.reference!==TARGET.outer_image_reference)throw Error();
 process.stdout.write(cp.execFileSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock','exec','-i',TARGET.outer_container_id,'/usr/bin/node','-'],{input:INNER,encoding:'utf8',timeout:30000,maxBuffer:16384,stdio:['pipe','pipe','pipe']}));
}}catch{console.error('application_target_transport_unconfirmed');process.exitCode=1;}`
  );
}
