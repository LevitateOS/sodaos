import test from 'node:test';
import assert from 'node:assert/strict';
import {chromium} from 'playwright';
import path from 'node:path';
import {buildForgejoModule} from '../../scripts/build-forgejo';
import {capturePageFixture} from '../../scripts/screenshot';
import {terminalMenu} from '../installed/sodaspaces-controls';
import {object} from '../../frontend/spaces/sodaspaces-api';
import type {} from './fixtures/native-workspace-fixture';

test('native Spaces entry boots one workspace with its original actor and native chrome', {skip: !process.env.SODA_PAGE_ORIGIN}, async t => {
 const origin = process.env.SODA_PAGE_ORIGIN || '', actor = process.env.SODA_PAGE_ACTOR || '';
 const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
 const page = await browser.newPage({ignoreHTTPSErrors: true, storageState: process.env.SODA_PAGE_STATE || ''});
 const errors: string[] = []; page.on('pageerror', error => errors.push(error.message));
 await page.route(origin + '/-/soda/api/**', async route => {
   const request = route.request(), pathname = new URL(request.url()).pathname;
   assert.equal(request.method(), 'GET'); assert.equal(request.headers()['x-soda-expected-user-id'], actor);
   const json = pathname.endsWith('/session') ? {user: {id: actor, login: 'soda-screenshot'}, csrf_token: 'synthetic-only', forgejo_url: origin} : {items: [], complete: true};
   await route.fulfill({json});
 });
 await page.goto(origin + '/?soda-view=spaces'); await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
 assert.equal(await page.locator('#soda-native-content').getAttribute('data-actor'), actor);
 assert.equal(await page.locator('soda-spaces').count(), 1);
 assert.equal(await page.locator('soda-terminal').count(), 0);
 assert.equal(await page.locator('#sodaspaces-root').count(), 0);
 assert.equal(await page.locator('#navbar').count(), 1);
 assert.equal(await page.locator('#navbar a[data-url="/user/logout"]').count(), 1);
 assert.deepEqual(errors, []);
});


test('native Spaces to drawer and back preserves exact sessions, retain and named End', {skip: !process.env.SODA_PAGE_ORIGIN}, async t => {
 const origin = process.env.SODA_PAGE_ORIGIN || '';
 const repository = object(JSON.parse(process.env.SODA_PAGE_REPOSITORY || '{}'));
 assert(typeof repository.id === 'string' && typeof repository.owner === 'string' && typeof repository.name === 'string');
 const query = new URLSearchParams({id: repository.id, owner: repository.owner, name: repository.name});
 const fixture = await buildForgejoModule(path.resolve('tests/frontend/fixtures/native-workspace-fixture.ts'), 'public/assets/native-workspace-fixture.js');
 const model = await buildForgejoModule(path.resolve('tests/frontend/fixtures/workspace-fixture.ts'), 'public/assets/workspace-fixture.js');
 const browser = await chromium.launch({headless: true, chromiumSandbox: true}); t.after(() => browser.close());
 const page = await browser.newPage({ignoreHTTPSErrors: true, storageState: process.env.SODA_PAGE_STATE || '', viewport: {width: 1440, height: 1000}});
 const errors: string[] = []; page.on('pageerror', error => errors.push(error.message));
 await page.route(origin + '/**', async route => {
   if (new URL(route.request().url()).pathname === '/assets/workspace-fixture.js') return route.fulfill({contentType: 'text/javascript', body: await model.text()});
   if (new URL(route.request().url()).pathname === '/assets/native-workspace-fixture.js') return route.fulfill({contentType: 'text/javascript', body: await fixture.text()});
   if (route.request().isNavigationRequest()) {
     const response = await route.fetch();
     const html = await response.text();
     assert(html.includes('id="navbar"'), 'required native host missing');
     return route.fulfill({response, body: html.replace('<head>', '<head><script type="module" src="/assets/native-workspace-fixture.js?' + query.toString().replaceAll('&', '&amp;') + '"></script>')});
   }
   await route.continue();
 });
 page.on('dialog', dialog => dialog.accept());
 await page.goto(origin + '/?soda-view=spaces');
 await page.getByRole('button', {name: 'Sessions', exact: true}).click();
 await page.locator('.soda-session-list button').filter({hasText: 'Build'}).click();
 await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
 if (process.env.SODA_PAGE_CAPTURES) {
   for (const theme of ['light', 'dark'] as const) for (const width of [390, 768, 1440]) {
     await page.setViewportSize({width, height: 900});
     await capturePageFixture(page, `spaces-${theme}-${width}`, '.soda-workspace-terminal:not([hidden]) .is-connected', theme);
   }
   await page.setViewportSize({width: 1440, height: 1000});
 }
 await terminalMenu(page, 'Keep for two hours');
 await page.waitForFunction(() => (window.nativeWorkspaceModel.spaces[0]?.terminals[0]?.retain_until || 0) > 0);
 const retained = await page.evaluate(() => window.nativeWorkspaceModel.spaces[0]?.terminals[0]?.retain_until);
 // Routed fixture: synthetic lifecycle; the connection parent separately proves
 // real BFCache. Restoration must recover this exact locator, never create.
 await page.evaluate(() => {
   window.dispatchEvent(new PageTransitionEvent('pagehide', {persisted: true}));
   window.dispatchEvent(new PageTransitionEvent('pageshow', {persisted: true}));
 });
 try {await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();}
 catch (error) {t.diagnostic(JSON.stringify({text: await page.locator('#soda-native-content').innerText(), errors})); throw error;}
 assert.equal(await page.locator('soda-spaces').count(), 1);
 assert.equal(await page.evaluate(() => window.nativeWorkspaceModel.sockets.flatMap(s => s.sent).filter(f => f.action === 'attach').at(-1)?.id), 'a'.repeat(32));
 assert.equal(await page.evaluate(() => window.nativeWorkspaceModel.spaces[0]?.terminals[0]?.retain_until), retained);
 await page.getByRole('button', {name: 'Open in drawer', exact: true}).click();
 await page.waitForURL('**/' + repository.owner + '/' + repository.name + '#sodaspaces');
 await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
 assert.equal(await page.locator('soda-spaces').count(), 1);
 assert.equal(await page.evaluate(() => window.nativeWorkspaceModel.sockets.flatMap(s => s.sent).filter(f => f.action === 'attach').at(-1)?.id), 'a'.repeat(32));
 await page.getByRole('link', {name: 'Open in Spaces', exact: true}).click();
 await page.waitForURL(origin + '/?soda-view=spaces');
 await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
 assert.equal(await page.locator('soda-spaces').count(), 1);
 assert.equal(await page.evaluate(() => window.nativeWorkspaceModel.sockets.flatMap(s => s.sent).filter(f => f.action === 'attach').at(-1)?.id), 'a'.repeat(32));
 assert.equal(await page.evaluate(() => window.nativeWorkspaceModel.spaces[0]?.terminals[0]?.retain_until), retained);
 await terminalMenu(page, 'Continue working');
 await page.waitForFunction(() => window.nativeWorkspaceModel.spaces[0]?.terminals[0]?.retain_until === 0);
 for (const name of ['Build', 'Edit']) {
   if (name === 'Edit') {
     await page.getByRole('button', {name: 'Sessions', exact: true}).click();
     await page.locator('.soda-session-list button').filter({hasText: name}).click();
     await page.locator('.soda-workspace-terminal:not([hidden]) .is-connected').waitFor();
   }
   await terminalMenu(page, 'End terminal…');
   const dialog = page.getByRole('dialog', {name: 'End terminal confirmation', exact: true});
   assert((await dialog.innerText()).includes(name));
   await dialog.getByRole('button', {name: 'End terminal', exact: true}).click();
   await page.waitForFunction(name => window.nativeWorkspaceModel.spaces[0]?.terminals.find(t => t.name === name)?.state === 'ended', name);
 }
 assert.deepEqual(await page.evaluate(() => window.nativeWorkspaceModel.calls.filter(c => c.body?.action === 'end').map(c => c.path.split('/').at(-1))), ['a'.repeat(32), 'b'.repeat(32)]);
 assert(!await page.evaluate(() => window.nativeWorkspaceModel.sockets.some(s => s.sent.some(f => f.action === 'create'))));
 assert.deepEqual(errors, []);
});
