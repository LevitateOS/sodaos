import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Link, MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { FileEditor } from "./file-editor";
import type { Session } from "./api";
import { useSession } from "./session";
const alice: Session = { user: { id: "1", login: "alice", soda_display_name: "Alice" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("sends the loaded SHA once and retains the draft on a native conflict", async () => {
  const sha = "a".repeat(40);
  const fetcher = vi.fn(async (_path: string, options: RequestInit) => new Response(JSON.stringify(options.method === "PUT" ? { error: { code: "provider_conflict", message: "Forgejo reported a conflict; reload before editing." } } : { items: [{ type: "file", sha, text: "old" }] }), { status: options.method === "PUT" ? 409 : 200, headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/edit?ref=feature%2Fone&path=file.txt"]}><Routes><Route path="/repositories/:owner/:repo/edit" element={<FileEditor session={alice} />} /></Routes></MemoryRouter>);
  const user = userEvent.setup(); const editor = await screen.findByDisplayValue("old"); await user.clear(editor); await user.type(editor, "new");
  await user.type(screen.getByLabelText(/Commit message/), "Update file"); await user.click(screen.getByRole("button", { name: "Commit file in Forgejo" }));
  await screen.findByText("Forgejo reported a conflict; reload before editing.");
  expect(screen.getByDisplayValue("new")).toBeTruthy();
  const writes = fetcher.mock.calls.filter(([, options]) => options.method === "PUT"); expect(writes).toHaveLength(1);
  expect(writes[0]?.[0]).toContain("ref=feature%2Fone"); expect(JSON.parse(String(writes[0]?.[1].body))).toEqual({ content: "bmV3", message: "Update file", sha });
});

it("does not let an old file save navigate away from a newly selected file", async () => {
  let finish!: (response: Response) => void;
  const pending = new Promise<Response>(resolve => { finish = resolve; });
  const fetcher = vi.fn(async (path: string, options: RequestInit) => options.method === "PUT" ? pending : Response.json({ items: [{ type: "file", sha: "a".repeat(40), text: path.includes("second.txt") ? "Second file" : "First file" }] }));
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  function Location() { const location = useLocation(); return <output aria-label="Current location">{location.pathname + location.search}</output>; }
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/edit?ref=main&path=first.txt"]}>
    <Link to="/repositories/alice/demo/edit?ref=main&path=second.txt">Other file</Link><Location />
    <Routes><Route path="/repositories/:owner/:repo/edit" element={<FileEditor session={alice} />} /></Routes>
  </MemoryRouter>);
  await screen.findByDisplayValue("First file"); const user = userEvent.setup();
  await user.type(screen.getByLabelText(/Commit message/), "First change");
  await user.click(screen.getByRole("button", { name: "Commit file in Forgejo" }));
  await waitFor(() => expect(fetcher.mock.calls.filter(([, options]) => options.method === "PUT")).toHaveLength(1));
  await user.click(screen.getByRole("link", { name: "Other file" })); await screen.findByDisplayValue("Second file");
  await act(async () => { finish(Response.json({ commit_sha: "b".repeat(40), content_sha: "c".repeat(40) })); });
  expect(screen.getByLabelText("Current location").textContent).toContain("/edit?ref=main&path=second.txt");
  expect(screen.getByDisplayValue("Second file")).toBeTruthy();
});

it("bounds text by UTF-8 bytes and never sends an oversized commit", async () => {
  const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/edit?ref=main&path=large.txt&new=1"]}><Routes><Route path="/repositories/:owner/:repo/edit" element={<FileEditor session={alice} />} /></Routes></MemoryRouter>);
  const user = userEvent.setup(); await user.click(screen.getByLabelText(/Text content/)); await user.paste("é".repeat(16385));
  await user.type(screen.getByLabelText(/Commit message/), "Large change"); await user.click(screen.getByRole("button", { name: "Commit file in Forgejo" }));
  await screen.findByText("File content exceeds 32 KiB. Use native Git."); expect(fetcher).not.toHaveBeenCalled();
});
