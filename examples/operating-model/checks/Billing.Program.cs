using OperatingModel.Billing;

if (BillingQuery.SumOutstandingCents(Array.Empty<long>()) != 0)
    return Fail("an empty outstanding amount sequence must total zero");
if (BillingQuery.SumOutstandingCents(new long[] { 125, 300, 75 }) != 500)
    return Fail("outstanding amounts must be added in cents");

try
{
    _ = BillingQuery.SumOutstandingCents(new long[] { 100, -1 });
    return Fail("negative outstanding amounts must be rejected");
}
catch (Exception)
{
}

try
{
    _ = BillingQuery.SumOutstandingCents(new long[] { long.MaxValue, 1 });
    return Fail("sum overflow must be rejected");
}
catch (Exception)
{
}

Console.WriteLine("Billing public operation compiled and passed its fixed behavior cases.");
return 0;

static int Fail(string message)
{
    Console.Error.WriteLine(message);
    return 1;
}
