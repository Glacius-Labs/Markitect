using RecursiveAssurance.Commerce;
if (CommerceComposer.CombineTwoCheckoutsAndOutstandingCents(2, 150, 3, 100, new long[] { 300 }) != 900) return Fail("Commerce must invoke both Checkout operations and combine their totals with the shared Billing rule");
if (!Throws(() => CommerceComposer.CombineTwoCheckoutsAndOutstandingCents(2, 150, 0, 100, new long[] { 300 }))) return Fail("Commerce must propagate a rejected second checkout");
if (!Throws(() => CommerceComposer.CombineTwoCheckoutsAndOutstandingCents(2, 150, 3, 100, new long[] { long.MaxValue }))) return Fail("Commerce must apply the shared Billing overflow rule after composing both checkouts");
Console.WriteLine("Commerce top parent compiled all descendants and passed its two-checkout invariant."); return 0;
static bool Throws(Action action) { try { action(); return false; } catch (ArgumentOutOfRangeException) { return true; } catch (OverflowException) { return true; } }
static int Fail(string message) { Console.Error.WriteLine(message); return 1; }