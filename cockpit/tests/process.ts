import { vi } from "bun:test";
import type { CockpitProcess } from "../src/cockpit/types";

export function pendingProcess() {
  let resolve!: (value: string) => void;
  let reject!: (reason: unknown) => void;
  let emit = (_chunk: string) => {};
  const promise = new Promise<string>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  const process: CockpitProcess = Object.assign(promise, {
    input: vi.fn((_data: string) => process),
    stream: vi.fn((callback: (chunk: string) => void) => {
      emit = callback;
      return process;
    }),
    close: vi.fn((reason?: string) => reject({ problem: reason })),
  });
  return { process, resolve, reject, emit: (chunk: string) => emit(chunk) };
}
