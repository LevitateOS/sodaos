// Real retained-file integration. No provider outputs/attempts are synthesized,
// no appliance is contacted and the selected run is never modified.
import assert from 'node:assert/strict';
import {copyFile, mkdir, readFile, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {journeyReport} from './report';
import {sha256} from './validate';

const flags = new Map<string, string>();
for (let index = 2; index < process.argv.length; index += 2) {
  const key = process.argv[index],
    value = process.argv[index + 1];
  assert(key && value && !flags.has(key));
  flags.set(key, value);
}
assert.deepEqual([...flags.keys()].sort(), ['--out', '--run']);
const run = flags.get('--run')!,
  out = flags.get('--out')!;
assert(out.startsWith('/home/') && path.resolve(out) === out && out !== run && !out.startsWith(run + '/'));
await mkdir(out, {mode: 0o700});
const report = await journeyReport(run);
assert(report.qualification === null && report.automatic_actions === false);
assert.equal(JSON.stringify(await journeyReport(run)), JSON.stringify(report), 'report must be deterministic');
for (const source of report.sources)
  assert.equal(sha256(await readFile(source.path)), source.sha256, 'selected receipts must remain unchanged');
// Exercise the real immutable request binding with a data-only copy of the
// existing held check, not another driver or an invented Factory execution.
assert(report.collection === null && report.readiness.ready === false && report.phases.length === 0);
const altered = path.join(out, 'changed-scenario');
await mkdir(altered, {mode: 0o700});
for (const name of ['scenario.json', 'readiness.json']) await copyFile(path.join(run, name), path.join(altered, name));
const original = JSON.parse(await readFile(path.join(run, 'request.json'), 'utf8'));
original.scenario.path = path.join(altered, 'scenario.json');
original.scenario.sha256 = '0'.repeat(64);
await writeFile(path.join(altered, 'request.json'), JSON.stringify(original), {flag: 'wx', mode: 0o444});
await assert.rejects(() => journeyReport(altered), /report_receipt_binding_changed/);
const forbidden = JSON.stringify(report);
for (const field of [
  'guest_ssh_key_file',
  'workspace_owner_password_file',
  'application_owner_files',
  'provider-execution',
  'raw_response',
  'CREDENTIALS_DIRECTORY',
])
  assert(!forbidden.includes(field), 'report must not expose private/provider references');
await writeFile(path.join(out, 'report.json'), JSON.stringify(report, null, 2) + '\n', {flag: 'wx', mode: 0o600});
const result = {
  format: 1,
  complete: true,
  scenario_receipt: report.sources.find((s) => s.role === 'scenario'),
  readiness_receipt: report.sources.find((s) => s.role === 'readiness' || s.role === 'run_readiness'),
  report_sha256: sha256(await readFile(path.join(out, 'report.json'))),
  deterministic: true,
  source_receipts_unchanged: true,
  changed_scenario_refused: true,
  collection: report.collection,
  qualified: false,
  provider_calls: 0,
  guest_mutations: 0,
};
await writeFile(path.join(out, 'result.json'), JSON.stringify(result, null, 2) + '\n', {flag: 'wx', mode: 0o600});
console.log(JSON.stringify(result));
