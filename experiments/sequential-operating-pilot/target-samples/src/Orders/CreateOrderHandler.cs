namespace Orders;

public sealed class CreateOrderHandler(Action<string, int> persist)
{
    private readonly Action<string, int> _persist = persist ?? throw new ArgumentNullException(nameof(persist));

    public bool Execute(string sku, int quantity)
    {
        if (quantity < 1 || quantity > 10 || string.IsNullOrWhiteSpace(sku))
        {
            return false;
        }

        try
        {
            _persist(sku, quantity);
            return true;
        }
        catch (InvalidOperationException)
        {
            return false;
        }
    }
}
