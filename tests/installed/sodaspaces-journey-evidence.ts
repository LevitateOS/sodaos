import assert from 'node:assert/strict';
import type {FrameLocator, Page} from 'playwright';
import type {MatrixProject} from './sodaspaces-matrix-input';

export const spacesPage = '/-/extensions/pages/soda/spaces';
export const spacesAPI = spacesPage + '/api';
const workspacePage = '/-/extensions/workspace';
const workspaceMount = '[data-extension-workspace]';
const workspaceFrame = '[data-extension-workspace-frame]';
const spacesMount = '[data-extension-page][data-extension-id="soda"][data-extension-page-id="spaces"]';

export async function nativeSpacesMount(page: Page): Promise<string> {
  const url = new URL(page.url());
  assert.equal(url.pathname, workspacePage, 'Native extension workspace required');
  assert.equal(url.hash, '', 'Native workspace hash is not allowed');
  assert.deepEqual(Array.from(url.searchParams.keys()), ['path'], 'Native workspace path authority required');
  assert.deepEqual(url.searchParams.getAll('path'), [spacesPage], 'Native Spaces path required');

  const workspace = page.locator(workspaceMount);
  await workspace.waitFor();
  assert.equal(await workspace.getAttribute('data-workspace-path'), spacesPage);
  const workspaceGeneration = await workspace.getAttribute('data-workspace-session-generation');
  assert(workspaceGeneration, 'Workspace mount has no native session');

  const frame = page.frameLocator(workspaceFrame);
  const mount = frame.locator(spacesMount);
  await mount.waitFor();
  const expectedFrameURL = new URL(spacesPage, url.origin).href;
  const spacesFrame = page
    .frames()
    .find((candidate) => candidate.parentFrame() === page.mainFrame() && candidate.url() === expectedFrameURL);
  assert(spacesFrame, 'Native Spaces frame is missing');
  const frameURL = new URL(spacesFrame.url());
  assert.equal(frameURL.origin, url.origin, 'Spaces must use the same-origin workspace frame');
  assert.equal(frameURL.pathname, spacesPage, 'Native Spaces frame path required');
  assert.equal(frameURL.search, '', 'Native Spaces frame query is not allowed');
  assert.equal(frameURL.hash, '', 'Native Spaces frame hash is not allowed');

  const api = new URL((await mount.getAttribute('data-extension-api-base')) || '', frameURL);
  assert.equal(api.origin, url.origin, 'Spaces API must be same-origin');
  assert.equal(api.pathname, spacesAPI + '/', 'Spaces mount API mismatch');
  assert.equal(api.search, '', 'Spaces API query is not allowed');
  assert.equal(api.hash, '', 'Spaces API hash is not allowed');
  const generation = await mount.getAttribute('data-extension-session-generation');
  assert(generation, 'Spaces mount has no native session');
  assert.equal(generation, workspaceGeneration, 'Workspace and Spaces must share native session authority');
  return generation;
}

export async function spacesContent(page: Page): Promise<Page | FrameLocator> {
  // The source-only workspace fixture deliberately mounts Spaces at `/` without
  // Forgejo's extension router. Keep this exact fixture opt-in local to tests.
  if (await page.locator('meta[name="soda-component-fixture"][content="spaces"]').count()) return page;
  await nativeSpacesMount(page);
  await focusNativeBrowsing(page);
  return page.frameLocator(workspaceFrame);
}

async function focusNativeBrowsing(page: Page) {
  const workspace = page.locator(workspaceMount);
  const frame = page.locator(workspaceFrame);
  const toggle = workspace.locator('[data-extension-workspace-toggle]');
  await toggle.waitFor({state: 'visible'});
  const state = async () => ({
    expanded: await toggle.getAttribute('aria-expanded'),
    label: (await toggle.innerText()).trim(),
  });
  let current = await state();
  if (current.expanded === 'true' && current.label === 'Focus browsing') {
    await toggle.click();
    current = await state();
  }
  assert.deepEqual(current, {expanded: 'false', label: 'Show panels'}, 'Native browsing focus control state mismatch');
  await page.waitForFunction(() => {
    const root = document.querySelector<HTMLElement>('[data-extension-workspace]');
    const frame = root?.querySelector<HTMLIFrameElement>('[data-extension-workspace-frame]');
    const toggle = root?.querySelector<HTMLButtonElement>('[data-extension-workspace-toggle]');
    return !!(
      root?.classList.contains('focus-view') &&
      toggle?.getAttribute('aria-expanded') === 'false' &&
      toggle.textContent?.trim() === 'Show panels' &&
      frame &&
      document.activeElement === frame &&
      frame.contentDocument?.hasFocus()
    );
  });
  assert.equal(await frame.getAttribute('data-extension-workspace-frame'), '');
}
export interface FirstUseEvidence {
  stage: string;
  project?: string;
  session?: MatrixSession;
  facts?: MatrixFacts;
  writes: string[];
  create_status?: number;
  join_status?: number;
  attached?: boolean;
}

export interface MatrixSession {
  name: string;
  environment: string;
  id: string;
  actor: string;
}
export interface MatrixFacts {
  pid: number;
  start: string;
  login: string;
  marker: string;
  tty: boolean;
  term: string;
}
export interface MatrixEvidence {
  stage?: string;
  sessions: Array<
    MatrixSession & {
      facts?: MatrixFacts;
      end_requested?: boolean;
      end_http?: 'accepted' | 'transport-unconfirmed';
      native_cleanup?: boolean;
    }
  >;
  same_document?: boolean;
  exact_reload?: boolean;
  cli?: unknown;
}
export interface MatrixNative {
  shell(page: Page, session: MatrixSession, initialize: boolean): Promise<MatrixFacts>;
  inspect(
    project: MatrixProject,
    actor: number,
    session: MatrixSession,
    facts: MatrixFacts,
    ended: boolean
  ): Promise<void>;
  cli?(page: Page, sessions: readonly (MatrixSession & {facts?: MatrixFacts})[]): Promise<unknown>;
}
