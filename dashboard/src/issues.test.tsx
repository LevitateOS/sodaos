import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { NewIssue } from "./issues";
import type { Session } from "./api";
import { useSession } from "./session";
const bob: Session = { user: { id: "2", login: "bob", soda_display_name: "Bob" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("creates a native issue without provisioning an environment", async () => {
  const fetcher = vi.fn(async () => new Response(JSON.stringify({ id: "9007199254740993", number: "12" }), { status: 201, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: bob, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/issues/new"]}><Routes><Route path="/repositories/:owner/:repo/issues/new" element={<NewIssue session={bob} />} /><Route path="/repositories/alice/demo/issues/12" element={<h1>Native issue destination</h1>} /></Routes></MemoryRouter>);
  const user = userEvent.setup(); await user.type(screen.getByLabelText("Title", { exact: false }), "Bug"); await user.type(screen.getByLabelText("Markdown body"), "Details"); await user.type(screen.getByLabelText(/Assignee usernames/), "alice");
  await user.click(screen.getByRole("button", { name: "Create issue in Forgejo" })); await screen.findByRole("heading", { name: "Native issue destination" });
  expect(fetcher).toHaveBeenCalledTimes(1); expect(fetcher).toHaveBeenCalledWith("/api/forgejo/repos/alice/demo/issues", expect.objectContaining({ method: "POST", body: JSON.stringify({ title: "Bug", body: "Details", assignees: ["alice"] }) }));
});
