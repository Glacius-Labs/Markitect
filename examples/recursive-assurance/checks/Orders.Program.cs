using RecursiveAssurance.Orders;
if (!OrderRules.TryCalculateLineTotalCents(3, 125, out var total) || total != 375) return Fail("three units at 125 cents must total 375 cents");
if (OrderRules.TryCalculateLineTotalCents(0, 125, out total) || total != 0) return Fail("zero quantity must fail with zero output");
if (OrderRules.TryCalculateLineTotalCents(3, -1, out total) || total != 0) return Fail("negative price must fail with zero output");
if (OrderRules.TryCalculateLineTotalCents(2, long.MaxValue, out total) || total != 0) return Fail("overflow must fail with zero output");
Console.WriteLine("Orders leaf API compiled and passed its bounded cases."); return 0;
static int Fail(string message) { Console.Error.WriteLine(message); return 1; }