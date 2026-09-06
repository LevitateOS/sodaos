import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { RepositorySettings } from "./repository-settings";
import type { Session } from "./api";
import { useSession } from "./session";
const session: Session = { user: { id: "2", login: "bob", soda_display_name: "Bob" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("submits only explicitly changed settings, not a copied repository object", async () => {
  const settings = { id: "9007199254740993", name: "demo", description: "Old", website: "", private: true, default_branch: "main", has_issues: true, has_pull_requests: true, has_wiki: false, has_actions: false, has_releases: true, has_packages: true, allow_merge_commits: true, allow_squash_merge: false, allow_rebase: false, allow_rebase_explicit: false, allow_fast_forward_only_merge: false };
  const writes: RequestInit[] = []; vi.stubGlobal("fetch", vi.fn(async (_url: string, init?: RequestInit) => { if (init?.method === "PATCH") writes.push(init); return new Response(JSON.stringify({ ...settings, description: init?.method === "PATCH" ? "Updated" : "Old" }), { headers: { "Content-Type": "application/json" } }); })); useSession.setState({ session, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/settings"]}><Routes><Route path="/repositories/:owner/:repo/settings" element={<RepositorySettings session={session} />} /></Routes></MemoryRouter>);
  const field = await screen.findByLabelText("Repository description"); const user = userEvent.setup(); await user.clear(field); await user.type(field, "Updated"); await user.click(screen.getByRole("button", { name: "Save changed native settings" }));
  expect(writes).toHaveLength(1); expect(JSON.parse(String(writes[0].body))).toEqual({ description: "Updated" }); expect((screen.getByLabelText("Private repository") as HTMLInputElement).checked).toBe(true);
});
