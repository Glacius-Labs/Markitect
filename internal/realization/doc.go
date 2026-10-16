// Package realization describes artifact ownership and file accounting over
// explicitly supplied model and snapshot values. It performs no acquisition,
// filesystem, process, provider or scheduling operations.
//
// An Owner is an explicit root or local-Package responsibility reference, not an
// organization Manager identity or an inferred assignment. Registry construction
// fixes artifacts and rejects ambiguous file ownership. Model membership and the
// existence and classification of actual files are separate accounting checks.
// Artifact paths retain snapshot's exact spelling; they need not mirror Packages.
// Definition links are many-to-many and retain model.Identity without introducing
// another model identity or canonical digest. Reasons are required explanations,
// not evidence that natural-language obligations have been satisfied.
//
// Registry storage belongs outside canonical Definition roots. Adapters supply
// acquired registry records and exact protected/ignored input classifications;
// this package neither reads a registry file nor interprets source provenance as
// a physical repository path. Operational .markitect/state is never realization.
// Fixed values expose no mutable input or output slice aliases. Claims, transfers,
// candidate composition, Definition assessments and run states are separate cuts.
package realization
