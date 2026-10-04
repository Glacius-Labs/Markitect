namespace Commerce.Orders.Domain;
public sealed record OrderLine(string ProductId, int Quantity, decimal UnitPrice) { public decimal Total => UnitPrice; }
