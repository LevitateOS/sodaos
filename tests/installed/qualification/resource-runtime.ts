// One fixed read-only native inventory shared by readiness and phase collection.
// Returned terminal metadata excludes names, screen/output bytes and credentials.
const TERMINAL_DIRECTORIES = String.raw`set -eu
check_dir(){ [ -d "$1" ] && [ ! -L "$1" ]; set -- $(stat -c '%u %a' "$1"); [ "$1" = 0 ]; [ $((0$2 & 022)) = 0 ]; }
check_dir /; check_dir /run
if [ -e /run/soda-terminals ] || [ -L /run/soda-terminals ]; then
 check_dir /run/soda-terminals
 count=0
 for d in /run/soda-terminals/*; do
  [ -e "$d" ] || [ -L "$d" ] || continue
  check_dir "$d"
  count=$((count+1)); [ "$count" -le 128 ]
  printf '%s\n' "$(basename "$d")"
 done
fi`;

const TERMINAL_BINDING = String.raw`set -eu
identifier=$1
check_dir(){ [ -d "$1" ] && [ ! -L "$1" ]; set -- $(stat -c '%u %a' "$1"); [ "$1" = 0 ]; [ $((0$2 & 022)) = 0 ]; }
check_dir /; check_dir /run; check_dir /run/soda-terminals; check_dir /run/soda-terminals/$identifier
file=/run/soda-terminals/$identifier/binding
[ -f "$file" ] && [ ! -L "$file" ]
before=$(stat -c '%u %a %h %s %d %i' "$file")
set -- $before; [ "$1" = 0 ]; [ $((0$2 & 077)) = 0 ]; [ "$3" = 1 ]; [ "$4" -gt 0 ] && [ "$4" -le 4096 ]
exec 3<"$file"
[ "$(stat -Lc '%u %a %h %s %d %i' /proc/$$/fd/3)" = "$before" ]
[ ! -L "$file" ] && [ "$(stat -c '%u %a %h %s %d %i' "$file")" = "$before" ]
head -c 4097 /proc/$$/fd/3
printf '\n'
if [ -e /run/soda-terminals/$identifier/subscription ] || [ -L /run/soda-terminals/$identifier/subscription ]; then printf 'subscription\n'; else printf 'ordinary\n'; fi`;

const TERMINAL_ACCOUNT = String.raw`set -eu
login=$1
check_dir(){ [ -d "$1" ] && [ ! -L "$1" ]; set -- $(stat -c '%u %a' "$1"); [ "$1" = 0 ]; [ $((0$2 & 022)) = 0 ]; }
check_dir /; check_dir /var; check_dir /var/lib; check_dir /var/lib/soda; check_dir /var/lib/soda/accounts
file=/var/lib/soda/accounts/$login
[ ! -e "$file.access" ] && [ ! -L "$file.access" ]
[ -f "$file" ] && [ ! -L "$file" ]
before=$(stat -c '%u %a %h %s %d %i' "$file")
set -- $before; [ "$1" = 0 ]; [ $((0$2 & 077)) = 0 ]; [ "$3" = 1 ]; [ "$4" -gt 0 ] && [ "$4" -le 32 ]
exec 3<"$file"
[ "$(stat -Lc '%u %a %h %s %d %i' /proc/$$/fd/3)" = "$before" ]
[ ! -L "$file" ] && [ "$(stat -c '%u %a %h %s %d %i' "$file")" = "$before" ]
head -c 33 /proc/$$/fd/3; printf '\n'
getent passwd $login`;

export const RESOURCE_RUNTIME_READER =
  'const RESOURCE_TERMINAL_DIRECTORIES=' +
  JSON.stringify(TERMINAL_DIRECTORIES) +
  ';\n' +
  'const RESOURCE_TERMINAL_BINDING=' +
  JSON.stringify(TERMINAL_BINDING) +
  ';\n' +
  'const RESOURCE_TERMINAL_ACCOUNT=' +
  JSON.stringify(TERMINAL_ACCOUNT) +
  ';\n' +
  String.raw`
function readResourceRuntime(identity,run,podman){
 const need=(value,reason)=>{if(!value)throw Error(reason)};
 const hex=/^[a-f0-9]{64}$/,projectID=/^p[a-f0-9]{24}$/,appID=/^a[a-f0-9]{24}$/,terminalID=/^[a-f0-9]{32}$/;
 const bounded=values=>{need(values.length<=256,'bounded_resource_inventory');return values};
 const lines=text=>bounded(text.split('\n').filter(Boolean));
 const unique=values=>{need(new Set(values).size===values.length,'duplicate_resource_identity');return values};
 const runtime_orphans=[],active_units=[],terminals=[];
 let terminalActorsConfirmed=true;
 const invoke=(cid,args)=>cid?podman(['exec',cid,'/usr/bin/systemctl',...args]):run('/usr/bin/systemctl',args);
 const state=(name,cid=null)=>{
  const pairs=lines(invoke(cid,['show',name,'-p','ActiveState','-p','MainPID','-p','ControlPID'])).map(line=>{const i=line.indexOf('=');need(i>0,'native_unit_fields');return[line.slice(0,i),line.slice(i+1)]});
  need(pairs.length===3&&new Set(pairs.map(x=>x[0])).size===3,'native_unit_fields');
  const fields=Object.fromEntries(pairs);
  need(['active','inactive','failed','activating','deactivating','reloading','maintenance','refreshing'].includes(fields.ActiveState),'native_unit_state');
  need(/^(0|[1-9][0-9]*)$/.test(fields.MainPID)&&/^(0|[1-9][0-9]*)$/.test(fields.ControlPID),'native_unit_pids');
  const main_pid=Number(fields.MainPID),control_pid=Number(fields.ControlPID);
  need(Number.isSafeInteger(main_pid)&&Number.isSafeInteger(control_pid),'native_unit_pids');
  return{unit_state:fields.ActiveState,main_pid,control_pid};
 };
 const unitInventory=(cid=null,project=null)=>{
  const patterns=cid?['soda-factory-*.service','soda-prepare-*.service','soda-terminal-*.service']:['soda-factory-*.service','soda-prepare-*.service','soda-project@*.service','soda-canlang@*.service'];
  const names=unique(lines(invoke(cid,['list-units','--all','--plain','--no-legend','--no-pager',...patterns])).map(line=>line.trim().split(/\s+/)[0]));
  for(const name of names){
   need(/^soda-(?:(?:factory|prepare)-[A-Za-z0-9_.-]+|project@p[a-f0-9]{24}|canlang@a[a-f0-9]{24}|terminal-[a-f0-9]{32})\.service$/.test(name),'native_unit_identity');
   const observed=state(name,cid);
   if(/^soda-(factory|prepare)-/.test(name)){
    if(!['inactive','failed'].includes(observed.unit_state)||observed.main_pid!==0||observed.control_pid!==0)active_units.push({project_container:cid,unit:name,state:observed.unit_state,main_pid:observed.main_pid,control_pid:observed.control_pid});
   }else if(name.startsWith('soda-terminal-')){
    const id=name.slice('soda-terminal-'.length,-'.service'.length);
    if(!project.terminal_ids.includes(id)&&(!['inactive','failed'].includes(observed.unit_state)||observed.main_pid!==0||observed.control_pid!==0))runtime_orphans.push({kind:'terminal',id,project_id:project.id,source:'unit',...observed});
   }else{
    const kind=name.startsWith('soda-project@')?'project':'application',id=name.slice(name.indexOf('@')+1,-'.service'.length);
    if(!identity[kind==='project'?'projects':'applications'].some(row=>row.id===id))runtime_orphans.push({kind,id,source:'unit',...observed});
   }
  }
 };
 const containers=(kind,pattern,label)=>{
  const ids=[...new Set([...lines(podman(['ps','--all','--no-trunc','--filter','label='+label,'--format','{{.ID}}'])),...lines(podman(['ps','--all','--no-trunc','--filter','name='+pattern,'--format','{{.ID}}']))])];
  bounded(ids);need(ids.every(id=>hex.test(id)),'native_container_identity');
  const result=new Map();
  for(const cid of ids){
   const format='{"id":{{json .ID}},"name":{{json .Name}},"running":{{json .State.Running}},"identity":{{json (index .Config.Labels "'+label+'")}}}';
   const row=JSON.parse(podman(['container','inspect','--format',format,cid]));
   need(row.id===cid&&typeof row.running==='boolean'&&(kind==='project'?projectID:appID).test(row.identity),'native_container_binding');
   need(row.name===(kind==='project'?'soda-':'soda-canlang-')+row.identity&&!result.has(row.identity),'native_container_binding');
   result.set(row.identity,row);
   if(!identity[kind==='project'?'projects':'applications'].some(item=>item.id===row.identity))runtime_orphans.push({kind,id:row.identity,source:'container',container_id:cid,running:row.running});
  }
  return result;
 };
 need(Array.isArray(identity.projects)&&Array.isArray(identity.applications),'registered_resource_inventory');
 bounded(identity.projects);bounded(identity.applications);
 unique(identity.projects.map(row=>row.id));unique(identity.applications.map(row=>row.id));
 need(identity.projects.every(row=>projectID.test(row.id))&&identity.applications.every(row=>appID.test(row.id)),'registered_resource_identity');
 const projectContainers=containers('project','^soda-p[0-9a-f]{24}$','org.soda.project');
 const appContainers=containers('application','^soda-canlang-a[0-9a-f]{24}$','io.soda.canlang.id');
 unitInventory();
 const project_runtime=identity.projects.map(p=>{
  const native=projectContainers.get(p.id);need(native,'registered_project_container_missing');
  const observed={id:p.id,container_id:native.id,running:native.running,...state('soda-project@'+p.id+'.service'),terminal_ids:[]};
  if(native.running){
   observed.terminal_ids=unique(lines(podman(['exec',native.id,'/bin/sh','-eu','-c',RESOURCE_TERMINAL_DIRECTORIES])));
   need(observed.terminal_ids.every(id=>terminalID.test(id)),'terminal_inventory_identity');
   unitInventory(native.id,observed);
   for(const id of observed.terminal_ids){
    try{
     const text=podman(['exec',native.id,'/bin/sh','-eu','-c',RESOURCE_TERMINAL_BINDING,'resource-binding',id]),split=text.lastIndexOf('\n'),binding=JSON.parse(text.slice(0,split)),subscription=text.slice(split+1);
     need(['ordinary','subscription'].includes(subscription),'terminal_subscription');
     need(binding&&Object.keys(binding).sort().join(',')==='account,cols,created_at,identity,rows'&&Array.isArray(binding.account)&&binding.account.length===5,'terminal_binding');
     const [login,uid,gid,home,shell]=binding.account;
     need(typeof login==='string'&&/^[a-z][a-z0-9_-]{0,30}$/.test(login)&&login!=='root','terminal_login');
     need(Number.isSafeInteger(uid)&&uid>0&&uid<=4294967295&&Number.isSafeInteger(gid)&&gid>=0&&gid<=4294967295,'terminal_account');
     need(typeof home==='string'&&home.startsWith('/')&&typeof shell==='string'&&shell.startsWith('/'),'terminal_account');
     need(Number.isSafeInteger(binding.identity)&&binding.identity>0&&Number.isSafeInteger(binding.created_at)&&binding.created_at>0&&Number.isSafeInteger(binding.cols)&&binding.cols>=2&&binding.cols<=500&&Number.isSafeInteger(binding.rows)&&binding.rows>=2&&binding.rows<=300,'terminal_binding');
     const account=lines(podman(['exec',native.id,'/bin/sh','-eu','-c',RESOURCE_TERMINAL_ACCOUNT,'resource-account',login]));
     need(account.length===2&&account[0]===String(binding.identity),'terminal_account_marker');
     const passwd=account[1].split(':');need(passwd.length===7&&passwd[0]===login&&passwd[2]===String(uid)&&passwd[3]===String(gid)&&passwd[5]===home&&passwd[6]===shell&&!['false','nologin'].includes(shell.split('/').pop()),'terminal_account_changed');
     terminals.push({id,project_id:p.id,actor_id:String(binding.identity),login,subscription:subscription==='subscription'});
    }catch{terminalActorsConfirmed=false;}
   }
  }
  return observed;
 });
 const application_runtime=identity.applications.map(a=>{const native=appContainers.get(a.id);return{id:a.id,container_id:native?native.id:null,running:native?native.running:false,...state('soda-canlang@'+a.id+'.service')};});
 const confirmedTerminals=terminalActorsConfirmed?terminals:null;
 need(Array.isArray(identity.connections),'registered_connection_inventory');bounded(identity.connections);
 const resource_inventory={
  projects:identity.projects.map(p=>({id:p.id,repository_id:String(p.repository_id),owner_id:String(p.owner_id)})),
  applications:identity.applications.map(a=>({id:a.id,data_id:a.data_id,owner_team:a.owner_team,origin:a.origin})),
  terminals:confirmedTerminals,
  connections:identity.connections.map(c=>({id:c.id,owner_id:String(c.owner_id),generation:c.generation,state:c.state}))
 };
 return{resource_inventory,project_runtime,application_runtime,terminals:confirmedTerminals,active_units,runtime_orphans};
}
`;
