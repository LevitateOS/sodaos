import test from 'node:test';
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {
  setupWorkspaceDriver,
  chooseFirstRepository,
  fixture,
  openSession,
  textContents,
} from './fixtures/workspace-driver';
import {captureSpacesComponent} from '../../scripts/screenshot';
setupWorkspaceDriver();

test('page stylesheet sizes the workspace host without drawer CSS', async (t) => {
  const page = await fixture(t, 'page', () => {
    document.querySelector('link[href="/assets/sodaspaces-drawer.css"]')?.remove();
  });
  const geometry = await page.locator('soda-spaces').evaluate((node) => {
    const workspace = node.querySelector('.soda-workspace')!;
    return {
      display: getComputedStyle(node).display,
      host: {width: node.clientWidth, height: node.clientHeight},
      workspace: {width: workspace.clientWidth, height: workspace.clientHeight},
    };
  });
  assert.equal(geometry.display, 'block');
  assert(geometry.host.width > 0 && geometry.host.height > 0, 'page CSS must size the custom-element host');
  assert(geometry.workspace.width > 0 && geometry.workspace.height > 0, 'workspace must inherit host geometry');
});

test('first use entry keeps its primary action in the initial viewport at presentation size', async (t) => {
  const page = await fixture(t, 'page', undefined, true);
  await page.setViewportSize({width: 1440, height: 1000});
  await page.evaluate(() => window.workspaceFixture.api.refresh());
  await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
  for (const size of [
    {width: 1440, height: 1000},
    {width: 1366, height: 768},
    {width: 1280, height: 720},
  ]) {
    await page.setViewportSize(size);
    await page.evaluate(async () => {
      await document.fonts.ready;
      await new Promise(requestAnimationFrame);
      await new Promise(requestAnimationFrame);
    });
    const action = await page.getByRole('button', {name: 'Explore ideas', exact: true}).boundingBox();
    assert(action, 'entry primary action is missing');
    assert(
      action.y >= 0 && action.y + action.height <= size.height,
      `initial request primary action sits below the fold at ${size.width}x${size.height}: ` + JSON.stringify(action)
    );
  }
  await page.setViewportSize({width: 1440, height: 1000});
  const frame = await page.locator('.soda-setup-panel').evaluate((node) => {
    const workspace = node.closest('.soda-workspace')!;
    const frame = workspace.querySelector(':scope > .soda-workspace-frame') as HTMLElement;
    const frameStyle = getComputedStyle(frame);
    const panelStyle = getComputedStyle(node);
    return {
      frameMaxHeight: frameStyle.maxHeight,
      panelOverflowY: panelStyle.overflowY,
      panelScroll: node.scrollHeight - node.clientHeight,
    };
  });
  assert.notEqual(
    frame.frameMaxHeight,
    '640px',
    'entry frame is capped below the presentation viewport: ' + JSON.stringify(frame)
  );
  assert.equal(frame.panelOverflowY, 'visible', 'entry form keeps a nested panel scroller: ' + JSON.stringify(frame));
  assert(frame.panelScroll <= 1, 'entry form overflows its panel: ' + JSON.stringify(frame));
});

async function workspaceIntroGeometry(page: Page) {
  // Wait for loaded-font ResizeObserver work and its Lit update before
  // comparing the two settled intro layouts around Join.
  await page.evaluate(async () => {
    await document.fonts.ready;
    await new Promise(requestAnimationFrame);
    await new Promise(requestAnimationFrame);
  });
  const workspace = page.locator('.soda-workspace');
  assert.equal(await page.locator('.soda-setup-footer:visible').count(), 0);
  assert.equal(await page.getByRole('button', {name: 'Back to workspace', exact: true}).count(), 0);
  assert.equal(await page.getByLabel('Workspace options', {exact: true}).count(), 0);
  assert.equal(await page.locator('[role=tab]:visible, .soda-empty-pane:visible').count(), 0);
  assert.equal(await page.locator('.soda-selected-project h1').innerText(), 'alice/Alpha');
  assert.equal(await page.locator('.soda-project-state').innerText(), 'Environment: Running');
  assert.equal(await page.getByRole('button', {name: 'Project settings', exact: true}).count(), 1);
  assert.equal(await workspace.locator('.ui.primary.button:visible').count(), 1);
  return workspace.evaluate((node) => {
    const box = (selector: string) => {
      const element = node.querySelector(selector);
      if (!element) throw Error(selector);
      const rect = element.getBoundingClientRect();
      return {x: rect.x, y: rect.y, width: rect.width, height: rect.height, center: rect.x + rect.width / 2};
    };
    const intro = box('.soda-workspace-intro'),
      heading = box('.soda-workspace-intro h2'),
      action = box('.soda-intro-action');
    const svg = node.querySelector('.soda-workspace-illustration');
    if (svg?.querySelector('path')?.namespaceURI !== 'http://www.w3.org/2000/svg')
      throw Error('Artwork lost SVG geometry');
    if (Math.abs(action.center - intro.center) > 1 || Math.abs(heading.center - intro.center) > 1)
      throw Error('Intro lost horizontal centering: ' + JSON.stringify({action, heading, intro}));
    if (node.scrollWidth > node.clientWidth) throw Error('Intro overflows the viewport');
    return {
      header: box('.soda-workspace-toolbar'),
      intro,
      heading,
      action,
      compact: node.classList.contains('is-compact'),
    };
  });
}
for (const theme of ['light', 'dark'])
  for (const width of [1536, 1440, 800, 640, 390])
    test(`first use ${theme}/${width}: explicit welcome to typed terminal, then exact re-entry`, async (t) => {
      const page = await fixture(t, 'page', undefined, true);
      const height = width === 1536 ? 1024 : 844;
      await page.setViewportSize({width, height});
      await page.evaluate((theme) => {
        document.documentElement.style.colorScheme = theme;
      }, theme);
      await page.evaluate(() => window.workspaceFixture.api.refresh());
      await page.getByRole('heading', {name: 'What do you want to build?'}).waitFor();
      const surface = page.locator('.soda-workspace');
      assert.equal(await surface.getByRole('button', {name: 'Explore ideas', exact: true}).isEnabled(), false);
      assert.equal(await surface.getByRole('textbox', {name: 'Describe what you want to build'}).count(), 1);
      assert.equal(await surface.getByRole('combobox', {name: 'Department', exact: true}).count(), 1);
      assert.equal(await surface.getByRole('heading', {name: 'What do you want to build?'}).count(), 1);
      assert.equal(await page.locator('.soda-workspace-navigation:visible').count(), 0);
      assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 0);
      await page.evaluate(() => document.fonts.ready);
      const setupBounds = await page.locator('.soda-workspace-frame').boundingBox();
      assert(setupBounds);
      const allocation = await page.locator('main[data-extension-panel]').boundingBox();
      assert(allocation, 'current native panels allocate the authored mount');
      assert(
        setupBounds.width >= Math.min(1240, allocation.width * 0.85),
        'setup frame is trapped inside its native allocation: ' + JSON.stringify({setupBounds, allocation})
      );
      assert(
        Math.abs(setupBounds.x - allocation.x - (allocation.width - setupBounds.width) / 2) <= 1,
        'setup frame is not centered in its allocation'
      );
      const welcome = await page.locator('.soda-setup-welcome').boundingBox();
      assert(welcome && welcome.width <= 560 && welcome.x > setupBounds.x, 'welcome lacks a focused reading measure');
      await captureSpacesComponent(page, `welcome-${theme}-${width}`);
      const picker = await chooseFirstRepository(page, `${theme}-${width}`);
      const configureBounds = await page.locator('.soda-workspace-frame').boundingBox();
      const configureHeading = await page.getByRole('heading', {name: 'Configure project'}).boundingBox();
      const configureAction = await page.getByRole('button', {name: 'Create project', exact: true}).boundingBox();
      assert(
        picker.frame &&
          configureBounds &&
          Math.abs(picker.frame.height - configureBounds.height) <= 1 &&
          Math.abs(picker.frame.y - configureBounds.y) <= 1,
        'panel jumps between setup steps: ' + JSON.stringify({picker: picker.frame, configure: configureBounds})
      );
      assert(
        picker.heading &&
          configureHeading &&
          Math.abs(picker.heading.x - configureHeading.x) <= 1 &&
          Math.abs(picker.heading.y - configureHeading.y) <= 1,
        'setup headings lose their alignment'
      );
      assert(
        picker.action &&
          configureAction &&
          Math.abs(picker.action.x + picker.action.width - configureAction.x - configureAction.width) <= 1,
        'primary actions lose their right alignment'
      );
      assert(
        picker.action && configureAction && picker.action.height >= 52 && configureAction.height >= 52,
        'setup primary actions need the prominent journey target'
      );
      const footerTop = (await page.locator('.soda-setup-footer').boundingBox())?.y;
      assert(
        footerTop &&
          picker.action &&
          configureAction &&
          picker.footer &&
          picker.action.y + picker.action.height <= picker.footer.y &&
          configureAction.y + configureAction.height <= footerTop,
        'setup actions are clipped below the progress footer: ' +
          JSON.stringify({picker: picker.action, pickerFooter: picker.footer, configure: configureAction, footerTop})
      );
      assert.equal(
        await page.getByRole('button', {name: 'Refresh status', exact: true}).count(),
        0,
        'ready configuration should not show routine recovery controls'
      );
      assert.equal(
        (await page.locator('[aria-current=step]').innerText()).replace(/\s+/g, ' '),
        '03 Create the project'
      );
      assert(
        configureBounds && configureBounds.x === setupBounds.x && configureBounds.width === setupBounds.width,
        'configuration changed the outer frame'
      );
      const form = await page.locator('.soda-project-journey').boundingBox();
      assert(
        form && form.width <= 720 && form.width < configureBounds.width,
        'configuration needs an inset form measure'
      );
      assert(
        await page
          .locator('.soda-configuration-fields:visible')
          .evaluate((node) => node.scrollHeight <= node.clientHeight),
        'ordinary configuration clips its helper text'
      );
      assert(configureBounds.height <= 720, 'tall screens stretch the setup actions away from the form');
      // Scrollable setup must keep its footer/actions reachable on a short viewport.
      await page.locator('.soda-setup-footer').scrollIntoViewIfNeeded();
      assert(await page.locator('.soda-setup-footer').isVisible());
      await page.locator('.soda-workspace').evaluate((node) => (node.scrollTop = 0));
      await captureSpacesComponent(page, `configure-${theme}-${width}`);
      assert.equal(await page.getByRole('checkbox').count(), 0, 'unavailable Tailnet must stay out of setup');
      assert.equal(
        await page.evaluate(
          () =>
            window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
        ),
        0
      );
      await page.getByRole('button', {name: 'Create project', exact: true}).click();
      await page.getByRole('heading', {name: 'Review your product direction', exact: true}).waitFor();
      await page.getByRole('button', {name: 'Accept and build', exact: true}).waitFor();
      assert.equal(
        await page.getByLabel('Desired outcome', {exact: true}).inputValue(),
        'Build a small project tracker'
      );
      assert.equal(
        await page.evaluate(
          () =>
            window.workspaceFixture.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')).length
        ),
        1
      );
      // A native Workspace entry leaves the saved request available while
      // the developer explicitly selects the retained terminal branch.
      await page.evaluate(() => {
        window.history.pushState({}, '', '/workspace?repository=7');
        (document.querySelector('soda-spaces') as unknown as {present(kind: 'page'): void}).present('page');
      });
      await page.getByRole('button', {name: 'Terminals', exact: true}).click();
      await page.getByRole('button', {name: 'Join project', exact: true}).waitFor();
      const joinedLayout = await workspaceIntroGeometry(page);
      assert.equal(await page.locator('.soda-workspace-navigation:visible').count(), joinedLayout.compact ? 0 : 1);
      assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 0);
      assert.equal(await page.getByRole('button', {name: 'Refresh status', exact: true}).count(), 0);
      await captureSpacesComponent(page, `join-${theme}-${width}`);
      await page.getByRole('button', {name: 'Join project', exact: true}).click();
      await page.getByRole('heading', {name: 'Open your first terminal'}).waitFor();
      assert.equal(
        await page.evaluate(() => window.workspaceFixture.calls.some((c) => c.path.endsWith('/development-keys'))),
        false
      );
      assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 1);
      const terminalLayout = await workspaceIntroGeometry(page);
      assert.deepEqual(terminalLayout.header, joinedLayout.header, 'project header jumps after Join');
      assert.deepEqual(terminalLayout.intro, joinedLayout.intro, 'intro stage jumps after Join');
      assert.equal(await page.locator('.soda-intro-helper:visible').innerText(), 'Project account: alice');
      assert.equal(
        await page.evaluate(() => window.workspaceFixture.sockets.length),
        0,
        'Join must not create a terminal'
      );
      await captureSpacesComponent(page, `first-terminal-${theme}-${width}`);
      await page.getByRole('button', {name: 'New terminal', exact: true}).click();
      await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
      assert.equal(await page.getByRole('dialog', {name: 'New terminal', exact: true}).count(), 0);
      await page.waitForFunction(() => document.activeElement?.classList.contains('xterm-helper-textarea'));
      await page.keyboard.type('spaces-ready');
      await page.waitForFunction(() => document.querySelector('.xterm-rows')?.textContent?.includes('spaces-ready'));
      assert(
        await page.locator('.soda-workspace').evaluate((node) => node.scrollWidth <= node.clientWidth),
        'page overflow'
      );
      const geometry = await page.locator('.soda-workspace').evaluate((node) => {
        const box = (selector: string) => {
          const element = node.querySelector(selector);
          if (!element) throw Error(selector);
          return element.getBoundingClientRect().toJSON() as {
            x: number;
            y: number;
            width: number;
            height: number;
            bottom: number;
            right: number;
          };
        };
        return {
          workspace: node.getBoundingClientRect().toJSON() as {bottom: number; height: number},
          header: box('.soda-workspace-toolbar'),
          canvas: box('.soda-workspace-canvas'),
          nav: box('.soda-workspace-navigation'),
          compact: node.classList.contains('is-compact'),
          overflow: node.scrollHeight > node.clientHeight,
        };
      });
      assert(!geometry.overflow, 'working workspace must not scroll outside its terminal');
      const terminalInsets = await page.locator('.soda-workspace-terminal:visible').evaluate((node) => {
        const screen = node.querySelector('.soda-terminal-screen')!.getBoundingClientRect();
        const grid = node.querySelector('.xterm-screen')!.getBoundingClientRect();
        return {
          left: grid.x - screen.x,
          top: grid.y - screen.y,
          right: screen.right - grid.right,
          bottom: screen.bottom - grid.bottom,
        };
      });
      assert(terminalInsets.left >= 8 && terminalInsets.top >= 8, 'terminal text touches its frame');
      assert(terminalInsets.right >= 0 && terminalInsets.bottom >= 0, 'fitted terminal cells escape the padded screen');
      assert(geometry.canvas.height > geometry.workspace.height * 0.65, 'chrome consumes too much terminal height');
      assert(geometry.workspace.bottom - geometry.canvas.bottom <= 26, 'terminal leaves unused space below it');
      assert(
        Math.abs(geometry.canvas.right - allocation.x - allocation.width) <= 1,
        'terminal must fill its native allocation: ' + JSON.stringify({canvas: geometry.canvas, allocation})
      );
      if (!geometry.compact) assert(geometry.header.x >= geometry.nav.right, 'project header overlaps sidebar');
      await captureSpacesComponent(page, `working-${theme}-${width}`);
      // New terminal is the "+" after the tabs; the toolbar no longer repeats it.
      assert.equal(
        await page.locator('.soda-workspace-tabs').getByRole('button', {name: 'New terminal', exact: true}).count(),
        1
      );
      assert.equal(await page.getByRole('button', {name: 'New terminal', exact: true}).count(), 1);
      assert.equal(await page.locator('.soda-workspace-tabs [aria-selected=true]').innerText(), 'Terminal 1');
      assert.equal(await page.getByLabel('Open tabs', {exact: true}).count(), 1);
      assert.equal(
        await page.getByLabel('Open tabs', {exact: true}).isVisible(),
        false,
        'single terminal needs no tab picker'
      );
      assert.equal(
        await page.getByLabel('Move terminal to pane', {exact: true}).count(),
        0,
        'single tab/pane has no move destination'
      );
      assert.equal(await page.locator('.soda-project-terminals:visible').count(), 0);
      await page.getByLabel('Workspace options', {exact: true}).click();
      assert.equal(
        await page.locator('.soda-project-terminals .soda-session-list [aria-current=true]:visible').count(),
        1
      );
      await page.keyboard.press('Escape');
      const terminalScreen = await page.locator('.soda-workspace-terminal:visible .xterm').elementHandle();
      const menuBaseline = await page.evaluate(() => ({
        calls: window.workspaceFixture.calls.length,
        sockets: window.workspaceFixture.sockets.length,
      }));
      await page.locator('.soda-workspace-terminal:visible').getByLabel('Terminal actions', {exact: true}).click();
      const terminalMenu = page.locator('.soda-workspace-terminal:visible .soda-menu[open] > div');
      assert.equal(await terminalMenu.locator('.soda-menu-heading').innerText(), 'Terminal 1');
      assert.deepEqual(await textContents(terminalMenu.getByRole('button')), [
        'Rename terminal',
        'Hide terminal',
        'Project settings',
        'End terminal…',
      ]);
      const menuBounds = await terminalMenu.boundingBox();
      assert(
        menuBounds &&
          menuBounds.x >= 0 &&
          menuBounds.x + menuBounds.width <= width &&
          menuBounds.y + menuBounds.height <= height,
        'terminal menu is clipped'
      );
      await captureSpacesComponent(page, `terminal-menu-${theme}-${width}`);
      await page.keyboard.press('Escape');
      await page.getByLabel('Pane actions', {exact: true}).click();
      const paneMenu = page.locator('.soda-pane-chrome .soda-menu[open] > div');
      assert.deepEqual(await textContents(paneMenu.getByRole('button')), ['Split right', 'Split below']);
      await captureSpacesComponent(page, `pane-menu-${theme}-${width}`);
      await page.keyboard.press('Escape');
      assert(await terminalScreen?.evaluate((node) => node.isConnected));
      assert.deepEqual(
        await page.evaluate(() => ({
          calls: window.workspaceFixture.calls.length,
          sockets: window.workspaceFixture.sockets.length,
        })),
        menuBaseline,
        'context menus changed terminal machinery'
      );
      const before = await page.evaluate(() => {
        const f = window.workspaceFixture;
        return {
          id: f.spaces[0]?.terminals[0]?.id,
          writes: f.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')),
          creates: f.sockets.flatMap((s) => s.sent).filter((c) => c.action === 'create').length,
        };
      });
      assert.equal(before.creates, 1);
      assert.equal(before.writes.length, 3);
      assert.deepEqual(before.writes[1]?.body, {ssh_keys: 'none'});
      await page.evaluate(() => window.workspaceFixture.remount());
      await openSession(page, 'Terminal 1');
      assert.deepEqual(
        await page.evaluate(() => {
          const f = window.workspaceFixture;
          return {
            id: f.spaces[0]?.terminals[0]?.id,
            writes: f.calls.filter((c) => c.method !== 'GET' && !c.path.includes('/api/factory/')),
            creates: f.sockets.flatMap((s) => s.sent).filter((c) => c.action === 'create').length,
          };
        }),
        before
      );
    });
