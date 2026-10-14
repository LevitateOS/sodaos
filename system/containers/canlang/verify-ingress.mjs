#!/usr/bin/env node
/** Finite private HTTPS smoke: existing pinned Caddy + actual installed Canlang target. */
import assert from 'node:assert/strict';
import {spawn, execFileSync} from 'node:child_process';
import {once} from 'node:events';
import {randomBytes, createHash} from 'node:crypto';
import {copyFile, mkdir, readFile, writeFile} from 'node:fs/promises';
import {createRequire} from 'node:module';
import {dirname, join, resolve} from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
const option=name=>{const i=process.argv.indexOf(name);assert.ok(i>=0&&process.argv[i+1],`${name} required`);return resolve(process.argv[i+1]);};
const state=option('--state'), toolRoot=option('--tool-root'), sourceBundle=option('--bundle');
assert.ok(state.startsWith('/home/'));
const repo=resolve(fileURLToPath(new URL('../../../',import.meta.url)));
const origin='https://board.soda.test:24443', backendPort=22000;
const caddyImage='docker.io/library/caddy:2.10.2@sha256:c3d7ee5d2b11f9dc54f947f68a734c84e9c9666c92c88a7f30b9cba5da182adb';
const commands=[];
await mkdir(join(state,'tls'),{mode:0o700});await mkdir(join(state,'data'),{mode:0o700});await mkdir(join(state,'caddy-data'),{mode:0o700});
process.env.TMPDIR=join(state,'tmp');
const env={...process.env,PATH:join(toolRoot,'bin')+':'+process.env.PATH};
const flags=['--root',join(state,'podman/root'),'--runroot',join(state,'podman/run'),'--tmpdir',join(state,'podman/tmp')];
const podman=args=>{commands.push(['podman',...flags,...args]);return execFileSync('podman',[...flags,...args],{env,encoding:'utf8',maxBuffer:8*1024*1024});};
const require=createRequire(join(toolRoot,'lib/package.json'));
const load=specifier=>import(pathToFileURL(require.resolve(specifier)).href);
const {buildSodaBundle}=await load('@canlang/cloudflare/deploy/soda');
const template=JSON.parse(await readFile(sourceBundle,'utf8'));
const input=buildSodaBundle(template.artifact,template.descriptor,{...template.environment,vars:{...template.environment.vars,CAN_AUTH_ORIGIN:origin}},template.compatibilityDate,template.compatibilityFlags);
await writeFile(join(state,'input.json'),JSON.stringify({...input,bundle:{...input.bundle,binaries:Object.fromEntries(Object.entries(input.bundle.binaries).map(([name,bytes])=>[name,Buffer.from(bytes).toString('base64')]))}}),{mode:0o600});
const credentialPath=join(state,'credentials.json');
const credentials={email:'owner@ingress.can.test',password:'IngressProof!'+randomBytes(24).toString('base64url'),timezone:'UTC'};
await writeFile(credentialPath,JSON.stringify(credentials),{mode:0o600});
const cli=join(toolRoot,'bin/can-soda');
const owner=JSON.parse(execFileSync(cli,['bootstrap','--bundle',join(state,'input.json'),'--data',join(state,'data'),'--credentials',credentialPath],{env,encoding:'utf8'}));assert.equal(owner.ok,true);
execFileSync('openssl',['req','-x509','-newkey','rsa:2048','-sha256','-days','1','-nodes','-subj','/CN=board.soda.test',
 '-addext','subjectAltName=DNS:board.soda.test,DNS:forgejo.soda.test','-keyout',join(state,'tls/key.pem'),'-out',join(state,'tls/cert.pem')],{stdio:'ignore'});
await copyFile(join(repo,'system/host/config/proxy.Caddyfile'),join(state,'Caddyfile'));
const mounts=['--volume',join(state,'Caddyfile')+':/etc/caddy/Caddyfile:ro','--volume',join(state,'tls')+':/etc/soda/tls:ro',
 '--volume',join(state,'routes')+':/home/soda-applications/routes:ro','--volume',join(state,'caddy-data')+':/data:rw'];
const caddyEnv=['--env','FORGEJO_ORIGIN=https://forgejo.soda.test:24443','--env','SODA_BIND=127.0.0.1'];
await writeFile(join(state,'validate.log'),podman(['run','--rm','--network','none','--security-opt','label=disable',...mounts,...caddyEnv,caddyImage,'caddy','validate','--config','/etc/caddy/Caddyfile','--adapter','caddyfile']));
const name=`canlang-ingress-proof-${process.pid}`;
const app=spawn(cli,['serve','--bundle',join(state,'input.json'),'--data',join(state,'data'),'--owner',owner.teamId,'--host','127.0.0.1','--port',String(backendPort)],{env,stdio:['ignore','pipe','pipe']});
let appLog='';app.stdout.on('data',bytes=>appLog+=bytes);app.stderr.on('data',bytes=>appLog+=bytes);
let proxyStarted=false, dispatcher;
try{
 const deadline=Date.now()+30000;
 while(true){assert.equal(app.exitCode,null,appLog);try{if((await fetch(`http://127.0.0.1:${backendPort}/auth/login`)).status===200)break;}catch{}
 assert.ok(Date.now()<deadline,'real Canlang backend readiness timed out');await new Promise(resolve=>setTimeout(resolve,100));}
 podman(['run','--detach','--name',name,'--network','host','--security-opt','label=disable',...mounts,...caddyEnv,caddyImage]);proxyStarted=true;
 const tlsDeadline=Date.now()+10000;let login;
 while(true){try{login=execFileSync('curl',['--silent','--show-error','--noproxy','*','--max-time','2','--resolve','board.soda.test:24443:127.0.0.1',
 '--cacert',join(state,'tls/cert.pem'),origin+'/auth/login'],{encoding:'utf8',stdio:['ignore','pipe','pipe']});if(JSON.parse(login).preSessionToken)break;}catch{}
 assert.ok(Date.now()<tlsDeadline,'Caddy private HTTPS readiness timed out');await new Promise(resolve=>setTimeout(resolve,100));}
 const {Agent,setGlobalDispatcher}=await load('undici');
 dispatcher=new Agent({connect:{ca:await readFile(join(state,'tls/cert.pem')),lookup:(host,options,callback)=>callback(null,options.all?[{address:'127.0.0.1',family:4}]:'127.0.0.1',4)}});setGlobalDispatcher(dispatcher);
 const {openSodaHttpClient}=await import(pathToFileURL(join(dirname(repo),'canlang/scripts/verify-soda-runtime.mjs')).href);
 const client=await openSodaHttpClient({toolRoot,origin,actors:[{label:'Ava',...credentials}]},'Ava');
 const created=await client.mutate('Entry.create',{label:'Private Caddy route'});assert.equal(created.records.length,1);
 await client.upload(created.records[0]);
 const file=await client.request('/files/read/SodaRuntime.Entry/'+created.records[0].id+'/attachment');assert.equal(file.status,200);
 assert.deepEqual(Buffer.from(await file.arrayBuffer()),Buffer.from('%PDF-1.4\ntrailer\n'));
 assert.equal((await client.request('/_can_dev/status')).status,404);
 const results={scope:'Local native Caddy supplied-TLS private-host smoke; no installed appliance activation',image:caddyImage,
 imageId:podman(['image','inspect','--format','{{.Id}}',caddyImage]).trim(),origin,backendPort,configValidated:true,certificateAndHostnameVerified:true,
 authenticatedHttpCrud:true,mcpDiscovery:true,canonicalFiles:true,developmentRouteAbsent:true,
 routeSha256:createHash('sha256').update(await readFile(join(state,'routes','a'+'1'.repeat(24)+'.caddy'))).digest('hex')};
 await writeFile(join(state,'results.json'),JSON.stringify(results,null,2));console.log(JSON.stringify(results,null,2));
}finally{
 if(dispatcher)await dispatcher.close();
 if(proxyStarted){await writeFile(join(state,'caddy.log'),podman(['logs',name]));podman(['stop','--time','10',name]);podman(['rm',name]);}
 if(app.exitCode===null){app.kill('SIGTERM');const [code]=await once(app,'exit');assert.equal(code,0,'Canlang graceful shutdown');}
 await writeFile(join(state,'app.log'),appLog);await writeFile(join(state,'commands.json'),JSON.stringify(commands,null,2));
}
