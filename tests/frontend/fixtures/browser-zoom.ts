import {mkdir, mkdtemp, rm, writeFile} from 'node:fs/promises';
import {resolve} from 'node:path';
import {chromium, type Page} from 'playwright';

type ChromeZoom = {
  tabs: {
    query(query: object): Promise<Array<{id?: number; url?: string}>>;
    setZoom(id: number, factor: number): Promise<void>;
    getZoom(id: number): Promise<number>;
  };
};

/** Chromium's actual per-tab zoom, in a disposable extension/profile. */
export async function browserZoomFixture() {
  const artifacts = resolve(import.meta.dirname, '../../../.artifacts');
  await mkdir(artifacts, {recursive: true});
  const profile = await mkdtemp(resolve(artifacts, 'zoom-'));
  const extension = resolve(profile, 'extension');
  const temporary = resolve(profile, 'tmp');
  await mkdir(extension);
  await mkdir(temporary);
  await writeFile(
    resolve(extension, 'manifest.json'),
    JSON.stringify({
      manifest_version: 3,
      name: 'Local browser zoom verification',
      version: '1.0',
      permissions: ['tabs'],
      background: {service_worker: 'worker.js'},
    })
  );
  await writeFile(resolve(extension, 'worker.js'), 'chrome.runtime.onInstalled.addListener(() => {});');
  const context = await chromium.launchPersistentContext(profile, {
    headless: true,
    channel: 'chromium',
    chromiumSandbox: true,
    viewport: {width: 1440, height: 1000},
    ignoreDefaultArgs: ['--disable-extensions'],
    args: [`--disable-extensions-except=${extension}`, `--load-extension=${extension}`],
    env: {...process.env, TMPDIR: temporary},
  });
  const worker = context.serviceWorkers()[0] ?? (await context.waitForEvent('serviceworker'));
  return {
    context,
    async setZoom(page: Page, factor: 1 | 2 | 4) {
      const observed = await worker.evaluate(
        async ({url, zoom}) => {
          const {chrome} = globalThis as unknown as {chrome: ChromeZoom};
          const matches = (await chrome.tabs.query({})).filter((tab) => tab.url === url);
          if (matches.length !== 1 || matches[0]?.id === undefined) throw Error('Unique zoom subject missing');
          const id = matches[0].id;
          await chrome.tabs.setZoom(id, zoom);
          return chrome.tabs.getZoom(id);
        },
        {url: page.url(), zoom: factor}
      );
      if (Math.abs(observed - factor) > 0.001) throw Error('Browser zoom setting differs from request');
      await page.waitForFunction((zoom) => Math.abs(devicePixelRatio - zoom) < 0.001, factor);
      return page.evaluate(() => ({width: innerWidth, height: innerHeight, dpr: devicePixelRatio}));
    },
    async close() {
      await context.close();
      await rm(profile, {recursive: true, force: true});
    },
  };
}
