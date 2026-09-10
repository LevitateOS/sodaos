// Synthetic observer receipts only: no SSH, provider, browser or native actions.
import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {object} from '../installed/sodaspaces-input.ts';
import {runnerInput} from '../installed/runners-input.ts';
import {verifyRunnerOperation, type RunnerState} from '../installed/runners-native.ts';

function input(phase: 'register' | 'start' | 'stop' | 'restart' | 'remove') {
  return runnerInput({phase,target:'fixture',architecture:'x86_64',revision:'a'.repeat(40),
    origin:'https://fixture.invalid',ca_file:'/private/ca',ssh_config:'/private/ssh',ssh_host:'fixture',
    operator_id:'1',denied_id:'2',runner_id:'one',preserved_ids:[],
    ...(phase === 'register' ? {registration:{scope:'system',uuid:'33834eef-e758-48c4-a676-1745426747aa',labels:'unique:host',token_file:'/private/token'}} : {})},
  '--allow-runner-'+phase,'fixture');
}
function snapshot(present=true, stopped=false): RunnerState {
  return {target:'fixture',architecture:'x86_64',packages:['forgejo-runner fixture','systemd fixture'],proof:null,
    states:{one:present ? {present:true,uid:123,gid:124,home:'/var/lib/soda/runners/one/state',shell:'/usr/sbin/nologin',
      registration:{descriptor:{sha256:'d'.repeat(64)},token:{sha256:'a'.repeat(64)},configuration:{sha256:'c'.repeat(64)}},tree:{sha256:'b'.repeat(64)}} : {present:false}},
    inventory:{forgejo_url:'https://fixture.invalid',runner_count:present ? 1 : 0,total_capacity:present ? 1 : 0,active_listeners:present && !stopped ? 1 : 0,
      runners:present ? [{id:'one',provider:'forgejo',registration_url:'https://fixture.invalid',account:'soda-runner-one',architecture:'x86-64',version:'fixture',capacity:1,
        service:{load:'loaded',active:stopped ? 'inactive' : 'active',sub:stopped ? 'dead' : 'running',enabled:stopped ? 'disabled' : 'enabled'}}] : []}};
}

test('all installed operation postconditions require the exact native lifecycle result',()=>{
  for(const phase of ['register','start','stop','restart','remove'] as const) {
    const before=snapshot(phase !== 'register',phase === 'start');
    const after=snapshot(phase !== 'remove',phase === 'stop');
    verifyRunnerOperation(input(phase),before,after);
    const wrongTarget=structuredClone(after); wrongTarget.target='other';
    assert.throws(()=>verifyRunnerOperation(input(phase),before,wrongTarget));
    const changedPackages=structuredClone(after); changedPackages.packages=['changed','systemd fixture'];
    assert.throws(()=>verifyRunnerOperation(input(phase),before,changedPackages));
    if(phase === 'remove') {
      const partial=structuredClone(after); partial.states.one={present:true};
      assert.throws(()=>verifyRunnerOperation(input(phase),before,partial));
      assert.throws(()=>verifyRunnerOperation(input(phase),before,before));
    } else {
      const unknown=structuredClone(after), row=unknown.inventory.runners[0]; assert(row);
      row.service.active='activating';
      assert.throws(()=>verifyRunnerOperation(input(phase),before,unknown));
      const boot=structuredClone(after), bootRow=boot.inventory.runners[0]; assert(bootRow);
      bootRow.service.enabled=phase === 'stop' ? 'enabled' : 'disabled';
      assert.throws(()=>verifyRunnerOperation(input(phase),before,boot));
    }
  }
});

test('lifecycle preserves account, credentials and client but permits native job work changes',()=>{
  for(const phase of ['start','stop','restart'] as const) {
    const before=snapshot(), after=snapshot(true,phase === 'stop');
    after.states.one={...object(before.states.one),tree:{sha256:'e'.repeat(64)}};
    verifyRunnerOperation(input(phase),before,after);
    for(const key of ['uid','gid','home','shell','registration']) {
      const changed=structuredClone(after);
      changed.states.one={...object(after.states.one),[key]:'changed'};
      assert.throws(()=>verifyRunnerOperation(input(phase),before,changed));
    }
    const upgraded=structuredClone(after), row=upgraded.inventory.runners[0]; assert(row);
    row.version='different client';
    assert.throws(()=>verifyRunnerOperation(input(phase),before,upgraded));
  }
});
