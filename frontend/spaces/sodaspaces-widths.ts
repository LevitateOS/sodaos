export function workspaceWidths(viewport: number, terminalMinimum: number, desired: number) {
  const minimum = Math.max(35, (terminalMinimum / Math.max(1, viewport)) * 100),
    maximum = Math.min(65, ((viewport - 480) / Math.max(1, viewport)) * 100);
  return {compact: minimum > maximum, minimum, maximum, actual: Math.max(minimum, Math.min(maximum, desired))};
}

export type ShellSurface = 'forge' | 'terminal';

export function shellPaneHidden(compact: boolean, surface: ShellSurface, pane: ShellSurface) {
  return compact && surface !== pane;
}
