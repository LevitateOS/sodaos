// Design-only state. No API, WebSocket, authentication, storage or shell execution.
// Rerendering sample text here is not the production terminal lifecycle design.
type Status = 'connected' | 'kept' | 'offline' | 'elsewhere' | 'uncertain' | 'ended';
type Layout = 'single' | 'columns' | 'rows' | 'grid';
type Project = 'api' | 'web';
interface Session { id: string; project: Project; name: string; status: Status; retention: string }
interface Group { tabs: string[]; active: string | null }
const projectNames: Record<Project, string> = {api: 'acme/api', web: 'acme/web'};
const samples: Record<string, string> = {
  shell: 'alex ~/api\n$ git status --short\n M src/routes.ts\n M tests/routes.test.ts\n\n$ git diff --stat\n src/routes.ts         | 12 +++++++-----\n tests/routes.test.ts  |  8 ++++++++\n 2 files changed, 15 insertions(+), 5 deletions(-)\n\n$ ',
  tests: 'alex ~/api\n$ go test ./...\n\nok   example/api/auth      0.142s\nok   example/api/routes    0.318s\nok   example/api/storage   0.096s\n\n$ ',
  server: 'alex ~/api\n$ ./dev-server\n\nListening on 127.0.0.1:8080\n\n14:06:32  GET  /health       200\n14:06:38  GET  /v1/projects  200\n14:06:41  GET  /v1/projects  200\n14:07:02  GET  /health       200\n',
  dev: 'alex ~/web\n$ bun run dev\n\nDevelopment server ready\n\n  Local:   http://localhost:3000\n\n14:05:24  connected\n14:06:12  updated src/app.ts\n14:06:31  updated src/styles.css\n\nWaiting for changes…\n',
  git: 'alex ~/web\n$ git log --oneline -4\n\na21f0e7 Refine project navigation\nb721d44 Add empty workspace state\nc2e1a98 Improve keyboard focus\nd8af124 Initial interface\n\n$ ',
  notes: 'alex ~/web\n$ ls\n\nREADME.md    bun.lock    src\npackage.json tests      tsconfig.json\n\n$ ',
};
let sessions: Session[] = [];
let groups: Group[] = [];
let focused = 0;
let layout: Layout = 'columns';
let maximized = false;
let sequence = 6;
const collapsed = new Set<Project>();
const canvas = document.getElementById('canvas');
const list = document.getElementById('project-list');
const search = document.getElementById('search');
const scene = document.getElementById('scene');
const layoutControl = document.getElementById('layout');
const paneSwitch = document.getElementById('pane-switch');
const workspace = document.getElementById('workspace');
const navToggle = document.getElementById('toggle-nav');
const dialog = document.getElementById('dialog');
const dialogTitle = document.getElementById('dialog-title');
const dialogBody = document.getElementById('dialog-body');
const dialogActions = document.getElementById('dialog-actions');
const summary = document.getElementById('summary');
const notice = document.getElementById('notice');
const inspector = document.getElementById('inspector');
if (!(canvas instanceof HTMLElement) || !(list instanceof HTMLElement) || !(search instanceof HTMLInputElement) ||
    !(scene instanceof HTMLSelectElement) || !(layoutControl instanceof HTMLSelectElement) || !(paneSwitch instanceof HTMLSelectElement) ||
    !(workspace instanceof HTMLElement) || !(navToggle instanceof HTMLButtonElement) || !(dialog instanceof HTMLDialogElement) ||
    !(dialogTitle instanceof HTMLElement) || !(dialogBody instanceof HTMLElement) || !(dialogActions instanceof HTMLElement) ||
    !(summary instanceof HTMLElement) || !(notice instanceof HTMLElement) || !(inspector instanceof HTMLElement)) throw Error('Preview markup missing');
const ui = {canvas, list, search, scene, layoutControl, paneSwitch, workspace, navToggle, dialog, dialogTitle, dialogBody, dialogActions, summary, notice, inspector};
function el<K extends keyof HTMLElementTagNameMap>(tag: K, text = '', className = ''): HTMLElementTagNameMap[K] {
  const item = document.createElement(tag); item.textContent = text; item.className = className; return item;
}
function button(text: string, label: string, action: () => void, className = '') {
  const item = el('button', text, className); item.type = 'button'; item.setAttribute('aria-label', label); item.title = label;
  item.addEventListener('click', event => { event.stopPropagation(); action(); }); return item;
}
let noticeTimer: ReturnType<typeof setTimeout> | undefined;
function announce(text: string) {
  clearTimeout(noticeTimer); ui.notice.textContent = text; ui.notice.hidden = false;
  noticeTimer = setTimeout(() => { ui.notice.hidden = true; }, 5000);
}
function setNav(open: boolean) {
  ui.workspace.dataset.nav = String(open); ui.navToggle.setAttribute('aria-expanded', String(open));
  ui.navToggle.setAttribute('aria-label', open ? 'Hide projects and terminals' : 'Show projects and terminals');
  if (open) ui.search.focus();
}
function modal(title: string, content: HTMLElement[], actions: HTMLButtonElement[] = []) {
  ui.dialogTitle.textContent = title; ui.dialogBody.replaceChildren(...content);
  ui.dialogActions.replaceChildren(button('Cancel', 'Cancel', () => ui.dialog.close()), ...actions);
  if (!ui.dialog.open) ui.dialog.showModal();
}
function sessionByID(id: string) { const item = sessions.find(s => s.id === id); if (!item) throw Error('Unknown sample session'); return item; }
function currentGroup() { const item = groups[focused]; if (!item) throw Error('Unknown sample pane'); return item; }
function choose(id: string) {
  const existing = groups.findIndex(g => g.tabs.includes(id));
  if (existing >= 0) focused = existing;
  else currentGroup().tabs.push(id);
  currentGroup().active = id;
  if (matchMedia('(max-width:800px)').matches) setNav(false);
  render(); document.getElementById('tab-' + id)?.focus();
}
function hide(id: string) {
  for (const group of groups) {
    group.tabs = group.tabs.filter(tab => tab !== id);
    if (group.active === id) group.active = group.tabs[0] ?? null;
  }
  const item = sessionByID(id);
  if (item.status === 'connected') { item.status = 'kept'; item.retention = '30m'; }
  render(); announce(`Preview: ${item.name} hidden, not ended. Find it under ${projectNames[item.project]}.`);
}
function changeLayout(next: Layout) {
  layout = next; maximized = false;
  const count = next === 'single' ? 1 : next === 'grid' ? 4 : 2;
  while (groups.length < count) groups.push({tabs: [], active: null});
  const excess = groups.splice(count); const first = groups[0];
  if (!first) throw Error('Missing first pane');
  for (const group of excess) first.tabs.push(...group.tabs);
  first.active ??= first.tabs[0] ?? null;
  focused = Math.min(focused, count - 1); render();
}
function newTerminal(project?: Project) {
  const choices: Project[] = project ? [project] : ['api', 'web'];
  modal('New terminal', [el('p', 'Choose its project. This preview adds a sample tab; it does not create a shell.'), ...choices.map(p => {
    const choice = button(projectNames[p], `Create sample terminal in ${projectNames[p]}`, () => {
      sequence++; const item: Session = {id: 'sample-new-' + sequence, project: p, name: 'Terminal ' + sequence, status: 'connected', retention: ''};
      sessions.push(item); ui.dialog.close(); choose(item.id); announce('Sample terminal added. No project or shell was created.');
    }, 'choice'); choice.append(el('small', 'Running · joined')); return choice;
  })]);
}
function sessionMenu(item: Session, index: number) {
  const rename = button('Rename…', 'Rename terminal', () => {
    const input = el('input'); input.value = item.name; input.maxLength = 40; input.setAttribute('aria-label', 'Terminal name');
    modal('Rename terminal', [el('p', projectNames[item.project] + ' · sample label only'), input], [button('Save label', 'Save label', () => {
      const value = input.value.trim(); if (!value) { input.focus(); return; }
      item.name = value; ui.dialog.close(); render();
    }, 'primary')]); input.select();
  }, 'menu-action');
  const move = button('Move to another pane', 'Move to another pane', () => {
    const target = (index + 1) % groups.length; const from = groups[index]; const to = groups[target];
    if (!from || !to || from === to) return;
    from.tabs = from.tabs.filter(id => id !== item.id); if (from.active === item.id) from.active = from.tabs[0] ?? null;
    to.tabs.push(item.id); to.active = item.id; focused = target; ui.dialog.close(); render();
  }, 'menu-action'); move.disabled = groups.length < 2;
  const keep = button('Keep for two hours', 'Keep for two hours', () => {
    item.retention = '2h'; if (item.status === 'connected') item.status = 'kept'; ui.dialog.close(); render(); announce('Preview: retention changed. Real deadlines also respect authentication expiry.');
  }, 'menu-action'); keep.disabled = !['connected', 'kept', 'offline'].includes(item.status);
  const end = button('End terminal…', 'End terminal', () => {
    modal('End ' + item.name + '?', [el('p', `This ends only ${item.name} in ${projectNames[item.project]}. Unsaved editor state and processes inside this terminal will be lost.`), el('p', 'Files and independent services remain. This is a simulated confirmation; no real process will be touched.')], [button('End sample terminal', 'End sample terminal', () => {
      item.status = 'ended'; ui.dialog.close(); render(); announce('Sample terminal ended. Other sample sessions were not changed.');
    }, 'danger')]);
  }, 'menu-action danger'); end.disabled = item.status === 'uncertain' || item.status === 'ended';
  modal(item.name + ' · ' + projectNames[item.project], [rename, move, keep, end]);
}
function projectDetails(project: Project) {
  const facts = el('dl', '', 'details-list');
  for (const [key, value] of [['Project', projectNames[project]], ['Environment', 'Running (sample)'], ['Your account', 'alex'], ['Scope', 'Shared project']]) facts.append(el('dt', key), el('dd', value));
  const stop = button('Stop environment…', 'Stop environment (not simulated)', () => {}, 'danger'); stop.disabled = true;
  const heading = el('div', '', 'dialog-heading');
  heading.append(el('h2', projectNames[project]), button('×', 'Close project details', () => {
    ui.inspector.hidden = true; ui.workspace.dataset.inspector = 'false';
    const trigger = Array.from(document.querySelectorAll('button')).find(b => b.getAttribute('aria-label') === `Details for ${projectNames[project]}`);
    trigger?.focus();
  }, 'icon-button'));
  ui.inspector.replaceChildren(heading, el('p', 'Project details · sample', 'inspector-subtitle'), facts,
    el('p', 'Project-wide Stop and access management belong here, away from everyday terminal controls. They are not simulated in this preview.'), stop);
  ui.inspector.hidden = false; ui.workspace.dataset.inspector = 'true'; heading.querySelector('button')?.focus();
}
function rowState(item: Session) {
  if (item.status === 'connected') return groups.some(g => g.active === item.id) ? 'In view' : 'Open';
  return {kept: 'Kept · ' + item.retention, offline: 'Offline', elsewhere: 'Elsewhere', uncertain: 'Unconfirmed', ended: 'Ended'}[item.status];
}
function renderNavigator() {
  const query = ui.search.value.toLowerCase().trim(); ui.list.replaceChildren(el('h2', 'MY TERMINALS', 'section-label'));
  let matches = 0;
  for (const project of ['api', 'web'] as const) {
    const items = sessions.filter(s => s.project === project && (projectNames[project] + ' ' + s.name).toLowerCase().includes(query));
    if (!items.length) continue;
    matches += items.length;
    const section = el('section', '', 'project'); const heading = el('div', '', 'project-heading');
    const expand = button((collapsed.has(project) ? '›  ' : '⌄  ') + projectNames[project], `Toggle ${projectNames[project]} terminals`, () => {
      if (collapsed.has(project)) collapsed.delete(project); else collapsed.add(project); renderNavigator();
    }); expand.setAttribute('aria-expanded', String(!collapsed.has(project)));
    heading.append(expand, button('+', `New terminal in ${projectNames[project]}`, () => newTerminal(project), 'icon-button'), button('⋯', `Details for ${projectNames[project]}`, () => projectDetails(project), 'icon-button'));
    const state = el('div', '', 'project-state'); state.append(el('span', '', 'dot'), el('span', 'Environment running · ' + items.length + ' sessions')); section.append(heading, state);
    if (!collapsed.has(project)) for (const item of items) {
      const row = button('', `${projectNames[project]} · ${item.name} · ${rowState(item)}`, () => choose(item.id), 'session-row');
      row.dataset.session = item.id; row.dataset.status = item.status; row.dataset.active = String(currentGroup().active === item.id);
      row.append(el('span', '>_', 'session-glyph'), el('span', item.name, 'session-name'), el('span', rowState(item), 'row-state')); section.append(row);
    }
    ui.list.append(section);
  }
  if (!matches) ui.list.append(el('p', 'No matching projects or terminal labels. Your open views have not changed.', 'empty-search'));
  if (!query || 'acme/docs'.includes(query)) {
    ui.list.append(el('h2', 'OTHER PROJECTS', 'section-label'));
    const docs = button('acme/docs', 'acme/docs — stopped sample project', () => announce('Preview: Start environment and New terminal are separate actions.'), 'session-row');
    docs.append(el('span', 'Stopped', 'row-state')); ui.list.append(docs);
  }
}
function renderStatus(item: Session) {
  const region = el('div', '', 'pane-status');
  const messages: Record<Exclude<Status, 'connected'>, [string, string]> = {
    kept: ['Kept · ' + item.retention + ' remaining', 'This terminal is retained. Continue deliberately to return to active work.'],
    offline: ['Connection lost · input paused', 'Your last sample screen stays visible. Reconnect the same terminal; keystrokes are not queued.'],
    elsewhere: ['Attached in another window', 'One writer per terminal. Detach in the other window before reconnecting here.'],
    uncertain: ['Native cleanup unconfirmed', 'This slot stays reserved. No replacement or automatic retry. Ask the operator to inspect the outcome.'],
    ended: ['This terminal ended', 'Its process cannot be restored. Open a new terminal explicitly when you are ready.'],
  };
  if (item.status === 'connected') return null;
  const [title, detail] = messages[item.status]; region.append(el('strong', title), el('p', detail));
  if (item.status === 'kept' || item.status === 'offline') region.append(button(item.status === 'kept' ? 'Continue working' : 'Reconnect existing', 'Simulate ' + (item.status === 'kept' ? 'Continue working' : 'Reconnect existing'), () => {
    const wasOffline = item.status === 'offline'; item.status = wasOffline ? 'kept' : 'connected'; item.retention = wasOffline ? '24m' : '';
    render(); announce('Preview only: no connection or lifetime request was sent.');
  }, 'primary'));
  if (item.status === 'kept') region.append(button('Keep for two hours', 'Simulate Keep for two hours', () => { item.retention = '2h'; render(); }));
  if (item.status === 'ended') region.append(button('New terminal in ' + projectNames[item.project], 'New terminal in ' + projectNames[item.project], () => newTerminal(item.project)));
  return region;
}
function renderPane(group: Group, index: number) {
  const pane = el('section', '', 'pane'); pane.dataset.focused = String(focused === index); pane.dataset.group = String(index);
  pane.hidden = maximized && focused !== index; pane.setAttribute('aria-label', 'Terminal pane ' + (index + 1));
  const top = el('div', '', 'pane-top'); const tabs = el('div', '', 'tabs'); tabs.setAttribute('role', 'tablist'); tabs.setAttribute('aria-label', 'Pane ' + (index + 1) + ' terminals');
  for (const id of group.tabs) {
    const item = sessionByID(id); const wrapper = el('div', '', 'tab-shell');
    const tab = button('', `${projectNames[item.project]} · ${item.name}`, () => choose(id), 'terminal-tab');
    tab.id = 'tab-' + id; tab.setAttribute('role', 'tab'); tab.setAttribute('aria-selected', String(group.active === id)); tab.setAttribute('aria-controls', 'panel-' + index); tab.tabIndex = group.active === id ? 0 : -1;
    tab.append(el('span', item.project, 'project-short'), document.createTextNode(item.name));
    tab.addEventListener('keydown', event => {
      if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
      event.preventDefault(); const position = group.tabs.indexOf(id);
      const next = event.key === 'Home' ? 0 : event.key === 'End' ? group.tabs.length - 1 : (position + (event.key === 'ArrowRight' ? 1 : -1) + group.tabs.length) % group.tabs.length;
      const nextID = group.tabs[next]; if (nextID) choose(nextID);
    });
    wrapper.append(tab, button('×', `Hide ${item.name} — keeps the terminal; does not End`, () => hide(id), 'hide-tab')); tabs.append(wrapper);
  }
  top.append(tabs, button('+', 'Choose project for new terminal in pane ' + (index + 1), () => { focused = index; newTerminal(); }, 'pane-plus')); pane.append(top);
  if (group.active) {
    const item = sessionByID(group.active); const context = el('div', '', 'context');
    context.append(el('span', 'alex @'), el('strong', projectNames[item.project]),
      button(maximized ? '↙' : '⤢', maximized ? 'Restore pane layout' : 'Maximize pane ' + (index + 1), () => {
        focused = index; maximized = !maximized; render();
      }, 'icon-button'),
      button('⋯', `Actions for ${item.name} in ${projectNames[item.project]}`, () => sessionMenu(item, index), 'icon-button'));
    const view = el('div'); view.id = 'panel-' + index; view.setAttribute('role', 'tabpanel'); view.setAttribute('aria-labelledby', 'tab-' + item.id); view.style.cssText = 'display:flex;flex-direction:column;min-height:0;flex:1';
    const status = renderStatus(item); if (status) view.append(status);
    if (item.status === 'uncertain' || item.status === 'elsewhere' || item.status === 'ended') {
      const empty = el('div', '', 'empty-pane'); empty.append(el('strong', 'No writable attachment'), el('p', 'Other project terminals remain independent.')); view.append(empty);
    } else {
      const output = el('pre', '', 'terminal-output'); output.tabIndex = 0; output.setAttribute('aria-label', 'Static example output — not an interactive shell');
      const sampleKey = item.id.replace('sample-api-', '').replace('sample-web-', '');
      const text = samples[sampleKey] ?? 'alex ~\n$ \n\nThis is a sample terminal view.\nNo shell has been started.';
      for (const line of text.split('\n')) output.append(el('span', line + '\n', line.startsWith('$') || line.startsWith('alex') ? 'prompt' : line.startsWith('ok ') ? 'code' : line.startsWith('14:') ? 'muted' : ''));
      view.append(output);
    }
    const foot = el('div', '', 'pane-footer'); foot.append(el('span', 'SAMPLE OUTPUT · NO LIVE SHELL'), el('span', item.status === 'connected' ? 'Connected · sample' : rowState(item), item.status === 'connected' ? 'input-state' : '')); view.append(foot); pane.append(context, view);
  } else {
    const empty = el('div', '', 'empty-pane'); empty.append(el('strong', 'A view, not a new shell'), el('p', 'Choose an existing terminal from the navigator, or create one explicitly.'), button('Choose terminal', 'Choose terminal for empty pane', () => setNav(true)), button('New terminal…', 'Choose a project for a new terminal', () => { focused = index; newTerminal(); })); pane.append(empty);
  }
  pane.addEventListener('click', () => {
    if (focused === index) return; focused = index;
    for (const child of ui.canvas.children) if (child instanceof HTMLElement) child.dataset.focused = String(child.dataset.group === String(index));
    ui.paneSwitch.value = String(index); renderNavigator();
  });
  return pane;
}
function render() {
  ui.summary.textContent = sessions.length + ' sample sessions · 2 projects';
  ui.layoutControl.value = layout; ui.canvas.dataset.layout = maximized ? 'single' : layout;
  ui.canvas.replaceChildren(...groups.map(renderPane)); renderNavigator();
  ui.paneSwitch.replaceChildren(...groups.map((g, index) => { const option = el('option', 'Pane ' + (index + 1) + (g.active ? ' · ' + sessionByID(g.active).project : '')); option.value = String(index); return option; })); ui.paneSwitch.value = String(focused);
}
function loadScene(away: boolean) {
  sessions = [
    {id: 'sample-api-shell', project: 'api', name: 'shell', status: away ? 'kept' : 'connected', retention: away ? '24m' : ''},
    {id: 'sample-api-tests', project: 'api', name: 'tests', status: away ? 'uncertain' : 'connected', retention: ''},
    {id: 'sample-api-server', project: 'api', name: 'server', status: away ? 'elsewhere' : 'connected', retention: ''},
    {id: 'sample-web-dev', project: 'web', name: 'dev', status: away ? 'offline' : 'connected', retention: ''},
    {id: 'sample-web-git', project: 'web', name: 'git', status: away ? 'ended' : 'connected', retention: ''},
    {id: 'sample-web-notes', project: 'web', name: 'shell', status: 'connected', retention: ''},
  ];
  groups = [{tabs: sessions.slice(0, 3).map(s => s.id), active: 'sample-api-shell'}, {tabs: sessions.slice(3).map(s => s.id), active: 'sample-web-dev'}];
  focused = 0; layout = 'columns'; maximized = false; collapsed.clear(); ui.search.value = '';
  ui.inspector.hidden = true; ui.workspace.dataset.inspector = 'false'; render();
}
ui.dialog.addEventListener('close', () => {
  // A sample rerender can replace the original dialog opener. Keep keyboard
  // focus in the selected tab instead of losing it to the document body.
  if (document.activeElement === document.body || document.activeElement === ui.dialog) {
    const active = currentGroup().active;
    if (active) document.getElementById('tab-' + active)?.focus();
    else ui.canvas.focus();
  }
});
ui.search.addEventListener('input', renderNavigator);
ui.scene.addEventListener('change', () => loadScene(ui.scene.value === 'away'));
ui.layoutControl.addEventListener('change', () => { const value = ui.layoutControl.value; if (value === 'single' || value === 'columns' || value === 'rows' || value === 'grid') changeLayout(value); });
ui.paneSwitch.addEventListener('change', () => { const index = Number(ui.paneSwitch.value); if (Number.isInteger(index) && groups[index]) { focused = index; render(); } });
ui.navToggle.addEventListener('click', () => setNav(ui.workspace.dataset.nav !== 'true'));
document.getElementById('switch-terminal')?.addEventListener('click', () => setNav(true));
document.getElementById('new-terminal')?.addEventListener('click', () => newTerminal());
document.getElementById('theme')?.addEventListener('click', event => {
  const light = document.documentElement.dataset.theme !== 'light'; document.documentElement.dataset.theme = light ? 'light' : 'dark';
  if (event.currentTarget instanceof HTMLButtonElement) { event.currentTarget.textContent = light ? 'Dark theme' : 'Light theme'; event.currentTarget.setAttribute('aria-label', 'Switch to ' + (light ? 'dark' : 'light') + ' theme'); }
});
for (const link of document.querySelectorAll('[data-native]')) link.addEventListener('click', event => { event.preventDefault(); announce('Native Forgejo navigation belongs here. This preview does not connect to Forgejo.'); });
loadScene(new URLSearchParams(location.search).get('scene') === 'away');
ui.scene.value = new URLSearchParams(location.search).get('scene') === 'away' ? 'away' : 'working';
if (new URLSearchParams(location.search).get('theme') === 'light') document.getElementById('theme')?.click();
if (matchMedia('(max-width:800px)').matches) setNav(false);
else ui.navToggle.setAttribute('aria-label', 'Hide projects and terminals');
export {};
