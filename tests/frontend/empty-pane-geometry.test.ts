import test from 'node:test';
import assert from 'node:assert/strict';
import {setupWorkspaceDriver, fixture, openSession, paneAction} from './fixtures/workspace-driver';
setupWorkspaceDriver();

// F05: the empty pane is the terminal-selection surface shown in a pane with
// no attached terminal. Its message and both actions (Use existing terminal /
// New terminal here) must stay inside the pane box and remain reachable at
// compact, short-height and reduced CSS viewport/allocation proxy sizes,
// with no horizontal spill. 720x500 derives from the 1440x1000 base halved.
for (const viewport of [
  {width: 320, height: 568},
  {width: 720, height: 422},
  {width: 1440 / 2, height: 1000 / 2},
]) {
  test(
    `empty pane ${viewport.width}x${viewport.height}: message and selection actions stay inside the pane`,
    {timeout: 30000},
    async (t) => {
      const page = await fixture(t);
      await page.setViewportSize({width: 1440, height: 1000});
      await page.evaluate(() => window.workspaceFixture.api.refresh());
      await openSession(page, 'Build');
      await paneAction(page, 'Split right');
      await page.locator('.soda-empty-pane:visible').waitFor();
      await page.setViewportSize(viewport);
      // Compact widths consolidate panes; select the empty pane if the
      // focused-pane control is offered so the same surface is measured.
      const focused = page.getByLabel('Focused pane', {exact: true});
      if ((await page.locator('.soda-empty-pane:visible').count()) === 0 && (await focused.count()) > 0) {
        const options = await focused.locator('option').allTextContents();
        const empty = options.findIndex((label) => /Pane 2/.test(label));
        if (empty >= 0) await focused.selectOption({index: empty});
      }
      const pane = page.locator('.soda-empty-pane:visible');
      await pane.waitFor();
      const box = await pane.boundingBox();
      assert(
        box && box.x >= 0 && box.y >= 0 && box.x + box.width <= viewport.width && box.y + box.height <= viewport.height,
        `empty pane escapes the ${viewport.width}x${viewport.height} viewport: ${JSON.stringify(box)}`
      );
      assert(
        await page.locator('.soda-workspace').evaluate((node) => node.scrollWidth <= node.clientWidth + 1),
        'workspace overflows horizontally beside the empty pane'
      );
      const spill = await pane.evaluate((node) => {
        const paneBox = node.getBoundingClientRect();
        const offenders: string[] = [];
        for (const child of node.querySelectorAll('p, button')) {
          const rect = child.getBoundingClientRect();
          if (
            rect.width > 0 &&
            (rect.left < paneBox.left - 1 ||
              rect.right > paneBox.right + 1 ||
              rect.top < paneBox.top - 1 ||
              rect.bottom > paneBox.bottom + 1)
          )
            offenders.push(`${child.tagName}:${child.textContent?.trim().slice(0, 24)}`);
        }
        return offenders;
      });
      assert.deepEqual(spill, [], `empty pane content spills outside its box at ${viewport.width}x${viewport.height}`);
      for (const name of ['Use existing terminal', 'New terminal here']) {
        const actionControl = pane.getByRole('button', {name, exact: true});
        await actionControl.scrollIntoViewIfNeeded();
        assert(await actionControl.isVisible(), `${name} is hidden in the empty pane`);
        assert(await actionControl.isEnabled(), `${name} is disabled in the empty pane`);
        const actionBox = await actionControl.boundingBox();
        assert(
          actionBox &&
            box &&
            actionBox.x >= box.x - 1 &&
            actionBox.y >= box.y - 1 &&
            actionBox.x + actionBox.width <= box.x + box.width + 1 &&
            actionBox.y + actionBox.height <= box.y + box.height + 1,
          `${name} renders outside the empty pane box`
        );
      }
      const focus = await pane.evaluate((node) => {
        const target = node.querySelector('button:not([disabled])');
        if (!(target instanceof HTMLElement)) return 'none';
        target.focus();
        return node.contains(document.activeElement) ? 'inside' : 'escaped';
      });
      assert.equal(focus, 'inside', 'empty pane has no keyboard-focusable selection action');
    }
  );
}
