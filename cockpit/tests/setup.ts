import { afterEach } from "bun:test";
import { JSDOM } from "jsdom";

// Install the DOM before importing Testing Library or React DOM.
const dom = new JSDOM("<!doctype html><html><body></body></html>", {
  url: "https://cockpit.example.test/",
  pretendToBeVisual: true,
});
const browserGlobals = new Set(["navigator", "Event", "CustomEvent", "EventTarget", "FormData", "AbortController", "AbortSignal"]);
for (const name of Object.getOwnPropertyNames(dom.window)) {
  if (name in globalThis && !browserGlobals.has(name)) continue;
  const value: unknown = Reflect.get(dom.window, name);
  Object.defineProperty(globalThis, name, { configurable: true, writable: true, value });
}
Object.defineProperty(globalThis, "window", { configurable: true, value: dom.window });
Object.defineProperty(globalThis, "document", { configurable: true, value: dom.window.document });
Object.defineProperty(globalThis, "IS_REACT_ACT_ENVIRONMENT", { configurable: true, writable: true, value: true });
globalThis.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
};
const { cleanup } = await import("@testing-library/react");
afterEach(cleanup);
