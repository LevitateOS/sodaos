import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { chromium } from 'playwright';

// Explicit opt-in: read only the existing local stock preview's public CSS.
// No login, fixture mutations, template reload or service lifecycle operations.
const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;
test('milestone sidebar and cards fit desktop and stacked mobile layouts', { skip: !origin }, async () => {
  assert.equal(origin, 'http://localhost:3300');
  const response = await fetch(`${origin}/assets/css/index.css`);
  assert.equal(response.status, 200);
  const nativeCSS = await response.text();
  const header = await readFile(new URL('../../appliance/forgejo/templates/custom/header.tmpl', import.meta.url), 'utf8');
  const styles = await Promise.all([...header.matchAll(/\/soda\/forgejo\/([^?]+\.css)\?v=/g)].map(async ([, file]) =>
    readFile(new URL(`../../assets/branding/forgejo/${file}`, import.meta.url), 'utf8')));
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  try {
    const page = await browser.newPage();
    await page.setContent(`<style>${nativeCSS}\n${styles.join('\n')}</style>
      <main class="page-content dashboard issues repository milestones soda-page soda-milestones">
        <div class="ui container soda-page-container">
          <div class="flex-container">
            <div class="flex-container-nav"><div class="ui secondary vertical filter menu">
              <div class="item">In your repositories <strong>9</strong></div>
              <a class="active repo name item"><span class="text truncate">alice/activity-workbench-with-a-long-repository-name</span><div class="ui green label">5</div></a>
            </div></div>
            <div class="flex-container-main content"><div class="milestone-list">
              <li class="milestone-card"><div class="milestone-header"><h3><span class="ui large label">bob/activity-field-notes</span><a>Overdue: onboarding examples</a></h3><div class="tw-flex tw-items-center"><span class="tw-mr-2">33%</span><progress value="33" max="100"></progress></div></div><div class="milestone-toolbar"><div class="group"><div>2 Open</div><div>1 Closed</div><div>Updated 3 hours ago</div></div></div><div class="markup content">Native milestone content</div></li>
            </div></div>
          </div>
        </div>
      </main>`);
    for (const width of [1440, 1024, 768, 700, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      const layout = await page.evaluate(() => {
        const rect = selector => {
          const { x, y, width, right, bottom } = document.querySelector(selector).getBoundingClientRect();
          return { x, y, width, right, bottom };
        };
        return { sidebar: rect('.flex-container-nav'), card: rect('.milestone-card'), overflow: document.documentElement.scrollWidth > innerWidth };
      });
      assert.equal(layout.overflow, false, `${width}px: horizontal overflow`);
      if (width > 700) {
        assert.equal(layout.sidebar.width, 240, `${width}px: bounded repository sidebar`);
        assert(layout.card.x >= layout.sidebar.right + 23, `${width}px: separate columns`);
        assert(layout.card.width > 300, `${width}px: usable milestone card`);
      } else {
        assert(layout.card.y >= layout.sidebar.bottom, `${width}px: stacked cards`);
        assert(Math.abs(layout.card.width - layout.sidebar.width) < 1, `${width}px: full-width cards`);
      }
    }
  } finally {
    await browser.close();
  }
});
