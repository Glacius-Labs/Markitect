namespace Commerce.Inventory.Contracts;
public interface IStockAvailability { bool CanReserve(string productId, int quantity); }
