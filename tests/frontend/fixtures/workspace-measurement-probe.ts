// Runs only in the local browser test before mounting the emitted workspace.
// Real observers still deliver geometry; the probe retains callbacks to simulate
// deliveries that were queued before retirement, and delays only fonts.ready.
interface MeasurementObserver {
  targets: Element[];
  disconnected: number;
  deliver: () => void;
}
interface MeasurementProbe {
  observers: MeasurementObserver[];
  signals: AbortSignal[];
  beforeFirstRender: number;
  minimumNotifications: number;
  releaseFonts: () => void;
}
declare global {
  interface Window { measurementProbe: MeasurementProbe }
}

export function installMeasurementProbe() {
  const probe: MeasurementProbe = window.measurementProbe = {
    observers: [], signals: [], beforeFirstRender: -1, minimumNotifications: 0,
    releaseFonts: () => {},
  };
  const ready = new Promise<FontFaceSet>(resolve => {
    probe.releaseFonts = () => resolve(document.fonts);
  });
  Object.defineProperty(document.fonts, 'ready', {configurable: true, value: ready});

  const Observer = window.ResizeObserver;
  window.ResizeObserver = class extends Observer {
    private record: MeasurementObserver;
    constructor(callback: ResizeObserverCallback) {
      super(callback);
      this.record = {targets: [], disconnected: 0, deliver: () => callback([], this)};
      probe.observers.push(this.record);
    }
    observe(target: Element, options?: ResizeObserverOptions) {
      this.record.targets.push(target);
      super.observe(target, options);
    }
    disconnect() {
      this.record.disconnected++;
      super.disconnect();
    }
  };
  const add = EventTarget.prototype.addEventListener;
  EventTarget.prototype.addEventListener = function(type, callback, options) {
    if ((this === document.fonts && type === 'loadingdone' || this === window.visualViewport && type === 'resize') &&
      typeof options === 'object' && options.signal) probe.signals.push(options.signal);
    add.call(this, type, callback, options);
  };
  const mount = window.createWorkspaceFixture;
  window.createWorkspaceFixture = mode => {
    const fixture = mount(mode);
    probe.beforeFirstRender = probe.observers.length;
    fixture.root.addEventListener('soda-workspace-minimum', () => probe.minimumNotifications++);
    return fixture;
  };
}
