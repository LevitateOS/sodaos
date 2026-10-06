import assert from 'node:assert/strict';
import {test} from 'node:test';
import {openComponentBrowser, origin} from './fixtures/component-browser';

test('toolbar components preserve native state and layout boundaries', {skip: !origin}, async (t) => {
  const {page, render, close} = await openComponentBrowser();
  try {
    await t.test('repository toolbar aligns native control variants and joins clone edges', async () => {
      for (const theme of ['light', 'dark'])
        for (const width of [1440, 960, 800, 720, 390, 320]) {
          await page.setViewportSize({width, height: 1000});
          await render(
            `<main class="soda-page soda-code"><div class="ui container soda-page-container"><div class="soda-repo-main">
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
          </div></div></div></div></main>`,
            theme
          );
          const controls = await page.locator('[id]').evaluateAll((els) =>
            els.map((el) => ({
              id: el.id,
              height: el.getBoundingClientRect().height,
              top: getComputedStyle(el).borderTopRightRadius,
              left: getComputedStyle(el).borderTopLeftRadius,
            }))
          );
          for (const c of controls) assert.equal(c.height, 44, `${theme}/${width}: ${c.id}`);
          for (const id of ['protocol', 'url', 'copy']) assert.equal(controls.find((c) => c.id === id)?.top, '0px');
          assert.equal(controls.find((c) => c.id === 'more')?.top, '0px');
          assert.equal(controls.find((c) => c.id === 'protocol')?.left, '0px');
          if (width <= 1000) {
            const rows = await page.evaluate(() => {
              const toolbar = document.querySelector('.repo-button-row.soda-toolbar');
              const tools = toolbar?.querySelector('.button-sequence');
              const clone = toolbar?.querySelector('.clone-panel');
              if (!toolbar || !tools || !clone) throw Error('Missing repository toolbar groups');
              return {
                toolbarWidth: toolbar.getBoundingClientRect().width,
                cloneWidth: clone.getBoundingClientRect().width,
                toolsBottom: tools.getBoundingClientRect().bottom,
                cloneTop: clone.getBoundingClientRect().top,
              };
            });
            assert.equal(rows.cloneWidth, rows.toolbarWidth, `${theme}/${width}: clone row width`);
            assert(rows.cloneTop >= rows.toolsBottom, `${theme}/${width}: clone row placement`);
          }
          assert.equal(
            await page.evaluate(() => document.documentElement.scrollWidth > innerWidth),
            false,
            `${theme}/${width}: toolbar overflow`
          );
        }
      await page.setViewportSize({width: 1654, height: 1000});
    });

    await t.test('shared buttons use compact type and quiet secondary surfaces', async () => {
      for (const theme of ['light', 'dark']) {
        await render(
          `<main class="soda-page"><button id="basic-neutral" class="ui basic button">Add file</button><button id="neutral" class="ui button">Cancel</button><a id="secondary" class="button secondary" href="#">Add file</a><button id="primary" class="ui primary button">Save</button><button id="compact" class="ui compact button">Filter</button><div class="ui buttons"><button id="joined-first" class="ui button">One</button><button id="joined-last" class="ui button">Two</button></div><form class="ui form soda-p-form"><button id="form-save" class="primary button">Save profile</button></form></main>`,
          theme
        );
        for (const id of ['basic-neutral', 'neutral', 'secondary', 'primary', 'form-save']) {
          const style = await page.locator('#' + id).evaluate((el) => ({
            height: el.getBoundingClientRect().height,
            radius: getComputedStyle(el).borderTopLeftRadius,
            font: getComputedStyle(el).fontSize,
          }));
          assert.equal(style.height, 44);
          assert.equal(style.radius, '0px');
          assert.equal(style.font, '13px');
        }
        assert.equal(await page.locator('#compact').evaluate((el) => el.getBoundingClientRect().height), 44);
        for (const id of ['neutral', 'basic-neutral'])
          assert.equal(
            await page.locator('#' + id).evaluate((el) => getComputedStyle(el).backgroundColor),
            'rgba(0, 0, 0, 0)'
          );
        await page.locator('#neutral').hover();
        await page.waitForFunction(() => {
          const neutral = document.querySelector('#neutral');
          return neutral && getComputedStyle(neutral).backgroundColor !== 'rgba(0, 0, 0, 0)';
        });
        assert.equal(
          await page.locator('#joined-first').evaluate((el) => getComputedStyle(el).borderTopRightRadius),
          '0px'
        );
        assert.equal(
          await page.locator('#joined-last').evaluate((el) => getComputedStyle(el).borderTopLeftRadius),
          '0px'
        );
      }
    });

    await t.test('native size classes and adjoining inputs share the selected 44px size', async () => {
      const variants = [
        'ui mini button',
        'ui tiny button',
        'ui small compact button',
        'ui basic button',
        'button secondary',
        'btn',
        'ui icon button',
        'soda-p-compact ui button',
        'soda-icon-action',
      ];
      for (const theme of ['light', 'dark']) {
        await render(
          `<main class="soda-page soda-settings-shell soda-settings">${variants.map((cls, i) => `<button id="size-${i}" class="${cls}">Action</button>`).join('')}<form class="ui form soda-p-form"><input id="single-value"><div id="single-selection" class="ui selection dropdown"><span class="text">English</span></div></form><div class="ui labeled button"><button id="watch" class="ui tiny button">Watch</button><a id="counter" class="ui basic label">24</a></div></main>`,
          theme
        );
        for (const el of await page.locator('[id]').all()) {
          assert.equal(
            await el.evaluate((e) => e.getBoundingClientRect().height),
            44,
            (await el.getAttribute('id')) ?? 'missing control ID'
          );
        }
        for (const el of await page.locator('button').all())
          assert.equal(await el.evaluate((e) => getComputedStyle(e).fontSize), '13px');
      }
    });
  } finally {
    await close();
  }
});
