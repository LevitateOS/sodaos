import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vite-plus/test";
import type { Session } from "./api";
import { Profile } from "./profile";
import { useSession } from "./session";

const alice: Session = { user: { id: "1", login: "alice", soda_display_name: "Alice" }, csrf_token: "test-csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
afterEach(() => { cleanup(); vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous" }); });
it("loads real API key data and saves only a Soda preference", async () => {
  const fetcher = vi.fn().mockImplementation(async (_path: string, options: RequestInit) => new Response(JSON.stringify(options.method === "PATCH" ? { display_name: "Alice Soda" } : { items: [] }), { headers: { "Content-Type": "application/json" } }));
  vi.stubGlobal("fetch", fetcher);
  useSession.setState({ session: alice, phase: "authenticated" });
  render(<Profile session={alice} />);
  await screen.findByText("No development keys registered.");
  const user = userEvent.setup();
  await user.clear(screen.getByLabelText("Soda display name"));
  await user.type(screen.getByLabelText("Soda display name"), "Alice Soda");
  await user.click(screen.getByRole("button", { name: "Save display name" }));
  await waitFor(() => expect(useSession.getState().session?.user.soda_display_name).toBe("Alice Soda"));
  const write = fetcher.mock.calls.find(([, options]) => options.method === "PATCH");
  if (!write) throw new Error("Missing preference write");
  expect(write[0]).toBe("/api/me/preferences");
  expect(JSON.parse(String(write[1].body))).toEqual({ display_name: "Alice Soda" });
  expect(screen.queryByRole("button", { name: /delete|rotate/i })).toBeNull();
});
