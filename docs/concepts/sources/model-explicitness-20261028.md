# Owner statements: what Markitect is, and how explicit the model is

Date: 28 October 2026 (UTC). Source: direct owner messages in the Claude Code Topics session, written in English. The owner statements are verbatim; the session's proposals they answer are summarized and labeled. The recorded meaning is the recorder's and adds no decision.

They open the Topics session's walk through Markitect's design ([topics page](../../workstreams/topics.md)) and lead to [DEC-079](../register.md#dec-079-the-model-is-as-explicit-as-good-documentation).

## 1. What Markitect is, restated

> Okay lets start by me explaining once more what markitect is and what it is trying to achieve. When i work on a project using AI agents i start by designing how it should work, what problem it solves, which technologies we use, usually with some sort of backend and so on. Then i start organizing and creating modules and some sort of base architecture by splitting the various parts into those modules. After that i start designing how it would be implemented and stuff like that. While designing and discussing there come a lot of markdown files as documentation, code, tests, CI CD files, infrastructure and so on. The documentation often gets messy pretty quick so i start defining rules and things like that. The design changes, implementation proceeds and at some point we get a lot of inconsistencies. Old files are forgotten or overlooked and start contradicting each other. Consistency runs dont know whats correct and what isnt, i loose overview about it myself and the project gets messy. What i want to do with markitect is to create a canonical model, which can easily get checked for consistency and be realized so that the repository and project stays clean and consistent and things dont get overlooked and rules actually arent violated and checked. To apply the model we stole the idea from management hierarchies of organisations and companies. The idea is to create responsibilities and different areas of abstraction for agents. So the root manager has quite broad knowledge and not so much deep knowledge. And the deeper we get, the deeper the knowledge and the less the agent must know about the surroundings. So its about having many agents, looking from different perspectives and communicating just as in an organization. This is the main picture of it. If you agree and can see that from the documentation too, we can proceed

Context (summary by the recorder): the session confirmed that the [founding intent](founding-intent-20261026.md) and the [vision](../../vision.md#markitect-in-brief) say the same, including "divide and conquer; abstraction and need to know" and "four eyes and different perspectives" in the vision's foundations, and noted that the comparisons so far have not yet shown the hierarchy paying for itself.

> So now the first problem is how to consistently and precisely model in a way, we can be explicit in expression, we can easily check and we can hold the model and the realization consistent. Thats where the analogy with strongly typed programming languages comes in. If we type the model, we can create a compiler which can structurally check and see what relates to what and when we change the model, we know what to check. Thats why we designed a canonical model and how it works just as we would program in prose and with it comes a compiler which is basically a command line tool. Now from here we have some topics: The canonical model and the compiler for example.

## 2. The language and its file format

> Yes lets start with the language. The language is inspired by programming languages such as Go and tools like kubernetes and terraform. The idea is to basically program in markdown or yaml. So the canonical model can be seen as some sort of meta programming language for documentation or modelling.

Context (summary by the recorder): the session noted that a Definition is a YAML file only, with Markdown inside its `body`, and recommended keeping that one format: the typed `spec` needs structure, one parser keeps the compiler simple, and `markitect docs` renders a readable view.

> Yes i agree with you. Lets talk about explicitness

## 3. How explicit the model is

Context (summary by the recorder): the session listed the open expressiveness gaps (maps, unions, defaults, uniqueness, formats, ranges, constraints across fields), noted that all of them describe the program's data rather than the language, and recommended adding them to the Libraries on first real need.

> We should be cautious here. We dont want to create a completely new programming language i think. Lets see what we are trying to achieve. We want to describe the project in a consistent model. We dont need full explicitness so that we actually code in markitect. If we introduce maps and model as explicit as code would, we might not gain much from that.

Context (summary by the recorder): the session proposed typing only what creates a relation the compiler must see or a contract at a boundary, and leaving everything else as prose.

> We can ask a simple question i think. Would a good documentation be that explicit? If yes we should support it, if not its too detailed. For example a foreach loop or a map is not mentioned in documentation. What is mentioned are endpoints, use cases, user interfaces such as cli or rest interfaces and so on. Also maybe definitions about what a book is and so on. The body and coding is basically just prose and up to the agent.

Context (summary by the recorder): the session proposed adopting this test as the rule for the language, noted that it confirms most of today's Libraries and shows two gaps that good documentation always covers (use cases, and how an interface is reached: REST routes, CLI commands), and asked to record it.

> Yes

## Recorded meaning

- **The test.** A statement belongs in the model, typed or as a Definition, when good documentation of the project would state it that explicitly. What good documentation would not state, such as loops, maps, internal data structures or algorithms, is too detailed and is left to the agents that realize the model. Bodies are prose.
- **One file format.** A Definition stays a YAML file with a Markdown body.
- **Consequences for the open language question.** Maps, unions, defaults and typed formats are not added. Uniqueness beyond an Entity's `key` and constraints across fields stay prose in Rules or Invariants. Ranges stay as `Values.Scalar` has them.
- **Not decided here.** Whether use cases and interface bindings (REST routes, CLI commands) get Library Kinds is a separate question that the test raised.
