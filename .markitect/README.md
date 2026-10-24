# Markitect self-model

[model.yaml](model.yaml) selects this repository's canonical language v1 tree under `model/`; [model-lock.yaml](model-lock.yaml) records explicit dependencies. This self-model is self-contained. Its nine fixed Core Definitions are byte-identical copies of the normative Core fixture under `testdata/core/model/markitect/core/`. Ordinary Rule and Check Definitions state responsibilities and maintenance obligations; there is no canonical Manager DTO.

[execution.yaml](execution.yaml) declares finite settings and named profile selections. It does not provide Runtime executable metadata, grant runtime authority or make Apply executable. [ignore.yaml](ignore.yaml) lists exact explained file exceptions and is empty by default.

Each `registry/<package>/registry.yaml` assigns exact artifact paths to its local Package responsibility and actual Definition links. Canonical, execution, registry and other captured control inputs are protected separately. Gitignore does not exempt files from the census. New paths require an explicit accounting decision.

[Module configurations](modules/) pin exact hook/workflow bytes and selected pipeline scalars to literal commands in canonical Check Definitions. Their verifier reads actual preparation and registry facts without executing hooks or CI.

Use `go run ./cmd/markitect check --repo .`, then `context <package> --repo .` for the relevant responsibility. Generate [the readable view](../docs/markitect/project.md) only through the reviewed `docs` preview/write workflow in [AGENTS.md](../AGENTS.md).
