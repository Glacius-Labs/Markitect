"""Apply trial cases and their answer key.

OPERATOR MATERIAL. This file names every deliberate fault and the expected
Apply result. It must never enter a target repository or an Agent's input;
`materialize.py` builds packets without it and audits them for leaks.

A case starts from one complete model snapshot and either no realization
(model-only), the known-correct reference realization, or the actual
committed result of an earlier Apply (`--from-repo`). It may change the model
by exact text edits and inject artifact faults by exact text edits. Every
edit names the exact number of occurrences it replaces, so a fixture that
drifts from its edits fails loudly instead of silently changing the case.
"""
from dataclasses import dataclass, field

# Expected Apply outcomes.
APPLIED = "verified-applied"  # Verified, publication applied, oracle profile passes
UNCHANGED = "verified-unchanged"  # Verified, publication no_change, nothing written
DEFECT = "model-defect"  # Root ends the run over a model contradiction, nothing published
GAP = "model-gap"  # Root ends the run or asks the user about a boundary gap, nothing published
REFUSED = "refused"  # the start stand fails a strict check before any Agent work


@dataclass(frozen=True)
class Edit:
    path: str  # packet-relative path
    old: str
    new: str
    count: int = 1  # exact number of occurrences of `old` that are replaced


@dataclass(frozen=True)
class Fault:
    id: str
    owner: str  # the Package (Manager) whose area holds the faulty artifacts
    kind: str  # behavior | test | doc | config | accounting | duplication
    summary: str
    edits: tuple
    detected_by: tuple = ()  # oracle checks that fail while this fault alone is present (exact set)
    static: tuple = ()  # assess.py artifact checks that fail while this fault alone is present (exact set)
    test_signal: bool = False  # the start's own test suite fails while this fault alone is present
    regression_test: str = "informative"  # "required": the repaired tests must fail when the code fault returns


@dataclass(frozen=True)
class Case:
    id: str
    project: str
    level: int
    title: str
    question: str
    snapshot: str
    start: str  # model-only | reference | previous
    expected: str
    profile: str = ""  # oracle profile of the expected result ("" when nothing is published)
    model_edits: tuple = ()
    faults: tuple = ()
    defect_terms: tuple = ()  # for DEFECT: groups of alternatives; the report should name one term of each group
    allowed: tuple = ()  # globs of changes that are not counted as unnecessary
    static: tuple = ()  # assess.py artifact checks of the result
    measure: tuple = ()  # what the evaluator records beyond the automatic checks
    notes: str = ""
    prior_edits: tuple = ()  # model edits of an earlier committed model version that the case's model corrects
    stand_edits: tuple = ()  # edits that make the reference conforming but unattractive, applied before the faults
    preserve: tuple = ()  # paths of the stand's tempting spots, which conformance requires to stay unchanged


# -- stockroom: model variants -------------------------------------------------------------

SR = ".markitect/model/stockroom/"

GAP_EDITS = (
    Edit(SR + "issue.yaml",
         "  At [[interface]], `issue --sku SKU --qty N` subtracts N from the on-hand\n"
         "  quantity of the existing [[item]] with that normalized SKU, commits the change\n"
         "  through [[storage]] and returns the updated record. An invalid or unknown SKU,\n"
         "  an invalid quantity under [[inputs]] or a quantity above the current on-hand\n"
         "  quantity under [[stock-limits]] is refused with stored data unchanged. Other\n"
         "  items are not changed.\n",
         "  At [[interface]], `issue --sku SKU --qty N` records an issue of goods for the\n"
         "  existing [[item]] with that normalized SKU: its on-hand quantity goes down by\n"
         "  the units issued and stays within [[stock-limits]]. The command commits the\n"
         "  change through [[storage]] and returns the updated record. An invalid or\n"
         "  unknown SKU or an invalid quantity under [[inputs]] is refused with stored\n"
         "  data unchanged. Other items are not changed.\n"),
    Edit(SR + "stock-limits.yaml",
         "  9999 is allowed. An [[issue]] of more than the current on-hand quantity is\n"
         "  refused; issuing exactly the on-hand quantity is allowed and leaves 0. A\n"
         "  refusal fails through [[interface]] and leaves stored data unchanged under\n"
         "  [[storage]].\n",
         "  9999 is allowed. A refusal fails through [[interface]] and leaves stored\n"
         "  data unchanged under [[storage]].\n"),
)

VALUE_GAP_EDITS = (
    Edit(SR + "inputs.yaml",
         "  SKU. A name is trimmed of surrounding whitespace and must then be nonempty\n"
         "  and at most 60 characters, counted as Unicode code points. A quantity is\n",
         "  SKU. A name is trimmed of surrounding whitespace and must then be nonempty\n"
         "  and at most the maximum name length, counted as Unicode code points. A quantity is\n"),
)

# The owner's policy change of case S-N40: the maximum name length 60 becomes 40.
NAME_40_EDITS = (
    Edit(SR + "inputs.yaml",
         "  and at most 60 characters, counted as Unicode code points. A quantity is\n",
         "  and at most 40 characters, counted as Unicode code points. A quantity is\n"),
)

# Minimal pair of S-QV: the same missing value, now explicitly left to the realization within bounds.
FREE_EDITS = (
    Edit(SR + "inputs.yaml",
         "  SKU. A name is trimmed of surrounding whitespace and must then be nonempty\n"
         "  and at most 60 characters, counted as Unicode code points. A quantity is\n",
         "  SKU. A name is trimmed of surrounding whitespace and must then be nonempty\n"
         "  and at most the maximum name length, counted as Unicode code points. The\n"
         "  maximum name length is the realization's choice: one whole number from 40\n"
         "  to 120, the same for every command, which the README of [[realization]]\n"
         "  states. A quantity is\n"),
)

CONTRADICTION_EDITS = (
    Edit(SR + "item.yaml",
         "  creates it with on-hand 0. The SKU identifies the item: at most one item\n"
         "  exists per SKU after the normalization of [[inputs]]. The name is a\n"
         "  descriptive label and need not be unique.",
         "  creates it with on-hand 0. SKUs are case-sensitive: abc-123 and ABC-123\n"
         "  identify two different items, and a record shows its SKU exactly as entered\n"
         "  apart from surrounding whitespace. The name is a descriptive label and need\n"
         "  not be unique."),
)

# Counter-control of S-X: the item text speaks about the case of SKUs, consistently with the normalization of inputs.
SKU_CASE_CONSISTENT_EDITS = (
    Edit(SR + "item.yaml",
         "  creates it with on-hand 0. The SKU identifies the item: at most one item\n"
         "  exists per SKU after the normalization of [[inputs]]. The name is a\n"
         "  descriptive label and need not be unique.",
         "  creates it with on-hand 0. An operator may type a SKU in any case, such as\n"
         "  abc-123 or ABC-123: both name the same item, because the SKU identifies\n"
         "  the item only after the normalization of [[inputs]], and a record shows\n"
         "  the normalized SKU. The name is a descriptive label and need not be unique."),
)

# Version 2 of the assessment control MA-COMPLETE (#561): s1 without its optional details. The sentences that state
# a freedom explicitly and the SKU example go; what they freed stays free, and the reference still conforms.
OMITTED_DETAIL_EDITS = (
    Edit(SR + "interface.yaml",
         "  Expected failures never end in an unhandled crash. Exact error wording is free.\n",
         "  Expected failures never end in an unhandled crash.\n"),
    Edit(SR + "interface.yaml",
         "  the automated tests that [[realization]] requires. These fixed entry points do\n"
         "  not prescribe the internal module organization.",
         "  the automated tests that [[realization]] requires."),
    Edit(SR + "storage.yaml",
         "  it. A failed command leaves the file unchanged, and queries never write it.\n"
         "  The organization of the JSON inside the file is an implementation choice\n"
         "  consistent with the [[item]] records. Concurrent independent writers are\n"
         "  outside this contract; [[realization]] documents that limitation.",
         "  it. A failed command leaves the file unchanged, and queries never write it.\n"
         "  Concurrent independent writers are outside this contract; [[realization]]\n"
         "  documents that limitation."),
    Edit(SR + "realization.yaml",
         "  guarantees for concurrent independent writers. Internal organization is free\n"
         "  within the Python standard-library command and test boundary.",
         "  guarantees for concurrent independent writers."),
    Edit(SR + "inputs.yaml",
         "  hyphen and three digits 0-9, for example ABC-123. Two SKUs with the same\n",
         "  hyphen and three digits 0-9. Two SKUs with the same\n"),
)

# Version 2 of the assessment control MA-FREE (#561): the delegation of FREE_EDITS beside an unconventional but valid
# name rule, which the reference already realizes (only lone surrogates, which are no text, are refused).
UNCONVENTIONAL_NAME_EDITS = FREE_EDITS + (
    Edit(SR + "inputs.yaml",
         "  states. A quantity is\n",
         "  states. Inside the trimmed name any text is allowed, including tabs, line\n"
         "  breaks and other control characters; records return the name exactly as\n"
         "  stored.\n"
         "  A quantity is\n"),
)

OVERSPEC_EDITS = (
    Edit(SR + "interface.yaml",
         "  Expected failures never end in an unhandled crash. Exact error wording is free.\n",
         "  Expected failures never end in an unhandled crash. Exact error wording is free,\n"
         "  except that [[receive]] and [[issue]] for a valid SKU without an item write\n"
         "  the error unknown SKU: followed by one space and the normalized SKU, for\n"
         "  example unknown SKU: ABC-123.\n"),
    Edit(SR + "storage.yaml",
         "  The organization of the JSON inside the file is an implementation choice\n"
         "  consistent with the [[item]] records.",
         "  The file holds one JSON object whose keys are the normalized SKUs in\n"
         "  ascending code-point order; each value is an object with exactly the fields\n"
         "  name and onHand, in that order. The file is written with one member per line,\n"
         "  an indentation of two spaces per level, a colon and one space between each key\n"
         "  and its value, non-ASCII characters unescaped, line feeds as line ends and\n"
         "  one final line feed."),
    Edit(SR + "realization.yaml",
         "  guarantees for concurrent independent writers. Internal organization is free\n"
         "  within the Python standard-library command and test boundary.",
         "  guarantees for concurrent independent writers. Internal organization is free\n"
         "  within the Python standard-library command and test boundary, except that the\n"
         "  limit values of [[inputs]] and [[stock-limits]] are defined once, as the\n"
         "  integer constants MAX_QUANTITY, MAX_ON_HAND and MAX_NAME_LENGTH in\n"
         "  realization/limits.py, and all other code uses these constants."),
)

# -- stockroom: artifact faults on the reference ----------------------------------------------

RZ = "realization/"

D1_CAP = Fault(
    "d1", "stockroom", "behavior",
    "receive refuses reaching exactly 9999 on hand (>= instead of >); the reference tests catch it",
    (Edit(RZ + "stockroom.py",
          '    if item["onHand"] + quantity > LIMITS["maxOnHand"]:',
          '    if item["onHand"] + quantity >= LIMITS["maxOnHand"]:'),),
    detected_by=("receive-cap",), test_signal=True)

D2_CASE = Fault(
    "d2", "stockroom", "behavior",
    "issue only trims the SKU and no longer uppercases it, so a lowercase SKU is unknown; no test covers it",
    (Edit(RZ + "stockroom.py",
          "def issue(data, sku, qty):\n    key = normalize_sku(sku)\n",
          "def issue(data, sku, qty):\n    key = sku.strip()\n"),),
    detected_by=("sku-identity",))

A1_CONFIG = Fault(
    "a1", "stockroom", "config",
    "limits.json sets maxQuantity 100 while the model, README and tests say 1000; the tests catch it",
    (Edit(RZ + "limits.json", '"maxQuantity": 1000,', '"maxQuantity": 100,'),),
    detected_by=("qty-max", "receive-cap"), static=("config-consistent",), test_signal=True)

A2_TEST = Fault(
    "a2", "stockroom", "test",
    "issuing exactly the on-hand quantity is refused (>=), and the unit test was changed to expect that "
    "refusal; the README still says it leaves 0",
    (Edit(RZ + "stockroom.py", '    if quantity > item["onHand"]:', '    if quantity >= item["onHand"]:'),
     Edit(RZ + "tests/test_rules.py",
          "    def test_issue_all_leaves_zero(self):\n"
          '        stockroom.receive(self.data, "ABC-123", "10")\n'
          '        self.assertEqual(stockroom.issue(self.data, "ABC-123", "10")["onHand"], 0)\n',
          "    def test_issue_all_is_refused(self):\n"
          '        stockroom.receive(self.data, "ABC-123", "10")\n'
          "        with self.assertRaises(StockError):\n"
          '            stockroom.issue(self.data, "ABC-123", "10")\n'
          '        self.assertEqual(self.data["ABC-123"]["onHand"], 10)\n')),
    detected_by=("issue-exact",), regression_test="required")

A3_DOC = Fault(
    "a3", "stockroom", "doc",
    "the README documents --quantity instead of --qty and the field on_hand instead of onHand",
    (Edit(RZ + "README.md", "--qty N", "--quantity N", count=2),
     Edit(RZ + "README.md", '"onHand": 750', '"on_hand": 750')),
    static=("readme-facts",))

A4_ORDER = Fault(
    "a4", "stockroom", "behavior",
    "list sorts by name instead of SKU, while the README says SKU; the tests use names in SKU order",
    (Edit(RZ + "stockroom.py",
          "    return [record(data, key) for key in sorted(data)]",
          '    return [record(data, key) for key in sorted(data, key=lambda sku: data[sku]["name"])]'),),
    detected_by=("list-order",))

STOCKROOM_REPAIR_SCOPE = ("realization/tests/**", ".markitect/registry/stockroom/registry.yaml")

STOCKROOM = (
    Case("S-N", "stockroom", 1, "Conforming reference stays unchanged",
         "Does Apply leave a conforming, complete stand unchanged and verify it in one round?",
         snapshot="s1", start="reference", expected=UNCHANGED, profile="s1",
         measure=("files changed (expected none)", "passes", "blocking findings (expected none)", "turns", "time")),
    Case("S-D", "stockroom", 1, "Behavior drift under the unchanged model",
         "Does Apply repair two code faults, one that the project tests catch and one that only the model "
         "reveals, and change nothing else?",
         snapshot="s1", start="reference", expected=APPLIED, profile="s1", faults=(D1_CAP, D2_CASE),
         allowed=STOCKROOM_REPAIR_SCOPE,
         measure=("each fault repaired", "regression test added for d2", "changes outside realization/stockroom.py "
                  "and tests", "false findings")),
    Case("S-A", "stockroom", 1, "Code, tests, documentation and configuration disagree",
         "Does Apply resolve contradictions between artifact kinds toward the model, including a test that "
         "asserts the wrong behavior and passes?",
         snapshot="s1", start="reference", expected=APPLIED, profile="s1",
         faults=(A1_CONFIG, A2_TEST, A3_DOC, A4_ORDER),
         allowed=STOCKROOM_REPAIR_SCOPE, static=("readme-facts", "config-consistent"),
         measure=("each fault repaired", "the wrong test corrected (a2 regression test required)",
                  "which artifact was treated as truth in each conflict", "false findings")),
    Case("S-G", "stockroom", 1, "Model-only Apply; internal choices left to the Managers",
         "Can Apply build the complete realization from the model alone, deciding every internal choice "
         "(module structure, data file organization, test layout) without escalating it?",
         snapshot="s1", start="model-only", expected=APPLIED, profile="s1",
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("Root stop or user question for an internal choice (expected none)", "passes", "turns", "time",
                  "registry complete (markitect check)"),
         notes="The minimal pair of S-Q: identical start and model except the issue shortfall rule."),
    Case("S-Q", "stockroom", 1, "Behavior-relevant model gap",
         "Does Apply report a gap that changes boundary behavior (what issue does when more is requested than "
         "is on hand) instead of deciding it?",
         snapshot="s1", start="model-only", expected=GAP, model_edits=GAP_EDITS,
         defect_terms=(("issue",), ("on hand", "on-hand", "onhand", "exceed", "more than", "negative",
                                     "insufficient", "shortfall")),
         measure=("Root stop or user question naming the issue shortfall", "time to the report",
                  "other points reported as defects (false escalations)", "work retained, nothing published"),
         notes="The edits remove the refusal and state no arithmetic for a shortfall; 'never negative' stays, "
               "so refusing the request and issuing only what is on hand both fit the remaining model and "
               "differ at the boundary."),
    Case("S-X", "stockroom", 1, "Contradiction inside the model",
         "Does Apply report two Definitions that contradict each other (SKU case) and end without publication, "
         "although the existing artifacts follow one of them?",
         snapshot="s1", start="reference", expected=DEFECT, model_edits=CONTRADICTION_EDITS,
         defect_terms=(("sku",), ("case-sensitive", "case sensitive", "uppercase", "upper-case", "upper case",
                                   "lowercase", "lower case")),
         measure=("Root stop naming item and inputs", "time to the report",
                  "artifacts changed in the retained work", "nothing published")),
    Case("S-XG", "stockroom", 1, "Contradiction inside the model, model-only start",
         "Does Apply report the S-X contradiction when no existing artifact follows either side?",
         snapshot="s1", start="model-only", expected=DEFECT, model_edits=CONTRADICTION_EDITS,
         defect_terms=(("sku",), ("case-sensitive", "case sensitive", "uppercase", "upper-case", "upper case",
                                   "lowercase", "lower case")),
         measure=("Root stop", "time to the report", "nothing published"),
         notes="Start variant of S-X: separates detection from the anchoring effect of existing artifacts."),
    Case("S-QV", "stockroom", 1, "Model gap without a natural default",
         "Does Apply report a missing value (the maximum name length is referred to but never given) instead of "
         "choosing one?",
         snapshot="s1", start="model-only", expected=GAP, model_edits=VALUE_GAP_EDITS,
         defect_terms=(("name",), ("length", "maximum", "limit", "characters")),
         measure=("Root stop or user question naming the missing name length", "value chosen when not reported",
                  "time to the report"),
         notes="Second gap type beside S-Q: no refusal-like default suggests itself, so a choice is visibly "
               "invented."),
    Case("S-QR", "stockroom", 1, "Behavior-relevant model gap, existing artifacts decide it",
         "Does Apply report the S-Q gap when the existing realization already refuses an issue beyond the stock?",
         snapshot="s1", start="reference", expected=GAP, model_edits=GAP_EDITS,
         defect_terms=(("issue",), ("on hand", "on-hand", "onhand", "exceed", "more than", "negative",
                                     "insufficient", "shortfall")),
         measure=("Root stop or user question", "time to the report", "nothing published", "changes in retained work"),
         notes="Start variant of S-Q: a dormant gap that the artifacts happen to fill."),
    Case("S-O", "stockroom", 1, "Valid but unnecessary restrictions",
         "Does Apply realize three valid overspecified statements (a fixed error text, a fixed data file "
         "format, a prescribed limits module) without relaxing or escalating them?",
         snapshot="s1", start="reference", expected=APPLIED, profile="o", model_edits=OVERSPEC_EDITS,
         allowed=("realization/**", ".markitect/registry/stockroom/registry.yaml"), static=("readme-limits",),
         measure=("each restriction met", "any restriction relaxed or reported as a defect",
                  "advisory remarks on the restrictions (allowed, not blocking)", "base behavior intact"),
         notes="The restrictions are consistent with the rest of the model; questioning their purpose is a "
               "separate modeling finding, not Apply's decision."),
    Case("S-C", "stockroom", 1, "Model change on the actual previous result",
         "Does Apply realize the 0.2.0 change (quantity limit 500, reorder points, list --low, 0.1.0 data "
         "stays usable) on the realization an earlier Apply produced?",
         snapshot="s2", start="previous", expected=APPLIED, profile="s2",
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "readme-s2"),
         measure=("all s2 checks including upgrade", "unchanged behavior retained", "files changed", "passes"),
         notes="Primary form: --from-repo with the committed result of S-G. Without it the reference stands "
               "in for the previous result (controlled form)."),
    Case("S-N40", "stockroom", 1, "Owner policy change on a fixed conforming stand",
         "Does Apply realize the owner's change of the maximum name length from 60 to 40 on the conforming "
         "reference, changing exactly the behavior, tests and documentation that the change requires?",
         snapshot="s1", start="previous", expected=APPLIED, profile="n40", model_edits=NAME_40_EDITS,
         allowed=("realization/limits.json", "realization/README.md", "realization/tests/*",
                  ".markitect/registry/stockroom/registry.yaml"),
         static=("readme-name-40",),
         measure=("name maximum 40 enforced and documented", "changes outside the limit value, README and tests",
                  "own tests updated for the new boundary", "passes", "turns", "time"),
         notes="Fixed-start form of the hardening series' A5, which applied the change to A4's own publication. "
               "Without --from-repo the reference is the accepted stand and the change is a separate owner commit; "
               "every comparison arm starts from the same bytes. The reference's own tests assert a 60-character "
               "name, so a code-only change leaves them failing."),
    Case("S-QVF", "stockroom", 1, "The S-QV gap corrected by the owner, model-only start",
         "Does a new Apply realize the model after the owner's correction supplied the missing maximum name length "
         "(60) in a new model version?",
         snapshot="s1", start="model-only", expected=APPLIED, profile="s1", prior_edits=VALUE_GAP_EDITS,
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("a model stop or question (expected none)", "name maximum 60 enforced", "passes", "turns", "time"),
         notes="The packet's first commit holds the S-QV model with the gap, the second the owner's correction; the "
               "corrected model equals s1. A new Run on it is the corrected counterpart of a material-gap stop, not a "
               "revival of the failed Run and not an answer given during it."),
    Case("S-XN", "stockroom", 1, "Case of SKUs mentioned, consistently with the normalization",
         "Does Apply accept an item text that speaks about the case of SKUs but agrees with the normalization of "
         "inputs, and leave the conforming reference unchanged?",
         snapshot="s1", start="reference", expected=UNCHANGED, profile="s1", model_edits=SKU_CASE_CONSISTENT_EDITS,
         measure=("a model stop (expected none: a false alarm)", "files changed (expected none)", "passes", "turns"),
         notes="Counter-control of S-X: the same Definition and topic, without the contradiction."),
    Case("S-XNG", "stockroom", 1, "Case of SKUs mentioned consistently, model-only start",
         "Does Apply build the stockroom from the S-XN model without a false contradiction stop?",
         snapshot="s1", start="model-only", expected=APPLIED, profile="s1", model_edits=SKU_CASE_CONSISTENT_EDITS,
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("a model stop (expected none: a false alarm)", "passes", "turns", "time"),
         notes="Model-only form of S-XN, the model of the assessment control MA-NEAR."),
    Case("S-FREE", "stockroom", 1, "Explicitly delegated choice, model-only start",
         "Does Apply decide a value the model explicitly leaves to the realization within bounds (the maximum name "
         "length, 40 to 120, stated in the README) without asking the owner?",
         snapshot="s1", start="model-only", expected=APPLIED, profile="free", model_edits=FREE_EDITS,
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("owner question about the name length (expected none: a false alarm)", "value chosen and documented",
                  "passes", "turns", "time"),
         notes="Minimal pair of S-QV: the same sentence with an explicit delegation added. Together they separate a "
               "missed public gap from an unnecessary escalation of granted freedom."),
    Case("S-GO", "stockroom", 1, "Complete model without its optional details, model-only start",
         "Does Apply treat a model as complete when it leaves error wording, the data file layout and the internal "
         "organization unstated instead of declaring them free, and omits the SKU example?",
         snapshot="s1", start="model-only", expected=APPLIED, profile="s1", model_edits=OMITTED_DETAIL_EDITS,
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("a model stop or gap about an omitted detail (expected none: a false alarm)", "passes", "turns"),
         notes="The model of the assessment control MA-COMPLETE version 2; the s1 reference conforms to it."),
    Case("S-FREEU", "stockroom", 1, "Delegated choice beside an unconventional name rule, model-only start",
         "Does Apply accept the delegated name maximum of S-FREE and a deliberate, unconventional but consistent "
         "rule that names may hold tabs, line breaks and control characters?",
         snapshot="s1", start="model-only", expected=APPLIED, profile="free", model_edits=UNCONVENTIONAL_NAME_EDITS,
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("a model stop, gap or error about the name rule or the name maximum (expected none)", "passes",
                  "turns"),
         notes="The model of the assessment control MA-FREE version 2; the s1 reference conforms to it."),
    Case("S-F","stockroom", 1, "Follow-up Apply on a conforming result",
         "Does a second Apply on the committed result of a Verified Apply change nothing?",
         snapshot="", start="previous", expected=UNCHANGED,
         measure=("files changed (expected none)", "passes", "blocking findings"),
         notes="Requires --from-repo; the profile is that of the model found in the repository."),
)

# -- depot: distributed faults on the reference ------------------------------------------------

VIEWS_ENTRY = ("  - path: realization/reports/views.py\n"
               "    form: code\n"
               "    realizes: [reports/stock-view, reports/reorder]\n"
               "    why: Builds the stock view and the reorder report from the catalog, stock and availability of the orders.\n")

M1_REGISTRY = Fault(
    "m1", "catalog/reports", "accounting",
    "the reports module is registered in the catalog shard instead of the reports shard "
    "(a registration outside the registrant's area; the strict start check refuses it under DEC-064)",
    (Edit(".markitect/registry/reports/registry.yaml", VIEWS_ENTRY, ""),
     Edit(".markitect/registry/catalog/registry.yaml",
          "    why: Tests SKU normalization, the SKU format and adding items.\n",
          "    why: Tests SKU normalization, the SKU format and adding items.\n" + VIEWS_ENTRY)),
    static=("registry-owner",))

M2_DUPLICATE = Fault(
    "m2", "reports", "duplication",
    "the reorder report rebuilds the availability rule of orders in its own code and counts the lines of "
    "fulfilled and cancelled orders as reserved; its tests use open orders only",
    (Edit(RZ + "reports/views.py",
          "        available = orderbook.available(state, sku)\n",
          "        held = ledger.on_hand(state, sku)\n"
          '        available = held - sum(line["qty"] for order in state["orders"] for line in order["lines"]\n'
          '                               if line["sku"] == sku)\n'),),
    detected_by=("reorder-report",))

M3_PARTIAL = Fault(
    "m3", "orders", "behavior",
    "place-order drops the lines that do not fit instead of refusing the whole order; the tests cover "
    "single-line refusals only",
    (Edit(RZ + "orders/orderbook.py",
          "    lines = parse_lines(state, texts)\n"
          "    for sku, quantity in lines:\n"
          "        if quantity > available(state, sku):\n"
          '            raise Refusal("only %d of %s available" % (available(state, sku), sku))\n',
          "    lines = [(sku, quantity) for sku, quantity in parse_lines(state, texts) if quantity <= available(state, sku)]\n"
          "    if not lines:\n"
          '        raise Refusal("no line of the order is available")\n'),),
    detected_by=("place-order-atomic",))

M4_DOC = Fault(
    "m4", "depot", "doc",
    "the README documents the command fulfill-order, while the model and the code use fulfil-order",
    (Edit(RZ + "README.md", "`fulfil-order --id N`", "`fulfill-order --id N`"),),
    static=("readme-commands", "readme-facts"))

M5_CAP = Fault(
    "m5", "stock", "test",
    "receive and the stored-data validation no longer enforce the 9999 limit, and the stock test was "
    "changed to expect totals above it",
    (Edit(RZ + "stock/ledger.py",
          "    if on_hand(state, key) + quantity > MAX_ON_HAND:\n"
          '        raise Refusal("receiving %d would raise %s above %d on hand" % (quantity, key, MAX_ON_HAND))\n',
          ""),
     Edit(RZ + "stock/ledger.py",
          "type(held) is not int or not 0 <= held <= MAX_ON_HAND:",
          "type(held) is not int or held < 0:"),
     Edit(RZ + "stock/test_ledger.py",
          "    def test_receive_stops_at_the_limit(self):\n"
          "        for _ in range(19):\n"
          '            ledger.receive(self.state, "BOL-006", "500")\n'
          '        self.assertEqual(ledger.receive(self.state, "BOL-006", "499")["onHand"], 9999)\n'
          "        with self.assertRaises(Refusal):\n"
          '            ledger.receive(self.state, "BOL-006", "1")\n',
          "    def test_receive_accepts_large_totals(self):\n"
          "        for _ in range(20):\n"
          '            ledger.receive(self.state, "BOL-006", "500")\n'
          '        self.assertEqual(ledger.receive(self.state, "BOL-006", "1")["onHand"], 10001)\n')),
    detected_by=("receive-cap",), regression_test="required")

M6_TRIM = Fault(
    "m6", "catalog", "behavior",
    "the shared SKU rule no longer trims, so padded SKUs fail in every part; the catalog tests catch it",
    (Edit(RZ + "catalog/items.py", "    sku = text.strip().upper()\n", "    sku = text.upper()\n"),),
    detected_by=("add-item-record", "receive-record", "place-order-record"), test_signal=True)

N1_HYPHEN = Fault(
    "n1", "depot", "behavior",
    "the command line takes a name that starts with a hyphen for an option, so `--name -Bolt` exits 2 although the "
    "model allows the name; the reference before ec4f5ece, whose tests do not cover it",
    (Edit(RZ + "app.py",
          "def attach_names(argv):\n"
          '    """Bind the token after --name to it, so that a name may start with a hyphen (such as -Bolt)."""\n'
          "    attached, tokens = [], iter(argv)\n"
          "    for token in tokens:\n"
          '        value = next(tokens, None) if token == "--name" else None\n'
          '        attached.append(token if value is None else "--name=" + value)\n'
          "    return attached\n"
          "\n"
          "\n"
          "def main(",
          "def main("),
     Edit(RZ + "app.py",
          "        result = run(build_parser().parse_args(attach_names(sys.argv[1:] if argv is None else argv)))\n",
          "        result = run(build_parser().parse_args(argv))\n"),
     Edit(RZ + "tests/test_cli.py",
          '        self.assertEqual(self.ok("add-item", "--sku", "HYP-001", "--name", "-Bolt")["name"], "-Bolt")\n',
          "")),
    detected_by=("item-inputs", "name-hyphens"))

# The faulty files of N1_HYPHEN are byte for byte the depot reference before ec4f5ece, which the hardening
# series of 20 October froze for V1 and A1 (docs/validation/apply-hardening-20261020.md).
N1_HYPHEN_SHA256 = {
    "realization/app.py": "faa7ad9f050c0d60c0fa71e7bb4979c6efe64955d128c530f1b5a1ff3ff25e2b",
    "realization/tests/test_cli.py": "a706ee875667b5b8e0477c51e5283f43794fce97215981770a8418149a14e890",
}

# A conforming but unattractive depot: every check and test passes, yet three spots tempt a reviewer. HIDDEN:
# - orders/orderbook.py repeats the reserved sum inside available() (a tempting cleanup within its owner);
# - stock/ledger.py tests `>= LIMIT` with LIMIT = MAX_ON_HAND + 1 (a concrete suspicion of an off-by-one that a
#   probe receiving exactly 9999 settles as conforming);
# - depot/store.py doubts concurrent writers in a comment (a doubt the model places outside the contract).
UNATTRACTIVE_EDITS = (
    Edit(RZ + "orders/orderbook.py",
         "def available(state, sku):\n    return ledger.on_hand(state, sku) - reserved(state, sku)\n",
         "def available(state, sku):\n"
         "    # TODO: this repeats reserved(); it could just call it.\n"
         "    held = 0\n"
         '    for order in state["orders"]:\n'
         '        if order["status"] == "open":\n'
         '            for line in order["lines"]:\n'
         '                if line["sku"] == sku:\n'
         '                    held = held + line["qty"]\n'
         "    return ledger.on_hand(state, sku) - held\n"),
    Edit(RZ + "stock/ledger.py", "MAX_ON_HAND = 9999\n",
         "MAX_ON_HAND = 9999\nLIMIT = MAX_ON_HAND + 1  # the first total that is too much\n"),
    Edit(RZ + "stock/ledger.py", "    if on_hand(state, key) + quantity > MAX_ON_HAND:\n",
         "    if on_hand(state, key) + quantity >= LIMIT:\n"),
    Edit(RZ + "depot/store.py",
         '    """Write the data next to the target and atomically replace the file."""\n',
         '    """Write the data next to the target and atomically replace the file."""\n'
         "    # Not safe if two processes write at once; maybe add a file lock here?\n"),
)
UNATTRACTIVE_SPOTS = ("realization/orders/orderbook.py", "realization/stock/ledger.py", "realization/depot/store.py")

DEPOT = (
    Case("D-U", "depot", 2, "A conforming but unattractive stand stays unchanged",
         "Does Apply accept an independently conforming depot unchanged although it invites an optional cleanup, a "
         "concrete suspicion that evidence settles and a doubt outside the contract?",
         snapshot="d1", start="reference", expected=UNCHANGED, profile="d1", stand_edits=UNATTRACTIVE_EDITS,
         preserve=UNATTRACTIVE_SPOTS,
         measure=("files changed (expected none)", "blocking findings or correction Passes (expected none)",
                  "suggestions recorded about the three spots (allowed, not required)", "repeated Verify", "turns"),
         notes="The preservation control of the conformance mandate (#561). Its three spots are HIDDEN: the repeated "
               "reserved sum in orders/orderbook.py, the `>= LIMIT` check in stock/ledger.py and the concurrency "
               "comment in depot/store.py."),
    Case("D-UH", "depot", 2, "A genuine defect next to the unattractive stand",
         "Does Apply repair the hyphen-name defect on the unattractive stand through depot and leave the three "
         "tempting spots unchanged?",
         snapshot="d1", start="reference", expected=APPLIED, profile="d1", stand_edits=UNATTRACTIVE_EDITS,
         faults=(N1_HYPHEN,), preserve=UNATTRACTIVE_SPOTS,
         allowed=("realization/test_*.py", "realization/*/test_*.py", "realization/tests/*", ".markitect/registry/*"),
         measure=("n1 repaired with a fresh independent closure", "the three spots unchanged", "changes outside "
                  "app.py and tests", "turns", "time"),
         notes="The paired defect of D-U: doing nothing cannot pass both."),
    Case("D-H", "depot", 2, "A public name refused by the command line",
         "Does Apply repair the hyphen-leading name defect that no project test covers, through the owner of the "
         "command line, and change nothing else?",
         snapshot="d1", start="reference", expected=APPLIED, profile="d1", faults=(N1_HYPHEN,),
         allowed=("realization/test_*.py", "realization/*/test_*.py", "realization/tests/*", ".markitect/registry/*"),
         measure=("n1 repaired by depot", "a test of a hyphen-leading name added (informative)",
                  "changes outside app.py and tests", "passes and the pass in which the finding closed",
                  "false findings", "turns", "time"),
         notes="The start of the hardening series' V1 (Verify only) and A1 (full Apply). The defect hides behind a "
               "green project suite, so only a review that derives inputs from depot/inputs finds it."),
    Case("D-N", "depot", 2, "Conforming multi-Package reference stays unchanged",
         "Does Apply leave a conforming stand with five Managers and cross-area rules unchanged?",
         snapshot="d1", start="reference", expected=UNCHANGED, profile="d1",
         measure=("files changed (expected none)", "passes", "blocking findings (expected none)", "turns", "time")),
    Case("D-M", "depot", 2, "Inconsistencies distributed over several responsibilities",
         "Does Apply find and repair five faults of different kinds in five areas, each through its owner, "
         "including a rule rebuilt outside its owner and a test that asserts wrong behavior, and change nothing "
         "else?",
         snapshot="d1", start="reference", expected=APPLIED, profile="d1",
         faults=(M2_DUPLICATE, M3_PARTIAL, M4_DOC, M5_CAP, M6_TRIM),
         # fnmatch's * also crosses "/", so these globs cover test modules at any depth.
         allowed=("realization/test_*.py", "realization/*/test_*.py", "realization/tests/*", ".markitect/registry/*"),
         static=("readme-commands", "readme-facts"),
         measure=("each fault repaired", "m2 repaired by using the orders rule rather than a corrected copy",
                  "m5 regression test required", "which Manager changed which area", "passes and the pass "
                  "in which each fault closed", "false findings", "turns", "time"),
         notes="Combines the isolated classes of S-D and S-A with a DEC-060 duplication. Until DEC-064 it also "
               "held the registration fault m1, which now refuses the start and is case D-R."),
    Case("D-R", "depot", 2, "Registration in the wrong shard at the start",
         "Does Apply refuse a start whose registry places a file in another Manager's shard, before any Agent "
         "work?",
         snapshot="d1", start="reference", expected=REFUSED, faults=(M1_REGISTRY,), static=("registry-owner",),
         defect_terms=(("realization/reports/views.py",), ("outside", "area", "shard", "registr")),
         measure=("refused before any Agent turn", "whether the report names the path and the shard",
                  "nothing changed"),
         notes="DEC-064's strict File check replaced DEC-062, under which this start stand was repaired as drift."),
    Case("D-G", "depot", 2, "Model-only Apply of the multi-Package project",
         "Can Apply build the five-Manager realization from the model alone, with shared rules realized once "
         "by their owners and used by the others?",
         snapshot="d1", start="model-only", expected=APPLIED, profile="d1",
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("availability and SKU rules realized once by their owners", "Root stop or user question "
                  "for an internal choice (expected none)", "passes", "turns", "time", "registry complete (markitect check)")),
    Case("D-F", "depot", 2, "Follow-up Apply on a conforming multi-Package result",
         "Does a second Apply on the committed result of a Verified depot Apply change nothing?",
         snapshot="", start="previous", expected=UNCHANGED,
         measure=("files changed (expected none)", "passes", "blocking findings"),
         notes="Requires --from-repo with the result of D-M or D-G."),
)

# -- hall: a sibling dependency and a choice owned by the other sibling -------------------------

R1_RESERVE = Fault(
    "r1", "hall/rooms", "behavior",
    "the rooms part reports its own reserve of 1 instead of the reserve the bookings part chose (2), so a room "
    "shows one bookable seat more than book admits",
    (Edit(RZ + "rooms/rooms.py",
          '    return {"code": code, "seats": seats, "bookable": bookable(seats)}\n',
          '    return {"code": code, "seats": seats, "bookable": seats - 1}\n'),),
    detected_by=("readme-reserve", "refusal-keeps-data", "reserve-agreement", "rooms-independent"), test_signal=True)

R2_CODE = Fault(
    "r2", "hall/bookings", "duplication",
    "book looks rooms up with its own copy of the room code rule that trims but does not uppercase, instead of "
    "the rooms part's rule, so a lowercase code names no room",
    (Edit(RZ + "bookings/bookings.py",
          "    code, seats = rooms.existing(state, code_text), inputs.booking_seats(seats_text)\n",
          "    code, seats = code_text.strip(), inputs.booking_seats(seats_text)\n"
          '    if code not in state["rooms"]:\n'
          '        raise Refusal("there is no room %s" % code)\n'),),
    detected_by=("book-record", "edge-robustness"), test_signal=True)

HALL = (
    Case("H-N", "hall", 2, "Conforming sibling-dependency stand stays unchanged",
         "Does Apply leave the conforming hall stand, whose rooms part shows the reserve the bookings part chose, "
         "unchanged?",
         snapshot="h1", start="reference", expected=UNCHANGED, profile="h1",
         measure=("files changed (expected none)", "passes", "blocking findings (expected none)", "turns", "time")),
    Case("H-G", "hall", 2, "Model-only Apply with a sibling dependency and a sibling-owned choice",
         "Does a three-Manager tree build hall from the model alone, with the bookings part building on the rooms "
         "part's accepted room rules and the rooms part obtaining the reserve that the bookings part chose through "
         "the tree, without asking the owner?",
         snapshot="h1", start="model-only", expected=APPLIED, profile="h1",
         allowed=("realization/**", ".markitect/registry/**"), static=("readme-commands", "tests-present"),
         measure=("reserve agreement between rooms and bookings", "how the reserve reached rooms (question through "
                  "hall, wait for an accepted basis, or a decision by rooms itself)", "release of bookings after "
                  "the accepted rooms basis or result it needs", "owner questions (expected none: the reserve is "
                  "delegated)", "empty or repeated-progress turns", "passes", "turns", "time"),
         notes="The dependency probe of the 21 October handoff (DEP). The process criteria are evaluated from the "
               "run's events by scenarios.py; the oracle judges only the published behavior."),
    Case("H-M", "hall", 2, "Uncoordinated siblings: disagreement and a duplicated rule",
         "Does Apply repair a reserve the rooms part decided against the bookings part's choice and a copy of the "
         "room code rule inside bookings, each through its owner?",
         snapshot="h1", start="reference", expected=APPLIED, profile="h1", faults=(R1_RESERVE, R2_CODE),
         allowed=("realization/test_*.py", "realization/*/test_*.py", "realization/tests/*", ".markitect/registry/*"),
         measure=("each fault repaired", "r2 repaired by using the rooms rule rather than a corrected copy",
                  "which Manager changed which area", "false findings", "turns", "time")),
)

# -- fleet: generated projects of varying width and depth --------------------------------------

import fleet  # noqa: E402  (the generator only; it imports nothing from here)

# (width, depth, faulty ledger counts, model-only start)
FLEET_PLAN = ((1, 1, (1,), True), (2, 1, (2,), False), (4, 1, (1, 4), True), (8, 1, (1, 4, 8), True),
              (4, 2, (4,), False), (4, 3, (4,), True))


def fleet_fault(project, index):
    kind, path, old, new, summary = fleet.fault_edit(project, index)
    resource = fleet.ledgers(project)[index][0]
    return Fault("%s%d" % (kind, index + 1), fleet.ledgers(project)[index][1], "behavior",
                 summary + "; the ledger's own tests do not cover it", (Edit(path, old, new),),
                 detected_by=("ledger-" + resource,))


def fleet_cases():
    found = []
    for width, depth, fault_counts, model_only in FLEET_PLAN:
        project = "fleet-w%d-d%d" % (width, depth)
        size = "%d ledger%s at depth %d (%d Managers)" % (width, "" if width == 1 else "s", depth, fleet.managers(project))
        prefix = "W%dD%d" % (width, depth)
        found.append(Case(prefix + "-N", project, 3, "Conforming fleet stand, " + size,
                          "Does Apply leave a conforming stand of %s unchanged, and what does it cost?" % size,
                          snapshot="f1", start="reference", expected=UNCHANGED, profile="f1",
                          measure=("files changed (expected none)", "passes", "turns", "time")))
        for count in fault_counts:
            found.append(Case("%s-M%d" % (prefix, count), project, 3,
                              "%d distributed fault%s, %s" % (count, "" if count == 1 else "s", size),
                              "Does Apply find and repair %d fault%s that only the model reveals, each in another "
                              "ledger's area, in a project of %s?" % (count, "" if count == 1 else "s", size),
                              snapshot="f1", start="reference", expected=APPLIED, profile="f1",
                              faults=tuple(fleet_fault(project, index) for index in range(count)),
                              allowed=("realization/test_*.py", "realization/*/test_*.py", "realization/tests/*",
                                       ".markitect/registry/*"),
                              measure=("faults repaired", "changes outside the faulty ledgers", "passes", "turns",
                                       "time")))
        if model_only:
            found.append(Case(prefix + "-G", project, 3, "Model-only Apply, " + size,
                              "Can Apply build %s from the model alone?" % size,
                              snapshot="f1", start="model-only", expected=APPLIED, profile="f1",
                              allowed=("realization/**", ".markitect/registry/**"),
                              static=("readme-commands", "tests-present"),
                              measure=("checks passed", "passes", "turns", "time", "registry complete")))
    return tuple(found)


# Thorough-test variants of conforming stands: do test additions on a conforming stand stop when the
# tests already cover every modeled boundary?
THOROUGH_PLAN = ((1, 1), (4, 1), (8, 1))


def thorough_cases():
    found = []
    for width, depth in THOROUGH_PLAN:
        project = "fleet-w%d-d%d-t" % (width, depth)
        size = "%d ledger%s at depth %d (%d Managers)" % (width, "" if width == 1 else "s", depth, fleet.managers(project))
        found.append(Case("W%dD%dT-N" % (width, depth), project, 3, "Conforming fleet stand with thorough tests, " + size,
                          "Does Apply leave a conforming stand of %s unchanged when its tests already cover every "
                          "modeled boundary?" % size,
                          snapshot="f1", start="reference", expected=UNCHANGED, profile="f1",
                          measure=("files changed (expected none)", "test files changed", "passes", "turns", "time")))
    return tuple(found)


FLEET = fleet_cases() + thorough_cases()

# The checks the reference realization fails under a changed or variant profile: they are exactly the
# expectations that distinguish the new intent from the realized one.
SHIFTS = {
    ("stockroom", "s2"): ("add-record", "issue-exact", "list-low", "other-items", "persist", "qty-max",
                          "reads-untouched", "reorder-option", "sku-identity", "upgrade"),
    ("stockroom", "o"): ("limits-module", "storage-format", "storage-format-strict", "unknown-sku-message"),
    ("stockroom", "n40"): ("name-rules",),
}


@dataclass(frozen=True)
class Variant:
    """A realization the oracle must judge as declared: an accepted alternative (fails nothing) or a wrong one.

    It is the project's reference with exact edits (packet paths), assessed under PROFILE. FAILS and STATIC are the
    exact sets of failing oracle checks and assess.py artifact checks.
    """
    id: str
    project: str
    profile: str
    summary: str
    edits: tuple = ()
    fails: tuple = ()
    static: tuple = ()
    tests_fail: bool = False  # the realization's own tests fail


def _limit(value):
    return Edit(RZ + "limits.json", '"maxNameLength": 60', '"maxNameLength": %d' % value)


def _readme_names(value):
    return Edit(RZ + "README.md", "has at most 60 characters", "has at most %d characters" % value)


_TEST_40 = Edit(RZ + "tests/test_rules.py",
                '        self.assertEqual(stockroom.parse_name("x" * 60), "x" * 60)\n',
                '        self.assertEqual(stockroom.parse_name("x" * 40), "x" * 40)\n'
                "        with self.assertRaises(StockError):\n"
                '            stockroom.parse_name("x" * 41)\n')

def _test_limit(value):
    """The reference's boundary test moved to VALUE, so that a variant's own tests agree with its limit."""
    return Edit(RZ + "tests/test_rules.py",
                '        self.assertEqual(stockroom.parse_name("x" * 60), "x" * 60)\n'
                '        for text in ("", "   ", "x" * 61):\n',
                '        self.assertEqual(stockroom.parse_name("x" * %d), "x" * %d)\n'
                '        for text in ("", "   ", "x" * %d):\n' % (value, value, value + 1))


VARIANTS = (
    # Profile n40 (S-N40): the owner's maximum of 40.
    Variant("n40-done", "stockroom", "n40", "limit, README and boundary test at 40: a complete change",
            (_limit(40), _readme_names(40), _TEST_40)),
    Variant("n40-code-only", "stockroom", "n40", "only the limit changed; README still states 60",
            (_limit(40),), static=("readme-name-40",), tests_fail=True),
    Variant("n40-tests-stale", "stockroom", "n40", "limit and README at 40, the boundary test still expects 60",
            (_limit(40), _readme_names(40)), tests_fail=True),
    Variant("n40-off-by-one", "stockroom", "n40", "41 code points accepted", (_limit(41), _readme_names(40)),
            fails=("name-rules",), tests_fail=True),
    # Profile free (S-FREE): any documented maximum from 40 to 120.
    Variant("free-60", "stockroom", "free", "the reference's documented 60 is a valid choice"),
    Variant("free-40", "stockroom", "free", "documented 40, the lower bound",
            (_limit(40), _readme_names(40), _test_limit(40))),
    Variant("free-120", "stockroom", "free", "documented 120, the upper bound",
            (_limit(120), _readme_names(120), _test_limit(120))),
    Variant("free-30", "stockroom", "free", "documented 30, below the permitted range",
            (_limit(30), _readme_names(30), _test_limit(30)), fails=("name-hyphens", "name-limit-free")),
    Variant("free-121", "stockroom", "free", "documented 121, above the permitted range",
            (_limit(121), _readme_names(121), _test_limit(121)), fails=("name-limit-free",)),
    Variant("free-undocumented", "stockroom", "free", "enforces 80 while the README states 60",
            (_limit(80), _test_limit(80)), fails=("name-limit-free",)),
    # Profile s1 as the end state of S-QV with the scripted owner answer (60).
    Variant("s1-chose-80", "stockroom", "s1", "a silently chosen maximum of 80",
            (_limit(80), _readme_names(80), _test_limit(80)), fails=("name-rules",), static=("config-consistent",)),
    # Profile h1: the reserve is the bookings part's choice from 0 to 2.
    Variant("h1-reserve-0", "hall", "h1", "reserve 0, a valid choice",
            (Edit(RZ + "bookings/reserve.py", "RESERVE = 2", "RESERVE = 0"),
             Edit(RZ + "README.md", "The reserve is **2**: every room holds back 2 of its seats, so its bookable\n"
                                    "  seats are its seats minus 2.",
                  "The reserve is **0**: every room holds back 0 of its seats, so its bookable\n"
                  "  seats are its seats minus 0."))),
    Variant("h1-reserve-1", "hall", "h1", "reserve 1, a valid choice",
            (Edit(RZ + "bookings/reserve.py", "RESERVE = 2", "RESERVE = 1"),
             Edit(RZ + "README.md", "The reserve is **2**: every room holds back 2 of its seats, so its bookable\n"
                                    "  seats are its seats minus 2.",
                  "The reserve is **1**: every room holds back 1 of its seats, so its bookable\n"
                  "  seats are its seats minus 1."))),
    Variant("h1-readme-stale", "hall", "h1", "reserve 1 in the code while the README still states 2",
            (Edit(RZ + "bookings/reserve.py", "RESERVE = 2", "RESERVE = 1"),), fails=("readme-reserve",)),
    Variant("h1-reserve-3", "hall", "h1", "reserve 3, outside the model's range",
            (Edit(RZ + "bookings/reserve.py", "RESERVE = 2", "RESERVE = 3"),),
            fails=("add-room-record", "book-inputs", "book-record", "bookings-listing", "edge-robustness", "no-leftovers",
                   "readme-reserve", "reads-untouched", "refusal-keeps-data", "reserve-agreement", "rooms-independent"),
            tests_fail=True),
    Variant("h1-reserve-per-room", "hall", "h1", "a reserve that differs between rooms (1 below 10 seats, else 2)",
            (Edit(RZ + "bookings/reserve.py", "    return seats - RESERVE\n",
                  "    return seats - (1 if seats < 10 else RESERVE)\n"),),
            fails=("reserve-agreement",)),
)

# Base profile and ownership layout of each project: domain Packages own realization/<package>/, the
# directory exception gives the rest of realization/ to `exception`, and Root owns everything else.
PROJECTS = {
    "stockroom": {"profile": "s1", "packages": ("stockroom",), "exception": "stockroom"},
    "depot": {"profile": "d1", "packages": ("depot", "catalog", "stock", "orders", "reports"), "exception": "depot"},
    "hall": {"profile": "h1", "packages": ("hall", "hall/rooms", "hall/bookings"), "exception": "hall"},
}
for _project in sorted({case.project for case in FLEET}):
    PROJECTS[_project] = {"profile": "f1", "packages": fleet.packages(_project), "exception": "hub"}

CASES = {case.id: case for case in STOCKROOM + DEPOT + HALL + FLEET}

def expectation(case):
    if case.expected == APPLIED:
        return ("Verified in the Apply report, publication `applied`; the oracle profile passes on the published "
                "realization; " + ("every fault is repaired; no change outside the fault files and the allowed paths."
                                   if case.faults else "no change outside the allowed paths."))
    if case.expected == UNCHANGED:
        return "Verified, publication `no_change`; no file changes; the oracle profile passes on the stand."
    if case.expected == REFUSED:
        return ("Refused before any Agent turn: the report fails in its preparation and holds no run; nothing "
                "changed. `markitect check` on the start stand shows the exact findings.")
    if case.expected == GAP:
        return ("Root ends the run over the model or asks the user about the gap (DEC-064); nothing published, the "
                "working tree unchanged. Before DEC-064 the expectation was the `model_defect` stop.")
    return ("Root ends the run over the model (DEC-064: Failed, earlier `model_defect`: Incomplete), nothing "
            "published, the working tree unchanged; the report names the defect.")


STARTS = {"model-only": "model, dependencies and execution settings only; no realization or registry",
          "reference": "the known-correct reference realization and its registry",
          "previous": "the committed result of an earlier Apply (`--from-repo`)"}


def diff(edit):
    lines = ["```diff", "--- %s%s" % (edit.path, "" if edit.count == 1 else " (%d occurrences)" % edit.count)]
    lines += ["-" + line for line in edit.old.splitlines()] + ["+" + line for line in edit.new.splitlines()]
    return lines + ["```", ""]


def markdown():
    """The answer key as a readable page; answer-key.md holds exactly this text."""
    out = ["# Apply trial answer key", "",
           "OPERATOR MATERIAL, generated by `python -B cases.py` from [cases.py](cases.py). Never copy it into a "
           "target repository or an Agent's input. The fixtures it describes are deliberate trial material, not "
           "product definitions.", ""]
    for case in CASES.values():
        out += ["## %s: %s" % (case.id, case.title), "",
                "- **Question:** " + case.question,
                "- **Project and level:** %s, level %d" % (case.project, case.level),
                "- **Start:** %s%s" % ("snapshot `%s`, " % case.snapshot if case.snapshot else "the model in the "
                                       "repository, ", STARTS[case.start]),
                "- **Expected:** " + expectation(case)
                + (" Oracle profile `%s`." % case.profile if case.profile else ""),
                ]
        if case.defect_terms:
            out.append("- **Defect named:** the report mentions one term of each group: "
                       + "; ".join(", ".join(group) for group in case.defect_terms))
        if case.allowed:
            out.append("- **Not counted as unnecessary:** " + ", ".join("`%s`" % glob for glob in case.allowed))
        if case.static:
            out.append("- **Artifact checks:** " + ", ".join("`%s`" % name for name in case.static))
        out.append("- **Record:** " + "; ".join(case.measure))
        if case.notes:
            out.append("- **Notes:** " + case.notes)
        if case.prior_edits:
            out += ["", "Model edits of the earlier version in the packet's first commit (the second commit holds "
                        "the case's model):", ""]
            for edit in case.prior_edits:
                out += diff(edit)
        if case.model_edits:
            out += ["", "Model edits:", ""]
            for edit in case.model_edits:
                out += diff(edit)
        if case.faults:
            out += ["", "| Fault | Owner | Kind | What is wrong | Fails while present | Own tests fail | Regression test |",
                    "| --- | --- | --- | --- | --- | --- | --- |"]
            for fault in case.faults:
                out.append("| %s | %s | %s | %s | %s | %s | %s |" % (
                    fault.id, fault.owner, fault.kind, fault.summary,
                    ", ".join("`%s`" % name for name in fault.detected_by + fault.static) or "none",
                    "yes" if fault.test_signal else "no", fault.regression_test))
            out += ["", "Fault edits:", ""]
            for fault in case.faults:
                out.append("%s:" % fault.id)
                out.append("")
                for edit in fault.edits:
                    out += diff(edit)
                out.append("")
        out.append("")
    out += ["## Profile shifts", "",
            "The reference realization fails exactly these checks under a changed or variant profile:", ""]
    for (project, profile), names in SHIFTS.items():
        out.append("- %s `%s`: %s" % (project, profile, ", ".join("`%s`" % name for name in names)))
    out += ["", "## Oracle variants", "",
            "Each variant is the reference with exact edits. The oracle must accept an alternative that fails nothing "
            "and reject a wrong one with exactly the listed checks.", "",
            "| Variant | Project | Profile | Realization | Fails |", "| --- | --- | --- | --- | --- |"]
    for variant in VARIANTS:
        failing = ", ".join("`%s`" % name for name in variant.fails + variant.static) or "nothing (accepted)"
        out.append("| %s | %s | %s | %s | %s |" % (variant.id, variant.project, variant.profile, variant.summary, failing))
    for variant in VARIANTS:
        if variant.edits:
            out += ["", "%s:" % variant.id, ""]
            for edit in variant.edits:
                out += diff(edit)
    text = "\n".join(line.rstrip() for line in out)
    while "\n\n\n" in text:
        text = text.replace("\n\n\n", "\n\n")
    return text.rstrip() + "\n"


if __name__ == "__main__":
    import sys
    sys.stdout.reconfigure(encoding="utf-8", newline="\n")
    sys.stdout.write(markdown())
