namespace Commerce.Orders.Domain;
public sealed class Order
{
    private Order(string id, DateTimeOffset createdAt, IReadOnlyList<OrderLine> lines) { Id = id; CreatedAt = createdAt; Lines = lines; }
    public string Id { get; }
    public DateTimeOffset CreatedAt { get; }
    public IReadOnlyList<OrderLine> Lines { get; }
    public OrderStatus Status { get; private set; } = OrderStatus.Open;
    public decimal Total => Lines.Sum(line => line.Total);
    public static Order Place(string id, DateTimeOffset createdAt, IReadOnlyList<OrderLine> lines)
    {
        if (string.IsNullOrWhiteSpace(id)) throw new ArgumentException("Order id is required.", nameof(id));
        if (lines.Count == 0) throw new ArgumentException("At least one line is required.", nameof(lines));
        if (lines.Any(line => string.IsNullOrWhiteSpace(line.ProductId) || line.Quantity <= 0 || line.UnitPrice < 0m)) throw new ArgumentException("Order lines require a product, positive quantity, and non-negative price.", nameof(lines));
        return new Order(id, createdAt, lines.ToArray());
    }
    public void MarkShipped() { if (Status != OrderStatus.Open) throw new InvalidOperationException("Only an open order can ship."); Status = OrderStatus.Shipped; }
}
