import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import type {MatrixProject} from './sodaspaces-matrix-input';

export const spacesPage = '/-/extensions/pages/soda/spaces';
export const spacesAPI = spacesPage + '/api';

export async function nativeSpacesMount(page: Page): Promise<string> {
  const url = new URL(page.url());
  assert(url.pathname === spacesPage && !url.search && !url.hash, 'Native Spaces page handoff required');
  const mount = page.locator('[data-extension-page][data-extension-id="soda"][data-extension-page-id="spaces"]');
  await mount.waitFor();
  assert.equal(
    new URL((await mount.getAttribute('data-extension-api-base')) || '', url.origin).pathname,
    spacesAPI + '/',
    'Spaces mount API mismatch'
  );
  const generation = await mount.getAttribute('data-extension-session-generation');
  assert(generation, 'Spaces mount has no native session');
  return generation;
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
