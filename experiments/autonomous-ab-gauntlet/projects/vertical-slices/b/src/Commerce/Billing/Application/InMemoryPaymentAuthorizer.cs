using Commerce.Billing.Contracts;
namespace Commerce.Billing.Application;
public sealed class InMemoryPaymentAuthorizer : IPaymentAuthorizer
{
    public bool CanAuthorize(string paymentToken, decimal amount) => !string.IsNullOrWhiteSpace(paymentToken) && amount > 0m;
}
