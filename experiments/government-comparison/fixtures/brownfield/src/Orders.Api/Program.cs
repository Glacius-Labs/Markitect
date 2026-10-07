using Orders.Api;

var builder = WebApplication.CreateBuilder(args);
var databasePath = Environment.GetEnvironmentVariable("ORDERS_DB") ?? "orders.db";
builder.Services.AddSingleton(new OrdersStore(databasePath));

var app = builder.Build();
var store = app.Services.GetRequiredService<OrdersStore>();
store.Initialize();

app.MapGet("/health", () => Results.Ok(new { status = "ok" }));
app.MapGet("/catalog", (OrdersStore orders) => Results.Ok(orders.GetCatalog()));
app.MapGet("/inventory/{sku}", (string sku, OrdersStore orders) =>
{
    var inventory = orders.GetInventory(sku);
    return inventory is null
        ? Results.NotFound(new ApiError("unknown_sku", $"Unknown SKU '{sku}'."))
        : Results.Ok(inventory);
});

Func<HttpRequest, CreateOrderRequest?, OrdersStore, IResult> orderHandler = (request, body, orders) =>
{
    if (body?.Items is not { Count: > 0 })
        return Results.BadRequest(new ApiError("validation", "At least one item is required."));

    var key = request.Headers["Idempotency-Key"].ToString();
    if (string.IsNullOrWhiteSpace(key))
        return Results.BadRequest(new ApiError("validation", "Idempotency-Key is required."));

    var result = orders.CreateOrder(key, body.Items);
    return result.Error switch
    {
        "validation" or "unknown_sku" or "order_limit" => Results.BadRequest(new ApiError(result.Error, result.Message!)),
        "idempotency_conflict" => Results.Conflict(new ApiError(result.Error, result.Message!)),
        _ => result.Replayed ? Results.Ok(result.Order) : Results.Created($"/orders/{result.Order!.Id}", result.Order)
    };
};

app.MapPost("/orders", orderHandler);
// Existing clients still use this route; it intentionally shares the established behavior.
app.MapPost("/legacy/orders", orderHandler);
app.MapGet("/orders/{id}", (string id, OrdersStore orders) =>
{
    var order = orders.GetOrder(id);
    return order is null
        ? Results.NotFound(new ApiError("not_found", $"Unknown order '{id}'."))
        : Results.Ok(order);
});

app.Run();
