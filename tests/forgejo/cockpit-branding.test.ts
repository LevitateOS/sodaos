import assert from 'node:assert/strict';
import test from 'node:test';
import {chromium} from 'playwright';
import payload from '../../internal/nativebuild/forgejo-payload.json';
import manifest from '../../package.json';

const source = new URL('../../', import.meta.url);
test('Cockpit branding is independent of retired custom-page tooling and uses current canonical assets', async () => {
  assert(!manifest.workspaces.includes('cockpit'));
  assert(!Object.values(manifest.scripts).some(command => command.includes('--cwd cockpit')));
  const staging = await Bun.file(new URL('scripts/stage.py', source)).text();
  assert(!staging.includes("shutil.copytree(source / 'cockpit/dist'"));
  const css = await Bun.file(new URL('assets/branding/cockpit/branding.css', source)).text();
  assert(css.includes('soda-symbol-brutalist.svg') && css.includes('soda-symbol-brutalist-dark.svg'));
  assert(css.includes('fonts/fonts.css') && css.includes('Barlow Condensed'));
  assert(!css.includes('login-background-') && !css.includes('<script') && !css.includes('<form'));
  const image: unknown = await Bun.file(new URL('appliance/locks/tailscale-image.json', source)).json();
  assert(image && typeof image === 'object' && 'reference' in image && image.reference === 'docker.io/tailscale/tailscale:v1.102.4');
});

test('branding stylesheet resolves local fonts, flat backgrounds and native theme tokens without an extension', async t => {
  // Deliberate component styling fixture, not a Cockpit login/PAM/native-page proof.
  const server = Bun.serve({hostname: '127.0.0.1', port: 0, fetch(request) {
    const path = new URL(request.url).pathname;
    if (path === '/') return new Response('<!doctype html><link rel="icon" href="data:,"><link rel="stylesheet" href="/branding.css"><body class="login-pf"><h1 id="brand">Soda OS</h1><label>Native password control<input type="password"></label></body>', {headers: {'Content-Type': 'text/html'}});
    const files: Record<string, string> = {'/branding.css': 'assets/branding/cockpit/branding.css', '/theme.css': 'assets/branding/cockpit/theme.css', '/theme/palette.css': 'assets/branding/theme/palette.css'};
    const font = Object.entries(payload).find(([name]) => name === 'public/assets/soda' + path)?.[1];
    const file = files[path] || (path.startsWith('/fonts/') ? font : undefined);
    return file ? new Response(Bun.file(new URL(file, source))) : new Response(null, {status: 404});
  }});
  t.after(() => server.stop(true));
  const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
  const page = await browser.newPage({viewport: {width: 390, height: 800}});
  await page.goto(server.url.href);
  for (const dark of [false, true]) {
    await page.evaluate(dark => document.documentElement.classList.toggle('pf-v6-theme-dark', dark), dark);
    const style = await page.evaluate(async () => {
      await document.fonts.ready;
      const brand = document.getElementById('brand'); if (!brand) throw Error('Missing branding fixture');
      return {background: getComputedStyle(document.body).backgroundImage, heading: getComputedStyle(brand).fontFamily,
        action: getComputedStyle(document.body).getPropertyValue('--color-primary').trim(), radius: getComputedStyle(document.documentElement).getPropertyValue('--pf-t--global--border--radius--medium').trim()};
    });
    assert.equal(style.background, 'none'); assert(style.heading.includes('Barlow Condensed'));
    assert.equal(style.radius, '0px'); assert(style.action && !style.action.startsWith('var('));
  }
});
