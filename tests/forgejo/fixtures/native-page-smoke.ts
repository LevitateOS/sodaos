// Test-only asset probe, served by the authorized stock Forgejo fixture. Import
// the real workspace without mounting it or dispatching any Soda API operation.
import '../../../frontend/spaces/sodaspaces-workspace.js';
import {Terminal} from './soda-terminal/xterm.mjs';
import {FitAddon} from './soda-terminal/addon-fit.mjs';

const mount = document.getElementById('soda-native-content');
if (!mount) throw new Error('Missing native page mount');
const screen = document.createElement('div');
screen.style.width = 'min(600px, 100%)';
screen.style.height = '240px';
mount.append(screen);
const terminal = new Terminal({fontSize: 14, allowProposedApi: false});
const fit = new FitAddon();
try {
  terminal.loadAddon(fit);
  terminal.open(screen);
  await document.fonts.ready;
  fit.fit();
  await new Promise<void>(resolve => terminal.write('Native page asset probe', resolve));
  if (terminal.cols < 20 || terminal.rows < 5 || terminal.buffer.active.getLine(0)?.translateToString(true) !== 'Native page asset probe') {
    throw new Error('Native xterm layout/output unavailable');
  }
  if (!customElements.get('soda-spaces')) throw new Error('Workspace module did not register');
  const viewport = screen.querySelector('.xterm-viewport');
  if (!viewport || getComputedStyle(viewport).position !== 'absolute') throw new Error('Native xterm stylesheet unavailable');
  mount.dataset.assetProbe = JSON.stringify({cols: terminal.cols, rows: terminal.rows, workspace: true, xterm: true});
} finally {
  terminal.dispose();
  screen.remove();
}
