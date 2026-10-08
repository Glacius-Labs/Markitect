using OperatingModel.Billing;
using OperatingModel.Orders;

if (!OrderRules.TryCalculateTotalCents(3, 199, out var acceptedOrderTotal))
    return Fail("valid Orders input must produce an order total");
var productOutstandingTotal = BillingQuery.SumOutstandingCents(new long[] { 950, acceptedOrderTotal });
if (productOutstandingTotal != 1547)
    return Fail("the Billing query must include the accepted Orders total alongside prior open amounts");

Console.WriteLine("The parent composition compiled both child APIs and passed its interaction case.");
return 0;

static int Fail(string message)
{
    Console.Error.WriteLine(message);
    return 1;
}
