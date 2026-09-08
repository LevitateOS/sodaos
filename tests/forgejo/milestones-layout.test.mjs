import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { chromium } from 'playwright';

// Explicit opt-in: read only the existing local stock preview's public CSS.
// No login, fixture mutations, template reload or service lifecycle operations.
const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;
test('milestone rows retain one canvas, clear boundaries and responsive columns', { skip: !origin }, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const response = await fetch(`${origin}/assets/css/index.css`);
  assert.equal(response.status, 200);
  const nativeCSS = await response.text();
  const header = await readFile(new URL('../../appliance/forgejo/templates/custom/header.tmpl', import.meta.url), 'utf8');
  const styles = await Promise.all([...header.matchAll(/\/soda\/forgejo\/([^?]+\.css)\?v=/g)].map(async ([, file]) =>
    readFile(new URL(`../../assets/branding/forgejo/${file}`, import.meta.url), 'utf8')));
  const palette = await readFile(new URL('../../assets/branding/theme/palette.css', import.meta.url), 'utf8');
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  try {
    const page = await browser.newPage();
    await page.setContent(`<style>${nativeCSS}\n${palette}\n${styles.join('\n')}</style>
      <main class="page-content dashboard issues repository milestones soda-page soda-milestones">
        <div class="ui container soda-page-container">
          <div class="flex-container">
            <div class="flex-container-nav"><div class="ui secondary vertical filter menu">
              <div class="item">In your repositories <strong>9</strong></div>
              <a class="active repo name item"><span class="text truncate">alice/activity-workbench-with-a-long-repository-name</span><div class="ui green label">5</div></a>
            </div></div>
            <div class="flex-container-main content"><div class="milestone-list">
              <li class="milestone-card soda-milestone-row"><div class="soda-milestone-main"><h3 class="soda-milestone-name"><svg width="20" height="20"></svg><a href="#detail">Keyboard and accessibility polish</a></h3><div class="soda-milestone-repository">bob/activity-field-notes</div><div class="markup soda-milestone-description"><h2>Make every action reachable</h2><ul><li>Clear focus rings</li><li>Predictable navigation</li><li>Helpful labels</li></ul><p>A longer description with <a href="#details">an embedded link</a> that remains keyboard accessible.</p></div><div class="soda-milestone-activity">Updated 18 hours ago</div></div><div class="soda-milestone-deadline">Sep 10, 2026</div><div class="soda-milestone-progress"><div class="soda-milestone-progress-value"><progress value="25" max="100" aria-label="Keyboard and accessibility polish"></progress><span>25%</span></div><div class="soda-milestone-counts"><span>3 Open</span><span>1 Closed</span></div></div></li><li class="milestone-card soda-milestone-row"><div class="soda-milestone-main"><h3 class="soda-milestone-name"><svg width="20" height="20"></svg><a href="#detail">Keyboard and accessibility polish</a></h3><div class="soda-milestone-repository">bob/activity-field-notes</div><div class="markup soda-milestone-description"><h2>Make every action reachable</h2><ul><li>Clear focus rings</li><li>Predictable navigation</li><li>Helpful labels</li></ul><p>A longer description with <a href="#details">an embedded link</a> that remains keyboard accessible.</p></div><div class="soda-milestone-activity">Updated 18 hours ago</div></div><div class="soda-milestone-deadline">Sep 10, 2026</div><div class="soda-milestone-progress"><div class="soda-milestone-progress-value"><progress value="25" max="100" aria-label="Keyboard and accessibility polish"></progress><span>25%</span></div><div class="soda-milestone-counts"><span>3 Open</span><span>1 Closed</span></div></div></li>
            </div></div>
          </div>
        </div>
      </main>`);
    for (const theme of ["light", "dark"]) for (const width of [1440, 1100, 1024, 900, 768, 700, 390, 320]) {
      await page.evaluate(theme => document.documentElement.style.colorScheme = theme, theme);
      await page.setViewportSize({ width, height: 1000 });
      const layout = await page.evaluate(() => {
        const rect = selector => {
          const { x, y, width, right, bottom } = document.querySelector(selector).getBoundingClientRect();
          return { x, y, width, right, bottom };
        };
        return { sidebar: rect('.flex-container-nav'), card: rect('.milestone-card'), overflow: document.documentElement.scrollWidth > innerWidth };
      });
      const rows = await page.locator('.soda-milestone-row').evaluateAll(es=>es.map(e=>({background:getComputedStyle(e).backgroundColor,border:getComputedStyle(e).borderTopWidth,padding:getComputedStyle(e).paddingTop,title:getComputedStyle(e.querySelector('h3')).fontSize,description:e.querySelector('.soda-milestone-description').getBoundingClientRect().height})));
      assert.deepEqual(rows.map(r=>r.background),['rgba(0, 0, 0, 0)','rgba(0, 0, 0, 0)']);
      assert.deepEqual(rows.map(r=>r.border),['0px','1px']);
      for(const row of rows){assert.equal(row.padding,'24px');assert.equal(row.title,'20px');assert(row.description<=43);}
      assert.equal(layout.overflow, false, `${width}px: horizontal overflow`);
      if (width > 900) {
        assert.equal(layout.sidebar.width, 240, `${width}px: bounded repository sidebar`);
        assert(layout.card.x >= layout.sidebar.right + 23, `${width}px: separate columns`);
        assert(layout.card.width > 300, `${width}px: usable milestone card`);
      } else {
        assert(layout.card.y >= layout.sidebar.bottom, `${width}px: stacked cards`);
        assert(Math.abs(layout.card.width - layout.sidebar.width) < 1, `${width}px: full-width cards`);
      }
    }
    const description = page.locator('.soda-milestone-description').first();
    await description.locator('a').focus();
    assert.equal(await description.evaluate(e=>getComputedStyle(e).display),'block','keyboard focus exposes the complete linked description');
  } finally {
    await browser.close();
  }
});
