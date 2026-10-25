// Compile with Bun's existing standalone-driver support. Rust owns the immutable
// request/scenario envelope; this program owns selected observations/validation.
import assert from 'node:assert/strict';
import {readFile, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {scenarioInput} from './validate';
import {checkScenario} from './readiness';
import {journeyStatus} from './status';
import type {Readiness} from './readiness';

async function main() {
  const [action, ...args] = process.argv.slice(2);
  const flags = new Map<string, string>();
  for (let i = 0; i < args.length; i += 2) {
    const key = args[i],
      value = args[i + 1];
    assert(key && value && !flags.has(key));
    flags.set(key, value);
  }
  if (action === 'check') {
    assert.equal([...flags.keys()].sort().join(','), '--out,--scenario');
    const out = flags.get('--out')!,
      input = flags.get('--scenario')!;
    assert(out.startsWith('/home/') && path.resolve(out) === out && input === path.join(out, 'scenario.json'));
    const scenario = scenarioInput(JSON.parse(await readFile(input, 'utf8')));
    const result = await checkScenario(scenario);
    // Only complete observations are saved; no raw errors, transcripts or logs.
    if (result.observation)
      await writeFile(path.join(out, 'observation.json'), JSON.stringify(result.observation, null, 2) + '\n', {
        flag: 'wx',
        mode: 0o600,
      });
    await writeFile(path.join(out, 'readiness.json'), JSON.stringify(result, null, 2) + '\n', {
      flag: 'wx',
      mode: 0o600,
    });
    console.log(JSON.stringify(journeyStatus(scenario, result)));
  } else if (action === 'status') {
    assert.equal([...flags.keys()].join(','), '--run');
    const run = flags.get('--run')!;
    assert(run.startsWith('/home/') && path.resolve(run) === run);
    // Status has no transport or effect call. It reports the last checked state.
    const result = JSON.parse(await readFile(path.join(run, 'readiness.json'), 'utf8')) as Readiness;
    assert(result.format === 1 && typeof result.ready === 'boolean' && Array.isArray(result.checks));
    const scenario = scenarioInput(JSON.parse(await readFile(path.join(run, 'scenario.json'), 'utf8')));
    console.log(JSON.stringify(journeyStatus(scenario, result)));
  } else throw Error('Unknown journey action');
}
await main().catch(() => {
  console.error('Journey input or read-only check failed; no execution admitted.');
  process.exitCode = 1;
});
