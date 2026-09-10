import type { Service, Registration, LifecycleAction } from "./types";
export function createPayload(data: {
  get(name: string): FormDataEntryValue | null | undefined;
}): Registration {
  const field = (name: string) => {
    const value = data.get(name);
    return typeof value === "string" ? value : "";
  };
  return {
    id: field("id"),
    provider: "forgejo",
    registration_url: "",
    registration_id: field("registration_id"),
    labels: field("forgejo_labels"),
    registration_token: field("registration_token"),
  };
}

export function statusText(service: Pick<Service, "active" | "sub">) {
  if (service.active === "active" && service.sub === "running") {
    return "Listening";
  }
  if (service.active === "failed") {
    return "Failed";
  }
  if (service.active === "activating" || service.active === "deactivating") {
    return service.active === "activating" ? "Starting" : "Stopping";
  }
  return "Stopped";
}

export function statusClass(service: Pick<Service, "active" | "sub">) {
  if (service.active === "active" && service.sub === "running") {
    return "good";
  }
  return service.active === "failed" ? "bad" : "neutral";
}

export function forgejoBrowserURL(origin: string) {
  try {
    const url = new URL(origin);
    if (!["http:", "https:"].includes(url.protocol) || url.username || url.password) return "";
    return url.origin;
  } catch { return ""; }
}

export function successMessage(action: LifecycleAction | "create", id: string) {
  if (action === "create") return `${id} was registered and its local listener started.`;
  if (action === "remove")
    return `${id} and its local account and state were removed. Remove its offline record in the provider.`;
  const pastTense = { start: "started", stop: "stopped", restart: "restarted" };
  return `${id} was ${pastTense[action]}.`;
}

export function errorMessage(error: unknown) {
  return error !== null &&
    typeof error === "object" &&
    "message" in error &&
    typeof error.message === "string" &&
    error.message.trim()
    ? error.message
    : "The operation failed without a diagnostic message.";
}
