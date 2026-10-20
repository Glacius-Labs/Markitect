// Package realization describes artifact ownership and file accounting over
// explicitly supplied model and snapshot values. It performs no acquisition,
// filesystem, process, provider or scheduling operations.
//
// An Owner is an explicit root or local-Package responsibility reference, not an
// organization Manager identity or an inferred assignment. Registry construction
// fixes artifacts and rejects ambiguous file ownership. Model membership and the
// existence and classification of actual files are separate accounting checks.
// Artifact paths retain snapshot's exact spelling. Definition links are
// many-to-many and retain model.Identity without introducing another model
// identity or canonical digest. Reasons are required explanations, not evidence
// that natural-language obligations have been satisfied.
//
// A Layout is the project's declared persistent ownership: explicit
// repository-relative directory and exact-file areas, each owned by Root or a
// local Package. The most specific area owns a path; equal specificity and
// paths no area covers are refused or reported as layout.no_owner to Root. It
// is validated against the local Package inventory before any work starts,
// including letter-case ambiguity for case-insensitive platforms. Registry
// entries are exact files or directories; a directory entry covers the
// eligible files below it that its registrant owns, and the most specific
// entry registers a file. When ownership is declared, accounting requires every
// entry to lie in its registrant's area; without it the registry shards alone
// decide. A Layout grants no write authority.
//
// Registry storage belongs outside canonical Definition roots. Adapters supply
// acquired registry records and exact protected/ignored input classifications;
// this package neither reads a registry file nor interprets source provenance as
// a physical repository path. Operational .markitect/state is never realization.
// Fixed values expose no mutable input or output slice aliases. A Protection
// names canonical or control inputs that no Manager may change.
package realization
