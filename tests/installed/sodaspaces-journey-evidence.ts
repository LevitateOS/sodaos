import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import type {MatrixProject} from './sodaspaces-matrix-input';

export const spacesPage = '/workspace';
export const spacesAPI = '/-/native/pages/soda/spaces/api';
const workspacePage = '/workspace';
// The product mounts one retained `soda-spaces` owner inside its
// contribution root; retired host wrappers are not part of the page.
const spacesOwner = 'soda-spaces';

const workspaceRequestID = /^d[0-9a-f]{24}$/;
const workspaceSubURL = /^(?:\/[A-Za-z0-9._~-]+)*$/;

function admittedWorkspaceRest(rest: string): boolean {
  if (rest === workspacePage || rest === `${workspacePage}/new`) return true;
  const requests = `${workspacePage}/requests/`;
  if (rest.startsWith(requests))
    return rest.length === requests.length + 25 && workspaceRequestID.test(rest.slice(requests.length));
  const segments = rest.split('/');
  return (
    segments.length === 4 &&
    segments[0] === '' &&
    segments[1] !== '' &&
    segments[2] !== '' &&
    segments[3] === 'workspace'
  );
}

export async function nativeSpacesMount(page: Page): Promise<string> {
  const hashBefore = new URL(page.url()).hash;
  const owner = page.locator(spacesOwner);
  await owner.waitFor();
  assert.equal(await owner.count(), 1, 'Exactly one native Workspace owner required');
  const mount = owner.locator('xpath=ancestor::*[@data-extension-id="soda"][1]');
  await mount.waitFor();
  const subURL = (await mount.getAttribute('data-app-sub-url')) ?? '';
  assert(workspaceSubURL.test(subURL), 'Native sub-URL must be admitted');
  const url = new URL(page.url());
  assert(url.pathname.startsWith(subURL), 'Native workspace prefix mismatch');
  const rest = subURL === '' ? url.pathname : url.pathname.slice(subURL.length);
  assert(admittedWorkspaceRest(rest), 'Native workspace shape required');
  const api = new URL((await mount.getAttribute('data-extension-api-base')) || '', url.origin);
  assert.equal(api.origin, url.origin, 'Spaces API must be same-origin');
  assert.equal(api.pathname, `${subURL}${spacesAPI}/`, 'Spaces mount API mismatch');
  assert.equal(api.search, '', 'Spaces API query is not allowed');
  assert.equal(api.hash, '', 'Spaces API hash is not allowed');
  const generation = await mount.getAttribute('data-extension-session-generation');
  assert(generation, 'Spaces mount has no native session');
  assert.equal(new URL(page.url()).hash, hashBefore, 'Native fragment must be preserved');
  return generation;
}

/** Validated mount facts callers need for exact request paths. */
export async function mountedSpacesFacts(page: Page): Promise<{subURL: string; apiBase: string}> {
  const owner = page.locator(spacesOwner);
  await owner.waitFor();
  const mount = owner.locator('xpath=ancestor::*[@data-extension-id="soda"][1]');
  const subURL = (await mount.getAttribute('data-app-sub-url')) ?? '';
  const base = await mount.getAttribute('data-extension-api-base');
  assert(base, 'Spaces mount has no API base');
  const api = new URL(base, new URL(page.url()).origin);
  return {subURL, apiBase: api.pathname.replace(/\/$/, '')};
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
  intent?: string;
  explore_status?: number;
  decisions?: string[];
  factory?: string[];
  generation?: string;
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
