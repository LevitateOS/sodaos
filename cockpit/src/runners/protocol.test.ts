import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "bun:test";

import {
  actions,
  coordinatorCommand,
  coordinatorPath,
  decodeResponse,
  encodeRequest,
} from "./protocol";

test("manifest exposes one dedicated Cockpit Runners page", async () => {
  const manifest = JSON.parse(
    await readFile(new URL("../../soda-runners/manifest.json", import.meta.url), "utf8"),
  );
  const page = await readFile(new URL("../../soda-runners/index.html", import.meta.url), "utf8");
  assert.deepEqual(Object.keys(manifest.menu), ["index"]);
  assert.equal(manifest.menu.index.label, "Runners");
  assert.equal(manifest.bridges, undefined);
  assert.match(page, /src="\.\.\/base1\/cockpit\.js"/);
});

test("coordinator command contains only its executable and allow-listed action", () => {
  for (const action of actions)
    assert.deepEqual(coordinatorCommand(action), [coordinatorPath, action]);
  assert.throws(() => coordinatorCommand("shell"), /unsupported runner action/);
});

test("registration input is serialized only into the stdin payload", () => {
  const payload = {
    id: "forgejo-one",
    provider: "forgejo",
    registration_url: "http://127.0.0.1:3000",
    registration_id: "33834eef-e758-48c4-a676-1745426747aa",
    labels: "soda:host",
    registration_token: "provider-input",
  };
  assert.equal(encodeRequest("create", payload), `${JSON.stringify(payload)}\n`);
  assert.deepEqual(coordinatorCommand("create"), [coordinatorPath, "create"]);
});

test("list response requires exact count, listener, and one-slot capacity facts", () => {
  const response = {
    runners: [
      {
        id: "one",
        provider: "forgejo",
        registration_url: "http://127.0.0.1:3000",
        account: "soda-runner-one",
        architecture: "AArch64",
        version: "fixture",
        capacity: 1,
        service: { load: "loaded", active: "active", sub: "running", enabled: "enabled" },
      },
    ],
    runner_count: 1,
    active_listeners: 1,
    total_capacity: 1,
  };
  assert.deepEqual(decodeResponse("list", JSON.stringify(response)), response);
  assert.throws(
    () => decodeResponse("list", JSON.stringify({ ...response, total_capacity: -1 })),
    /invalid total_capacity/,
  );
  assert.throws(
    () => decodeResponse("list", JSON.stringify({ ...response, runner_count: 2 })),
    /inconsistent/,
  );
});
