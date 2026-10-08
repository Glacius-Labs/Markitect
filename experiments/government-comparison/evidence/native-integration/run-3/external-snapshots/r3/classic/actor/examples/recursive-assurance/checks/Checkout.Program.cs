using RecursiveAssurance.Checkout;
if (CheckoutComposer.AddAcceptedOrderToOutstandingCents(2, 150, new long[] { 300 }) != 600) return Fail("Checkout must retain the 300-cent accepted line with 300 cents already outstanding");
if (!Throws(() => CheckoutComposer.AddAcceptedOrderToOutstandingCents(0, 150, new long[] { 300 }))) return Fail("Checkout must reject an invalid order instead of bypassing Orders");
if (!Throws(() => CheckoutComposer.AddAcceptedOrderToOutstandingCents(2, 150, new long[] { long.MaxValue }))) return Fail("Checkout must propagate Billing overflow");
Console.WriteLine("Checkout parent compiled its descendants and passed the child-interaction case."); return 0;
static bool Throws(Action action) { try { action(); return false; } catch (ArgumentOutOfRangeException) { return true; } catch (OverflowException) { return true; } }
static int Fail(string message) { Console.Error.WriteLine(message); return 1; }