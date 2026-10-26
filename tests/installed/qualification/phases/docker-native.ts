import {RESOURCE_RUNTIME_READER} from '../resource-runtime';
import {assertResourceInventory, diagnoseClosure, type ResourceScope} from '../closure';
import {check, sha} from './support';
import {nativeRequest, validateWitness, type DockerBinding, type DockerIO, type Witness} from './docker-binding';

// Fixed inner program. Reads selected metadata only; never exports connection
// credentials, Factory output/Summary, command outcomes, or arbitrary SQL.
export const INNER_READER =
  RESOURCE_RUNTIME_READER +
  String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
const need=(x,r)=>{if(!x)throw Error(r)};
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const run=(p,a,input)=>cp.execFileSync(p,a,{input,encoding:'utf8',timeout:30000,maxBuffer:2<<20,stdio:[input===undefined?'ignore':'pipe','pipe','pipe']}).trim();
const podman=a=>run('/usr/bin/podman',['--remote=false',...a]);
const sql=q=>JSON.parse(podman(['exec','soda-postgres','psql','-X','-U','postgres','-d','soda','-Atc',q]));
const mounted=p=>{const x=cp.spawnSync('/usr/bin/findmnt',['-rn','--mountpoint',p,'-o','OPTIONS'],{encoding:'utf8',timeout:10000});need([0,1].includes(x.status),'mount_observation');return x.status===0};
const unit=n=>Object.fromEntries(run('/usr/bin/systemctl',['show',n,'-p','ActiveState','-p','MainPID','-p','FragmentPath','-p','DropInPaths']).split('\n').map(l=>{const i=l.indexOf('=');return[l.slice(0,i),l.slice(i+1)]}));
try{
 need(process.getuid()===0&&fs.readFileSync('/run/systemd/container','utf8').trim()==='docker','inner_root_systemd');
 const bytes=fs.readFileSync('/usr/share/soda/release.json'),p=JSON.parse(bytes);need(hash(bytes)===REQUEST.payloadSHA&&p.Revision===REQUEST.source,'installed_payload');
 const configs={};for(const [role,image]of Object.entries(p.Images)){const actual=podman(['image','inspect','--format','{{.ID}}',image.Config]).replace(/^sha256:/,'');need(actual===image.Config.slice(7),'loaded_image_'+role);configs[role]='sha256:'+actual;}
 const profile=run('/usr/bin/curl',['--fail','--silent','--show-error','--max-time','20','--unix-socket','/run/soda/host/host.sock','--request','POST','--data','{}','--output','/dev/null','--write-out','%{http_code}','http://localhost/profile']);need(profile==='200','native_profile');
 const services={};for(const name of ['soda-host.service','soda-dashboard.service','soda-identity.service','soda-proxy.service','soda-postgres.service','forgejo.service']){const s=unit(name);need(s.ActiveState==='active'&&Number(s.MainPID)>0,'native_service');services[name]={active:s.ActiveState,pid:Number(s.MainPID),fragment:s.FragmentPath,dropins:s.DropInPaths.split(' ').filter(Boolean)};}
 const h=services['soda-host.service'],exe=fs.realpathSync('/proc/'+h.pid+'/exe');need(exe==='/usr/libexec/soda/soda-host'&&!mounted(exe)&&run('/usr/bin/findmnt',['-rn','-T',exe,'-o','TARGET'])==='/'&&h.fragment==='/usr/lib/systemd/system/soda-host.service'&&h.dropins.every(x=>['/usr/lib/systemd/system/soda-host.service.d/10-images.conf','/usr/lib/systemd/system/service.d/10-timeout-abort.conf'].includes(x)),'stock_host');
 const maintain='/usr/libexec/soda/soda-project-maintain';need(!mounted(maintain)&&run('/usr/bin/findmnt',['-rn','-T',maintain,'-o','TARGET'])==='/','stock_maintenance_helper');
 for(const role of ['dashboard','forgejo','postgres','proxy'])need(podman(['inspect','--type','container','--format','{{.Image}}','soda-'+role]).replace(/^sha256:/,'')===p.Images[role].Config.slice(7),'stock_service_image_'+role);
 const base={source:p.Revision,payload_sha256:hash(bytes),application_configs:configs,services,host_sha256:hash(fs.readFileSync('/proc/'+h.pid+'/exe')),maintain_sha256:hash(fs.readFileSync(maintain)),dashboard_sha256:podman(['exec','soda-dashboard','sha256sum','/proc/1/exe']).split(/\s+/)[0],profile_http:200};
 if(REQUEST.mode==='installation'){console.log(JSON.stringify(base));process.exit(0)}
 const facts=sql("SELECT json_build_object('projects',(SELECT coalesce(json_agg(row_to_json(x)),'[]') FROM (SELECT id,repository_id::text,owner_id::text,repository,ready FROM projects ORDER BY id)x),'memberships',(SELECT coalesce(json_agg(row_to_json(x)),'[]') FROM (SELECT project_id,user_id,login FROM memberships ORDER BY project_id,user_id)x),'applications',(SELECT coalesce(json_agg(row_to_json(x)),'[]') FROM (SELECT id,data_id,owner_team,origin,revision FROM applications ORDER BY id)x),'connections',(SELECT coalesce(json_agg(row_to_json(x)),'[]') FROM (SELECT id,owner_id::text,generation,state FROM identity_connections ORDER BY id)x),'runs',(SELECT coalesce(json_agg(row_to_json(x)),'[]') FROM (SELECT id,active,settled,data->>'outcome' outcome,data->>'reconciled' reconciled FROM factory_runs ORDER BY id)x),'policies',(SELECT coalesce(json_agg(row_to_json(x)),'[]') FROM (SELECT repository,revision,data->>'enabled' enabled,data->>'paused' paused FROM factory_policies ORDER BY repository)x),'leases',(SELECT count(*) FROM identity_leases),'nonterminal',(SELECT count(*) FROM identity_executions WHERE state<>'terminal'),'held',(SELECT count(*) FROM factory_reservations WHERE state='held'),'unsettled',(SELECT count(*) FROM factory_runs WHERE active OR NOT settled OR coalesce(data->>'reconciled','false')<>'true'),'preparations',(SELECT count(*) FROM project_preparations WHERE coalesce(data->'state'->>'phase','') IN ('preparing','running')),'restores',(SELECT count(*) FROM application_restores WHERE state='pending'),'setup',(SELECT count(*) FROM application_setups WHERE state IN ('pending','running')),'effective_dispatch',(SELECT count(*) FROM factory_dispatch d JOIN factory_policies p USING(repository) WHERE d.open AND (p.data->>'enabled')::boolean AND NOT (p.data->>'paused')::boolean))");
 need(['leases','nonterminal','held','unsettled','preparations','restores','setup','effective_dispatch'].every(k=>facts[k]===0),'all_native_custody_idle');
 need(facts.policies.every(x=>x.paused==='true')&&facts.runs.every(x=>!x.active&&x.settled&&x.reconciled==='true'),'all_factory_paused_retired');
 const operator=JSON.parse(run('/usr/bin/curl',['--fail','--silent','--show-error','--max-time','20','--unix-socket','/run/soda/operator/operator.sock','http://localhost/operator/projects']));need(operator.factory_work===0&&operator.preparation_work===0,'operator_idle');
 const runtime=readResourceRuntime(facts,run,podman);
 need(runtime.active_units.length===0,'native_work_units_idle');
 const custody=Object.fromEntries(['leases','nonterminal','held','unsettled','preparations','restores','effective_dispatch'].map(k=>[k,facts[k]]));custody.setups=facts.setup;
 const history={runs:facts.runs.length,charged_minutes:null};
 const identities=(values,fields)=>values.map(x=>fields.map(k=>x[k])).sort((a,b)=>String(a[0]).localeCompare(String(b[0])));
 need(REQUEST.resources&&JSON.stringify(identities(runtime.resource_inventory.projects,['id','repository_id','owner_id']))===JSON.stringify(identities(REQUEST.resources.projects,['id','repository_id','owner_id']))&&JSON.stringify(identities(runtime.resource_inventory.applications,['id','data_id','owner_team','origin']))===JSON.stringify(identities(REQUEST.resources.applications,['id','data_id','owner_team','origin'])),'exact_shared_resource_inventory');
 if(REQUEST.mode==='snapshot'){
  need(runtime.runtime_orphans.length===0&&Array.isArray(runtime.terminals),'resource_inventory_confirmed');
  need([...runtime.project_runtime,...runtime.application_runtime].every(x=>x.unit_state==='inactive'&&x.main_pid===0&&x.control_pid===0&&!x.running),'all_resources_stopped');
 }
 const w=REQUEST.witness;need((REQUEST.mode==='idle'&&!w)||w&&facts.projects.some(x=>x.id===w.project)&&facts.memberships.some(x=>x.project_id===w.project&&x.login===w.login),'joined_project_witness');
 const projects=[];for(const project of facts.projects){
  need(/^p[a-f0-9]{24}$/.test(project.id),'canonical_project_id');
  const current=runtime.project_runtime.find(x=>x.id===project.id),s={ActiveState:current.unit_state,MainPID:current.main_pid,ControlPID:current.control_pid},cid=current.container_id,running=current.running;
  need(podman(['inspect','--type','container','--format','{{.Image}}',cid]).replace(/^sha256:/,'')===p.Images['project-os'].Config.slice(7),'stock_project_image');
  const terminals=current.terminal_ids;
  const observed={project:project.id,container:cid,running,unit:s.ActiveState,main_pid:s.MainPID,control_pid:s.ControlPID,terminals};
  if(REQUEST.mode==='idle'){projects.push(observed);continue;}
  need(REQUEST.mode==='snapshot'&&s.ActiveState==='inactive'&&Number(s.MainPID)===0&&Number(s.ControlPID)===0&&!running,'all_projects_stopped');
  const mounts=JSON.parse(podman(['inspect','--type','container','--format','{{json .Mounts}}',cid]));need(!mounts.some(x=>x.Destination==='/usr/libexec/soda/project-factory-roles'),'stock_project_helper_mounts');
  const root=podman(['mount',cid]);
  try{
   need(root.startsWith('/var/lib/containers/storage/overlay/')&&root.endsWith('/merged'),'owned_project_root');
   const passwd=fs.readFileSync(root+'/etc/passwd','utf8').trim().split('\n').map(x=>x.split(':'));
   const accounts=passwd.filter(x=>x[0]==='root'||x[0].startsWith('soda-')||x[0]===w.login).map(x=>({login:x[0],uid:Number(x[2]),gid:Number(x[3]),home:x[5],shell:x[6]}));
   const helper=root+'/usr/libexec/soda/project-factory-roles';need(!mounted(helper),'stock_project_helper');
   const map=podman(['inspect','--type','container','--format','{{json .HostConfig.IDMappings}}',cid]);
   const mappings=JSON.parse(map),mapped=(id,values)=>{need(Array.isArray(values)&&values.length===1,'canonical_private_id_mapping');const parts=values[0].split(':').map(Number);need(parts.length===3&&parts.every(Number.isSafeInteger)&&parts[0]===0&&parts[1]>0&&parts[2]===262144&&id>=0&&id<parts[2],'canonical_private_id_mapping');return parts[1]+id};
   let sample={};if(project.id===w.project){
    const account=accounts.filter(x=>x.login===w.login);need(account.length===1&&account[0].home===w.home,'actual_member_home');
    const path=root+w.home+'/'+w.relative;need(fs.realpathSync(root+w.home)===root+w.home&&fs.realpathSync(path)===path,'witness_no_symlink');const st=fs.lstatSync(path);need(st.isFile()&&!st.isSymbolicLink()&&st.nlink===1&&(st.mode&511)===384&&st.size>0&&st.size<=4096,'private_member_file');
    need(st.uid===mapped(account[0].uid,mappings.UidMap)&&st.gid===mapped(account[0].gid,mappings.GidMap),'actual_member_file_owner');
    const actual=hash(fs.readFileSync(path));need(actual===w.sha256,'actual_member_file');
    sample={login:w.login,home:w.home,passwd_uid:account[0].uid,passwd_gid:account[0].gid,relative:w.relative,sha256:actual,bytes:st.size,mode:st.mode&511,uid:st.uid,gid:st.gid,inode:st.ino};
   }
   projects.push({...observed,accounts,...sample,id_mappings_sha256:hash(map),helper_sha256:hash(fs.readFileSync(helper))});
  }finally{podman(['unmount',cid]);}
 }
 const sample=w?projects.find(x=>x.project===w.project):null;
 if(REQUEST.mode==='idle'){console.log(JSON.stringify({...base,...runtime,custody,history,facts,projects,project:sample}));process.exit(0);}
 // Hash canonical retained rows server-side, excluding all credential/output
 // values from returned metadata. Only stable business tables are included.
 const ledgers={};for(const table of ['users','projects','memberships','applications','application_revisions','application_projects','factory_assignments','factory_attempt_allowances','factory_runs','factory_reservations','factory_usage','factory_policies','factory_dispatch','factory_sponsorships','factory_connection_usage_budgets','identity_grants','identity_executions']){
  const row=sql("SELECT json_build_object('count',count(*),'md5',md5(coalesce(string_agg(v,'|' ORDER BY v),''))) FROM (SELECT row_to_json(x)::text v FROM "+table+" x)t");ledgers[table]=row;
 }
 const identity=JSON.parse(fs.readFileSync('/etc/soda/identity.json','utf8'));const k=fs.statSync(identity.key_file);need(k.isFile()&&k.uid===2001&&(k.mode&511)===384,'broker_key_metadata');
 const key={uid:k.uid,gid:k.gid,mode:k.mode&511,inode:k.ino,bytes:k.size,mtime_ms:k.mtimeMs};
 const protectedState={payload_sha256:base.payload_sha256,application_configs:configs,facts,ledgers,projects,key_metadata:key,public_ca_sha256:hash(fs.readFileSync('/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt'))};
 console.log(JSON.stringify({...base,...runtime,custody,history,protected:protectedState,protected_sha256:hash(JSON.stringify(protectedState)),project:sample}));
}catch(e){console.error(JSON.stringify({reason:e instanceof Error&&/^[a-z0-9_]+$/.test(e.message)?e.message:'inner_observation_unavailable'}));process.exitCode=1;}
`;

// Outer host program is the only place Docker is addressed. The SSH target key
// is fixed by the controller. No socket or caller command enters the collector.
export const OUTER_READER = String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
const need=(x,r)=>{if(!x)throw Error(r)};const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const run=(p,a,input,timeout=30000)=>cp.execFileSync(p,a,{input,encoding:'utf8',timeout,maxBuffer:3<<20,stdio:[input===undefined?'ignore':'pipe','pipe','pipe']}).trim();
const d=a=>run('/usr/bin/docker',['--host','unix:///var/run/docker.sock',...a]);
const protectedFile=p=>{const s=fs.lstatSync(p);need(s.isFile()&&!s.isSymbolicLink()&&s.nlink===1&&s.uid===0&&(s.mode&63)===0,'protected_installer_file');return fs.readFileSync(p)};
try{
 need(process.getuid()===0,'outer_root');
 const root=REQUEST.host.install_directory,rootStat=fs.lstatSync(root);need(rootStat.isDirectory()&&!rootStat.isSymbolicLink()&&rootStat.uid===0&&(rootStat.mode&511)===448,'protected_install_directory');
 const identityBytes=protectedFile(root+'/identity.json'),identity=JSON.parse(identityBytes),metadataBytes=protectedFile(root+'/docker.json'),meta=JSON.parse(metadataBytes),compose=protectedFile(root+'/compose.yaml');
 need(identity.format===1&&identity.release===REQUEST.tag&&identity.source_revision===REQUEST.source&&identity.image===REQUEST.image&&identity.metadata_sha256===REQUEST.metadataSHA&&identity.compose_sha256===REQUEST.composeSHA,'public_installer_identity');
 need(hash(metadataBytes)===REQUEST.metadataSHA&&hash(compose)===REQUEST.composeSHA&&meta.CandidateSHA256===REQUEST.candidateSHA&&meta.PayloadSHA256===REQUEST.payloadSHA,'installed_metadata_compose');
 need(protectedFile(root+'/.env').toString()==='SODA_IMAGE='+REQUEST.image+'\nSODA_PROJECT_SUBNET='+identity.project_subnet+'\n','fixed_compose_environment');
 const info=JSON.parse(d(['info','--format','{{json .}}']));need(info.OSType==='linux'&&['x86_64','amd64'].includes(info.Architecture)&&String(info.CgroupVersion)==='2'&&!info.SecurityOptions.some(x=>x.includes('rootless')),'rootful_cgroup2_engine');
 const ci=JSON.parse(d(['container','inspect','--format','{"id":{{json .Id}},"image":{{json .Image}},"descriptor":{{json .ImageManifestDescriptor}},"reference":{{json .Config.Image}},"labels":{{json .Config.Labels}},"privileged":{{json .HostConfig.Privileged}},"network":{{json .HostConfig.NetworkMode}},"mounts":{{json .Mounts}},"running":{{json .State.Running}}}','sodaos-soda-1']));
 need(/^[a-f0-9]{64}$/.test(ci.id)&&ci.running&&ci.reference===REQUEST.image&&(ci.image===REQUEST.config||ci.image===REQUEST.manifest&&ci.descriptor?.digest===REQUEST.manifest)&&ci.privileged&&ci.network==='host'&&ci.labels['com.docker.compose.project']==='sodaos'&&ci.labels['com.docker.compose.service']==='soda','exact_running_outer');
 const vendor=['/usr/libexec/soda/soda-host','/usr/libexec/soda/soda-project-maintain','/usr/lib/systemd/system/soda-host.service','/usr/lib/systemd/system/soda-host.service.d/10-images.conf','/usr/lib/systemd/system/service.d/10-timeout-abort.conf'];
 const changes=d(['container','diff','sodaos-soda-1']).split('\n').filter(Boolean);need(changes.every(x=>/^[ACD] \/[^\n]*$/.test(x))&&!changes.some(x=>vendor.includes(x.slice(2))),'stock_vendor_files_unchanged');
 const expected=[{Name:'sodaos_config',Destination:'/etc'},{Name:'sodaos_home',Destination:'/home'},{Name:'sodaos_state',Destination:'/var'}];
 need(ci.mounts.length===3&&ci.mounts.every(x=>x.Type==='volume')&&JSON.stringify(ci.mounts.map(x=>({Name:x.Name,Destination:x.Destination})).sort((a,b)=>a.Destination.localeCompare(b.Destination)))===JSON.stringify(expected),'exact_three_volumes');
 const volumes=expected.map(x=>{const v=JSON.parse(d(['volume','inspect','--format','{{json .}}',x.Name]));need(v.Name===x.Name&&v.Driver==='local'&&v.Labels['com.docker.compose.project']==='sodaos'&&v.Labels['com.docker.compose.volume']===x.Name.slice(7)&&(v.Options===null||Object.keys(v.Options).length===0),'owned_named_volume');return{name:v.Name,destination:x.Destination,created_at:v.CreatedAt,mountpoint:v.Mountpoint,labels:v.Labels,driver:v.Driver};});
 const fixture=fs.lstatSync(REQUEST.host.fixture_directory);need(fixture.isDirectory()&&!fixture.isSymbolicLink()&&fixture.uid===0&&(fixture.mode&511)===448,'protected_operator_fixture');
 const installer=REQUEST.host.fixture_directory+'/public-install-docker.sh';const script=protectedFile(installer);need(script.length===REQUEST.installerBytes&&hash(script)===REQUEST.installerSHA,'actual_public_installer_bytes');
 const executionBytes=protectedFile(REQUEST.host.fixture_directory+'/installer-execution.json'),execution=JSON.parse(executionBytes);
 need(execution.format===1&&execution.action==='public-streamed-install'&&execution.source===REQUEST.source&&execution.image===REQUEST.image&&execution.installer_sha256===REQUEST.installerSHA&&execution.exit_code===0&&execution.install_dir===root&&execution.initial_containers===0&&execution.initial_volumes===0&&execution.initial_images===0,'actual_installer_execution');
 need(execution.installer_url==='https://github.com/LevitateOS/sodaos/releases/download/'+REQUEST.tag+'/install-docker.sh'&&JSON.stringify(execution.arguments)===JSON.stringify(['--qualification','--yes','--bind-ip',identity.bind_ip,'--project-subnet',identity.project_subnet,'--install-dir',root]),'actual_canonical_installer_arguments');
 const started=Date.parse(execution.started_at),finished=Date.parse(execution.finished_at);need(Number.isFinite(started)&&finished>started&&finished<=Date.now()+5000&&volumes.every(x=>Date.parse(x.created_at)>=started-1000&&Date.parse(x.created_at)<=finished+1000),'fresh_volume_creation');
 const inner=JSON.parse(run('/usr/bin/docker',['--host','unix:///var/run/docker.sock','exec','-i','sodaos-soda-1','/usr/bin/node','-'],'const REQUEST='+JSON.stringify(REQUEST)+';\n'+INNER,180000));
 console.log(JSON.stringify({target:REQUEST.target,host_uuid:fs.readFileSync("/sys/class/dmi/id/product_uuid","utf8").trim().toLowerCase(),outer_id:ci.id,engine_image:ci.image,registry_manifest:REQUEST.manifest,registry_config:REQUEST.config,stock_vendor_files_unchanged:true,identity_sha256:hash(identityBytes),metadata_sha256:hash(metadataBytes),compose_sha256:hash(compose),installer_sha256:hash(script),installer_execution_sha256:hash(executionBytes),installer_execution_text:executionBytes.toString('utf8'),installer_execution:execution,volumes,os:info.OSType,architecture:'x86_64',rootful:true,cgroup_version:2,host_network:true,fresh_volumes:true,inner}));
}catch(e){console.error(JSON.stringify({reason:e instanceof Error&&/^[a-z0-9_]+$/.test(e.message)?e.message:'docker_observation_unavailable'}));process.exitCode=1;}
`;

export function nativeReadScript(
  b: DockerBinding,
  mode: 'installation' | 'idle' | 'snapshot',
  witness?: Witness,
  scope?: ResourceScope
) {
  if (mode !== 'installation') check(scope, 'shared_resource_scope_required');
  if (mode === 'snapshot' || witness) validateWitness(witness!);
  return (
    'const REQUEST=' +
    JSON.stringify({
      ...nativeRequest(b, witness),
      mode,
      ...(scope
        ? {
            resources: {
              projects: scope.observation.inventory.projects,
              applications: scope.observation.inventory.applications,
            },
          }
        : {}),
    }) +
    ';\nconst INNER=' +
    JSON.stringify(INNER_READER) +
    ';\n' +
    OUTER_READER
  );
}
export async function readDocker(
  io: DockerIO,
  b: DockerBinding,
  mode: 'installation' | 'idle' | 'snapshot',
  witness?: Witness,
  scope?: ResourceScope
) {
  check(b.executionReleased === true, 'docker_execution_held');
  const value = JSON.parse(await io.sshRead(nativeReadScript(b, mode, witness, scope)));
  check(
    value.target === b.target &&
      value.registry_manifest === b.manifest &&
      value.registry_config === b.config &&
      value.inner.source === b.source &&
      value.inner.payload_sha256 === b.payloadSHA,
    'docker_readback_binding'
  );
  check(
    Object.keys(value.inner.application_configs).sort().join(',') === Object.keys(b.payload.Images).sort().join(','),
    'all_images_measured'
  );
  for (const [role, image] of Object.entries(b.payload.Images) as any[])
    check(value.inner.application_configs[role] === image.Config, 'measured_image_' + role);
  check(value.stock_vendor_files_unchanged === true, 'stock_vendor_image_files');
  const hostPin = b.candidate.ContentSHA256['host:/usr/libexec/soda/soda-host'];
  if (hostPin !== undefined) check(value.inner.host_sha256 === hostPin, 'stock_host_bytes');
  const maintainPin = b.candidate.ContentSHA256['host:/usr/libexec/soda/soda-project-maintain'];
  if (maintainPin !== undefined) check(value.inner.maintain_sha256 === maintainPin, 'stock_maintenance_bytes');
  check(
    value.inner.dashboard_sha256 === b.candidate.ContentSHA256['dashboard:/usr/local/bin/soda-dashboard'],
    'stock_dashboard_bytes'
  );
  if (mode !== 'installation') {
    check(scope, 'shared_resource_scope_required');
    check(value.host_uuid === scope.scenario.target.uuid, 'exact_docker_host_uuid');
    assertResourceInventory(scope, value.inner.resource_inventory);
    if (mode === 'snapshot')
      check(
        diagnoseClosure(scope.scenario, scope.observation, value.inner).snapshot_ready,
        'shared_snapshot_closure_required'
      );
  }
  if (mode === 'snapshot')
    check(
      value.inner.protected_sha256 === sha(JSON.stringify(value.inner.protected)) &&
        value.inner.project.sha256 === witness!.sha256,
      'protected_native_state'
    );
  return value;
}
export async function observeInstallation(io: DockerIO, b: DockerBinding) {
  const measured = await readDocker(io, b, 'installation');
  check(
    sha(measured.installer_execution_text) === measured.installer_execution_sha256,
    'exact_installer_execution_bytes'
  );
  await io.saveBytes(
    'installation/installer-execution.json',
    new TextEncoder().encode(measured.installer_execution_text)
  );
  await io.save('installation/readback.json', measured);
  const native = {
    target: b.target,
    tag: b.tag,
    registry_manifest: b.manifest,
    registry_config: b.config,
    compose_sha256: measured.compose_sha256,
    installer_sha256: measured.installer_sha256,
    installer_execution_sha256: measured.installer_execution_sha256,
    candidate_sha256: b.candidateSHA,
    payload_sha256: b.payloadSHA,
    soda_revision: b.source,
    forgejo_revision: b.forgejo,
    application_configs: measured.inner.application_configs,
    os: measured.os,
    architecture: measured.architecture,
    rootful: measured.rootful,
    cgroup_version: measured.cgroup_version,
    host_network: measured.host_network,
    fresh_volumes: measured.fresh_volumes,
    outer_id: measured.outer_id,
  };
  await io.save('installation/native.json', native);
  return {native, measured};
}
