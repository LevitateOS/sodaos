import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Alert, Button, Form, FormGroup, Spinner, TextArea, TextInput } from "@patternfly/react-core";
import { APIError, request, type Session } from "./api";
import { useSession } from "./session";
import { repositoryPath } from "./repositories";

function encode(bytes: Uint8Array) { return btoa(Array.from(bytes, byte => String.fromCharCode(byte)).join("")); }
export function FileEditor({ session }: { session: Session }) {
  const { owner = "", repo = "" } = useParams(); const [query] = useSearchParams(); const navigate = useNavigate();
  const ref = query.get("ref") ?? ""; const originalPath = query.get("path") ?? ""; const creating = query.get("new") === "1";
  const [path, setPath] = useState(originalPath); const [text, setText] = useState(""); const [binary, setBinary] = useState<string | null>(null);
  const [requiresUpload, setRequiresUpload] = useState(false);
  const [sha, setSHA] = useState(""); const [message, setMessage] = useState(""); const [loaded, setLoaded] = useState(creating);
  const [error, setError] = useState(""); const [busy, setBusy] = useState(false); const [reading, setReading] = useState(false);
  const active = useRef(true); const uploadRevision = useRef(0); const targetRevision = useRef(0);
  const route = repositoryPath(owner, repo); const api = `/api/forgejo/repos/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`;
  useEffect(() => { active.current = true; return () => { active.current = false; uploadRevision.current++; }; }, []);
  useEffect(() => {
    const controller = new AbortController(); targetRevision.current++; uploadRevision.current++; setBusy(false); setMessage(""); setReading(false); setPath(originalPath); setText(""); setBinary(null); setRequiresUpload(false); setSHA(""); setLoaded(creating); setError("");
    if (!creating) request<{ items: { sha: string; type: string; text: string | null }[] }>(`${api}/contents?${new URLSearchParams({ ref, path: originalPath })}`, { signal: controller.signal }).then(result => {
      if (controller.signal.aborted) return;
      const file = result.items[0]; if (result.items.length !== 1 || !file || file.type !== "file") { setError("Select one existing file."); return; }
      setSHA(file.sha); setText(file.text ?? ""); setBinary(file.text === null ? "" : null); setRequiresUpload(file.text === null); setLoaded(true);
    }).catch(failure => {
      if (controller.signal.aborted) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "File version could not be loaded.");
    }); return () => { controller.abort(); targetRevision.current++; uploadRevision.current++; };
  }, [api, ref, originalPath, creating, session]);
  async function upload(file?: File) {
    const revision = ++uploadRevision.current; setReading(false);
    if (!file) return;
    if (file.size > 32768) { setError("Uploads are limited to 32 KiB. Use native Git for larger files."); return; }
    setReading(true); setError("");
    try {
      const bytes = new Uint8Array(await file.arrayBuffer());
      if (active.current && revision === uploadRevision.current) { setBinary(encode(bytes)); setRequiresUpload(false); if (creating && !path) setPath(file.name); }
    } catch { if (active.current && revision === uploadRevision.current) setError("Could not read the selected local file."); }
    finally { if (active.current && revision === uploadRevision.current) setReading(false); }
  }
  async function save() {
    if (busy || reading || requiresUpload || !loaded || !ref || !path || !message.trim()) return;
    const bytes = new TextEncoder().encode(text);
    if (binary === null && bytes.length > 32768) { setError("File content exceeds 32 KiB. Use native Git."); return; }
    const revision = targetRevision.current; setBusy(true); setError("");
    try {
      await request(`${api}/files?${new URLSearchParams({ ref, path })}`, { method: creating ? "POST" : "PUT", csrf: session.csrf_token, body: { content: binary ?? encode(bytes), message, sha: creating ? "" : sha } });
      if (active.current && revision === targetRevision.current && useSession.getState().session === session) navigate(`${route}?${new URLSearchParams({ ref, path })}`);
    } catch (failure) {
      if (!active.current || revision !== targetRevision.current || useSession.getState().session !== session) return;
      if (failure instanceof APIError && failure.status === 401) useSession.getState().invalidate(session);
      else setError(failure instanceof Error ? failure.message : "Commit not confirmed. Inspect native history before retrying.");
    } finally { if (active.current && revision === targetRevision.current) setBusy(false); }
  }
  return <section aria-busy={busy || reading}><h1>{creating ? "Create/upload file" : "Edit/replace file"}</h1><Link to={`${route}?${new URLSearchParams({ ref, path: originalPath })}`}>Back to files</Link><p>Target branch: <strong>{ref || "Missing — return to files and select an explicit branch"}</strong></p>
    {error && <Alert isInline variant="danger" title={error} />}{!loaded && !error && <Spinner aria-label="Loading file precondition" />}
    <Form onSubmit={event => { event.preventDefault(); void save(); }}>
      <FormGroup label="Repository-relative file path" fieldId="edit-file-path" isRequired><TextInput id="edit-file-path" value={path} isRequired isDisabled={busy || !creating} onChange={(_, value) => setPath(value)} /></FormGroup>
      <FormGroup label="Text content (up to 32 KiB UTF-8)" fieldId="edit-file-text"><TextArea id="edit-file-text" value={text} isDisabled={busy || !loaded || binary !== null} rows={20} spellCheck={false} onChange={(_, value) => setText(value)} /></FormGroup>
      {binary !== null && <p>Binary/upload mode. {requiresUpload ? "Choose replacement content before committing an unavailable binary file." : "Selected upload will be committed as bytes."} <Button variant="link" isDisabled={busy} onClick={() => { uploadRevision.current++; setBinary(null); setRequiresUpload(false); setReading(false); }}>Use text instead</Button></p>}
      <FormGroup label="Upload replacement bytes (up to 32 KiB)" fieldId="edit-file-upload"><input id="edit-file-upload" type="file" disabled={busy || !loaded} onChange={event => void upload(event.currentTarget.files?.[0])} /></FormGroup>
      <FormGroup label="Commit message" fieldId="edit-file-message" isRequired><TextArea id="edit-file-message" value={message} isRequired isDisabled={busy} maxLength={8192} onChange={(_, value) => setMessage(value)} /></FormGroup>
      <Button type="submit" isLoading={busy} isDisabled={busy || reading || !loaded || !ref || !path || !message.trim() || requiresUpload}>Commit file in Forgejo</Button>
    </Form>{!creating && <p>Loaded file SHA: <code>{sha}</code>. Forgejo rejects stale content or protected-branch writes. A failed commit preserves this draft; it never silently overwrites a newer file.</p>}
  </section>;
}
