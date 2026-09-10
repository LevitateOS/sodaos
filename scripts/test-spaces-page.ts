// Go owns HTML/actor authorization; Chromium consumes those exact fixture bytes
// with the production page module. No server installation or provider is involved.
import assert from 'node:assert/strict';
import {mkdir, mkdtemp, stat} from 'node:fs/promises';
import path from 'node:path';
const root = path.resolve(import.meta.dirname, '..');
await mkdir(path.join(root, '.artifacts'), {recursive: true});
const run = await mkdtemp(path.join(root, '.artifacts/pages-'));
console.log(`Go/browser fixtures retained at ${run}`);
const fixtures = {
  SODA_SPACES_PAGE_HTML: path.join(run, 'spaces.json'),
  SODA_RUNNERS_PAGE_HTML: path.join(run, 'runners.json'),
  SODA_REPOSITORY_SETTINGS_HTML: path.join(run, 'repository-settings.json'),
};
const env = {...process.env, ...fixtures};
const producers = [
  'TestSpacesHTMLSessionAuthorityAndBoundedException',
  'TestRunnerOperatorGatesBeforeNativeAndDecode',
  'TestRepositorySettingsUsesFreshStableIdentityAndSharedControls',
];
const produced = Bun.spawn(['go', 'test', '-mod=readonly', '-count=1', './internal/web', '-run', `^(${producers.join('|')})$`], {cwd: root, env, stdout: 'inherit', stderr: 'inherit'});
const producerStatus = await produced.exited;
if (producerStatus) process.exit(producerStatus);
for (const [name, file] of Object.entries(fixtures)) {
  const info = await stat(file);
  assert(info.isFile() && info.size > 0, `Missing or empty required fixture: ${name}`);
}
const browser = Bun.spawn(['bun', 'test', '--timeout', '90000',
  'tests/frontend/spaces-page.test.ts',
  'tests/frontend/runners.test.ts',
  'tests/frontend/repository-settings.test.ts',
], {cwd: root, env, stdout: 'inherit', stderr: 'inherit'});
process.exit(await browser.exited);
