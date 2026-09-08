import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { chromium } from 'playwright';

// Explicit local opt-in. Native CSS plus the complete candidate CSS cascade,
// with small native markup contracts; no account, fixture or provider writes.
const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;
test('expanded components preserve native state and layout boundaries', { skip: !origin }, async t => {
  assert.equal(origin, 'http://localhost:3300');
  const header = await readFile(new URL('../../appliance/forgejo/templates/custom/header.tmpl', import.meta.url), 'utf8');
  const files = [...header.matchAll(/\/soda\/forgejo\/([^?]+\.css)\?v=/g)].map(([, name]) => name);
  const styles = await Promise.all(files.map(name => readFile(new URL(`../../assets/branding/forgejo/${name}`, import.meta.url), 'utf8')));
  const palette = await readFile(new URL('../../assets/branding/theme/palette.css', import.meta.url), 'utf8');
  const browser = await chromium.launch({ channel: 'chrome', headless: true, chromiumSandbox: true });
  try {
    const page = await browser.newPage({ viewport: { width: 1654, height: 1000 } });
    async function render(markup, theme = 'light') {
      await page.mouse.move(1600,990);
      await page.setContent(`<link rel="stylesheet" href="${origin}/assets/css/index.css">
        <link rel="stylesheet" href="${origin}/assets/css/theme-forgejo-${theme}.css">
        <style>${palette}\n${styles.join('\n').replace(/@import[^;]+;/g, '')}</style>${markup}`);
    }

    await t.test('form focus and errors survive the shared input cascade in both themes', async () => {
      for (const theme of ['light', 'dark']) {
        await render(`<main class="soda-page"><form class="ui form soda-form">
          <div class="field"><input id="normal"></div>
          <div class="field error"><input id="field-input"><textarea id="field-textarea"></textarea>
            <div id="field-dropdown" class="ui selection dropdown"></div></div>
          <input id="input-error" class="error"><textarea id="textarea-error" class="error"></textarea>
        </form><div id="native-error" style="border:1px solid var(--color-error-border)"></div>
          <div id="focus-color" style="border:1px solid var(--soda-page-link)"></div></main>`, theme);
        const borders = await page.evaluate(() => Object.fromEntries(
          [...document.querySelectorAll('[id]')].map(el => [el.id, getComputedStyle(el).borderTopColor])));
        for (const id of ['field-input', 'field-textarea', 'field-dropdown', 'input-error', 'textarea-error']) {
          assert.equal(borders[id], borders['native-error'], `${theme}: ${id} must retain the native error border`);
        }
        assert.notEqual(borders.normal, borders['native-error']);
        await page.locator('#normal').focus();
        assert.equal(await page.locator('#normal').evaluate(el => getComputedStyle(el).borderTopColor), borders['focus-color']);
        await page.locator('#field-input').focus();
        assert.equal(await page.locator('#field-input').evaluate(el => getComputedStyle(el).borderTopColor), borders['native-error']);
      }
    });

    await t.test('repository toolbar aligns native control variants and joins clone edges', async () => {
      for (const theme of ['light', 'dark']) for (const width of [1440, 390, 320]) {
        await page.setViewportSize({ width, height: 1000 });
        await render(`<main class="soda-page soda-code"><div class="ui container soda-page-container">
          <div class="repo-button-row soda-toolbar"><div class="button-sequence">
            <div class="soda-code-ref-selector"><div class="ui dropdown custom"><button id="branch" class="branch-dropdown-button ui basic small compact button">main</button></div></div>
            <a id="new-pull-request" class="ui compact basic button">↗</a>
            <a id="find" class="ui compact basic button">Find a file</a>
            <button id="add" class="ui dropdown basic compact button">Add file</button>
          </div><div class="clone-panel ui action tiny input">
            <button id="protocol" class="ui small primary button">HTTP</button>
            <input id="url" class="soda-code-clone-url" value="http://localhost:3300/alice/activity-workbench" readonly>
            <button id="copy" class="ui small icon button">⧉</button>
            <button id="more" class="ui small dropdown icon button">…</button><script type="application/json">{}</script>
          </div></div></div></main>`, theme);
        const controls = await page.locator('[id]').evaluateAll(els => els.map(el => ({id:el.id,height:el.getBoundingClientRect().height,top:getComputedStyle(el).borderTopRightRadius,left:getComputedStyle(el).borderTopLeftRadius})));
        for (const c of controls) assert.equal(c.height,44,`${theme}/${width}: ${c.id}`);
        for (const id of ['protocol','url','copy']) assert.equal(controls.find(c=>c.id===id).top,'0px');
        assert.equal(controls.find(c=>c.id==='more').top,'6px');
        assert.equal(controls.find(c=>c.id==='protocol').left,'6px');
        assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth > innerWidth),false, `${theme}/${width}: toolbar overflow`);
      }
      await page.setViewportSize({width:1654,height:1000});
    });

    await t.test('shared buttons use compact type and quiet secondary surfaces', async () => {
      for (const theme of ['light','dark']) {
        await render(`<main class="soda-page"><button id="basic-neutral" class="ui basic button">Add file</button><button id="neutral" class="ui button">Cancel</button><a id="secondary" class="button secondary" href="#">Add file</a><button id="primary" class="ui primary button">Save</button><button id="compact" class="ui compact button">Filter</button><div class="ui buttons"><button id="joined-first" class="ui button">One</button><button id="joined-last" class="ui button">Two</button></div><form class="ui form soda-p-form"><button id="form-save" class="primary button">Save profile</button></form></main>`, theme);
        for (const id of ['basic-neutral','neutral','secondary','primary','form-save']) {
          const style=await page.locator('#'+id).evaluate(el=>({height:el.getBoundingClientRect().height,radius:getComputedStyle(el).borderTopLeftRadius,font:getComputedStyle(el).fontSize}));
          assert.equal(style.height,44);assert.equal(style.radius,'6px');assert.equal(style.font,'14px');
        }
        assert.equal(await page.locator('#compact').evaluate(el=>el.getBoundingClientRect().height),36);
        for (const id of ['neutral','basic-neutral']) assert.equal(await page.locator('#'+id).evaluate(el=>getComputedStyle(el).backgroundColor),'rgba(0, 0, 0, 0)');
        await page.locator('#neutral').hover();
        await page.waitForFunction(()=>getComputedStyle(document.querySelector('#neutral')).backgroundColor!=='rgba(0, 0, 0, 0)');
        assert.equal(await page.locator('#joined-first').evaluate(el=>getComputedStyle(el).borderTopRightRadius),'0px');
        assert.equal(await page.locator('#joined-last').evaluate(el=>getComputedStyle(el).borderTopLeftRadius),'0px');
      }
    });

    await t.test('settings panels do not give native tables panel padding', async () => {
      await render(`<main class="soda-page soda-admin"><section class="admin-setting-content">
        <h4 class="ui top attached header">Notices</h4>
        <table id="settings-table" class="ui attached segment"><tbody><tr><td>Notice</td></tr></tbody></table>
        <div id="settings-panel" class="ui attached segment">Panel</div>
      </section></main><table id="native-table" class="ui attached segment"><tbody><tr><td>Reference</td></tr></tbody></table>`);
      const padding = await page.evaluate(() => Object.fromEntries(
        [...document.querySelectorAll('[id]')].map(el => [el.id, getComputedStyle(el).paddingTop])));
      assert.equal(padding['settings-table'], padding['native-table']);
      assert.equal(padding['settings-panel'], '0px');
    });

    await t.test('native fluid repository views keep their wider canvas', async () => {
      await render(`<main class="page-content repository"><div class="soda-page-marker soda-repository-header"></div>
        <div id="ordinary" class="ui container">Ordinary content</div>
        <div id="fluid" class="ui container fluid padded">Native diff / blame canvas</div>
        <div id="pull-files" class="ui container fluid padded soda-pull-files-container">Pull files</div>
      </main>`);
      const widths = await page.evaluate(() => Object.fromEntries(
        [...document.querySelectorAll('[id]')].map(el => [el.id, el.getBoundingClientRect().width])));
      assert.equal(widths.ordinary, 1120);
      assert.equal(widths['pull-files'], 1440);
      assert.equal(await page.locator('#pull-files').evaluate(el => el.getBoundingClientRect().left), (1654 - 1440) / 2);
      assert(widths.fluid > 1440, 'native fluid canvas must not inherit the ordinary content cap');
      await page.setViewportSize({ width: 390, height: 844 });
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.setViewportSize({ width: 1654, height: 1000 });
    });

    await t.test('repository-context status pages bound native unit navigation at 390px', async () => {
      await page.setViewportSize({ width: 390, height: 844 });
      await render(`<main class="page-content ui repository soda-page soda-status">
        <div class="secondary-nav soda-page-marker soda-repository-header">
          <div class="ui container"><div class="repo-header"><div class="flex-item tw-items-center">
            <div class="flex-item-main"><div class="flex-item-title">alice/activity-workbench</div></div>
          </div></div></div>
          <overflow-menu class="ui container secondary pointing tabular top attached borderless menu">
            <div class="overflow-menu-items">
              <a class="item">Code</a><a class="item">Issues <span class="ui small label">13</span></a>
              <a class="item">Pull requests</a><a class="item">Projects</a><a class="item">Releases</a>
              <a class="item">Packages</a><a class="item">Wiki</a><a class="item">Activity</a><a class="item">Actions</a>
            </div>
          </overflow-menu>
        </div>
        <div id="status-card" class="ui container center soda-page-container soda-status-card">
          <h1 class="error-code">404</h1><p>The page you are trying to reach is unavailable.</p>
        </div>
      </main>`);
      const bounds = await page.locator('#status-card').evaluate(card => ({
        pageWidth: document.documentElement.scrollWidth,
        viewportWidth: innerWidth,
        cardLeft: card.getBoundingClientRect().left,
        cardRight: card.getBoundingClientRect().right,
      }));
      assert.equal(bounds.pageWidth, bounds.viewportWidth);
      assert(bounds.cardLeft >= 16, 'status card must retain its mobile gutter');
      assert(bounds.cardRight <= bounds.viewportWidth - 16 + 1, 'status card must remain inside the viewport');
      await page.setViewportSize({ width: 1654, height: 1000 });
    });

    await t.test('settings search, row and modal forms keep native control sizes', async () => {
      await render(`<main class="soda-page soda-native-forms soda-settings"><section class="user-setting-content">
        <form class="ui form"><input id="principal"></form>
        <form class="ui form ignore-dirty"><input id="search"></form>
        <div class="flex-item"><form class="ui form"><input id="row"></form></div>
        <dialog open><form class="ui form"><input id="modal"></form></dialog>
      </section></main><form class="ui form"><input id="native"></form>`);
      const heights = await page.evaluate(() => Object.fromEntries(
        [...document.querySelectorAll('input')].map(el => [el.id, getComputedStyle(el).minHeight])));
      assert.equal(heights.principal, '44px');
      for (const id of ['search', 'row', 'modal']) assert.equal(heights[id], heights.native, id);
    });

    await t.test('primary action colors preserve native button states in both themes', async () => {
      for (const theme of ['light', 'dark']) {
        await render(`<main class="soda-page">
          <div id="action-color" style="background:var(--soda-page-action)"></div>
          <div id="red-color" style="background:var(--color-red)"></div>
          <div id="green-color" style="background:var(--color-green)"></div>
          <div id="transparent-color" style="background:transparent"></div>
          <section class="soda-settings"><button id="settings-primary" class="ui primary button">Save</button></section>
          <form class="ui form soda-form"><button id="form-primary" class="ui primary button">Save</button></form>
          <section class="page-content repository"><div class="soda-page-marker soda-repository-header"></div>
            <button id="repository-primary" class="primary button">New issue</button>
          </section>
          <a id="anchor-primary" class="ui primary button" href="#">New issue</a>
          <button id="tiny-primary" class="ui primary tiny button">Tiny</button>
          <button id="tiny-native" class="ui tiny button">Tiny</button>
          <button id="disabled-primary" class="ui primary tiny disabled button">Disabled</button>
          <button id="loading-primary" class="ui primary tiny loading button">Loading</button>
          <button id="red-primary" class="ui primary red tiny button">Delete</button>
          <button id="positive-primary" class="ui primary positive tiny button">Approve</button>
          <button id="basic-primary" class="ui primary basic tiny button">Basic</button>
        </main>`, theme);
        const states = await page.evaluate(() => Object.fromEntries(
          [...document.querySelectorAll('[id]')].map(el => {
            const style = getComputedStyle(el);
            return [el.id, {
              background: style.backgroundColor,
              color: style.color,
              cursor: style.cursor,
              fontSize: style.fontSize,
              opacity: style.opacity,
              paddingBlock: `${style.paddingTop} ${style.paddingBottom}`,
              pointerEvents: style.pointerEvents,
            }];
          })));
        for (const id of ['settings-primary', 'form-primary', 'repository-primary', 'tiny-primary', 'disabled-primary', 'loading-primary']) {
          assert.equal(states[id].background, states['action-color'].background, `${theme}: ${id} must use the Soda action color`);
        }
        assert.equal(states['anchor-primary'].color, states['form-primary'].color, 'primary anchor text remains legible');
        assert.notEqual(states['anchor-primary'].color, states['anchor-primary'].background);
        assert.equal(states['tiny-primary'].fontSize, states['tiny-native'].fontSize);
        assert.equal(states['tiny-primary'].paddingBlock, states['tiny-native'].paddingBlock);
        assert.notEqual(states['disabled-primary'].opacity, '1');
        assert.equal(states['disabled-primary'].pointerEvents, 'none');
        assert.equal(states['loading-primary'].color, 'rgba(0, 0, 0, 0)');
        assert.equal(states['loading-primary'].cursor, 'default');
        assert.equal(states['red-primary'].background, states['red-color'].background);
        assert.equal(states['positive-primary'].background, states['green-color'].background);
        assert.equal(states['basic-primary'].background, states['transparent-color'].background);
        assert.notEqual(states['basic-primary'].background, states['action-color'].background);
      }
    });

    await t.test('owner package pages opt their shared native profile card into profile styling', async () => {
      await render(`<div class="soda-page">
        <aside class="soda-profile-card-context"><div id="profile-avatar-card" class="ui card">
          <div class="content profile-avatar-name"><span class="header">Alice</span><span class="username">alice</span></div>
          <div class="actions button-row"><button id="shared-follow" class="primary button">Follow</button></div>
        </div></aside>
        <div id="profile-surface" style="background:var(--soda-page-surface)"></div>
        <div id="profile-action" style="background:var(--soda-page-action)"></div>
      </div>`);
      const colors = await page.evaluate(() => Object.fromEntries(
        [...document.querySelectorAll('[id]')].map(el => [el.id, getComputedStyle(el).backgroundColor])));
      assert.equal(colors['profile-avatar-card'], 'rgba(0, 0, 0, 0)');
      assert.equal(colors['shared-follow'], colors['profile-action']);
    });

    await t.test('package cleanup preview contains a wide native table at 390px', async () => {
      await page.setViewportSize({ width: 390, height: 844 });
      await render(`<main class="soda-page soda-packages"><div class="ui container soda-page-container">
        <section class="soda-package-section soda-package-cleanup-preview">
          <h4 class="ui top attached header soda-package-section-heading">Cleanup preview</h4>
          <div class="ui attached segment soda-package-section-body">Versions selected by the native rule</div>
          <div id="preview" class="ui attached table segment soda-package-preview-table">
            <table class="ui very basic striped table unstackable"><thead><tr>
              <th>Version</th><th>Published</th><th>Publisher</th><th>Repository</th><th>Architecture</th><th id="last-heading">Blob size</th>
            </tr></thead><tbody><tr>
              <td>release-candidate-with-a-long-native-version</td><td>2026-09-08</td><td>alice</td><td>shared-project</td><td>linux-amd64</td><td id="last-cell">123456789 bytes</td>
            </tr></tbody></table>
          </div>
        </section></div></main>`);
      const before = await page.locator('#preview').evaluate(wrapper => ({
        clientWidth: wrapper.clientWidth,
        scrollWidth: wrapper.scrollWidth,
        pageOverflow: document.documentElement.scrollWidth > innerWidth,
        lastRight: wrapper.querySelector('#last-cell').getBoundingClientRect().right,
        wrapperRight: wrapper.getBoundingClientRect().right,
      }));
      assert.equal(before.pageOverflow, false);
      assert(before.scrollWidth > before.clientWidth, 'native cleanup table must overflow its bounded wrapper');
      assert(before.lastRight > before.wrapperRight, 'the final native column should initially be beyond the viewport');
      const after = await page.locator('#preview').evaluate(wrapper => {
        wrapper.scrollLeft = wrapper.scrollWidth;
        const last = wrapper.querySelector('#last-cell').getBoundingClientRect();
        const bounds = wrapper.getBoundingClientRect();
        return { scrollLeft: wrapper.scrollLeft, lastRight: last.right, wrapperRight: bounds.right };
      });
      assert(after.scrollLeft > 0, 'cleanup preview must accept horizontal scrolling');
      assert(after.lastRight <= after.wrapperRight + 1, 'the final native column must be reachable');
      await page.setViewportSize({ width: 1654, height: 1000 });
    });
  } finally { await browser.close(); }
});
