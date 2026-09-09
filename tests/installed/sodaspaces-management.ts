// SPDX-License-Identifier: Apache-2.0
// Explicit existing-project E2E extension of sodaspaces.ts, not a second runner.
import assert from 'node:assert/strict';
import {lstat} from 'node:fs/promises';
import type {Page, Locator} from 'playwright';
import {projectView, newManagedTerminal} from './sodaspaces-controls.ts';
import {object, type JourneyInput, type ManagementRequest} from './sodaspaces-input.ts';
import {savedKeysResponse, keyPreviewResponse, type SavedKey} from '../../frontend/spaces/sodaspaces-api.ts';
export interface ManagementEvidence extends Record<string, unknown> {
  last_ssh_failure?: {user: string; exit: number; kind: 'authentication' | 'transport' | 'command'};
  observed_addresses?: string[];
}
class SSHFailure extends Error {
  constructor(public status: number, public stderr: string) {super('Management SSH failed');}
}

export async function exerciseManagement({page, input, request, authenticate, settled, permit, stage, evidence}: {
  page: Page; input: JourneyInput; request: ManagementRequest;
  authenticate: (index: number) => Promise<void>; settled: () => Promise<void>;
  permit: (index: number, route: string, body: unknown, method?: string) => void;
  stage: (value: string) => void; evidence: ManagementEvidence;
}) {
  assert.deepEqual(Object.keys(request).sort(), ['cid','key_a','key_a_public','key_b','key_b_public','original_alice','original_bob','project','ssh_config','target']);
  assert.equal(request.target, input.target);
  assert.match(request.project, /^p[0-9a-f]{24}$/);
  assert.match(request.cid, /^[0-9a-f]{64}$/);
  assert.deepEqual(input.users.map(u => u.login), ['alice','bob']);
  for (const name of ['ssh_config','key_a','key_b','key_a_public','key_b_public','original_alice','original_bob'] as const) {
    assert(request[name].startsWith('/'));
    const info = await lstat(request[name]);
    assert(info.isFile() && !(info.mode & 0o077) && info.size <= 16384);
  }
  const keys = await Promise.all((['key_a_public','key_b_public'] as const).map(async n => {
    const key = (await Bun.file(request[n]).text()).trim();
    assert.match(key, /^ssh-ed25519 [A-Za-z0-9+/]{68}$/);
    return key;
  }));
  assert(keys[0] && keys[1]);
  assert.notEqual(keys[0], keys[1]);
  let projectIP: string | undefined;
  const sshArgs = (user: string, key: string | null) => ['-F', request.ssh_config, ...(user==='host'?[]:['-o','HostName='+projectIP]), '-T', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', '-o', 'ControlMaster=no', '-o', 'ControlPath=none', '-o', 'IdentityAgent=none', '-o', 'IdentitiesOnly=yes', '-o', 'PreferredAuthentications=publickey', ...(key ? ['-i', key] : []), 'soda-e2e-' + user];
  function ssh(user: string, key: string | null, program: string) {
    try {
      const child = Bun.spawnSync(['ssh', ...sshArgs(user,key), 'python3', '-'], {stdin: Buffer.from(program), stdout: 'pipe', stderr: 'pipe', timeout:30000, maxBuffer:65536});
      if (child.exitCode !== 0) throw new SSHFailure(child.exitCode, child.stderr.toString());
      return child.stdout.toString();
    } catch(error) {
      if (!(error instanceof SSHFailure)) throw error;
      evidence.last_ssh_failure={user,exit:error.status,kind:/Permission denied \(publickey/.test(String(error.stderr))?'authentication':/Connection refused|Connection closed|Connection reset|kex_exchange_identification/.test(String(error.stderr))?'transport':'command'};
      throw error;
    }
  }
  // Pinned management SSH is observation only. All lifecycle/key writes use UI/API.
  assert.equal(ssh('host', null, "import socket; print(socket.gethostname())\n").trim(), input.target);
  function snapshot(running: boolean) {
    const code = `import subprocess,json,hashlib\nfrom pathlib import Path\np=${JSON.stringify(request.project)}\ncid=subprocess.check_output(['podman','--remote=false','inspect','--format','{{.ID}}','soda-'+p],text=True).strip()\nassert cid==${JSON.stringify(request.cid)}\nr=subprocess.check_output(['podman','--remote=false','inspect','--format','{{.State.Running}}',cid],text=True).strip()\nboot=subprocess.check_output(['systemctl','show','soda-project@'+p+'.service','--property=UnitFileState','--value'],text=True).strip()\nassert r==${JSON.stringify(running ? 'true':'false')} and boot==${JSON.stringify(running?'enabled':'disabled')}\nresult={'cid':cid,'image':subprocess.check_output(['podman','--remote=false','inspect','--format','{{.Image}}',cid],text=True).strip()}\n`;
    const guest = "import json,hashlib,os; from pathlib import Path; names=['/etc/passwd','/etc/group','/etc/ssh/ssh_host_ed25519_key.pub','/etc/ssh/authorized_keys/alice','/etc/ssh/authorized_keys/bob','/var/lib/soda/accounts/alice','/var/lib/soda/accounts/bob']; print(json.dumps({n:[hashlib.sha256(Path(n).read_bytes()).hexdigest(),Path(n).stat().st_uid,Path(n).stat().st_gid,Path(n).stat().st_mode] for n in names},sort_keys=True))";
    const observed=object(JSON.parse(ssh('host', null, code + (running ? `result['files']=json.loads(subprocess.check_output(['podman','--remote=false','exec',cid,'python3','-I','-c',${JSON.stringify(guest)}],text=True))\nresult['ip']=subprocess.check_output(['podman','--remote=false','inspect','--format','{{(index .NetworkSettings.Networks "soda-projects").IPAddress}}',cid],text=True).strip()\n` : '') + 'print(json.dumps(result))\n')));
    assert(typeof observed.cid === 'string' && typeof observed.image === 'string');
    if(running) {
      assert(typeof observed.ip === 'string');
      object(observed.files);
      assert.match(observed.ip,/^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/);
      assert(observed.ip.split('.').every(n=>Number(n)<=255));
      projectIP=observed.ip;
      (evidence.observed_addresses ??= []).push(projectIP);
      delete observed.ip; // Mutable current endpoint, not persistent root identity.
    }
    return observed;
  }
  async function action(route: string, body: unknown, control: Locator, method='POST') {
    permit(0, route, body, method);
    const pending = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda'+route && r.request().method()===method);
    await control.click();
    const response = await pending;
    assert.equal(response.status(),200);
    const value = object(await response.json());
    await settled();
    return value;
  }
  const control = (name: string) => page.locator(`[data-project-controls][data-repository-id="${input.repository_id}"] [data-control="${name}"]`);
  const life = `/api/environments/${request.project}/lifecycle`;
  const access = `/api/environments/${request.project}/access-keys`;
  async function apply() {
    const pending = page.waitForResponse(r => new URL(r.url()).pathname === '/-/soda'+access && r.request().method()==='GET');
    await page.getByRole('button',{name:'Review this project’s SSH keys',exact:true}).click();
    const response = await pending; assert.equal(response.status(),200);
    const preview = keyPreviewResponse(await response.json(), input.users[0].login);
    assert(preview.saved_fingerprints.length > 0); // Never remove the original key.
    const changed = await action(access, {revision:preview.revision,saved_fingerprints:preview.saved_fingerprints,confirm_empty:false}, page.getByRole('button',{name:'Apply reviewed saved keys to this project',exact:true}));
    assert.equal(changed.applied,true);
  }
  async function save(key: string) {
    await projectView(page, input.repository_id, 'Access');
    await control('public-key').fill(key);
    const saved = await action('/api/me/development-keys',{public_key:key},control('save-key'));
    const entry = savedKeysResponse(saved).find(k => k.public_key?.trim()===key);
    assert(entry && /^[1-9][0-9]*$/.test(entry.id));
    return entry;
  }
  async function remove(entry: SavedKey) {
    const button = control('key-list').locator('li').filter({hasText:entry.fingerprint}).getByRole('button',{name:'Remove saved key',exact:true});
    const response = await action('/api/me/development-keys/'+entry.id,{},button,'DELETE');
    assert.equal(response.existing_project_access_changed,false);
  }
  function login(user: string,key: string) {
    assert.equal(ssh(user,key,'import pwd,os; print(pwd.getpwuid(os.getuid()).pw_name)\n').trim(),user);
  }
  function refused(key: string) {
    let denial;
    try { ssh('alice',key,"raise RuntimeError('revoked key authenticated')\n"); }
    catch (error) { denial=error; }
    assert(denial instanceof SSHFailure && denial.status===255 && /Permission denied \(publickey/.test(String(denial.stderr)), 'Expected authentication refusal, not a network failure');
  }
  const held: Bun.Subprocess<'pipe', 'pipe', 'pipe'>[]=[];
  async function hold(user: string,key: string) {
    const program="import os,sys; print(os.getpid(),flush=True); [(print(os.getpid(),flush=True)) for line in sys.stdin]";
    const child=Bun.spawn(['ssh',...sshArgs(user,key),"exec python3 -u -c '"+program+"'"],{stdin:'pipe',stdout:'pipe',stderr:'pipe'});
    const lines: string[]=[]; let buffer='';
    const output = (async () => {
      for await (const chunk of child.stdout) {
        buffer+=new TextDecoder().decode(chunk); const split=buffer.split('\n'); buffer=split.pop() ?? ''; lines.push(...split);
      }
    })();
    const errors = (async () => {for await (const _ of child.stderr) { /* Discard private SSH diagnostics. */ }})();
    child.exited.then(() => Promise.all([output, errors])).catch(() => {});
    async function next() {
      const until=Date.now()+15000;
      while (!lines.length && child.exitCode===null && Date.now()<until) await new Promise(r=>setTimeout(r,50));
      const line = lines.shift(); assert(line && /^\d+$/.test(line)); return line;
    }
    const first=await next();
    const check=async()=>{child.stdin.write('check\n'); await child.stdin.flush(); assert.equal(await next(),first);};
    held.push(child); return check;
  }
  const result=evidence;
  stage('management: owner authentication and stable baseline');
  await authenticate(0);
  const before=snapshot(true);
  result.before=before;
  const controls = await projectView(page, input.repository_id, 'Access');
  assert.equal(await controls.getAttribute('data-environment-id'), request.project);
  assert.equal(await control('command').inputValue(),'ssh alice@'+projectIP);
  // Persist a new exact run-owned home marker; never alter existing files.
  const name='.soda-e2e-'+Date.now();
  const create=`from pathlib import Path\np=Path.home()/${JSON.stringify(name)}\np.mkdir(mode=0o700)\n(p/'marker').write_text('persistent E2E marker\\n')\nprint(p.name)\n`;
  assert.equal(ssh('alice',request.original_alice,create).trim(),name);
  stage('management: Stop interrupts the explicitly opened browser terminal');
  const opened=page.waitForEvent('websocket');
  assert(input.terminal_actions?.includes('create'));
  await newManagedTerminal(page, input.repository_path.slice(1), 'Management stop observation', request.project);
  const socket=await opened; let terminalClosed=false;
  socket.on('close',()=>{terminalClosed=true;});
  stage('management: explicit Stop and disabled host-boot start');
  await projectView(page, input.repository_id);
  await page.getByLabel(/I understand Stop interrupts/).check();
  await action(life,{action:'stop',confirm_stop:true},page.getByRole('button',{name:'Stop',exact:true}));
  result.stopped=snapshot(false);
  const untilClosed=Date.now()+10000;
  while(!terminalClosed && Date.now()<untilClosed) await new Promise(r=>setTimeout(r,50));
  assert(terminalClosed);
  stage('management: explicit same-container Start and persistence');
  await action(life,{action:'start'},page.getByRole('button',{name:'Start',exact:true}));
  stage('management: stable state after Start');
  result.started=snapshot(true);
  assert.deepEqual(result.started,before);
  await projectView(page, input.repository_id, 'Access');
  assert.equal(await control('command').inputValue(),'ssh alice@'+projectIP);
  stage('management: SSH availability and home persistence after Start');
  const readyUntil=Date.now()+30000;
  for (;;) {
    try {login('alice',request.original_alice); break;}
    catch(error) {
      if(result.last_ssh_failure?.kind!=='transport' || Date.now()>=readyUntil) throw error;
      await new Promise(r=>setTimeout(r,500)); // Read/auth observation only; never retry Start.
    }
  }
  const read=`from pathlib import Path\nassert (Path.home()/${JSON.stringify(name)}/'marker').read_text()=='persistent E2E marker\\n'\nprint('preserved')\n`;
  assert.equal(ssh('alice',request.original_alice,read).trim(),'preserved');
  result.lifecycle={same_container:true,boot_policy:true,accounts_keys_host_key_preserved:true,terminal_interrupted:true,home_marker:name};
  stage('management: nonadministrator has no lifecycle control');
  await authenticate(1);
  await projectView(page, input.repository_id);
  assert(await page.getByRole('button',{name:'Stop',exact:true}).isHidden());
  permit(1,life,{action:'stop',confirm_stop:true});
  const denied = await page.evaluate(async ({actor,route}) => {
    const s: unknown=await (await fetch('/-/soda/api/session',{credentials:'same-origin',cache:'no-store',redirect:'error'})).json();
    if(!s || typeof s!=='object' || !('user' in s) || !s.user || typeof s.user!=='object' || !('id' in s.user) || !('csrf_token' in s) || typeof s.csrf_token!=='string') throw Error('Invalid session');
    if(s.user.id!==actor) throw new Error('actor mismatch');
    const r=await fetch('/-/soda'+route,{method:'POST',credentials:'same-origin',redirect:'error',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf_token,'X-Soda-Expected-User-ID':actor},body:JSON.stringify({action:'stop',confirm_stop:true})});
    return r.status;
  },{actor:input.users[1].id,route:life});
  assert.equal(denied,403);
  assert.deepEqual(snapshot(true),before);
  login('bob',request.original_bob);
  await authenticate(0);
  try {
    stage('management: add temporary key A and explicit native Apply');
    const a=await save(keys[0]); await apply(); login('alice',request.key_a);
    const checkA=await hold('alice',request.key_a), checkBob=await hold('bob',request.original_bob);
    stage('management: replacement key B authenticates');
    const b=await save(keys[1]); await apply(); login('alice',request.key_b);
    stage('management: remove saved A does not implicitly revoke native access');
    await remove(a); login('alice',request.key_a);
    stage('management: explicit revocation refuses new A authentication');
    await apply(); login('alice',request.key_b); refused(request.key_a);
    await checkA(); await checkBob(); login('alice',request.original_alice);
    stage('management: remove only temporary B and restore original managed key set');
    await remove(b); await apply(); refused(request.key_b);
    login('alice',request.original_alice); login('bob',request.original_bob);
    assert.deepEqual(snapshot(true),before);
    result.keys={replacement_login:true,removed_key_authentication_refused:true,saved_removal_not_revocation:true,existing_sessions_survived:true,original_key_set_restored:true};
  } finally {
    // Close only run-owned SSH sessions; never roll back/replay failed key changes.
    for(const child of held) child.stdin.end();
    for(const child of held) {
      const until=Date.now()+10000;
      while(child.exitCode===null && Date.now()<until) await new Promise(r=>setTimeout(r,50));
      if(child.exitCode===null) child.kill('SIGTERM');
    }
  }
  return result;
}
