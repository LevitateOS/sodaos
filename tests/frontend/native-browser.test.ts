import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdir, mkdtemp, lstat, rm} from 'node:fs/promises';
import path from 'node:path';
import {launchNativeBrowser} from '../installed/native-browser.ts';

const root = path.resolve(import.meta.dirname, '../..');

test('native browser refuses long private socket paths before starting a process', async () => {
  await assert.rejects(
    launchNativeBrowser(
      {
        executablePath() {
          assert.fail('must not launch');
        },
        connectOverCDP() {
          assert.fail('must not connect');
        },
      },
      '/' + 'x'.repeat(110),
      '/unused'
    ),
    /socket path too long/
  );
});

test('native browser refuses an occupied socket without removing its owner', async () => {
  await mkdir(path.join(root, '.artifacts'), {recursive: true});
  const run = await mkdtemp(path.join(root, '.artifacts', 'soda-cdp-'));
  const socket = path.join(run, 'cdp.sock');
  const owner = Bun.serve({
    unix: socket,
    fetch() {
      return new Response('test owner');
    },
  });
  try {
    await assert.rejects(
      launchNativeBrowser(
        {
          executablePath() {
            assert.fail('must not launch');
          },
          connectOverCDP() {
            assert.fail('must not connect');
          },
        },
        run,
        run
      ),
      /attachment failed/
    );
    assert((await lstat(socket)).isSocket());
    assert(owner.url.protocol === 'unix:');
  } finally {
    owner.stop(true);
    await rm(run, {recursive: true}); // Exact temporary test-owned directory only.
  }
});
