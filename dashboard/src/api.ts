export interface Session {
  user: { id: string; login: string; soda_display_name: string };
  csrf_token: string;
  soda_operator: boolean;
  forgejo_url: string;
}
export interface DevelopmentKey { id: string; public_key: string; fingerprint: string }

export class APIError extends Error {
  constructor(public readonly status: number, public readonly code: string, message: string) { super(message); }
}

export async function request<T>(path: string, options: {
  method?: "GET" | "POST" | "PATCH" | "PUT" | "DELETE";
  body?: unknown;
  csrf?: string;
  signal?: AbortSignal;
} = {}): Promise<T> {
  // Only literal same-origin API paths; never attach CSRF or cookies to a
  // provider-supplied URL or follow an unexpected login redirect.
  if (!path.startsWith("/api/") || path.includes("\\") || path.includes("#")) throw new Error("Invalid API path");
  const headers = new Headers({ Accept: "application/json" });
  if (options.body !== undefined) headers.set("Content-Type", "application/json");
  if (options.csrf) headers.set("X-CSRF-Token", options.csrf);
  const response = await fetch(path, {
    method: options.method ?? "GET", credentials: "same-origin", redirect: "error", cache: "no-store",
    headers, signal: options.signal,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  if (response.status === 204) return undefined as T;
  let data;
  try {
    if (!response.headers.get("Content-Type")?.startsWith("application/json")) throw new Error();
    data = await response.json();
  } catch {
    throw new APIError(response.status, "invalid_response", "The server returned an unexpected response.");
  }
  if (!response.ok) throw new APIError(response.status, typeof data?.error?.code === "string" ? data.error.code : "request_failed", typeof data?.error?.message === "string" ? data.error.message : "Request failed.");
  return data as T;
}
