// Native reboot owns its kernel/installed-root evidence. Docker recreation is
// never reported as a native reboot, nor a reboot as physical power-off.
import assert from 'node:assert/strict';
import type {ResourceScope} from '../closure';
import {check, sha} from './support';
import type {DockerIO, Witness} from './docker-binding';
import {
  applianceProgram,
  applianceStateScript,
  observeApplianceBoot,
  readAppliance,
  type ApplianceBinding,
} from './appliance-native';

function rebootScript(b: ApplianceBinding, before: any, boot: any, scope: ResourceScope, w: Witness) {
  const operation = scope.scenario.factory[0]!.subject.operation_id;
  check(/^[a-f0-9]{32}$/.test(operation), 'native_reboot_operation');
  return applianceProgram(
    b.target.uuid,
    'const EXPECTED=' +
      JSON.stringify({
        uuid: b.target.uuid,
        boot: boot.boot_id,
        source: b.source,
        payload: b.payloadSHA,
        protected: before.inner.protected_sha256,
        operation,
      }) +
      ';const READER=' +
      JSON.stringify(applianceStateScript(b, 'snapshot', scope, w)) +
      ';\n' +
      String.raw`
const fs=require('node:fs'),cp=require('node:child_process');
const need=x=>{if(!x)throw Error()};
try{
 need(fs.readFileSync('/proc/sys/kernel/random/boot_id','utf8').trim()===EXPECTED.boot);
 const current=JSON.parse(cp.execFileSync('/usr/bin/node',['-'],{input:READER,encoding:'utf8',timeout:180000,maxBuffer:4<<20,stdio:['pipe','pipe','pipe']}));need(current.protected_sha256===EXPECTED.protected);
 const directory='/var/lib/soda/qualification-reboots';
 try{fs.mkdirSync(directory,{mode:448});const p=fs.openSync('/var/lib/soda','r');try{fs.fsyncSync(p)}finally{fs.closeSync(p)}}catch(e){if(e.code!=='EEXIST')throw e;}
 for(let p=directory;;p=require('node:path').dirname(p)){const s=fs.lstatSync(p);need(s.isDirectory()&&!s.isSymbolicLink()&&s.uid===0&&(s.mode&18)===0);if(p==='/')break;}
 const leaf=directory+'/'+EXPECTED.operation+'.json',fd=fs.openSync(leaf,fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_EXCL|fs.constants.O_NOFOLLOW,384);
 try{fs.writeFileSync(fd,JSON.stringify({...EXPECTED,action:'native-installed-reboot',automatic_retry:false})+'\n');fs.fsyncSync(fd)}finally{fs.closeSync(fd)}
 const d=fs.openSync(directory,'r');try{fs.fsyncSync(d)}finally{fs.closeSync(d)}
 cp.execFileSync('/usr/bin/systemctl',['reboot'],{timeout:30000,stdio:['ignore','pipe','pipe']});console.log(JSON.stringify({requested:true,operation:EXPECTED.operation}));
}catch{console.error('native_reboot_request_unconfirmed');process.exitCode=1;}`
  );
}

export async function exerciseAppliancePersistence(
  b: ApplianceBinding,
  w: Witness,
  installation: any,
  io: DockerIO,
  scope: ResourceScope
) {
  check(
    !scope.scenario.authority.held &&
      scope.scenario.authority.scope_receipt_file !== null &&
      scope.scenario.permitted_effects.native_reboot === true,
    'native_reboot_execution_held'
  );
  const before = await readAppliance(io, b, 'snapshot', scope, w),
    boot = await observeApplianceBoot(io, b);
  check(
    installation.uuid === boot.uuid &&
      installation.deployment === boot.deployment &&
      installation.boot_id === boot.boot_id,
    'original_native_installation_observation'
  );
  await io.save('persistence/before.json', before);
  const operation = scope.scenario.factory[0]!.subject.operation_id;
  await io.save('persistence/reboot.request.json', {
    action: 'native-installed-reboot',
    target: b.target.name,
    operation,
    old_boot_id: boot.boot_id,
    protected_before_sha256: before.inner.protected_sha256,
    automatic_retry: false,
  });
  let acknowledgment = false;
  try {
    const reply = JSON.parse(await io.sshRead(rebootScript(b, before, boot, scope, w), 210000));
    acknowledgment = reply.requested === true && reply.operation === operation;
  } catch {
    /* Only read this exact intent after a lost response; never reboot again. */
  }
  const deadline = Date.now() + 1_200_000;
  let afterBoot: any;
  while (Date.now() < deadline) {
    try {
      const observed = await observeApplianceBoot(io, b);
      if (observed.boot_id !== boot.boot_id) {
        afterBoot = observed;
        break;
      }
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }
  check(afterBoot && afterBoot.boot_id !== boot.boot_id, 'actual_new_native_boot_required');
  assert.deepEqual(afterBoot.root, boot.root, 'same_original_installed_root');
  assert.deepEqual(afterBoot.images, boot.images, 'same_images_after_native_reboot');
  const after = await readAppliance(io, b, 'snapshot', scope, w);
  assert.deepEqual(after.inner.protected, before.inner.protected, 'complete_native_retained_state_after_reboot');
  const journal = JSON.parse(
    await io.sshRead(
      applianceProgram(
        b.target.uuid,
        'const FILE=' +
          JSON.stringify('/var/lib/soda/qualification-reboots/' + operation + '.json') +
          ';\n' +
          String.raw`
const fs=require('node:fs');try{const s=fs.lstatSync(FILE);if(!s.isFile()||s.isSymbolicLink()||s.uid!==0||s.nlink!==1||(s.mode&511)!==384||s.size>4096)throw Error();console.log(fs.readFileSync(FILE,'utf8'));}catch{console.error('native_reboot_journal_unconfirmed');process.exitCode=1;}`
      )
    )
  );
  check(
    journal.uuid === b.target.uuid &&
      journal.boot === boot.boot_id &&
      journal.operation === operation &&
      journal.source === b.source &&
      journal.payload === b.payloadSHA &&
      journal.protected === before.inner.protected_sha256,
    'exact_native_reboot_journal'
  );
  const native = {
    format: 1,
    target: b.target.name,
    uuid: boot.uuid,
    source_revision: b.source,
    payload_sha256: b.payloadSHA,
    old_boot_id: boot.boot_id,
    new_boot_id: afterBoot.boot_id,
    deployment: afterBoot.deployment,
    root: afterBoot.root,
    protected_before_sha256: before.inner.protected_sha256,
    protected_after_sha256: sha(JSON.stringify(after.inner.protected)),
    actual_kernel_reboot: true,
    physical_power_off: false,
    acknowledgment,
    automatic_retry: false,
  };
  await io.save('persistence/after.json', after);
  await io.save('persistence/native.json', native);
  return {native};
}
