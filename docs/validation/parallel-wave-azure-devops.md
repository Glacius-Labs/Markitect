# Azure DevOps adapter wave evidence

This bounded wave adds an independent read-only adapter for one provider
object family: Azure DevOps Git repository `defaultBranch` metadata. Its
contract and exact limits are documented in the [adapter README](../../cmd/markitect-adapter-azure-devops/README.md).

The source depends only on the existing command-adapter request/result
protocol and normalized semantic model. It introduces no Core, CLI, shared
DTO, schema, or other-adapter changes. The target identity names one Azure
DevOps organization and project; repository mappings bind exact canonical
`identity.key` values to repository GUIDs and captured files. Each canonical
resource supplies `data.defaultBranch`; provider parameters do not repeat this
desired value.

The adapter runs offline from exact staged JSON captures. It validates the
recorded `GET` URL and API version, HTTP status, body repository/project IDs,
and branch shape. `observe` returns a digest of the captured bytes and selected
metadata. `plan` reports conformance findings and always has an empty
`operations` list. `verify` fails when the capture no longer matches the saved
completed plan. Negative evidence includes malformed or multiple JSON values,
wrong URL/body identities, non-200 responses, unsupported API versions,
ambiguous mappings, missing captures, malformed branch values, and staged-path
escapes. Capture inputs must be regular files of at most 1 MiB. JSON keys must
be unique ignoring case throughout each capture, matching Go's case-insensitive
field decoding; repeated `api-version` query parameters are rejected, while
unconsumed Azure response fields remain accepted when their names do not
collide ignoring case. The adapter test suite also
round-trips Markitect's real `app.AdapterRequest` and `app.SemanticModel` DTOs
through the adapter wire decoder to check the existing SPI boundary.

## Evidence boundary

The result proves only that the supplied capture is internally consistent
with the configured target/mapping and that its recorded `defaultBranch`
matches or differs from canonical `data.defaultBranch`. A capture digest binds
the observation to those bytes; it does not authenticate the capture, prove
when it was obtained, establish the caller's permissions, prove current remote
state, or describe other repository settings. No live Azure request,
credential, apply action, or remote operation is part of this wave. The
official [Get Repository REST API 7.1 documentation](https://learn.microsoft.com/en-us/rest/api/azure/devops/git/repositories/get-repository?view=azure-devops-rest-7.1)
defines the endpoint and response fields used by the fixture contract.

The adapter is tested as an isolated package using deterministic captures.
Those tests do not substitute for Markitect's normal Windows/Linux CI gates,
human review, signed provenance, or adopter acceptance. Broader provider
integration and any live observation remain separate decisions.
