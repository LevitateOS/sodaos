import assert from 'node:assert/strict';
import {mkdir, mkdtemp, readFile, rm} from 'node:fs/promises';
import {homedir} from 'node:os';
import {resolve} from 'node:path';
import {test} from 'node:test';
import {chromium} from 'playwright';

import {buildExtensionStylesheets} from '../../scripts/build-soda-extension.js';

const root = resolve(import.meta.dir, '..', '..');

// Native delivery serves the document value source raw; strip only the font
// and palette imports the browser would resolve over HTTP (palette is inlined
// below, fonts do not affect color computation).
async function documentCss(): Promise<string> {
  const palette = await readFile(resolve(root, 'assets/branding/theme/palette.css'), 'utf8');
  const components = await readFile(resolve(root, 'assets/branding/forgejo/components.css'), 'utf8');
  return `${palette}\n${components.replace(/@import[^;]+;/g, '')}`;
}

const expected = {
  light: {
    surface: 'rgb(255, 255, 255)',
    edge: 'rgb(133, 133, 133)',
    text: 'rgb(16, 16, 16)',
    action: 'rgb(223, 0, 27)',
    onAction: 'rgb(255, 255, 255)',
    link: 'rgb(223, 0, 27)',
  },
  dark: {
    surface: 'rgb(16, 16, 16)',
    edge: 'rgb(128, 128, 128)',
    text: 'rgb(255, 255, 255)',
    action: 'rgb(223, 0, 27)',
    onAction: 'rgb(255, 255, 255)',
    link: 'rgb(255, 101, 117)',
  },
} as const;

test('emitted dialog/action colors compute under native-selected schemes', {timeout: 120000}, async () => {
  // Short temp root: Chrome socket paths overflow the 108-char unix limit
  // under deep checkout TMPDIRs. Removed with the workdir below.
  const cacheRoot = resolve(homedir(), '.cache', 'soda-tests');
  await mkdir(cacheRoot, {recursive: true});
  const work = await mkdtemp(resolve(cacheRoot, 'e-colors-'));
  await mkdir(resolve(work, 'tmp'), {recursive: true});
  const context = await chromium.launchPersistentContext(resolve(work, 'pw-profile'), {
    channel: 'chrome',
    headless: true,
    chromiumSandbox: true,
    env: {...process.env, TMPDIR: resolve(work, 'tmp')},
  });
  try {
    const assets = resolve(work, 'assets');
    await buildExtensionStylesheets(assets);
    const emitted = (
      await Promise.all(
        ['components.css', 'sodaspaces-page.css'].map(async (name) => readFile(resolve(assets, name), 'utf8'))
      )
    ).join('\n');
    const doc = await documentCss();
    const [first] = context.pages();
    const page = first ?? (await context.newPage());
    for (const scheme of ['light', 'dark'] as const) {
      // Fountain's theme stylesheets select the scheme externally through
      // color-scheme; light-dark() reads that used value.
      await page.setContent(
        `<style>html{color-scheme:${scheme}}</style><style>${doc}</style><style>${emitted}</style>` +
          '<div class="soda-page"><div class="soda-page-container"><a id="soda-link" href="#">x</a></div></div>' +
          '<div class="soda-workspace-dialog" id="soda-dialog">x</div>' +
          '<div class="soda-drop-edge" id="soda-action">x</div>'
      );
      const colors = await page.evaluate(() => {
        const styleOf = (id: string) => {
          const element = document.getElementById(id);
          if (!element) throw new Error(`Missing fixture element ${id}`);
          const style = getComputedStyle(element);
          return {background: style.backgroundColor, border: style.borderTopColor, color: style.color};
        };
        return {dialog: styleOf('soda-dialog'), action: styleOf('soda-action'), link: styleOf('soda-link')};
      });
      const want = expected[scheme];
      assert.equal(colors.dialog.background, want.surface);
      assert.equal(colors.dialog.border, want.edge);
      assert.equal(colors.dialog.color, want.text);
      assert.equal(colors.action.background, want.action);
      assert.equal(colors.action.color, want.onAction);
      assert.equal(colors.link.color, want.link);
    }
    await page.close();
  } finally {
    await context.close();
    await rm(work, {recursive: true, force: true});
  }
});
