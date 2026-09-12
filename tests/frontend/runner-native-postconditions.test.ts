// Synthetic observer receipts only: no SSH, provider, browser or native actions.
import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {object} from '../installed/sodaspaces-input.ts';
import {runnerInput} from '../installed/runners-input.ts';
import {verifyRunnerContention, verifyRunnerOperation, type RunnerState} from '../installed/runners-native.ts';

function input(phase: 'register' | 'start' | 'stop' | 'restart' | 'remove') {
  return runnerInput({phase,target:'fixture',architecture:'x86_64',revision:'a'.repeat(40),
    origin:'https://fixture.invalid',ca_file:'/private/ca',ssh_config:'/private/ssh',ssh_host:'fixture',
    operator_id:'1',denied_id:'2',runner_id:'one',preserved_ids:[],
    ...(phase === 'register' ? {registration:{scope:'system',uuid:'33834eef-e758-48c4-a676-1745426747aa',labels:'unique:host',token_file:'/private/token'}} : {})},
  '--allow-runner-'+phase,'fixture');
}
function snapshot(present=true, stopped=false, start='42'): RunnerState {
  return {target:'fixture',architecture:'x86_64',packages:['forgejo-runner fixture','systemd fixture'],proof:null,
    boot_id:'11111111-1111-4111-8111-111111111111',prior_survivors:[],processes:{one:present && !stopped ? [{pid:1234,uid:123,start}] : []},
    states:{one:present ? {present:true,uid:123,gid:124,home:'/var/lib/soda/runners/one/state',shell:'/usr/sbin/nologin',
      registration:{descriptor:{sha256:'d'.repeat(64)},token:{sha256:'a'.repeat(64)},configuration:{sha256:'c'.repeat(64)}},tree:{sha256:'b'.repeat(64)}} : {present:false}},
    inventory:{forgejo_url:'https://fixture.invalid',runner_count:present ? 1 : 0,total_capacity:present ? 1 : 0,active_listeners:present && !stopped ? 1 : 0,
      runners:present ? [{id:'one',provider:'forgejo',registration_url:'https://fixture.invalid',account:'soda-runner-one',architecture:'x86-64',version:'fixture',capacity:1,
        service:{load:'loaded',active:stopped ? 'inactive' : 'active',sub:stopped ? 'dead' : 'running',enabled:stopped ? 'disabled' : 'enabled'}}] : []}};
}

test('contention accepts either serialized winner, but departure never invents acknowledgement',()=>{
  const before=snapshot(), restarted=snapshot(true,false,'43'), stopped=snapshot(true,true);
  const overlap={...input('restart'),phase:'contention' as const};
  const departure={...input('restart'),phase:'departure' as const};
  assert.equal(verifyRunnerContention(overlap,before,restarted),'restart-last');
  assert.equal(verifyRunnerContention(overlap,before,stopped),'stop-last');
  assert.throws(()=>verifyRunnerContention(overlap,before,before));
  assert.equal(verifyRunnerContention(departure,before,before),'cancelled-before-observed-effect');
  assert.equal(verifyRunnerContention(departure,before,restarted),'restart-observed-response-unconfirmed');
  assert.throws(()=>verifyRunnerContention(departure,before,stopped));
  const survivor=structuredClone(restarted); survivor.prior_survivors=[{pid:1234,uid:123,start:'42'}];
  for(const request of [overlap,departure]) {
    assert.throws(()=>verifyRunnerContention(request,before,survivor));
    const changed=structuredClone(restarted); changed.states.one={...object(restarted.states.one),uid:999};
    assert.throws(()=>verifyRunnerContention(request,before,changed));
    const boot=structuredClone(before); boot.boot_id='22222222-2222-4222-8222-222222222222';
    assert.throws(()=>verifyRunnerContention(request,before,boot));
  }
});

test('all installed operation postconditions require the exact native lifecycle result',()=>{
  for(const phase of ['register','start','stop','restart','remove'] as const) {
    const before=snapshot(phase !== 'register',phase === 'start');
    const after=snapshot(phase !== 'remove',phase === 'stop','43');
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
    const before=snapshot(), after=snapshot(true,phase === 'stop','43');
    after.states.one={...object(before.states.one),tree:{sha256:'e'.repeat(64)}};
    verifyRunnerOperation(input(phase),before,after);
    for(const key of ['uid','gid','home','shell','registration']) {
      const changed=structuredClone(after);
      changed.states.one={...object(after.states.one),[key]:'changed'};
      assert.throws(()=>verifyRunnerOperation(input(phase),before,changed));
    }
    if(phase === 'stop' || phase === 'restart') {
      const escaped=structuredClone(after); escaped.prior_survivors=[{pid:1234,uid:123,start:'42'}];
      assert.throws(()=>verifyRunnerOperation(input(phase),before,escaped));
      const oldScope=structuredClone(after); oldScope.processes=structuredClone(before.processes);
      assert.throws(()=>verifyRunnerOperation(input(phase),before,oldScope));
    }
    const rebooted=structuredClone(after); rebooted.boot_id='22222222-2222-4222-8222-222222222222';
    assert.throws(()=>verifyRunnerOperation(input(phase),before,rebooted));
    const upgraded=structuredClone(after), row=upgraded.inventory.runners[0]; assert(row);
    row.version='different client';
    assert.throws(()=>verifyRunnerOperation(input(phase),before,upgraded));
  }
});
