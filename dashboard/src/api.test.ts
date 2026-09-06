import { afterEach, expect, it, vi } from "vite-plus/test";
import { APIError, request } from "./api";

afterEach(() => vi.unstubAllGlobals());

it("keeps credentials same-origin, protects mutations and does not retry writes", async () => {
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: "conflict", message: "Refresh before retrying." } }), { status: 409, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetcher);
  await expect(request("/api/me/preferences", { method: "PATCH", body: { display_name: "Alice" }, csrf: "test-csrf" })).rejects.toMatchObject({ status: 409, code: "conflict" });
  expect(fetcher).toHaveBeenCalledTimes(1);
  const [path, options] = fetcher.mock.calls[0];
  expect(path).toBe("/api/me/preferences");
  expect(options.credentials).toBe("same-origin");
  expect(options.redirect).toBe("error");
  expect(options.headers.get("X-CSRF-Token")).toBe("test-csrf");
  expect(options.headers.get("Authorization")).toBeNull();
});

it("rejects HTML and foreign URLs rather than treating them as a session", async () => {
  const fetcher = vi.fn().mockResolvedValue(new Response("<html>login</html>", { headers: { "Content-Type": "text/html" } }));
  vi.stubGlobal("fetch", fetcher);
  await expect(request("https://other.example/api/session")).rejects.toThrow("Invalid API path");
  expect(fetcher).not.toHaveBeenCalled();
  await expect(request("/api/session")).rejects.toBeInstanceOf(APIError);
});

it("accepts a no-content logout response", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 204 })));
  await expect(request<void>("/api/session/logout", { method: "POST", body: {}, csrf: "test-csrf" })).resolves.toBeUndefined();
});
