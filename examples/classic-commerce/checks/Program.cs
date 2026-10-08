using Commerce.Application;
using Commerce.Domain;

var handler = new CreateOrderHandler();
AssertEqual(37.50m, handler.Handle(3, 12.50m), "whole-value total");
AssertEqual(0.30m, handler.Handle(3, 0.10m), "fractional decimal total");
AssertEqual(0m, handler.Handle(2, 0m), "zero unit price");
AssertEqual(1m, handler.Handle(1, 1m), "one item allowed");
AssertThrowsArgumentOutOfRange(() => handler.Handle(0, 1m), "zero quantity");
AssertThrowsArgumentOutOfRange(() => handler.Handle(-1, 1m), "negative quantity");
AssertThrowsArgumentOutOfRange(() => handler.Handle(2, -0.01m), "negative unit price");
AssertEqual("application", new EffectAxis().Boundary, "EffectAxis boundary");
Console.WriteLine("PASS: declared order cases and application boundary");

static void AssertEqual<T>(T expected, T actual, string check)
{
    if (!EqualityComparer<T>.Default.Equals(expected, actual))
    {
        throw new InvalidOperationException($"{check}: expected '{expected}', got '{actual}'");
    }
}

static void AssertThrowsArgumentOutOfRange(Action action, string check)
{
    try
    {
        action();
    }
    catch (ArgumentOutOfRangeException)
    {
        return;
    }

    throw new InvalidOperationException($"{check}: expected ArgumentOutOfRangeException");
}
