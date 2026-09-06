import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { PullReview } from "./pull-review";
import type { Pull } from "./pulls";
import type { Session } from "./api";
import { useSession } from "./session";
const session: Session = { user: { id: "2", login: "bob", soda_display_name: "Bob" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
const pull: Pull = { id: "10", number: "12", title: "Change", body: "", state: "open", user: { id: "1", login: "alice" }, head: { ref: "topic", sha: "a".repeat(40), label: "alice:topic" }, base: { ref: "main", sha: "b".repeat(40), label: "main" }, merge_base: "c".repeat(40), merged: false, mergeable: true, draft: false, merge_methods: ["merge"], requested_reviewers: [] };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("keeps the draft on a stale native review and never retries or reloads it as success", async () => {
  const changed = vi.fn(); const writes: RequestInit[] = [];
  vi.stubGlobal("fetch", vi.fn(async (_url: string, init?: RequestInit) => {
    if (init?.method === "POST") { writes.push(init); return new Response(JSON.stringify({ error: { code: "pull_changed", message: "Pull request revisions changed." } }), { status: 409, headers: { "Content-Type": "application/json" } }); }
    return new Response(JSON.stringify({ items: [], next_page: null }), { headers: { "Content-Type": "application/json" } });
  })); useSession.setState({ session, phase: "authenticated" });
  render(<MemoryRouter><PullReview session={session} owner="alice" repo="demo" pull={pull} onChanged={changed} /></MemoryRouter>);
  const user = userEvent.setup(); await user.type(screen.getByLabelText("Review body"), "Please explain this change"); await user.click(screen.getByRole("button", { name: "Submit native review" }));
  await screen.findByText("Pull request revisions changed."); expect((screen.getByLabelText("Review body") as HTMLTextAreaElement).value).toBe("Please explain this change"); expect(changed).not.toHaveBeenCalled(); expect(writes).toHaveLength(1);
  expect(JSON.parse(String(writes[0].body))).toMatchObject({ head: pull.head.sha, base: pull.base.sha, merge_base: pull.merge_base, event: "COMMENT" });
});
