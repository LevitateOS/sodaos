import { create } from "zustand";
import { APIError, request, type Session } from "./api";

type Phase = "loading" | "authenticated" | "anonymous" | "signing-out" | "error";
let revision = 0;
let pending: AbortController | undefined;
export const useSession = create<{
  phase: Phase; session: Session | null; error: string;
  load: () => Promise<void>;
  logout: () => Promise<void>;
  invalidate: (expected: Session) => void;
}>((set, get) => ({
  phase: "loading", session: null, error: "",
  async load() {
    const current = ++revision;
    pending?.abort();
    const controller = new AbortController();
    pending = controller;
    set({ phase: "loading", session: null, error: "" });
    try {
      const session = await request<Session>("/api/session", { signal: controller.signal });
      if (!session || typeof session.user?.id !== "string" || typeof session.user.login !== "string" || typeof session.user.soda_display_name !== "string" || typeof session.csrf_token !== "string" || !session.csrf_token || typeof session.soda_operator !== "boolean" || typeof session.forgejo_url !== "string") {
        throw new Error("Invalid session response");
      }
      if (current === revision) set({ phase: "authenticated", session });
    } catch (error) {
      if (current !== revision || controller.signal.aborted) return;
      set(error instanceof APIError && error.status === 401
        ? { phase: "anonymous", session: null }
        : { phase: "error", session: null, error: "Cannot read your session. Check the connection and try again." });
    }
  },
  async logout() {
    const session = get().session;
    if (!session || get().phase !== "authenticated") return;
    const current = ++revision;
    pending?.abort();
    // Unmount all feature views/drafts while sign-out is pending.
    set({ phase: "signing-out", session: null, error: "" });
    try {
      await request<void>("/api/session/logout", { method: "POST", body: {}, csrf: session.csrf_token });
      if (current === revision) set({ phase: "anonymous" });
    } catch {
      if (current === revision) set({ phase: "error", error: "Sign-out could not be confirmed. Check your session before trying again." });
    }
  },
  invalidate(expected) {
    // A late 401 from the previous account cannot clear a newer session.
    if (get().session !== expected) return;
    revision++;
    pending?.abort();
    set({ phase: "anonymous", session: null, error: "" });
  },
}));
