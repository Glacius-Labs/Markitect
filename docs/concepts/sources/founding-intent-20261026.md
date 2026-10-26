# Founding intent: a typed canonical model

Date: 26 October 2026. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The session had reviewed the comparison evidence and measured the Makerspace study model and Markitect's self-model against draft quality criteria. The owner statements below are verbatim, typos included. Assistant questions they answer are quoted and labeled. The recorder's notes are labeled and add no decision. This is provenance, not a change to concept 1.0 or to any register entry.

Earlier owner accounts of the same origin: [owner origin account](owner-origin-20261015.md) (15 October), [foundations and trust](foundations-and-trust-20261017.md) and [conceptual influences](conceptual-influences-20261017.md) (17 October). The owner noted that the story has been told many times and is still not visible to agents working in the repository; the last statement below asks that every agent knows it.

## Owner statements (verbatim)

### 1. Programming in documentation

Context: the assistant had proposed a revised list of eight model quality criteria, split into hard rules and signals.

> This list is at least better than no criteria at all. In the end, what one could argue here is that we are basically programming in documentation or yaml. So the same signals should in theory apply here. For example dependencies, mutation testing, crap, clean architecture. Also of course large file sizes are a bad signal, since definitions should be pretty simple and direct. If we have 500 words in a definition its basically just prosa compensating for missing definitions.

### 2. Projects define their own Kinds

Context: the assistant had said that the shipped vocabulary has no Kind with typed fields for operations and that one Package costs one Manager.

> Wait, we already planned on providing like some sort of importable packages or modules to enrich the language. But even just with the core language, new kinds can just be defined. We dont rely on having everything be defined already. We have full freedom to do so ourselves. And yes, the code analogy should be the basis here and i agree with what you said. Also we dont care if we have many agents, not each definition is its own package, just as not each source file or class oder function is its own namespace in code. We do the canonical model so that we can more or less precisely model and program in prosa, but in a typed way.

### 3. Libraries and the compiler's job

Assistant questions (quoted): "Who defines Kinds?" and "Can the compiler already check what point 1 needs?", where point 1 was compiler checks such as an undeclared refusal code or an operation without a permitted role.

> > Who defines Kinds?
>
> Of course both. How could we possibly provide all the language that each project could need? So of course each project should be able to add kinds and therefore form its own language. Additionaly they can import modules we offer, with common kinds for different things, so not every project has to reinvent the wheel. Just like in Programming and Libraries.
>
> > Can the compiler already check what point 1 needs?
>
> The compiler should only check for structural correctness. Just like a regular static compiler in a typed programming language. It should not check meaning, this is the job of ai agents and the management hierarchy. But it should basically already know everything it needs from just the core.

### 4. How the canonical model was intended

Context: before the assistant's scratch-model check had run.

> Lets first discuss further about the canonical model and how it was intended to be. For the language modules (we had a name for that but i cant recall what it was) we should orient on what libraries languages like go or for example C# provide. Kind are basically just like classes or data types or maybe even functions or something like that. The body is more or less like the function bodies in code. The props are like parameters or properties of types. More or less this was the analogy. It came into mind, because i had fought a hard battle with keeping a project with classic markdown documentation, rules, skills, agents etc consistent with the code. Each consistency check run took hours and each one of them found more and more inconsistencies. At some points i wasnt sure, if the project, which was basically developed completely using agentic coding, really follows all the rules and what rules are there even? What is defined, what isnt, what came silently into the project and so on. So basically i lost the overview and confidence in the project and what we have here. That reminded me on my preference for strongly typed languages. There we have a compiler, dependencies and the compiler greatly helps, because it can tell me about issues during compile time. When just writing markdown who knows, if all the relevant documentation was really updated and how do i find out? So i decided to create a typed canonical model and invented the markitect apply agent organization based on conways law, clean architecture, clean code and modularity. Thats the whole idea

### 5. What follows

Context: answers the assistant's analogy table (Kind, Property, Definition, body, Package, imports, manifest and lock as `go.mod`/`go.sum`, Managers per Package as teams per module, Apply as build plus CI) and its first sketch of standard Libraries.

> Yes you got it. So basically we should provide a rich standard library, a clean core and since people wont write the yaml or anything like that we need skills or something like that, so ai agents write the canonical model correctly and use the tools Markitect provides correctly. That means we first need to ensure the canonical model and how it works can serve the purpose. We also oriented a bit on kubernetes and terraform here. I dont know how often i told this story already and why its not in the project and not clear that those are the things that markitect wants to do. We should definately ensure each agent that works in the repo knows that. Also we might need to work on the canonical model, see if it fits what i intended and what we need and that we provide comfortable tooling and skills to model effectively and cleanly. This also means we need to somehow define how a good model should look and how to spot bad practices. Also we should somehow define how a project is described well via the canonical model and what should be modeled so that we can talk about a good canonical model. Then we also need to define the standard library. Our evaluations of course are not good if the canonical model is bad by itself

### 6. Documentation request

> Yes: draft the owner source from my statements in this conversation, a short founding-intent section for the vision brief and a pointer in AGENTS.md, and coordinate the shared files with the Coordinator before opening a PR.

## Recorder's notes

Interpretation by the recorder, not owner wording:

- **The analogy.** Kind ≈ type, class or function signature; Property ≈ field or parameter; Definition ≈ declaration; body ≈ function body; Package ≈ namespace or module; imports ≈ imports; model manifest and lock ≈ `go.mod` and `go.sum`; the compiler ≈ a static type checker that checks structure, not meaning. The Apply organization follows Conway's law: Managers per Package, like teams per module.
- **"Modules" are Libraries.** The language specification calls importable models that supply reusable definitions *Libraries* ([language contract §1](../../design/language-contract.md#scope)); `software-basics` is the one example so far. The owner wants a rich standard set of them, oriented on what Go or C# ship, and project-defined Kinds for everything else.
- **Who writes the model.** People decide; AI agents write the YAML through skills and Markitect's tooling.
- **Kubernetes and Terraform** are named as orientation without detail. Recorder's reading: a declared desired state in typed resources (Kubernetes resources also carry `kind` and `spec`) that tooling plans, applies and checks for drift. The kubectl/terraform command style is already recorded in the register.
- **Consequences the owner names:** check that the model and its mechanics serve this purpose; define what a well-modeled project contains, what a good model looks like and how to spot bad practices; define the standard library; provide comfortable tooling and skills. Evaluations depend on a good model.
- **Existing entries that may read against this intent**, listed for the owner without a decision:
  - [DEC-003](../register.md#dec-003-the-compiler-is-a-borrowed-concept) calls the compiler a borrowed concept. It addresses guarantees about realization, but can be read as downplaying the typed model.
  - [V1 concept §2.2](../v1-concept.md#22-not-commissioned) excludes interactive model authoring from V1, while the owner calls modeling skills and tooling necessary.
  - The language contract says that a body link alone declares no dependency, while the analogy treats references as dependencies.
  - The Makerspace study model and the self-model use generic Kinds with prose bodies and no project Kinds; the self-model has no links. The session's measurements are in its draft quality criteria, which are not part of this record.
