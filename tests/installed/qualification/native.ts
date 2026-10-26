// Fixed read-only observations for check. No effect endpoints, worker dispatch,
// raw run DTOs, provider state, issue prose or credential file contents.
import {object} from '../sodaspaces-input';
import {FIXTURE_IDENTITIES_SQL, issueIdentitiesSQL} from './facts';
import type {QualificationScenario} from './scenario';
import {pinnedSSH} from './validate';
import {RESOURCE_RUNTIME_READER} from './resource-runtime';

function readinessSQL(s: QualificationScenario): string {
  const factory = s.factory.map(
    ({subject: b}) => `json_build_object(
    'repository_id','${b.repository_id}','issue_number',${b.issue_number},
    'connection_provider',(SELECT data->>'provider_id' FROM identity_connections WHERE id='${b.connection_id}'),
    'roles',(SELECT data->'roles' FROM factory_policies WHERE repository=${b.repository_id}),
    'acceptance',(SELECT decision FROM issue_acceptance_heads WHERE repository=${b.repository_id} AND issue=${b.issue_number}),
    'requirements',(SELECT decision FROM project_requirement_heads WHERE project_id='${b.project_id}'),
    'approval',(SELECT decision FROM project_approval_heads WHERE project_id='${b.project_id}'),
    'publish_actor',(SELECT data#>>'{publish_actor,actor_id}' FROM factory_policies WHERE repository=${b.repository_id}),
    'review_actor',(SELECT data#>>'{review_actor,actor_id}' FROM factory_policies WHERE repository=${b.repository_id}),
    'required_checks',(SELECT data->'required_checks' FROM factory_policies WHERE repository=${b.repository_id}),
    'target_branch',(SELECT data->>'target_branch' FROM factory_policies WHERE repository=${b.repository_id}),
    'accounting',json_build_object(
      'sponsor_minutes',(SELECT data->'allowance_minutes' FROM factory_sponsorships WHERE repository=${b.repository_id} AND connection='${b.connection_id}'),
      'sponsor_active',(SELECT data->'active' FROM factory_sponsorships WHERE repository=${b.repository_id} AND connection='${b.connection_id}'),
      'sponsor_max_concurrent',(SELECT data->'max_concurrent' FROM factory_sponsorships WHERE repository=${b.repository_id} AND connection='${b.connection_id}'),
      'rolling_limit_minutes',(SELECT data->'rolling_minutes' FROM factory_connection_usage_budgets WHERE connection='${b.connection_id}'),
      'repository_charged_minutes',(SELECT coalesce(sum(minutes),0) FROM factory_usage WHERE repository=${b.repository_id} AND connection='${b.connection_id}'),
      'held_reservations',(SELECT count(*) FROM factory_reservations WHERE connection='${b.connection_id}' AND state='held'),
      'rolling_settled_micros',(SELECT LEAST(COALESCE(SUM(GREATEST(0,FLOOR(EXTRACT(EPOCH FROM (LEAST(ended_at,transaction_timestamp())-GREATEST(started_at,transaction_timestamp()-interval '24 hours')))*1000000))),0),9223372036854775807)::bigint::text FROM factory_usage WHERE connection='${b.connection_id}' AND ended_at>transaction_timestamp()-interval '24 hours' AND started_at<transaction_timestamp())),
    'preparations',(SELECT coalesce(json_agg(json_build_object('id',id,'role',role,'requirements',requirements,'approval',approval,'source',data#>>'{preparation,source_commit}','phase',data#>>'{state,phase}','setup_digest',data#>>'{preparation,setup_digest}','ready',data#>'{state,ready}','stopped',data#>'{state,stopped}','setup_exit',data#>'{state,setup_exit}','check_exit',data#>'{state,check_exit}')),'[]') FROM project_preparations WHERE project_id='${b.project_id}'),
    'runs',(SELECT coalesce(json_agg(json_build_object('id',r.id,'active',r.active,'settled',r.settled,'deadline',r.data->>'deadline','reconciled',r.data->'reconciled')),'[]') FROM factory_runs r WHERE EXISTS(SELECT 1 FROM factory_assignments a WHERE a.run=r.id AND a.repository=${b.repository_id} AND a.issue=${b.issue_number})),
    'allowances',(SELECT coalesce(json_agg(json_build_object('root',root_assignment,'active',data->'active','closed',data->'closed','active_seconds',data->'active_seconds','active_minutes',data#>'{limits,active_minutes}')),'[]') FROM factory_attempt_allowances WHERE repository=${b.repository_id} AND issue=${b.issue_number}))`
  );
  return `BEGIN READ ONLY; SELECT json_build_object(
    'projects',(SELECT coalesce(json_agg(row_to_json(p)),'[]') FROM (SELECT id,repository_id::text,repository FROM projects ORDER BY id)p),
    'custody',json_build_object(
      'leases',(SELECT count(*) FROM identity_leases),
      'nonterminal',(SELECT count(*) FROM identity_executions WHERE state<>'terminal'),
      'held',(SELECT count(*) FROM factory_reservations WHERE state='held'),
      'unsettled',(SELECT count(*) FROM factory_runs WHERE active OR NOT settled OR coalesce(data->>'reconciled','false')<>'true'),
      'preparations',(SELECT count(*) FROM project_preparations WHERE coalesce(data#>>'{state,phase}','') IN ('preparing','running')),
      'restores',(SELECT count(*) FROM application_restores WHERE state='pending'),
      'setups',(SELECT count(*) FROM application_setups WHERE state IN ('pending','running')),
      'effective_dispatch',(SELECT count(*) FROM factory_dispatch d JOIN factory_policies p USING(repository) WHERE d.open AND (p.data->>'enabled')::boolean AND NOT (p.data->>'paused')::boolean)),
    'factory',json_build_array(${factory.join(',')}),
    'history',json_build_object('runs',(SELECT count(*) FROM factory_runs),'charged_minutes',(SELECT coalesce(sum(minutes),0) FROM factory_usage))
  ); COMMIT;`;
}

const INNER = String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
const run=(p,a)=>cp.execFileSync(p,a,{encoding:'utf8',timeout:30000,maxBuffer:2<<20,stdio:['ignore','pipe','pipe']}).trim();
const pod=a=>run('/usr/bin/podman',['--remote=false',...a]);
const sql=(db,q)=>JSON.parse(pod(['exec','soda-postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-U','postgres','-d',db,'-c',q]));
let stage='identity';try{
if(process.getuid()!==0)throw Error('root observation required');
if(REQUEST.target.distribution==='docker'&&fs.readFileSync('/run/systemd/container','utf8').trim()!=='docker')throw Error('inner target');
const bytes=fs.readFileSync('/usr/share/soda/release.json'),payload=JSON.parse(bytes);
const identity=sql('soda',IDENTITY_SQL),selected=sql('soda',READINESS_SQL),issues=sql('forgejo',ISSUE_SQL);
const ci_ids=REQUEST.factory.map(f=>f.subject.actors.ci);if(!ci_ids.every(id=>/^[1-9][0-9]*$/.test(id)))throw Error('actor identity');
stage='ci_actors';
const ci_actors=sql('forgejo',"BEGIN READ ONLY; SELECT coalesce(json_agg(json_build_object('id',id::text,'admin',is_admin)),'[]') FROM public.\"user\" WHERE id IN ("+(ci_ids.join(',')||'0')+"); COMMIT;");
stage='unit_inventory';
const runtime=readResourceRuntime(identity,run,pod);
const projects=[];
for(const observed of runtime.project_runtime){
 stage='project_inventory';
 const p=selected.projects.find(p=>p.id===observed.id);if(!p)throw Error('registered project changed');
 let source=null,check_script_sha256=null;
 if(REQUEST.factory.some(f=>f.subject.repository_id===p.repository_id)){
  if(!/^[A-Za-z0-9_-][A-Za-z0-9_.-]*\/[A-Za-z0-9_-][A-Za-z0-9_.-]*$/.test(p.repository)||p.repository.includes('..'))throw Error('repository identity');
  const state=selected.factory.find(f=>f.repository_id===p.repository_id),branch=state?.target_branch;
  stage='target_branch';
  if(typeof branch!=='string'||!/^refs\/heads\/[A-Za-z0-9_][A-Za-z0-9_./-]*$/.test(branch)||branch.includes('..'))throw Error('target branch');
  const git=['exec','--user','1000','soda-forgejo','git','--no-optional-locks','--git-dir=/data/git/repositories/'+p.repository.toLowerCase()+'.git'];
  source=pod([...git,'rev-parse',branch]);
  const scope=REQUEST.factory.find(f=>f.subject.repository_id===p.repository_id).subject;
  stage='check_script';
  if(scope.check_script_path!==undefined){const check=scope.check_script_path;if(!/^[A-Za-z0-9_][A-Za-z0-9_./-]*$/.test(check)||check.includes('..'))throw Error('check identity');const checkBytes=cp.execFileSync('/usr/bin/podman',['--remote=false',...git,'show',branch+':'+check],{timeout:30000,maxBuffer:1<<20,stdio:['ignore','pipe','pipe']});check_script_sha256=crypto.createHash('sha256').update(checkBytes).digest('hex');}
 }
 projects.push({...observed,source,check_script_sha256});
}
stage='loaded_images';
const loaded_images={};for(const [role,image]of Object.entries(payload.Images))loaded_images[role]='sha256:'+pod(['image','inspect','--format','{{.ID}}',image.Config]).replace(/^sha256:/,'');
console.log(JSON.stringify({...identity,...runtime,issues,observed_at:new Date().toISOString(),uuid:REQUEST.target.uuid,distribution:REQUEST.target.distribution,source_revision:payload.Revision,payload_sha256:crypto.createHash('sha256').update(bytes).digest('hex'),loaded_images,expected_images:payload.Images,custody:selected.custody,factory_readiness:selected.factory,ci_actors,history:selected.history,project_runtime:projects}));
}catch{console.error(JSON.stringify({stage}));process.exitCode=1;}
`;

/** One fixed selected-metadata program; never a caller-supplied shell command. */
export async function observeNative(s: QualificationScenario): Promise<unknown> {
  const argv = await pinnedSSH(s);
  const inner =
    'const REQUEST=' +
    JSON.stringify(s) +
    ';\nconst IDENTITY_SQL=' +
    JSON.stringify(FIXTURE_IDENTITIES_SQL) +
    ';\nconst READINESS_SQL=' +
    JSON.stringify(readinessSQL(s)) +
    ';\nconst ISSUE_SQL=' +
    JSON.stringify(issueIdentitiesSQL(s.factory)) +
    ';\n' +
    RESOURCE_RUNTIME_READER +
    INNER;
  const program =
    'const REQUEST=' +
    JSON.stringify({uuid: s.target.uuid, distribution: s.target.distribution}) +
    ';\nconst INNER=' +
    JSON.stringify(inner) +
    ';\n' +
    String.raw`
const fs=require('node:fs'),cp=require('node:child_process');
try{
 if(process.getuid()!==0||fs.readFileSync('/sys/class/dmi/id/product_uuid','utf8').trim().toLowerCase()!==REQUEST.uuid)throw Error('target identity');
 if(REQUEST.distribution==='appliance'){process.stdout.write(cp.execFileSync('/usr/bin/node',['-'],{input:INNER,encoding:'utf8',timeout:180000,maxBuffer:2<<20,stdio:['pipe','pipe','pipe']}))}else{
  const docker=a=>cp.execFileSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock',...a],{encoding:'utf8',timeout:30000,maxBuffer:2<<20,stdio:['ignore','pipe','pipe']}).trim();
  const ids=docker(['ps','--filter','label=com.docker.compose.project=sodaos','--filter','label=com.docker.compose.service=soda','--format','{{.ID}}']).split('\n').filter(Boolean);if(ids.length!==1)throw Error('one vendor deployment');
  const d=JSON.parse(docker(['inspect','--type','container',ids[0]]))[0];
  const selected=JSON.parse(cp.execFileSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock','exec','-i',d.Id,'/usr/bin/node','-'],{input:INNER,encoding:'utf8',timeout:180000,maxBuffer:2<<20,stdio:['pipe','pipe','pipe']}));
  console.log(JSON.stringify({...selected,uuid:REQUEST.uuid,distribution:'docker',docker:{container_id:d.Id,container_name:d.Name.replace(/^\//,''),reference:d.Config.Image,engine_image:d.Image,descriptor_digest:d.ImageManifestDescriptor?.digest??null}}));
 }
}catch(error){let stage='transport';try{const inner=JSON.parse(String(error.stderr));if(['identity','ci_actors','unit_inventory','project_inventory','target_branch','check_script','loaded_images'].includes(inner.stage))stage=inner.stage;}catch{}console.error(JSON.stringify({stage}));process.exitCode=1;}
`;
  const child = Bun.spawnSync(argv, {
    stdin: new TextEncoder().encode(program),
    stdout: 'pipe',
    stderr: 'pipe',
    timeout: 240000,
    maxBuffer: 2 << 20,
  });
  if (child.exitCode !== 0) {
    let stage = 'transport';
    try {
      const failure = object(JSON.parse(new TextDecoder().decode(child.stderr)));
      if (
        typeof failure.stage === 'string' &&
        [
          'identity',
          'ci_actors',
          'unit_inventory',
          'project_inventory',
          'target_branch',
          'check_script',
          'loaded_images',
        ].includes(failure.stage)
      )
        stage = failure.stage;
    } catch {}
    throw new NativeReadError(stage);
  }
  return JSON.parse(new TextDecoder().decode(child.stdout));
}

export class NativeReadError extends Error {
  constructor(readonly stage: string) {
    super('Read-only native observation unconfirmed');
  }
}
