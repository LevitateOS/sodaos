import assert from "node:assert/strict";
import { test } from "bun:test";

import { createPayload, forgejoBrowserURL, providerURL, statusText, successMessage } from "./ui";

function formData(values: Record<string, string>) {
  return { get: (name: string) => values[name] ?? "" };
}

test("provider forms retain only the provider-native registration fields", () => {
  const payload = createPayload(
    formData({
      id: "forgejo-one",
      provider: "forgejo",
      registration_url: "https://external.invalid",
      registration_id: "33834eef-e758-48c4-a676-1745426747aa",
      forgejo_labels: "soda:host",
      registration_token: "provider-input",
    }),
  );
  assert.deepEqual(payload, {
    id: "forgejo-one",
    provider: "forgejo",
    registration_url: "",
    registration_id: "33834eef-e758-48c4-a676-1745426747aa",
    labels: "soda:host",
    registration_token: "provider-input",
  });
  assert.equal(Object.hasOwn(payload, "project_id"), false);
});

test("local status never claims provider availability or idle capacity", () => {
  assert.equal(statusText({ active: "active", sub: "running" }), "Listening");
  assert.equal(statusText({ active: "failed", sub: "failed" }), "Failed");
  assert.match(successMessage("remove", "one"), /provider/);
  assert.equal(successMessage("stop", "one"), "one was stopped.");
});

test("Forgejo links use the configured browser origin, never a guessed host port", () => {
  assert.equal(forgejoBrowserURL("https://forgejo.example.test"), "https://forgejo.example.test");
  assert.equal(forgejoBrowserURL("soda.lan"), "");
  assert.equal(forgejoBrowserURL("javascript:alert(1)"), "");
  assert.equal(providerURL("forgejo", "http://127.0.0.1:3000", "https://forgejo.example.test"), "https://forgejo.example.test");
  assert.equal(providerURL("github", "https://github.com/example/repo", "https://forgejo.example.test"), "https://github.com/example/repo");
});
