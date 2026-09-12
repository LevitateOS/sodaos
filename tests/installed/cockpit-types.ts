/** Minimal upstream browser API used only by the authorized stock-Cockpit probe. */
interface CockpitProcess extends PromiseLike<string> { input(data: string): CockpitProcess }
interface CockpitHTTP { request(options: {method: string; path: string; body: string}): PromiseLike<string>; close(): void }
export interface Cockpit {
  manifests: Record<string, unknown>;
  spawn(command: string[], options: {err: 'message' | 'out'}): CockpitProcess;
  http(path: string, options: {superuser: 'require'; headers: Record<string, string>}): CockpitHTTP;
  logout(reload: boolean): void;
}
declare global {interface Window {cockpit: Cockpit}}
