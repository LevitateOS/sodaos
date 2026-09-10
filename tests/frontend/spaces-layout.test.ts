import {test} from 'bun:test';
import assert from 'node:assert/strict';
import {emptyLayout, putEntry, selectTab, hideTab, forgetEntry, focusedPane, parseLayout, serializeLayout, migrateLayout, splitPane, moveTab, resizeSplit, consolidate, projectLayout, panes} from '../../frontend/spaces/sodaspaces-layout';
import type {WorkspaceLayout, LayoutEntry} from '../../frontend/spaces/sodaspaces-layout';
const key = (n: number) => n.toString(16).padStart(8, '0') + '-0000-0000-0000-000000000000';
const env = 'p' + '1'.repeat(24), terminal = 'a'.repeat(32), request = 'b'.repeat(32);
const entry: LayoutEntry = {key: key(2), environmentId: env, locator: {kind: 'pending', requestId: request}};
const original = () => selectTab(putEntry(emptyLayout(key(1)), entry), entry.key);

test('locator promotion preserves owner, pane and selection; serialized state contains no effects', () => {
  const layout = original(), before = structuredClone(layout);
  const promoted = putEntry(layout, {...entry, locator: {kind: 'existing', id: terminal}});
  assert.deepEqual(layout, before); assert.equal(promoted.entries[0]?.key, entry.key);
  assert.equal(promoted.tree, layout.tree); assert.equal(focusedPane(promoted).selected, entry.key);
  assert.deepEqual(parseLayout(serializeLayout(promoted)), promoted);
  assert.throws(() => putEntry(promoted, {...entry, locator: {kind: 'existing', id: request}}));
  assert.throws(() => putEntry(promoted, {...entry, environmentId: 'p' + '2'.repeat(24)}));
});
test('Hide retains locator, only confirmed Forget removes it; unsent New cannot be restored', () => {
  const layout = original(), hidden = hideTab(layout, entry.key);
  assert.equal(hidden.entries.length, 1); assert.equal(focusedPane(hidden).selected, null);
  assert.deepEqual(parseLayout(serializeLayout(hidden)), hidden);
  assert.equal(forgetEntry(hidden, entry.key).entries.length, 0);
  const draft = selectTab(putEntry(layout, {key: key(3), environmentId: env, locator: {kind: 'new'}}), key(3));
  const saved = parseLayout(serializeLayout(draft));
  assert.deepEqual(saved.entries, layout.entries); assert.equal(focusedPane(saved).selected, entry.key);
  assert.equal(draft.entries.length, 2);
});
test('v1 migration preserves stored order, hidden and pending work, without newest selection', () => {
  const v1 = {version: 1, entries: [{environmentId: env, requestId: request}, {environmentId: env, id: terminal, hidden: true}, {environmentId: env, id: 'c'.repeat(32)}]};
  let n = 0; const layout = migrateLayout(JSON.stringify(v1), () => key(++n));
  assert.deepEqual(layout.entries.map(e => e.locator.kind), ['pending', 'existing', 'existing']);
  assert.deepEqual(focusedPane(layout).tabs, [key(2), key(4)]);
  assert.equal(focusedPane(layout).selected, key(2)); assert.equal(layout.entries.length, 3);
  assert.throws(() => migrateLayout(JSON.stringify({...v1, entries: [...v1.entries, v1.entries[0]]}), () => key(++n)));
  assert.throws(() => migrateLayout(JSON.stringify({version: 1, entries: [{environmentId: env, id: 'pending'}]}), () => key(++n)));
});
for (const [name, change] of Object.entries<(layout: WorkspaceLayout) => unknown>({
  version: l => ({...l, version: 3}),
  'extra state': l => ({...l, transcript: 'private'}),
  'unknown owner': l => ({...l, tree: {...focusedPane(l), tabs: [key(99)]}}),
  'duplicate tab': l => ({...l, tree: {...focusedPane(l), tabs: [entry.key, entry.key]}}),
  'unselected nonempty pane': l => ({...l, tree: {...focusedPane(l), selected: null}}),
  'selected nonmember': l => ({...l, tree: {...focusedPane(l), selected: key(99)}}),
  'missing focused pane': l => ({...l, focused: key(99)}),
  'node aliases owner': l => ({...l, tree: {...focusedPane(l), key: entry.key}}),
  'duplicate owner': l => ({...l, entries: [...l.entries, entry]}),
  'cross-project locator alias': l => ({...l, entries: [...l.entries, {...entry, key: key(3), environmentId: 'p' + '2'.repeat(24)}]}),
  'unknown locator kind': l => ({...l, entries: [{...entry, locator: {kind: 'latest'}}]}),
  'mixed locator': l => ({...l, entries: [{...entry, locator: {kind: 'pending', requestId: request, id: terminal}}]}),
  'noncanonical locator': l => ({...l, entries: [{...entry, locator: {kind: 'existing', id: terminal.toUpperCase()}}]}),
  'credentials on entry': l => ({...l, entries: [{...entry, csrf: 'private'}]}),
  'unbounded sidebar': l => ({...l, sidebar: 100000}),
})) test('refuses stored ' + name, () => assert.throws(() => parseLayout(JSON.stringify(change(original())))));
for (const ratio of [0, 1, -0.1, 1.1, null, '0.5']) test('refuses split ratio ' + ratio, () => {
  const layout = original(); assert.throws(() => parseLayout(JSON.stringify({...layout, tree: {kind: 'split', key: key(4), axis: 'right', ratio, first: layout.tree, second: {kind: 'pane', key: key(5), tabs: [], selected: null}}})));
});
test('split/move/reorder/consolidate retain locators and collapse only emptied sources', () => {
  const initial = original(), split = splitPane(initial, key(1), 'right', key(3), key(4));
  assert.equal(panes(split.tree).length, 2); assert.equal(focusedPane(split).selected, null);
  assert.equal(split.entries, initial.entries);
  const moved = moveTab(split, entry.key, key(3));
  assert.equal(moved.tree.kind, 'pane'); assert.equal(moved.tree.key, key(3));
  const second = {...entry, key: key(5), locator: {kind: 'existing' as const, id: terminal}};
  const again = splitPane(selectTab(putEntry(moved, second), second.key), key(3), 'below', key(6), key(7));
  const two = moveTab(again, second.key, key(6)); assert.equal(panes(two.tree).length, 2);
  const combined = consolidate(two); assert.deepEqual(focusedPane(combined).tabs, [second.key, entry.key]);
  assert.deepEqual(combined.entries, two.entries); assert.deepEqual(parseLayout(serializeLayout(two)), two);
  assert.deepEqual(initial, original());
});
test('measured compact projection and maximize do not rewrite desired tree or ratios', () => {
  const layout = splitPane(original(), key(1), 'right', key(3), key(4)); layout.tree = resizeSplit(layout.tree, key(4), 0.6);
  const before = serializeLayout(layout), minimum = () => ({width: 500, height: 280});
  const wide = projectLayout(layout, {x: 0, y: 0, width: 1400, height: 800}, minimum);
  assert.equal(wide.panes.length, 2); assert.equal(wide.dividers.length, 1);
  assert(wide.panes.every(p => p.width >= 500 && p.height >= 280));
  const narrow = projectLayout(layout, {x: 0, y: 0, width: 900, height: 800}, minimum);
  assert(narrow.compact); assert.equal(narrow.panes[0]?.pane.key, key(3));
  const maximize = projectLayout(layout, {x: 0, y: 0, width: 1400, height: 800}, minimum, false, key(1));
  assert.equal(maximize.panes.length, 1); assert.equal(maximize.panes[0]?.pane.key, key(1));
  assert.equal(serializeLayout(layout), before);
  assert.deepEqual(projectLayout(layout, {x: 0, y: 0, width: 1400, height: 800}, minimum), wide);
});

test('capacity, byte and tree bounds refuse rather than truncate unknown locators', () => {
  let layout = emptyLayout(key(1));
  for (let n = 2; n <= 65; n++) layout = putEntry(layout, {key: key(n), environmentId: env, locator: {kind: 'existing', id: n.toString(16).padStart(32, '0')}});
  assert.equal(parseLayout(serializeLayout(layout)).entries.length, 64);
  assert.throws(() => putEntry(layout, {...entry, key: key(66)}));
  assert.throws(() => parseLayout(' '.repeat(32769)));
  assert.throws(() => migrateLayout(' '.repeat(16385), () => key(1)));
  let tree = layout.tree;
  for (let n = 0; n < 64; n++) tree = {kind: 'split', key: key(100 + 2 * n), axis: 'right', ratio: 0.5, first: tree, second: {kind: 'pane', key: key(101 + 2 * n), tabs: [], selected: null}};
  assert.throws(() => parseLayout(JSON.stringify({...layout, tree})));
});
