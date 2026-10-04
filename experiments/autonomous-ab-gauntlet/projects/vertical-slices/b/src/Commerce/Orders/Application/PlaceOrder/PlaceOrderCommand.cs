using Commerce.Orders.Domain;
namespace Commerce.Orders.Application.PlaceOrder;
public sealed record PlaceOrderCommand(string OrderId, DateTimeOffset CreatedAt, IReadOnlyList<OrderLine> Lines, string PaymentToken);
