import assert from 'node:assert/strict';
import {test} from 'node:test';
import {openComponentBrowser, origin} from './fixtures/component-browser';

test('layout components preserve native state and layout boundaries', {skip: !origin}, async (t) => {
  const {page, render, close} = await openComponentBrowser();
  try {
    await t.test('page and compact empty states stay open and fit narrow layouts', async () => {
      for (const theme of ['light', 'dark']) {
        for (const width of [1440, 390, 320]) {
          await page.setViewportSize({width, height: 1000});
          await render(
            `<main class="soda-page"><section class="soda-empty soda-empty--page"><div class="soda-empty-symbol" aria-hidden="true"><svg></svg></div><h2 class="soda-empty-title">There are no blocked users.</h2></section><section class="soda-empty soda-empty--compact"><div class="soda-empty-symbol" aria-hidden="true"><svg></svg></div><h2 class="soda-empty-title">There are no deploy keys yet.</h2></section></main>`,
            theme
          );
          const state = await page.evaluate(() => {
            const full = document.querySelector('.soda-empty--page');
            const compact = document.querySelector('.soda-empty--compact');
            const fullSymbol = full?.querySelector('.soda-empty-symbol');
            const compactSymbol = compact?.querySelector('.soda-empty-symbol');
            if (!full || !compact || !fullSymbol || !compactSymbol) throw Error('Missing empty state markup');
            return {
              border: getComputedStyle(full).borderTopWidth,
              background: getComputedStyle(full).backgroundColor,
              fullIcon: fullSymbol.getBoundingClientRect().width,
              compactIcon: compactSymbol.getBoundingClientRect().width,
              fits: document.documentElement.scrollWidth <= innerWidth,
            };
          });
          assert.deepEqual(state, {
            border: '0px',
            background: 'rgba(0, 0, 0, 0)',
            fullIcon: 32,
            compactIcon: 32,
            fits: true,
          });
        }
      }
      await page.setViewportSize({width: 1654, height: 1000});
    });

    await t.test('native fluid repository views keep their wider canvas', async () => {
      await render(`<main class="page-content repository"><div class="soda-page-marker soda-repository-header"></div>
        <div id="ordinary" class="ui container">Ordinary content</div>
        <div id="fluid" class="ui container fluid padded">Native diff / blame canvas</div>
        <div id="pull-files" class="ui container fluid padded soda-pull-files-container">Pull files</div>
      </main>`);
      const widths = await page.evaluate(() =>
        Object.fromEntries(
          [...document.querySelectorAll('[id]')].map((el) => [el.id, el.getBoundingClientRect().width])
        )
      );
      assert.equal(widths.ordinary, 1120);
      assert.equal(widths['pull-files'], 1440);
      assert.equal(
        await page.locator('#pull-files').evaluate((el) => el.getBoundingClientRect().left),
        (1654 - 1440) / 2
      );
      assert(widths.fluid !== undefined);
      assert(widths.fluid > 1440, 'native fluid canvas must not inherit the ordinary content cap');
      await page.setViewportSize({width: 390, height: 844});
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
      await page.setViewportSize({width: 1654, height: 1000});
    });

    await t.test('repository-context status pages bound native unit navigation at 390px', async () => {
      await page.setViewportSize({width: 390, height: 844});
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
      const bounds = await page.locator('#status-card').evaluate((card) => ({
        pageWidth: document.documentElement.scrollWidth,
        viewportWidth: innerWidth,
        cardLeft: card.getBoundingClientRect().left,
        cardRight: card.getBoundingClientRect().right,
      }));
      assert.equal(bounds.pageWidth, bounds.viewportWidth);
      assert(bounds.cardLeft >= 16, 'status card must retain its mobile gutter');
      assert(bounds.cardRight <= bounds.viewportWidth - 16 + 1, 'status card must remain inside the viewport');
      await page.setViewportSize({width: 1654, height: 1000});
    });

    await t.test('owner package pages opt their shared native profile card into profile styling', async () => {
      await render(`<div class="soda-page">
        <aside class="soda-profile-card-context"><div id="profile-avatar-card" class="ui card">
          <div class="content profile-avatar-name"><span class="header">Alice</span><span class="username">alice</span></div>
          <div class="actions button-row"><button id="shared-follow" class="primary button">Follow</button></div>
        </div></aside>
        <div id="profile-surface" style="background:var(--soda-page-surface)"></div>
        <div id="profile-action" style="background:var(--soda-button-primary-bg)"></div>
      </div>`);
      const colors = await page.evaluate(() =>
        Object.fromEntries(
          [...document.querySelectorAll('[id]')].map((el) => [el.id, getComputedStyle(el).backgroundColor])
        )
      );
      assert.equal(colors['profile-avatar-card'], 'rgba(0, 0, 0, 0)');
      assert.equal(colors['shared-follow'], colors['profile-action']);
    });

    await t.test('package cleanup preview contains a wide native table at 390px', async () => {
      await page.setViewportSize({width: 390, height: 844});
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
      const before = await page.locator('#preview').evaluate((wrapper) => {
        const last = wrapper.querySelector('#last-cell');
        if (!last) throw Error('Missing last table cell');
        return {
          clientWidth: wrapper.clientWidth,
          scrollWidth: wrapper.scrollWidth,
          pageOverflow: document.documentElement.scrollWidth > innerWidth,
          lastRight: last.getBoundingClientRect().right,
          wrapperRight: wrapper.getBoundingClientRect().right,
        };
      });
      assert.equal(before.pageOverflow, false);
      assert(before.scrollWidth > before.clientWidth, 'native cleanup table must overflow its bounded wrapper');
      assert(before.lastRight > before.wrapperRight, 'the final native column should initially be beyond the viewport');
      const after = await page.locator('#preview').evaluate((wrapper) => {
        wrapper.scrollLeft = wrapper.scrollWidth;
        const cell = wrapper.querySelector('#last-cell');
        if (!cell) throw Error('Missing last table cell');
        const last = cell.getBoundingClientRect();
        const bounds = wrapper.getBoundingClientRect();
        return {scrollLeft: wrapper.scrollLeft, lastRight: last.right, wrapperRight: bounds.right};
      });
      assert(after.scrollLeft > 0, 'cleanup preview must accept horizontal scrolling');
      assert(after.lastRight <= after.wrapperRight + 1, 'the final native column must be reachable');
      await page.setViewportSize({width: 1654, height: 1000});
    });
  } finally {
    await close();
  }
});
