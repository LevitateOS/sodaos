// Native transport has no outer Docker container. The same retained-state
// reader owns resource/custody/ledger semantics for both installed topologies.
import assert from 'node:assert/strict';
import {assertResourceInventory, diagnoseClosure, type ResourceScope} from '../closure';
import type {QualificationScenario, PublicArtifact} from '../scenario';
import {publicBytes} from '../validate';
import {check, sha, SHA} from './support';
import {INNER_READER} from './docker-native';
import {validateWitness, type DockerIO, type Witness} from './docker-binding';

const UUID = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/;
export interface ApplianceInstallation {
  readonly candidate: PublicArtifact;
  readonly payload: PublicArtifact;
  readonly media: PublicArtifact;
  readonly console: PublicArtifact;
  readonly release: PublicArtifact;
}
export type ApplianceBinding = Awaited<ReturnType<typeof loadApplianceBinding>>;

/** The controller supplies public paths; none of the private per-attempt
 * implementation or credential files is imported by this binding. */
export async function loadApplianceBinding(s: QualificationScenario, files: ApplianceInstallation) {
  check(s.target.distribution === 'appliance', 'appliance_scenario_required');
  const bytes = await Promise.all([files.candidate, files.payload, files.media, files.console].map(publicBytes));
  const [candidate, payload, media, installation] = bytes.map((b) => JSON.parse(b.toString()));
  check(
    files.candidate.sha256 === s.candidate.candidate_sha256 && files.payload.sha256 === s.candidate.payload_sha256,
    'appliance_candidate_binding'
  );
  check(
    candidate.PayloadSHA256 === files.payload.sha256 &&
      payload.Revision === s.candidate.source_revision &&
      candidate.Host?.Revision === payload.Revision &&
      media.Revision === payload.Revision &&
      media.PayloadSHA256 === files.payload.sha256 &&
      SHA.test(media.DeploymentCommit),
    'appliance_media_binding'
  );
  check(files.release.sha256 === s.candidate.release_manifest.sha256, 'appliance_admitted_release_manifest');
  // The controller-local scenario path is not mounted in the protected worker.
  // Read the explicitly staged public copy, keeping its original byte identity.
  const manifest = JSON.parse((await publicBytes(files.release)).toString());
  check(
    manifest.soda_revision === payload.Revision &&
      Array.isArray(manifest.assets) &&
      [
        ['candidate.json', files.candidate],
        ['payload.json', files.payload],
        ['media.json', files.media],
      ].every(([name, reference]) =>
        manifest.assets.some((a: any) => a.name === name && a.sha256 === (reference as PublicArtifact).sha256)
      ),
    'appliance_public_release_binding'
  );
  check(
    installation.format === 1 &&
      installation.action === 'guided-public-appliance-install' &&
      installation.domain === s.target.name &&
      installation.domain_uuid === s.target.uuid &&
      installation.source_revision === payload.Revision &&
      installation.tag === payload.ReleaseTag &&
      installation.iso_sha256 === media.ISO.SHA256 &&
      installation.rootfs_sha256 === media.Rootfs.SHA256 &&
      installation.media_payload_sha256 === files.payload.sha256 &&
      installation.first_boot?.dmi_uuid === s.target.uuid &&
      UUID.test(installation.first_boot.boot_id) &&
      UUID.test(installation.first_boot.root_partuuid) &&
      installation.first_boot.deployment === media.DeploymentCommit,
    'appliance_original_installation_binding'
  );
  check(
    payload.Images &&
      Object.keys(payload.Images).sort().join(',') === 'canlang-runtime,dashboard,forgejo,postgres,project-os,proxy',
    'six_appliance_images'
  );
  return {
    target: s.target,
    source: payload.Revision as string,
    payloadSHA: files.payload.sha256,
    payload,
    candidate,
    media,
    installation,
    public_files: files,
  };
}

/** Direct pinned SSH interpreter. Never run the program on the controller or
 * inside an outer container when the selected distribution is an appliance. */
export function applianceProgram(uuid: string, program: string): string {
  check(UUID.test(uuid) && program.length > 0 && program.length <= 1 << 20, 'appliance_fixed_program');
  return (
    'const UUID=' +
    JSON.stringify(uuid) +
    ';const PROGRAM=' +
    JSON.stringify(program) +
    ';\n' +
    String.raw`
const fs=require('node:fs'),cp=require('node:child_process');try{
 if(process.getuid()!==0||fs.existsSync('/run/systemd/container')||fs.readFileSync('/sys/class/dmi/id/product_uuid','utf8').trim().toLowerCase()!==UUID)throw Error();
 process.stdout.write(cp.execFileSync('/usr/bin/node',['-'],{input:PROGRAM,encoding:'utf8',timeout:180000,maxBuffer:4<<20,stdio:['pipe','pipe','pipe']}));
}catch{console.error('selected_appliance_command_unconfirmed');process.exitCode=1;}`
  );
}

const BOOT_READER = String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
const run=(p,a)=>cp.execFileSync(p,a,{encoding:'utf8',timeout:30000,maxBuffer:2<<20,stdio:['ignore','pipe','pipe']}).trim();
try{
 if(process.getuid()!==0||fs.existsSync('/run/systemd/container')||fs.readFileSync('/proc/cmdline','utf8').split(/\s+/).some(x=>x.startsWith('coreos.live')||x==='rd.live.image'))throw Error();
 const marker=fs.lstatSync('/etc/soda/installed');if(!marker.isFile()||marker.isSymbolicLink())throw Error();
 const root=JSON.parse(run('/usr/bin/findmnt',['--json','--target','/sysroot','-o','SOURCE,FSTYPE,UUID'])).filesystems;if(root.length!==1||root[0].fstype!=='xfs')throw Error();
 const source=root[0].source.replace(/\[.*$/,'');if(!source.startsWith('/dev/'))throw Error();
 const blocks=JSON.parse(run('/usr/bin/lsblk',['--json','--bytes','--paths','-o','NAME,PATH,TYPE,SIZE,PKNAME,FSTYPE,UUID,PARTUUID'])).blockdevices,rows=[];
 const walk=(x,parent)=>{rows.push({...x,parent});for(const c of x.children||[])walk(c,x.path||x.name)};for(const d of blocks)walk(d,null);
 const device=rows.find(x=>(x.path||x.name)===source),disk=rows.find(x=>(x.path||x.name)===device?.parent);if(device?.type!=='part'||!device.partuuid||disk?.type!=='disk')throw Error();
 const booted=JSON.parse(run('/usr/bin/ostree',['admin','status','--json'])).deployments.filter(x=>x.booted);if(booted.length!==1||(booted[0].unlocked&&booted[0].unlocked!=='none'))throw Error();
 const bytes=fs.readFileSync('/usr/share/soda/release.json'),payload=JSON.parse(bytes),images={};for(const [role,id]of Object.entries(payload.Images)){const actual=run('/usr/bin/podman',['--remote=false','image','inspect','--format','{{.ID}}',id.Config]).replace(/^sha256:/,'');if(actual!==id.Config.slice(7))throw Error();images[role]=id.Config;}
 console.log(JSON.stringify({uuid:fs.readFileSync('/sys/class/dmi/id/product_uuid','utf8').trim().toLowerCase(),boot_id:fs.readFileSync('/proc/sys/kernel/random/boot_id','utf8').trim(),deployment:booted[0].checksum,root:{partuuid:device.partuuid,uuid:root[0].uuid,filesystem:root[0].fstype,disk:disk.path||disk.name,bytes:Number(disk.size)},images,source:payload.Revision,payload_sha256:crypto.createHash('sha256').update(bytes).digest('hex')}));
}catch{console.error('installed_appliance_boot_unconfirmed');process.exitCode=1;}`;

export async function observeApplianceBoot(io: DockerIO, b: ApplianceBinding) {
  const boot = JSON.parse(await io.sshRead(applianceProgram(b.target.uuid, BOOT_READER), 210000));
  check(
    boot.uuid === b.target.uuid &&
      UUID.test(boot.boot_id) &&
      boot.deployment === b.media.DeploymentCommit &&
      boot.source === b.source &&
      boot.payload_sha256 === b.payloadSHA &&
      boot.root?.filesystem === 'xfs' &&
      boot.root.partuuid === b.installation.first_boot.root_partuuid &&
      boot.root.bytes === b.installation.first_boot.disk_bytes,
    'same_installed_appliance_boot'
  );
  assert.deepEqual(
    boot.images,
    Object.fromEntries(
      Object.entries(b.payload.Images).map(([role, image]) => [role, (image as {Config: string}).Config])
    )
  );
  return boot;
}

export function applianceStateScript(
  b: ApplianceBinding,
  mode: 'installation' | 'idle' | 'snapshot',
  scope: ResourceScope,
  w?: Witness
) {
  check(
    scope.scenario.target.uuid === b.target.uuid && scope.scenario.target.distribution === 'appliance',
    'appliance_resource_scope'
  );
  const request = {
    source: b.source,
    payloadSHA: b.payloadSHA,
    mode,
    distribution: 'appliance',
    factorySelected: scope.scenario.factory.length > 0,
    resources: {projects: scope.observation.inventory.projects, applications: scope.observation.inventory.applications},
    ...(w ? {witness: validateWitness(w)} : {}),
  };
  return applianceProgram(b.target.uuid, 'const REQUEST=' + JSON.stringify(request) + ';\n' + INNER_READER);
}
export async function readAppliance(
  io: DockerIO,
  b: ApplianceBinding,
  mode: 'installation' | 'idle' | 'snapshot',
  scope: ResourceScope,
  w?: Witness
) {
  const inner = JSON.parse(await io.sshRead(applianceStateScript(b, mode, scope, w), 210000));
  check(inner.source === b.source && inner.payload_sha256 === b.payloadSHA, 'appliance_state_binding');
  for (const [field, path] of [
    ['host_sha256', 'host:/usr/libexec/soda/soda-host'],
    ['maintain_sha256', 'host:/usr/libexec/soda/soda-project-maintain'],
    ['dashboard_sha256', 'dashboard:/usr/local/bin/soda-dashboard'],
  ]) {
    const pin = b.candidate.ContentSHA256?.[path!];
    if (field === 'dashboard_sha256' || pin !== undefined)
      check(SHA.test(pin) && inner[field!] === pin, 'appliance_stock_binary_bytes');
  }
  if (mode !== 'installation') {
    assertResourceInventory(scope, inner.resource_inventory);
    const closure = diagnoseClosure(scope.scenario, scope.observation, inner);
    if (mode === 'snapshot') {
      check(
        closure.snapshot_ready && inner.protected_sha256 === sha(JSON.stringify(inner.protected)),
        'appliance_stopped_snapshot'
      );
    }
  }
  return {inner};
}
