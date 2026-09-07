import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Link, MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { ForkRepository, ImportRepository } from "./repository-copy";
import type { Session } from "./api";
import { useSession } from "./session";
const alice: Session = { user: { id: "1", login: "alice", soda_display_name: "Alice" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });

it("clears import credentials but retains non-secret fields on native failure without retry", async () => {
  const fetcher = vi.fn(async (_path: string, _options: RequestInit) => Response.json({ error: { code: "provider_validation", message: "Native import rejected" } }, { status: 422 }));
  vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  render(<MemoryRouter><ImportRepository session={alice} /></MemoryRouter>);
  const user = userEvent.setup();
  await user.type(screen.getByLabelText(/HTTPS Git clone URL/), "https://git.example/source.git");
  await user.type(screen.getByLabelText(/New repository name/), "copy");
  await user.type(screen.getByLabelText("Optional source password"), "synthetic-password");
  await user.type(screen.getByLabelText("Optional source token"), "synthetic-token");
  await user.click(screen.getByRole("button", { name: "Import through Forgejo" }));
  await screen.findByText("Native import rejected");
  expect(screen.getByDisplayValue("https://git.example/source.git")).toBeTruthy(); expect(screen.getByDisplayValue("copy")).toBeTruthy();
  expect((screen.getByLabelText("Optional source password") as HTMLInputElement).value).toBe("");
  expect((screen.getByLabelText("Optional source token") as HTMLInputElement).value).toBe("");
  expect(fetcher).toHaveBeenCalledTimes(1);
  const body = JSON.parse(String(fetcher.mock.calls[0]?.[1].body)); expect(body.password).toBe("synthetic-password"); expect(body.token).toBe("synthetic-token"); expect(body.owner).toBeUndefined();
});

it("ignores a pending fork response after selecting another source repository", async () => {
  let finish!: (value: Response) => void; const pending = new Promise<Response>(resolve => { finish = resolve; });
  const fetcher = vi.fn(async () => pending); vi.stubGlobal("fetch", fetcher); useSession.setState({ session: alice, phase: "authenticated" });
  function Location() { const location = useLocation(); return <output aria-label="Current location">{location.pathname}</output>; }
  render(<MemoryRouter initialEntries={["/repositories/bob/first/fork"]}><Link to="/repositories/bob/second/fork">Other source</Link><Location /><Routes><Route path="/repositories/:owner/:repo/fork" element={<ForkRepository session={alice} />} /></Routes></MemoryRouter>);
  const user = userEvent.setup(); await user.click(screen.getByRole("button", { name: "Fork in Forgejo" })); await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
  await user.click(screen.getByRole("link", { name: "Other source" })); await screen.findByDisplayValue("second");
  await act(async () => { finish(Response.json({ id: "9", name: "first", owner: { id: "1", login: "alice" } })); });
  expect(screen.getByLabelText("Current location").textContent).toBe("/repositories/bob/second/fork"); expect(screen.getByDisplayValue("second")).toBeTruthy();
});

it("clears import inputs when the acting account changes", async () => {
  useSession.setState({ session: alice, phase: "authenticated" });
  const view = render(<MemoryRouter><ImportRepository session={alice} /></MemoryRouter>);
  const user = userEvent.setup(); await user.type(screen.getByLabelText(/HTTPS Git clone URL/), "https://git.example/private.git");
  await user.type(screen.getByLabelText("Optional source password"), "synthetic-password");
  const bob: Session = { ...alice, user: { id: "2", login: "bob", soda_display_name: "Bob" }, csrf_token: "bob-csrf" };
  useSession.setState({ session: bob, phase: "authenticated" });
  view.rerender(<MemoryRouter><ImportRepository session={bob} /></MemoryRouter>);
  await waitFor(() => expect((screen.getByLabelText("Optional source password") as HTMLInputElement).value).toBe(""));
  expect((screen.getByLabelText(/HTTPS Git clone URL/) as HTMLInputElement).value).toBe("");
});
