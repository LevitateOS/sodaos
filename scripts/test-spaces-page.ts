// The existing native fixture owns login, actual Forgejo HTML/assets and the
// isolated Go backend. Browser consumers intercept only their operation APIs.
import {mkdir, mkdtemp} from 'node:fs/promises';
import path from 'node:path';
const root = path.resolve(import.meta.dirname, '..');
await mkdir(path.join(root, '.artifacts'), {recursive: true});
const run = await mkdtemp(path.join(root, '.artifacts/pages-'));
console.log(`Native page fixtures retained at ${run}`);
const child = Bun.spawn(['go', 'test', '-mod=readonly', '-count=1', './internal/web', '-run', '^TestNativeConnectionFixture$'], {
 cwd: root, env: {...process.env, SODA_NATIVE_CONNECTION_FIXTURE: path.join(run, 'native'), SODA_PAGE_CONSUMERS: '1'}, stdout: 'inherit', stderr: 'inherit',
});
process.exit(await child.exited);
