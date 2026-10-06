using UnknownOntology;
var axis = new EffectAxis("focus-duration", "minutes", 2, 10);
var capability = new Capability("guided-focus", "focus-duration", 6);
var mission = new Mission("daily-reflection", "focus-duration", "guided-focus", "minutes", 5);
if (!MissionEvaluator.IsSupported(mission, axis, capability)) return Fail("the selected Mission, EffectAxis and Capability should form a supported request");
if (MissionEvaluator.IsSupported(mission, new EffectAxis("travel-distance", "kilometers", 1, 10), capability)) return Fail("a Mission must not be evaluated against a different EffectAxis");
if (MissionEvaluator.IsSupported(mission with { CapabilityName = "other-capability" }, axis, capability)) return Fail("the required Capability reference must match");
if (MissionEvaluator.IsSupported(mission with { RequestedUnit = "hours" }, axis, capability)) return Fail("the requested unit must match the selected EffectAxis");
if (MissionEvaluator.IsSupported(mission with { RequestedMagnitude = 1 }, axis, capability)) return Fail("the EffectAxis inclusive minimum must be enforced");
if (MissionEvaluator.IsSupported(mission with { RequestedMagnitude = 11 }, axis, capability)) return Fail("the EffectAxis inclusive maximum must be enforced");
if (MissionEvaluator.IsSupported(mission with { RequestedMagnitude = 7 }, axis, capability)) return Fail("the selected Capability maximum must be enforced");
Console.WriteLine("Unknown ontology target API compiled and passed all three-kind behavior cases."); return 0;
static int Fail(string message) { Console.Error.WriteLine(message); return 1; }