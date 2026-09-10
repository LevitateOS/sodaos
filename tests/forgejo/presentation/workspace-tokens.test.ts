import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';

// Scoped to authored workspace presentation, not native overrides, vendor ANSI
// styles or canonical definitions. Geometry remains explicit at its owner.
export function visualFaults(source: string): string[] {
  const text = source.replace(/\/\*[\s\S]*?\*\//g, '')
    // Exact independent native Hide hit-region reservation, not a spacing role.
    .replace('#sodaspaces-drawer .soda-workspace-toolbar { padding-right: 80px; }', '');
  return [
    ...text.matchAll(/#[\da-f]{3,8}\b|\b(?:rgb|rgba|hsl|hsla|oklch)\s*\(/gi),
    ...text.matchAll(/var\([^,]+,\s*(?!(?:auto|none|inherit|initial|unset|currentColor|transparent)\b)[a-z]+\s*\)/gi),
    ...text.matchAll(/\bfont(?:-family|-size|-weight)?\s*:(?!\s*(?:var\(|inherit\b))[^;}\n]+/g),
    ...text.matchAll(/\bborder-radius\s*:\s*[1-9][\d.]*px/g),
    ...text.matchAll(/\bfontFamily\s*:\s*['"][^'"]+['"]|\b(?:fontSize|lineHeight)\s*:\s*[\d.]+/g),
    ...text.matchAll(/\b(?:color|background(?:-color)?)\s*:\s*['"]?(?!(?:var|inherit|transparent|currentColor|none|initial|unset)\b)[a-z]+(?=[;}"'\s])/gi),
  ].map(match => match[0]).concat([...text.matchAll(/\b(?:padding|margin|gap)(?:-(?:top|right|bottom|left|block|inline))?\s*:\s*([^;}]+)/g)]
    .filter(match => /\b[1-9][\d.]*px\b/.test((match[1] || '').replace(/var\([^)]*\)/g, '')))
    .map(match => match[0]));
}
test('workspace raw-visual guard rejects literals including fallbacks and inline templates', () => {
  for (const bad of ['color: #fff', 'color: var(--missing, #4583db)', 'style="color:rgb(1,2,3)"', 'font: 14px serif', 'font-family: Barlow', 'border-radius: 8px', 'padding: 8px', 'gap: 6px', 'fontFamily: "Mono"', 'fontSize: 14', 'background: red;', 'color: var(--missing, red)']) assert(visualFaults(bad).length, bad);
  assert.deepEqual(visualFaults('font: var(--soda-font-meta); width: 480px; padding: 0; border-radius: var(--soda-control-radius);'), []);
});
test('every authored workspace CSS/TS module consumes canonical visual roles', async () => {
  const root = resolve(import.meta.dir, '../../..'), directory = resolve(root, 'frontend/spaces');
  const files = [...new Bun.Glob('sodaspaces*.{css,ts}').scanSync(directory)];
  assert(files.length >= 11, 'empty or moved source inventory must be updated');
  for (const file of files) assert.deepEqual(visualFaults(await readFile(resolve(directory, file), 'utf8')), [], file);
});
