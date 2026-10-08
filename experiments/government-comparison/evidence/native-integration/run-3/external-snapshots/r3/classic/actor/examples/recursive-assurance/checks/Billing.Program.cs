using RecursiveAssurance.Billing;
if (BillingQuery.SumOutstandingCents(Array.Empty<long>()) != 0) return Fail("empty outstanding input must total zero");
if (BillingQuery.SumOutstandingCents(new long[] { 100, 250, 75 }) != 425) return Fail("outstanding amounts must be summed");
if (!Throws(() => BillingQuery.SumOutstandingCents(new long[] { -1 }))) return Fail("negative amounts must be rejected");
if (!Throws(() => BillingQuery.SumOutstandingCents(new long[] { long.MaxValue, 1 }))) return Fail("overflow must be rejected");
Console.WriteLine("Billing leaf API compiled and passed its bounded cases."); return 0;
static bool Throws(Action action) { try { action(); return false; } catch (ArgumentOutOfRangeException) { return true; } catch (OverflowException) { return true; } }
static int Fail(string message) { Console.Error.WriteLine(message); return 1; }