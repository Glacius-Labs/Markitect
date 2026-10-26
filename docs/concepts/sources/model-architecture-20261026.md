# Owner decision: architecture in the canonical model

Date: 26 October 2026. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read them in that session's transcript. The owner statements are verbatim. The session's explanations they answer are summarized and labeled; the recorded meaning is the recorder's and adds no decision.

The statements belong to the [canonical model track](../../design/model-track-charter.md) ([DEC-074](../register.md#dec-074-canonical-model-track-parallel-to-v1)).

## 1. The project owner decides the architecture

Context (summary by the recorder): the session had asked whether new study projects should come with owner-decided architecture. Asked what that meant, it explained that the requirements author, acting as owner, writes down structural decisions as well as behavior. For Makerspace that would be a domain core, one module per area and a protocol adapter; a core that never imports protocol or I/O code; authorization and audit checked once, centrally; areas that talk only through named interfaces; and test and documentation obligations. Both arms get the same decisions, and hidden checks test them. Today the Makerspace requirements leave "internal package layout and design" to the implementer, so architecture drift cannot be measured.

Owner question (verbatim):

> What do you mean with owner decided architecture?

Owner reply after the explanation (verbatim):

> Of course the project owner should decide the architecture and encode it in the canonical model

## 2. One architecture, not two

Owner question (verbatim):

> Wait a second. Lets discuss this again. Maybe i was a bit too quick on this one. So if we have packages and we write architecture into the canonical model, but the canonical model itself should be like coding and therefore has an architecture by itself, wouldnt that mean that we have two architectures?

Context (summary by the recorder): the session agreed that describing an architecture in Definitions beside the Package tree, for example as Component items or a component list, would create two structures to keep consistent. It proposed the refinement that the architecture is encoded as the model's own structure:
- the Package tree is the module structure;
- the ownership areas in `execution.yaml` place each Package's code;
- dependency rules on Packages, which the model's own references obey too;
- Interface Definitions as a Package's public surface;
- placement and technical decisions that the tree cannot show, for example that authorization is realized once, in one Package;
- layers within an area become child Packages.

It proposed to drop a separate Component Kind and component list from the modeling guide, and asked: "Should I record this refinement with the Coordinator and the Evaluation session, and adjust the guide?"

Owner reply (verbatim):

> Yes do that

## Recorded meaning

Recorder's interpretation:

- **The first decision stands, refined by the second.** The project owner decides the architecture, and a complete canonical model encodes it. Leaving architecture open is legitimate only where the owner explicitly chooses it.
- **"Encode" means as the model's own structure.** The architecture is the Package tree with its ownership areas, Package dependency rules, Interface Definitions and placement decisions. There is no second, descriptive architecture beside it. The list is the session's proposal, which the owner approved with "Yes do that", not owner wording.
- **Evaluation consequence**, as the reason for the question: study projects get owner-decided architecture in both arms, and structural checks can test it. The study design belongs to the Evaluation session.
