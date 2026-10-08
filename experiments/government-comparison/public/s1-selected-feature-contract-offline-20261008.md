# Selected-feature contract: offline correction

Date: 2026-10-08. Assignment `s1-selected-feature-contract-offline-20261008`.
Base `f4bea6dd9b92d72d828751d9a05411c86edd66fa`. **Offline only; no run grant.**

The new pure module `runtime/selected_feature_contract.py` separates two contracts:

- `config.features` is not typed by the pinned Config schema. The parser retains
  only the eight requested names: apps, goals, hooks, memories, multi_agent,
  plugins, shell_tool and unified_exec. Each is missing, null, present with an
  actual boolean, or invalid. Only the expected eight values satisfy this
  selected precondition. The rest is explicitly not evaluated and unpublished;
  no claim is made that the whole feature map is valid.
- Managed `featureRequirements` is a schema-wide boolean map. Every value must
  still be boolean. Any name outside the interpreted eight remains unresolved
  and makes the result insufficient, without exposing that name or value.
  This includes the three extra names in the old sanitizer's FLAGS set; that set
  is not a schema name enum. A known selected conflict remains incompatible,
  including when an uninterpreted requirement is also present.

Both maps retain the 256-entry ceiling. Fixed safe codes distinguish container
absence/null/type/size, selected field absence/null/type, opposite-value conflict,
invalid typed requirements and unresolved managed constraints. Arbitrary errors,
foreign field names and values are not published as diagnostics.

The actual `sanitize_config_read` consumer removes feature maps from the archived
parser input at the top level and in layers, then attaches the new selected
observations. Nonselected feature origins are omitted. `sanitize_requirements`
uses a distinct typed-map parser. `assess` checks the derived observations, then
normalizes their verified eight booleans for the archived consumer's unchanged
origin/default/model/legacy/other-managed/Windows gates. No positive state is
supplied by tests or invented from layer echoes. Strict JSON/duplicate rejection,
raw warning/provisional guards, metadata privacy and all unaffected gates are
reused from the hash-pinned preserved client. No transport, launch profile,
reservation or driver was introduced or changed.

[Focused offline evidence](../evidence/s1-selected-feature-contract-offline-20261008/offline-validation.json)
records 10/10 tests, including all 40 selected-field negative variants. Real
sanitizers and `assess` process raw synthetic inputs; parsers are not mocked.
Cases include foreign string/null/object values at top/layer scope, nonselected
origins, bounds, schema-wide requirement types and unknown names, known conflicts,
duplicate JSON, warnings, origins, and the saved terminal receipt. The narrow
independent source/test review is retained with the evidence packet.

The fourth real policy session remains **insufficient**, byte for byte. Its raw
feature map is unavailable; these synthetic extras are not evidence of its
actual failure. Effective selected features, requirements, active permissions,
real tools, serving identity and the earlier exec denial cause remain unobserved.
The new module is an offline candidate component, not a repaired real run.

No new actual preparation/admission/freeze, CLI/app-server/metadata/Actor/model/
provider/product/study execution or ledger write occurred. History remains four
policy trees, six CLI metadata calls, five Actors, 53,331 known tokens and unknown
total, and native product 15/16/13/2,250. S1 stays open; all six cells NOT RUN.
A later actual invocation requires new finite authority after this source is
accepted. There is no automatic continuation.
