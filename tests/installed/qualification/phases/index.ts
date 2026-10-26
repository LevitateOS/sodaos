// The maintained phase bundle's public composition boundary. No phase runs on import.
export {collectDockerPhases, dockerPhaseConfiguration, innerDocker} from './docker-phases';
export type {DockerPhaseDetails, DockerPhaseRuntime} from './docker-phases';
export {FactoryPermits, deriveFactoryEvidence, exerciseFactoryPhase} from './factory-phase';
export {PersistencePermits, retireOwnedResources, exercisePersistence} from './docker-persistence';
export {nativeExecutionScript, nativeExecutionSucceeded} from './factory-execution';
export {networkGuard} from './support';
export {issueFactoryCommand, reconcileFactoryCommand, factoryCommandBinding} from './factory-command';
export {adoptSetup} from '../setup';
export {applicationFixture, adoptApplication, applicationNativeProbe} from '../application';
export type {ApplicationFixture, ApplicationHandoff, ApplicationTransport} from '../application';
export {openApplicationOwner} from '../application-owner';
export type {ApplicationRecordWitness, ApplicationJournal} from '../application-owner';
