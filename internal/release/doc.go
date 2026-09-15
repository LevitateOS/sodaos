// Package release groups Soda release construction: build primitives,
// host-image assembly, artifact qualification, and signed delivery. The
// subpackages form one directional pipeline:
//
//	image and qualify coordinate builds
//	  ↓
//	build executes shared build primitives (OCI, CoreOS, bundle)
//	  ↓
//	deliver owns payload metadata, signing and publication
//
// image assembles and prepares host images but never qualifies or publishes
// them. qualify admits artifacts and observes guest state but never builds.
// deliver signs and publishes but never rebuilds tested bytes. acceptance
// (top-level) remains the outside VM/evidence harness both qualify and the
// soda-acceptance tool consume.
package release
