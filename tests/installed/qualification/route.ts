// Source capability, not an installed outcome or paid authority. The collector
// checks actual App owner/record, terminal and cold tools before Factory opens.
import type {QualificationScenario} from './scenario';

export function implementedDockerRoute(s: QualificationScenario): boolean {
  if (
    s.target.distribution !== 'docker' ||
    s.factory.length !== 1 ||
    s.cases.slice(0, 4).join(',') !== 'installation,browser_project,factory,persistence'
  )
    return false;
  if (s.cases.length === 4) return true;
  return (
    s.cases.slice(4).every((c) => ['app', 'basic_capacity', 'cold_recovery'].includes(c)) &&
    s.cases.includes('app') &&
    s.cases.includes('cold_recovery') &&
    s.applications?.length === 1
  );
}
