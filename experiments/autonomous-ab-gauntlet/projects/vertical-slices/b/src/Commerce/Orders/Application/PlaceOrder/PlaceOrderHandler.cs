using Commerce.Billing.Contracts;
using Commerce.Inventory.Contracts;
using Commerce.Orders.Domain;
namespace Commerce.Orders.Application.PlaceOrder;
public sealed class PlaceOrderHandler(IStockAvailability stock, IPaymentAuthorizer payments)
{
    public Order Handle(PlaceOrderCommand command)
    {
        var order = Order.Place(command.OrderId, command.CreatedAt, command.Lines);
        if (command.Lines.Any(line => !stock.CanReserve(line.ProductId, line.Quantity))) throw new InvalidOperationException("Insufficient stock.");
        if (!payments.CanAuthorize(command.PaymentToken, order.Total)) throw new InvalidOperationException("Payment could not be authorized.");
        return order;
    }
}
