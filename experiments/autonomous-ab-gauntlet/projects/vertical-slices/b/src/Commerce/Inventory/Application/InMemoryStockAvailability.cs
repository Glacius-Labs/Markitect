using Commerce.Inventory.Contracts;
namespace Commerce.Inventory.Application;
public sealed class InMemoryStockAvailability : IStockAvailability
{
    private readonly Dictionary<string, int> _available = new(StringComparer.Ordinal) { ["SKU-1"] = 20, ["SKU-2"] = 8 };
    public bool CanReserve(string productId, int quantity) => quantity > 0 && _available.TryGetValue(productId, out var count) && count >= quantity;
}
