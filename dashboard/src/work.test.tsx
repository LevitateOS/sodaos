import { act, cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { MyWork, WorkResults } from "./work";
import type { Session } from "./api";
import { useSession } from "./session";
const session: Session = { user: { id: "2", login: "bob", soda_display_name: "Bob" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
const result = (title: string) => ({ items: [{ id: "9007199254740993", number: "12", title, state: "open", kind: "issues", repository: { name: "demo", owner: "alice", full_name: "alice/demo" } }], next_page: null });
const response = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status, headers: { "Content-Type": "application/json" } });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("ignores superseded native search results even when transport resolves after abort", async () => {
  let finish!: (value: Response) => void; let signal: AbortSignal | undefined;
  vi.stubGlobal("fetch", vi.fn((url: string, init?: RequestInit) => { if (url.includes("q=old")) { signal = init?.signal as AbortSignal; return new Promise<Response>(resolve => { finish = resolve; }); } return Promise.resolve(response(result("New result"))); })); useSession.setState({ session, phase: "authenticated" });
  const view = render(<MemoryRouter><WorkResults session={session} query="q=old" /></MemoryRouter>);
  view.rerender(<MemoryRouter><WorkResults session={session} query="q=new" /></MemoryRouter>); await screen.findByText(/New result/); expect(signal?.aborted).toBe(true);
  await act(async () => { finish(response(result("Old private result"))); }); expect(screen.queryByText(/Old private result/)).toBeNull();
});
it("keeps Soda environments and requested reviews visible when assigned work fails", async () => {
  vi.stubGlobal("fetch", vi.fn(async (url: string) => {
    if (url === "/api/environments") return response({ items: [{ id: "project-one", name: "Shared project", repository: "alice/demo" }] });
    if (url.includes("assigned=true")) return response({ error: { code: "provider_unavailable", message: "Assigned issues unavailable" } }, 503);
    if (url.includes("review_requested=true")) return response(result("Review this"));
    return response({ items: [], next_page: null });
  })); useSession.setState({ session, phase: "authenticated" });
  render(<MemoryRouter><MyWork session={session} /></MemoryRouter>); await screen.findByText("Assigned issues unavailable"); await screen.findByRole("link", { name: "Shared project" }); await screen.findByText(/Review this/);
});
