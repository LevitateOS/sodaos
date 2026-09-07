import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { Compare, History, CommitDetail, Refs } from "./history";
import { useSession } from "./session";
import type { Session } from "./api";

const alice: Session = { user: { id: "1", login: "alice", soda_display_name: "Alice" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
const sha = "a".repeat(40), head = "b".repeat(40);
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });

it("labels per-commit files honestly without a Forgejo frontend escape", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => Response.json({ total_commits: "1", base_sha: sha, head_sha: head, commits: [{ sha: head, commit: { message: "Change" } }], files: [{ filename: "same.txt", status: "modified" }, { filename: "same.txt", status: "modified" }] })));
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/compare?base=main&head=feature%2Fone"]}><Routes><Route path="/repositories/:owner/:repo/compare" element={<Compare session={alice} />} /></Routes></MemoryRouter>);
  await screen.findByRole("heading", { name: "Files touched by the reported commits" });
  expect(screen.getAllByText("same.txt — modified")).toHaveLength(2);
  expect(screen.queryByRole("heading", { name: "Changed files" })).toBeNull();
  expect(screen.getByText(/The combined diff is not yet available in Soda/)).toBeTruthy();
  expect(screen.getByText(sha)).toBeTruthy(); expect(screen.getByText(head)).toBeTruthy();
  for (const link of screen.getAllByRole("link")) expect(link.getAttribute("href")).toMatch(/^\/repositories\//);
});

it("retains encoded ref/path pagination and pinned Soda commit navigation", async () => {
  const fetcher = vi.fn(async (_path: string) => Response.json({ items: [{ sha, created: "", commit: { message: "Change", author: { name: "Alice", date: "today" } } }], next_page: 2 }));
  vi.stubGlobal("fetch", fetcher);
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/history?ref=feature%2Fone&path=docs%2Fhello+world.txt"]}><Routes><Route path="/repositories/:owner/:repo/history" element={<History session={alice} />} /></Routes></MemoryRouter>);
  const link = await screen.findByRole("link", { name: "Change" });
  expect(link.getAttribute("href")).toBe(`/repositories/alice/demo/commits/${sha}`);
  expect(screen.getByText(/Line attribution is not yet available in Soda/)).toBeTruthy();
  for (const anchor of screen.getAllByRole("link")) expect(anchor.getAttribute("href")).toMatch(/^\/repositories\//);
  await userEvent.setup().click(screen.getByRole("button", { name: "Next page" }));
  await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));
  const query = new URL(String(fetcher.mock.calls[1]?.[0]), "https://soda.example").searchParams;
  expect(query.get("ref")).toBe("feature/one"); expect(query.get("path")).toBe("docs/hello world.txt"); expect(query.get("page")).toBe("2");
});

it("invalidates the session when only the commit diff reports expiry", async () => {
  useSession.setState({ session: alice, phase: "authenticated" });
  vi.stubGlobal("fetch", vi.fn(async (path: string) => path.endsWith("/diff") ? Response.json({ error: { code: "expired", message: "Sign in" } }, { status: 401 }) : Response.json({ sha, commit: { message: "Change", author: { name: "Alice", date: "today" } }, files: [] })));
  render(<MemoryRouter initialEntries={[`/repositories/alice/demo/commits/${sha}`]}><Routes><Route path="/repositories/:owner/:repo/commits/:sha" element={<CommitDetail session={alice} />} /></Routes></MemoryRouter>);
  await waitFor(() => expect(useSession.getState().session).toBeNull());
});

it("keeps a ref draft after native denial without repeating the write", async () => {
  useSession.setState({ session: alice, phase: "authenticated" });
  const fetcher = vi.fn(async (_path: string, options: RequestInit) => options.method === "POST" ? Response.json({ error: { code: "provider_forbidden", message: "Native protection denied" } }, { status: 403 }) : Response.json({ items: [], next_page: null }));
  vi.stubGlobal("fetch", fetcher);
  render(<MemoryRouter initialEntries={["/repositories/alice/demo/refs"]}><Routes><Route path="/repositories/:owner/:repo/refs" element={<Refs session={alice} />} /></Routes></MemoryRouter>);
  const user = userEvent.setup(); await user.type(screen.getByLabelText(/New name/), "feature/one"); await user.type(screen.getByLabelText(/Source branch/), "main");
  await user.click(screen.getByRole("button", { name: "Create branch" })); await screen.findByText("Native protection denied");
  expect(screen.getByDisplayValue("feature/one")).toBeTruthy(); expect(fetcher.mock.calls.filter(([, options]) => options.method === "POST")).toHaveLength(1);
});
