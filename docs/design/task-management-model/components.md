# Components

Components name responsibilities, not compulsory classes, teams or permanent agents. Their explicit composition and dependency direction is owned by [Relationships](relations.md).

<a id="comp-app"></a>
## COMP-APP — Task application

Type: Component. Root of the example.

Compose the three responsibility areas into one local executable. Own application-wide setup, developer tooling and repository delivery. The executable's public name and invocation contract are defined by the CLI Component. No separate server process or shared runtime is needed.

<a id="comp-cli"></a>
## COMP-CLI — Command-line interaction

Type: Component.

Parse invocation and validate command/argument syntax. Resolve the selected task file once, call the offered task action, and translate its outcome through REQ-ERRORS and the UseCase's public messages. Formatting and file-selection syntax belong here; this component does not directly mutate the task collection or decode stored records.

The complete syntax is `tasks [--file PATH] COMMAND [ARGS]`. `--file` may appear once, before the command, and takes one nonempty path argument. Without it, use `tasks.data` in the process working directory. An absolute path is used directly; a relative path is resolved against that same working directory. Repeated options, other options and extra arguments are syntax failures. The three command contracts are owned by their UseCases. Version 1 does not promise a help command or localization.

<a id="comp-management"></a>
## COMP-MANAGEMENT — Task actions

Type: Component.

Implement the three offered UseCase contracts using the Task Concept and the selected storage capability. Preserve responsibility for input meaning, reference allocation, changes to task state and the no-write behavior of reads or repeated completion. Expose meaningful outcomes to the CLI rather than printing there. This boundary does not prescribe an object-oriented service, interface framework or dependency-injection container.

<a id="comp-storage"></a>
## COMP-STORAGE — Local task storage

Type: Component.

Read and save the selected task collection, detect invalid stored data and fulfill the durability and data-protection requirements. Keep encoding and filesystem mechanisms here. Do not determine task-action policy or user-facing messages. An absent selected file is distinguishable from a failed read. File representation is deliberately not fixed by this specification; see the relevant Decision.
