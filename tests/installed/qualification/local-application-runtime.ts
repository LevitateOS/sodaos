// One actual native runtime lifetime shared by App and recovery integration.
import assert from 'node:assert/strict';
import {spawn, spawnSync} from 'node:child_process';
import path from 'node:path';

export async function startLocalApplication(input: {
  runtime: string;
  bundle: string;
  data: string;
  owner_team: string;
  env: NodeJS.ProcessEnv;
  retired(result: {exit_code: number | null; signal: NodeJS.Signals | null; writer_lock_available: boolean}): void;
}) {
  const {runtime, bundle, data, env, owner_team: team} = input;
  const child = spawn(
    runtime,
    ['serve', '--bundle', bundle, '--data', data, '--owner', team, '--host', '127.0.0.1', '--port', '0'],
    {env, stdio: ['ignore', 'pipe', 'pipe']}
  );
  const closed = new Promise<{code: number | null; signal: NodeJS.Signals | null}>((resolve) =>
    child.once('close', (code, signal) => resolve({code, signal}))
  );
  // Runtime diagnostics can include implementation details. Consume them in
  // memory and expose only the command's bounded readiness metadata.
  child.stderr.on('data', () => {});
  let text = '';
  const ready = new Promise<string>((resolve, reject) => {
    const timer = setTimeout(() => reject(Error('application_runtime_readiness_timeout')), 30000);
    child.once('error', () => {
      clearTimeout(timer);
      reject(Error('application_runtime_unavailable'));
    });
    void closed.then(() => {
      clearTimeout(timer);
      reject(Error('application_runtime_exited_before_ready'));
    });
    child.stdout.on('data', (bytes: Buffer) => {
      text += bytes.toString();
      if (text.length > 65536) {
        clearTimeout(timer);
        reject(Error('application_runtime_readiness_invalid'));
        return;
      }
      let newline: number;
      while ((newline = text.indexOf('\n')) >= 0) {
        const line = text.slice(0, newline);
        text = text.slice(newline + 1);
        try {
          const message = JSON.parse(line);
          if (
            message.ok === true &&
            message.command === 'serve' &&
            /^http:\/\/127\.0\.0\.1:[1-9][0-9]*$/.test(message.origin)
          ) {
            clearTimeout(timer);
            resolve(message.origin);
          }
        } catch {
          /* Only the canonical readiness line is admitted. */
        }
      }
    });
  });
  const stop = async () => {
    if (child.exitCode === null && child.signalCode === null) child.kill('SIGTERM');
    let timer: ReturnType<typeof setTimeout> | undefined;
    let forced = false;
    const result = await Promise.race([
      closed,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => {
          forced = true;
          child.kill('SIGKILL');
          void closed.then(() => reject(Error('application_runtime_retirement_forced')));
        }, 20000);
      }),
    ]).finally(() => clearTimeout(timer));
    assert(!forced && result.code === 0 && result.signal === null, 'application_runtime_retirement_unconfirmed');
    const lock = spawnSync('flock', ['--nonblock', path.join(data, '.writer-lock'), 'true'], {
      env,
      stdio: ['ignore', 'pipe', 'pipe'],
      timeout: 5000,
    });
    input.retired({exit_code: result.code, signal: result.signal, writer_lock_available: lock.status === 0});
    assert.equal(lock.status, 0, 'application_writer_not_retired');
  };
  try {
    return {origin: await ready, stop};
  } catch (error) {
    await stop();
    throw error;
  }
}
