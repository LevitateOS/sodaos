import { createHash } from 'node:crypto';
import assert from 'node:assert/strict';
import { readFile, readdir, stat } from 'node:fs/promises';
import { test } from 'node:test';
const root = new URL('../../../appliance/forgejo/templates/', import.meta.url);
async function templates(dir = root, prefix = '') {
  const result = [];
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) result.push(...await templates(new URL(`${entry.name}/`, dir), `${prefix}${entry.name}/`));
    else if (entry.name.endsWith('.tmpl')) result.push(prefix + entry.name);
  }
  return result.sort();
}
test('presentation inventory covers every production override and its local callers', async () => {
  const { entries, upstream } = JSON.parse(await readFile(new URL('inventory.json', import.meta.url)));
  assert.equal(upstream, '15.0.7');
  assert.deepEqual(entries.map(e => e.template).sort(), await templates());
  const sources = new Map(await Promise.all(entries.map(async e => [e.template, await readFile(new URL(e.template, root), 'utf8')])));
  for (const entry of entries) {
    assert(entry.composition && entry.owner && entry.states.length && entry.verification);
    const callers = [...sources].filter(([, source]) => [...source.matchAll(/{{\s*template\s+"([^"]+)"/g)].some(m => m[1] + '.tmpl' === entry.template)).map(([name]) => name).sort();
    assert.deepEqual(entry.callers.sort(), callers, entry.template);
    const beforeRoles = sources.get(entry.template).replace(/ soda-p-(editor-container|form-host|form|title|heading|section|gap|toolbar)\b/g, '');
    assert.equal(createHash('sha256').update(beforeRoles).digest('hex'), entry.beforeRoleHash, `${entry.template}: ${entry.presentationReview ? "unreviewed structural change" : "change outside declared role additions"}`);
    for (const role of entry.roles) assert(sources.get(entry.template).includes(role), `${entry.template}: missing ${role}`);
  }
});

// When the authorized local embedded-template export is present, validate the
// upstream side of the caller inventory too. No network or export is implicit.
test('embedded native callers match the inventory when the local export is available', async t => {
  const upstream = new URL('../../../.artifacts/forgejo-presentation/upstream/templates/', import.meta.url);
  try { await stat(upstream); } catch (error) { if (error.code === 'ENOENT') return t.skip('requires local embedded 15.0.7 template export'); throw error; }
  const {entries} = JSON.parse(await readFile(new URL('inventory.json', import.meta.url)));
  const local = new Set(entries.map(e => e.template));
  const sources = await Promise.all((await templates(upstream)).filter(p => !local.has(p)).map(async p => [p, await readFile(new URL(p,upstream),'utf8')]));
  for (const entry of entries) {
    const callers = sources.filter(([,source]) => [...source.matchAll(/{{\s*template\s+"([^"]+)"/g)].some(m => m[1]+'.tmpl' === entry.template)).map(([p]) => p).sort();
    assert.deepEqual(entry.upstreamCallers,callers,entry.template);
  }
});
