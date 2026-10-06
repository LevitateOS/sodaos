import {cancelIfHidden, send} from './sodaspaces-terminal-attachment.js';
import type {AttachmentInput, TerminalState} from './sodaspaces-terminal-attachment.js';
import type {ConnectionState} from './sodaspaces-attention.js';
import type {Renderer, TerminalView} from './sodaspaces-terminal.js';
import type {FitAddon} from '@xterm/addon-fit';
import type {ITerminalAddon} from '@xterm/xterm';

export type TerminalFit = Pick<FitAddon, 'fit'> & ITerminalAddon;

export interface ScreenInput {
  attachment: AttachmentInput;
  readTerminal: () => TerminalView | undefined;
  setTerminal: (terminal: TerminalView | undefined) => void;
  readFit: () => TerminalFit | undefined;
  setFit: (fit: TerminalFit | undefined) => void;
  readState: () => TerminalState;
  isViewVisible: () => boolean;
  isHidden: () => boolean;
  isHiddenOrInert: () => boolean;
  contains: (element: Element | null) => boolean;
  queryScreen: () => HTMLElement | null;
  focusControls: () => void;
  readHostHeight: () => number;
  readLastSize: () => string;
  setLastSize: (size: string) => void;
  setScreenVisible: (visible: boolean) => void;
  readMinimumSize: () => {width: number; height: number} | undefined;
  setMinimumSize: (minimum: {width: number; height: number}) => void;
  dispatchGeometry: (minimum: {width: number; height: number}) => void;
  setObserver: (observer: ResizeObserver) => void;
  updateComplete: () => Promise<boolean>;
  isLive: (generation: number) => boolean;
  detach: (message: string, stale?: boolean, reason?: ConnectionState) => void;
  hostStyles: () => CSSStyleDeclaration;
}

export function clearScreen(input: ScreenInput) {
  input.readTerminal()?.dispose();
  input.setTerminal(undefined);
  input.setFit(undefined);
  input.setLastSize('');
  input.queryScreen()?.replaceChildren();
  input.setScreenVisible(false);
}

function canFit(input: ScreenInput, screen: HTMLElement | null): screen is HTMLElement {
  return (
    !!input.readTerminal() &&
    !!input.readFit() &&
    input.isViewVisible() &&
    !input.isHidden() &&
    input.readState() === 'ready' &&
    !!screen?.isConnected &&
    !!screen.clientWidth &&
    !!screen.clientHeight
  );
}

export function resize(input: ScreenInput) {
  const screen = input.queryScreen();
  if (!canFit(input, screen)) return;
  input.readFit()!.fit();
  const {cols, rows} = input.readTerminal()!,
    fitted = cols + ':' + rows;
  if (cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300 && input.readLastSize() !== fitted) {
    input.setLastSize(fitted);
    send(input.attachment, {
      type: 'resize',
      cols,
      rows,
    });
  }
}

function geometryScreen(input: ScreenInput) {
  const screen = input.queryScreen();
  if (
    !input.readTerminal() ||
    input.readState() !== 'ready' ||
    !input.isViewVisible() ||
    !screen?.isConnected ||
    input.isHidden()
  )
    return;
  return screen;
}

function paneMinimum(input: ScreenInput, screen: HTMLElement) {
  const {cols, rows} = input.readTerminal()!;
  const grid = screen.querySelector<HTMLElement>('.xterm-screen')?.getBoundingClientRect();
  if (!grid?.width || !grid.height || !cols || !rows) return;
  const css = getComputedStyle(screen),
    scrollbar = screen.querySelector<HTMLElement>('.scrollbar.vertical')?.getBoundingClientRect().width || 0;
  return {
    width:
      Math.ceil((grid.width / cols) * 56 + scrollbar + parseFloat(css.paddingLeft) + parseFloat(css.paddingRight)) + 2,
    height:
      Math.ceil(
        (grid.height / rows) * 12 +
          input.readHostHeight() -
          screen.offsetHeight +
          parseFloat(css.paddingTop) +
          parseFloat(css.paddingBottom)
      ) + 2,
  };
}

function publishMinimum(input: ScreenInput, minimum: {width: number; height: number}) {
  if (
    minimum.width <= 0 ||
    minimum.height <= 0 ||
    (minimum.width === input.readMinimumSize()?.width && minimum.height === input.readMinimumSize()?.height)
  )
    return;
  input.setMinimumSize(minimum);
  input.dispatchGeometry(minimum);
}

function measureMinimum(input: ScreenInput) {
  const screen = geometryScreen(input);
  if (!screen) return;
  const minimum = paneMinimum(input, screen);
  if (minimum) publishMinimum(input, minimum);
}

export async function awaitScreen(input: ScreenInput, n: number) {
  input.setScreenVisible(true);
  await input.updateComplete();
  if (!input.isLive(n) || cancelIfHidden(input.attachment)) return;
  const screen = input.queryScreen();
  if (!screen) throw Error('Missing terminal screen');
  await document.fonts.ready;
  if (!input.isLive(n) || cancelIfHidden(input.attachment)) return;
  return screen;
}

function requiredToken(styles: CSSStyleDeclaration, name: string) {
  const value = styles.getPropertyValue(name).trim();
  if (!value) throw Error(`Missing terminal token ${name}`);
  return value;
}

function terminalTheme(styles: CSSStyleDeclaration) {
  return {
    background: requiredToken(styles, '--soda-terminal-screen'),
    foreground: requiredToken(styles, '--soda-terminal-text'),
    cursor: requiredToken(styles, '--soda-terminal-text'),
  };
}

function interceptControlFocus(input: ScreenInput, event: KeyboardEvent) {
  if (event.ctrlKey && event.shiftKey && event.key === 'Enter') {
    event.preventDefault();
    event.stopPropagation();
    input.focusControls();
    return false;
  }
  return true;
}

function sendInput(input: ScreenInput, n: number, screen: HTMLElement, data: string) {
  if (!input.isLive(n) || !input.isViewVisible() || input.isHidden() || !screen.contains(document.activeElement))
    return;
  if (data.length > 65536) {
    input.detach('Input too large. Nothing was replayed.');
    return;
  }
  const bytes = new TextEncoder().encode(data);
  for (let i = 0; i < bytes.length; i += 16384)
    if (
      !send(input.attachment, {
        type: 'input',
        data: btoa(String.fromCharCode(...bytes.subarray(i, i + 16384))),
      })
    )
      break;
}

export function openTerminal(
  input: ScreenInput,
  n: number,
  screen: HTMLElement,
  Terminal: Renderer['Terminal'],
  FitAddon: Renderer['FitAddon']
) {
  const styles = input.hostStyles();
  const fontSize = Number.parseFloat(requiredToken(styles, '--soda-font-mono-size')),
    lineHeight = Number(requiredToken(styles, '--soda-font-mono-line'));
  if (!Number.isFinite(fontSize) || fontSize <= 0 || !Number.isFinite(lineHeight) || lineHeight < 1)
    throw Error('Invalid terminal typography');
  const terminal = new Terminal({
    allowProposedApi: true,
    disableStdin: true,
    scrollback: 1000,
    windowOptions: {},
    convertEol: false,
    cols: 80,
    rows: 24,
    fontFamily: requiredToken(styles, '--soda-font-mono-family'),
    fontSize,
    lineHeight,
    theme: terminalTheme(styles),
  });
  input.setTerminal(terminal);
  const fit = new FitAddon();
  input.setFit(fit);
  terminal.loadAddon(fit);
  for (const code of [0, 1, 2, 8, 52]) terminal.parser.registerOscHandler(code, () => true);
  terminal.attachCustomKeyEventHandler((event) => interceptControlFocus(input, event));
  terminal.onData((data) => sendInput(input, n, screen, data));
  terminal.onRender(() => measureIfCurrent(input, n, terminal));
  terminal.open(screen);
  if (screen.clientWidth && screen.clientHeight) fit.fit();
  return terminal;
}

function measureIfCurrent(input: ScreenInput, n: number, terminal: TerminalView) {
  if (input.isLive(n) && input.readTerminal() === terminal) measureMinimum(input);
}

function shouldFocusScreen(input: ScreenInput) {
  return (
    document.hasFocus() &&
    document.visibilityState !== 'hidden' &&
    input.isViewVisible() &&
    !input.isHiddenOrInert() &&
    (document.activeElement === document.body || input.contains(document.activeElement))
  );
}

export async function screenReady(input: ScreenInput, n: number, terminal: TerminalView, screen: HTMLElement) {
  try {
    await input.updateComplete();
    if (!input.isLive(n) || input.readTerminal() !== terminal) return;
    if (shouldFocusScreen(input)) terminal.focus();
    if (!input.isLive(n) || input.readTerminal() !== terminal || !screen.isConnected) return;
    const observer = new ResizeObserver(() => resize(input));
    input.setObserver(observer);
    observer.observe(screen);
    resize(input);
  } catch {
    if (input.isLive(n)) input.detach('Terminal rendering failed. No input or creation was replayed.');
  }
}
