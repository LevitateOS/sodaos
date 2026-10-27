import {check, COMMIT, SHA} from './support';
import {dirname, isAbsolute, normalize} from 'node:path';
import {factoryCommandID} from './factory-native';

// Source-only template. Null helper/scope pins always refuse before guest use.
export function contextStatusScript(c: any, runID: string, assignmentID: string, metadata: any) {
  check(
    Number.isSafeInteger(c?.repository?.id) &&
      c.repository.id > 0 &&
      /^p[a-f0-9]{24}$/.test(c.repository.project) &&
      COMMIT.test(c.repository.source_commit) &&
      Number.isSafeInteger(c.issue?.number) &&
      c.issue.number > 0 &&
      /^d[a-f0-9]{24}$/.test(c.issue.acceptance) &&
      /^[a-f0-9]{32}$/.test(c.connection?.id),
    'context_status_actual_scope'
  );
  check(/^[a-f0-9]{32}$/.test(runID) && /^[a-f0-9]{32}$/.test(assignmentID), 'context_status_assignment');
  const safePath = (path: unknown): path is string =>
    typeof path === 'string' &&
    isAbsolute(path) &&
    normalize(path) === path &&
    path.startsWith('/home/') &&
    path.length < 4096 &&
    !/[\u0000-\u001f\u007f]/.test(path);
  check(
    COMMIT.test(metadata?.source_revision) &&
      COMMIT.test(metadata?.installed_source_revision) &&
      SHA.test(metadata?.payload_sha256) &&
      safePath(metadata?.helper?.path) &&
      SHA.test(metadata.helper.SHA256) &&
      Number.isSafeInteger(metadata.helper.Bytes) &&
      metadata.helper.Bytes > 0 &&
      metadata.helper.Bytes < 64 << 20 &&
      safePath(metadata?.scope?.path) &&
      SHA.test(metadata.scope.SHA256) &&
      metadata.helper.path !== metadata.scope.path &&
      dirname(metadata.helper.path) === dirname(metadata.scope.path),
    'context_status_admitted_helper_scope'
  );
  return (
    'const REQUEST=' +
    JSON.stringify({run_id: runID, assignment_id: assignmentID}) +
    ';const EXPECTED=' +
    JSON.stringify({
      source: metadata.source_revision,
      installed_source: metadata.installed_source_revision,
      payload_sha256: metadata.payload_sha256,
      policy_command_id: factoryCommandID(c.operation_id, 'open-policy'),
      preparation: c.coder_preparation,
      reviewer_preparation: c.reviewer_preparation,
      helper: metadata.helper,
      scope: metadata.scope,
      repository: c.repository.id,
      issue: c.issue.number,
      project: c.repository.project,
      acceptance: c.issue.acceptance,
      source_commit: c.repository.source_commit,
      connection: c.connection.id,
      owner: c.connection.owner_id,
      operation: c.operation_id,
    }) +
    ';\n' +
    String.raw`
const fs=require('node:fs'),cp=require('node:child_process'),crypto=require('node:crypto');
try{
 if(process.getuid()!==0)throw Error();
 const sha=b=>crypto.createHash('sha256').update(b).digest('hex');
 const path=require('node:path'),directory=path.dirname(EXPECTED.helper.path);if(directory!==path.dirname(EXPECTED.scope.path))throw Error();
 for(let p=directory;;p=path.dirname(p)){const dir=fs.lstatSync(p);if(!dir.isDirectory()||dir.isSymbolicLink()||dir.uid!==0||dir.gid!==0||(dir.mode&18)!==0||(p===directory&&(dir.mode&511)!==448))throw Error();if(p==='/')break;}
 const scopeStat=fs.lstatSync(EXPECTED.scope.path);if(!scopeStat.isFile()||scopeStat.isSymbolicLink()||scopeStat.uid!==0||scopeStat.gid!==0||scopeStat.nlink!==1||(scopeStat.mode&511)!==384||scopeStat.size<=0||scopeStat.size>4096)throw Error();
 const scopeBytes=fs.readFileSync(EXPECTED.scope.path);if(sha(scopeBytes)!==EXPECTED.scope.SHA256)throw Error();const scope=JSON.parse(scopeBytes);if(scope.format!==1||scope.tool_source_revision!==EXPECTED.source||scope.repository!==EXPECTED.repository||scope.issue!==EXPECTED.issue||scope.project!==EXPECTED.project||scope.acceptance!==EXPECTED.acceptance||scope.source_commit!==EXPECTED.source_commit||scope.connection!==EXPECTED.connection||scope.owner!==EXPECTED.owner||scope.operation!==EXPECTED.operation||scope.installed_source_revision!==EXPECTED.installed_source||scope.payload_sha256!==EXPECTED.payload_sha256||scope.policy_command_id!==EXPECTED.policy_command_id||scope.preparation!==EXPECTED.preparation||scope.reviewer_preparation!==EXPECTED.reviewer_preparation)throw Error();
 const release=fs.readFileSync('/usr/share/soda/release.json');if(sha(release)!==scope.payload_sha256||JSON.parse(release).Revision!==EXPECTED.installed_source)throw Error();
 const s=fs.lstatSync(EXPECTED.helper.path);if(!s.isFile()||s.isSymbolicLink()||s.uid!==0||s.gid!==0||s.nlink!==1||(s.mode&511)!==448||s.size!==EXPECTED.helper.Bytes||sha(fs.readFileSync(EXPECTED.helper.path))!==EXPECTED.helper.SHA256)throw Error();
 const encoded=cp.execFileSync(EXPECTED.helper.path,['factory-context-status',EXPECTED.scope.path],{input:JSON.stringify(REQUEST),encoding:'utf8',timeout:7000,maxBuffer:8192,env:{PATH:'/usr/bin:/bin'},stdio:['pipe','pipe','pipe']});
 const result=JSON.parse(encoded);if(result.format!==1||result.tool_source_revision!==EXPECTED.source||result.scope_sha256!==EXPECTED.scope.SHA256||result.run_id!==REQUEST.run_id||result.assignment_id!==REQUEST.assignment_id||result.repository!==EXPECTED.repository||result.issue!==EXPECTED.issue||result.project!==EXPECTED.project||!/^[a-f0-9]{32}$/.test(result.attempt_root))throw Error();
 console.log(JSON.stringify(result));
}catch{console.error('canonical_context_status_unavailable');process.exitCode=1;}
`
  );
}
