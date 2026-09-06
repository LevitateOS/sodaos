import { afterEach, expect, it, vi } from "vite-plus/test";
import type { Session } from "./api";
import { useSession } from "./session";

const alice: Session = { user: { id: "1", login: "alice", soda_display_name: "Alice" }, csrf_token: "alice-csrf", soda_operator: false, forgejo_url: "https://forgejo.example" };
const response = (session: Session) => new Response(JSON.stringify(session), { headers: { "Content-Type": "application/json" } });
afterEach(() => { vi.unstubAllGlobals(); useSession.setState({ session: null, phase: "anonymous", error: "" }); });

it("ignores late session results even if a transport ignores cancellation", async () => {
  let finish!: (value: Response) => void;
  const first = new Promise<Response>((resolve) => { finish = resolve; });
  const bob = { ...alice, user: { ...alice.user, id: "2", login: "bob" } };
  vi.stubGlobal("fetch", vi.fn().mockReturnValueOnce(first).mockResolvedValueOnce(response(bob)));
  const old = useSession.getState().load();
  await useSession.getState().load();
  finish(response(alice));
  await old;
  expect(useSession.getState().session?.user.login).toBe("bob");
  useSession.getState().invalidate(alice);
  expect(useSession.getState().session?.user.login).toBe("bob");
});

it("clears feature identity while signing out and never retries an ambiguous write", async () => {
  useSession.setState({ session: alice, phase: "authenticated" });
  let fail!: (reason: Error) => void;
  const fetcher = vi.fn().mockReturnValue(new Promise((_resolve, reject) => { fail = reject; }));
  vi.stubGlobal("fetch", fetcher);
  const logout = useSession.getState().logout();
  expect(useSession.getState().session).toBeNull();
  expect(useSession.getState().phase).toBe("signing-out");
  fail(new Error("network unavailable"));
  await logout;
  expect(useSession.getState().phase).toBe("error");
  expect(useSession.getState().session).toBeNull();
  expect(fetcher).toHaveBeenCalledTimes(1);
});

it("turns an expired server session into deliberate sign-in state", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: "unauthenticated", message: "Sign in." } }), { status: 401, headers: { "Content-Type": "application/json" } })));
  await useSession.getState().load();
  expect(useSession.getState().phase).toBe("anonymous");
  expect(useSession.getState().session).toBeNull();
});
