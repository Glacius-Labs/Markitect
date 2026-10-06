namespace Orders;

public sealed class CreateOrderHandler(Action<int> persist)
{
    private readonly Action<int> _persist = persist ?? throw new ArgumentNullException(nameof(persist));

    public bool Execute(int quantity)
    {
        if (quantity < 1 || quantity > 10)
        {
            return false;
        }

        _persist(quantity);
        return true;
    }
}