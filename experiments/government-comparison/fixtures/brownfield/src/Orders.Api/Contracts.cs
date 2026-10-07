namespace Orders.Api;

public sealed record ApiError(string Code, string Message);
public sealed record CatalogItem(string Sku, int UnitPriceCents);
public sealed record InventoryItem(string Sku, int OnHand, int Reserved, int Available);
public sealed record OrderLineRequest(string Sku, decimal Quantity);
public sealed record CreateOrderRequest(IReadOnlyList<OrderLineRequest>? Items);
public sealed record OrderLine(string Sku, int Quantity, int UnitPriceCents);
public sealed record Order(string Id, IReadOnlyList<OrderLine> Items, long TotalCents, string Status);

public sealed record CreateOrderResult(Order? Order, bool Replayed, string? Error, string? Message)
{
    public static CreateOrderResult Created(Order order) => new(order, false, null, null);
    public static CreateOrderResult Replay(Order order) => new(order, true, null, null);
    public static CreateOrderResult Failure(string error, string message) => new(null, false, error, message);
}
