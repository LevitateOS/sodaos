import {check, COMMIT, SHA, sha} from './support';

export function factoryCommandID(operation: string, label: string) {
  check(/^[a-f0-9]{32}$/.test(operation) && /^[a-z][a-z0-9-]{0,95}$/.test(label), 'factory_operation_command');
  return sha('qualification-factory\0' + operation + '\0' + label).slice(0, 32);
}

export function validateFactoryPhase(c: any) {
  check(c?.execution_released === true && /^[a-f0-9]{32}$/.test(c.operation_id), 'factory_phase_held');
  const r = c.repository;
  check(
    Number.isSafeInteger(r?.id) &&
      r.id > 0 &&
      /^[A-Za-z0-9._-]+$/.test(r.owner) &&
      /^[A-Za-z0-9._-]+$/.test(r.name) &&
      /^p[a-f0-9]{24}$/.test(r.project) &&
      COMMIT.test(r.source_commit),
    'factory_repository_scope'
  );
  check(
    !['.', '..'].includes(r.owner) &&
      !['.', '..'].includes(r.name) &&
      typeof c.target_branch === 'string' &&
      /^refs\/heads\/[A-Za-z0-9._/-]+$/.test(c.target_branch) &&
      !c.target_branch.includes('..') &&
      !c.target_branch.endsWith('/'),
    'factory_target_branch'
  );
  check(
    Number.isSafeInteger(c.issue?.number) &&
      c.issue.number > 0 &&
      Number.isSafeInteger(c.issue.native_id) &&
      c.issue.native_id > 0 &&
      /^d[a-f0-9]{24}$/.test(c.issue.acceptance) &&
      SHA.test(c.issue.title_digest) &&
      SHA.test(c.issue.content_digest),
    'factory_accepted_issue_scope'
  );
  check(
    /^[a-f0-9]{32}$/.test(c.connection?.id) &&
      /^[1-9][0-9]*$/.test(c.connection.owner_id) &&
      Number.isSafeInteger(c.connection.generation) &&
      c.connection.generation > 0 &&
      /^[a-f0-9]{32}$/.test(c.connection.grant),
    'factory_existing_connection'
  );
  check(
    Number.isSafeInteger(c.funding?.sponsor_minutes) &&
      c.funding.sponsor_minutes > 0 &&
      Number.isSafeInteger(c.funding.rolling_minutes) &&
      c.funding.rolling_minutes > 0 &&
      c.funding.max_concurrent === 1,
    'factory_existing_funding'
  );
  check(
    ['intent_minutes', 'attempt_minutes', 'maximum_seconds'].every((k) => Number.isSafeInteger(c.limits?.[k])) &&
      c.limits.intent_minutes >= 1 &&
      c.limits.intent_minutes <= 5 &&
      c.limits.attempt_minutes >= 1 &&
      c.limits.attempt_minutes <= 10 &&
      c.limits.maximum_seconds >= 60 &&
      c.limits.maximum_seconds <= 1800,
    'factory_finite_limits'
  );
  check(
    /^d[a-f0-9]{24}$/.test(c.requirements) && /^d[a-f0-9]{24}$/.test(c.approval) && SHA.test(c.approved_check_sha256),
    'factory_current_approval'
  );
  check(
    [c.coder_preparation, c.reviewer_preparation].every((p) => /^f[a-f0-9]{24}$/.test(p)),
    'factory_exact_standing_preparations'
  );
  check(
    typeof c.check_script_path === 'string' &&
      /^[A-Za-z0-9_][A-Za-z0-9_./-]{0,255}$/.test(c.check_script_path) &&
      c.check_script_path.split('/').every((part: string) => part !== '' && part !== '.' && part !== '..'),
    'factory_explicit_check_script'
  );
  check(
    Array.isArray(c.required_checks) &&
      c.required_checks.length === 1 &&
      typeof c.required_checks[0] === 'string' &&
      /^[A-Za-z0-9._/-]{1,256}$/.test(c.required_checks[0]),
    'factory_exact_check_context'
  );
  check(
    Number.isSafeInteger(c.actors?.coder) &&
      Number.isSafeInteger(c.actors?.reviewer) &&
      Number.isSafeInteger(c.actors?.ci) &&
      [c.actors.coder, c.actors.reviewer, c.actors.ci].every((x) => x > 0) &&
      new Set([c.actors.coder, c.actors.reviewer, c.actors.ci]).size === 3,
    'factory_distinct_native_actors'
  );
  check(
    /^[A-Za-z0-9._-]{1,64}$/.test(c.ci_login) && c.ci_token_path === '/var/lib/soda-qualification/ci-token',
    'factory_fixed_ci_custody'
  );
  return c;
}

// Literal fixed reads. SQL produces selected typed fields; no run JSON,
// provider state/output/summary, credentials, worker logs or report prose.
export const FACTORY_SNAPSHOT = String.raw`
const cp=require('child_process');
const run=(file,args)=>cp.execFileSync(file,args,{encoding:'utf8',timeout:20000,maxBuffer:1<<20,stdio:['ignore','pipe','pipe']}).trim();
const pod=args=>run('/usr/bin/podman',['--remote=false',...args]);
const sql=(statement,database='soda')=>JSON.parse(pod(['exec','soda-postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-U','postgres','-d',database,'-c','BEGIN READ ONLY;'+statement+';COMMIT;']));
try{
 if(process.getuid()!==0)throw Error();
 const R=REQUEST.repository.id,I=REQUEST.issue.number,C=REQUEST.connection.id,P=REQUEST.repository.project;
 if(!Number.isSafeInteger(R)||R<=0||!Number.isSafeInteger(I)||I<=0||!/^p[a-f0-9]{24}$/.test(P)||!/^([a-f0-9]{32})$/.test(C))throw Error();
 const q="WITH a AS(SELECT id,run,stage,data FROM factory_assignments WHERE repository="+R+" AND issue="+I+") SELECT json_build_object("+
 "'at',clock_timestamp(),'acceptance',(SELECT json_build_object('id',d.id,'title_digest',d.data->>'title_digest','content_digest',d.data->>'content_digest','native_revision',d.data->>'native_revision') FROM issue_acceptance_heads h JOIN issue_acceptance_decisions d ON d.id=h.decision WHERE h.repository="+R+" AND h.issue="+I+"),"+
 "'connection',(SELECT json_build_object('id',id,'owner_id',owner_id::text,'state',state,'generation',generation,'provider',data->>'provider_id') FROM identity_connections WHERE id='"+C+"'),"+
 "'funding',json_build_object('sponsor_minutes',(SELECT data->'allowance_minutes' FROM factory_sponsorships WHERE repository="+R+" AND connection='"+C+"'),'rolling_minutes',(SELECT data->'rolling_minutes' FROM factory_connection_usage_budgets WHERE connection='"+C+"'),'repository_used_minutes',(SELECT COALESCE(sum(minutes),0) FROM factory_usage WHERE repository="+R+" AND connection='"+C+"'),'rolling_used_micros',(SELECT COALESCE(sum(GREATEST(0,FLOOR(EXTRACT(EPOCH FROM(LEAST(ended_at,transaction_timestamp())-GREATEST(started_at,transaction_timestamp()-interval '24 hours')))*1000000))),0) FROM factory_usage WHERE connection='"+C+"' AND ended_at>transaction_timestamp()-interval '24 hours' AND started_at<transaction_timestamp())),"+
 "'preparations',(SELECT COALESCE(json_agg(json_build_object('id',id,'role',role,'source',data#>>'{preparation,source_commit}','requirements',requirements,'approval',approval,'phase',data#>>'{state,phase}','ready',data#>'{state,ready}','stopped',data#>'{state,stopped}','setup_exit',data#>'{state,setup_exit}','check_exit',data#>'{state,check_exit}','setup_digest',data#>>'{preparation,setup_digest}','owner',data#>>'{factory_admission,owner_assignment}','attempt_root',data#>>'{factory_admission,attempt_root}','publication',data#>>'{factory_admission,publication_id}','child',data#>>'{factory_admission,child_assignment}','not_after',data#>>'{factory_admission,not_after_unix}')),'[]'::json) FROM project_preparations WHERE project_id='"+P+"'),"+
 "'assignments',(SELECT COALESCE(json_agg(json_build_object('id',a.id,'run',a.run,'stage',a.stage,'root',a.data->>'attempt_root','parent',a.data->>'parent_assignment','publication_assignment',a.data->>'publication_assignment','project',a.data->>'project_id','phase',a.data->>'phase','role',a.data->>'role','source',a.data->>'source_commit','preparation',a.data->>'preparation','connection',a.data->>'connection','acceptance',a.data->>'acceptance','outcome',a.data->>'outcome','reported',a.data#>'{result,reported}')),'[]'::json) FROM a),"+
 "'runs',(SELECT COALESCE(json_agg(json_build_object('id',r.id,'active',r.active,'settled',r.settled,'project',r.data->>'project_id','role',r.data->>'role','source',r.data->>'input_sha','started',r.data->>'started','deadline',r.data->>'deadline','context_pending',r.data->'context_pending','reconciled',r.data->'reconciled','outcome',r.data->>'outcome','credential_returned',r.data->'credential_returned','image',r.data->>'image','harness',r.data->>'harness','model',r.data->>'model','lease',r.data->>'identity_lease_id','generation',r.data->'identity_generation','binding',json_build_object('kind',r.data#>>'{identity_binding,kind}','id',r.data#>>'{identity_binding,id}','project',r.data#>>'{identity_binding,project}','child_id',r.data#>>'{identity_binding,child_id}','invocation_id',r.data#>>'{identity_binding,invocation_id}','generation',r.data#>'{identity_binding,generation}'),'execution',json_build_object('state',e.state,'lease_id',e.lease_id,'binding_id',e.data#>>'{binding,id}','invocation_id',e.data#>>'{binding,invocation_id}','project',e.data#>>'{binding,project}','child_id',e.data#>>'{binding,child_id}','generation',e.data#>'{binding,generation}'),'current_lease',json_build_object('connection',l.connection_id,'project',l.data->>'project_id','execution',l.data->>'execution_id','kind',l.data->>'kind','generation',l.data->'generation','binding_id',l.data#>>'{binding,id}','invocation_id',l.data#>>'{binding,invocation_id}'))),'[]'::json) FROM factory_runs r JOIN a ON a.run=r.id LEFT JOIN identity_executions e ON e.kind='factory' AND e.execution_id=r.id LEFT JOIN identity_leases l ON l.id=r.data->>'identity_lease_id'),"+
 "'allowances',(SELECT COALESCE(json_agg(json_build_object('root',l.root_assignment,'active',l.data->'active','closed',l.data->'closed','active_seconds',l.data->'active_seconds','active_minutes',l.data#>'{limits,active_minutes}')),'[]'::json) FROM factory_attempt_allowances l WHERE repository="+R+" AND issue="+I+"),"+
 "'usage',(SELECT COALESCE(json_agg(json_build_object('run',u.run,'connection',u.connection,'minutes',u.minutes,'started_at',u.started_at,'ended_at',u.ended_at)),'[]'::json) FROM factory_usage u JOIN a ON a.run=u.run),"+
 "'submissions',(SELECT COALESCE(json_agg(json_build_object('id',s.submission,'assignment',s.assignment,'tool',s.tool,'state',s.state)),'[]'::json) FROM factory_mcp_submissions s JOIN a ON a.id=s.assignment WHERE s.tool IN('factory.finish_implementation','factory.finish_review','factory.finish_incomplete')),"+
 "'reservations',(SELECT COALESCE(json_agg(json_build_object('assignment',v.assignment,'state',v.state,'connection',v.connection,'planned_minutes',v.data->'planned_minutes')),'[]'::json) FROM factory_reservations v JOIN a ON a.id=v.assignment),"+
 "'publications',(SELECT COALESCE(json_agg(json_build_object('id',p.data->>'id','assignment_id',p.assignment,'run',p.run,'stage',p.stage,'candidate',p.data->>'candidate','acceptance',p.data->>'acceptance','project',p.data->>'project_id','pr_number',p.data->'pr_number','pr_id',p.data->'pr_id','issue_id',p.data#>'{pr_create,issue_id}','actor',(p.data#>>'{pr_create,work,actor_id}')::bigint,'created_unix',p.data->'created_unix','reviews',(SELECT COALESCE(json_agg(json_build_object('run',v->>'run_id','actor',(v#>>'{work,actor_id}')::bigint,'head',v#>>'{work,head_oid}','event',v#>>'{work,event}','effect',v#>>'{outcome,effect}','completion',v#>>'{outcome,completion}')),'[]'::json) FROM jsonb_array_elements(COALESCE(p.data->'review_operations','[]'::jsonb))v))),'[]'::json) FROM factory_publications p WHERE repository="+R+" AND issue="+I+"),"+
 "'checks',(SELECT COALESCE(json_agg(json_build_object('pr',c.pr,'head',c.data->>'head_oid','base',c.data->>'base_oid','verdict',c.data->>'verdict','checks',c.data->'checks','results',c.data->'results','policy_revision',c.data->'policy_revision','native_revision',c.data->'native_revision')),'[]'::json) FROM factory_check_assessments c WHERE c.repository="+R+" AND c.pr IN(SELECT (data->>'pr_number')::bigint FROM factory_publications WHERE repository="+R+" AND issue="+I+")),"+
 "'merges',(SELECT COALESCE(json_agg(json_build_object('publication',publication,'stage',stage,'head',data->>'head_oid','merged_commit',data->>'merged_commit','pr_number',data->'pr_number','pr_id',data->'pr_id','issue_id',data->'issue_id','coder_actor',data->'pr_author_id','reviewer_actor',data->'reviewer_id','effect',data#>>'{operation,effect}','completion',data#>>'{operation,completion}','closed_unix',data->'closed_unix','closing_occurrence',data->>'original_closing_occurrence')),'[]'::json) FROM factory_merges WHERE repository="+R+" AND issue="+I+"),"+
 "'completions',(SELECT COALESCE(json_agg(json_build_object('id',event,'publication',publication,'merged_commit',data->>'merged_commit','acceptance',data->>'acceptance_id','closing_occurrence',occurrence)),'[]'::json) FROM factory_completions WHERE repository="+R+" AND data->>'issue'='"+I+"'),"+
 "'global',json_build_object('unsettled',(SELECT count(*) FROM factory_runs WHERE active OR NOT settled OR COALESCE(data->>'reconciled','false')<>'true'),'held',(SELECT count(*) FROM factory_reservations WHERE state='held'),'leases',(SELECT count(*) FROM identity_leases),'nonterminal',(SELECT count(*) FROM identity_executions WHERE state<>'terminal')))";
 const facts=sql(q);
 for(const key of ['assignments','runs','allowances','usage','reservations','submissions','publications','checks','merges','completions'])if(!Array.isArray(facts[key])||facts[key].length>64)throw Error();
 if(!Array.isArray(facts.preparations)||facts.preparations.length>256)throw Error();
 const container=pod(['inspect','--type','container','--format','{{.ID}}','soda-'+P]);
 if(!/^[a-f0-9]{64}$/.test(container))throw Error();
 const allProjects=pod(['ps','--filter','label=org.soda.project','--format','{{.ID}}']).split('\n').filter(Boolean);if(allProjects.length>16)throw Error();
 const units=[];for(const prefix of [[],...allProjects.map(id=>['exec',id])]){
  const lines=prefix.length?pod([...prefix,'systemctl','list-units','--all','--no-legend','--plain','soda-factory-*.service','soda-prepare-*.service']):run('/usr/bin/systemctl',['list-units','--all','--no-legend','--plain','soda-factory-*.service','soda-prepare-*.service']);
  const names=lines.split('\n').filter(Boolean).map(l=>l.trim().split(/\s+/)[0]);if(names.length>256)throw Error();
  for(const name of names){if(!/^soda-(factory|prepare)-[A-Za-z0-9_.-]+\.service$/.test(name))throw Error();const args=['show',name,'--property=ActiveState,MainPID,ControlPID'];const text=prefix.length?pod([...prefix,'systemctl',...args]):run('/usr/bin/systemctl',args);const fields=Object.fromEntries(text.split('\n').map(l=>{const i=l.indexOf('=');return[l.slice(0,i),l.slice(i+1)];}));if(!['active','inactive','failed','activating','deactivating','reloading','maintenance','refreshing'].includes(fields.ActiveState)||!/^(0|[1-9][0-9]*)$/.test(fields.MainPID)||!/^(0|[1-9][0-9]*)$/.test(fields.ControlPID)||!Number.isSafeInteger(Number(fields.MainPID))||!Number.isSafeInteger(Number(fields.ControlPID)))throw Error();if(!['inactive','failed'].includes(fields.ActiveState)||Number(fields.MainPID)>0||Number(fields.ControlPID)>0)units.push({unit:name,container:prefix[1]||null,state:fields.ActiveState,pid:Number(fields.MainPID),control_pid:Number(fields.ControlPID)});}
 }
 facts.global.native_units=units;facts.container=container;
 facts.integrated_source=pod(['exec','--user','1000','soda-forgejo','git','--no-optional-locks','--git-dir=/data/git/repositories/'+REQUEST.repository.owner+'/'+REQUEST.repository.name+'.git','rev-parse','--verify',REQUEST.target_branch]);if(!/^[a-f0-9]{40}$/.test(facts.integrated_source))throw Error();
 facts.native_issue=sql("SELECT json_build_object('id',id,'index',index,'repository',repo_id,'closed',is_closed,'pull',is_pull) FROM issue WHERE id="+REQUEST.issue.native_id+" AND repo_id="+R,"forgejo");
 for(const r of facts.runs.filter(r=>r.active)){
  const text=run('/usr/bin/systemctl',['show','soda-factory-'+r.id+'.service','--property=LoadState,ActiveState,SubState,MainPID,InvocationID']);r.unit=Object.fromEntries(text.split('\n').map(l=>{const i=l.indexOf('=');return[l.slice(0,i),l.slice(i+1)];}));
  const b=r.binding;r.native_bound=r.context_pending!==true&&r.unit.LoadState==='loaded'&&r.unit.ActiveState==='active'&&r.unit.SubState==='running'&&Number(r.unit.MainPID)>0&&b.kind==='factory'&&b.id===r.id&&b.project===container&&b.invocation_id===r.unit.InvocationID&&Number.isSafeInteger(r.generation)&&r.generation>0&&b.generation===r.generation&&facts.connection?.state==='ready'&&facts.connection.generation===r.generation&&r.execution.state==='live'&&r.execution.lease_id===r.lease&&r.execution.binding_id===r.id&&r.execution.invocation_id===b.invocation_id&&r.execution.project===container&&r.execution.child_id===b.child_id&&r.execution.generation===r.generation&&facts.assignments.some(a=>a.run===r.id&&a.project===P&&a.preparation===b.child_id&&a.source===r.source&&a.role===r.role)&&r.current_lease.connection===C&&r.current_lease.project===P&&r.current_lease.execution===r.id&&r.current_lease.kind==='factory'&&r.current_lease.generation===r.generation&&r.current_lease.binding_id===r.id&&r.current_lease.invocation_id===b.invocation_id;
 }
 console.log(JSON.stringify(facts));
}catch{console.error('factory_selected_metadata_unavailable');process.exitCode=1;}
`;

export function snapshotScript(scope: any) {
  validateFactoryPhase(scope);
  return 'const REQUEST=' + JSON.stringify(scope) + ';\n' + FACTORY_SNAPSHOT;
}
