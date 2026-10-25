// Shared installed-journey input. This describes scope; it grants no authority
// and performs no discovery, validation, provider call or target mutation.
import type {FixtureObservation} from './facts';

/** Public artifact identity; never use this to fingerprint a secret file. */
export interface PublicArtifact {
  readonly path: string;
  readonly sha256: string;
}

export interface CandidateReference {
  readonly source_revision: string;
  readonly candidate_sha256: string;
  readonly payload_sha256: string;
  readonly release_manifest: PublicArtifact;
}

export interface DriverReference {
  readonly source_revision: string;
  readonly executable: PublicArtifact;
}

export type JourneyCase =
  | 'installation'
  | 'browser_project'
  | 'factory'
  | 'persistence'
  | 'app'
  | 'basic_capacity'
  | 'two_issue_capacity'
  | 'cold_recovery';

/** Requested target pins, checked against a separate runtime observation. */
export interface TargetSelection {
  readonly distribution: 'appliance' | 'docker';
  readonly name: string;
  readonly uuid: string;
  readonly origin: string;
  readonly ssh_config_file: string;
}

export interface TargetReference extends TargetSelection {
  /** Null until an actual Docker deployment returns its identity. */
  readonly docker: {
    readonly container_id: string;
    readonly container_name: string;
    readonly manifest: string;
    readonly config: string;
  } | null;
}

/** Native numeric identities use their canonical decimal-string wire form. */
export interface ProjectReference {
  readonly id: string;
  readonly repository_id: string;
  readonly owner_id: string;
}

export interface ApplicationReference {
  readonly id: string;
  readonly data_id: string;
  readonly owner_team: string;
  readonly origin: string;
}

export interface TerminalReference {
  readonly id: string;
  readonly project_id: string;
  readonly actor_id: string;
}

export interface ConnectionReference {
  readonly id: string;
  readonly owner_id: string;
  readonly generation: number;
  /** Retain the canonical state; eligibility belongs to Identity/control. */
  readonly state: string;
}

/** Returned inventory, including retained resources from preceding attempts. */
export interface FixtureInventory {
  readonly projects: readonly ProjectReference[];
  readonly applications: readonly ApplicationReference[];
  /** Null means unobserved, not an empty inventory or permission to stop none. */
  readonly terminals: readonly TerminalReference[] | null;
  readonly connections: readonly ConnectionReference[];
}

/** Existing resources this journey may manage; membership alone is not consent. */
export interface ManagedResources {
  readonly project_ids: readonly string[];
  readonly application_ids: readonly string[];
  readonly terminal_ids: readonly string[];
}

export interface FactorySubject {
  // Requested selections/pins, not proof that the appliance currently has them.
  readonly repository_id: string;
  readonly project_id: string;
  readonly issue_number: number;
  readonly native_issue_id: string;
  readonly source_commit: string;
  readonly acceptance: string;
  readonly requirements: string;
  readonly approval: string;
  readonly coder_preparation: string;
  readonly reviewer_preparation: string;
  readonly connection_id: string;
  readonly operation_id: string;
  readonly actors: {
    readonly coder: string;
    readonly reviewer: string;
    readonly ci: string;
  };
  readonly required_checks: readonly string[];
  readonly check_script_sha256: string;
}

// These fields preserve internal/factory/publication.go's distinct identities.
// Future publication facts are outputs, never an invented initial assignment ID.
export interface PublicationReference {
  readonly id: string;
  readonly assignment_id: string;
  readonly run: string;
  readonly project_id: string;
  readonly acceptance: string;
  readonly candidate: string;
}

export interface ModelLimits {
  readonly sponsor_minutes: number;
  readonly rolling_minutes: number;
  readonly max_concurrent: number;
  readonly intent_minutes: number;
  readonly attempt_minutes: number;
  readonly maximum_seconds: number;
  readonly automatic_retry: false;
  readonly automatic_refill: false;
  readonly extra_inquiries: false;
}

export interface FactoryCase {
  readonly subject: FactorySubject;
  /** Per-repository sponsorship and the referenced connection's rolling limit. */
  readonly limits: ModelLimits;
}

export interface PermittedEffects {
  readonly fixture_setup: boolean;
  readonly project_lifecycle: readonly string[];
  readonly application_lifecycle: readonly string[];
  readonly terminal_lifecycle: readonly string[];
  readonly factory_repositories: readonly string[];
  readonly current_context_phases: readonly ('implement' | 'review')[];
  readonly same_image_recreation: boolean;
  readonly cold_restore: boolean;
}

/** Paths only: private values and their digests never belong in this input. */
export interface CredentialReferences {
  readonly guest_ssh_key_file: string;
  readonly workspace_owner_password_file: string;
  readonly application_owner_files: readonly string[];
}

export interface QualificationScenario {
  readonly format: 1;
  readonly candidate: CandidateReference;
  readonly driver: DriverReference;
  readonly target: TargetSelection;
  readonly cases: readonly JourneyCase[];
  readonly managed: ManagedResources;
  readonly factory: readonly FactoryCase[];
  readonly credentials: CredentialReferences;
  readonly permitted_effects: PermittedEffects;
  readonly authority: {
    /** A user hold survives even when a matching scope was already approved. */
    readonly held: boolean;
    /** Null means no recorded approval; a non-null path still needs checking. */
    readonly scope_receipt_file: string | null;
  };
}

/** The collector snapshots these resolved bytes before dispatch. References to
 * older canonical receipts preserve history without copying their contents.
 * This document is input, not another controller request or qualification seal.
 */
export interface ResolvedQualificationInput {
  readonly format: 1;
  readonly scenario: QualificationScenario;
  readonly observation: FixtureObservation;
  readonly prior_attempts: readonly {
    readonly request: PublicArtifact;
    readonly executed: PublicArtifact | null;
    readonly release_evidence: PublicArtifact | null;
  }[];
}
