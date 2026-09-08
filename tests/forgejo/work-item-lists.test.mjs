import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { test } from 'node:test';
import { chromium } from 'playwright';

const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;

// Read-only native markup samples, then isolated component fixtures. No session,
// writes to Forgejo, script replacement, or claim of verified native captures.
test('work items share appearance across native callers and keep usable metadata', { skip: !origin }, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  try {
    const page = await browser.newPage();
    const samples = {};
    for (const family of ['issues', 'pulls', 'milestones']) {
      const response = await page.goto(`${origin}/alice/activity-workbench/${family}`);
      assert.equal(response.status(), 200);
      assert.equal(page.url(), `${origin}/alice/activity-workbench/${family}`);
      const selector = family === 'milestones' ? '.soda-milestone-row' : '#issue-list > .flex-item';
      assert(await page.locator(selector).count() >= 2);
      samples[family] = await page.locator(selector).evaluateAll(es => [es.find(e => e.querySelector('.label')) || es[0], es.at(-1)].map(e => e.outerHTML).join(''));
    }
    const header = await readFile(new URL('../../appliance/forgejo/templates/custom/header.tmpl', import.meta.url), 'utf8');
    const files = [...header.matchAll(/\/soda\/forgejo\/([^?]+\.css)\?v=/g)].map(m => m[1]);
    const styles = await Promise.all(files.map(file => readFile(new URL(`../../assets/branding/forgejo/${file}`, import.meta.url), 'utf8')));
    const nativeResponse = await fetch(`${origin}/assets/css/index.css`);
    assert.equal(nativeResponse.status, 200);
    const palette = await readFile(new URL('../../assets/branding/theme/palette.css', import.meta.url), 'utf8');
    const css = `<style>${await nativeResponse.text()}\n${palette}\n${styles.join('\n')}</style>`;

    // The same unmodified native issue partial is used in all four compositions.
    for (const theme of ['light', 'dark']) for (const family of ['issues', 'pulls']) {
      const nativeTheme = await fetch(`${origin}/assets/css/theme-forgejo-${theme}.css`);
      assert.equal(nativeTheme.status, 200);
      for (const context of ['soda-repo-work-items', 'soda-page', 'soda-page soda-pulls', 'soda-notifications']) {
        await page.setContent(`<base href="${origin}"><style>${await nativeTheme.clone().text()}</style>${css}
          <main class="page-content soda-page-marker ${context}" style="max-width:1120px;margin:auto;padding:16px">
            <section class="soda-list"><div id="issue-list" class="flex-list">${samples[family]}</div></section>
            <section class="milestone-list">${samples.milestones}</section>
            <section class="soda-list" id="unrelated"><div class="flex-list"><div class="flex-item"><div class="flex-item-main"><div class="flex-item-title">Repository collection</div></div></div></div></section>
          </main>`);
        await page.evaluate(theme => document.documentElement.style.colorScheme = theme, theme);
        // Add only native writable-row markup and stress content to this fixture.
        await page.locator('#issue-list .flex-item-icon').first().evaluate(e => e.insertAdjacentHTML('afterbegin', '<input type="checkbox" autocomplete="off" class="issue-checkbox tw-mr-4" data-issue-id="1" aria-label="Select fixture issue">'));
        await page.locator('#issue-list .issue-title').first().evaluate(e => e.textContent = 'Keyboard navigation — Übersetzung 日本語 ' + 'very-long-unbroken-title'.repeat(5));
        await page.locator('#issue-list .labels-list .label').first().evaluate(e => e.textContent = 'Ein sehr langes lokalisiertes Label '.repeat(3));
        if (family === 'pulls') await page.locator('.issue-meta-branch .truncated-name').first().evaluate(e => e.textContent = 'long/branch/'.repeat(20));

        for (const width of [1440, 1024, 700, 601, 600, 390, 320]) {
          await page.setViewportSize({ width, height: 1000 });
          await page.mouse.move(0, 0);
          const rows = await page.locator('#issue-list > .flex-item, .soda-milestone-row').evaluateAll(es => es.map(e => {
            const style = getComputedStyle(e);
            const title = getComputedStyle(e.querySelector('.flex-item-title,.soda-milestone-name'));
            const link = getComputedStyle(e.querySelector('.issue-title,.soda-milestone-name > a'));
            return { padding: style.paddingBlock, background: style.backgroundColor, border: style.borderTopWidth, font: title.font, color: link.color };
          }));
          assert.deepEqual(rows.map(r => r.border), ['0px', '1px', '0px', '1px']);
          for (const row of rows) {
            assert.equal(row.padding, '24px');
            assert.equal(row.background, 'rgba(0, 0, 0, 0)');
            assert.equal(row.font, rows[0].font);
            assert.equal(row.color, rows[0].color);
          }
          assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `${family}/${context}/${theme}/${width}: overflow`);
          assert.equal(await page.locator('#unrelated .flex-item').evaluate(e => getComputedStyle(e).paddingTop), '16px');
          assert.equal(await page.locator('#unrelated .flex-item-title').evaluate(e => getComputedStyle(e).fontSize), '18px');
          const title = page.locator('#issue-list .issue-title').first();
          await title.focus();
          assert(await title.evaluate(e => e === document.activeElement));
          await title.hover();
          const hoverColor = await title.evaluate(e => getComputedStyle(e).color);
          const milestoneTitle = page.locator('.soda-milestone-name > a').first();
          await milestoneTitle.hover();
          assert.equal(await milestoneTitle.evaluate(e => getComputedStyle(e).color), hoverColor);
          assert.equal(await page.locator('#issue-list > .flex-item').first().evaluate(e => getComputedStyle(e).backgroundColor), 'rgba(0, 0, 0, 0)');
        }
        const checkbox = page.getByRole('checkbox', { name: 'Select fixture issue' });
        await checkbox.focus();
        await page.keyboard.press('Space');
        assert(await checkbox.isChecked());
        assert.equal(await checkbox.getAttribute('data-issue-id'), '1');
        if (family === 'pulls') assert(await page.locator('#issue-list .assignee').first().isVisible());
        assert(await page.locator('#issue-list .flex-item-trailing').first().isVisible());
      }
    }
    // A small inspection gallery from these native snippets and the real registry,
    // clearly separated from screenshots of the live routes.
    const dir = new URL('../../.artifacts/work-item-consistency/components/', import.meta.url);
    await mkdir(dir, { recursive: true });
    const registry = files.map(file => `<link rel="stylesheet" href="${origin}/assets/soda/forgejo/${file}">`).join('\n');
    for (const theme of ['light', 'dark']) for (const family of ['issues', 'pulls']) {
      await writeFile(new URL(`${family}-${theme}.html`, dir), `<!doctype html><html style="color-scheme:${theme}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><base href="${origin}"><link rel="stylesheet" href="${origin}/assets/css/index.css"><link rel="stylesheet" href="${origin}/assets/css/theme-forgejo-${theme}.css">${registry}</head><body><main class="page-content soda-page" style="max-width:1120px;margin:auto;padding:24px"><h1>Work-item component reference</h1><p>Native markup samples with production styles; separate from live-route acceptance.</p><section><h2>${family}</h2><div class="soda-list"><div id="issue-list" class="flex-list">${samples[family]}</div></div></section><section><h2>Milestones</h2><div class="milestone-list">${samples.milestones}</div></section></main></body></html>`);
    }
  } finally {
    await browser.close();
  }
});
