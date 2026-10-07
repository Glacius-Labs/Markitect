namespace Fulfillment;

public sealed class FulfillOrderHandler
{
    private const int MinimumQuantity = 1;
    private const int MaximumQuantity = 10;

    private readonly Action<int> _dispatch;

    public FulfillOrderHandler(Action<int> dispatch)
    {
        _dispatch = dispatch ?? throw new ArgumentNullException(nameof(dispatch));
    }

    public bool Execute(int quantity)
    {
        if (quantity < MinimumQuantity || quantity > MaximumQuantity)
        {
            return false;
        }

        try
        {
            _dispatch(quantity);
            return true;
        }
        catch (InvalidOperationException)
        {
            return false;
        }
    }
}
