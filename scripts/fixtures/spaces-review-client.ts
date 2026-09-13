import {mountSpacesPage} from '../../frontend/spaces/sodaspaces-page.js';
import {parseLayout} from '../../frontend/spaces/sodaspaces-layout.js';
import {object} from '../../frontend/spaces/sodaspaces-api.js';
// Production always uses WSS. This development-only entry maps its own loopback
// mock transport to WS; no production module or remote destination is changed.
const NativeWebSocket = WebSocket;
class FixtureWebSocket extends NativeWebSocket {
  constructor(input: string | URL, protocols?: string | string[]) {
    const target = new URL(input);
    if (target.host !== location.host || !target.pathname.startsWith('/-/soda/api/environments/')) throw Error('Fixture socket must stay local');
    target.protocol = location.protocol === 'http:' ? 'ws:' : 'wss:';
    super(target, protocols);
  }
}
window.WebSocket = FixtureWebSocket;
const root = document.querySelector<HTMLElement>('#spaces-fixture-mount');
const scenario = document.querySelector<HTMLSelectElement>('#fixture-scenario');
const theme = document.querySelector<HTMLSelectElement>('#fixture-theme');
const fault = document.querySelector<HTMLSelectElement>('#fixture-fault');
const notice = document.querySelector<HTMLElement>('#fixture-notice');
if (!root || !scenario || !theme || !fault || !notice) throw Error('Fixture shell missing');
const ui = {root, scenario, theme, fault, notice};
let revision = '';
let mounted: ReturnType<typeof mountSpacesPage> | undefined;
async function control(path: string, body?: unknown) {
  const response = await fetch('/_fixture/' + path, {method: body === undefined ? 'GET' : 'POST', headers: body === undefined ? {} : {'Content-Type': 'application/json'}, ...(body === undefined ? {} : {body: JSON.stringify(body)})});
  if (!response.ok) throw Error('Fixture control failed');
  return object(await response.json());
}
async function mount(reset = false) {
  mounted?.dispose();
  const state = await control(reset ? 'reset' : 'state', reset ? {scenario: ui.scenario.value} : undefined);
  revision = String(state.revision);
  ui.scenario.value = String(state.scenario);
  ui.fault.value = String(state.fault || 'healthy');
  if (reset || sessionStorage.getItem('spaces-fixture-generation') !== String(state.generation)) {
    sessionStorage.setItem('soda-spaces:v3:1', JSON.stringify(parseLayout(JSON.stringify(state.layout))));
    sessionStorage.setItem('spaces-fixture-generation', String(state.generation));
  }
  ui.notice.textContent = 'Simulated data and shell · no appliance connection';
  mounted = mountSpacesPage(ui.root, '1');
}
async function run(action: () => Promise<void>) {
  try {await action();} catch (error) {ui.notice.textContent = error instanceof Error ? error.message : 'Fixture unavailable';}
}
ui.scenario.addEventListener('change', () => void run(() => mount(true)));
document.querySelector('#fixture-reset')?.addEventListener('click', () => void run(() => mount(true)));
ui.fault.addEventListener('change', () => void run(async () => {
  await control('fault', {fault: ui.fault.value});
  await mount();
}));
ui.theme.value = localStorage.getItem('spaces-fixture-theme') || 'dark';
function setTheme() {
  document.documentElement.style.colorScheme = ui.theme.value;
  localStorage.setItem('spaces-fixture-theme', ui.theme.value);
}
ui.theme.addEventListener('change', setTheme); setTheme();
void run(() => mount());

setInterval(() => {
  if (!revision || document.hidden) return;
  void control('state').then(state => {if (String(state.revision) !== revision) location.reload();}).catch(() => undefined);
}, 1000);
