import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import type {MatrixProject} from './sodaspaces-matrix-input';

export const spacesPage = '/workspace';
export const spacesAPI = '/-/native/pages/soda/spaces/api';
const workspacePage = '/workspace';
const workspaceMount = '[data-extension-workspace]';
const spacesMount = '[data-extension-page][data-extension-id="soda"][data-extension-page-id="spaces"]';

export async function nativeSpacesMount(page: Page): Promise<string> {
  const hashBefore = new URL(page.url()).hash;
  const workspace = page.locator(workspaceMount);
  await workspace.waitFor();
  assert.equal(await page.locator(workspaceMount).count(), 1, 'Exactly one native Workspace owner required');
  assert((await workspace.getAttribute('data-workspace-direct')) !== null, 'Direct native layout required');
  const subURL = (await workspace.getAttribute('data-app-sub-url')) ?? '';
  const url = new URL(page.url());
  assert.equal(url.pathname, `${subURL}${workspacePage}`, 'Native workspace required');
  assert.equal(
    await workspace.getAttribute('data-workspace-path'),
    `${subURL}${url.pathname}${url.search}`,
    'Native Workspace URL must be canonical'
  );
  const workspaceGeneration = await workspace.getAttribute('data-workspace-session-generation');
  assert(workspaceGeneration, 'Workspace mount has no native session');

  const mount = page.locator(spacesMount);
  await mount.waitFor();
  const api = new URL((await mount.getAttribute('data-extension-api-base')) || '', url.origin);
  assert.equal(api.origin, url.origin, 'Spaces API must be same-origin');
  assert.equal(api.pathname, spacesAPI + '/', 'Spaces mount API mismatch');
  assert.equal(api.search, '', 'Spaces API query is not allowed');
  assert.equal(api.hash, '', 'Spaces API hash is not allowed');
  const generation = await mount.getAttribute('data-extension-session-generation');
  assert(generation, 'Spaces mount has no native session');
  assert.equal(generation, workspaceGeneration, 'Workspace and Spaces must share native session authority');
  assert.equal(new URL(page.url()).hash, hashBefore, 'Native fragment must be preserved');
  return generation;
}

export async function spacesContent(page: Page): Promise<Page> {
  // The source-only workspace fixture deliberately mounts Spaces at `/` without
  // Forgejo's extension router. Keep this exact fixture opt-in local to tests.
  if (await page.locator('meta[name="soda-component-fixture"][content="spaces"]').count()) return page;
  await nativeSpacesMount(page);
  return page;
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
