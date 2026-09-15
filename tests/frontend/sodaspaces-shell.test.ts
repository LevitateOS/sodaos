import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {
  applyWorkspaceFrameNavigation,
  workspaceAuthLocation,
  workspaceEntryLocation,
  workspaceFrameAction,
  workspaceFrameLocator,
} from '../../frontend/spaces/sodaspaces-frame.ts';
import {bindWorkspaceShellLayout, dividerKeyWidth} from '../../frontend/spaces/sodaspaces-shell-layout.ts';
import {shellPaneHidden} from '../../frontend/spaces/sodaspaces-widths.ts';

test('workspace frame locator admits same-origin Forgejo paths only', () => {
  assert.equal(workspaceFrameLocator('/'), '/');
  assert.equal(workspaceFrameLocator('/alice/repo'), '/alice/repo');
  assert.equal(workspaceFrameLocator('/alice/repo?tab=issues'), '/alice/repo?tab=issues');
  assert.equal(workspaceFrameLocator(''), undefined);
  assert.equal(workspaceFrameLocator('https://evil.test/'), undefined);
  assert.equal(workspaceFrameLocator('//evil.test/x'), undefined);
  assert.equal(workspaceFrameLocator('/foo/../bar'), undefined);
  assert.equal(workspaceFrameLocator('/-/soda/workspace'), undefined);
  assert.equal(workspaceFrameLocator('/-/soda/api/session'), undefined);
  assert.equal(workspaceFrameLocator('/login?code=secret'), undefined);
  assert.equal(workspaceFrameLocator('/callback?state=x'), undefined);
  assert.equal(workspaceFrameLocator('/user/login'), undefined);
  assert.equal(workspaceFrameLocator('/login/oauth/authorize'), undefined);
  assert.equal(workspaceFrameLocator('/admin?soda-view=runners'), undefined);
});

test('framed login, consent, and callback leave the shell', () => {
  assert.equal(workspaceAuthLocation('/user/login'), '/user/login');
  assert.equal(workspaceAuthLocation('/user/login?redirect_to=%2F'), '/user/login?redirect_to=%2F');
  assert.equal(workspaceAuthLocation('/login/oauth/authorize?client_id=app'), '/login/oauth/authorize?client_id=app');
  assert.equal(workspaceAuthLocation('/login/oauth/grant'), '/login/oauth/grant');
  assert.equal(workspaceAuthLocation('/-/soda/login?destination=spaces'), '/-/soda/login?destination=spaces');
  assert.equal(workspaceAuthLocation('/-/soda/oauth/callback?code=x&state=y'), '/-/soda/oauth/callback?code=x&state=y');
  assert.equal(workspaceAuthLocation('/alice?code=secret'), '/alice?code=secret');
  assert.equal(workspaceAuthLocation('/'), undefined);
  assert.equal(workspaceAuthLocation('/alice/repo'), undefined);
  assert.deepEqual(workspaceFrameAction('/user/login'), {leave: '/user/login'});
  assert.deepEqual(workspaceFrameAction('/alice/repo'), {to: '/alice/repo'});
});

test('workspace entry wraps ordinary Forgejo paths and keeps native Soda hosts', () => {
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/', ''), '/-/soda/workspace');
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/alice/repo', ''),
    '/-/soda/workspace?to=%2Falice%2Frepo'
  );
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/native/alice/repo', '/native'),
    '/native/-/soda/workspace?to=%2Falice%2Frepo'
  );
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/?soda-view=spaces', ''), undefined);
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/admin?soda-view=runners', ''), undefined);
  assert.equal(workspaceEntryLocation('https://forgejo.example.test/user/login', ''), undefined);
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/?soda-view=spaces&soda-connect=failed', ''),
    undefined
  );
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/login/oauth/authorize?client_id=app', ''),
    undefined
  );
  assert.equal(
    workspaceEntryLocation('https://forgejo.example.test/-/soda/oauth/callback?code=x&state=y', ''),
    undefined
  );
});

test('shell compact surfaces hide the inactive pane only', () => {
  assert.equal(shellPaneHidden(false, 'forge', 'forge'), false);
  assert.equal(shellPaneHidden(false, 'forge', 'terminal'), false);
  assert.equal(shellPaneHidden(true, 'forge', 'forge'), false);
  assert.equal(shellPaneHidden(true, 'forge', 'terminal'), true);
  assert.equal(shellPaneHidden(true, 'terminal', 'forge'), true);
});

test('divider keys move the workspace width', () => {
  assert.equal(dividerKeyWidth(50, 'Home'), 35);
  assert.equal(dividerKeyWidth(50, 'End'), 65);
  assert.equal(dividerKeyWidth(50, 'ArrowLeft'), 55);
  assert.equal(dividerKeyWidth(50, 'ArrowRight'), 45);
  assert.equal(dividerKeyWidth(50, 'Enter'), undefined);
});

const shellMarkup = `<!doctype html><body class="soda-workspace-shell">
<span class="sodaspaces-measure" aria-hidden="true">MMMMMMMMMMMMMMMM</span>
<nav id="sodaspaces-surfaces" hidden aria-label="Workspace surface">
<button type="button" id="soda-surface-forge">Forge</button>
<button type="button" id="soda-surface-terminal">Terminal</button>
</nav>
<iframe id="soda-forgejo-frame" title="Forgejo" src="/"></iframe>
<div id="soda-workspace-divider" role="separator" tabindex="0"></div>
<div id="soda-workspace-root" data-actor="1"></div>
</body>`;

function bindShell(width: number) {
  const dom = new JSDOM(shellMarkup, {url: 'https://forge.test/-/soda/workspace', pretendToBeVisual: true});
  Object.defineProperty(dom.window, 'innerWidth', {value: width, writable: true});
  bindWorkspaceShellLayout(dom.window.document);
  return dom.window.document;
}

test('wide shell keeps both surfaces and no Forgejo header', () => {
  const doc = bindShell(1440);
  assert.equal(doc.getElementById('navbar'), null);
  assert.equal(doc.getElementById('soda-notification-preview'), null);
  assert.equal(doc.body.classList.contains('sodaspaces-compact'), false);
  assert.equal(doc.getElementById('sodaspaces-surfaces')?.hidden, true);
  assert.equal(doc.getElementById('soda-forgejo-frame')?.hasAttribute('data-surface-hidden'), false);
  assert.equal(doc.getElementById('soda-workspace-root')?.hasAttribute('data-surface-hidden'), false);
  assert.equal(doc.body.style.getPropertyValue('--soda-space-width'), '50vw');
});

test('compact shell switches Forge and Terminal without a second header', () => {
  const doc = bindShell(390);
  const switcher = doc.getElementById('sodaspaces-surfaces');
  const frame = doc.getElementById('soda-forgejo-frame');
  const workspace = doc.getElementById('soda-workspace-root');
  assert(switcher && frame && workspace);
  assert.equal(doc.getElementById('navbar'), null);
  assert(doc.body.classList.contains('sodaspaces-compact'));
  assert.equal(switcher.hidden, false);
  assert.equal(frame.hasAttribute('data-surface-hidden'), false);
  assert.equal(workspace.hasAttribute('data-surface-hidden'), true);
  doc.getElementById('soda-surface-terminal')?.click();
  assert.equal(frame.hasAttribute('data-surface-hidden'), true);
  assert.equal(workspace.hasAttribute('data-surface-hidden'), false);
  doc.getElementById('soda-surface-forge')?.click();
  assert.equal(frame.hasAttribute('data-surface-hidden'), false);
  assert.equal(workspace.hasAttribute('data-surface-hidden'), true);
});

test('wide shell divider grows the workspace from the keyboard', () => {
  const doc = bindShell(1600);
  const divider = doc.getElementById('soda-workspace-divider');
  assert(divider);
  divider.dispatchEvent(
    new (divider.ownerDocument.defaultView as typeof window).KeyboardEvent('keydown', {key: 'ArrowLeft'})
  );
  assert.equal(doc.body.style.getPropertyValue('--soda-space-width'), '55vw');
});

test('workspace host replaces itself when the frame lands on login', () => {
  const replaced: string[] = [];
  applyWorkspaceFrameNavigation(
    {
      location: {
        href: 'https://forge.test/-/soda/workspace?to=%2Falice%2Frepo',
        pathname: '/-/soda/workspace',
        search: '?to=%2Falice%2Frepo',
        replace(href: string) {
          replaced.push(href);
        },
      },
      history: {replaceState() {}},
    } as unknown as Window,
    '/user/login?redirect_to=%2Falice%2Frepo'
  );
  assert.deepEqual(replaced, ['/user/login?redirect_to=%2Falice%2Frepo']);
});

test('workspace host records admitted child paths without leaving', () => {
  const dom = new JSDOM(shellMarkup, {url: 'https://forge.test/-/soda/workspace', pretendToBeVisual: true});
  applyWorkspaceFrameNavigation(dom.window as unknown as Window, '/alice/repo');
  assert.equal(dom.window.location.pathname, '/-/soda/workspace');
  assert.equal(dom.window.location.search, '?to=%2Falice%2Frepo');
});
