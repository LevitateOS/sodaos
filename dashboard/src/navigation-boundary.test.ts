// @vitest-environment node
import { readdirSync, readFileSync } from "node:fs";
import { expect, it } from "vite-plus/test";

it("does not construct Forgejo frontend escape links in dashboard components", () => {
  // Keep the configured provider origin server-side for protocol use, not as
  // an app-owned navigation destination. This source guard supplements the
  // history DOM checks; it does not prove OAuth/ingress or content-link closure.
  const directory = new URL("./", import.meta.url);
  for (const file of readdirSync(directory).filter(name => name.endsWith(".tsx") && !name.endsWith(".test.tsx"))) {
    expect(readFileSync(new URL(file, directory), "utf8"), file).not.toContain("forgejo_url");
  }
});
