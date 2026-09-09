import { initSync, parse } from "es-module-lexer";
import { dirname, relative, resolve } from "node:path";

/** Check the actual installed file graph; only Cockpit's public base API is external. */
export async function packageInventory(root: string): Promise<Record<string, string>> {
  initSync();
  const inventory: Record<string, string> = {};
  for (const name of [...new Bun.Glob("**/*").scanSync({ cwd: root, onlyFiles: true, dot: true })].sort()) {
    const path = resolve(root, name);
    if (/\.(?:tsx?|map|mjs)$|(?:^|\/)(?:node_modules|src|tests)(?:\/|$)/.test(name))
      throw new Error(`Non-runtime file: ${name}`);
    const content = await Bun.file(path).bytes();
    inventory[name] = new Bun.CryptoHasher("sha256").update(content).digest("hex");
    if (!/\.(?:html|css|js)$/.test(name)) continue;
    const text = new TextDecoder().decode(content);
    const references = name.endsWith(".html")
      ? [...text.matchAll(/(?:src|href)="([^"]+)"/g)].map((match) => match[1])
      : name.endsWith(".css")
        ? [...text.matchAll(/url\(\s*["']?([^\s"')]+)["']?\s*\)/g)].map((match) => match[1])
        : parse(text, name)[0]
            .filter((item) => item.d !== -2)
            .map((item) => {
              if (!item.n) throw new Error(`Non-static asset import in ${name}`);
              return item.n;
            });
    for (const reference of references) {
      if (reference === undefined) throw new Error(`Missing asset reference in ${name}`);
      if (name === "index.html" && reference === "../base1/cockpit.js") continue;
      if (reference.startsWith("data:")) continue;
      if (reference.startsWith("/") || /^[a-z]+:/i.test(reference))
        throw new Error(`External runtime asset: ${name}: ${reference}`);
      const target = resolve(dirname(path), reference.replace(/[?#].*$/, ""));
      if (relative(root, target).startsWith("..") || !(await Bun.file(target).exists()))
        throw new Error(`Asset escapes package: ${name}: ${reference}`);
    }
    if (/@vite\/client|localhost:5173|react-refresh/.test(text))
      throw new Error(`Development runtime in ${name}`);
  }
  for (const required of ["index.html", "manifest.json", "LICENSES.txt"])
    if (!inventory[required]) throw new Error(`Missing ${required}`);
  if (
    !Object.keys(inventory).some((name) => name.endsWith(".js")) ||
    !Object.keys(inventory).some((name) => name.endsWith(".css"))
  )
    throw new Error("Missing compiled browser assets");
  return inventory;
}
