import {check, COMMIT} from './support';
import {validateFactoryPhase} from './factory-native';

// No token is returned to the collector. The one root-owned guest token is
// an independently enrolled CI actor, never a provider credential or new grant.
export const FACTORY_CI = String.raw`
const fs=require('fs'),cp=require('child_process'),crypto=require('crypto'),http=require('http');
const digest=v=>crypto.createHash('sha256').update(v).digest('hex');
const pod=args=>cp.execFileSync('/usr/bin/podman',['--remote=false',...args],{encoding:'utf8',timeout:20000,maxBuffer:1<<20,stdio:['ignore','pipe','pipe']}).trim();
const sql=statement=>JSON.parse(pod(['exec','soda-postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-U','postgres','-d','soda','-c','BEGIN READ ONLY;'+statement+';COMMIT;']));
const localHTTP=(socket,path,body)=>new Promise((resolve,reject)=>{const raw=Buffer.from(JSON.stringify(body));const r=http.request({socketPath:socket,path,method:'POST',headers:{'Content-Type':'application/json','Content-Length':raw.length}},res=>{let chunks=[],size=0;res.on('data',b=>{size+=b.length;if(size>1<<20){r.destroy();reject(Error());}else chunks.push(b);});res.on('end',()=>{try{if(res.statusCode!==200)throw Error();resolve(JSON.parse(Buffer.concat(chunks).toString()));}catch{reject(Error());}});});r.setTimeout(20000,()=>r.destroy());r.on('error',()=>reject(Error()));r.end(raw);});
let issued=false;
(async()=>{
 if(process.getuid()!==0)throw Error();
 const s=REQUEST.scope,x=REQUEST.check,R=s.repository.id,I=s.issue.number,P=s.repository.project;
 if(!/^f[a-f0-9]{24}$/.test(x.preparation)||!/^([a-f0-9]{40})$/.test(x.head)||!/^([a-f0-9]{32})$/.test(x.publication)||!Number.isSafeInteger(x.pr)||x.pr<=0||!Number.isSafeInteger(x.baseline_unix_ms))throw Error();
 if(x.before_preparations.includes(x.preparation)||x.before_preparations.length>256)throw Error();
 const record=()=>sql("SELECT json_build_object('id',p.id,'role',p.role,'project',p.project_id,'source',p.data#>>'{preparation,source_commit}','requirements',p.requirements,'approval',p.approval,'setup_digest',p.data#>>'{preparation,setup_digest}','owner',p.data#>>'{factory_admission,owner_assignment}','root',p.data#>>'{factory_admission,attempt_root}','publication',p.data#>>'{factory_admission,publication_id}','not_after',p.data#>>'{factory_admission,not_after_unix}','candidate',u.data->>'candidate','acceptance',u.data->>'acceptance','pr_number',u.data->'pr_number','stage',u.stage,'publication_owner',u.assignment,'publication_id',u.data->>'id') FROM project_preparations p JOIN factory_publications u ON u.assignment=p.data#>>'{factory_admission,owner_assignment}' AND u.data->>'id'=p.data#>>'{factory_admission,publication_id}' WHERE p.id='"+x.preparation+"' AND p.project_id='"+P+"' AND u.repository="+R+" AND u.issue="+I);
 const validate=r=>{if(!r||r.role!=='soda-reviewer'||r.project!==P||r.source!==x.head||r.candidate!==x.head||r.acceptance!==s.issue.acceptance||r.requirements!==s.requirements||r.approval!==s.approval||r.publication!==x.publication||r.publication_id!==x.publication||r.publication_owner!==x.owner_assignment||r.owner!==x.owner_assignment||r.root!==x.attempt_root||Number(r.pr_number)!==x.pr||r.stage!=='published'||Number(r.not_after)<=Date.now()/1000||!/^([a-f0-9]{32})$/.test(r.owner)||!/^([a-f0-9]{32})$/.test(r.root))throw Error();};
 const first=record();validate(first);
 const inspect=async()=>{
  const n=await localHTTP('/run/soda/host/host.sock','/prepare-inspect',{project:P,id:x.preparation});
  if(n.id!==x.preparation||n.project!==P||n.role!=='soda-reviewer'||n.source_commit!==x.head||n.setup_digest!==first.setup_digest||n.phase!=='ready'||n.ready!==true||n.stopped!==false||n.setup_exit!==0||n.check_exit!==0)throw Error();
  return{id:n.id,project:n.project,role:n.role,source_commit:n.source_commit,setup_digest:n.setup_digest,phase:n.phase,ready:n.ready,stopped:n.stopped,setup_exit:n.setup_exit,check_exit:n.check_exit,output_digest:digest(Buffer.from(n.output||'')),output_bytes:Buffer.byteLength(n.output||'')};
 };
 const measured=await inspect();
 const account=pod(['exec','soda-'+P,'getent','passwd','soda-reviewer']).split(':');if(account.length!==7||account[0]!=='soda-reviewer'||!account[5].startsWith('/'))throw Error();
 const code='set -eu; dir=/var/lib/soda/factory/preparations/$1; for p in "$dir/request.json" "$dir/started.json" "$dir/finished.json" "$dir/snapshot/check.sh"; do test -f "$p" && test ! -L "$p"; stat -c "%u:%g:%a:%h:%Y:%s" "$p"; done; sha256sum "$dir/snapshot/check.sh"';
 const lines=pod(['exec','--user','0','soda-'+P,'/bin/sh','-eu','-c',code,'qualification-candidate-check',x.preparation]).split('\n');if(lines.length!==5)throw Error();
 const times=lines.slice(0,4).map(line=>{const a=line.split(':');if(a.length!==6||a[0]!=='0'||a[1]!=='0'||a[2]!=='644'||a[3]!=='1'||!a.slice(4).every(x=>/^[0-9]+$/.test(x)))throw Error();return{mtime_unix:Number(a[4]),bytes:Number(a[5])};});
 if(times.slice(0,3).some(t=>t.mtime_unix*1000<x.baseline_unix_ms-2000)||times[2].mtime_unix<times[1].mtime_unix||lines[4].split(/\s+/)[0]!==s.approved_check_sha256)throw Error();
 const checkout=account[5]+'/checkouts/'+x.preparation;
 if(pod(['exec','--user','soda-reviewer','soda-'+P,'git','--no-optional-locks','-C',checkout,'rev-parse','HEAD'])!==x.head)throw Error();
 // The CI context is a status name. Its executable source path is separately
 // pinned, and must match the canonical protected preparation check snapshot.
 const sourceCheck=cp.execFileSync('/usr/bin/podman',['--remote=false','exec','--user','soda-reviewer','soda-'+P,'git','--no-optional-locks','-C',checkout,'show',x.head+':'+s.check_script_path],{timeout:20000,maxBuffer:1<<20,stdio:['ignore','pipe','pipe']});
 if(digest(sourceCheck)!==s.approved_check_sha256)throw Error();
 const tokenPath='/var/lib/soda-qualification/ci-token';
 for(let p=tokenPath;;p=require('path').dirname(p)){const m=fs.lstatSync(p);if(m.uid!==0||m.isSymbolicLink()||(m.mode&0o022))throw Error();if(p===tokenPath&&(!m.isFile()||(m.mode&0o777)!==0o600||m.nlink!==1||m.size>4096))throw Error();if(p==='/')break;}
 const token=fs.readFileSync(tokenPath,'utf8').trim();if(!/^[a-f0-9]{40}$/.test(token))throw Error();
 const api=(method,path,body)=>new Promise((resolve,reject)=>{const raw=body===undefined?undefined:Buffer.from(JSON.stringify(body));const r=http.request({hostname:'127.0.0.1',port:3000,path:'/api/v1'+path,method,headers:{Authorization:'token '+token,'Content-Type':'application/json',...(raw?{'Content-Length':raw.length}:{})}},res=>{let chunks=[],size=0;res.on('data',b=>{size+=b.length;if(size>1<<20){r.destroy();reject(Error());}else chunks.push(b);});res.on('end',()=>{try{if(res.statusCode<200||res.statusCode>=300)throw Error();resolve(JSON.parse(Buffer.concat(chunks).toString()));}catch{reject(Error());}});});r.setTimeout(20000,()=>r.destroy());r.on('error',()=>reject(Error()));r.end(raw);});
 const actor=await api('GET','/user');if(actor.id!==s.actors.ci||actor.login!==s.ci_login||actor.is_admin!==false)throw Error();
 const address='/repos/'+s.repository.owner+'/'+s.repository.name;
 const current=await api('GET',address+'/pulls/'+x.pr);
 if(current.state!=='open'||current.merged!==false||current.head?.sha!==x.head||current.number!==x.pr)throw Error();
 const body={state:'success',context:s.required_checks[0],description:'Fresh native approved candidate check '+x.preparation+' '+measured.output_digest.slice(0,16),target_url:REQUEST.origin+'/'+s.repository.owner+'/'+s.repository.name+'/pulls/'+x.pr+'/commits/'+x.head+'?qualification='+s.operation_id};
 const journal='/var/lib/soda-qualification/ci-journal';
 const syncDir=p=>{const d=fs.openSync(p,fs.constants.O_RDONLY|fs.constants.O_DIRECTORY|fs.constants.O_NOFOLLOW);try{fs.fsyncSync(d);}finally{fs.closeSync(d);}};
 if(!fs.existsSync(journal)){fs.mkdirSync(journal,{mode:0o700});syncDir('/var/lib/soda-qualification');}const jm=fs.lstatSync(journal);if(!jm.isDirectory()||jm.isSymbolicLink()||jm.uid!==0||(jm.mode&0o777)!==0o700)throw Error();
 const readJournal=p=>{const m=fs.lstatSync(p);if(!m.isFile()||m.isSymbolicLink()||m.nlink!==1||m.uid!==0||(m.mode&0o777)!==0o600||m.size>65536)throw Error();return fs.readFileSync(p,'utf8');};
 const writeJournal=(p,v)=>{const fd=fs.openSync(p,fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_EXCL|fs.constants.O_NOFOLLOW,0o600);try{fs.writeFileSync(fd,JSON.stringify(v,null,2)+'\n');fs.fsyncSync(fd);}finally{fs.closeSync(fd);}syncDir(journal);};
 const identity=s.operation_id+'-'+x.head,requestPath=journal+'/'+identity+'.request.json',receiptPath=journal+'/'+identity+'.receipt.json';
 const request={operation_id:s.operation_id,repository:R,issue:I,project:P,publication:x.publication,pr:x.pr,head:x.head,actor:s.actors.ci,preparation:x.preparation,measurement:measured,record_times:times,checkout,approved_check_sha256:s.approved_check_sha256,body};
 if(fs.existsSync(requestPath)){
  const previous=JSON.parse(readJournal(requestPath));
  if(JSON.stringify(previous)!==JSON.stringify(request))throw Error();
  // A prior POST is never repeated, including an uncertain response. A saved
  // receipt can be read; otherwise the owner must inspect native status once.
  if(!fs.existsSync(receiptPath)){console.log(JSON.stringify({state:'uncertain',post_repeated:false,request_retained:true}));return;}
  console.log(readJournal(receiptPath));return;
 }
 if(REQUEST.mode!=='publish')throw Error();
 const second=record();validate(second);if(JSON.stringify(second)!==JSON.stringify(first)||JSON.stringify(await inspect())!==JSON.stringify(measured))throw Error();
 writeJournal(requestPath,request);
 issued=true;
 const status=await api('POST',address+'/statuses/'+x.head,body);
 if(status.id<=0||status.status!=='success'||status.context!==s.required_checks[0]||status.creator?.id!==s.actors.ci)throw Error();
 const result={state:'success',status_id:status.id,context:status.context,actor:status.creator.id,head:x.head,project:P,publication:x.publication,pr:x.pr,preparation:x.preparation,measurement:measured,record_times:times,checkout,approved_check_sha256:s.approved_check_sha256,request_sha256:digest(Buffer.from(JSON.stringify(request,null,2)+'\n')),fresh_after_baseline:true};
 writeJournal(receiptPath,result);console.log(JSON.stringify(result));
})().catch(()=>{console.error(JSON.stringify({ci_unconfirmed:true,request_may_be_issued:issued,no_automatic_retry:true}));process.exitCode=1;});
`;

export function ciScript(scope: any, origin: string, checkRequest: any, mode: 'publish' | 'inspect' = 'publish') {
  validateFactoryPhase(scope);
  check(/^https:\/\/[A-Za-z0-9.-]+$/.test(origin), 'ci_public_origin');
  check(
    /^f[a-f0-9]{24}$/.test(checkRequest.preparation) &&
      COMMIT.test(checkRequest.head) &&
      [checkRequest.publication, checkRequest.owner_assignment, checkRequest.attempt_root].every((x: any) =>
        /^[a-f0-9]{32}$/.test(x)
      ) &&
      Number.isSafeInteger(checkRequest.pr) &&
      checkRequest.pr > 0,
    'ci_candidate_identity'
  );
  check(
    Array.isArray(checkRequest.before_preparations) &&
      checkRequest.before_preparations.length <= 256 &&
      checkRequest.before_preparations.every((x: any) => /^f[a-f0-9]{24}$/.test(x)) &&
      Number.isSafeInteger(checkRequest.baseline_unix_ms),
    'ci_fresh_baseline'
  );
  return 'const REQUEST=' + JSON.stringify({scope, origin, check: checkRequest, mode}) + ';\n' + FACTORY_CI;
}
