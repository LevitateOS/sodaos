// Private request bytes travel over the existing protected SSH stdin and curl
// config stdin. Only the caller receives responses; this module saves no HTTP
// bodies, cookies or credentials to the qualification evidence.
import assert from 'node:assert/strict';
import {isIP} from 'node:net';
import type {ApplicationHttpRequest, ApplicationHttpResponse} from '../application-owner';
import type {DockerIO} from './docker-binding';

export interface ContainerHttpTarget {
  readonly container: string;
  readonly image: string;
  readonly restore_directory?: string;
}
export interface ContainerBytesResponse {
  readonly status: number;
  readonly body_base64: string;
  /** Ordinary response headers; Set-Cookie is preserved separately. */
  readonly headers: Record<string, string>;
  readonly set_cookie: readonly string[];
}

function validateRequest(origin: string, address: string, request: ApplicationHttpRequest, local: boolean) {
  const url = new URL(origin);
  assert(
    url.origin === origin &&
      !url.username &&
      !url.password &&
      (url.protocol === 'https:' || (local && url.protocol === 'http:' && url.hostname === '127.0.0.1')),
    'namespace_origin_invalid'
  );
  assert(isIP(address) !== 0, 'namespace_address_invalid');
  assert(
    request &&
      ['GET', 'POST'].includes(request.method) &&
      request.route.startsWith('/') &&
      !request.route.startsWith('//') &&
      !request.route.includes('#') &&
      !/[\r\n\0]/.test(request.route) &&
      new URL(request.route, origin).origin === origin &&
      (request.body === null || (typeof request.body === 'string' && Buffer.byteLength(request.body) <= 2 << 20)) &&
      request.headers &&
      typeof request.headers === 'object' &&
      !Array.isArray(request.headers) &&
      Object.keys(request.headers).length <= 64,
    'namespace_request_invalid'
  );
  for (const [name, value] of Object.entries(request.headers)) {
    assert(
      /^[!#$%&'*+.^_`|~0-9A-Za-z-]{1,128}$/.test(name) &&
        typeof value === 'string' &&
        Buffer.byteLength(value) <= 16384 &&
        !/[\r\n\0]/.test(value) &&
        (name.toLowerCase() !== 'host' || value === url.host),
      'namespace_header_invalid'
    );
  }
  return url;
}

// This is also the native local integration seam: a real loopback App can use
// its normal HTTP origin without Docker or an invented response protocol.
export function nativeHttpScript(
  origin: string,
  address: string,
  request: ApplicationHttpRequest,
  caFile = '/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt'
) {
  const url = validateRequest(origin, address, request, true);
  assert(caFile.startsWith('/') && !/[\r\n\0]/.test(caFile), 'namespace_ca_invalid');
  const input = {
    origin,
    hostname: url.hostname,
    port: url.port || (url.protocol === 'https:' ? '443' : '80'),
    tls: url.protocol === 'https:',
    address,
    request,
    ca: caFile,
  };
  return 'const INPUT=' + JSON.stringify(input) + ';\n' + NATIVE_HTTP;
}

const NATIVE_HTTP = String.raw`
const cp=require('node:child_process');
const need=(ok)=>{if(!ok)throw Error('namespace_http_unconfirmed')};
try{
 const quote=value=>JSON.stringify(value);
 const r=INPUT.request;
 let config='silent\nshow-error\nhttp1.1\ninclude\nmax-time = 20\nconnect-timeout = 10\nmax-filesize = 2097152\nnoproxy = "*"\n';
 config+='proto = '+quote(INPUT.tls?'=https':'=http')+'\n';
 config+='request = '+quote(r.method)+'\nurl = '+quote(INPUT.origin+r.route)+'\n';
 const address=INPUT.address.includes(':')?'['+INPUT.address+']':INPUT.address;
 config+='resolve = '+quote(INPUT.hostname+':'+INPUT.port+':'+address)+'\n';
 if(INPUT.tls)config+='cacert = '+quote(INPUT.ca)+'\n';
 config+='header = "Expect:"\n';
 for(const [name,value]of Object.entries(r.headers))config+='header = '+quote(name+': '+value)+'\n';
 if(r.body!==null)config+='data-binary = '+quote(r.body)+'\n';
 const raw=cp.execFileSync('/usr/bin/curl',['--disable','--config','-'],{input:config,timeout:22000,maxBuffer:(2<<20)+(64<<10),stdio:['pipe','pipe','pipe'],env:{PATH:'/usr/sbin:/usr/bin:/sbin:/bin',LC_ALL:'C'}});
 let offset=0,status=0,headers={},cookies=[];
 for(let pass=0;pass<8;pass++){
  const end=raw.indexOf('\r\n\r\n',offset);need(end>=offset&&end-offset<=65536);
  const lines=raw.subarray(offset,end).toString('latin1').split('\r\n');
  const first=/^HTTP\/1\.[01] ([1-5][0-9]{2})(?: |$)/.exec(lines.shift()||'');need(first);
  status=Number(first[1]);offset=end+4;
  if(status>=100&&status<200){need(status!==101);continue;}
  const omit=new Set(['connection','keep-alive','proxy-authenticate','proxy-authorization','te','trailer','transfer-encoding','upgrade']);
  for(const line of lines){
   const at=line.indexOf(':');need(at>0&&!/^[ \t]/.test(line));
   const name=line.slice(0,at).toLowerCase(),value=line.slice(at+1).trim();
   need(/^[!#$%&'*+.^_\u0060|~0-9a-z-]+$/.test(name)&&!/[\r\n\0]/.test(value));
   if(name==='set-cookie'){need(cookies.length<64);cookies.push(value);}
   else if(!omit.has(name))headers[name]=headers[name]===undefined?value:headers[name]+', '+value;
  }
  break;
 }
 need(status>=200&&status<=599);
 const body=raw.subarray(offset);need(body.length<=2<<20);
 console.log(JSON.stringify({status,body_base64:body.toString('base64'),headers,set_cookie:cookies}));
}catch{console.error('namespace_http_unconfirmed');process.exitCode=1;}
`;

export function containerNativeScript(target: ContainerHttpTarget, native: string) {
  assert(/^[a-f0-9]{64}$/.test(target.container), 'namespace_container_invalid');
  assert(/^(?:sha256:[a-f0-9]{64}|[^\s@]+@sha256:[a-f0-9]{64})$/.test(target.image), 'namespace_image_invalid');
  if (target.restore_directory !== undefined)
    assert(
      /^\/(?:home|var\/home)\/[A-Za-z0-9_./-]+$/.test(target.restore_directory) &&
        !target.restore_directory.endsWith('/') &&
        target.restore_directory.split('/').every((part) => part !== '.' && part !== '..'),
      'namespace_restore_directory_invalid'
    );
  return (
    'const TARGET=' +
    JSON.stringify(target) +
    ';\nconst INNER=' +
    JSON.stringify(native) +
    ';\n' +
    String.raw`
const fs=require('node:fs'),cp=require('node:child_process');
const need=ok=>{if(!ok)throw Error('namespace_container_unconfirmed')};
try{
 need(process.getuid()===0);
 const run=(args,input,maxBuffer=4<<20)=>cp.execFileSync('/usr/bin/docker',['--host','unix:///var/run/docker.sock',...args],{input,timeout:30000,maxBuffer,stdio:[input===undefined?'ignore':'pipe','pipe','pipe'],env:{PATH:'/usr/sbin:/usr/bin:/sbin:/bin',LC_ALL:'C'}});
 const records=JSON.parse(run(['container','inspect',TARGET.container]));need(records.length===1);
 const c=records[0];need(c.Id===TARGET.container&&c.Config.Image===TARGET.image&&c.State.Running&&!c.State.Paused);
 const image=run(['image','inspect','--format','{{.Id}}',TARGET.image]).toString('utf8').trim();need(c.Image===image);
 if(TARGET.restore_directory===undefined)need(c.Name==='/sodaos-soda-1');
 else{
  const root=TARGET.restore_directory,meta=fs.lstatSync(root);need(meta.isDirectory()&&!meta.isSymbolicLink()&&meta.uid===0&&(meta.mode&511)===448&&fs.realpathSync(root)===root);
  need(c.HostConfig.NetworkMode==='none'&&c.HostConfig.Privileged===true&&c.HostConfig.CgroupnsMode==='private'&&c.HostConfig.PidMode!=='host'&&c.HostConfig.IpcMode!=='host');
  need(c.Mounts.length===3&&['etc','home','var'].every(name=>c.Mounts.some(m=>m.Type==='bind'&&m.Destination==='/'+name&&m.Source===root+'/'+name&&m.RW===true)));
 }
 const response=run(['exec','-i',TARGET.container,'/usr/bin/node','-'],INNER);
 process.stdout.write(response);
}catch{console.error('namespace_container_unconfirmed');process.exitCode=1;}
`
  );
}

export async function requestContainerBytes(
  io: Pick<DockerIO, 'sshRead'>,
  target: ContainerHttpTarget,
  origin: string,
  address: string,
  request: ApplicationHttpRequest
): Promise<ContainerBytesResponse> {
  validateRequest(origin, address, request, false);
  const response: ContainerBytesResponse = JSON.parse(
    await io.sshRead(containerNativeScript(target, nativeHttpScript(origin, address, request)), 40000)
  );
  assert(
    Number.isSafeInteger(response.status) &&
      response.status >= 200 &&
      response.status <= 599 &&
      typeof response.body_base64 === 'string' &&
      /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(response.body_base64) &&
      Buffer.byteLength(response.body_base64, 'base64') <= 2 << 20 &&
      response.headers &&
      typeof response.headers === 'object' &&
      !Array.isArray(response.headers) &&
      Object.keys(response.headers).length <= 256 &&
      Object.values(response.headers).every((value) => typeof value === 'string') &&
      Array.isArray(response.set_cookie) &&
      response.set_cookie.length <= 64 &&
      response.set_cookie.every((value) => typeof value === 'string' && !/[\r\n\0]/.test(value)),
    'namespace_response_invalid'
  );
  return response;
}

export async function requestContainerHttp(
  io: Pick<DockerIO, 'sshRead'>,
  target: ContainerHttpTarget,
  origin: string,
  address: string,
  request: ApplicationHttpRequest
): Promise<ApplicationHttpResponse> {
  const response = await requestContainerBytes(io, target, origin, address, request);
  return {
    status: response.status,
    text: Buffer.from(response.body_base64, 'base64').toString('utf8'),
    set_cookie: response.set_cookie,
  };
}
