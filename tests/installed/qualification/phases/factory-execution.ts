import {check} from './support';

/** Selected native state only. The fixed host Inspect response is projected
 * inside the guest: output, reason and verification prose never leave it. */
export function nativeExecutionScript(c: any, runID: string) {
  check(
    Number.isSafeInteger(c.repository?.id) &&
      c.repository.id > 0 &&
      /^p[a-f0-9]{24}$/.test(c.repository.project) &&
      Number.isSafeInteger(c.issue?.number) &&
      c.issue.number > 0 &&
      /^[a-f0-9]{32}$/.test(runID),
    'native_execution_scope'
  );
  return (
    'const REQUEST=' +
    JSON.stringify({repository: c.repository.id, issue: c.issue.number, project: c.repository.project, run: runID}) +
    ';\n' +
    String.raw`
const cp=require('node:child_process'),http=require('node:http');
const run=(p,a)=>cp.execFileSync(p,a,{encoding:'utf8',timeout:20000,maxBuffer:1<<20,stdio:['ignore','pipe','pipe']}).trim();
const sql=q=>JSON.parse(run('/usr/bin/podman',['--remote=false','exec','soda-postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-U','postgres','-d','soda','-c','BEGIN READ ONLY;'+q+';COMMIT;']));
(async()=>{
 if(process.getuid()!==0)throw Error();
 const row=sql("SELECT json_build_object('count',count(*)) FROM factory_assignments WHERE repository="+REQUEST.repository+" AND issue="+REQUEST.issue+" AND data->>'project_id'='"+REQUEST.project+"' AND run='"+REQUEST.run+"'");if(row.count!==1)throw Error();
 const body=Buffer.from(JSON.stringify({project:REQUEST.project,id:REQUEST.run}));
 const native=await new Promise((resolve,reject)=>{const request=http.request({socketPath:'/run/soda/host/host.sock',path:'/factory-inspect',method:'POST',headers:{'Content-Type':'application/json','Content-Length':body.length}},response=>{let parts=[],size=0;response.on('data',part=>{size+=part.length;if(size>4<<20){request.destroy();reject(Error());}else parts.push(part)});response.on('end',()=>{try{if(response.statusCode===404){resolve(null);return;}if(response.statusCode!==200)throw Error();resolve(JSON.parse(Buffer.concat(parts).toString()));}catch{reject(Error())}});});request.on('error',()=>reject(Error()));request.setTimeout(20000,()=>request.destroy());request.end(body);});
 if(native===null){console.log(JSON.stringify({available:false,run:REQUEST.run,project:REQUEST.project}));return;}
 if(native.id!==REQUEST.run||native.project!==REQUEST.project)throw Error();
 const raw=run('/usr/bin/systemctl',['show','soda-factory-'+REQUEST.run+'.service','-p','LoadState','-p','ActiveState','-p','MainPID','-p','ControlPID']);
 const unit=Object.fromEntries(raw.split('\n').map(line=>{const at=line.indexOf('=');return[line.slice(0,at),line.slice(at+1)]}));
 if(!/^(0|[1-9][0-9]*)$/.test(unit.MainPID)||!/^(0|[1-9][0-9]*)$/.test(unit.ControlPID))throw Error();
 console.log(JSON.stringify({available:true,run:native.id,project:native.project,role:native.role,phase:native.phase,exit_code:native.exit_code??null,live:native.live,delivered:native.delivered,retirement:native.retirement??null,credential_returned:native.credential_returned,generation:native.generation,lease_id:native.lease_id??null,container:native.container,unit:native.unit,invocation:native.invocation,native_unit:{load_state:unit.LoadState,active_state:unit.ActiveState,main_pid:Number(unit.MainPID),control_pid:Number(unit.ControlPID)},observed_at:new Date().toISOString()}));
})().catch(()=>{console.error('native_execution_metadata_unavailable');process.exitCode=1;});
`
  );
}

/** Retired runs bind their recorded credential generation. A later Ready
 * generation is expected after credential return and is not an enrollment need. */
export function nativeExecutionSucceeded(n: any, r: any, a: any, container: string): boolean {
  return (
    n?.available === true &&
    n.run === r.id &&
    n.project === a.project &&
    n.role === a.role &&
    n.phase === 'completed' &&
    n.exit_code === 0 &&
    n.live === false &&
    n.delivered === true &&
    n.retirement === 'confirmed' &&
    n.credential_returned === true &&
    Number.isSafeInteger(n.generation) &&
    n.generation > 0 &&
    n.generation === r.generation &&
    n.lease_id === r.lease &&
    n.container === container &&
    n.unit === 'soda-factory-' + r.id + '.service' &&
    n.invocation === r.binding?.invocation_id &&
    ['inactive', 'failed'].includes(n.native_unit?.active_state) &&
    n.native_unit.main_pid === 0 &&
    n.native_unit.control_pid === 0
  );
}
