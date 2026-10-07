using OperatingModel.Orders;

if (!OrderRules.TryCalculateTotalCents(2, 125, out var total) || total != 250)
    return Fail("two units at 125 cents must total 250 cents");
if (OrderRules.TryCalculateTotalCents(0, 125, out total) || total != 0)
    return Fail("zero quantity must fail and reset the result");
if (OrderRules.TryCalculateTotalCents(2, -1, out total) || total != 0)
    return Fail("negative unit price must fail and reset the result");
if (OrderRules.TryCalculateTotalCents(2, long.MaxValue, out total) || total != 0)
    return Fail("signed 64-bit multiplication overflow must fail and reset the result");

Console.WriteLine("Orders public operation compiled and passed its fixed behavior cases.");
return 0;

static int Fail(string message)
{
    Console.Error.WriteLine(message);
    return 1;
}
