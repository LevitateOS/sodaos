// Go owns HTML/actor authorization; Chromium consumes those exact fixture bytes
// with the production page module. No server installation or provider is involved.
import {mkdtemp} from 'node:fs/promises';
import path from 'node:path';
const root = path.resolve(import.meta.dirname, '..');
const run = await mkdtemp(path.join(root, '.artifacts/spaces-page-'));
const env = {...process.env, SODA_SPACES_PAGE_HTML: path.join(run, 'page.json')};
for (const command of [['go', 'test', '-mod=readonly', './internal/web', '-run', '^TestSpacesHTMLSessionAuthorityAndBoundedException$'], ['bun', 'test', '--timeout', '90000', 'tests/frontend/spaces-page.test.ts']]) {
  const child = Bun.spawn(command, {cwd: root, env, stdout: 'inherit', stderr: 'inherit'});
  const status = await child.exited;
  if (status) process.exit(status);
}
console.log(`Go/browser fixture retained at ${run}`);
