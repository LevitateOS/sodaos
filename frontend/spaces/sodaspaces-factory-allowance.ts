// Coder-time allowance holds. The factory status read carries the latest
// dispatch pass's waits; an allowance_nearly_used wait means SodaOS held a
// run because the sponsorship allowance would cut it short, and an
// allowance_exhausted wait marked coder_time means it is used up. The Feed and
// Work views say so in plain words and name the setting that fixes it.

export interface AllowanceHold {
  readonly repository: string;
  readonly issue?: string;
  readonly minutesLeft: number;
  readonly allowanceMinutes: number;
}

const settingsPath = 'Settings → SodaOS → Factory control';

function minutes(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= 100000 ? value : undefined;
}

// allowanceHolds reads the allowance holds from one factory status body.
// Anything malformed is skipped: the notice is advisory, never a gate.
export function allowanceHolds(status: unknown, repository: string): AllowanceHold[] {
  if (!status || typeof status !== 'object') return [];
  const observation = (status as Record<string, unknown>).dispatch_observation;
  if (!observation || typeof observation !== 'object') return [];
  const waits = (observation as Record<string, unknown>).waits;
  if (!Array.isArray(waits)) return [];
  const holds: AllowanceHold[] = [];
  for (const wait of waits.slice(0, 64)) {
    if (!wait || typeof wait !== 'object') continue;
    const raw = wait as Record<string, unknown>;
    if (raw.coder_time !== true) continue;
    const left = raw.reason === 'allowance_exhausted' ? 0 : minutes(raw.minutes_left);
    const total = minutes(raw.allowance_minutes) ?? 0;
    if (left === undefined) continue;
    const issue = typeof raw.issue === 'string' && /^\d+$/.test(raw.issue) ? raw.issue : undefined;
    holds.push({repository, ...(issue ? {issue} : {}), minutesLeft: left, allowanceMinutes: total});
  }
  return holds;
}

// allowanceHoldText says why work waits and where to fix it.
export function allowanceHoldText(holds: readonly AllowanceHold[]): string {
  const first = holds[0];
  if (!first) return '';
  const issues = [...new Set(holds.flatMap((hold) => (hold.issue ? [`#${hold.issue}`] : [])))];
  const what = issues.length > 0 ? `Issue ${issues.join(', ')} is waiting` : 'Work is waiting';
  const left =
    first.minutesLeft === 0
      ? 'the coder-time allowance is used up'
      : first.allowanceMinutes > 0
        ? `the coder-time allowance is nearly used (${first.minutesLeft} of ${first.allowanceMinutes} minutes left)`
        : `the coder-time allowance is nearly used (${first.minutesLeft} minutes left)`;
  return `${what}: ${left}, too little for a run to do useful work. Raise Allowance minutes in ${settingsPath} to continue.`;
}
