import {check, id, object} from './sodaspaces-api.js';

export interface AcceptanceSource {
  readonly id: string;
  readonly digest: string;
  readonly content_version: number;
}
export interface AcceptanceDependency {
  readonly occurrence: string;
  readonly depends_on: string;
}
export interface AcceptancePreview {
  readonly repository: string;
  readonly issue: string;
  readonly title_digest: string;
  readonly title_occurrence: string;
  readonly body_digest: string;
  readonly content_version: number;
  readonly native_revision: number;
  readonly comments: readonly AcceptanceSource[];
  readonly dependencies: readonly AcceptanceDependency[];
}

function counter(value: unknown): number {
  check(typeof value === 'number' && Number.isSafeInteger(value) && value >= 0);
  return value;
}
function digest(value: unknown): string {
  check(typeof value === 'string' && /^[a-f0-9]{64}$/.test(value));
  return value;
}

export function acceptanceCommentIDs(input: string): string[] {
  if (!input.trim()) return [];
  const ids = input.trim().split(/[\s,]+/);
  check(ids.length <= 50 && ids.every(id) && new Set(ids).size === ids.length);
  return ids;
}

export function acceptancePreview(
  value: unknown,
  repository: string,
  issue: string,
  selected: readonly string[]
): AcceptancePreview {
  const raw = object(value);
  check(id(repository) && id(issue) && raw.repository === repository && raw.issue === issue);
  check(raw.title_occurrence === 'original' || id(raw.title_occurrence));
  check(Array.isArray(raw.comments) && raw.comments.length === selected.length);
  const wanted = new Set(selected);
  const comments = raw.comments.map((value) => {
    const source = object(value);
    check(id(source.id) && wanted.delete(source.id));
    return {id: source.id, digest: digest(source.digest), content_version: counter(source.content_version)};
  });
  check(wanted.size === 0 && Array.isArray(raw.dependencies) && raw.dependencies.length <= 50);
  const seen = new Set<string>();
  const dependencies = raw.dependencies.map((value) => {
    const dependency = object(value);
    check(id(dependency.occurrence) && id(dependency.depends_on) && !seen.has(dependency.occurrence));
    seen.add(dependency.occurrence);
    return {occurrence: dependency.occurrence, depends_on: dependency.depends_on};
  });
  const revision = counter(raw.native_revision);
  check(revision > 0);
  return {
    repository,
    issue,
    title_digest: digest(raw.title_digest),
    title_occurrence: raw.title_occurrence,
    body_digest: digest(raw.body_digest),
    content_version: counter(raw.content_version),
    native_revision: revision,
    comments,
    dependencies,
  };
}

export function acceptanceReceipt(value: unknown, command: string, decision: string): void {
  const raw = object(value);
  check(raw.command_id === command && raw.decision_id === decision && raw.head === decision && counter(raw.depth) > 0);
}

export function acceptanceErrorMessage(value: unknown): string | undefined {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return undefined;
  const raw = object(value);
  if (typeof raw.error !== 'string' || !/^[a-z_]{1,64}$/.test(raw.error)) return undefined;
  return typeof raw.message === 'string' && raw.message.length <= 1024 ? raw.message : undefined;
}
