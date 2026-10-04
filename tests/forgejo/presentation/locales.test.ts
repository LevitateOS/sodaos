import assert from 'node:assert/strict';
import {test} from 'node:test';
import {spawnSync} from 'node:child_process';
import {mkdtempSync, writeFileSync, readFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join, resolve} from 'node:path';

const root = resolve(import.meta.dirname, '..', '..', '..');

function merge(native: string, additions: string): {status: number; out: string} {
  const dir = mkdtempSync(join(tmpdir(), 'soda-locales-'));
  const nativeFile = join(dir, 'native.ini');
  const additionsFile = join(dir, 'additions.ini');
  const outFile = join(dir, 'locale.ini');
  writeFileSync(nativeFile, native);
  writeFileSync(additionsFile, additions);
  const result = spawnSync(
    'cargo',
    [
      'run',
      '--release',
      '--locked',
      '-p',
      'soda-forgejo-locales',
      '--bin',
      'soda-forgejo-locales',
      '--',
      '--native',
      nativeFile,
      '--additions',
      additionsFile,
      '--out',
      outFile,
    ],
    {encoding: 'utf8', cwd: root}
  );
  const status = result.status ?? 1;
  return {status, out: status === 0 ? readFileSync(outFile, 'utf8') : result.stderr};
}

test('locale generation preserves native bytes and rejects namespace/duplicate-key collisions', () => {
  const native = '[common]\nhome = Home %s\n[settings]\nprofile = Profile\n';
  const extra = '[soda]\nnav_personal = Personal\n';
  const merged = merge(native, extra);
  assert.equal(merged.status, 0, merged.out);
  assert.equal(merged.out, native + '\n' + extra);
  for (const additions of ['[soda]\nx = one\nx = two\n', '[settings]\nprofile = Wrong\n']) {
    const refused = merge(native, additions);
    assert.notEqual(refused.status, 0, 'duplicate namespace/key accepted');
  }
  const incomplete = merge('[soda]\nx = y\n', extra);
  assert.notEqual(incomplete.status, 0, 'incomplete native catalog accepted');
});
