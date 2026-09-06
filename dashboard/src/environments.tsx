import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Alert, Button, Spinner } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";

interface Environment { id: string; name: string; repository: string; repository_id: string; owner_id: string; provisioned: boolean }
interface Observed { id: string; ip: string; running: boolean }
interface Detail { environment: Environment; observed: Observed | null; native_unavailable: boolean; login: string; environment_administrator: boolean }
interface Connection { login: string; connection: { environment: Observed; host_key: string; fingerprint: string }; routing_verified: boolean }
export function EnvironmentList({ session }: { session: Session }) {
  const [items, setItems] = useState<Environment[] | null>(null); const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    request<{ items: Environment[] }>("/api/environments", { signal: controller.signal }).then(result => { if (!controller.signal.aborted) setItems(result.items); }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Environments unavailable.");
    });
    return () => controller.abort();
  }, [session]);
  return <section><h1>Persistent environments</h1><p>Environment discovery follows Soda’s trusted-team policy. This does not grant permission to read a private Forgejo repository.</p><Link to="/repositories">Select an owned repository to create an environment</Link>
    {error && <Alert isInline variant="danger" title={error} />}{!items && !error && <Spinner aria-label="Loading environments" />}
    {items?.length === 0 && <p>No environments have been reserved.</p>}
    <ul>{items?.map(item => <li key={item.id}><Link to={`/environments/${item.id}`}>{item.name}</Link> — {item.repository} — {item.provisioned ? "Provisioning recorded; inspect live state" : "Incomplete reservation; retained for inspection"}</li>)}</ul>
  </section>;
}
export function EnvironmentDetail({ session }: { session: Session }) {
  const { id = "" } = useParams(); const api = `/api/environments/${encodeURIComponent(id)}`;
  const [detail, setDetail] = useState<Detail | null>(null); const [connection, setConnection] = useState<Connection | null>(null);
  const [members, setMembers] = useState<{ user_id: string; login: string }[]>([]);
  const [error, setError] = useState(""); const [connectionError, setConnectionError] = useState("");
  const [attempt, setAttempt] = useState(0); const [busy, setBusy] = useState(false);
  useEffect(() => {
    const controller = new AbortController(); setDetail(null); setConnection(null); setMembers([]); setError(""); setConnectionError("");
    async function load() {
      try {
        const result = await request<Detail>(api, { signal: controller.signal });
        if (controller.signal.aborted) return; setDetail(result);
        const memberResult = await request<{ items: { user_id: string; login: string }[] }>(`${api}/members`, { signal: controller.signal });
        if (controller.signal.aborted) return; setMembers(memberResult.items);
        if (result.login) {
          try {
            const value = await request<Connection>(`${api}/connection`, { signal: controller.signal });
            if (!controller.signal.aborted) setConnection(value);
          } catch (failure) { if (!controller.signal.aborted) setConnectionError(failure instanceof Error ? failure.message : "Connection details unavailable."); }
        }
      } catch (failure) {
        if (controller.signal.aborted) return;
        if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
        else setError(failure instanceof Error ? failure.message : "Environment unavailable.");
      }
    }
    void load(); return () => controller.abort();
  }, [api, session, attempt]);
  async function join() {
    if (busy) return; setBusy(true); setError("");
    try {
      await request(`${api}/join`, { method: "POST", body: {}, csrf: session.csrf_token });
      if (useSession.getState().session === session) setAttempt(value => value + 1);
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Join was not confirmed. Inspect membership before retrying.");
    } finally { setBusy(false); }
  }
  const endpoint = connection?.connection.environment;
  const destination = connection && endpoint?.running && endpoint.ip ? `${connection.login}@${endpoint.ip}` : null;
  return <section aria-busy={busy}><h1>{detail?.environment.name ?? "Environment"}</h1><code>{id}</code>
    {error && <Alert isInline variant="danger" title={error} />}
    <Button variant="secondary" isDisabled={busy} onClick={() => setAttempt(value => value + 1)}>Refresh observed state</Button>
    {!detail && !error && <Spinner aria-label="Inspecting environment" />}
    {detail && <><p>Associated repository: {detail.environment.repository}</p>
      {!detail.environment.provisioned && <Alert isInline variant="warning" title="Provisioning incomplete. The reservation and any native state are retained. Ask the operator to inspect; do not recreate the environment." />}
      <p>Native observation: {detail.native_unavailable ? "Unavailable — no running or reachability claim" : detail.observed?.running ? `Running at ${detail.observed.ip || "an unavailable address"}` : "Stopped"}</p>
      <h2>Membership</h2><p>{detail.environment_administrator ? "You may view the environment membership. This does not grant host administration." : "Only your own membership is shown."}</p>
      <ul>{members.map(member => <li key={member.user_id}>{member.login}</li>)}</ul>
      {detail.login ? <p>Joined as <strong>{detail.login}</strong>.</p> : <><p>The creator must explicitly join too. Register a <Link to="/profile">public development-access key</Link> first.</p><Button isDisabled={busy || !detail.environment.provisioned} isLoading={busy} onClick={() => void join()}>Add me to this project</Button></>}
      <p>Joining installs your registered public keys through the native account helper. Later key edits do not synchronize existing accounts. Joining does not grant Git permissions.</p>
      <h2>Connect</h2>{connectionError && <Alert isInline variant="warning" title={connectionError} />}
      {connection && !endpoint?.running && <p>The environment is stopped. Ask the operator to start the existing container; do not replace it.</p>}
      {destination && connection && <><Alert isInline variant="warning" title="Routing has not been verified. Your client needs an operator-approved route to this project subnet. A reported bridge address is not proof of access." />
        <p>Verify the Ed25519 host key delivered over this trusted HTTPS session before accepting the SSH prompt:</p><pre>{connection.connection.fingerprint}{"\n"}{connection.connection.host_key}</pre>
        <pre>{`ssh ${destination}\nscp ./local-file ${destination}:~/\nsftp ${destination}`}</pre>
        <p>Use the same user and IP with your editor’s Remote SSH integration. Keep your private key on your own client. Never disable host-key verification.</p>
      </>}
    </>}
  </section>;
}
