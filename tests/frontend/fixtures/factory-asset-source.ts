import {factorySource} from '../../../scripts/factory-source.js';

/** Serve the real shared CSS import graph from raw-source browser fixtures. */
export function factoryAssetSource(pathname: string): Response | undefined {
  const style = /\/factory\/styles\/(base|feed|conversation|output|issues)\.css$/.exec(pathname)?.[1];
  const theme = /\/theme\/(palette|tokens)\.css$/.exec(pathname)?.[1];
  const font = /\/fonts\/([a-z0-9-]+\/[a-z0-9-]+\.woff2|fonts\.css)$/.exec(pathname)?.[1];
  const source = style
    ? `frontend/factory/styles/${style}.css`
    : theme
      ? `assets/branding/theme/${theme}.css`
      : font
        ? `assets/branding/fonts/${font}`
        : undefined;
  return source
    ? new Response(Bun.file(factorySource(source)), {
        headers: {'Content-Type': source.endsWith('.woff2') ? 'font/woff2' : 'text/css'},
      })
    : undefined;
}
