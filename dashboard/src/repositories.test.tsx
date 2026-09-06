import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { CreateRepository, RepositoryDetail } from "./repositories";
import { useSession } from "./session";
import type { Session } from "./api";
const alice: Session = { user: { id: "1", login: "alice", soda_display_name: "Alice" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("creates only a Forgejo repository, without provisioning or joining", async () => {
  const fetcher = vi.fn(async () => new Response(JSON.stringify({ id: "7", name: "demo", owner: { id: "1", login: "alice" } }), { headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/new"]}><Routes><Route path="/new" element={<CreateRepository session={alice} />} /><Route path="/repositories/alice/demo" element={<h1>Created repository destination</h1>} /></Routes></MemoryRouter>);
  const user = userEvent.setup(); await user.type(screen.getByLabelText(/Repository name/), "demo"); await user.click(screen.getByRole("button", { name: "Create repository in Forgejo" }));
  await screen.findByRole("heading", { name: "Created repository destination" });
  expect(fetcher).toHaveBeenCalledTimes(1);
  expect(fetcher).toHaveBeenCalledWith("/api/forgejo/repositories", expect.objectContaining({ method: "POST", body: JSON.stringify({ name: "demo", description: "", private: true, auto_init: true }) }));
});
it("passes slash refs as query values and renders untrusted files only as text", async () => {
  const fetcher = vi.fn(async (path: string) => new Response(JSON.stringify(path.includes("/contents?") ? { items: [{ path: "index.html", name: "index.html", type: "file", size: "25", text: "<script>window.bad=1</script>", unavailable: false }] } : { id: "7", name: "demo", owner: { id: "1", login: "alice" }, default_branch: "main", empty: false }), { headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/repositories/alice/demo?ref=feature%2Fone&path=index.html"]}><Routes><Route path="/repositories/:owner/:repo" element={<RepositoryDetail session={alice} />} /></Routes></MemoryRouter>);
  await screen.findByText("<script>window.bad=1</script>");
  expect(document.querySelector("script")).toBeNull();
  expect(fetcher.mock.calls.some(([path]) => path.includes("ref=feature%2Fone"))).toBe(true);
});
