import test, {before, after, type TestContext} from 'node:test';
import assert from 'node:assert/strict';
import {existsSync, readFileSync} from 'node:fs';
import path from 'node:path';
import type {Page} from 'playwright';
import {browserZoomFixture} from './fixtures/browser-zoom.js';

// NFX-F06 local geometry with Fountain's A04 styles and Soda branding.
// Zoom uses Chromium's actual tabs.setZoom/getZoom setting. The container
// markup is transcribed, so this remains authored-fixture evidence rather
// than native template rendering or installed qualification. The markup is
// transcribed from Fountain templates/extensions/{page,panels}.tmpl and
// templates/base/{head,footer}.tmpl in the canonical Fountain source, the body
// flex column from web_src/css/base.css, and the container rules are
// served from Fountain's own CSS files, not retyped. The shared owner has no
// competing page mount; the panels home (.extension-workspace-panels, flex:1 with a min-height
// floor) supplies definite host space. Soda invents no host height.

const root = path.resolve(import.meta.dirname, '../..');
const fountain = [path.join(root, '../forgejo-ext'), path.join(root, '../../forgejo-ext')].find((p) =>
  existsSync(path.join(p, 'web_src/css/features/extensions.css'))
);
assert(fountain, 'Fountain checkout (forgejo-ext) is required beside Soda for F06 native CSS');

let browser: Awaited<ReturnType<typeof browserZoomFixture>>;
let server: ReturnType<typeof Bun.serve>;

function fountainCss(rel: string): string {
  return readFileSync(path.join(fountain!, 'web_src/css', rel), 'utf8');
}

const shellHead = (
  title: string
) => `<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1"><title>${title}</title><link rel="icon" href="data:,">
<link rel="stylesheet" href="/assets/native/container.css"><link rel="stylesheet" href="/assets/native/extensions.css">
<link rel="stylesheet" href="/assets/sodaspaces-drawer.css"><link rel="stylesheet" href="/assets/sodaspaces-page.css"><link rel="stylesheet" href="/assets/sodaspaces-terminal.css"><link rel="stylesheet" href="/assets/soda/forgejo/components.css"><link rel="stylesheet" href="/assets/soda/forgejo/components-buttons.css">
<style>:root{--page-margin-x:32px}@media(max-width:767.98px){:root{--page-margin-x:8px}}html,body{height:100%}body{margin:0;display:flex;flex-direction:column}.full.height{flex-grow:1;padding-bottom:80px}.page-content{display:block}</style>`;

before(async () => {
  const wsBuild = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/workspace-fixture.ts')],
    target: 'browser',
    format: 'esm',
    minify: true,
    define: {'process.env.NODE_ENV': '"production"'},
    plugins: [
      {
        name: 'terminal-vendor',
        setup(b) {
          b.onResolve({filter: /^\.\/soda-terminal\/(?:xterm|addon-fit)\.mjs$/}, (a) => ({
            path: a.path,
            external: true,
          }));
        },
      },
    ],
  });
  assert(wsBuild.success, `workspace fixture build failed: ${wsBuild.logs.join('\n')}`);
  const drawerBuild = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/drawer-fixture.ts')],
    target: 'browser',
    format: 'esm',
    define: {'process.env.NODE_ENV': '"production"'},
  });
  assert(drawerBuild.success, `drawer fixture build failed: ${drawerBuild.logs.join('\n')}`);
  const panelBuild = await Bun.build({
    entrypoints: [path.join(root, 'tests/frontend/fixtures/persistent-panel-fixture.ts')],
    target: 'browser',
    format: 'esm',
    minify: true,
    define: {'process.env.NODE_ENV': '"production"'},
    plugins: [
      {
        name: 'terminal-vendor',
        setup(b) {
          b.onResolve({filter: /^\.\/soda-terminal\/(?:xterm|addon-fit)\.mjs$/}, (a) => ({
            path: a.path,
            external: true,
          }));
        },
      },
    ],
  });
  assert(panelBuild.success, `panel fixture build failed: ${panelBuild.logs.join('\n')}`);
  const nativeBuild = await Bun.build({
    entrypoints: [path.join(fountain, 'web_src/js/features/extensions.js')],
    target: 'browser',
    format: 'esm',
  });
  assert(nativeBuild.success, `native host fixture build failed: ${nativeBuild.logs.join('\n')}`);
  const scripts: Record<string, Blob> = {
    '/assets/workspace-fixture.js': wsBuild.outputs[0]!,
    '/assets/drawer-fixture.js': drawerBuild.outputs[0]!,
    '/forge/-/native/assets/soda/persistent-panel.js': panelBuild.outputs[0]!,
    '/assets/native-host.js': nativeBuild.outputs[0]!,
  };
  const sodaCss: Record<string, string> = {
    '/assets/sodaspaces-drawer.css': 'frontend/spaces/sodaspaces-project.css',
    '/assets/sodaspaces-page.css': 'frontend/spaces/sodaspaces-workspace.css',
    '/assets/sodaspaces.css': 'frontend/spaces/sodaspaces.css',
    '/assets/sodaspaces-terminal.css': 'frontend/spaces/sodaspaces-terminal.css',
    '/assets/soda/forgejo/components.css': 'assets/branding/forgejo/components.css',
    '/assets/soda/forgejo/components-buttons.css': 'assets/branding/forgejo/components-buttons.css',
  };
  server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(req) {
      const url = new URL(req.url);
      if (scripts[url.pathname])
        return new Response(scripts[url.pathname], {headers: {'Content-Type': 'text/javascript'}});
      if (url.pathname === '/forge/-/native/session')
        return new Response(null, {status: 204, headers: {'X-Extension-Session-Generation': 'fixture-generation'}});
      if (url.pathname.startsWith('/forge/-/native/assets/soda/')) {
        const relative = url.pathname.slice('/forge/-/native/assets/soda/'.length);
        const canonical = relative === 'components.css' ? '/assets/soda/forgejo/components.css' : '/assets/' + relative;
        const stylesheet = sodaCss[canonical];
        if (stylesheet)
          return new Response(Bun.file(path.join(root, stylesheet)), {headers: {'Content-Type': 'text/css'}});
        if (relative.startsWith('soda-terminal/'))
          return new Response(
            Bun.file(path.join(root, '.artifacts/browser-terminal/vendor', path.basename(relative))),
            {
              headers: {'Content-Type': relative.endsWith('.mjs') ? 'text/javascript' : 'text/css'},
            }
          );
      }
      if (url.pathname === '/assets/native/container.css')
        return new Response(fountainCss('modules/container.css'), {headers: {'Content-Type': 'text/css'}});
      if (url.pathname === '/assets/native/extensions.css')
        return new Response(fountainCss('features/extensions.css'), {headers: {'Content-Type': 'text/css'}});
      if (sodaCss[url.pathname])
        return new Response(Bun.file(path.join(root, sodaCss[url.pathname]!)), {headers: {'Content-Type': 'text/css'}});
      if (url.pathname.startsWith('/assets/soda-terminal/'))
        return new Response(
          Bun.file(path.join(root, '.artifacts/browser-terminal/vendor', path.basename(url.pathname)))
        );
      if (url.pathname.startsWith('/assets/soda/forgejo/icons/'))
        return new Response(Bun.file(path.join(root, 'assets/branding/icons/octicons', path.basename(url.pathname))), {
          headers: {'Content-Type': 'image/svg+xml'},
        });
      if (url.pathname.startsWith('/assets/soda/'))
        return new Response(Bun.file(path.join(root, 'assets/branding', url.pathname.slice('/assets/soda/'.length))));
      const html = {headers: {'Content-Type': 'text/html', 'X-Extension-Session-Generation': 'fixture-generation'}};
      // Fountain page.tmpl, global scope: base/head + page-content/ui container.
      if (url.pathname === '/page')
        return new Response(
          shellHead('Spaces page') +
            `<body><nav id="navbar">native</nav><div class="full height"><div role="main" aria-label="Spaces" class="page-content extensions"><div class="ui container"><section class="extension-page extension-page-owner-presented" aria-labelledby="extension-page-title"><h1 id="extension-page-title" class="ui dividing header">Spaces</h1></section></div></div></div><aside id="extension-panels" class="extension-workspace-panels" data-extension-panels><section class="extension-workspace-panel"><main data-extension-panel></main></section></aside><footer class="page-footer">native footer</footer><script type="module" src="/assets/workspace-fixture.js"></script></body>`,
          html
        );
      // Fountain panels.tmpl via base/footer.tmpl: panels home after .full.height.
      if (url.pathname === '/forge/workspace')
        return new Response(
          shellHead('Native panel') +
            `<link rel="stylesheet" href="/forge/-/native/assets/soda/soda-terminal/xterm.css"><body data-page-data='{}'><div class="full height"><div role="main" class="page-content"><section class="extension-page extension-page-owner-presented"><h1>Workspace</h1></section></div></div><aside id="extension-panels" class="extension-workspace-panels" aria-label="Workspace panels" data-extension-panels data-app-sub-url="/forge" data-extension-session-generation="fixture-generation"><section class="extension-workspace-panel" aria-labelledby="extension-panel-soda-workspace"><h2 id="extension-panel-soda-workspace">Workspace</h2><div data-extension-panel data-extension-contract-version="1" data-app-sub-url="/forge" data-extension-id="soda" data-extension-panel-id="workspace" data-extension-owner-id="workspace" data-extension-entry="/forge/-/native/assets/soda/persistent-panel.js" data-extension-api-base="/forge/-/native/pages/soda/spaces/api/" data-extension-asset-base="/forge/-/native/assets/soda/" data-extension-session-generation="fixture-generation"></div></section></aside><footer class="page-footer">native footer</footer><script type="module" src="/forge/-/native/assets/soda/persistent-panel.js"></script></body>`,
          html
        );
      // Project settings (drawer fixture) inside the same real panel mount.
      if (url.pathname === '/settings-panel')
        return new Response(
          shellHead('Settings panel') +
            `<body><div class="full height"><div role="main" class="page-content"><p>Native content</p></div></div><aside id="extension-panels" class="extension-workspace-panels" aria-label="Workspace panels" data-extension-panels><section class="extension-workspace-panel" aria-labelledby="extension-panel-soda-workspace"><h2 id="extension-panel-soda-workspace">Workspace</h2><div data-extension-panel><main></main></div></section></aside><footer class="page-footer">native footer</footer><script type="module" src="/assets/drawer-fixture.js"></script></body>`,
          html
        );
      return new Response(null, {status: 404});
    },
  });
  browser = await browserZoomFixture();
});

after(async () => {
  await browser?.close();
  server?.stop(true);
});

async function applyZoom(page: Page, factor: 2 | 4) {
  const css = {width: Math.round(1440 / factor), height: Math.round(1000 / factor)};
  const actual = await browser.setZoom(page, factor);
  assert.deepEqual(actual, {...css, dpr: factor}, `zoom ${factor}x did not take effect`);
  return css;
}

function trackErrors(t: TestContext, page: Page) {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  t.after(async () => {
    await page.close();
    assert.deepEqual(errors, []);
  });
}

test(
  'F06 page presentation: workspace uses the retained native panels allocation at zoom 1 and 4',
  {timeout: 60000},
  async (t) => {
    const page = await browser.context.newPage();
    page.setDefaultTimeout(5000);
    trackErrors(t, page);
    await page.goto(server.url.href + 'page');
    await browser.setZoom(page, 1);
    await page.waitForFunction(() => !!window.createWorkspaceFixture);
    await page.evaluate(async () => {
      window.workspaceFixture = window.createWorkspaceFixture('page');
      await window.workspaceFixture.api.ready;
      await window.workspaceFixture.api.refresh();
    });
    await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
    await page.getByRole('button', {name: 'Terminals', exact: true}).click();
    await page.getByLabel('Workspace options', {exact: true}).click();
    await page.locator('.soda-session-list button').filter({hasText: 'Build'}).click();
    await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
    const at1x = await page.evaluate(() => {
      const panel = document.querySelector('.extension-workspace-panel')!;
      const container = panel.getBoundingClientRect();
      const padding = getComputedStyle(panel);
      const containerWidth = container.width - parseFloat(padding.paddingLeft) - parseFloat(padding.paddingRight);
      const mount = document.querySelector('[data-extension-panel]')!.getBoundingClientRect();
      const terminal = document.querySelector('.soda-workspace-terminal:not([hidden])')!.getBoundingClientRect();
      return {
        container: {w: Math.round(containerWidth), h: Math.round(container.height)},
        mount: {w: Math.round(mount.width), h: Math.round(mount.height)},
        terminal: {w: Math.round(terminal.width), h: Math.round(terminal.height)},
        hOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
        hostHeight: (document.querySelector('[data-extension-panel]') as HTMLElement).style.height,
      };
    });
    console.log('F06 page @1x', JSON.stringify(at1x));
    // The retained panel supplies the available width after its native padding.
    assert(
      at1x.container.w <= 1440 && at1x.container.w >= 1380,
      `native container width unexpected: ${at1x.container.w}`
    );
    assert(Math.abs(at1x.mount.w - at1x.container.w) <= 2, 'Soda mount does not bind to the native container width');
    assert(at1x.hOverflow <= 1, 'page overflows horizontally inside the native container');
    assert.equal(at1x.hostHeight, '', 'Soda must not invent host height on the page mount');
    const css = await applyZoom(page, 4);
    await page.waitForFunction(
      () =>
        (document.querySelector('.soda-workspace-terminal:not([hidden])')?.getBoundingClientRect().height ?? 0) >= 44,
      null,
      {timeout: 5000}
    );
    const at4x = await page.evaluate(() => {
      const terminal = document.querySelector('.soda-workspace-terminal:not([hidden])')!.getBoundingClientRect();
      return {
        terminalH: Math.round(terminal.height),
        hOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      };
    });
    console.log('F06 page @4x', JSON.stringify(at4x), 'css', JSON.stringify(css));
    assert(at4x.terminalH >= 44, `terminal collapsed inside native container at zoom 4x: ${at4x.terminalH}`);
    assert(at4x.hOverflow <= 1, 'page overflows horizontally inside the native container at zoom 4x');
    for (const name of ['New terminal', 'Project settings']) {
      const control = page.getByRole('button', {name, exact: true});
      await control.scrollIntoViewIfNeeded();
      assert(await control.isVisible(), `${name} unreachable in native container at zoom 4x`);
    }
  }
);

test('F06 panel container: Soda binds to the host-sized panels home at zoom 1 and 4', {timeout: 60000}, async (t) => {
  const page = await browser.context.newPage();
  page.setDefaultTimeout(5000);
  trackErrors(t, page);
  await page.goto(server.url.href + 'forge/workspace');
  await browser.setZoom(page, 1);
  await page.locator('[data-extension-panel] soda-spaces').waitFor();
  const dims = await page.evaluate(() => {
    const aside = document.querySelector('.extension-workspace-panels')!.getBoundingClientRect();
    const section = document.querySelector('.extension-workspace-panel')!.getBoundingClientRect();
    const mount = document.querySelector('[data-extension-panel]')!.getBoundingClientRect();
    const root = document.querySelector('[data-extension-panel]') as HTMLElement;
    return {
      aside: {w: Math.round(aside.width), h: Math.round(aside.height)},
      section: {w: Math.round(section.width), h: Math.round(section.height)},
      mount: {w: Math.round(mount.width), h: Math.round(mount.height)},
      inlineHeight: root.style.height,
      hOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  });
  console.log('F06 panel @1x', JSON.stringify(dims));
  assert(dims.hOverflow <= 1, 'panel page overflows horizontally');
  assert(dims.mount.w > 0 && dims.mount.h > 0, 'panel mount has no size');
  assert.equal(dims.inlineHeight, '', 'Soda panel must not invent host height');
  // Current A04/H-C source: the visible
  // panels home fills the remaining viewport column (flex:1, 24rem floor)
  // and the mount binds the height:100% Soda chain against it.
  assert(dims.aside.h >= 900, `panels home not host-sized: ${dims.aside.h}px`);
  assert(dims.mount.h >= 800, `panel mount not host-sized: ${dims.mount.h}px`);
  await page.getByRole('button', {name: 'team/repo', exact: true}).click();
  await page.getByRole('button', {name: 'Terminals', exact: true}).click();
  await page.getByLabel('Workspace options', {exact: true}).click();
  await page.locator('.soda-session-list button').filter({hasText: 'Build'}).click();
  await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor({state: 'attached'});
  const at1x = await page.evaluate(() => {
    const term = document.querySelector('.soda-workspace-terminal:not([hidden])');
    const mount = document.querySelector('[data-extension-panel]')!.getBoundingClientRect();
    return {
      mountH: Math.round(mount.height),
      terminalH: term ? Math.round(term.getBoundingClientRect().height) : -1,
      connected: !!document.querySelector('.soda-workspace-terminal:not([hidden]) .is-connected'),
      inlineHeight: (document.querySelector('[data-extension-panel]') as HTMLElement).style.height,
      hOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  });
  console.log('F06 panel terminal @1x', JSON.stringify(at1x));
  assert(at1x.connected, 'terminal did not connect inside the host-sized panel container');
  assert(at1x.mountH >= 800, `panel mount not host-sized with terminal open: ${at1x.mountH}px`);
  assert(at1x.terminalH >= 700, `terminal not usable inside host-sized panel: ${at1x.terminalH}px`);
  assert.equal(at1x.inlineHeight, '', 'Soda panel must not invent host height with terminal open');
  assert(at1x.hOverflow <= 1, 'panel page overflows horizontally with terminal open');
  const css = await applyZoom(page, 4);
  await page.waitForFunction(
    () => (document.querySelector('.soda-workspace-terminal:not([hidden])')?.getBoundingClientRect().height ?? 0) >= 44,
    null,
    {timeout: 5000}
  );
  const at4x = await page.evaluate(() => {
    const term = document.querySelector('.soda-workspace-terminal:not([hidden])');
    const mount = document.querySelector('[data-extension-panel]')!.getBoundingClientRect();
    const aside = document.querySelector('.extension-workspace-panels')!.getBoundingClientRect();
    return {
      aside: {w: Math.round(aside.width), h: Math.round(aside.height)},
      mount: {w: Math.round(mount.width), h: Math.round(mount.height)},
      terminalH: term ? Math.round(term.getBoundingClientRect().height) : -1,
      connected: !!document.querySelector('.soda-workspace-terminal:not([hidden]) .is-connected'),
      hOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    };
  });
  console.log('F06 panel @4x', JSON.stringify(at4x), 'css', JSON.stringify(css));
  // Compact floor: 18rem min-height keeps a usable mount at 360x250 CSS.
  assert(at4x.connected, 'terminal did not stay connected in panel container at zoom 4x');
  assert(at4x.mount.h >= 200, `panel mount collapsed at zoom 4x: ${at4x.mount.h}px`);
  assert(at4x.terminalH >= 100, `terminal not usable in panel container at zoom 4x: ${at4x.terminalH}px`);
  assert(at4x.hOverflow <= 1, 'panel page overflows horizontally at zoom 4x');
  for (const name of ['Projects', 'New terminal']) {
    const control = page.getByRole('button', {name, exact: true});
    await control.scrollIntoViewIfNeeded();
    assert(await control.isVisible(), `${name} unreachable in panel container at zoom 4x`);
  }
});

test(
  'F06 page presentation with a growing host: work content scrolls inside the workspace',
  {timeout: 60000},
  async (t) => {
    const page = await browser.context.newPage();
    page.setDefaultTimeout(5000);
    trackErrors(t, page);
    await page.goto(server.url.href + 'page');
    await browser.setZoom(page, 1);
    await page.waitForFunction(() => !!window.createWorkspaceFixture);
    await page.evaluate(async () => {
      window.workspaceFixture = window.createWorkspaceFixture('page');
      await window.workspaceFixture.api.ready;
      await window.workspaceFixture.api.refresh();
    });
    await page.getByRole('button', {name: 'alice/Alpha', exact: true}).click();
    await page.locator('soda-factory-feed').first().waitFor();
    // Installed hosts may let the page body grow with content instead of
    // supplying definite space; the mount's viewport-height fallback must
    // still engage the workspace's own scroller for tall work content.
    await page.addStyleTag({content: 'html,body{height:auto}body{min-height:100dvh}'});
    const grown = await page.evaluate(() => {
      const section = document.querySelector('soda-factory-feed section')!;
      for (let i = 0; i < 60; i++) {
        const p = document.createElement('p');
        p.textContent = `Tall feed content ${i + 1}`;
        section.append(p);
      }
      const mount = document.querySelector('.soda-spaces-page-mount') as HTMLElement;
      const scroller = document.querySelector('.soda-project-content') as HTMLElement;
      return {
        mountH: mount.clientHeight,
        viewportH: innerHeight,
        clientH: scroller.clientHeight,
        scrollH: scroller.scrollHeight,
      };
    });
    console.log('F06 growing host', JSON.stringify(grown));
    assert(grown.mountH <= grown.viewportH, `page mount grew past the viewport: ${grown.mountH}px`);
    assert(grown.scrollH > grown.clientH, 'work content does not overflow its own scroller under a growing host');
    const feedBox = await page.locator('soda-factory-feed').first().boundingBox();
    assert(feedBox, 'feed has no layout box');
    await page.mouse.move(feedBox.x + 120, Math.min(feedBox.y + 80, grown.viewportH - 40));
    await page.mouse.wheel(0, 1200);
    await page.waitForFunction(() => (document.querySelector('.soda-project-content') as HTMLElement).scrollTop > 0);
  }
);

test(
  'F06 settings in panel container: host-sized at zoom 1, tabs reachable at zoom 4x',
  {timeout: 60000},
  async (t) => {
    const page = await browser.context.newPage();
    page.setDefaultTimeout(5000);
    trackErrors(t, page);
    await page.goto(server.url.href + 'settings-panel');
    await browser.setZoom(page, 1);
    await page.waitForFunction(() => typeof window.createDrawerFixture === 'function');
    await page.evaluate(async () => {
      window.drawerFixture = window.createDrawerFixture({});
      window.drawerFixture.setReply(async () => null);
      window.drawerFixture.api.setPresentation('settings');
      await window.drawerFixture.api.refresh();
    });
    await page.getByRole('heading', {name: 'AI accounts'}).waitFor();
    const at1x = await page.evaluate(() => {
      const aside = document.querySelector('.extension-workspace-panels')!.getBoundingClientRect();
      const mount = document.querySelector('[data-extension-panel]')!.getBoundingClientRect();
      return {
        aside: {w: Math.round(aside.width), h: Math.round(aside.height)},
        mount: {w: Math.round(mount.width), h: Math.round(mount.height)},
        inlineHeight: (document.querySelector('[data-extension-panel]') as HTMLElement).style.height,
        hOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      };
    });
    console.log('F06 settings-panel @1x', JSON.stringify(at1x));
    assert(at1x.hOverflow <= 1, 'settings panel page overflows horizontally');
    assert.equal(at1x.inlineHeight, '', 'Soda settings panel must not invent host height');
    assert(at1x.aside.h >= 900, `settings panels home not host-sized: ${at1x.aside.h}px`);
    assert(at1x.mount.h >= 800, `settings panel mount not host-sized: ${at1x.mount.h}px`);
    const css = await applyZoom(page, 4);
    console.log('F06 settings-panel css', JSON.stringify(css));
    for (const tab of ['Overview', 'Access', 'Network', 'Factory'] as const) {
      await page.getByRole('tab', {name: tab, exact: true}).click();
      const panel = page.locator('[role=tabpanel]:visible');
      await panel.waitFor();
      const geo = await panel.evaluate((n) => ({
        h: n.getBoundingClientRect().height,
        hOverflow: n.scrollWidth - n.clientWidth,
      }));
      console.log(`F06 settings tab ${tab} @4x`, JSON.stringify(geo));
      assert(geo.h >= 44, `${tab} panel squeezed inside native panel container at zoom 4x: ${geo.h}`);
      assert(geo.hOverflow <= 1, `${tab} panel overflows horizontally in native panel container at zoom 4x`);
      const first = panel.locator('button:not([disabled]), [href], input:not([disabled]), select, textarea').first();
      await first.scrollIntoViewIfNeeded();
      assert(await first.isVisible(), `${tab} first control unreachable in native panel container at zoom 4x`);
    }
  }
);
