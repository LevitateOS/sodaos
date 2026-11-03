import {createRequire} from 'node:module';
import {dirname, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';

const root = resolve(import.meta.dir, '..');
export const factoryRoot = dirname(fileURLToPath(import.meta.resolve('@factory/product/package.json')));
export const factorySource = (path: string) => resolve(factoryRoot, path);

/** Fountain owns its public asset URLs and one Lit runtime. Factory owns the
 * implementation and shared design assets behind those host URLs. */
export function factoryBrowserImports(): Bun.BunPlugin {
  const host = createRequire(resolve(root, 'package.json'));
  return {
    name: 'factory-product-imports',
    setup(build) {
      build.onResolve({filter: /^lit(?:\/|$)/}, (args) => ({path: host.resolve(args.path)}));
      build.onResolve({filter: /(?:fonts|theme)\//}, (args) => {
        if (!args.path.startsWith('.')) return;
        const path = resolve(args.resolveDir, args.path.split('?')[0]!);
        for (const directory of ['fonts', 'theme']) {
          const prefix = resolve(root, 'assets/branding', directory) + '/';
          if (path.startsWith(prefix))
            return {path: factorySource(`assets/branding/${directory}/${path.slice(prefix.length)}`)};
        }
        return undefined;
      });
    },
  };
}
