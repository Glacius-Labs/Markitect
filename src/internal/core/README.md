# Structural Core

`internal/core` provides a deterministic compiler for explicitly supplied Schemas and Definitions. It performs no file, Git, provider, network, policy, or execution operations. A Host decodes and selects the inputs, supplies their provenance, and binds the resulting Model to a source revision.

```go
model, diagnostics := core.Compile(schemas, definitions, revision)
```

A successful Model contains Schemas and Definitions in stable identity order, resolved reference edges, source provenance, the supplied revision, and a semantic digest. Compilation returns diagnostics and an empty Model on any structural failure. The digest covers canonical Schema contracts and Definition values; it excludes source paths, source digests, and revision so evidence remains separately bound.

Definition identity is the exact tuple `(apiVersion, kind, namespace, name)`. `DefinitionIdentity.Key()` encodes that tuple as JSON, avoiding delimiter collisions. Package/module origin belongs to `Source`, never identity. `Schema.APIVersion` and `KindIdentity` are also exact nominal identities; multiple API versions can be explicitly compiled together.

Properties are closed typed contracts. Supported types are `string`, `boolean`, `integer`, `number`, `enum`, `object`, `reference`, and `kindReference`. `MinCount` and `MaxCount` are independent of the value type. `MaxCount == 1` means a singleton scalar/object; any other maximum means a list, even for one element. `Unbounded` is `-1`; unbounded values still fit within compiler limits. Omitted values count as zero, and `null` is invalid. `object` values reject undeclared fields. Enum values are strings.

A `reference` Property declares an exact target `KindIdentity`. Its value has a required `namespace` (the empty string is a real namespace) and `name`; `apiVersion` and `kind` may be omitted and are normalized from the Property contract, or supplied and checked exactly. Target Definitions must exist among the explicitly supplied Schemas and Definitions. Each resolved reference becomes an Edge with source identity, exact property path (including list index), target identity, and provenance. Cycles are allowed. A `kindReference` value contains exact `apiVersion` and `kind`; the target Kind must be supplied. Kind references do not create Definition edges or activate any policy.

Input is bounded: at most 256 Schemas, 10,000 Definitions, 2,048 object fields, depth 32 for Schema objects, depth 72 for direct Go input structures, 10,000 values per list, 8 MiB per normalized Schema or Definition, and 128 MiB combined normalized input. A compile-wide preflight also caps expanded traversal at 100,000 nodes and 128 MiB of projected JSON-escaped content; repeated aliases in caller-owned Go maps and slices count at every occurrence. Definition Specs accept exact builtin JSON-compatible scalars, `json.Number`, `map[string]any`, and `[]any`; structs, pointers, custom named types, and custom marshalers are rejected before encoding. Host YAML codecs should apply the same or stricter byte/count limits before allocation.
