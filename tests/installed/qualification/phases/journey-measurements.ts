// Supplemental observations share Factory's existing read cadence. There is no
// polling timer, background writer, extra SSH read or provider operation here.
import assert from 'node:assert/strict';
import type {openApplicationOwner, ApplicationRecordWitness} from '../application-owner';
import {projectNativeMeasurement} from './native-measurement';
import {exerciseFactoryPhase, type FactoryIO, type FactoryPermits} from './factory-phase';

export interface TerminalProbe {
  /** Must read the retained real TTY and check its original native identity. */
  sample(): Promise<void>;
  dispose(): void;
}
export interface JourneyProbes {
  application: {owner: Awaited<ReturnType<typeof openApplicationOwner>>; witness: ApplicationRecordWitness};
  terminal: TerminalProbe;
  save(relative: string, value: unknown): Promise<unknown>;
}

export async function startJourneyMeasurements(input: JourneyProbes) {
  let sequence = 0,
    stopped = false,
    failed = false,
    nativeSamples = 0;
  const started = new Date().toISOString();
  // App and terminal requests are independent; neither records its wire body.
  const response = async (probe: 'application' | 'terminal') => {
    const began = new Date().toISOString(),
      clock = performance.now();
    try {
      if (probe === 'application') await input.application.owner.readRendered(input.application.witness);
      else await input.terminal.sample();
      return {
        probe,
        started: began,
        finished: new Date().toISOString(),
        elapsed_ms: performance.now() - clock,
        observed: true,
      };
    } catch {
      failed = true;
      return {
        probe,
        started: began,
        finished: new Date().toISOString(),
        elapsed_ms: performance.now() - clock,
        observed: false,
      };
    }
  };
  const sample = async (snapshot?: unknown) => {
    assert(!stopped && sequence < 256, 'measurement_window_closed_or_full');
    const index = sequence++;
    let native = null;
    if (snapshot !== undefined) {
      try {
        native = projectNativeMeasurement(snapshot);
        nativeSamples++;
      } catch {
        failed = true;
      }
    }
    const responses = await Promise.all([response('application'), response('terminal')]);
    await input.save('measurements/sample-' + String(index).padStart(3, '0') + '.json', {
      format: 1,
      sequence: index,
      native,
      responses,
      // Never infer these from response speed, native activity or elapsed time.
      lifecycle_evidence: 'factory/',
      capacity: null,
    });
  };
  try {
    await sample();
    assert(!failed, 'measurement_startup_unconfirmed');
  } catch (error) {
    input.terminal.dispose();
    throw error;
  }
  let finished: Promise<{observed: boolean; native_samples: number; response_samples: number}> | undefined;
  return {
    async observeSnapshot(snapshot: unknown) {
      // Baseline failure stops before opening policy. Later probe failures are
      // deferred so Factory can still pause and reconcile its own accounting.
      const first = nativeSamples === 0;
      try {
        await sample(snapshot);
      } catch {
        failed = true;
      }
      if (first) assert(!failed && nativeSamples > 0, 'measurement_native_startup_unconfirmed');
    },
    finish() {
      return (finished ??= (async () => {
        stopped = true;
        input.terminal.dispose();
        const result = {
          observed: !failed && nativeSamples > 0,
          native_samples: nativeSamples,
          response_samples: sequence,
        };
        await input.save('measurements/result.json', {
          format: 1,
          started,
          finished: new Date().toISOString(),
          ...result,
          cadence: 'existing Factory native reads; no extra polling',
          owner_team: input.application.witness.owner_team,
          record: {
            id: input.application.witness.id,
            version: input.application.witness.version,
            sha256: input.application.witness.record_sha256,
          },
          capacity: null,
          lifecycle_evidence: 'factory/',
          production_qualification: false,
        });
        return result;
      })());
    },
  };
}

/** App login/record and the genuine terminal must already have their owners.
 * The Factory baseline arms native observation before its first policy effect.
 * This wrapper grants no authority and never opens another paid trial. */
export async function exerciseMeasuredFactoryPhase(
  page: Parameters<typeof exerciseFactoryPhase>[0],
  cfg: Parameters<typeof exerciseFactoryPhase>[1],
  binding: Parameters<typeof exerciseFactoryPhase>[2],
  io: FactoryIO,
  permits: FactoryPermits,
  probes: JourneyProbes
) {
  assert(!io.observeNative, 'single_native_measurement_owner');
  const measurements = await startJourneyMeasurements(probes);
  try {
    const factory = await exerciseFactoryPhase(
      page,
      cfg,
      binding,
      {...io, observeNative: measurements.observeSnapshot},
      permits
    );
    const supplemental = await measurements.finish();
    assert(supplemental.observed, 'supplemental_measurements_incomplete');
    return {factory, supplemental};
  } finally {
    await measurements.finish();
  }
}
