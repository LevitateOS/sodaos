import test from 'node:test';
import assert from 'node:assert/strict';
import {journeyInput, matchesTerminalReservation} from '../installed/sodaspaces-input';
import {matrixInput} from '../installed/sodaspaces-matrix-input';
import {inspectMatrixProcess, matrixShellCommand} from '../installed/sodaspaces-matrix-native';
import {cliProtocolObservation} from '../installed/sodaspaces-cli';
const base = journeyInput(
  {
    ca_file: '/synthetic/ca',
    origin: 'https://fixture.invalid',
    target: 'fixture',
    repository_id: '7',
    repository_path: '/alice/Alpha',
    revision: '1'.repeat(40),
    terminal_actions: ['create', 'end'],
    users: [
      {id: '1', login: 'alice', password_file: '/synthetic/a'},
      {id: '2', login: 'bob', password_file: '/synthetic/b'},
    ],
  },
  false,
  true
);
const scope = () => ({
  target: base.target,
  revision: base.revision,
  actors: ['1', '2'],
  sessions_per_actor: 6,
  actions: ['create', 'attach', 'end'],
  ssh_config: '/synthetic/ssh',
  projects: [
    {
      environment: 'p' + '1'.repeat(24),
      repository_id: '7',
      repository_path: '/alice/Alpha',
      ssh: ['soda-matrix-a-alice', 'soda-matrix-a-bob'],
    },
    {
      environment: 'p' + '2'.repeat(24),
      repository_id: '8',
      repository_path: '/alice/Beta',
      ssh: ['soda-matrix-b-alice', 'soda-matrix-b-bob'],
    },
  ],
  cli: [],
  cli_effects: [],
  provider_use: 'none',
});
test('matrix approval cannot inherit a single-terminal, foreign actor/project or provider scope', () => {
  assert.deepEqual(matrixInput(scope(), base), scope());
  assert.throws(() => matrixInput(base, base));
  for (const change of [
    {target: 'another'},
    {revision: '2'.repeat(40)},
    {actors: ['2', '1']},
    {sessions_per_actor: 7},
    {actions: ['create', 'end']},
    {actions: [...scope().actions, 'stop']},
    {provider_use: 'browser-and-ssh-for-declared-clis'},
    {unknown: true},
    {projects: [scope().projects[0], scope().projects[0]]},
  ])
    assert.throws(() => matrixInput({...scope(), ...change}, base));
  const cli = ['codex', 'claude', 'pi'].map((tool) => ({
    tool,
    version: 'declared-version',
    prompt_file: '/synthetic/prompt',
    expected_text: '非echo',
    ready_text: 'Fixture CLI ready',
    minimum_output_bytes: 4096,
  }));
  const cli_effects = [
    'personal-cli-state',
    'provider-calls',
    'browser-and-ssh-pty',
    'interactive-input',
    'interrupt-and-disconnect',
  ];
  assert.equal(
    matrixInput({...scope(), cli, cli_effects, provider_use: 'browser-and-ssh-for-declared-clis'}, base).cli.length,
    3
  );
  assert.throws(() => matrixInput({...scope(), cli}, base));
  for (const change of [{ready_text: ''}, {version: ''}, {minimum_output_bytes: 1}])
    assert.throws(() =>
      matrixInput(
        {
          ...scope(),
          cli: cli.map((item) => ({...item, ...change})),
          cli_effects,
          provider_use: 'browser-and-ssh-for-declared-clis',
        },
        base
      )
    );
  assert.throws(() =>
    matrixInput(
      {...scope(), cli: [cli[0], cli[0], cli[0]], cli_effects, provider_use: 'browser-and-ssh-for-declared-clis'},
      base
    )
  );
  assert.throws(() =>
    matrixInput(
      {
        ...scope(),
        cli: cli.map((item) => ({...item, api_key: 'forbidden'})),
        cli_effects,
        provider_use: 'browser-and-ssh-for-declared-clis',
      },
      base
    )
  );
});
test('exact native observer requires original account, PID/start, unit, cgroup and records; quoted shell parses without execution', async () => {
  const request = matrixInput(scope(), base),
    project = request.projects[0];
  const session = {id: 'a'.repeat(32), name: 'probe', actor: '1', environment: project.environment};
  const facts = {pid: 123, start: '456', login: 'alice', marker: 'probe', tty: true, term: 'screen-256color'};
  const live = {
    login: 'alice',
    start: '456',
    record: true,
    socket: true,
    owned_socket: true,
    populated: '1',
    state: 'active',
  };
  await inspectMatrixProcess(request, project, 0, session, facts, false, (command) => {
    assert(command.includes(session.id));
    assert(command.startsWith('sh -c '));
    const quoted = command.slice('sh -c '.length);
    assert(quoted.startsWith("'") && quoted.endsWith("'"));
    const code = quoted.slice(1, -1).replaceAll(`'\\''`, `'`);
    const result = Bun.spawnSync(['sh', '-n'], {
      stdin: Buffer.from(code),
      stdout: 'pipe',
      stderr: 'pipe',
    });
    assert.equal(result.exitCode, 0, result.stderr.toString());
    return JSON.stringify(live);
  });
  await assert.rejects(
    inspectMatrixProcess(request, project, 0, session, facts, false, () => JSON.stringify({...live, login: 'bob'}))
  );
  for (const change of [{start: '457'}, {owned_socket: false}, {record: false}, {populated: '0'}, {state: 'inactive'}])
    await assert.rejects(
      inspectMatrixProcess(request, project, 0, session, facts, false, () => JSON.stringify({...live, ...change}))
    );
  let reads = 0;
  await inspectMatrixProcess(request, project, 0, session, facts, true, () =>
    JSON.stringify(
      ++reads === 1 ? live : {...live, start: null, record: false, socket: false, populated: '0', state: 'inactive'}
    )
  );
  assert.equal(reads, 2);
});
test('matrix shell producer reports the live parent PID and start time', async () => {
  const marker = 'SODA_FACT_' + 'a'.repeat(32),
    command = matrixShellCommand(marker, 'probe', true),
    script = `parent=$$; raw=$(cat /proc/$parent/stat); rest=$(printf '%s\\n' "$raw" | sed 's/.*) //'); start=$(printf '%s\\n' "$rest" | awk '{print $20}'); printf 'SODA_EXPECT:%s:%s\\n' "$parent" "$start"; ${command}; IFS= read -r release || :`;
  const child = Bun.spawn(['/bin/bash', '-c', script], {
    stdin: 'pipe',
    stdout: 'pipe',
    stderr: 'pipe',
    env: {...process.env, TERM: 'screen-256color'},
    timeout: 15000,
    killSignal: 'SIGKILL',
  });
  const reader = child.stdout.getReader();
  const decoder = new TextDecoder();
  let output = '';
  try {
    while (!output.split('\n').some((line) => line.startsWith(marker + ':'))) {
      const next = await reader.read();
      assert(!next.done, `shell producer exited before facts: ${output}`);
      output += decoder.decode(next.value, {stream: true});
      assert(output.length < 4096, 'shell producer output exceeded bound');
    }
    const expected = output.split('\n').find((line) => line.startsWith('SODA_EXPECT:'));
    const observed = output.split('\n').find((line) => line.startsWith(marker + ':'));
    assert(expected && observed, output);
    const [, expectedPid, expectedStart] = expected.split(':');
    const facts = JSON.parse(observed.slice(marker.length + 1));
    assert.equal(facts.pid, Number(expectedPid));
    assert.equal(facts.start, expectedStart);
    const live = await Bun.file(`/proc/${expectedPid}/stat`).text();
    const liveStart = live
      .slice(live.lastIndexOf(') ') + 2)
      .trim()
      .split(/\s+/)[19];
    assert.equal(liveStart, expectedStart, 'persistent parent shell must remain alive while observed');
  } finally {
    reader.releaseLock();
    if (child.exitCode === null) {
      child.stdin.end();
    }
    await child.exited;
  }
  assert.equal(child.exitCode, 0, 'persistent shell should exit cleanly after release');
});
test('reservation permission admits only the selected name and measured bounded geometry', () => {
  const body = {cols: 194, rows: 41, name: 'selected'};
  assert(matchesTerminalReservation(JSON.stringify(body), 'selected'));
  for (const changed of [
    {...body, name: 'other'},
    {...body, cols: true},
    {...body, rows: 301},
    {...body, command: 'id'},
    null,
  ])
    assert(!matchesTerminalReservation(JSON.stringify(changed), 'selected'));
  assert(!matchesTerminalReservation(' '.repeat(1025), 'selected'));
});
test('CLI parser records observed protocol only, never generated agent semantics or a pass', () => {
  assert.deepEqual(cliProtocolObservation('plain log', 'answer'), {
    expected_output: false,
    unicode: false,
    cursor: false,
    alternate_screen: false,
    mouse_mode: false,
    bracketed_paste: false,
  });
  const observation = cliProtocolObservation('\x1b[?1049h\x1b[?1006h\x1b[?2004h\x1b[2;3H答案', '答案');
  assert(Object.values(observation).every(Boolean));
  assert(!('outcome' in observation));
});
