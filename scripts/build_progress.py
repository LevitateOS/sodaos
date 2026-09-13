"""Plain build checkpoints shared by the existing shell and Python recipes."""
from contextlib import contextmanager
import os
from pathlib import Path
import signal
import subprocess
import sys
import time


def clock():
    return time.monotonic_ns()


def duration(ns):
    seconds = max(0, ns // 1_000_000_000)
    return f'{seconds // 3600:02}:{seconds // 60 % 60:02}:{seconds % 60:02}'


def origin():
    if 'SODA_BUILD_START_NS' not in os.environ:
        os.environ['SODA_BUILD_START_NS'] = str(clock())
    return int(os.environ['SODA_BUILD_START_NS'])


def emit(kind, label, started=None):
    now = clock()
    line = f'{kind:<8} {label}'
    if started is not None:
        line += f' | section {duration(now - started)} | total {duration(now - origin())}'
    print(line, file=sys.stderr, flush=True)
    log = os.environ.get('SODA_BUILD_TIMING_LOG')
    if log:
        # Only checkpoint text, never captured commands or private inputs.
        fd = os.open(log, os.O_WRONLY | os.O_APPEND | os.O_NOFOLLOW)
        with os.fdopen(fd, 'a') as stream:
            stream.write(line + '\n')
    return line


def create_log(path):
    if os.environ.get('SODA_BUILD_TIMING_LOG'):
        return
    path = Path(path)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    os.close(fd)
    os.environ['SODA_BUILD_TIMING_LOG'] = str(path.absolute())
    emit('LOG', str(path))


def finish(label, code):
    if os.environ.get('SODA_BUILD_CHILD') == '1':
        return
    log = os.environ.get('SODA_BUILD_TIMING_LOG')
    if log:
        rows = Path(log).read_text().splitlines()
        print('\nSECTION SUMMARY', file=sys.stderr, flush=True)
        for row in rows:
            if row.startswith(('DONE ', 'FAILED ', 'CANCELLED ')):
                print('  ' + row, file=sys.stderr, flush=True)
    outcome = 'SUCCESS' if code == 0 else 'CANCELLED' if code in (130, 143) else 'FAILED'
    emit(outcome, f'{label} | total {duration(clock() - origin())} | exit {code}')
    if log:
        print(f'TIMINGS  {log}', file=sys.stderr, flush=True)


class Progress:
    def __init__(self):
        self.active = None

    def next(self, label):
        self.end()
        origin()
        self.active = (label, clock())
        emit('START', label)

    def end(self, code=0):
        if self.active:
            label, started = self.active
            self.active = None
            emit('DONE' if code == 0 else 'CANCELLED' if code in (130, 143) else 'FAILED', label, started)

    @contextmanager
    def section(self, label):
        # Helpers temporarily own a section and must restore their caller's state.
        previous = self.active
        self.active = None
        self.next(label)
        try:
            yield
        except BaseException as error:
            self.end(exit_code(error))
            raise
        else:
            self.end()
        finally:
            self.active = previous


def exit_code(error):
    if isinstance(error, KeyboardInterrupt):
        return 130
    if isinstance(error, SystemExit):
        return error.code if isinstance(error.code, int) else 1
    if isinstance(error, subprocess.CalledProcessError):
        return error.returncode if error.returncode >= 0 else 128 - error.returncode
    return 1


def supervise(argv):
    """Forward cancellation to this invocation's entire child process group."""
    origin()
    env = dict(os.environ, SODA_BUILD_SUPERVISED='1')
    process = subprocess.Popen(argv, env=env, start_new_session=True)
    cancelled = 0

    class Interrupted(BaseException):
        pass

    def stop(signum, frame):
        nonlocal cancelled
        cancelled = signum
        raise Interrupted()

    previous = {sig: signal.signal(sig, stop) for sig in (signal.SIGINT, signal.SIGTERM)}
    try:
        try:
            result = process.wait()
        except Interrupted:
            # Forward outside the signal handler: Popen.wait holds a non-reentrant
            # lock, so waiting inside that handler would deadlock cancellation.
            for sig in previous:
                signal.signal(sig, signal.SIG_IGN)
            try:
                os.killpg(process.pid, cancelled)
            except ProcessLookupError:
                pass
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
                emit('CANCELLED', f'Build interrupted | total {duration(clock() - origin())}')
            return 128 + cancelled
        return result if result >= 0 else 128 - result
    finally:
        for sig, handler in previous.items():
            signal.signal(sig, handler)


if __name__ == '__main__':
    action, *args = sys.argv[1:]
    if action == 'supervise':
        raise SystemExit(supervise(args))
    if action == 'clock':
        print(clock())
    elif action == 'emit':
        emit(args[0], args[1], int(args[2]) if len(args) > 2 else None)
    elif action == 'finish':
        finish(args[0], int(args[1]))
    elif action == 'create-log':
        create_log(Path(args[0]))
