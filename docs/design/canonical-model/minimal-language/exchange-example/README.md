# Complete simplified Exchange example

As of: 16 October 2026. **Current status: not conformant under the newly chosen spelling rules because of Command/command.** At the user's request, the domain modelling of these two Definitions is to be clarified before any renaming or merging. The current verifier must report the conflict and must not correct it itself.

**Authoring organization:** The chosen convention is package by feature / vertical slices: related Kinds, Properties and concrete items are stored together by domain, still with one file per Definition. The older structure of this vertical slice still separates vocabulary/ and project/ in order to try out imports and reference forms. It is not a recommended template for the Package split. [Section 7.3](../language-contract.md#packages) shows the domain grouping; the structure verifier does not decide on its domain appropriateness on the basis of folder names.

This design example uses the completed [v1 language baseline](../language-contract.md), confirmed by the owner on 16 October 2026. It shows one file per Definition, file-local imports, short Property IDs, and the nine Definition Core. The known Command/command collision is in the example vocabulary; it is not an open language decision. This example is not an input format for the current product compiler, and it is not an executed connector. The product compiler is implemented in Go. The Python probe covers only the checks listed in its verification record; it does not choose Python or PyYAML for the product.

The report from before the design baseline remains in [verification-before-v1-design-baseline.json](verification-before-v1-design-baseline.json); [verification.json](verification.json) records the updated status. The completed design changes no language rule, Core Definition, canonical Core bytes or external lock binding. The product compiler is implemented in Go. This example's Python probe remains bounded to its recorded checks and does not establish broad conformance.

## Models, Packages and files

| Model | Manifest | Version | Content |
| --- | --- | --- | --- |
| connector-example | [model.yaml](model.yaml) | Local, without a release number | Vocabulary and project: 24 Definitions. |
| markitect-core | [dependencies/core/model.yaml](dependencies/core/model.yaml) | Synthetic 1.0.0 | Core: nine Definitions. |

```text
model.yaml                              # connector-example; root: model
model/
  example/
    connector/
      vocabulary/                       # example/connector/vocabulary
        exchange.yaml
        ...
      project/                          # example/connector/project
        api.yaml
        ...
dependencies/
  core/                                 # explicitly provided local storage location
    model.yaml                          # markitect-core; version: "1.0.0"; root: model
    model/
      markitect/
        core/                           # markitect/core
          kind.yaml
          ...
```

In total there are **33 Definitions in three occupied Packages**, distributed across two models. The three intermediate folders under the respective model roots also form Packages. Model ID, installation folder and root do not automatically become a Package prefix. The model root itself is not a Package and must not directly contain any definition files. Every Definition lies in a named subfolder; that subfolder's non-empty relative path to the respective root is its Package name. Parent Packages are not inherited. A different storage location of the models preserves these names.

The [manifest contract from section 8.1](../language-contract.md#models) uses exactly model.yaml with formatVersion, id, optional local version, required relative root and required dependencies mapping. connector-example binds markitect-core exactly to "1.0.0". Its own version is deliberately omitted; as a published or consumed dependency model, it would require an exact version. The Core version is synthetic and does not claim a published library.

The verifier receives the entry manifest and the Core manifest explicitly as local inputs. dependencies/core/ is the storage location chosen for this vertical slice, not a general installation or lock convention. The verifier follows the direct model requirements, checks exact identity/version and loads the model roots reachable through them. There is no network resolution, version search or automatic installation. The [chosen lock format](../language-contract.md#lock) is now stored as [model-lock.yaml](model-lock.yaml). Structure, the external model set, versions and real canonical content digests are checked. [Section 8.3](../language-contract.md#canonical-bytes) determines model-digest/1; [canonical/core.json](canonical/core.json) stores the exact 3769 Core bytes.

A Package occupied with Definitions belongs to exactly one model; even different local IDs do not allow this Package to be split. Shared intermediate folders are possible: an additional model with markitect/http could own the same empty intermediate folder markitect. Across models, only occupied paths and their derivable intermediate paths bind names; purely empty subtrees have no effect on this. The counter-probes check this merging as well as collisions of bound names, including straße/strasse. The three Definition groups each lie under their own root:

| Package | Definition folder |
| --- | --- |
| markitect/core | [dependencies/core/model/markitect/core/](dependencies/core/model/markitect/core/) |
| example/connector/vocabulary | [model/example/connector/vocabulary/](model/example/connector/vocabulary/) |
| example/connector/project | [model/example/connector/project/](model/example/connector/project/) |

The verifier captures all YAML definition files under every model root. The explicitly bound model.yaml and its neighbouring model-lock.yaml are metadata and are not captured as Definitions, even with root: .; a counter-probe checks this adjacency and the rejection of an additional root Definition. The earlier package.yaml files, together with definitionFiles and versions per Package, are dropped. There is no additional mandatory definitions/ folder. The local id remains explicit in every Definition and does not follow the file name. Renaming a file within the same Package folder preserves the identity; a different Package folder changes it. Package names and full qualification use /, short alias qualification still uses dots. Local IDs and field names contain no dots; `Connector.Exchange` is an alias reference to the local ID `Exchange`.

Only imports in a definition file make foreign Packages visible. These must come from the file's own model or from a direct model dependency. Packages of the same model are not automatically visible to one another either; parent and child folders do not open any names. Transitive models are available for contract checking; direct access to their Packages needs a direct model dependency of its own and a file-local import. Aliases do not change the set of available Definitions.

The API uses kind: Connector.Container, the concrete Command kind: example/connector/vocabulary/Command, and list-documents still kind: Exchange. All forms resolve to the same canonical Package/ID pairs.

The report checks manifests, version equality and Package membership separately from the overall Definition graph, which is still invalid. The lock check now actually passes against the canonical Core content. Its SHA-256 digest is 562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1. For this, the Core is fully validated as a model of its own together with its dependency closure. No digest is output for connector-example because of Command/command; the passed external lock binding does not declare the overall Definition graph valid. Artificial comparison digests remain limited to explicitly named counter-probes.

The report additionally binds the file bytes read via SHA-256. These file hashes remain byte provenance and are to be distinguished from the separately computed canonical model digests. Empty folders appear only in the captured folder inventory. The common model ID/version label grammar is chosen: lowercase NFC Unicode segments according to XID, with Unicode decimal digits also at the start, separated by . or -. The probe checks this form in manifests, requirements and the lock, as well as the scopes of collisions and exact version equality. Definitions directly in root are rejected as language errors, without automatically moving files. The counter-probes check the exclusion of the root from the Package inventory, its lack of a Package identity and the unchanged source file on rejection. The physical root path rules are also chosen: . or an exactly spelled relative /-path with existing folders according to the Package folder grammar. Invalid path forms, wrong actual spellings, redirections and overlapping bound model roots are language errors. Both stored root: model already correspond to this form.

**Core:** [Kind](dependencies/core/model/markitect/core/kind.yaml), [Property](dependencies/core/model/markitect/core/property.yaml), [name](dependencies/core/model/markitect/core/name.yaml), [type](dependencies/core/model/markitect/core/type.yaml), [minCount](dependencies/core/model/markitect/core/min-count.yaml), [maxCount](dependencies/core/model/markitect/core/max-count.yaml), [values](dependencies/core/model/markitect/core/values.yaml), [target](dependencies/core/model/markitect/core/target.yaml) and the shared Property [properties](dependencies/core/model/markitect/core/properties.yaml).

**Exchange contract:** [Exchange](model/example/connector/vocabulary/exchange.yaml), [requester](model/example/connector/vocabulary/requester.yaml), [receiver](model/example/connector/vocabulary/receiver.yaml), [command](model/example/connector/vocabulary/command-property.yaml), [requestChannel](model/example/connector/vocabulary/request-channel.yaml), [replyChannel](model/example/connector/vocabulary/reply-channel.yaml), [timeoutSeconds](model/example/connector/vocabulary/timeout-seconds.yaml) and [maxAttempts](model/example/connector/vocabulary/max-attempts.yaml).

**Supplementary vocabulary:** [Container](model/example/connector/vocabulary/container.yaml) with [location](model/example/connector/vocabulary/location.yaml), [Command](model/example/connector/vocabulary/command-kind.yaml) with [operationId](model/example/connector/vocabulary/operation-id.yaml), [Channel](model/example/connector/vocabulary/channel.yaml) with [topicPattern](model/example/connector/vocabulary/topic-pattern.yaml), [delivery](model/example/connector/vocabulary/delivery.yaml) and [broker](model/example/connector/vocabulary/broker.yaml), as well as [MqttService](model/example/connector/vocabulary/mqtt-service.yaml).

**Project items:** [list-documents](model/example/connector/project/list-documents.yaml), [api](model/example/connector/project/api.yaml), [onprem](model/example/connector/project/onprem.yaml), [list-documents-command](model/example/connector/project/list-documents-command.yaml), [requests](model/example/connector/project/requests.yaml), [replies](model/example/connector/project/replies.yaml) and [messaging](model/example/connector/project/messaging.yaml).

Under the new collision-independent case rule, the IDs Command and command are the same name in the package and therefore an error. The different files command-kind.yaml and command-property.yaml do not resolve this Definition conflict. The remaining file names were explicitly switched to lowercase kebab-case; in doing so, the open domain decision was not replaced by an automatic renaming. An actual renaming is to be distinguished from the strict check: the verifier itself never changes the Definitions.

## Placement after the v1 consolidation

The [language specification v1](../language-contract.md) owns the rules. [Historical derivations](../language-history.md) and [procedure proposals that were not chosen](../procedure-proposals.md) are separate from it. The nine Core Definitions remain a normative part of the specification; the Definition, manifest and lock excerpts below are informative views of the example files.

The verifier reads sections via stable anchors instead of titles or numbers. It compares the normative Core self-description and the informative excerpts with their respective sources. This README is therefore now itself a hash-bound verification input. The [report from before the editorial consolidation](verification-before-v1-consolidation.json) is kept as a historical finding; [verification.json](verification.json) binds the newly checked state. The consolidation changes neither the nine Core YAML contents nor the canonical projection or the Core digest.

## Chosen corrections after the v1 review

The [reference resolution](../language-contract.md#names) allows full qualification even within its own Package without an import. Visible Definitions with the same name are permitted; every reference used must be unambiguous. Duplicate import aliases or duplicate Package imports remain declaration errors.

Only Packages with Definitions of their own are importable. Empty folders and intermediate Packages remain part of the folder structure, but do not open any Definition names. They remain excluded from the digest; intermediate paths of occupied Packages still take part in the cross-model collision check. [package_probes.py](package_probes.py) checks own and imported targets, ambiguous short/alias references, strict spelling, empty Packages and the [exhaustive file capture](../language-contract.md#packages). Every `.yaml` is read as a Definition; only the explicitly bound manifest/lock paths are metadata. `.yml` and extension variants are errors; other regular files are ignored. The counter-probes also run on actual temporary file trees and check unchanged source files.

[canonical_probes.py](canonical_probes.py) confirms identical bytes for short and fully qualified own structural references as well as the rejection of empty Package imports before the projection. The [report from before these corrections](verification-before-v1-package-fixes.json) is kept for historical reference; [verification.json](verification.json) is explicitly generated for the new state. The nine Core Definitions and their stored bytes remain unchanged. These three corrections do not change any of the smaller body clarifications; the naming simplification selected afterwards is in the following section.

## Name binding of empty folders

The [cross-model name binding](../language-contract.md#models) follows the same occupied Package paths that appear in the canonical content, including their derivable intermediate paths. Purely empty subtrees do not bind any names across model boundaries. Locally, all folders including empty subtrees are still checked strictly; they remain Packages and do not become importable.

[package_probes.py](package_probes.py) loads actual separate model roots with two peer libraries: occupied `straße/feature` and empty `strasse/empty` in different models must be valid together. Adding and removing the empty subtree preserves all canonical bytes, real digests and the existing lock binding. As soon as `strasse/empty` contains a Definition, the derived intermediate paths collide. The counter-probes also check the repeated logical validation, standalone model checking, candidate order, local spelling/collision errors and unchanged sources. The [report from before this clean-up](verification-before-empty-package-bindings.json) is kept; [verification.json](verification.json) binds the updated state. Core bytes and the external lock file remain unchanged.

## Non-empty body with Unicode White_Space

The [body rule](../language-contract.md#body) requires at least one character outside Unicode White_Space 15.1.0. The verifier and the link display text check use the same explicit character set from [name_profile.py](name_profile.py). Python `strip()` no longer determines the language rule. Empty strings and texts consisting only of White_Space are rejected; existing whitespace is not removed or rewritten. Ordinary string values and the optional title keep their previous rules.

The counter-probes check all 25 White_Space characters individually, mixed whitespace and characters outside this set, in particular U+001C to U+001F, which Python additionally treats as whitespace. Their structural admissibility does not lift the documented capability limit of the Markdown probe. [canonical_probes.py](canonical_probes.py) confirms unchanged leading and trailing whitespace characters in the body and different digests compared with the text without this whitespace. A body consisting only of White_Space is rejected before the projection. The [report from before this clarification](verification-before-body-whitespace.json) is kept; [verification.json](verification.json) binds the new state. The stored Definitions, Core bytes and the external lock file do not change. The preservation of NUL in link display text, chosen afterwards, is in the following section.

## Literal NUL in link display text

The [body link profile](../language-contract.md#body-links) preserves U+0000, as decoded from YAML, unchanged in display text. An actual U+FFFD remains distinct from it. [body_links.py](body_links.py) still uses CommonMark-normalized text to recognize protected contexts, but reads active body links from the corresponding literal inline text. This restoration belongs to the content evaluation and does not use optional diagnostic coordinates. If the bounded probe cannot restore the literal inline text, it reports a capability limit instead of outputting a replaced or invented display text.

The counter-probes check NUL alone and together with U+FFFD, repeated links, block quotes, headings, lists, body link escapes and the YAML NUL escapes including their source ranges. Code, autolinks, raw HTML blocks and ordinary reference links remain protected; CommonMark's equal treatment of NUL and U+FFFD in ordinary reference names also remains in effect. [canonical_probes.py](canonical_probes.py) confirms unchanged body values and different digests for NUL and U+FFFD. [diagnostic_probes.py](diagnostic_probes.py) checks the preservation with missing YAML metadata and with a forced failure of the optional coordinate mapping. The [report from before this correction](verification-before-literal-nul.json) is kept; [verification.json](verification.json) binds the new state. Core bytes and the lock remain unchanged. The Core equivalence of missing imports and `imports: []`, corrected afterwards, is in the following section.

## Missing and empty imports in the Core comparison

The comparison with the [nine fixed Core Definitions](../language-contract.md#core-definitions) takes into account the general equivalence of missing imports with `imports: []`. The author sources remain unchanged. Non-empty imports are checked completely; the following extension then compares the resolved contents.

[canonical_probes.py](canonical_probes.py) checks each of the nine Definitions individually with an empty import list and then all together, also in combination with explicitly set ID default names. The canonical bytes and the fixed Core digest must remain identical. Negative counter-probes prevent null, other value types or invalid non-empty import lists from disappearing through the comparison. The [report from before this correction](verification-before-core-empty-imports.json) is kept; [verification.json](verification.json) binds the new state. The stored Core files, their byte template and the external lock file remain unchanged.

## Core comparison by resolved content

The subsequent critical comparison had shown that fully qualified own Core references were rejected despite an identical target. This correction is now implemented: the comparison uses the ordinary name resolution for kind, target and properties and compares effective default values. Unordered declarations are sorted, but not deduplicated. Imports are fully checked before the comparison; after successful resolution, their written form and valid unused imports do not affect the content comparison. There is no second selection rule for the Core and no exception from strict spelling or the ambiguity check.

[core_probes.py](core_probes.py) checks each of the nine Definitions with a fully qualified kind, the field/target references, all alias references, renamed aliases, valid unused imports and combinations with explicit default names as well as a changed declaration order. These variants pass through the complete content check and must yield the same Core bytes; no check step is suspended for this. Counter-probes check real contract, target, cardinality and text changes, forbidden fields, missing Definitions, faulty imports, spelling errors, ambiguities and duplicates. A target with the same name in a different Package remains distinct. All comparisons leave the author sources unchanged.

The [report from before this correction](verification-before-semantic-core.json) is kept; [verification.json](verification.json) binds the new state, including the new counter-probes. The stored Core Definitions, the canonical byte template and the external lock file remain unchanged.

## Local IDs and field names without dots

The [v1 naming rules](../language-contract.md#spelling) forbid dots in all local Definition IDs and effective field names, including `spec.name` and nested object contracts. The stored Exchange files already use short IDs and field names without dots and need no renaming. `Core.Kind` and `Connector.Exchange` respectively remain alias references; Package names and full qualification keep `/`.

| Item | Author form |
| --- | --- |
| Local Kind ID | `Exchange` |
| Local Property ID and default field name | `requester` |
| Direct reference to this Kind | `Exchange` |
| Reference with the declared alias Connector | `Connector.Exchange` |
| Full reference to this Kind | `example/connector/vocabulary/Exchange` |

Segmented IDs and field names are rejected, even if they are unused or a Property sets an explicitly different field name. There is no automatic migration. The name, Package and canonical counter-probes check this rejection and, as before, Unicode, alias resolution, decimal numbers, exact version labels such as `"1.0"` and unchanged dots in text values. The [report from before the naming simplification](verification-before-dotless-names.json) is kept; [verification.json](verification.json) binds the new state. The nine Core Definitions, their canonical bytes and the external lock binding remain unchanged. Exact diagnostic coordinates are now governed by the separate tool contract that follows. The treatment of `x` as an ordinary version character, chosen afterwards, is in the following section.

## Diagnostic coordinates as an independent tool profile

The [language specification](../language-contract.md#body-links) requires complete link recognition, target resolution and occurrence order. The separate [diagnostic contract](../diagnostic-contract.md) stipulates exact UTF-8/YAML coordinates. [body_links.py](body_links.py) reads only contents by default; [diagnostic_coordinates.py](diagnostic_coordinates.py) optionally adds positions that can be evidenced. The canonical projection does not invoke any coordinate mapping.

The report separates `body_link_validation` and `canonical_models` from `diagnostic_coordinates`. Missing or outdated YAML metadata, a Markdown mapping that cannot be evidenced, or pure coordinate aborts yield `not_completed` there. Partial coordinates that can already be evidenced are kept; missing ones are not invented. This separate status changes neither a valid content verdict nor its digest. Syntax/target errors and an incomplete content check still block the canonical output.

[diagnostic_probes.py](diagnostic_probes.py) checks this separation with missing and outdated provenance, forced inline/YAML mapping errors and memory aborts exclusively in the coordinate phase. The same checked contents must yield the same canonical bytes; order and repetitions are preserved. The previous exact source range probes are continued as tool profile tests. The [report from before the separation](verification-before-diagnostic-separation.json) is kept; [verification.json](verification.json) binds the newly checked state, including the diagnostic contract.

## Version labels with ordinary x

The [v1 label rules](../language-contract.md#models) no longer reserve `x`. `"x"`, `"1.x"` and `"release.x"` are bound exactly as text in manifest, dependency requirement, lock and canonical bytes. `"1.x"` does not select numeric versions such as `"1.7"`. `latest`, wildcards and range operators remain forbidden; Unicode, NFC, spelling and collision rules apply unchanged.

The manifest, lock and canonical counter-probes check successful exact binding, non-matching numeric versions and unchanged label texts in the canonical bytes. The [report from before this selection](verification-before-literal-x.json) is kept for historical reference; [verification.json](verification.json) binds the newly checked state. The stored model versions and the Core digest do not change as a result.

## Verification and limits of the claims

From the repository root directory with Python 3.13 (Unicode 15.1.0) in an isolated verification environment. The [verification dependencies](requirements.txt) concern only this design probe. Once, under PowerShell:

```powershell
$probeEnvironment = Join-Path ([System.IO.Path]::GetTempPath()) 'markitect-canonical-model-probe-py313'
python -m venv $probeEnvironment
$probePython = Join-Path $probeEnvironment 'Scripts/python.exe'
& $probePython -m pip install -r docs/design/canonical-model/minimal-language/exchange-example/requirements.txt
```

The Unicode table version and the CommonMark parser version are bound. The environment lies outside the checkout so that its library files do not become part of the project coverage. Under Linux/macOS, an environment is correspondingly created outside the checkout and its bin/python is used. The following PowerShell commands use $probePython from the setup block. The regular verification run under PowerShell:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py
```

The normal run is not successful because of the open Definition conflict. The Core digest and the lock check pass separately. A successful --self-test confirms the marked counter-probes and the stored canonical Core bytes, but does not declare the faulty Connector model conformant. A report on the current source state can be written explicitly; an error remains an error and leads to a non-zero exit code:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py --report docs/design/canonical-model/minimal-language/exchange-example/verification.json
```

The verifier's counter-probes can be run separately from the still faulty example:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py --self-test
```

This test mode explicitly uses synthetic valid and invalid inputs. Its success does not declare the stored example conformant and does not constitute a model decision about Command. The sources are not corrected in any mode. The body link check is run as a separate part of the report; it can pass for the existing links while the overall model is rejected because of the Definition conflict. The number resolution_calls in the report counts the verifier's resolution calls, not distinct graph edges.

The verification covers model-lock.yaml, the complete direct/transitive model set, version equality, synthetic digest deviations and unchanged sources, as well as model.yaml, direct/transitive version bindings, access via direct model dependencies, shared intermediate folders and unique membership of occupied Packages, as well as the /-form for Packages and full qualification, the rejection of old dot forms and path-like evasive spellings, the agreement of the normative Core with the sources and of the informative YAML excerpts of this example page with their files, as well as folder-based Package capture, moving/renaming, missing implicit parent imports, Unicode folder collisions as well as file-local imports and aliases, exact canonical spelling, collisions independent of case, Kind/Property/value references, closed field contracts, value types, cardinalities and effective field names. For names, the Unicode 15.1.0 profile from section 7.2 of the language contract applies: XID identifiers, strict NFC author form and complete culture-independent name comparison. straße and strasse collide in all namespaces; deviating references are still rejected. Normal text values remain unchanged. The counter-probes also check names without case, combining marks, Unicode import aliases and Unicode file names. PyYAML only parses the YAML syntax. The probe's own scalar rules according to section 5.1 read only true/false as booleans, keep dates as text and use arbitrary-precision integers and exact finite decimal numbers without type conversion. Anchors, aliases, merge keys, explicit tags, document sequences, null and non-textual keys are rejected. Resource limits of the number probe are explicitly named in the report; they are not language-level number ranges.

The list counter-probes compare ordinary value lists while preserving their order and repetitions, but compare Core declarations and imports without regard to order. Missing and empty optional lists yield the same comparison value, also within objects. Ordinary data fields with the same name receive no meta meaning. Duplicate and cardinality errors are rejected before the comparison. The earlier list comparison probe is kept as a counter-probe of its own. The complete canonical encoding model-digest/1 now additionally has fixed byte vectors and equality/difference probes.

The two active body links in list-documents reference api and onprem. The [body link parser](body_links.py) uses CommonMark 0.31.2 with the chosen link extension: code, ordinary links/images and raw HTML ranges remain protected; active links are resolved strictly. The content check keeps all ordered link occurrences without a coordinate obligation. The separate diagnostic report adds UTF-8 byte ranges and YAML source positions that can be evidenced, according to the [tool contract](../diagnostic-contract.md). The counter-probes also check YAML escapes, folding, Unicode, faulty syntax and repeated links. The limits of the probe are 2000000 UTF-8 bytes per body and 64 nesting levels. A known whitespace deviation of the bound parser version concerns U+001C to U+001F as well as U+0085; the probe explicitly reports such an otherwise possible evaluation as not completed. This is not an additional language restriction. An abort is explicitly not a successful or complete model check.

The [verifier](verify.py) is a bounded structural probe for this explicit input set. The [current report](verification.json) names its inputs, file hashes, errors and limits. The [previous passed report](verification-before-strict-names.json) remains unchanged as a historical finding from before the new naming rule; it does not evidence today's source state. A later run changes an old report only with the explicit --report call; its new input hashes must then be assessed anew. The verifier replaces neither the product compiler nor the complete language specification.

The selection is limited to the Exchange example. Container here has only the location; additional component and interface models are not presupposed. Channel is the MQTT vocabulary of this project and not a universal Core Kind. The contents of the Commands, topic parameters, authorization, runtime errors and desired delivery effects are partly stated as binding prose in the body. A successful structural check does not confirm their implementation or their freedom from contradiction.

A general YAML/Markdown conformance proof, a production Go implementation of the new canonical encoding, a semantic index, implementation, independent semantic verification and broker/provider runtimes are not a result of this probe. Neither network services nor model providers are called.

**Version labels:** Versions are bound as exact text, without numeric comparison, padding of components, ordering or compatibility assumptions. The counter-probes check a matching two-part version as well as the rejection of "1.0" and "1.0.1" respectively for a requirement of "1.0.0", also in the lock. The canonical bytes also distinguish "1.0" and "1.0.0". Version ranges and latest remain forbidden. The common character grammar from section 8.1 is chosen; the counter-probes also check digit starts, dots, different scripts, combining characters, NFC, invalid separators and version collisions such as straße/strasse within the same model.

**Physical root paths:** [root_paths.py](root_paths.py) checks the author form before any path interpretation, actually existing names and directories, forbidden redirections and non-overlapping loaded roots. [root_path_probes.py](root_path_probes.py) checks this on the file system: nested and Unicode paths, unchanged source files, wrong case and NFC, straße/strasse collisions, missing folders, files instead of folders, impermissible path normalization and nested models. A real junction under Windows, or a symlink on other supported hosts, is rejected both as a root path and within the model tree. The absolute manifest storage locations remain free; the relative root spelling follows the Package folder grammar and does not extend it with the characters of the model ID/version label grammar.

**Object cycles and resources:** [Section 5.3](../language-contract.md#cycles-resources) permits recursive object contracts, provided the graph of mandatory object fields is acyclic. The value checker follows actually present finite objects and does not change any Definitions. [object_resource_probes.py](object_resource_probes.py) checks breakable self/multi-node cycles, impossible mandatory cycles, finite values and empty lists, closed fields, reference cycles and a long contract graph without recursive up-front expansion.

[resources.py](resources.py) separates tool aborts from language errors. --max-object-depth configures the maximum actually visited object depth of this probe (default 128); value checking starts at depth 0 outside an object value. The report also names the Python recursion limit as well as the existing body link and number budgets. An abort yields not_completed and, for the affected scope, neither bytes nor a digest; existing export files are kept. Successful runs with sufficiently large budgets yield the same canonical bytes. The normal CLI exit codes are 0 for success, 1 for proven errors and 2 for an uncompleted check; --self-test separately assesses only the marked counter-probes.

The [final reconciliation](../language-contract.md#conformance) now also contains the chosen acyclicity rule for model dependencies. Self, mutual and longer model cycles are language errors (failed) and receive no digest; the earlier provisional classification as not_completed is superseded. Valid chains and shared libraries are checked separately, as is the rejection when loading physical manifests and before the canonical projection. Within the chosen minimal scope, no further open language decision has been identified. Go implementation, general parser conformance and semantic procedures are separate from this.

## Canonical bytes and real content binding

[canonical_bytes.py](canonical_bytes.py) implements the projection chosen in section 8.3: own model metadata without root, own occupied Packages/Definitions, resolved references and body link targets, effective defaults and exactly preserved text values. It uses fixed-sorted UTF-8 JSON without BOM or final line break, and typed exact numbers. Empty Packages, author formatting, source positions, lock files and foreign contents are not encoded. Every model version and its dependency closure are checked before the projection; load errors are kept per model.

[canonical_probes.py](canonical_probes.py) checks equality across quote, comment and key order, valid structural reference forms, defaults and declaration orders, as well as differences in text, body link targets, ordinary list order, types and exact numbers. The [Core byte vector](canonical/core.json) is stored as a fixed artifact; an independent standard JSON encoder is compared against the projected tree. Invalid inputs receive no digest. None of these probes replaces the Go product compiler or a general parser conformance proof.

The explicit output of a valid single model version is additionally possible:

```text
& $probePython -B docs/design/canonical-model/minimal-language/exchange-example/verify.py --canonical-model markitect-core --canonical-output docs/design/canonical-model/minimal-language/exchange-example/canonical/core.json
```

This export checks only the chosen model and its dependency closure. It does not automatically update the lock and does not confirm the other, faulty Connector model. Normal verification runs write neither sources, canonical artifacts nor locks.

## Is the canonical form therefore complete?

Before the new naming rule, the Definition and reference graph of this example was structurally closed. The present Command/command conflict is explicitly rejected. This question is to be resolved in the project vocabulary; both chosen modelling directions are in principle expressible with the Core. The example does not evidence completeness of the language for every project. The [v1 specification and its limits of the claims](../language-contract.md#conformance) define the chosen minimal language scope; production compiler/report contracts and evidence remain separate work. The generic connection to realization and Apply is a further procedural task.


<a id="definition-excerpts"></a>

## Informative Definition excerpts

These excerpts mirror the files of the same name in the executable example. Only the [v1 specification](../language-contract.md) is authoritative for language rules. The last requester excerpt shows the same ID default without an additional name entry; it is not a second Definition. The verifier compares the excerpts with the files.

**exchange.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: Exchange
kind: Kind
body: >-
  Describes a Request/Reply contract between requester and receiver. Replies keep
  the MessageId of the Request. timeoutSeconds applies per attempt; maxAttempts includes the first
  attempt.
spec:
  properties:
  - requester
  - receiver
  - command
  - requestChannel
  - replyChannel
  - timeoutSeconds
  - maxAttempts
```

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Application in the requester role of this exchange.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**receiver.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: receiver
kind: Property
body: >-
  Application in the handler role of this exchange.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**command-property.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: command
kind: Property
body: Contract of the requested operation.
spec:
  type: reference
  target: Command
  minCount: 1
  maxCount: 1
```

**request-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requestChannel
kind: Property
body: >-
  Communication path for requests.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**reply-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: replyChannel
kind: Property
body: >-
  Communication path for replies to the issuing instance.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**timeout-seconds.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: timeoutSeconds
kind: Property
body: >-
  Positive maximum waiting period per attempt in seconds.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**max-attempts.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: maxAttempts
kind: Property
body: >-
  Positive attempt budget including the first attempt.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Application in the requester role of this exchange.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

<a id="manifest-excerpts"></a>

## Informative manifest and lock excerpts

The version of the Core example is synthetic; the digest binds its actually checked canonical bytes.

**model.yaml**

```yaml
formatVersion: 1
id: connector-example
root: model
dependencies:
  markitect-core: "1.0.0"
```

**dependencies/core/model.yaml**

```yaml
formatVersion: 1
id: markitect-core
version: "1.0.0"
root: model
dependencies: {}
```

**model-lock.yaml**

```yaml
formatVersion: 1
models:
  markitect-core:
    version: "1.0.0"
    digest: "sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1"
```
