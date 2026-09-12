import {check, object} from '../spaces/sodaspaces-api.js';

const revision = (v: unknown, host = false): string => {check(typeof v === 'string' && (host ? /^[a-f0-9]{64}$/ : /^(?:0|[a-f0-9]{32})$/).test(v)); return v;};
const text = (v: unknown, max = 253): string => {check(typeof v === 'string' && v.length <= max && !/[\x00-\x1f\x7f]/.test(v)); return v;};
const bool = (v: unknown): boolean => {check(typeof v === 'boolean'); return v;};
const list = (v: unknown, max: number): unknown[] => {check(Array.isArray(v) && v.length <= max); return v;};
const address = (v: unknown): string => {
  const s = text(v, 64); check(/^[a-f0-9:.]+$/.test(s));
  if (s.includes(':')) {const url = new URL('http://[' + s + ']/'); check(url.hostname.startsWith('['));}
  else check(/^(?:0|[1-9][0-9]{0,2})(?:\.(?:0|[1-9][0-9]{0,2})){3}$/.test(s) && s.split('.').every(part => Number(part) <= 255));
  return s;
};
const dns = (v: unknown): string => {const s = text(v); check(s === '' || /^[a-z0-9.-]+$/i.test(s)); return s;};
const tag = (v: unknown): string => {const s = text(v, 67); check(/^tag:[a-zA-Z][a-zA-Z0-9-]{0,62}$/.test(s)); return s;};
function peer(value: unknown) {
  const v = object(value), id = text(v.id, 128); check(id !== '');
  return {id, dns_name: dns(v.dns_name), addresses: list(v.addresses, 16).map(address), online: bool(v.online), exit_node: bool(v.exit_node), expired: bool(v.expired)};
}
export function hostView(value: unknown) {
  const v = object(value), p = object(v.preferences);
  const state = text(v.state, 32);
  check(['NoState', 'InUseOtherUser', 'NeedsLogin', 'NeedsMachineAuth', 'Stopped', 'Starting', 'Running'].includes(state));
  check(Number.isSafeInteger(v.health_issues) && typeof v.health_issues === 'number' && v.health_issues >= 0 && v.health_issues <= 128);
  const peers = list(v.peers, 128).map(peer); check(new Set(peers.map(p => p.id)).size === peers.length);
  const result = {revision: revision(v.revision, true), state, tailnet: text(v.tailnet), magic_dns_enabled: bool(v.magic_dns_enabled),
    have_node_key: bool(v.have_node_key), expired: bool(v.expired), dns_name: dns(v.dns_name), addresses: list(v.addresses, 16).map(address),
    peers, health_issues: v.health_issues, preferences: {want_running: bool(p.want_running), exit_node_id: text(p.exit_node_id, 128),
      exit_node_ip: p.exit_node_ip === '' ? '' : address(p.exit_node_ip), allow_lan: bool(p.allow_lan), advertise_exit_node: bool(p.advertise_exit_node)}};
  check(state !== 'Running' || (result.have_node_key && result.addresses.length > 0 && result.tailnet !== ''));
  return result;
}
export type Host = ReturnType<typeof hostView>;
export function enrollmentView(value: unknown) {
  const v = object(value), tags = list(v.tags, 8).map(tag);
  check(tags.every((t, i) => i === 0 || t > (tags[i - 1] || '')));
  const result = {revision: revision(v.revision), binding: text(v.binding, 32), tailnet: text(v.tailnet), tags,
    configured: bool(v.configured), admission: bool(v.admission), default: bool(v.default), preauthorized: bool(v.preauthorized),
    credential_checked: bool(v.credential_checked), enrollment_verified: bool(v.enrollment_verified), runtime_supported: bool(v.runtime_supported)};
  // Do not accidentally unlock unimplemented project runtime on a newer/mixed helper.
  check(!result.runtime_supported && !result.enrollment_verified && !result.default);
  if (result.configured) {
    check(result.revision !== '0' && /^[a-f0-9]{32}$/.test(result.binding) && /^[a-z0-9][a-z0-9.@_-]{0,252}$/i.test(result.tailnet) && tags.length > 0 && result.credential_checked);
  } else check(result.revision === '0' && result.binding === '' && result.tailnet === '' && tags.length === 0 && !result.admission && !result.preauthorized && !result.credential_checked);
  return result;
}
export type Enrollment = ReturnType<typeof enrollmentView>;
export function settingsView(value: unknown) {
  const v = object(value), unavailable = bool(v.host_unavailable);
  check((v.host === null) === unavailable);
  return {host: v.host === null ? null : hostView(v.host), host_unavailable: unavailable, enrollment: enrollmentView(v.enrollment)};
}
export type Settings = ReturnType<typeof settingsView>;
export function authenticationURL(value: unknown): string {
  const s = text(value, 256);
  check(/^https:\/\/login\.tailscale\.com\/a\/[A-Za-z0-9_-]{1,128}$/.test(s)); return s;
}
export function hostResult(value: unknown, action: string) {
  const v = object(value), outcome = text(v.outcome, 16), unavailable = bool(v.readback_unavailable);
  check(['observed', 'pending', 'confirmed', 'unconfirmed'].includes(outcome) && (v.host === null) === unavailable);
  const authURL = v.auth_url === undefined ? '' : authenticationURL(v.auth_url);
  check(!authURL || ['signin', 'authentication'].includes(action));
  check(outcome !== 'pending' || authURL !== '');
  return {outcome, host: v.host === null ? null : hostView(v.host), readback_unavailable: unavailable, authURL};
}
export function enrollmentResult(value: unknown, action: string, previous: string) {
  const v = object(value), enrollment = enrollmentView(v.enrollment), saved = bool(v.saved);
  check(v.outcome === 'confirmed' && bool(v.credential_checked) && saved === (action !== 'check'));
  check(saved ? enrollment.configured && enrollment.revision !== previous : enrollment.revision === previous);
  return {saved, enrollment};
}
