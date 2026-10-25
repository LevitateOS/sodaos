// Protected worker adapter. Rust owns authority, credential delivery, native
// custody and sealing; the maintained Docker composition owns all product work.
import assert from 'node:assert/strict';
import {lstat, mkdir, open, readFile} from 'node:fs/promises';
import path from 'node:path';
import type {ResolvedQualificationInput} from './scenario';
import {scenarioInput} from './validate';
import {collectDockerPhases, dockerPhaseConfiguration, type DockerPhaseDetails} from './phases/docker-phases';
import {check, COMMIT, sha} from './phases/support';

declare const COLLECTOR_TOOL_REVISION: string;
const INPUT = '/run/soda-qualification-candidate';
const OUTPUT = '/run/soda-qualification-output';
const RUNTIME = '/run/soda-qualification-runtime';
const RESOLVED = '/run/soda-qualification-journey.json';
const retained = new Map<string, {SHA256: string; Bytes: number}>();
let totalBytes = 0;

async function syncDirectory(directory: string) {
  const handle = await open(directory, 'r');
  try {
    await handle.sync();
  } finally {
    await handle.close();
  }
}
async function privateDirectory(directory: string) {
  const stat = await lstat(directory);
  check(
    stat.isDirectory() && !stat.isSymbolicLink() && stat.uid === process.getuid!() && (stat.mode & 0o777) === 0o700,
    'private_evidence_directory'
  );
}
async function saveBytes(relative: string, bytes: Uint8Array) {
  check(
    /^[A-Za-z0-9._/-]{1,256}$/.test(relative) &&
      !relative.startsWith('/') &&
      relative.split('/').every((part) => part !== '' && part !== '.' && part !== '..') &&
      bytes.length > 0 &&
      bytes.length <= 16 << 20 &&
      !retained.has(relative) &&
      retained.size < 1024 &&
      totalBytes + bytes.length <= 64 << 20,
    'bounded_evidence_path'
  );
  totalBytes += bytes.length;
  let directory = OUTPUT;
  await privateDirectory(directory);
  for (const part of relative.split('/').slice(0, -1)) {
    const parent = directory;
    directory = path.join(directory, part);
    try {
      await mkdir(directory, {mode: 0o700});
      await syncDirectory(parent);
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'EEXIST') throw error;
    }
    await privateDirectory(directory);
  }
  const handle = await open(path.join(OUTPUT, relative), 'wx', 0o600);
  try {
    await handle.writeFile(bytes);
    await handle.sync();
  } finally {
    await handle.close();
  }
  await syncDirectory(directory);
  retained.set(relative, {SHA256: sha(bytes), Bytes: bytes.length});
}
const save = (relative: string, value: unknown) =>
  saveBytes(relative, new TextEncoder().encode(JSON.stringify(value, null, 2) + '\n'));
function command(executable: string, args: string[], stdin?: string, timeout = 45000) {
  const result = Bun.spawnSync([executable, ...args], {
    stdin: stdin === undefined ? 'ignore' : new TextEncoder().encode(stdin),
    stdout: 'pipe',
    stderr: 'pipe',
    timeout,
    maxBuffer: 4 << 20,
    env: {PATH: '/usr/sbin:/usr/bin:/sbin:/bin', HOME: OUTPUT + '/.work/home', TMPDIR: OUTPUT + '/.work/tmp'},
  });
  check(result.exitCode === 0, 'bounded_native_command_unconfirmed');
  return new TextDecoder().decode(result.stdout).trim();
}

async function main() {
  process.umask(0o077);
  assert.deepEqual(process.argv.slice(2, 6), ['--candidate', INPUT, '--out', OUTPUT]);
  check(process.argv.length === 8 && process.argv[6] === '--target', 'fixed_worker_arguments');
  check(typeof process.getuid === 'function' && process.getuid() !== 0, 'separate_qualifier_identity');
  check(COMMIT.test(COLLECTOR_TOOL_REVISION), 'full_collector_source_required');
  const input = JSON.parse(await readFile(RESOLVED, 'utf8')) as ResolvedQualificationInput;
  const scenario = scenarioInput(input.scenario);
  check(
    input.format === 1 &&
      input.admission !== undefined &&
      !scenario.authority.held &&
      scenario.authority.scope_receipt_file !== null &&
      scenario.driver.source_revision === COLLECTOR_TOOL_REVISION &&
      scenario.target.name === process.argv[7],
    'journey_execution_held_or_unbound'
  );
  const details = JSON.parse(await readFile(RUNTIME + '/details.json', 'utf8')) as DockerPhaseDetails;
  check(
    details.public_download_directory === RUNTIME + '/public-docker' &&
      details.package_directory === INPUT + '/packages',
    'fixed_worker_public_paths'
  );
  // Validate the complete selected composition before credentials or transport.
  const cfg = dockerPhaseConfiguration(input, details);
  check(scenario.trust, 'explicit_public_trust_required');
  for (const [name, pin] of [
    ['known_hosts', scenario.trust.known_hosts],
    ['appliance-ca.crt', scenario.trust.appliance_ca],
  ] as const)
    check(sha(await readFile(RUNTIME + '/trust/' + name)) === pin.sha256, 'exact_worker_public_trust');
  const credentials = process.env.CREDENTIALS_DIRECTORY;
  check(
    typeof credentials === 'string' && /^\/run\/credentials\/soda-qualify-[a-f0-9]{32}\.service$/.test(credentials),
    'systemd_credential_directory'
  );
  await privateDirectory(OUTPUT);
  const work = OUTPUT + '/.work';
  const home = work + '/home';
  const result = await collectDockerPhases(input, details, {
    save,
    saveBytes,
    sshRead(program, timeout = 45000) {
      check(Number.isSafeInteger(timeout) && timeout >= 1000 && timeout <= 1800000, 'bounded_fixed_ssh_timeout');
      return command(
        '/usr/bin/ssh',
        [
          '-T',
          '-F',
          '/dev/null',
          '-p',
          '22',
          '-oBatchMode=yes',
          '-oIdentitiesOnly=yes',
          '-oIdentityAgent=none',
          '-oStrictHostKeyChecking=yes',
          '-oUpdateHostKeys=no',
          '-oForwardAgent=no',
          '-oForwardX11=no',
          '-oPermitLocalCommand=no',
          '-oClearAllForwardings=yes',
          '-oRemoteCommand=none',
          '-oConnectTimeout=10',
          '-oControlMaster=no',
          '-oControlPath=none',
          '-oControlPersist=no',
          '-oGlobalKnownHostsFile=/dev/null',
          '-oUserKnownHostsFile=' + RUNTIME + '/trust/known_hosts',
          '-i',
          credentials + '/guest-ssh-key',
          'root@' + cfg.guest.ssh_host,
          '/usr/bin/node -',
        ],
        program,
        timeout
      );
    },
    async openBrowser() {
      // Initialize Playwright only after scope/trust admission. Its data files
      // live in the controller's fixed read-only runtime, not the source tree.
      const {chromium} = await import('playwright');
      await mkdir(home + '/.pki/nssdb', {recursive: true, mode: 0o700});
      const database = 'sql:' + home + '/.pki/nssdb';
      command('/usr/bin/certutil', ['-N', '--empty-password', '-d', database]);
      command('/usr/bin/certutil', [
        '-A',
        '-d',
        database,
        '-n',
        'Admitted target CA',
        '-t',
        'C,,',
        '-i',
        RUNTIME + '/trust/appliance-ca.crt',
      ]);
      return chromium.launchPersistentContext(work + '/profile', {
        executablePath: RUNTIME + '/chromium/chrome',
        headless: true,
        chromiumSandbox: true,
        viewport: {width: 1280, height: 900},
        env: {
          PATH: '/usr/sbin:/usr/bin:/sbin:/bin',
          HOME: home,
          TMPDIR: work + '/tmp',
          XDG_CONFIG_HOME: home + '/.config',
          XDG_DATA_HOME: home + '/.local/share',
          DEBUG: '',
          PWDEBUG: '0',
        },
      });
    },
    async ownerPassword() {
      return (await readFile(credentials + '/workspace-owner-password', 'utf8')).trim();
    },
  });
  for (const kind of ['installation', 'browser_project', 'factory', 'persistence'] as const) {
    const time = result.times[kind];
    const files = Object.fromEntries(
      [...retained]
        .filter(([name]) => name.startsWith(kind + '/'))
        .map(([name, id]) => [name.slice(kind.length + 1), id.SHA256])
    );
    check(result.phases[kind] && time && Object.keys(files).length > 0, 'actual_phase_evidence_required');
    await save(kind + '/observation.json', {
      Owner: kind === 'factory' ? 'P06' : 'P02',
      RequestedRevision: scenario.candidate.source_revision,
      ToolRevision: COLLECTOR_TOOL_REVISION,
      RequestedArchitecture: 'x86_64',
      Target: scenario.target.name,
      ClientPlatform: 'linux/amd64',
      Action: 'exec',
      Outcome: 'completed',
      Execution: 'completed',
      Evidence: 'completed',
      Cleanup:
        'Authorized TTYs ended, retained resources stopped and browser closed; controller owns transient worker retirement.',
      Topology:
        'Pinned root SSH to Docker host; fixed native commands inside stock outer; trusted normal-owner browser.',
      ExitCode: 0,
      ToolDirty: false,
      Invocation: [
        'soda-qualification-collector',
        '--candidate',
        INPUT,
        '--out',
        OUTPUT,
        '--target',
        scenario.target.name,
      ],
      Files: files,
      Artifacts: result.artifacts,
      Started: time.started,
      Finished: time.finished,
    });
  }
  await save('collector.json', {
    candidate_sha256: scenario.candidate.candidate_sha256,
    payload_sha256: scenario.candidate.payload_sha256,
    source_revision: scenario.candidate.source_revision,
    purpose: 'production',
    target: scenario.target.name,
    complete: true,
  });
  console.log(JSON.stringify({complete: true, distribution: 'docker', noQualificationSealOrPublication: true}));
}

await main().catch(async (error: unknown) => {
  await save('failure.json', {
    complete: false,
    phase: 'docker_phases',
    noAutomaticRetry: true,
    reason:
      error instanceof Error && /^[a-z0-9_]+$/.test(error.message) ? error.message : 'collector_phase_unconfirmed',
  }).catch(() => {});
  console.error('Protected journey collection incomplete; evidence retained without retry or finalization.');
  process.exitCode = 1;
});
