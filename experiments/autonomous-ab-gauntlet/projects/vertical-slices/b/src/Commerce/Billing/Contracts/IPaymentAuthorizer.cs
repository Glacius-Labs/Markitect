namespace Commerce.Billing.Contracts;
public interface IPaymentAuthorizer { bool CanAuthorize(string paymentToken, decimal amount); }
