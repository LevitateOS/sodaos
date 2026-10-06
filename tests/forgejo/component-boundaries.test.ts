import assert from 'node:assert/strict';
import {test} from 'node:test';
import {openComponentBrowser, origin} from './fixtures/component-browser';

test('expanded components preserve native state and layout boundaries', {skip: !origin}, async (t) => {
  const {page, render, close} = await openComponentBrowser();
  try {
    await t.test('native form icons retain clearance and section headings share type', async () => {
      for (const theme of ['light', 'dark'])
        for (const width of [1440, 390, 320]) {
          await page.setViewportSize({width, height: 1000});
          await render(
            `<main class="soda-page soda-settings-shell soda-settings"><div class="user-setting-content"><h4 class="ui top attached header soda-p-heading">Native heading</h4><section><h4 class="ui top attached header soda-p-heading">Nested native heading</h4></section><form class="ui form soda-p-form"><fieldset class="soda-form-section"><legend>Form heading</legend><div class="ui left icon input"><input placeholder="Search"><i class="icon">⌕</i></div><div class="ui icon input"><input placeholder="Search"><i class="icon">⌕</i></div></fieldset></form></div></main>`,
            theme
          );
          for (const heading of await page.locator('h4,legend').all())
            assert.equal(await heading.evaluate((el) => getComputedStyle(el).fontSize), '22px');
          assert.equal(
            await page.locator('.left.input input').evaluate((el) => getComputedStyle(el).paddingInlineStart),
            '40px'
          );
          assert.equal(
            await page.locator('.input:not(.left) input').evaluate((el) => getComputedStyle(el).paddingInlineEnd),
            '40px'
          );
        }
      await page.setViewportSize({width: 1654, height: 1000});
    });

    await t.test('static guidance inside forms is visible without exposing validation messages', async () => {
      for (const theme of ['light', 'dark']) {
        await render(
          `<main class="soda-page"><div class="ui form"><div class="ui info message soda-notice">Context</div><div class="ui warning message soda-notice">Consequences</div><div class="ui error message">Inactive validation</div></div></main>`,
          theme
        );
        assert(await page.locator('.info.message').isVisible());
        assert(await page.locator('.warning.message').isVisible());
        assert.equal(await page.locator('.error.message').isVisible(), false);
      }
    });

    await t.test('settings headings stay above their bodies at every width', async () => {
      for (const width of [1440, 900, 899, 390, 320]) {
        await page.setViewportSize({width, height: 1000});
        await render(
          `<main class="soda-page soda-settings-shell soda-settings"><div class="user-setting-content"><section class="soda-settings-section"><h2>Email addresses</h2><div class="soda-settings-section-body"><p>Description</p><form class="ui form soda-p-form"><label>Email<input></label></form></div></section><h2 class="soda-settings-inventory-heading">Keys<div class="ui right"><button class="ui primary button">Add key</button></div></h2></div></main>`
        );
        const placement = await page.evaluate(() => {
          const headingElement = document.querySelector('.soda-settings-section > h2');
          const bodyElement = document.querySelector('.soda-settings-section-body');
          if (!headingElement || !bodyElement) throw Error('Missing settings sections');
          const heading = headingElement.getBoundingClientRect();
          const body = bodyElement.getBoundingClientRect();
          return {
            above: heading.bottom <= body.top,
            inset: String(Math.round(body.left - heading.left)) + 'px',
            fits: document.documentElement.scrollWidth <= innerWidth,
          };
        });
        assert.deepEqual(placement, {above: true, inset: width <= 1000 ? '16px' : '40px', fits: true});
      }
      await page.setViewportSize({width: 1654, height: 1000});
    });

    await t.test('form focus and errors survive the shared input cascade in both themes', async () => {
      for (const theme of ['light', 'dark']) {
        await render(
          `<main class="soda-page"><form class="ui form soda-form">
          <div class="field"><input id="normal"></div>
          <div class="field error"><input id="field-input"><textarea id="field-textarea"></textarea>
            <div id="field-dropdown" class="ui selection dropdown"></div></div>
          <input id="input-error" class="error"><textarea id="textarea-error" class="error"></textarea>
        </form><div id="native-error" style="border:1px solid var(--color-error-border)"></div>
          <div id="focus-color" style="border:1px solid var(--soda-page-link)"></div></main>`,
          theme
        );
        const borders = await page.evaluate(() =>
          Object.fromEntries(
            [...document.querySelectorAll('[id]')].map((el) => [el.id, getComputedStyle(el).borderTopColor])
          )
        );
        for (const id of ['field-input', 'field-textarea', 'field-dropdown', 'input-error', 'textarea-error']) {
          assert.equal(borders[id], borders['native-error'], `${theme}: ${id} must retain the native error border`);
        }
        assert.notEqual(borders.normal, borders['native-error']);
        await page.locator('#normal').focus();
        assert.equal(
          await page.locator('#normal').evaluate((el) => getComputedStyle(el).borderTopColor),
          borders['focus-color']
        );
        await page.locator('#field-input').focus();
        assert.equal(
          await page.locator('#field-input').evaluate((el) => getComputedStyle(el).borderTopColor),
          borders['native-error']
        );
      }
    });

    await t.test('settings panels do not give native tables panel padding', async () => {
      await render(`<main class="soda-page soda-admin"><section class="admin-setting-content">
        <h4 class="ui top attached header">Notices</h4>
        <table id="settings-table" class="ui attached segment"><tbody><tr><td>Notice</td></tr></tbody></table>
        <div id="settings-panel" class="ui attached segment">Panel</div>
      </section></main><table id="native-table" class="ui attached segment"><tbody><tr><td>Reference</td></tr></tbody></table>`);
      const padding = await page.evaluate(() =>
        Object.fromEntries([...document.querySelectorAll('[id]')].map((el) => [el.id, getComputedStyle(el).paddingTop]))
      );
      assert.equal(padding['settings-table'], padding['native-table']);
      assert.equal(padding['settings-panel'], '0px');
    });

    await t.test('settings search, row and modal forms keep native control sizes', async () => {
      await render(`<main class="soda-page soda-native-forms soda-settings"><section class="user-setting-content">
        <form class="ui form"><input id="principal"></form>
        <form class="ui form ignore-dirty"><input id="search"></form>
        <div class="flex-item"><form class="ui form"><input id="row"></form></div>
        <dialog open><form class="ui form"><input id="modal"></form></dialog>
      </section></main><form class="ui form"><input id="native"></form>`);
      const heights = await page.evaluate(() =>
        Object.fromEntries([...document.querySelectorAll('input')].map((el) => [el.id, getComputedStyle(el).minHeight]))
      );
      assert.equal(heights.principal, '44px');
      for (const id of ['search', 'row', 'modal']) assert.equal(heights[id], heights.native, id);
    });

    await t.test('primary action colors preserve native button states in both themes', async () => {
      for (const theme of ['light', 'dark']) {
        await render(
          `<main class="soda-page">
          <div id="action-color" style="background:var(--soda-button-primary-bg)"></div>
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
        </main>`,
          theme
        );
        const states = await page.evaluate(() =>
          Object.fromEntries(
            [...document.querySelectorAll('[id]')].map((el) => {
              const style = getComputedStyle(el);
              return [
                el.id,
                {
                  background: style.backgroundColor,
                  color: style.color,
                  cursor: style.cursor,
                  fontSize: style.fontSize,
                  opacity: style.opacity,
                  paddingBlock: `${style.paddingTop} ${style.paddingBottom}`,
                  pointerEvents: style.pointerEvents,
                },
              ];
            })
          )
        );
        for (const id of [
          'settings-primary',
          'form-primary',
          'repository-primary',
          'tiny-primary',
          'disabled-primary',
          'loading-primary',
        ]) {
          assert(states[id] && states['action-color']);
          assert.equal(
            states[id].background,
            states['action-color'].background,
            `${theme}: ${id} must use the filled primary`
          );
        }
        assert(states['anchor-primary'] && states['form-primary']);
        assert.equal(
          states['anchor-primary'].color,
          states['form-primary'].color,
          'primary anchor text remains legible'
        );
        assert(states['anchor-primary']);
        assert.notEqual(states['anchor-primary'].color, states['anchor-primary'].background);
        assert(states['tiny-primary'] && states['tiny-native']);
        assert.equal(states['tiny-primary'].fontSize, states['tiny-native'].fontSize);
        assert(states['tiny-primary'] && states['tiny-native']);
        assert.equal(states['tiny-primary'].paddingBlock, states['tiny-native'].paddingBlock);
        assert(states['disabled-primary']);
        assert.notEqual(states['disabled-primary'].opacity, '1');
        assert(states['disabled-primary']);
        assert.equal(states['disabled-primary'].pointerEvents, 'none');
        assert(states['loading-primary']);
        assert.equal(states['loading-primary'].color, 'rgba(0, 0, 0, 0)');
        assert(states['loading-primary']);
        assert.equal(states['loading-primary'].cursor, 'default');
        assert(states['red-primary'] && states['red-color']);
        assert.equal(states['red-primary'].background, states['red-color'].background);
        assert(states['positive-primary'] && states['green-color']);
        assert.equal(states['positive-primary'].background, states['green-color'].background);
        assert(states['basic-primary'] && states['action-color']);
        assert.equal(states['basic-primary'].background, states['action-color'].background);
        assert(states['basic-primary']);
        assert.notEqual(states['basic-primary'].color, states['basic-primary'].background);
      }
    });
  } finally {
    await close();
  }
});
