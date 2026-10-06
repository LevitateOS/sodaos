import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {chromium} from 'playwright';
import type {Page} from 'playwright';

// Explicit local opt-in. Native CSS plus the complete candidate CSS cascade,
// with small native markup contracts; no account, fixture or provider writes.
export const origin = process.env.SODA_FORGEJO_LAYOUT_ORIGIN;
// Inline same-directory concern stylesheets so assertions cover the delivered
// cascade, not just the composing entry file. Parent-directory imports keep
// their existing handling below.
async function inlineLocalImports(css: string): Promise<string> {
  const refs = [...css.matchAll(/@import\s+"\.\/([^"?]+\.css)[^;]*;\r?\n?/g)];
  let out = css;
  for (const [directive, name] of refs) {
    assert(name);
    const body = await readFile(new URL(`../../../assets/branding/forgejo/${name}`, import.meta.url), 'utf8');
    out = out.replace(directive, () => body);
  }
  return out;
}

export interface ComponentBrowser {
  page: Page;
  render(markup: string, theme?: string): Promise<void>;
  close(): Promise<void>;
}

export async function openComponentBrowser(): Promise<ComponentBrowser> {
  assert.equal(origin, 'http://localhost:3300');
  const header = await readFile(
    new URL('../../../frontend/forgejo/templates/custom/header.tmpl', import.meta.url),
    'utf8'
  );
  const files = [...header.matchAll(/\/soda\/forgejo\/([^?]+\.css)\?v=/g)].map(([, name]) => name);
  const styles = await Promise.all(
    files.map(async (name) =>
      inlineLocalImports(await readFile(new URL(`../../../assets/branding/forgejo/${name}`, import.meta.url), 'utf8'))
    )
  );
  const palette = await readFile(new URL('../../../assets/branding/theme/palette.css', import.meta.url), 'utf8');
  const browser = await chromium.launch({channel: 'chrome', headless: true, chromiumSandbox: true});
  const page = await browser.newPage({viewport: {width: 1654, height: 1000}});
  async function render(markup: string, theme = 'light') {
    await page.mouse.move(1600, 990);
    await page.setContent(`<link rel="stylesheet" href="${origin}/assets/css/index.css">
        <link rel="stylesheet" href="${origin}/assets/css/theme-forgejo-${theme}.css">
        <style>${palette}\n${styles.join('\n').replace(/@import[^;]+;/g, '')}</style>${markup}`);
  }
  return {page, render, close: () => browser.close()};
}
