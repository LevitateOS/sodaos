import {test} from 'bun:test';
import {YAML} from 'bun';
import {object} from '../installed/sodaspaces-input.ts';
import assert from 'node:assert/strict';
import {mkdtemp, writeFile, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {runnerInput} from '../installed/runners-input.ts';
import {dispatchedRunnerJob, runnerJobObservation, exerciseRunnerProvider} from '../installed/runners-provider.ts';
import {preserveRunnerBaseline, type RunnerState} from '../installed/runners-native.ts';

const base = () => ({phase:'list',target:'runner-fixture',architecture:'x86_64',revision:'a'.repeat(40),
  origin:'https://fixture.invalid',ca_file:'/private/ca',ssh_config:'/private/ssh',ssh_host:'runner-fixture',
  operator_id:'1',denied_id:'2',runner_id:'probe-one',preserved_ids:['baseline']});
const registration = {scope:'system',uuid:'33834eef-e758-48c4-a676-1745426747aa',labels:'soda-native:host',token_file:'/private/registration'};
const provider = {repository_id:'7',repository_path:'fixture/trusted',workflow:'native-support.yaml',commit:'b'.repeat(40),observation:'proof-unique',hold_seconds:0,token_file:'/private/provider'};

test('runner installed phases are separate opt-ins with no implicit mutation or cleanup', () => {
  for (const phase of ['list','register','start','stop','restart','remove','dispatch','job']) {
    const input={...base(),phase,...(phase === 'register' ? {registration} : {}),...(['dispatch','job'].includes(phase) ? {provider:{...provider,...(phase === 'job' ? {run_id:12} : {})}} : {})};
    assert.deepEqual(runnerInput(input,'--allow-runner-'+phase,'runner-fixture'),input);
    for (const permission of ['--allow-native','--allow-runner-all','--allow-runner-'+(phase === 'list' ? 'remove' : 'list')]) assert.throws(()=>runnerInput(input,permission,'runner-fixture'));
    assert.throws(()=>runnerInput(input,'--allow-runner-'+phase,undefined));
    assert.throws(()=>runnerInput(input,'--allow-runner-'+phase,'retained-other-host'));
  }
  for (const phase of ['reboot','cleanup','provider-remove','all']) assert.throws(()=>runnerInput({...base(),phase},'--allow-runner-'+phase,'runner-fixture'));
});

test('actual native role expectations are explicit and read-only, never a weaker mutation fixture', () => {
  const native_admins = {operator:true,denied:false};
  const input = {...base(),native_admins};
  assert.deepEqual(runnerInput(input,'--allow-runner-list','runner-fixture'),input);
  assert.equal(runnerInput(base(),'--allow-runner-list','runner-fixture').native_admins,undefined);
  for (const roles of [null,{},[true,false],{operator:true},{operator:1,denied:false},{operator:true,denied:'false'},{...native_admins,token:'not-allowed'}]) {
    assert.throws(()=>runnerInput({...base(),native_admins:roles},'--allow-runner-list','runner-fixture'));
  }
  for (const phase of ['register','start','stop','restart','remove','dispatch','job','overlap','departure']) {
    const other = {...input,phase,...(phase === 'register' ? {registration} : {}),...(['dispatch','job'].includes(phase) ? {provider:{...provider,...(phase === 'job' ? {run_id:12} : {})}} : {}),...(phase === 'overlap' ? {cockpit:{origin:'https://cockpit.invalid',password_file:'/private/password'}} : {})};
    assert.throws(()=>runnerInput(other,'--allow-runner-'+phase,'runner-fixture'));
  }
  assert.throws(()=>runnerInput({...input,operator_id:input.denied_id},'--allow-runner-list','runner-fixture'));
});

test('contention cases have exact distinct grants and only overlap accepts Cockpit credentials', () => {
  const cockpit={origin:'https://127.0.0.1:39090',password_file:'/private/operator-password'};
  for(const phase of ['overlap','departure']) {
    const value={...base(),phase,...(phase === 'overlap' ? {cockpit} : {})};
    assert.deepEqual(runnerInput(value,'--allow-runner-'+phase,'runner-fixture'),value);
    for(const grant of ['restart','stop','list',phase === 'overlap' ? 'departure' : 'overlap']) assert.throws(()=>runnerInput(value,'--allow-runner-'+grant,'runner-fixture'));
    assert.throws(()=>runnerInput(value,'--allow-runner-'+phase,'other'));
    assert.throws(()=>runnerInput({...value,registration},'--allow-runner-'+phase,'runner-fixture'));
  }
  assert.throws(()=>runnerInput({...base(),phase:'departure',cockpit},'--allow-runner-departure','runner-fixture'));
  for(const extra of [{origin:base().origin},{origin:'http://127.0.0.1:39090'},{origin:'https://user:password@host'},{origin:'https://host/path'},{password_file:'relative'},{password:'secret'}]) {
    assert.throws(()=>runnerInput({...base(),phase:'overlap',cockpit:{...cockpit,...extra}},'--allow-runner-overlap','runner-fixture'));
  }
});

test('runner scope refuses ambiguous identities, paths, retained targets and credential values', () => {
  for (const extra of [{operator_id:'2'},{denied_id:'01'},{runner_id:'../baseline'},{preserved_ids:['probe-one']},{preserved_ids:['baseline','baseline']},
    {origin:'http://fixture.invalid'},{origin:'https://user:secret@fixture.invalid'},{origin:'https://fixture.invalid/extra'},{ssh_host:'-Fother'},
    {ssh_config:'relative'},{target:'host;command'},{revision:'main'},{registration},{provider},{token:'synthetic-secret'}]) {
    assert.throws(()=>runnerInput({...base(),...extra},'--allow-runner-list','runner-fixture'));
  }
  for (const extra of [{scope:'repository'},{labels:'soda:docker://image'},{token_file:'relative'},{token:'synthetic-secret'},{uuid:'not-a-uuid'}]) {
    assert.throws(()=>runnerInput({...base(),phase:'register',registration:{...registration,...extra}},'--allow-runner-register','runner-fixture'));
  }
  for (const extra of [{repository_path:'../unrelated'},{workflow:'../../other.yml'},{commit:'main'},{observation:'input\nsecret'},{run_id:0},{run_id:12},{hold_seconds:601},{hold_seconds:-1},{token:'secret'}]) {
    assert.throws(()=>runnerInput({...base(),phase:'dispatch',provider:{...provider,...extra}},'--allow-runner-dispatch','runner-fixture'));
  }
});

test('provider results use the returned exact run and never expose payloads or infer success', () => {
  assert.deepEqual(dispatchedRunnerJob({id:12,run_number:3,jobs:['native'],private:'not retained'}),{run_id:12,run_number:3});
  for (const bad of [{id:0,run_number:3,jobs:['native']},{id:12,run_number:3,jobs:['native','other']},{}]) assert.throws(()=>dispatchedRunnerJob(bad));
  const input=runnerInput({...base(),phase:'job',provider:{...provider,run_id:12}},'--allow-runner-job','runner-fixture');
  const run={id:12,repository:{id:7,private:'never retained'},workflow_id:provider.workflow,commit_sha:provider.commit,event:'',trigger_event:'workflow_dispatch',event_payload:JSON.stringify({inputs:{observation:provider.observation,hold_seconds:'0'},private:'synthetic-secret'}),status:'running'};
  for(const status of ['waiting','running','success','failure','cancelled']) {
    const observed=runnerJobObservation(input,{...run,status});
    assert.equal(observed.status,status);
    assert(!JSON.stringify(observed).includes('synthetic-secret'));
  }
  for (const extra of [{id:13},{repository:{id:8}},{workflow_id:'other.yml'},{commit_sha:'c'.repeat(40)},{event:'workflow_dispatch',trigger_event:'push'},{trigger_event:undefined},{event_payload:'{}'},{status:'online'}]) assert.throws(()=>runnerJobObservation(input,{...run,...extra}));
});

test('provider transport uses private stdin, fixed preflight and one non-replayed dispatch', async () => {
  const directory=await mkdtemp(path.join(tmpdir(),'soda-runner-provider-'));
  const previous=process.env.SODA_NATIVE_VALIDATE;
  process.env.SODA_NATIVE_VALIDATE='runner-fixture';
  try {
    const tokenFile=path.join(directory,'token'), caFile=path.join(directory,'ca');
    const secret='synthetic-private-provider-token';
    await writeFile(tokenFile,secret,{mode:0o600}); await writeFile(caFile,'synthetic-ca',{mode:0o600});
    const input=runnerInput({...base(),ca_file:caFile,phase:'dispatch',provider:{...provider,token_file:tokenFile}},'--allow-runner-dispatch','runner-fixture');
    for(const fail of ['none','repository','dispatch']) {
      const urls: string[]=[];
      const execute=(args: string[],stdin: Buffer)=>{
        assert(!JSON.stringify(args).includes(secret));
        assert(stdin.toString().includes('Authorization: token '+secret));
        assert(!args.includes('-L') && args.includes('--cacert') && args.includes('@-'));
        const url=args.at(-1); assert(url); urls.push(url);
        let value: unknown = {version:'15.0.7'};
        let status=200;
        if(url.endsWith('/repos/fixture/trusted')) value={id:fail === 'repository' ? 8 : 7};
        if(url.endsWith('/dispatches')) {
          assert.equal(url,input.origin+'/api/v1/repos/fixture/trusted/actions/workflows/native-support.yaml/dispatches');
          assert(args.includes('POST'));
          value={id:12,run_number:3,jobs:['native']}; status=fail === 'dispatch' ? 502 : 201;
          const body=args[args.indexOf('--data-binary')+1]; assert(body);
          assert.deepEqual(JSON.parse(body),{ref:provider.commit,inputs:{observation:provider.observation,hold_seconds:'0'},return_run_info:true});
        }
        return {exitCode:0,stdout:Buffer.from('HTTP/1.1 '+status+' Fixture\r\n\r\n'+JSON.stringify(value))};
      };
      if(fail === 'none') assert.deepEqual(await exerciseRunnerProvider(input,execute),{run_id:12,run_number:3});
      else await assert.rejects(exerciseRunnerProvider(input,execute));
      assert.equal(urls.filter(url=>url.endsWith('/dispatches')).length,fail === 'repository' ? 0 : 1);
    }
  } finally {
    if(previous === undefined) delete process.env.SODA_NATIVE_VALIDATE; else process.env.SODA_NATIVE_VALIDATE=previous;
    await rm(directory,{recursive:true,force:true});
  }
});

test('trusted runner workflow stays manual-only and its two native steps parse without executing', async () => {
  const workflow=object(YAML.parse(await Bun.file(new URL('../fixtures/runner/native-support.yaml',import.meta.url)).text()));
  assert.deepEqual(Object.keys(object(workflow.on)),['workflow_dispatch']);
  const jobs=object(workflow.jobs); assert.deepEqual(Object.keys(jobs),['native']);
  const job=object(jobs.native); assert(job['runs-on'] === 'soda-native');
  assert(Array.isArray(job.steps) && job.steps.length === 2);
  for (const value of job.steps) {
    const step=object(value); assert(typeof step.run === 'string' && step.shell === 'bash');
    const shell=Bun.spawnSync(['bash','-n'],{stdin:Buffer.from(step.run),stdout:'pipe',stderr:'pipe'});
    assert.equal(shell.exitCode,0);
    const program=step.run.split("python3 -I - <<'PY'\n")[1]?.split('\nPY')[0]; assert(program);
    const python=Bun.spawnSync(['python3','-I','-c','import sys; compile(sys.stdin.read(), "runner-workflow", "exec")'],{stdin:Buffer.from(program),stdout:'pipe',stderr:'pipe'});
    assert.equal(python.exitCode,0);
  }
});

test('preservation compares actual baseline hashes and rejects missing or changed retained runners', () => {
  const input=runnerInput(base(),'--allow-runner-list','runner-fixture');
  const snapshot: RunnerState={target:input.target,architecture:input.architecture,proof:null,boot_id:'11111111-1111-4111-8111-111111111111',processes:{baseline:[]},prior_survivors:[],packages:['runner fixture','systemd fixture'],
    states:{baseline:{present:true,uid:1001,tree:{sha256:'a'.repeat(64)}}},
    inventory:{runners:[{id:'baseline',provider:'forgejo',registration_url:input.origin,account:'soda-runner-baseline',architecture:'x86-64',version:'fixture',capacity:1,service:{load:'loaded',active:'inactive',sub:'dead',enabled:'disabled'}}],forgejo_url:input.origin,runner_count:1,total_capacity:1,active_listeners:0}};
  preserveRunnerBaseline(input,snapshot,structuredClone(snapshot));
  const changed=structuredClone(snapshot); changed.states.baseline={present:true,uid:1002};
  assert.throws(()=>preserveRunnerBaseline(input,snapshot,changed));
  const restarted=structuredClone(snapshot); restarted.processes.baseline=[{pid:1234,uid:1001,start:'42'}];
  assert.throws(()=>preserveRunnerBaseline(input,snapshot,restarted));
  const missing=structuredClone(snapshot);missing.states.baseline={present:false};
  assert.throws(()=>preserveRunnerBaseline(input,missing,missing));
});
