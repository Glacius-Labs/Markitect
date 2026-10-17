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
// Fixed values expose no mutable input or output slice aliases. Path checks use
// explicit protected areas, creation selections and current ownership facts.
// Existing owners may repair dangling artifacts without complete accounting;
// new exact-file claims cannot acquire registered or unexplained existing files.
// A claim records conditional admissibility, not live workspace authority. Host
// admission, transfers and run states remain separate responsibilities.
// Candidate capture derives actual tree and logical registry differences. Every
// changed/restored artifact needs current ownership and model basis; unchanged old
// problems do not imply completion. Replay composes one delivery's owned changes
// onto the run's accepted aggregate (core rule D2) without a common base and
// without accepting deliveries or causing effects.
// Definition census checks copied proposals against exact own or project scope,
// with artifact eligibility computed from the bound classification facts. It
// checks minimum form and references, never evidence existence or semantic support;
// those remain Host and independent reviewer responsibilities respectively.
// Candidate records retain closed bounded immutable historical subjects. Decode
// reuses the same capture/accounting/census core without reconstructing compiler
// success. A record hash binds bytes; acquisition and live applicability remain
// external obligations. A record carries the path claims its new files used.
package realization
