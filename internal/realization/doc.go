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
// A Layout is the project's persistent ownership map: artifact roots that mirror
// the model hierarchy plus explicit exceptions, deepest mapping wins. Root owns
// the files directly in an artifact root and every file outside the roots; a
// file in an unmapped subdirectory of a root has no owner, which accounting
// reports as layout.no_owner to Root. It is validated against the local Package inventory
// before any work starts, including letter-case ambiguity for case-insensitive
// platforms. When a Layout is declared, accounting additionally requires every
// registration to sit in the registry shard of its path's owner; without one the
// registry shards alone decide. A Layout grants no write authority.
//
// Registry storage belongs outside canonical Definition roots. Adapters supply
// acquired registry records and exact protected/ignored input classifications;
// this package neither reads a registry file nor interprets source provenance as
// a physical repository path. Operational .markitect/state is never realization.
// Fixed values expose no mutable input or output slice aliases. A Protection
// names canonical or control inputs that no Manager may change.
package realization
