import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { ActionRunDetail } from "./actions";
import { ActionConfiguration } from "./action-configuration";
import type { Session } from "./api";
import { useSession } from "./session";
const session: Session = { user: { id: "2", login: "bob", soda_display_name: "Bob" }, csrf_token: "csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
const response = (data: unknown, status = 200) => new Response(JSON.stringify(data), { status, headers: { "Content-Type": "application/json" } });
afterEach(() => { cleanup(); vi.useRealTimers(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("polls only the viewed run and cancels its timer and transport on unmount", async () => {
 vi.useFakeTimers(); let signal: AbortSignal | undefined; const fetch = vi.fn(async (_url: string, init?: RequestInit) => { signal = init?.signal as AbortSignal; return response({ id: "7", number: "4", status: "running", title: "Build", sha: "abc" }); }); vi.stubGlobal("fetch", fetch); useSession.setState({ session, phase: "authenticated" });
 const view = render(<MemoryRouter initialEntries={["/alice/demo/7"]}><Routes><Route path="/:owner/:repo/:run" element={<ActionRunDetail session={session} />} /></Routes></MemoryRouter>);
 await act(async () => {}); expect(fetch).toHaveBeenCalledTimes(1); await act(async () => { await vi.advanceTimersByTimeAsync(5000); }); expect(fetch).toHaveBeenCalledTimes(2); view.unmount(); expect(signal?.aborted).toBe(true); await act(async () => { await vi.advanceTimersByTimeAsync(15000); }); expect(fetch).toHaveBeenCalledTimes(2);
});
it("stops polling after native denial instead of retrying with another authority", async () => {
 vi.useFakeTimers(); const fetch = vi.fn(async () => response({ error: { code: "provider_forbidden", message: "Native permission denied" } }, 403)); vi.stubGlobal("fetch", fetch); useSession.setState({ session, phase: "authenticated" });
 render(<MemoryRouter initialEntries={["/alice/demo/7"]}><Routes><Route path="/:owner/:repo/:run" element={<ActionRunDetail session={session} />} /></Routes></MemoryRouter>); await act(async () => {}); expect(screen.getByText("Native permission denied")).toBeTruthy(); await act(async () => { await vi.advanceTimersByTimeAsync(15000); }); expect(fetch).toHaveBeenCalledTimes(1);
});
it("clears write-only secret input after submission even when native save fails", async () => {
 const fetch = vi.fn(async (_url: string, init?: RequestInit) => init?.method === "PUT" ? response({ error: { code: "provider_forbidden", message: "Native permission denied" } }, 403) : response({ items: [], next_page: null })); vi.stubGlobal("fetch", fetch); useSession.setState({ session, phase: "authenticated" });
 render(<MemoryRouter initialEntries={["/alice/demo"]}><Routes><Route path="/:owner/:repo" element={<ActionConfiguration session={session} />} /></Routes></MemoryRouter>); await screen.findByText("No native entries on this page."); fireEvent.change(screen.getByLabelText("Name"), { target: { value: "FIXTURE" } }); fireEvent.change(screen.getByLabelText("Secret value (write-only)"), { target: { value: "fixture-not-real-secret" } }); fireEvent.click(screen.getByRole("button", { name: "Save native secret" })); await screen.findByText(/Native permission denied/); expect((screen.getByLabelText("Secret value (write-only)") as HTMLInputElement).value).toBe(""); expect(fetch.mock.calls.filter(([, init]) => init?.method === "PUT")).toHaveLength(1);
});
