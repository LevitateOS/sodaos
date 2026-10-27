// Protected source compilation, exact ancestor extraction and bundle production.
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {spawn} from 'node:child_process';
import {lstat, mkdir, readFile, realpath, unlink, writeFile} from 'node:fs/promises';
import {dirname, join, relative} from 'node:path';
export function sourceChecks(context) {
  const {root, work, command, childEnv, tools, config, APP, CONTRACTS_VERSION} = context;
  async function run(name, args, cwd = root) {
    const child = spawn(command(name), args, {cwd, env: childEnv, stdio: ['ignore', 'pipe', 'pipe']});
    const output = [];
    const errors = [];
    let size = 0;
    const collect = (chunk) => {
      size += chunk.length;
      if (size > 64 * 1024 * 1024) child.kill('SIGTERM');
    };
    child.stdout.on('data', (chunk) => {
      collect(chunk);
      output.push(chunk);
    });
    child.stderr.on('data', (chunk) => {
      collect(chunk);
      errors.push(chunk);
    });
    const timer = setTimeout(() => child.kill('SIGTERM'), 300_000);
    try {
      const code = await new Promise((done, reject) => {
        child.once('error', reject);
        child.once('close', done);
      });
      if (code !== 0)
        await writeFile(join(work, 'failed-command.txt'), Buffer.concat([...output, ...errors]), {mode: 0o600});
      assert.equal(code, 0, `${name} ${args[0]} failed; inspect the actual source/tool prerequisite.`);
      assert(size <= 64 * 1024 * 1024, 'Command output exceeded its bound.');
      return Buffer.concat(output).toString('utf8');
    } finally {
      clearTimeout(timer);
    }
  }

  async function generatedSource(commit) {
    assert.equal((await git(['merge-base', '--is-ancestor', commit, 'HEAD'])).trim(), '');
    const output = join(work, 'baseline');
    await mkdir(output, {mode: 0o700});
    // Extract the actual generated ancestor, including imported .can modules.
    // Git is a system tool; unlike Canlang commands it is not inside the archive.
    const files = (await git(['ls-tree', '-r', '--name-only', '-z', commit])).split('\0').filter(Boolean);
    for (const file of files) {
      assert(!file.split('/').some((part) => part === '..' || part === '.git'), 'Unsafe baseline source path.');
      const target = join(output, file);
      await mkdir(dirname(target), {recursive: true, mode: 0o700});
      const type = (await git(['ls-tree', commit, '--', file])).split(' ', 1)[0];
      assert(type === '100644' || type === '100755', 'Baseline must contain ordinary source files.');
      await writeFile(target, await git(['show', `${commit}:${file}`], true), {mode: 0o600});
    }
    return output;
  }

  async function git(args, bytes = false) {
    const child = spawn('git', args, {cwd: root, env: childEnv, stdio: ['ignore', 'pipe', 'pipe']});
    const output = [];
    let size = 0;
    child.stdout.on('data', (chunk) => {
      size += chunk.length;
      if (size > 2 * 1024 * 1024) child.kill('SIGTERM');
      output.push(chunk);
    });
    child.stderr.resume();
    const code = await new Promise((done, reject) => {
      child.once('error', reject);
      child.once('close', done);
    });
    assert.equal(code, 0, 'The actual generated baseline source is unavailable.');
    assert(size <= 2 * 1024 * 1024, 'Generated source file exceeds its bound.');
    const value = Buffer.concat(output);
    return bytes ? value : value.toString('utf8');
  }

  async function compile(sourceRoot, label) {
    const source = join(sourceRoot, config.source);
    const actual = await realpath(source);
    assert(
      !relative(sourceRoot, actual).startsWith('..') && (await lstat(source)).isFile(),
      'Generated source is missing or escapes its checkout.'
    );
    const directory = join(work, label);
    await mkdir(directory, {mode: 0o700});
    const catalog = join(tools, 'lib/node_modules/@canlang/cloudflare/dist/native-catalog.json');
    await run('can', ['check', '--catalog=' + catalog, config.source], sourceRoot);
    const bytes = await run(
      'can',
      ['compile', '--format=json', '--native-scenario-receipts', '--catalog=' + catalog, config.source],
      sourceRoot
    );
    const artifact = JSON.parse(bytes);
    assert(
      artifact.models.some((model) => model.name === APP + '.EquipmentRequest'),
      'The accepted EquipmentRequest model is missing.'
    );
    assert(
      artifact.operations.every((operation) => operation.name.startsWith(APP + '.')),
      'The accepted contract names one app.'
    );
    const file = join(directory, 'artifact.json');
    await writeFile(file, bytes, {mode: 0o600});
    const examples = JSON.parse(await run('can', ['test', '--artifact', file], sourceRoot));
    assert(examples.ok === true && examples.executed > 0, 'Nonempty real operation examples must pass.');
    return {artifact, file, directory};
  }

  async function bundle(compiled, environmentName) {
    const artifact = compiled.artifact;
    const selected = config[environmentName];
    const bindings = [
      {binding: 'DB', kind: 'd1', logicalName: 'northstar-identity'},
      {binding: 'STATE_DB', kind: 'd1', logicalName: 'northstar-state'},
    ];
    const descriptor = {
      identity: {
        appName: APP,
        sourceRevision: 'sha256:' + artifact.sources[0].sha256,
        languageVersion: artifact.language_version,
        compilerVersion: artifact.tool_version,
        contractsVersion: CONTRACTS_VERSION,
        artifactDigest: createHash('sha256').update(JSON.stringify(artifact)).digest('hex'),
      },
      requiredCapabilities: artifact.requires.map((item) => item.capability),
      resourceBindings: bindings,
      secrets: [],
      schedules: [],
    };
    const environment = {
      environment: 'northstar-' + environmentName,
      resources: bindings.map((requirement) => ({
        requirement,
        resourceId: selected[requirement.binding === 'DB' ? 'identity_resource' : 'state_resource'],
      })),
      secretsPresent: [],
      vars: {CAN_AUTH_ORIGIN: selected.origin},
    };
    const descriptorFile = join(compiled.directory, 'descriptor.json');
    const environmentFile = join(compiled.directory, environmentName + '.environment.json');
    const bundleFile = join(compiled.directory, environmentName + '.bundle.json');
    await writeFile(descriptorFile, JSON.stringify(descriptor), {mode: 0o600});
    await writeFile(environmentFile, JSON.stringify(environment), {mode: 0o600});
    await run('can-soda', [
      'build',
      '--artifact',
      compiled.file,
      '--descriptor',
      descriptorFile,
      '--environment',
      environmentFile,
      '--out',
      bundleFile,
    ]);
    return bundleFile;
  }

  async function publish(from, target) {
    const directory = dirname(target);
    await mkdir(directory, {recursive: true, mode: 0o700});
    assert.equal(await realpath(directory), directory, 'Bundle output directory must not be a symlink.');
    // A previously committed/copied transport is never an input to this check.
    try {
      const existing = await lstat(target);
      assert(existing.isFile() && !existing.isSymbolicLink(), 'Unsafe preexisting bundle output.');
      await unlink(target);
    } catch (error) {
      if (error.code !== 'ENOENT') throw error;
    }
    await writeFile(target, await readFile(from), {flag: 'wx', mode: 0o600});
  }

  return {run, generatedSource, compile, bundle, publish};
}
