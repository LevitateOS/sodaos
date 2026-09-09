import { resolve } from "node:path";

export async function licenses(moduleIDs: string[], root: string) {
  const packages = new Map<string, string>();
  for (const id of moduleIDs.filter((id) => id.includes("/node_modules/"))) {
    const directory = id.match(/^(.*\/node_modules\/(?:@[^/]+\/)?[^/]+)\//)?.[1];
    if (!directory || !(await Bun.file(resolve(directory, "package.json")).exists())) continue;
    const metadata: unknown = await Bun.file(resolve(directory, "package.json")).json();
    if (!metadata || typeof metadata !== "object" || !("name" in metadata) ||
        typeof metadata.name !== "string" || !("version" in metadata) ||
        typeof metadata.version !== "string" ||
        ("license" in metadata && typeof metadata.license !== "string"))
      throw new Error(`Invalid package metadata: ${directory}`);
    const name = `${metadata.name}@${metadata.version}`;
    if (packages.has(name)) continue;
    const files = [...new Bun.Glob("*").scanSync({ cwd: directory, onlyFiles: true })].filter((file) =>
      /^(LICENSE|COPYING|NOTICE)(\.|$)/i.test(file),
    );
    const text = (await Promise.all(files.map((file) => Bun.file(resolve(directory, file)).text()))).join("\n");
    if (!text && !metadata.name.startsWith("@patternfly/"))
      throw new Error(`No license text for bundled ${name}`);
    packages.set(name, `${name} (${"license" in metadata ? metadata.license : "see license text"})\n${text}`);
  }
  const vendor = await Promise.all(
    [...new Bun.Glob("*.txt").scanSync({ cwd: resolve(root, "vendor"), onlyFiles: true })]
      .sort()
      .map(async (name) => `${name}\n${await Bun.file(resolve(root, "vendor", name)).text()}`),
  );
  return [
    ...vendor,
    ...[...packages].sort(([a], [b]) => a.localeCompare(b)).map(([, text]) => text),
  ].join("\n\n---\n\n");
}
