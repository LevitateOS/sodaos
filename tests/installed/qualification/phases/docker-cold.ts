// Transport for the Rust stopped-storage owner. No copied per-attempt driver,
// caller shell command, automatic retry or interpretation of provider output.
import {check, SHA} from './support';
import type {ResourceScope} from '../closure';
import type {DockerBinding, DockerIO, Witness} from './docker-binding';
import type {ColdCapture, ColdDestination} from './final-state-recovery';
import {nativeReadScript} from './docker-native';

export interface DockerColdPaths {
  readonly operator: {readonly path: string; readonly sha256: string};
  readonly capture_directory: string;
  readonly restore_directory: string;
}
const PRELUDE = String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
const need=x=>{if(!x)throw Error('cold_transport_unconfirmed')};
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const run=(file,args,input,timeout=1800000)=>cp.execFileSync(file,args,{input,timeout,encoding:'utf8',maxBuffer:4<<20,stdio:[input===undefined?'ignore':'pipe','pipe','pipe']});
const read=p=>{const s=fs.lstatSync(p);need(s.isFile()&&!s.isSymbolicLink()&&s.uid===0&&(s.mode&63)===0&&s.nlink===1&&s.size<=512<<20);return fs.readFileSync(p)};
const privateParent=p=>{const path=require('node:path');let current=path.dirname(p),first=true;for(;;){const s=fs.lstatSync(current);need(s.isDirectory()&&!s.isSymbolicLink()&&s.uid===0&&(s.mode&18)===0&&(!first||(s.mode&511)===448));if(current==='/')break;current=path.dirname(current);first=false}};
const pin=()=>{need(process.getuid()===0);for(let p=PATHS.operator.path;;p=require("node:path").dirname(p)){const st=fs.lstatSync(p);need(st.uid===0&&!st.isSymbolicLink()&&(st.mode&18)===0);if(p==="/")break}privateParent(PATHS.capture_directory);privateParent(PATHS.restore_directory);const s=fs.lstatSync(PATHS.operator.path);need(s.isFile()&&!s.isSymbolicLink()&&s.uid===0&&(s.mode&18)===0&&hash(fs.readFileSync(PATHS.operator.path))===PATHS.operator.sha256)};
`;
function paths(value: DockerColdPaths) {
  const home = (v: string) =>
    /^\/home\/[A-Za-z0-9._/-]+$/.test(v) && v.split('/').every((p) => p !== '.' && p !== '..');
  check(
    home(value.capture_directory) &&
      home(value.restore_directory) &&
      value.capture_directory !== value.restore_directory &&
      !value.capture_directory.startsWith(value.restore_directory + '/') &&
      !value.restore_directory.startsWith(value.capture_directory + '/'),
    'fresh_distinct_cold_paths_required'
  );
  check(
    /^\/(usr|home)\/[A-Za-z0-9._/-]+$/.test(value.operator.path) &&
      !value.operator.path.split('/').some((p) => p === '.' || p === '..') &&
      SHA.test(value.operator.sha256),
    'pinned_cold_operator_required'
  );
  return 'const PATHS=' + JSON.stringify(value) + ';\n' + PRELUDE;
}
export async function preflightDockerCold(io: DockerIO, selected: DockerColdPaths) {
  const program =
    paths(selected) +
    '\ntry{pin();need(!fs.existsSync(PATHS.capture_directory)&&!fs.existsSync(PATHS.restore_directory)&&!fs.existsSync(PATHS.capture_directory+".stop-request.json"));console.log(JSON.stringify({format:1,operator_sha256:PATHS.operator.sha256,fresh_destinations:true}));}catch{console.error("cold_tool_or_destinations_unavailable");process.exitCode=1;}';
  const observed = JSON.parse(await io.sshRead(program));
  check(
    observed.operator_sha256 === selected.operator.sha256 && observed.fresh_destinations,
    'cold_executor_preflight_unconfirmed'
  );
  await io.save('supplemental/recovery/tool-preflight.json', observed);
}

export function dockerColdExecutor(
  b: DockerBinding,
  w: Witness,
  scope: ResourceScope,
  io: DockerIO,
  selected: DockerColdPaths
) {
  check(
    b.executionReleased &&
      !scope.scenario.authority.held &&
      scope.scenario.authority.scope_receipt_file !== null &&
      scope.scenario.permitted_effects.cold_restore,
    'cold_execution_held'
  );
  const prefix = paths(selected);
  return {
    async capture(current: any): Promise<ColdCapture> {
      check(
        current?.target === b.target &&
          /^[a-f0-9]{64}$/.test(current.outer_id) &&
          SHA.test(current.inner?.protected_sha256),
        'current_cold_native_binding_required'
      );
      const expected = {outer: current.outer_id, protected: current.inner.protected_sha256};
      await io.save('supplemental/recovery/capture.request.json', {
        format: 1,
        action: 'stop-and-capture-final-state',
        ...expected,
        paths: selected,
        automatic_retry: false,
      });
      const program =
        prefix +
        '\nconst EXPECTED=' +
        JSON.stringify(expected) +
        ',HOST=' +
        JSON.stringify(b.host) +
        ',IMAGE=' +
        JSON.stringify(b.image) +
        ',READER=' +
        JSON.stringify(nativeReadScript(b, 'snapshot', w, scope)) +
        ';\n' +
        String.raw`
try{pin();const before=JSON.parse(run('/usr/bin/node',['-'],READER,210000));need(before.outer_id===EXPECTED.outer&&before.inner.protected_sha256===EXPECTED.protected);
const marker=PATHS.capture_directory+'.stop-request.json',fd=fs.openSync(marker,fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_EXCL|fs.constants.O_NOFOLLOW,384);try{fs.writeFileSync(fd,JSON.stringify({format:1,outer:EXPECTED.outer,protected:EXPECTED.protected,automatic_retry:false}));fs.fsyncSync(fd)}finally{fs.closeSync(fd)}const parent=fs.openSync(require('node:path').dirname(marker),fs.constants.O_RDONLY|fs.constants.O_DIRECTORY);try{fs.fsyncSync(parent)}finally{fs.closeSync(parent)}
run('/usr/bin/docker',['--host','unix:///var/run/docker.sock','compose','--project-name','sodaos','--project-directory',HOST.install_directory,'--env-file',HOST.install_directory+'/.env','-f',HOST.install_directory+'/compose.yaml','stop'],undefined,300000);
process.stdout.write(run(PATHS.operator.path,['journey','cold','capture','--container',EXPECTED.outer,'--project','sodaos','--image',IMAGE,'--settings',HOST.install_directory,'--out',PATHS.capture_directory]));
}catch{console.error('cold_capture_unconfirmed_no_retry');process.exitCode=1;}`;
      try {
        return JSON.parse(await io.sshRead(program, 1800000));
      } catch {
        // Observe the one completed set, never repeat stop/copy after lost SSH.
        return JSON.parse(
          await io.sshRead(
            prefix +
              '\ntry{pin();const marker=JSON.parse(read(PATHS.capture_directory+".stop-request.json")),bytes=read(PATHS.capture_directory+"/complete.json"),seal=JSON.parse(bytes);need(marker.outer===' +
              JSON.stringify(expected.outer) +
              '&&marker.protected===' +
              JSON.stringify(expected.protected) +
              '&&seal.format===1&&seal.source.container===marker.outer);console.log(JSON.stringify({format:1,action:"cold-capture",complete:true,directory:PATHS.capture_directory,container:marker.outer,seal_sha256:hash(bytes),writers_started:false,automatic_retry:false}));}catch{console.error("cold_capture_incomplete_preserve_no_retry");process.exitCode=1;}'
          )
        );
      }
    },
    async restore(capture: ColdCapture): Promise<ColdDestination> {
      check(
        capture.directory === selected.capture_directory && capture.complete && SHA.test(capture.seal_sha256),
        'exact_cold_set_required'
      );
      await io.save('supplemental/recovery/restore.request.json', {
        format: 1,
        source: capture.directory,
        seal_sha256: capture.seal_sha256,
        out: selected.restore_directory,
        automatic_retry: false,
      });
      const program =
        prefix +
        '\ntry{pin();need(hash(read(PATHS.capture_directory+"/complete.json"))===' +
        JSON.stringify(capture.seal_sha256) +
        ');process.stdout.write(run(PATHS.operator.path,["journey","cold","restore","--source",PATHS.capture_directory,"--out",PATHS.restore_directory]));}catch{console.error("cold_restore_unconfirmed_no_retry");process.exitCode=1;}';
      try {
        return JSON.parse(await io.sshRead(program, 1800000));
      } catch {
        return JSON.parse(
          await io.sshRead(
            prefix +
              '\ntry{pin();const request=JSON.parse(read(PATHS.restore_directory+"/request.json")),result=JSON.parse(read(PATHS.restore_directory+"/complete.json"));need(request.source===PATHS.capture_directory&&request.seal_sha256===' +
              JSON.stringify(capture.seal_sha256) +
              '&&result.complete&&result.directory===PATHS.restore_directory&&result.image===' +
              JSON.stringify(b.image) +
              ');console.log(JSON.stringify(result));}catch{console.error("cold_restore_incomplete_preserve_no_retry");process.exitCode=1;}'
          )
        );
      }
    },
  };
}
