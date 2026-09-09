import {check, object, projectId, terminalID} from './sodaspaces-api.js';
import type {TerminalLocator} from './sodaspaces-terminal.js';

// Presentation locators, never native authority. Live handles/bindings stay in SodaSpaces.
export interface LayoutEntry {key: string; environmentId: string; locator: TerminalLocator}
export interface Pane {kind: 'pane'; key: string; tabs: string[]; selected: string | null}
export interface Split {kind: 'split'; key: string; axis: 'right' | 'below'; ratio: number; first: PaneTree; second: PaneTree}
export type PaneTree = Pane | Split;
export interface WorkspaceLayout {version: 2; entries: LayoutEntry[]; tree: PaneTree; focused: string; sidebar: number | null}
const localKey = (v: unknown): v is string => typeof v === 'string' && /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(v);
const size = (text: string) => new TextEncoder().encode(text).length;
export const layoutLimit = 64;
export function emptyLayout(key: string): WorkspaceLayout {
  check(localKey(key));
  return {version: 2, entries: [], tree: {kind: 'pane', key, tabs: [], selected: null}, focused: key, sidebar: 256};
}
export function panes(tree: PaneTree): Pane[] {return tree.kind === 'pane' ? [tree] : [...panes(tree.first), ...panes(tree.second)];}
export function paneFor(tree: PaneTree, key: string) {return panes(tree).find(pane => pane.tabs.includes(key));}
export function focusedPane(layout: WorkspaceLayout): Pane {const pane = panes(layout.tree).find(pane => pane.key === layout.focused); check(pane); return pane;}
function changePane(tree: PaneTree, key: string, change: (pane: Pane) => Pane): PaneTree {
  return tree.kind === 'pane' ? tree.key === key ? change(tree) : tree : {...tree, first: changePane(tree.first, key, change), second: changePane(tree.second, key, change)};
}
function withoutTab(tree: PaneTree, key: string): PaneTree {
  if (tree.kind === 'pane') {const tabs = tree.tabs.filter(tab => tab !== key); return {...tree, tabs, selected: tree.selected === key ? tabs[0] || null : tree.selected};}
  const first = withoutTab(tree.first, key), second = withoutTab(tree.second, key);
  // Only collapse a source made empty by this removal, not deliberately empty splits.
  if (first.kind === 'pane' && !first.tabs.length && paneFor(tree.first, key)) return second;
  if (second.kind === 'pane' && !second.tabs.length && paneFor(tree.second, key)) return first;
  return {...tree, first, second};
}
export function selectTab(layout: WorkspaceLayout, key: string, destination = layout.focused): WorkspaceLayout {
  check(layout.entries.some(entry => entry.key === key));
  const pane = paneFor(layout.tree, key) || panes(layout.tree).find(pane => pane.key === destination); check(pane);
  return {...layout, focused: pane.key, tree: changePane(layout.tree, pane.key, p => ({...p, tabs: p.tabs.includes(key) ? p.tabs : [...p.tabs, key], selected: key}))};
}
export function hideTab(layout: WorkspaceLayout, key: string): WorkspaceLayout {
  const tree = withoutTab(layout.tree, key), remaining = panes(tree), first = remaining[0]; check(first);
  return {...layout, tree, focused: remaining.some(p => p.key === layout.focused) ? layout.focused : first.key};
}
export function forgetEntry(layout: WorkspaceLayout, key: string): WorkspaceLayout {
  return {...hideTab(layout, key), entries: layout.entries.filter(entry => entry.key !== key)};
}
export function putEntry(layout: WorkspaceLayout, entry: LayoutEntry): WorkspaceLayout {
  check(localKey(entry.key) && projectId(entry.environmentId));
  const prior = layout.entries.find(item => item.key === entry.key);
  check(!prior || prior.environmentId === entry.environmentId);
  if (entry.locator.kind !== 'new') {
    const locator = entry.locator;
    check(terminalID(locator.kind === 'existing' ? locator.id : locator.requestId));
    check(!layout.entries.some(item => item.key !== entry.key && sameLocator(item.locator, locator)));
    if (prior?.locator.kind === 'existing') check(sameLocator(prior.locator, locator));
    if (prior?.locator.kind === 'pending' && locator.kind === 'pending') check(sameLocator(prior.locator, locator));
  } else check(!prior || prior.locator.kind === 'new');
  check(prior || layout.entries.length < layoutLimit);
  return {...layout, entries: prior ? layout.entries.map(item => item === prior ? entry : item) : [...layout.entries, entry]};
}
export function sameLocator(a: TerminalLocator, b: TerminalLocator) {
  return a.kind === 'existing' && b.kind === 'existing' ? a.id === b.id : a.kind === 'pending' && b.kind === 'pending' ? a.requestId === b.requestId : false;
}

export function splitPane(layout: WorkspaceLayout, paneKey: string, axis: Split['axis'], emptyKey: string, splitKey: string): WorkspaceLayout {
  check(panes(layout.tree).length < layoutLimit && panes(layout.tree).some(p => p.key === paneKey));
  const fresh: Pane = {kind: 'pane', key: emptyKey, tabs: [], selected: null};
  function divide(tree: PaneTree): PaneTree {
    if (tree.kind === 'pane') return tree.key === paneKey ? {kind: 'split', key: splitKey, axis, ratio: 0.5, first: tree, second: fresh} : tree;
    return {...tree, first: divide(tree.first), second: divide(tree.second)};
  }
  return {...layout, tree: divide(layout.tree), focused: emptyKey};
}
export function moveTab(layout: WorkspaceLayout, key: string, destination: string, before?: string): WorkspaceLayout {
  check(layout.entries.some(entry => entry.key === key) && panes(layout.tree).some(pane => pane.key === destination));
  const source = paneFor(layout.tree, key);
  const tree = source?.key === destination ? layout.tree : withoutTab(layout.tree, key);
  return {...layout, focused: destination, tree: changePane(tree, destination, pane => {
    const tabs = pane.tabs.filter(tab => tab !== key), index = before ? tabs.indexOf(before) : -1;
    tabs.splice(index < 0 ? tabs.length : index, 0, key); return {...pane, tabs, selected: key};
  })};
}
export function resizeSplit(tree: PaneTree, key: string, ratio: number): PaneTree {
  check(Number.isFinite(ratio) && ratio > 0 && ratio < 1);
  if (tree.kind === 'pane') return tree;
  return tree.key === key ? {...tree, ratio} : {...tree, first: resizeSplit(tree.first, key, ratio), second: resizeSplit(tree.second, key, ratio)};
}
export function consolidate(layout: WorkspaceLayout): WorkspaceLayout {
  const focused = focusedPane(layout), ordered = panes(layout.tree).flatMap(p => p.tabs);
  const tabs = focused.selected ? [focused.selected, ...ordered.filter(key => key !== focused.selected)] : ordered;
  return {...layout, tree: {...focused, tabs, selected: tabs[0] || null}};
}
export interface Area {x: number; y: number; width: number; height: number}
export interface Minimum {width: number; height: number}
export interface PaneArea extends Area {pane: Pane}
export interface DividerArea extends Area {key: string; axis: Split['axis']; ratio: number; minimum: number; maximum: number; origin: number; extent: number}
export interface Projection {panes: PaneArea[]; dividers: DividerArea[]; compact: boolean}
export const paneGap = 6;
export function minimumSize(tree: PaneTree, minimum: (pane: Pane) => Minimum): Minimum {
  if (tree.kind === 'pane') return minimum(tree);
  const a = minimumSize(tree.first, minimum), b = minimumSize(tree.second, minimum);
  return tree.axis === 'right' ? {width: a.width + b.width + paneGap, height: Math.max(a.height, b.height)} : {width: Math.max(a.width, b.width), height: a.height + b.height + paneGap};
}
export function projectLayout(layout: WorkspaceLayout, area: Area, minimum: (pane: Pane) => Minimum, single = false, maximized?: string): Projection {
  const needed = minimumSize(layout.tree, minimum), compact = single || needed.width > area.width || needed.height > area.height;
  const shown = panes(layout.tree).find(p => p.key === maximized) || (compact ? focusedPane(layout) : undefined);
  if (shown) return {panes: [{...area, pane: shown}], dividers: [], compact};
  const output: Projection = {panes: [], dividers: [], compact};
  function place(tree: PaneTree, rect: Area) {
    if (tree.kind === 'pane') {output.panes.push({...rect, pane: tree}); return;}
    const a = minimumSize(tree.first, minimum), b = minimumSize(tree.second, minimum), right = tree.axis === 'right';
    const extent = (right ? rect.width : rect.height) - paneGap, low = (right ? a.width : a.height) / extent, high = 1 - (right ? b.width : b.height) / extent;
    const ratio = Math.max(low, Math.min(high, tree.ratio)), length = extent * ratio;
    output.dividers.push({...rect, key: tree.key, axis: tree.axis, ratio, minimum: low, maximum: high, origin: right ? rect.x : rect.y, extent,
      ...(right ? {x: rect.x + length, width: paneGap} : {y: rect.y + length, height: paneGap})});
    place(tree.first, {...rect, ...(right ? {width: length} : {height: length})});
    place(tree.second, {...rect, ...(right ? {x: rect.x + length + paneGap, width: extent - length} : {y: rect.y + length + paneGap, height: extent - length})});
  }
  place(layout.tree, area); return output;
}

// Closed-shape, depth/node/byte-bounded parser. Never "repair" untrusted storage by
// silently dropping a locator, selecting a newest session, or overwriting its bytes.
export function parseLayout(text: string): WorkspaceLayout {
  check(size(text) <= 32768); const value = object(JSON.parse(text));
  check(Object.keys(value).sort().join(',') === 'entries,focused,sidebar,tree,version' && value.version === 2);
  check(Array.isArray(value.entries) && value.entries.length <= layoutLimit);
  const keys = new Set<string>(), locators = new Set<string>();
  const entries: LayoutEntry[] = value.entries.map((raw: unknown) => {
    const item = object(raw); check(Object.keys(item).sort().join(',') === 'environmentId,key,locator');
    check(localKey(item.key) && !keys.has(item.key) && projectId(item.environmentId)); keys.add(item.key);
    const rawLocator = object(item.locator); let locator: TerminalLocator;
    if (rawLocator.kind === 'existing') {check(Object.keys(rawLocator).sort().join(',') === 'id,kind' && terminalID(rawLocator.id)); locator = {kind: 'existing', id: rawLocator.id};}
    else {check(rawLocator.kind === 'pending' && Object.keys(rawLocator).sort().join(',') === 'kind,requestId' && terminalID(rawLocator.requestId)); locator = {kind: 'pending', requestId: rawLocator.requestId};}
    const identity = locator.kind === 'existing' ? 'id:' + locator.id : 'request:' + locator.requestId;
    check(!locators.has(identity)); locators.add(identity);
    return {key: item.key, environmentId: item.environmentId, locator};
  });
  const nodes = new Set<string>(), tabs = new Set<string>(); let leaves = 0;
  function readTree(raw: unknown, depth: number): PaneTree {
    check(depth <= 64 && nodes.size < 127); const node = object(raw);
    check(localKey(node.key) && !keys.has(node.key) && !nodes.has(node.key)); nodes.add(node.key);
    if (node.kind === 'pane') {
      check(++leaves <= layoutLimit && Object.keys(node).sort().join(',') === 'key,kind,selected,tabs');
      check(Array.isArray(node.tabs) && node.tabs.length <= layoutLimit);
      const ordered = node.tabs.map((tab: unknown) => {check(typeof tab === 'string' && keys.has(tab) && !tabs.has(tab)); tabs.add(tab); return tab;});
      check(ordered.length ? typeof node.selected === 'string' && ordered.includes(node.selected) : node.selected === null);
      return {kind: 'pane', key: node.key, tabs: ordered, selected: typeof node.selected === 'string' ? node.selected : null};
    }
    check(node.kind === 'split' && Object.keys(node).sort().join(',') === 'axis,first,key,kind,ratio,second');
    check((node.axis === 'right' || node.axis === 'below') && typeof node.ratio === 'number' && Number.isFinite(node.ratio) && node.ratio > 0 && node.ratio < 1);
    return {kind: 'split', key: node.key, axis: node.axis, ratio: node.ratio, first: readTree(node.first, depth + 1), second: readTree(node.second, depth + 1)};
  }
  const tree = readTree(value.tree, 1);
  check(typeof value.focused === 'string' && panes(tree).some(pane => pane.key === value.focused));
  check(value.sidebar === null || typeof value.sidebar === 'number' && Number.isFinite(value.sidebar) && value.sidebar >= 220 && value.sidebar <= 360);
  return {version: 2, entries, tree, focused: value.focused, sidebar: value.sidebar};
}
export function serializeLayout(layout: WorkspaceLayout): string {
  // An unsent draft has no native locator. Keep live drafts, not reload commands.
  let saved = layout;
  for (const entry of layout.entries) if (entry.locator.kind === 'new') saved = forgetEntry(saved, entry.key);
  const text = JSON.stringify(saved); parseLayout(text); return text;
}
export function migrateLayout(text: string, key: () => string): WorkspaceLayout {
  check(text.length <= 16384); const value = object(JSON.parse(text));
  check(Object.keys(value).sort().join(',') === 'entries,version' && value.version === 1 && Array.isArray(value.entries) && value.entries.length <= layoutLimit);
  let layout = emptyLayout(key()); let selected: string | undefined;
  for (const raw of value.entries) {
    const item = object(raw);
    check(Object.keys(item).every(k => ['environmentId', 'id', 'requestId', 'hidden'].includes(k)) && projectId(item.environmentId));
    check(item.hidden === undefined || typeof item.hidden === 'boolean');
    check(terminalID(item.id) && item.requestId === undefined || terminalID(item.requestId) && item.id === undefined);
    const owner = key(), locator: TerminalLocator = typeof item.id === 'string' ? {kind: 'existing', id: item.id} : {kind: 'pending', requestId: String(item.requestId)};
    layout = putEntry(layout, {key: owner, environmentId: item.environmentId, locator});
    if (!item.hidden) {layout = selectTab(layout, owner); selected ||= owner;}
  }
  if (selected) layout = selectTab(layout, selected);
  // Validate generated keys and collisions as well as imported fields.
  return parseLayout(serializeLayout(layout));
}
