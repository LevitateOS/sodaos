import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { EnvironmentDetail } from "./environments";
import { useSession } from "./session";
import type { Session } from "./api";
const alice: Session = { user: { id: "1", login: "alice", soda_display_name: "Alice" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("requires an explicit join and never reports a failed native join as membership", async () => {
  const fetcher = vi.fn(async (path: string, options: RequestInit) => {
    if (options.method === "POST") return new Response(JSON.stringify({ error: { code: "account_incomplete", message: "Native provisioning failed; membership was not recorded." } }), { status: 502, headers: { "Content-Type": "application/json" } });
    const value = path.endsWith("/members") ? { items: [] } : { environment: { id: "p1", name: "demo", repository: "alice/demo", provisioned: true }, observed: { running: true, ip: "10.89.0.2" }, native_unavailable: false, login: "", environment_administrator: true };
    return new Response(JSON.stringify(value), { headers: { "Content-Type": "application/json" } });
  });
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/environments/p1"]}><Routes><Route path="/environments/:id" element={<EnvironmentDetail session={alice} />} /></Routes></MemoryRouter>);
  const join = await screen.findByRole("button", { name: "Add me to this project" });
  expect(fetcher.mock.calls.some(([, options]) => options.method === "POST")).toBe(false);
  await userEvent.setup().click(join);
  await screen.findByText("Native provisioning failed; membership was not recorded.");
  expect(screen.queryByText(/Joined as/)).toBeNull();
  const write = fetcher.mock.calls.find(([, options]) => options.method === "POST");
  expect(write?.[0]).toBe("/api/environments/p1/join"); expect(write?.[1].body).toBe("{}");
  expect(fetcher.mock.calls.some(([path]) => path.endsWith("/connection"))).toBe(false);
});

it.each(["detail", "members"])("preserves own connection when %s cannot verify current ownership", async (failureAt) => {
  const fetcher = vi.fn(async (path: string) => {
    const value = path.endsWith("/members") ? { items: [{ user_id: "1", login: "linux-alice" }], authority_unavailable: failureAt === "members" }
      : path.endsWith("/connection") ? { login: "linux-alice", connection: { environment: { running: true, ip: "10.89.0.2" }, host_key: "public-test-key", fingerprint: "public-test-fingerprint" }, routing_verified: false }
        : { environment: { id: "p1", name: "demo", repository: "alice/demo", provisioned: true }, observed: { running: true }, login: "linux-alice", environment_administrator: failureAt !== "detail", authority_unavailable: failureAt === "detail" };
    return new Response(JSON.stringify(value), { headers: { "Content-Type": "application/json" } });
  });
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/environments/p1"]}><Routes><Route path="/environments/:id" element={<EnvironmentDetail session={alice} />} /></Routes></MemoryRouter>);
  await screen.findByText(/Current Forgejo ownership could not be verified/);
  await screen.findByText(/ssh linux-alice@10.89.0.2/);
  expect(screen.queryByText(/You may view the environment membership/)).toBeNull();
  expect(screen.queryByRole("button", { name: "Add me to this project" })).toBeNull();
});
