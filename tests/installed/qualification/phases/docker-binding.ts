import path from 'node:path';
import {readFile} from 'node:fs/promises';
import {check, sha, SHA, COMMIT, digestRef} from './support';

export type FileIdentity = {SHA256: string; Bytes: number};
export const DOWNLOADS = [
  'docker.json',
  'sodaos-docker-compose.yaml',
  'install-docker.sh',
  'candidate.json',
  'payload.json',
] as const;
export type DockerBinding = {
  executionReleased: boolean;
  host: DockerHostPaths;
  target: string;
  tag: string;
  source: string;
  forgejo: string;
  candidateSHA: string;
  payloadSHA: string;
  metadataSHA: string;
  composeSHA: string;
  installerSHA: string;
  installerBytes: number;
  manifest: string;
  config: string;
  image: string;
  payload: any;
  candidate: any;
  metadata: any;
  downloads: Record<string, FileIdentity>;
};
export type Witness = {project: string; login: string; home: string; relative: string; sha256: string; session: string};
export type DockerIO = {
  save: (relative: string, value: unknown) => Promise<unknown>;
  saveBytes: (relative: string, bytes: Uint8Array) => Promise<unknown>;
  sshRead: (fixedProgram: string, timeoutMs?: number) => string | Promise<string>;
};

export function validateWitness(w: Witness) {
  check(
    /^p[a-f0-9]{24}$/.test(w?.project) && /^[a-z][a-z0-9_-]{0,30}$/.test(w.login) && w.login !== 'root',
    'exact_joined_member'
  );
  check(
    /^\/home\/[a-z][a-z0-9_-]{0,30}$/.test(w.home) &&
      /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(w.relative) &&
      !['.', '..'].includes(w.relative) &&
      SHA.test(w.sha256) &&
      /^[a-f0-9]{32}$/.test(w.session),
    'exact_member_witness'
  );
  return w;
}

// Root's public asset identities bind the five actual downloaded files. No
// historical native receipt, development input, or Boolean pass is accepted.
export async function loadDockerBinding(
  directory: string,
  cfg: any,
  assets: Record<string, FileIdentity>
): Promise<DockerBinding> {
  check(
    cfg?.format === 1 && cfg.purpose === 'production' && cfg.distribution === 'docker' && cfg.development === undefined,
    'fresh_production_docker_only'
  );
  check(
    typeof cfg.execution_released === 'boolean' && /^[A-Za-z0-9._-]{1,128}$/.test(cfg.target),
    'exact_docker_target'
  );
  check(
    cfg.guest?.ssh_user === 'root' &&
      cfg.guest.ssh_port === 22 &&
      /^[A-Za-z0-9.-]+$/.test(cfg.guest.ssh_host) &&
      cfg.guest.origin === 'https://' + cfg.guest.ssh_host,
    'exact_docker_host'
  );
  check(
    COMMIT.test(cfg.expected?.source_revision) &&
      SHA.test(cfg.expected.candidate_sha256) &&
      SHA.test(cfg.expected.payload_sha256),
    'exact_docker_candidate'
  );
  validateDockerHostPaths(cfg.host);
  const files: Record<string, Buffer> = {};
  for (const name of DOWNLOADS) {
    const id = assets[name];
    check(
      id &&
        Object.keys(id).sort().join(',') === 'Bytes,SHA256' &&
        SHA.test(id.SHA256) &&
        Number.isSafeInteger(id.Bytes) &&
        id.Bytes > 0 &&
        id.Bytes <= 1 << 20,
      'canonical_download_identity'
    );
    const bytes = await readFile(directory + '/' + name);
    check(bytes.length === id.Bytes && sha(bytes) === id.SHA256, 'exact_download_' + name);
    files[name] = bytes;
  }
  const m = JSON.parse(files['docker.json']!.toString()),
    c = JSON.parse(files['candidate.json']!.toString()),
    p = JSON.parse(files['payload.json']!.toString());
  check(
    m.Format === 1 &&
      m.Purpose === 'production' &&
      ['docker', 'all'].includes(m.Distribution) &&
      m.Architecture === 'x86_64' &&
      m.Platform === 'linux/amd64',
    'production_docker_metadata'
  );
  check(
    m.SodaRevision === cfg.expected.source_revision &&
      m.SodaRevision === p.Revision &&
      c.Host?.Revision === p.Revision &&
      COMMIT.test(m.ForgejoRevision),
    'docker_source_binding'
  );
  check(
    assets['candidate.json']!.SHA256 === cfg.expected.candidate_sha256 &&
      m.CandidateSHA256 === cfg.expected.candidate_sha256 &&
      assets['payload.json']!.SHA256 === cfg.expected.payload_sha256 &&
      m.PayloadSHA256 === cfg.expected.payload_sha256 &&
      c.PayloadSHA256 === cfg.expected.payload_sha256,
    'docker_candidate_binding'
  );
  check(
    m.ReleaseTag === p.ReleaseTag && /^v[0-9][A-Za-z0-9._-]{0,127}$/.test(m.ReleaseTag),
    'exact_public_release_tag'
  );
  digestRef(m.Image?.Manifest);
  digestRef(m.Image?.Config);
  check(
    m.Image.Revision === p.Revision &&
      m.Image.Architecture === 'amd64' &&
      m.Compose.SHA256 === assets['sodaos-docker-compose.yaml']!.SHA256 &&
      m.Compose.Bytes === assets['sodaos-docker-compose.yaml']!.Bytes,
    'docker_image_compose_binding'
  );
  check(
    p.Architecture === 'x86_64' &&
      p.Images &&
      Object.keys(p.Images).sort().join(',') === 'canlang-runtime,dashboard,forgejo,postgres,project-os,proxy',
    'six_shipping_images'
  );
  for (const image of Object.values(p.Images) as any[]) digestRef(image.Config);
  return {
    executionReleased: cfg.execution_released,
    host: cfg.host,
    target: cfg.target,
    tag: m.ReleaseTag,
    source: p.Revision,
    forgejo: m.ForgejoRevision,
    candidateSHA: m.CandidateSHA256,
    payloadSHA: m.PayloadSHA256,
    metadataSHA: assets['docker.json']!.SHA256,
    composeSHA: m.Compose.SHA256,
    installerSHA: assets['install-docker.sh']!.SHA256,
    installerBytes: assets['install-docker.sh']!.Bytes,
    manifest: m.Image.Manifest,
    config: m.Image.Config,
    image: 'ghcr.io/levitateos/sodaos@' + m.Image.Manifest,
    payload: p,
    candidate: c,
    metadata: m,
    downloads: assets,
  };
}

export function nativeRequest(b: DockerBinding, witness?: Witness) {
  return {
    host: b.host,
    target: b.target,
    tag: b.tag,
    source: b.source,
    forgejo: b.forgejo,
    candidateSHA: b.candidateSHA,
    payloadSHA: b.payloadSHA,
    metadataSHA: b.metadataSHA,
    composeSHA: b.composeSHA,
    installerSHA: b.installerSHA,
    installerBytes: b.installerBytes,
    manifest: b.manifest,
    config: b.config,
    image: b.image,
    ...(witness ? {witness: validateWitness(witness)} : {}),
  };
}

// Unlike the candidate's mounted files, these are real public GETs made by the
// phase. No supplied URL, local-cache fallback or registry credential is used.
export async function downloadDockerAssets(b: DockerBinding, io: DockerIO) {
  check(b.executionReleased === true, 'docker_execution_held');
  const result = [];
  for (const name of DOWNLOADS) {
    const url = 'https://github.com/LevitateOS/sodaos/releases/download/' + b.tag + '/' + name,
      id = b.downloads[name]!;
    const response = await fetch(url, {
      method: 'GET',
      redirect: 'follow',
      credentials: 'omit',
      signal: AbortSignal.timeout(180000),
    });
    check(
      response.status === 200 && new URL(response.url).protocol === 'https:' && response.body !== null,
      'public_docker_download'
    );
    const reader = response.body.getReader(),
      parts: Uint8Array[] = [];
    let bytes = 0;
    try {
      for (;;) {
        const part = await reader.read();
        if (part.done) break;
        bytes += part.value.length;
        check(bytes <= id.Bytes, 'public_download_bound');
        parts.push(part.value);
      }
    } finally {
      await reader.cancel().catch(() => {});
      reader.releaseLock();
    }
    const body = new Uint8Array(bytes);
    let offset = 0;
    for (const part of parts) {
      body.set(part, offset);
      offset += part.length;
    }
    check(bytes === id.Bytes && sha(body) === id.SHA256, 'public_download_identity');
    await io.saveBytes('downloads/' + name, body);
    result.push({name, url, SHA256: sha(body), Bytes: bytes});
  }
  await io.save('downloads/public-downloads.json', result);
  return result;
}

export interface DockerHostPaths {
  install_directory: string;
  fixture_directory: string;
}
export function validateDockerHostPaths(host: DockerHostPaths) {
  for (const p of [host?.install_directory, host?.fixture_directory])
    check(
      typeof p === 'string' &&
        p.startsWith('/home/') &&
        path.normalize(p) === p &&
        p.length < 4096 &&
        !/[\u0000-\u001f\u007f]/.test(p),
      'explicit_protected_host_directory'
    );
  check(host.install_directory !== host.fixture_directory, 'distinct_host_directories');
  return host;
}
