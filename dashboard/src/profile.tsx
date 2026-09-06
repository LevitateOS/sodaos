import { useEffect, useRef, useState } from "react";
import { Alert, Button, Form, FormGroup, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type DevelopmentKey, type Session } from "./api";
import { useSession } from "./session";

export function Profile({ session }: { session: Session }) {
  const [name, setName] = useState(session.user.soda_display_name);
  const [publicKey, setPublicKey] = useState("");
  const [keys, setKeys] = useState<DevelopmentKey[] | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const keyRead = useRef<AbortController | null>(null);
  useEffect(() => {
    const controller = new AbortController();
    keyRead.current = controller;
    setKeys(null); setError("");
    request<{ items: DevelopmentKey[] }>("/api/me/development-keys", { signal: controller.signal }).then((result) => {
      if (!controller.signal.aborted) setKeys(result.items);
    }).catch((failure) => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError("Cannot load development keys. Existing keys have not been removed.");
    });
    return () => controller.abort();
  }, [session, attempt]);

  async function save(kind: "name" | "key") {
    if (busy) return;
    setBusy(true); setError(""); setNotice("");
    try {
      if (kind === "name") {
        const result = await request<{ display_name: string }>("/api/me/preferences", { method: "PATCH", body: { display_name: name }, csrf: session.csrf_token });
        if (useSession.getState().session !== session) return;
        setName(result.display_name);
        setNotice("Soda display name saved. Your Forgejo account was not changed.");
        setBusy(false);
        useSession.setState({ session: { ...session, user: { ...session.user, soda_display_name: result.display_name } } });
      } else {
        // An older GET must not overwrite the list returned by this write.
        keyRead.current?.abort();
        const result = await request<{ items: DevelopmentKey[] }>("/api/me/development-keys", { method: "POST", body: { public_key: publicKey }, csrf: session.csrf_token });
        if (useSession.getState().session !== session) return;
        setKeys(result.items); setPublicKey("");
        setNotice("Public key registered for future joins. Existing environment accounts were not changed.");
      }
    } catch (failure) {
      if (useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof APIError ? failure.message : "The operation could not be confirmed. Refresh before retrying; it may already have completed.");
    } finally {
      if (useSession.getState().session === session) setBusy(false);
    }
  }

  return <section aria-labelledby="profile-heading" aria-busy={busy}>
    <h1 id="profile-heading">Profile and development keys</h1>
    <p>Forgejo login: <strong>{session.user.login}</strong>. This display name is a Soda preference, not your upstream account profile.</p>
    {error && <div className="feedback"><Alert isInline variant="danger" title={error} /><Button variant="link" isDisabled={busy} onClick={() => setAttempt(attempt + 1)}>Refresh keys</Button></div>}
    {notice && <div className="feedback"><Alert isInline variant="success" title={notice} /></div>}
    <Form onSubmit={(event) => { event.preventDefault(); void save("name"); }}>
      <FormGroup label="Soda display name" fieldId="soda-name"><TextInput id="soda-name" value={name} onChange={(_event, value) => setName(value)} isDisabled={busy} /></FormGroup>
      <Button type="submit" isDisabled={busy}>Save display name</Button>
    </Form>
    <h2>Development-access public keys</h2>
    <p>Never submit a private key. These keys are distinct from Forgejo Git keys and are installed when you explicitly join an environment.</p>
    {keys === null && !error ? <Spinner aria-label="Loading development keys" /> : keys?.length === 0 ? <p>No development keys registered.</p> : <ul className="key-list">{keys?.map((key) => <li key={key.id}><code>{key.fingerprint}</code></li>)}</ul>}
    <Form onSubmit={(event) => { event.preventDefault(); void save("key"); }}>
      <FormGroup label="Public SSH key" fieldId="public-key" isRequired><TextArea id="public-key" value={publicKey} onChange={(_event, value) => setPublicKey(value)} isDisabled={busy} isRequired spellCheck={false} /></FormGroup>
      <Button type="submit" isDisabled={busy || !publicKey.trim()}>Register public key</Button>
    </Form>
  </section>;
}
